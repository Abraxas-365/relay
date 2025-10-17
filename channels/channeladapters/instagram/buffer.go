package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
)

// BufferedMessage represents an Instagram message waiting in the buffer
// Messages are buffered when users send multiple messages in quick succession
type BufferedMessage struct {
	MessageID   kernel.MessageID      `json:"message_id"`
	SenderID    string                `json:"sender_id"`
	Content     string                `json:"content"`
	ReceivedAt  time.Time             `json:"received_at"`
	Attachments []channels.Attachment `json:"attachments,omitempty"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
	MessageType string                `json:"message_type,omitempty"` // text, image, video, postback, reaction
}

// MessageBuffer represents the complete buffer state for an Instagram user
// Contains all messages from a user within the buffer time window
type MessageBuffer struct {
	ChannelID    kernel.ChannelID  `json:"channel_id"`
	SenderID     string            `json:"sender_id"`
	Messages     []BufferedMessage `json:"messages"`
	FirstMessage time.Time         `json:"first_message"`
	LastMessage  time.Time         `json:"last_message"`
	TimerKey     string            `json:"timer_key,omitempty"`
}

// BufferService handles Instagram message buffering with Redis
//
// The buffer service combines multiple messages from the same user into a single
// message to improve conversation context and reduce processing overhead.
//
// Features:
//   - Configurable buffer time window
//   - Optional timer reset on each new message
//   - Automatic buffer expiration
//   - Combines text, attachments, and metadata
//
// Example:
//
//	User sends:
//	  1. "Hey"
//	  2. "Can you"
//	  3. "help me?"
//
//	Service buffers for 5 seconds, then combines into:
//	  "Hey\nCan you\nhelp me?"
type BufferService struct {
	redis  *redis.Client
	config BufferConfig
}

// BufferConfig holds buffer configuration
// This would typically come from InstagramConfig
type BufferConfig struct {
	Enabled              bool `json:"buffer_enabled"`
	TimeSeconds          int  `json:"buffer_time_seconds"`
	ResetOnMessage       bool `json:"buffer_reset_on_message"`
	MaxMessagesPerBuffer int  `json:"max_messages_per_buffer,omitempty"` // Optional limit
}

// NewBufferService creates a new Instagram buffer service
//
// Parameters:
//   - redisClient: Redis client for state management
//   - config: Buffer configuration settings
//
// Returns:
//   - *BufferService: Configured buffer service ready to use
func NewBufferService(redisClient *redis.Client, config BufferConfig) *BufferService {
	// Set defaults if not provided
	if config.TimeSeconds <= 0 {
		config.TimeSeconds = 5 // Default 5 seconds
	}
	if config.MaxMessagesPerBuffer <= 0 {
		config.MaxMessagesPerBuffer = 10 // Default max 10 messages
	}

	return &BufferService{
		redis:  redisClient,
		config: config,
	}
}

// getBufferKey generates Redis key for Instagram message buffer
//
// Format: relay:instagram:buffer:{channelID}:{senderID}
func (s *BufferService) getBufferKey(channelID kernel.ChannelID, senderID string) string {
	return fmt.Sprintf("relay:instagram:buffer:%s:%s", channelID, senderID)
}

// getTimerKey generates Redis key for buffer timer
//
// Format: relay:instagram:buffer:timer:{channelID}:{senderID}
func (s *BufferService) getTimerKey(channelID kernel.ChannelID, senderID string) string {
	return fmt.Sprintf("relay:instagram:buffer:timer:%s:%s", channelID, senderID)
}

// AddMessage adds an Instagram message to the buffer or triggers flush if buffering is disabled
//
// Flow:
//  1. If buffering disabled, return message immediately
//  2. Get existing buffer from Redis
//  3. Add new message to buffer
//  4. Set/reset timer based on configuration
//  5. Return nil (message buffered) or message (should process now)
//
// Parameters:
//   - ctx: Context for Redis operations
//   - channelID: Channel ID for buffer isolation
//   - message: Incoming Instagram message to buffer
//
// Returns:
//   - *channels.IncomingMessage: Combined message if ready to process, nil if buffered
//   - bool: true if message should be processed now, false if buffered
//   - error: Any error during buffering
func (s *BufferService) AddMessage(
	ctx context.Context,
	channelID kernel.ChannelID,
	message channels.IncomingMessage,
) (*channels.IncomingMessage, bool, error) {
	// If buffering is disabled, return message immediately
	if !s.config.Enabled {
		return &message, true, nil
	}

	bufferKey := s.getBufferKey(channelID, message.SenderID)
	timerKey := s.getTimerKey(channelID, message.SenderID)

	// Get existing buffer
	buffer, err := s.getBuffer(ctx, bufferKey)
	if err != nil && err != redis.Nil {
		return nil, false, fmt.Errorf("failed to get buffer: %w", err)
	}

	now := time.Now()

	// Initialize new buffer if doesn't exist
	if buffer == nil {
		buffer = &MessageBuffer{
			ChannelID:    channelID,
			SenderID:     message.SenderID,
			Messages:     []BufferedMessage{},
			FirstMessage: now,
			LastMessage:  now,
		}
	}

	// Check if buffer has reached max messages (prevent memory issues)
	if len(buffer.Messages) >= s.config.MaxMessagesPerBuffer {
		// Flush immediately
		combinedMsg := s.combineMessages(buffer)
		s.redis.Del(ctx, bufferKey, timerKey)
		return combinedMsg, true, nil
	}

	// Add message to buffer
	bufferedMsg := BufferedMessage{
		MessageID:   message.MessageID,
		SenderID:    message.SenderID,
		Content:     s.extractContent(message),
		ReceivedAt:  now,
		Attachments: message.Content.Attachments,
		Metadata:    message.Metadata,
		MessageType: message.Content.Type,
	}

	buffer.Messages = append(buffer.Messages, bufferedMsg)
	buffer.LastMessage = now

	// Save buffer
	if err := s.saveBuffer(ctx, bufferKey, buffer); err != nil {
		return nil, false, fmt.Errorf("failed to save buffer: %w", err)
	}

	// Calculate TTL for buffer timeout
	bufferDuration := time.Duration(s.config.TimeSeconds) * time.Second

	// If BufferResetOnMessage is true, reset the timer on each new message
	if s.config.ResetOnMessage {
		// Delete old timer if exists
		s.redis.Del(ctx, timerKey)

		// Set new timer
		s.redis.SetEX(ctx, timerKey, "1", bufferDuration)

		// Set buffer expiry (slightly longer than timer)
		s.redis.Expire(ctx, bufferKey, bufferDuration+time.Second)

		// Return nil to indicate message is buffered (don't process yet)
		return nil, false, nil
	}

	// If NOT resetting on each message, check if this is first message
	exists, _ := s.redis.Exists(ctx, timerKey).Result()
	if exists == 0 {
		// First message - start timer
		s.redis.SetEX(ctx, timerKey, "1", bufferDuration)
		s.redis.Expire(ctx, bufferKey, bufferDuration+time.Second)

		// Return nil to indicate message is buffered
		return nil, false, nil
	}

	// Timer already running, just add to buffer
	return nil, false, nil
}

// CheckAndFlush checks if buffer should be flushed and returns combined message
//
// This is typically called by the BufferWorker periodically to check for expired buffers.
//
// Parameters:
//   - ctx: Context for Redis operations
//   - channelID: Channel ID to check
//   - senderID: Sender ID to check
//
// Returns:
//   - *channels.IncomingMessage: Combined message if buffer expired, nil otherwise
//   - error: Any error during check/flush
func (s *BufferService) CheckAndFlush(
	ctx context.Context,
	channelID kernel.ChannelID,
	senderID string,
) (*channels.IncomingMessage, error) {
	if !s.config.Enabled {
		return nil, nil
	}

	bufferKey := s.getBufferKey(channelID, senderID)
	timerKey := s.getTimerKey(channelID, senderID)

	// Check if timer has expired
	exists, _ := s.redis.Exists(ctx, timerKey).Result()
	if exists > 0 {
		// Timer still running, don't flush
		return nil, nil
	}

	// Timer expired, flush buffer
	buffer, err := s.getBuffer(ctx, bufferKey)
	if err != nil {
		if err == redis.Nil {
			return nil, nil // No buffer exists
		}
		return nil, fmt.Errorf("failed to get buffer: %w", err)
	}

	if buffer == nil || len(buffer.Messages) == 0 {
		return nil, nil
	}

	// Combine messages
	combinedMessage := s.combineMessages(buffer)

	// Delete buffer and timer
	s.redis.Del(ctx, bufferKey, timerKey)

	return combinedMessage, nil
}

// FlushNow immediately flushes the buffer for a user
//
// This is useful for forcing a flush before the timer expires,
// for example when a user expects an immediate response.
//
// Parameters:
//   - ctx: Context for Redis operations
//   - channelID: Channel ID
//   - senderID: Sender ID
//
// Returns:
//   - *channels.IncomingMessage: Combined message, nil if no buffer
//   - error: Any error during flush
func (s *BufferService) FlushNow(
	ctx context.Context,
	channelID kernel.ChannelID,
	senderID string,
) (*channels.IncomingMessage, error) {
	bufferKey := s.getBufferKey(channelID, senderID)
	timerKey := s.getTimerKey(channelID, senderID)

	buffer, err := s.getBuffer(ctx, bufferKey)
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	if buffer == nil || len(buffer.Messages) == 0 {
		return nil, nil
	}

	combinedMessage := s.combineMessages(buffer)

	// Delete buffer and timer
	s.redis.Del(ctx, bufferKey, timerKey)

	return combinedMessage, nil
}

// getBuffer retrieves buffer from Redis
func (s *BufferService) getBuffer(ctx context.Context, key string) (*MessageBuffer, error) {
	data, err := s.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var buffer MessageBuffer
	if err := json.Unmarshal([]byte(data), &buffer); err != nil {
		return nil, err
	}

	return &buffer, nil
}

// saveBuffer saves buffer to Redis
func (s *BufferService) saveBuffer(ctx context.Context, key string, buffer *MessageBuffer) error {
	data, err := json.Marshal(buffer)
	if err != nil {
		return err
	}

	// Buffer expires after timeout + 1 second (safety margin)
	expiry := time.Duration(s.config.TimeSeconds+1) * time.Second
	return s.redis.Set(ctx, key, data, expiry).Err()
}

// combineMessages combines buffered Instagram messages into a single message
//
// Combines:
//   - All message texts with line breaks
//   - All attachments into single array
//   - Metadata from all messages
//   - Adds buffer metadata (message count, duration, etc.)
//
// Parameters:
//   - buffer: Message buffer to combine
//
// Returns:
//   - *channels.IncomingMessage: Combined message ready for processing
func (s *BufferService) combineMessages(buffer *MessageBuffer) *channels.IncomingMessage {
	if len(buffer.Messages) == 0 {
		return nil
	}

	// Use first message as base
	firstMsg := buffer.Messages[0]

	// Combine all message contents with line breaks
	var combinedContent string
	var allAttachments []channels.Attachment
	combinedMetadata := make(map[string]any)
	messageTypes := make([]string, 0)

	for i, msg := range buffer.Messages {
		// Add text content
		if msg.Content != "" {
			if i > 0 && combinedContent != "" {
				combinedContent += "\n"
			}
			combinedContent += msg.Content
		}

		// Collect attachments
		allAttachments = append(allAttachments, msg.Attachments...)

		// Collect message types
		if msg.MessageType != "" {
			messageTypes = append(messageTypes, msg.MessageType)
		}

		// Merge metadata
		for k, v := range msg.Metadata {
			// Avoid overwriting, use array for duplicates
			if existing, exists := combinedMetadata[k]; exists {
				// Convert to array if not already
				if arr, isArray := existing.([]any); isArray {
					combinedMetadata[k] = append(arr, v)
				} else {
					combinedMetadata[k] = []any{existing, v}
				}
			} else {
				combinedMetadata[k] = v
			}
		}
	}

	// Add buffer metadata
	combinedMetadata["buffered"] = true
	combinedMetadata["message_count"] = len(buffer.Messages)
	combinedMetadata["first_message_at"] = buffer.FirstMessage
	combinedMetadata["last_message_at"] = buffer.LastMessage
	combinedMetadata["buffer_duration_seconds"] = buffer.LastMessage.Sub(buffer.FirstMessage).Seconds()
	combinedMetadata["message_types"] = messageTypes

	// Determine primary content type
	contentType := "text"
	if len(allAttachments) > 0 {
		contentType = allAttachments[0].Type
	}

	// Create combined message
	return &channels.IncomingMessage{
		MessageID: firstMsg.MessageID,
		ChannelID: buffer.ChannelID,
		SenderID:  buffer.SenderID,
		Content: channels.MessageContent{
			Type:        contentType,
			Text:        combinedContent,
			Attachments: allAttachments,
		},
		Timestamp: buffer.FirstMessage.Unix(),
		Metadata:  combinedMetadata,
	}
}

// extractContent extracts text content from Instagram message
//
// Handles different message types and extracts the appropriate content
func (s *BufferService) extractContent(msg channels.IncomingMessage) string {
	// Text content
	if msg.Content.Text != "" {
		return msg.Content.Text
	}

	// Caption from media
	if msg.Content.Caption != "" {
		return msg.Content.Caption
	}

	// For non-text messages, return a placeholder
	if msg.Content.Type != "" && msg.Content.Type != "text" {
		return fmt.Sprintf("[%s]", msg.Content.Type)
	}

	// Check metadata for special types
	if postbackPayload, ok := msg.Metadata["postback_payload"].(string); ok {
		return fmt.Sprintf("[Button: %s]", postbackPayload)
	}

	if reaction, ok := msg.Metadata["reaction_emoji"].(string); ok {
		return fmt.Sprintf("[Reaction: %s]", reaction)
	}

	return ""
}

// GetBufferStats returns statistics about current buffers
//
// Useful for monitoring and debugging buffer behavior
func (s *BufferService) GetBufferStats(ctx context.Context) (map[string]any, error) {
	pattern := "relay:instagram:buffer:*"
	var cursor uint64
	bufferCount := 0
	timerCount := 0

	for {
		keys, nextCursor, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		for _, key := range keys {
			if contains(key, ":timer:") {
				timerCount++
			} else {
				bufferCount++
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return map[string]any{
		"active_buffers":      bufferCount,
		"active_timers":       timerCount,
		"buffer_enabled":      s.config.Enabled,
		"buffer_time_seconds": s.config.TimeSeconds,
		"reset_on_message":    s.config.ResetOnMessage,
	}, nil
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

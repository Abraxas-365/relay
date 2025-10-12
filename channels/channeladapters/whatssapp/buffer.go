package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
)

// BufferedMessage represents a message waiting in the buffer
type BufferedMessage struct {
	MessageID   kernel.MessageID      `json:"message_id"`
	SenderID    string                `json:"sender_id"`
	Content     string                `json:"content"`
	ReceivedAt  time.Time             `json:"received_at"`
	Attachments []channels.Attachment `json:"attachments,omitempty"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
}

// MessageBuffer represents the complete buffer state for a user
type MessageBuffer struct {
	ChannelID    kernel.ChannelID  `json:"channel_id"`
	SenderID     string            `json:"sender_id"`
	Messages     []BufferedMessage `json:"messages"`
	FirstMessage time.Time         `json:"first_message"`
	LastMessage  time.Time         `json:"last_message"`
	TimerKey     string            `json:"timer_key,omitempty"`
}

// BufferService handles message buffering with Redis
type BufferService struct {
	redis  *redis.Client
	config channels.WhatsAppConfig
}

// NewBufferService creates a new buffer service
func NewBufferService(redisClient *redis.Client, config channels.WhatsAppConfig) *BufferService {
	return &BufferService{
		redis:  redisClient,
		config: config,
	}
}

// getBufferKey generates Redis key for message buffer
func (s *BufferService) getBufferKey(channelID kernel.ChannelID, senderID string) string {
	return fmt.Sprintf("relay:buffer:%s:%s", channelID, senderID)
}

// getTimerKey generates Redis key for buffer timer
func (s *BufferService) getTimerKey(channelID kernel.ChannelID, senderID string) string {
	return fmt.Sprintf("relay:buffer:timer:%s:%s", channelID, senderID)
}

// AddMessage adds a message to the buffer or triggers flush if buffer is disabled
func (s *BufferService) AddMessage(
	ctx context.Context,
	channelID kernel.ChannelID,
	message channels.IncomingMessage,
) (*channels.IncomingMessage, bool, error) {
	// If buffering is disabled, return message immediately
	if !s.config.BufferEnabled {
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

	// Add message to buffer
	bufferedMsg := BufferedMessage{
		MessageID:   message.MessageID,
		SenderID:    message.SenderID,
		Content:     s.extractContent(message),
		ReceivedAt:  now,
		Attachments: message.Content.Attachments,
		Metadata:    message.Metadata,
	}

	buffer.Messages = append(buffer.Messages, bufferedMsg)
	buffer.LastMessage = now

	// Save buffer
	if err := s.saveBuffer(ctx, bufferKey, buffer); err != nil {
		return nil, false, fmt.Errorf("failed to save buffer: %w", err)
	}

	// Calculate TTL for buffer timeout
	bufferDuration := time.Duration(s.config.BufferTimeSeconds) * time.Second

	// If BufferResetOnMessage is true, reset the timer on each new message
	if s.config.BufferResetOnMessage {
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
func (s *BufferService) CheckAndFlush(
	ctx context.Context,
	channelID kernel.ChannelID,
	senderID string,
) (*channels.IncomingMessage, error) {
	if !s.config.BufferEnabled {
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

	// Buffer expires after timeout + 1 second
	expiry := time.Duration(s.config.BufferTimeSeconds+1) * time.Second
	return s.redis.Set(ctx, key, data, expiry).Err()
}

// combineMessages combines buffered messages into a single message
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

	for i, msg := range buffer.Messages {
		if i > 0 {
			combinedContent += "\n"
		}
		combinedContent += msg.Content

		// Collect attachments
		allAttachments = append(allAttachments, msg.Attachments...)

		// Merge metadata
		for k, v := range msg.Metadata {
			combinedMetadata[k] = v
		}
	}

	// Add buffer metadata
	combinedMetadata["buffered"] = true
	combinedMetadata["message_count"] = len(buffer.Messages)
	combinedMetadata["first_message_at"] = buffer.FirstMessage
	combinedMetadata["last_message_at"] = buffer.LastMessage
	combinedMetadata["buffer_duration_seconds"] = buffer.LastMessage.Sub(buffer.FirstMessage).Seconds()

	// Create combined message
	return &channels.IncomingMessage{
		MessageID: firstMsg.MessageID,
		ChannelID: buffer.ChannelID,
		SenderID:  buffer.SenderID,
		Content: channels.MessageContent{
			Type:        "text",
			Text:        combinedContent,
			Attachments: allAttachments,
		},
		Timestamp: buffer.FirstMessage.Unix(),
		Metadata:  combinedMetadata,
	}
}

// extractContent extracts text content from message
func (s *BufferService) extractContent(msg channels.IncomingMessage) string {
	if msg.Content.Text != "" {
		return msg.Content.Text
	}
	if msg.Content.Caption != "" {
		return msg.Content.Caption
	}
	if msg.Content.Type != "" {
		return fmt.Sprintf("[%s]", msg.Content.Type)
	}
	return ""
}

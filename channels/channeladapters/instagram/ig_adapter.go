package instagram

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
)

const (
	// instagramAPIBaseURL is the base URL for Instagram Graph API
	instagramAPIBaseURL = "https://graph.facebook.com"

	// defaultAPIVersion is the default Instagram API version to use
	defaultAPIVersion = "v24.0"

	// maxRetries defines maximum retry attempts for API calls
	maxRetries = 3

	// requestTimeout defines the timeout for HTTP requests
	requestTimeout = 30 * time.Second
)

// InstagramAdapter implements ChannelAdapter for Instagram Messaging API
// It handles Instagram Direct Messages through Meta's Graph API
type InstagramAdapter struct {
	config        channels.InstagramConfig
	httpClient    *http.Client
	bufferService *BufferService
	apiURL        string
}

// NewInstagramAdapter creates a new Instagram adapter instance
//
// Parameters:
//   - config: Instagram channel configuration containing page credentials
//   - redisClient: Redis client for message buffering (can be nil if buffering disabled)
//
// Returns:
//   - *InstagramAdapter: Configured adapter ready to send/receive messages
func NewInstagramAdapter(config channels.InstagramConfig, redisClient *redis.Client) *InstagramAdapter {
	apiVersion := defaultAPIVersion

	// Create buffer service configuration
	bufferConfig := BufferConfig{
		Enabled:        config.BufferEnabled,
		TimeSeconds:    config.BufferTimeSeconds,
		ResetOnMessage: config.BufferResetOnMessage,
	}

	return &InstagramAdapter{
		config:        config,
		httpClient:    &http.Client{Timeout: requestTimeout},
		bufferService: NewBufferService(redisClient, bufferConfig),
		apiURL:        fmt.Sprintf("%s/%s/%s", instagramAPIBaseURL, apiVersion, config.PageID),
	}
}

// ============================================================================
// ChannelAdapter Interface Implementation
// ============================================================================

// GetType returns the channel type for this adapter
func (a *InstagramAdapter) GetType() channels.ChannelType {
	return channels.ChannelTypeInstagram
}

// SendMessage sends a message via Instagram Direct Message
//
// Supports:
//   - Text messages
//   - Images with optional captions
//   - Videos
//   - Quick replies (buttons)
//   - Generic templates
//
// Parameters:
//   - ctx: Context for request cancellation and timeout
//   - msg: Outgoing message containing recipient and content
//
// Returns:
//   - error: nil if successful, error with details if failed
func (a *InstagramAdapter) SendMessage(ctx context.Context, msg channels.OutgoingMessage) error {
	// Build Instagram API payload based on message type
	payload := a.buildMessagePayload(msg)

	// Construct the messages endpoint
	url := fmt.Sprintf("%s/messages", a.apiURL)

	log.Printf("🌐 Instagram API URL: %s", url)
	log.Printf("📦 Payload: %+v", payload)

	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set required headers
	req.Header.Set("Authorization", "Bearer "+a.config.PageToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute request with retry logic
	var resp *http.Response
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = a.httpClient.Do(req)
		if err == nil {
			break
		}

		if attempt < maxRetries {
			log.Printf("⚠️  Instagram API request failed (attempt %d/%d): %v", attempt+1, maxRetries, err)
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to send request after %d attempts: %w", maxRetries, err)
	}
	defer resp.Body.Close()

	// Read response body
	body, _ := io.ReadAll(resp.Body)

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("❌ Instagram API Error - Status: %d, Body: %s", resp.StatusCode, string(body))
		return a.parseAPIError(resp.StatusCode, body)
	}

	log.Printf("✅ Instagram message sent successfully - Response: %s", string(body))
	return nil
}

// ValidateConfig validates the Instagram channel configuration
//
// Checks:
//   - Required fields are present
//   - Page ID format is valid
//   - Token is not empty
func (a *InstagramAdapter) ValidateConfig(config channels.ChannelConfig) error {
	instagramConfig, ok := config.(channels.InstagramConfig)
	if !ok {
		return channels.ErrInvalidChannelConfig().WithDetail("reason", "invalid config type")
	}

	return instagramConfig.Validate()
}

// ProcessWebhook processes incoming Instagram webhook events
//
// Handles:
//   - Message events (text, images, videos)
//   - Message reactions
//   - Message echoes (sent messages)
//   - Read receipts
//   - Delivery confirmations
//
// Parameters:
//   - ctx: Context for processing
//   - payload: Raw webhook payload from Instagram
//   - headers: HTTP headers including signature
//
// Returns:
//   - *channels.IncomingMessage: Parsed message or nil if not a message event
//   - error: Processing error if any
func (a *InstagramAdapter) ProcessWebhook(
	ctx context.Context,
	payload []byte,
	headers map[string]string,
) (*channels.IncomingMessage, error) {
	// Verify webhook signature for security
	if err := a.verifySignature(payload, headers); err != nil {
		log.Printf("❌ Instagram webhook signature verification failed: %v", err)
		return nil, err
	}

	// Parse webhook payload
	var webhook InstagramWebhook
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("failed to parse Instagram webhook: %w", err)
	}

	log.Printf("📥 Instagram webhook received - Object: %s", webhook.Object)

	// Extract incoming message from webhook
	incomingMsg, err := a.extractIncomingMessage(webhook)
	if err != nil {
		return nil, fmt.Errorf("failed to extract message from webhook: %w", err)
	}

	if incomingMsg == nil {
		log.Printf("ℹ️  Instagram webhook contained no processable message (likely status update)")
		return nil, nil // No message to process (status update, echo, etc.)
	}

	log.Printf("✅ Instagram message extracted - From: %s, Type: %s", incomingMsg.SenderID, incomingMsg.Content.Type)

	// Add to buffer if buffering is enabled
	processedMsg, shouldProcess, err := a.bufferService.AddMessage(
		ctx,
		incomingMsg.ChannelID,
		*incomingMsg,
	)

	if err != nil {
		return nil, fmt.Errorf("buffer error: %w", err)
	}

	// If shouldProcess is false, message is buffered - return nil
	if !shouldProcess {
		log.Printf("📦 Instagram message buffered for channel: %s, sender: %s", incomingMsg.ChannelID, incomingMsg.SenderID)
		return nil, nil
	}

	// Message should be processed immediately
	return processedMsg, nil
}

// GetFeatures returns the capabilities of the Instagram channel
//
// Instagram supports:
//   - Text messages (up to 1000 chars)
//   - Images (JPEG, PNG)
//   - Videos (MP4, MOV)
//   - Quick replies (buttons)
//   - Generic templates
//   - Reactions (emojis)
func (a *InstagramAdapter) GetFeatures() channels.ChannelFeatures {
	return a.config.GetFeatures()
}

// TestConnection tests connectivity to Instagram API
//
// Validates:
//   - Page access token is valid
//   - Page ID exists and is accessible
//   - API permissions are sufficient
//
// Parameters:
//   - ctx: Context with timeout
//   - config: Configuration to test
//
// Returns:
//   - error: nil if connection successful, error with details if failed
func (a *InstagramAdapter) TestConnection(ctx context.Context, config channels.ChannelConfig) error {
	instagramConfig, ok := config.(channels.InstagramConfig)
	if !ok {
		return channels.ErrInvalidChannelConfig().WithDetail("reason", "invalid config type")
	}

	// Test by fetching page information
	url := fmt.Sprintf("%s/%s/%s?fields=id,name,instagram_business_account",
		instagramAPIBaseURL,
		defaultAPIVersion,
		instagramConfig.PageID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+instagramConfig.PageToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return channels.ErrProviderAPIError().
			WithDetail("reason", "failed to connect to Instagram API").
			WithCause(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("❌ Instagram API test failed - Status: %d, Body: %s", resp.StatusCode, string(body))

		return channels.ErrProviderAuthFailed().
			WithDetail("status", resp.StatusCode).
			WithDetail("response", string(body))
	}

	log.Printf("✅ Instagram API connection test successful")
	return nil
}

// ============================================================================
// Message Payload Building
// ============================================================================

// buildMessagePayload constructs the Instagram API message payload
// based on the outgoing message content type
func (a *InstagramAdapter) buildMessagePayload(msg channels.OutgoingMessage) map[string]any {
	payload := map[string]any{
		"recipient": map[string]any{
			"id": msg.RecipientID,
		},
		"messaging_type": "RESPONSE", // RESPONSE for replies, UPDATE for proactive messages
	}

	switch msg.Content.Type {
	case "text":
		payload["message"] = a.buildTextMessage(msg)

	case "image":
		payload["message"] = a.buildImageMessage(msg)

	case "video":
		payload["message"] = a.buildVideoMessage(msg)

	case "template":
		payload["message"] = a.buildTemplateMessage(msg)

	default:
		// Default to text message
		payload["message"] = map[string]any{
			"text": msg.Content.Text,
		}
	}

	return payload
}

// buildTextMessage creates a text message payload
func (a *InstagramAdapter) buildTextMessage(msg channels.OutgoingMessage) map[string]any {
	message := map[string]any{
		"text": msg.Content.Text,
	}

	// Add quick replies if present (using buttons as quick replies)
	if msg.Content.Interactive != nil && len(msg.Content.Interactive.Buttons) > 0 {
		message["quick_replies"] = a.buildQuickReplies(msg.Content.Interactive.Buttons)
	}

	return message
}

// buildImageMessage creates an image message payload
func (a *InstagramAdapter) buildImageMessage(msg channels.OutgoingMessage) map[string]any {
	message := map[string]any{
		"attachment": map[string]any{
			"type": "image",
			"payload": map[string]any{
				"url":         msg.Content.MediaURL,
				"is_reusable": true,
			},
		},
	}

	return message
}

// buildVideoMessage creates a video message payload
func (a *InstagramAdapter) buildVideoMessage(msg channels.OutgoingMessage) map[string]any {
	message := map[string]any{
		"attachment": map[string]any{
			"type": "video",
			"payload": map[string]any{
				"url":         msg.Content.MediaURL,
				"is_reusable": true,
			},
		},
	}

	return message
}

// buildTemplateMessage creates a template/generic message payload
func (a *InstagramAdapter) buildTemplateMessage(msg channels.OutgoingMessage) map[string]any {
	var buttons []map[string]any
	if msg.Content.Interactive != nil {
		buttons = a.buildButtons(msg.Content.Interactive.Buttons)
	}

	// Instagram uses generic template for structured messages
	message := map[string]any{
		"attachment": map[string]any{
			"type": "template",
			"payload": map[string]any{
				"template_type": "generic",
				"elements": []map[string]any{
					{
						"title":    msg.Content.Text,
						"subtitle": msg.Content.Caption,
						"buttons":  buttons,
					},
				},
			},
		},
	}

	return message
}

// buildQuickReplies converts buttons to Instagram quick reply format
func (a *InstagramAdapter) buildQuickReplies(buttons []channels.Button) []map[string]any {
	quickReplies := make([]map[string]any, 0, len(buttons))

	for _, btn := range buttons {
		quickReplies = append(quickReplies, map[string]any{
			"content_type": "text",
			"title":        btn.Title,
			"payload":      btn.ID,
		})
	}

	return quickReplies
}

// buildButtons converts buttons to Instagram format
func (a *InstagramAdapter) buildButtons(buttons []channels.Button) []map[string]any {
	igButtons := make([]map[string]any, 0, len(buttons))

	for _, btn := range buttons {
		button := map[string]any{
			"type":  "postback",
			"title": btn.Title,
		}

		if btn.URL != "" {
			button["type"] = "web_url"
			button["url"] = btn.URL
		} else if btn.Phone != "" {
			button["type"] = "phone_number"
			button["payload"] = btn.Phone
		} else {
			button["payload"] = btn.ID
		}

		igButtons = append(igButtons, button)
	}

	return igButtons
}

// ============================================================================
// Webhook Processing
// ============================================================================

// extractIncomingMessage extracts and converts Instagram webhook to IncomingMessage
func (a *InstagramAdapter) extractIncomingMessage(webhook InstagramWebhook) (*channels.IncomingMessage, error) {
	if webhook.Object != "instagram" {
		return nil, fmt.Errorf("unexpected webhook object type: %s", webhook.Object)
	}

	for _, entry := range webhook.Entry {
		for _, messaging := range entry.Messaging {
			// Skip if this is an echo (message we sent)
			if messaging.Message != nil && messaging.Message.IsEcho {
				continue
			}

			// Process regular message
			if messaging.Message != nil {
				return a.processMessage(messaging)
			}

			// Process postback (button clicks)
			if messaging.Postback != nil {
				return a.processPostback(messaging)
			}

			// Process reactions
			if messaging.Reaction != nil {
				return a.processReaction(messaging)
			}
		}
	}

	return nil, nil // No message found
}

// processMessage processes a regular Instagram message
func (a *InstagramAdapter) processMessage(messaging WebhookMessaging) (*channels.IncomingMessage, error) {
	msg := messaging.Message

	incomingMsg := &channels.IncomingMessage{
		MessageID: kernel.MessageID(msg.Mid),
		ChannelID: kernel.NewChannelID(messaging.Recipient.ID),
		SenderID:  messaging.Sender.ID,
		Content: channels.MessageContent{
			Type: "text",
		},
		Timestamp: messaging.Timestamp,
		Metadata: map[string]any{
			"instagram_message_id": msg.Mid,
			"page_id":              messaging.Recipient.ID,
		},
	}

	// Extract content based on message type
	if msg.Text != "" {
		incomingMsg.Content.Text = msg.Text
	} else if len(msg.Attachments) > 0 {
		// Handle attachments (images, videos, etc.)
		attachment := msg.Attachments[0]
		incomingMsg.Content.Type = attachment.Type
		incomingMsg.Content.MediaURL = attachment.Payload.URL

		incomingMsg.Metadata["attachment_type"] = attachment.Type
	}

	// Handle quick reply
	if msg.QuickReply != nil {
		incomingMsg.Metadata["quick_reply_payload"] = msg.QuickReply.Payload
	}

	return incomingMsg, nil
}

// processPostback processes button postback events
func (a *InstagramAdapter) processPostback(messaging WebhookMessaging) (*channels.IncomingMessage, error) {
	postback := messaging.Postback

	return &channels.IncomingMessage{
		MessageID: kernel.MessageID(fmt.Sprintf("postback_%d", messaging.Timestamp)),
		ChannelID: kernel.NewChannelID(messaging.Recipient.ID),
		SenderID:  messaging.Sender.ID,
		Content: channels.MessageContent{
			Type: "postback",
			Text: postback.Title,
		},
		Timestamp: messaging.Timestamp,
		Metadata: map[string]any{
			"postback_payload": postback.Payload,
			"page_id":          messaging.Recipient.ID,
		},
	}, nil
}

// processReaction processes message reactions
func (a *InstagramAdapter) processReaction(messaging WebhookMessaging) (*channels.IncomingMessage, error) {
	reaction := messaging.Reaction

	return &channels.IncomingMessage{
		MessageID: kernel.MessageID(fmt.Sprintf("reaction_%d", messaging.Timestamp)),
		ChannelID: kernel.NewChannelID(messaging.Recipient.ID),
		SenderID:  messaging.Sender.ID,
		Content: channels.MessageContent{
			Type: "reaction",
			Text: reaction.Emoji,
		},
		Timestamp: messaging.Timestamp,
		Metadata: map[string]any{
			"reaction_emoji":     reaction.Emoji,
			"reaction_action":    reaction.Action,
			"reacted_message_id": reaction.Mid,
			"page_id":            messaging.Recipient.ID,
		},
	}, nil
}

// ============================================================================
// Security & Validation
// ============================================================================

// verifySignature verifies the Instagram webhook signature using HMAC-SHA256
//
// Instagram signs webhooks with the app secret to ensure authenticity
func (a *InstagramAdapter) verifySignature(payload []byte, headers map[string]string) error {
	if a.config.AppSecret == "" {
		log.Printf("⚠️  Instagram app secret not configured, skipping signature verification")
		return nil // Skip verification if no secret configured
	}

	// Get signature from headers (try both cases)
	signature := headers["X-Hub-Signature-256"]
	if signature == "" {
		signature = headers["x-hub-signature-256"]
	}

	if signature == "" {
		return channels.ErrInvalidWebhookSignature().
			WithDetail("reason", "missing X-Hub-Signature-256 header")
	}

	// Remove "sha256=" prefix
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate expected signature using HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(a.config.AppSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant-time comparison
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return channels.ErrInvalidWebhookSignature().
			WithDetail("reason", "signature mismatch")
	}

	return nil
}

// parseAPIError parses Instagram API error responses
func (a *InstagramAdapter) parseAPIError(statusCode int, body []byte) error {
	var apiError struct {
		Error struct {
			Message      string `json:"message"`
			Type         string `json:"type"`
			Code         int    `json:"code"`
			ErrorSubcode int    `json:"error_subcode"`
			FBTraceID    string `json:"fbtrace_id"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &apiError); err != nil {
		return channels.ErrProviderAPIError().
			WithDetail("status", statusCode).
			WithDetail("body", string(body))
	}

	return channels.ErrProviderAPIError().
		WithDetail("status", statusCode).
		WithDetail("error_type", apiError.Error.Type).
		WithDetail("error_code", apiError.Error.Code).
		WithDetail("error_message", apiError.Error.Message).
		WithDetail("trace_id", apiError.Error.FBTraceID)
}

// ============================================================================
// Instagram Webhook Data Structures
// ============================================================================

// InstagramWebhook represents the top-level Instagram webhook payload
type InstagramWebhook struct {
	Object string                  `json:"object"` // Should be "instagram"
	Entry  []InstagramWebhookEntry `json:"entry"`
}

// InstagramWebhookEntry represents an entry in the webhook
type InstagramWebhookEntry struct {
	ID        string             `json:"id"`   // Page ID
	Time      int64              `json:"time"` // Timestamp
	Messaging []WebhookMessaging `json:"messaging"`
}

// WebhookMessaging represents a messaging event
type WebhookMessaging struct {
	Sender    WebhookUser      `json:"sender"`
	Recipient WebhookUser      `json:"recipient"`
	Timestamp int64            `json:"timestamp"`
	Message   *WebhookMessage  `json:"message,omitempty"`
	Postback  *WebhookPostback `json:"postback,omitempty"`
	Reaction  *WebhookReaction `json:"reaction,omitempty"`
	Read      *WebhookRead     `json:"read,omitempty"`
	Delivery  *WebhookDelivery `json:"delivery,omitempty"`
}

// WebhookUser represents a user (sender or recipient)
type WebhookUser struct {
	ID string `json:"id"`
}

// WebhookMessage represents an incoming Instagram message
type WebhookMessage struct {
	Mid         string              `json:"mid"`
	Text        string              `json:"text,omitempty"`
	Attachments []WebhookAttachment `json:"attachments,omitempty"`
	QuickReply  *WebhookQuickReply  `json:"quick_reply,omitempty"`
	ReplyTo     *WebhookReplyTo     `json:"reply_to,omitempty"`
	IsEcho      bool                `json:"is_echo,omitempty"`
}

// WebhookAttachment represents a media attachment
type WebhookAttachment struct {
	Type    string            `json:"type"` // image, video, audio, file
	Payload AttachmentPayload `json:"payload"`
}

// AttachmentPayload contains attachment details
type AttachmentPayload struct {
	URL string `json:"url"`
}

// WebhookQuickReply represents a quick reply interaction
type WebhookQuickReply struct {
	Payload string `json:"payload"`
}

// WebhookReplyTo represents a message reply context
type WebhookReplyTo struct {
	Mid string `json:"mid"`
}

// WebhookPostback represents a button postback event
type WebhookPostback struct {
	Mid     string `json:"mid,omitempty"`
	Title   string `json:"title"`
	Payload string `json:"payload"`
}

// WebhookReaction represents a message reaction
type WebhookReaction struct {
	Mid    string `json:"mid"`    // Message ID being reacted to
	Action string `json:"action"` // "react" or "unreact"
	Emoji  string `json:"emoji"`
}

// WebhookRead represents a read receipt
type WebhookRead struct {
	Watermark int64 `json:"watermark"`
}

// WebhookDelivery represents a delivery confirmation
type WebhookDelivery struct {
	Mids      []string `json:"mids"`
	Watermark int64    `json:"watermark"`
}

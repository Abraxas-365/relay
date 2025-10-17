package whatsapp

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
	whatsappAPIBaseURL = "https://graph.facebook.com"
	defaultAPIVersion  = "v24.0"
)

// WhatsAppAdapter implements ChannelAdapter for WhatsApp Business API
type WhatsAppAdapter struct {
	config        channels.WhatsAppConfig
	httpClient    *http.Client
	bufferService *BufferService
	apiURL        string
}

// NewWhatsAppAdapter creates a new WhatsApp adapter
func NewWhatsAppAdapter(config channels.WhatsAppConfig, redisClient *redis.Client) *WhatsAppAdapter {
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = defaultAPIVersion
	}

	return &WhatsAppAdapter{
		config:        config,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		bufferService: NewBufferService(redisClient, config),
		apiURL:        fmt.Sprintf("%s/%s/%s", whatsappAPIBaseURL, apiVersion, config.PhoneNumberID),
	}
}

// GetType returns the channel type
func (a *WhatsAppAdapter) GetType() channels.ChannelType {
	return channels.ChannelTypeWhatsApp
}

// SendMessage sends a message via WhatsApp
func (a *WhatsAppAdapter) SendMessage(ctx context.Context, msg channels.OutgoingMessage) error {
	// Build WhatsApp API payload
	payload := a.buildMessagePayload(msg)

	// Build URL using the pre-configured apiURL
	url := fmt.Sprintf("%s/messages", a.apiURL)

	// ✅ LOG THE ACTUAL URL BEING CALLED
	log.Printf("🌐 WhatsApp API URL: %s", url)
	log.Printf("📦 Payload: %+v", payload)
	log.Printf("🔑 Token (first 20 chars): %s...", a.config.AccessToken[:20])

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+a.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("❌ WhatsApp API Error - Status: %d, Body: %s", resp.StatusCode, string(body))
		return fmt.Errorf("whatsapp API error %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("✅ WhatsApp message sent successfully - Response: %s", string(body))
	return nil
}

// ValidateConfig validates the WhatsApp configuration
func (a *WhatsAppAdapter) ValidateConfig(config channels.ChannelConfig) error {
	whatsappConfig, ok := config.(channels.WhatsAppConfig)
	if !ok {
		return channels.ErrInvalidChannelConfig().WithDetail("reason", "invalid config type")
	}

	return whatsappConfig.Validate()
}

// ProcessWebhook processes incoming WhatsApp webhooks WITH BUFFERING
func (a *WhatsAppAdapter) ProcessWebhook(
	ctx context.Context,
	payload []byte,
	headers map[string]string,
) (*channels.IncomingMessage, error) {
	// Verify signature
	if err := a.verifySignature(payload, headers); err != nil {
		return nil, err
	}

	// Parse webhook
	var webhook WhatsAppWebhook
	if err := json.Unmarshal(payload, &webhook); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Extract message from webhook
	incomingMsg, err := a.extractIncomingMessage(webhook)
	if err != nil {
		return nil, err
	}

	if incomingMsg == nil {
		return nil, nil // No message (status update, etc.)
	}

	// Add to buffer
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
		return nil, nil
	}

	// Message should be processed immediately
	return processedMsg, nil
}

// GetFeatures returns WhatsApp channel features
func (a *WhatsAppAdapter) GetFeatures() channels.ChannelFeatures {
	return a.config.GetFeatures()
}

// TestConnection tests the WhatsApp API connection
func (a *WhatsAppAdapter) TestConnection(ctx context.Context, config channels.ChannelConfig) error {
	whatsappConfig, ok := config.(channels.WhatsAppConfig)
	if !ok {
		return channels.ErrInvalidChannelConfig()
	}

	// Use configured API version or default
	apiVersion := whatsappConfig.APIVersion
	if apiVersion == "" {
		apiVersion = defaultAPIVersion
	}

	// Test by fetching phone number info
	url := fmt.Sprintf("%s/%s/%s",
		whatsappAPIBaseURL,
		apiVersion,
		whatsappConfig.PhoneNumberID,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+whatsappConfig.AccessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return channels.ErrProviderAPIError().WithCause(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return channels.ErrProviderAuthFailed().
			WithDetail("status", resp.StatusCode).
			WithDetail("response", string(body))
	}

	return nil
}

// buildMessagePayload builds WhatsApp API payload
func (a *WhatsAppAdapter) buildMessagePayload(msg channels.OutgoingMessage) map[string]any {
	payload := map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                msg.RecipientID,
	}

	// Handle different content types
	if msg.Content.Type == "text" {
		payload["type"] = "text"
		payload["text"] = map[string]any{
			"body": msg.Content.Text,
		}
	} else if msg.Content.Type == "template" && msg.TemplateID != "" {
		payload["type"] = "template"
		payload["template"] = a.buildTemplatePayload(msg)
	}
	// Add more content types as needed

	return payload
}

// buildTemplatePayload builds template message payload
func (a *WhatsAppAdapter) buildTemplatePayload(msg channels.OutgoingMessage) map[string]any {
	template := map[string]any{
		"name":     msg.TemplateID,
		"language": map[string]string{"code": "en"},
	}

	if len(msg.Variables) > 0 {
		components := []map[string]any{}
		parameters := []map[string]any{}

		for _, value := range msg.Variables {
			parameters = append(parameters, map[string]any{
				"type": "text",
				"text": value,
			})
		}

		components = append(components, map[string]any{
			"type":       "body",
			"parameters": parameters,
		})

		template["components"] = components
	}

	return template
}

// verifySignature verifies WhatsApp webhook signature
func (a *WhatsAppAdapter) verifySignature(payload []byte, headers map[string]string) error {
	if a.config.AppSecret == "" {
		return nil // Skip verification if no secret configured
	}

	signature := headers["X-Hub-Signature-256"]
	if signature == "" {
		signature = headers["x-hub-signature-256"]
	}

	if signature == "" {
		return channels.ErrInvalidWebhookSignature()
	}

	// Remove "sha256=" prefix
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(a.config.AppSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return channels.ErrInvalidWebhookSignature()
	}

	return nil
}

// extractIncomingMessage extracts message from webhook
func (a *WhatsAppAdapter) extractIncomingMessage(webhook WhatsAppWebhook) (*channels.IncomingMessage, error) {
	for _, entry := range webhook.Entry {
		for _, change := range entry.Changes {
			if change.Value.MessagingProduct != "whatsapp" {
				continue
			}

			for _, msg := range change.Value.Messages {
				return &channels.IncomingMessage{
					MessageID: msg.ID,
					ChannelID: kernel.NewChannelID(a.config.PhoneNumberID),
					SenderID:  msg.From,
					Content: channels.MessageContent{
						Type: msg.Type,
						Text: a.extractText(msg),
					},
					Timestamp: msg.Timestamp,
					Metadata: map[string]any{
						"whatsapp_message_id": msg.ID,
					},
				}, nil
			}
		}
	}

	return nil, nil // No message found
}

// extractText extracts text from message
func (a *WhatsAppAdapter) extractText(msg WebhookMessage) string {
	if msg.Text != nil {
		return msg.Text.Body
	}
	if msg.Image != nil && msg.Image.Caption != "" {
		return msg.Image.Caption
	}
	return ""
}

// WhatsApp webhook structures
type WhatsAppWebhook struct {
	Object string         `json:"object"`
	Entry  []WebhookEntry `json:"entry"`
}

type WebhookEntry struct {
	ID      string          `json:"id"`
	Changes []WebhookChange `json:"changes"`
}

type WebhookChange struct {
	Value WebhookValue `json:"value"`
	Field string       `json:"field"`
}

type WebhookValue struct {
	MessagingProduct string           `json:"messaging_product"`
	Metadata         WebhookMetadata  `json:"metadata"`
	Messages         []WebhookMessage `json:"messages"`
	Statuses         []WebhookStatus  `json:"statuses"`
}

type WebhookMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type WebhookMessage struct {
	ID        kernel.MessageID `json:"id"`
	From      string           `json:"from"`
	Timestamp int64            `json:"timestamp,string"`
	Type      string           `json:"type"`
	Text      *WebhookText     `json:"text,omitempty"`
	Image     *WebhookMedia    `json:"image,omitempty"`
	Document  *WebhookMedia    `json:"document,omitempty"`
	Audio     *WebhookMedia    `json:"audio,omitempty"`
	Video     *WebhookMedia    `json:"video,omitempty"`
}

type WebhookText struct {
	Body string `json:"body"`
}

type WebhookMedia struct {
	ID       string `json:"id"`
	MimeType string `json:"mime_type"`
	SHA256   string `json:"sha256"`
	Caption  string `json:"caption,omitempty"`
}

type WebhookStatus struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Timestamp   int64  `json:"timestamp,string"`
	RecipientID string `json:"recipient_id"`
}

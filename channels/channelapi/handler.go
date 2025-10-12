package channelapi

import (
	"context"
	"log"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// MessageProcessor interface for processing messages through the engine
type MessageProcessor interface {
	ProcessMessage(ctx context.Context, msg engine.Message) error
}

// ChannelHandler handles generic channel operations
type ChannelHandler struct {
	messageProcessor MessageProcessor
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(messageProcessor MessageProcessor) *ChannelHandler {
	return &ChannelHandler{
		messageProcessor: messageProcessor,
	}
}

// ProcessIncomingMessage processes incoming messages from ANY channel
// This handler expects incoming_message and channel in fiber.Locals
func (h *ChannelHandler) ProcessIncomingMessage(c *fiber.Ctx) error {
	// Get message from context (set by channel-specific handler)
	incomingMsg, ok := c.Locals("incoming_message").(*channels.IncomingMessage)
	if !ok || incomingMsg == nil {
		log.Printf("❌ No incoming message in context")
		return c.SendStatus(fiber.StatusOK)
	}

	// Get channel from context
	channel, ok := c.Locals("channel").(*channels.Channel)
	if !ok || channel == nil {
		log.Printf("❌ No channel in context")
		return c.SendStatus(fiber.StatusOK)
	}

	log.Printf("📨 Processing incoming message from %s via channel %s", incomingMsg.SenderID, channel.Name)

	// Transform channels.IncomingMessage → engine.Message
	engineMsg := h.transformToEngineMessage(channel, incomingMsg)

	// Process message through the engine (asynchronously to not block webhook)
	go func() {
		if err := h.messageProcessor.ProcessMessage(context.Background(), engineMsg); err != nil {
			log.Printf("❌ Failed to process message through engine: %v", err)
		} else {
			log.Printf("✅ Message processed successfully: %s", engineMsg.ID.String())
		}
	}()

	// Respond to webhook immediately
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":     "received",
		"message_id": engineMsg.ID.String(),
	})
}

// transformToEngineMessage converts channels.IncomingMessage to engine.Message
func (h *ChannelHandler) transformToEngineMessage(
	channel *channels.Channel,
	incomingMsg *channels.IncomingMessage,
) engine.Message {
	// Build metadata for engine message content
	contentMetadata := make(map[string]any)

	// Add original metadata
	if incomingMsg.Content.Metadata != nil {
		for k, v := range incomingMsg.Content.Metadata {
			contentMetadata[k] = v
		}
	}

	// Add additional fields as metadata
	if incomingMsg.Content.MediaURL != "" {
		contentMetadata["media_url"] = incomingMsg.Content.MediaURL
	}
	if incomingMsg.Content.Caption != "" {
		contentMetadata["caption"] = incomingMsg.Content.Caption
	}
	if incomingMsg.Content.MimeType != "" {
		contentMetadata["mime_type"] = incomingMsg.Content.MimeType
	}
	if incomingMsg.Content.Filename != "" {
		contentMetadata["filename"] = incomingMsg.Content.Filename
	}

	// Add location if present
	if incomingMsg.Content.Location != nil {
		contentMetadata["location"] = map[string]any{
			"latitude":  incomingMsg.Content.Location.Latitude,
			"longitude": incomingMsg.Content.Location.Longitude,
			"name":      incomingMsg.Content.Location.Name,
			"address":   incomingMsg.Content.Location.Address,
		}
	}

	// Add contact if present
	if incomingMsg.Content.Contact != nil {
		contentMetadata["contact"] = map[string]any{
			"name":         incomingMsg.Content.Contact.Name,
			"phone_number": incomingMsg.Content.Contact.PhoneNumber,
			"email":        incomingMsg.Content.Contact.Email,
			"organization": incomingMsg.Content.Contact.Organization,
		}
	}

	// Add interactive if present
	if incomingMsg.Content.Interactive != nil {
		contentMetadata["interactive"] = incomingMsg.Content.Interactive
	}

	// Add attachments as metadata (detailed)
	if len(incomingMsg.Content.Attachments) > 0 {
		attachmentDetails := make([]map[string]any, len(incomingMsg.Content.Attachments))
		for i, att := range incomingMsg.Content.Attachments {
			attachmentDetails[i] = map[string]any{
				"type":      att.Type,
				"url":       att.URL,
				"mime_type": att.MimeType,
				"filename":  att.Filename,
				"size":      att.Size,
				"caption":   att.Caption,
			}
		}
		contentMetadata["attachment_details"] = attachmentDetails
	}

	// Build context metadata for engine message
	contextMetadata := make(map[string]any)
	contextMetadata["original_message_id"] = incomingMsg.MessageID.String()
	contextMetadata["channel_type"] = string(channel.Type)
	contextMetadata["channel_name"] = channel.Name
	contextMetadata["provider"] = channel.GetProvider()
	contextMetadata["raw_timestamp"] = incomingMsg.Timestamp

	// Add original message metadata
	if incomingMsg.Metadata != nil {
		for k, v := range incomingMsg.Metadata {
			contextMetadata[k] = v
		}
	}

	return engine.Message{
		ID:        kernel.NewMessageID(uuid.NewString()),
		TenantID:  channel.TenantID,
		ChannelID: channel.ID,
		SenderID:  incomingMsg.SenderID,
		Content: engine.MessageContent{
			Type:        incomingMsg.Content.Type,
			Text:        incomingMsg.Content.Text,
			Attachments: h.extractAttachmentURLs(incomingMsg.Content),
			Metadata:    contentMetadata,
		},
		Context:   contextMetadata,
		Status:    engine.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// extractAttachmentURLs extracts attachment URLs as []string for engine.MessageContent
func (h *ChannelHandler) extractAttachmentURLs(content channels.MessageContent) []string {
	urls := make([]string, 0)

	// Add media URL if present
	if content.MediaURL != "" {
		urls = append(urls, content.MediaURL)
	}

	// Add attachment URLs
	for _, att := range content.Attachments {
		if att.URL != "" {
			urls = append(urls, att.URL)
		}
	}

	return urls
}

package channelapi

import (
	"context"
	"log"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine/triggerhandler"
	"github.com/gofiber/fiber/v2"
)

// ChannelHandler handles generic channel operations
type ChannelHandler struct {
	triggerHandler *triggerhandler.TriggerHandler
}

// NewChannelHandler creates a new channel handler
func NewChannelHandler(triggerHandler *triggerhandler.TriggerHandler) *ChannelHandler {
	return &ChannelHandler{
		triggerHandler: triggerHandler,
	}
}

// ProcessIncomingMessage processes incoming messages from ANY channel
func (h *ChannelHandler) ProcessIncomingMessage(c *fiber.Ctx) error {
	// Get message from context (set by channel-specific handler)
	incomingMsg, ok := c.Locals("incoming_message").(*channels.IncomingMessage)
	if !ok || incomingMsg == nil {
		log.Printf("⚠️ No incoming message in context")
		return c.SendStatus(fiber.StatusOK)
	}

	// Get channel from context
	channel, ok := c.Locals("channel").(*channels.Channel)
	if !ok || channel == nil {
		log.Printf("⚠️ No channel in context")
		return c.SendStatus(fiber.StatusOK)
	}

	log.Printf("📨 Processing incoming message from %s via channel %s",
		incomingMsg.SenderID, channel.Name)

	// Prepare trigger data
	triggerData := map[string]any{
		"text":            incomingMsg.Content.Text,
		"message_id":      incomingMsg.MessageID.String(),
		"channel_id":      channel.ID.String(),
		"sender_id":       incomingMsg.SenderID,
		"message_type":    incomingMsg.Content.Type,
		"conversation_id": incomingMsg.SenderID, // For AI memory
	}

	// Add attachments
	if len(incomingMsg.Content.Attachments) > 0 {
		attachments := make([]map[string]any, len(incomingMsg.Content.Attachments))
		for i, att := range incomingMsg.Content.Attachments {
			attachments[i] = map[string]any{
				"type":      att.Type,
				"url":       att.URL,
				"mime_type": att.MimeType,
				"filename":  att.Filename,
			}
		}
		triggerData["attachments"] = attachments
	}

	// Add metadata
	if incomingMsg.Metadata != nil {
		triggerData["metadata"] = incomingMsg.Metadata
	}

	// ✅ FIX: Create independent context for goroutine
	// DO NOT use c.Context() - it gets cancelled when HTTP request ends
	workflowCtx := context.Background()

	// Trigger workflows (async)
	go func() {
		log.Printf("🔔 Triggering workflow for channel %s, sender %s",
			channel.ID.String(), incomingMsg.SenderID)

		// ✅ Use workflowCtx instead of c.Context()
		if err := h.triggerHandler.HandleChannelWebhookTrigger(
			workflowCtx, // ← FIX: Use background context
			channel.TenantID,
			channel.ID,
			triggerData,
		); err != nil {
			log.Printf("❌ Failed to trigger workflows: %v", err)
		}
	}()

	// Respond immediately
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "received",
	})
}

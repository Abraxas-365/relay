package whatsapp

import (
	"log"
	"net/http"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/gofiber/fiber/v2"
)

// WebhookHandler handles WhatsApp-specific webhook operations
type WebhookHandler struct {
	channelRepo channels.ChannelRepository
	adapter     *WhatsAppAdapter
}

// NewWebhookHandler creates a new WhatsApp webhook handler
func NewWebhookHandler(
	channelRepo channels.ChannelRepository,
	adapter *WhatsAppAdapter,
) *WebhookHandler {
	return &WebhookHandler{
		channelRepo: channelRepo,
		adapter:     adapter,
	}
}

// VerifyWebhook handles Meta's webhook verification challenge
// GET /webhooks/whatsapp/:tenantId/:channelId
func (h *WebhookHandler) VerifyWebhook(c *fiber.Ctx) error {
	tenantID := kernel.TenantID(c.Params("tenantId"))
	channelID := kernel.NewChannelID(c.Params("channelId"))

	log.Printf("🔐 Verifying WhatsApp webhook - Tenant: %s, Channel: %s", tenantID, channelID)

	// Get channel to verify it exists and get verify token
	channel, err := h.channelRepo.FindByID(c.Context(), channelID, tenantID)
	if err != nil {
		log.Printf("❌ Channel not found: %s (tenant: %s)", channelID, tenantID)
		return fiber.NewError(http.StatusNotFound, "Channel not found")
	}

	// Parse channel config
	config, err := channel.GetConfigStruct()
	if err != nil {
		log.Printf("❌ Invalid channel config: %v", err)
		return fiber.NewError(http.StatusInternalServerError, "Invalid channel config")
	}

	whatsappConfig, ok := config.(channels.WhatsAppConfig)
	if !ok {
		return fiber.NewError(http.StatusBadRequest, "Not a WhatsApp channel")
	}

	// Extract verification parameters from query
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	// Verify the token matches
	if mode == "subscribe" && token == whatsappConfig.WebhookVerifyToken {
		log.Printf("✅ Webhook verified successfully for channel: %s", channelID)
		return c.SendString(challenge)
	}

	log.Printf("❌ Webhook verification failed - Invalid token for channel: %s", channelID)
	return fiber.NewError(http.StatusForbidden, "Verification failed")
}

// ReceiveWebhook handles incoming WhatsApp webhook (parsing only)
// POST /webhooks/whatsapp/:tenantId/:channelId
func (h *WebhookHandler) ReceiveWebhook(c *fiber.Ctx) error {
	tenantID := kernel.TenantID(c.Params("tenantId"))
	channelID := kernel.NewChannelID(c.Params("channelId"))

	log.Printf("📥 Received WhatsApp webhook - Tenant: %s, Channel: %s", tenantID, channelID)

	// Get channel
	channel, err := h.channelRepo.FindByID(c.Context(), channelID, tenantID)
	if err != nil {
		log.Printf("❌ Channel not found: %s", channelID)
		// Return 200 to prevent Meta from retrying
		return c.SendStatus(fiber.StatusOK)
	}

	// Check if channel is active
	if !channel.IsActive {
		log.Printf("⚠️  Channel is inactive: %s", channelID)
		return c.SendStatus(fiber.StatusOK)
	}

	// Get config
	config, err := channel.GetConfigStruct()
	if err != nil {
		log.Printf("❌ Invalid channel config: %v", err)
		return c.SendStatus(fiber.StatusOK)
	}

	whatsappConfig, ok := config.(channels.WhatsAppConfig)
	if !ok {
		log.Printf("❌ Not a WhatsApp channel: %s", channelID)
		return c.SendStatus(fiber.StatusOK)
	}

	// Create adapter instance with this channel's config
	adapter := NewWhatsAppAdapter(whatsappConfig, h.adapter.bufferService.redis)

	// Read payload
	body := c.Body()

	// Extract headers
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// Process webhook using adapter (WhatsApp-specific parsing)
	incomingMsg, err := adapter.ProcessWebhook(c.Context(), body, headers)
	if err != nil {
		log.Printf("❌ Failed to process webhook: %v", err)
		// Return 200 to prevent Meta from retrying
		return c.SendStatus(fiber.StatusOK)
	}

	// If message is nil, it means it's buffered or not a message event
	if incomingMsg == nil {
		log.Printf("📦 Message buffered or status update for channel: %s", channelID)
		return c.SendStatus(fiber.StatusOK)
	}

	// Store parsed message in context for the next handler
	c.Locals("incoming_message", incomingMsg)
	c.Locals("channel", channel)

	// Continue to next handler (generic message processor)
	return c.Next()
}


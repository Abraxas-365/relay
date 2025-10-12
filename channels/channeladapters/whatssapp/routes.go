package whatsapp

import (
	"github.com/gofiber/fiber/v2"
)

// WebhookRoutes handles WhatsApp webhook route setup
type WebhookRoutes struct {
	handler               *WebhookHandler
	messageProcessHandler fiber.Handler // Generic handler from channelapi
}

// NewWebhookRoutes creates a new webhook routes instance
func NewWebhookRoutes(
	handler *WebhookHandler,
	messageProcessHandler fiber.Handler,
) *WebhookRoutes {
	return &WebhookRoutes{
		handler:               handler,
		messageProcessHandler: messageProcessHandler,
	}
}

// Setup configures WhatsApp webhook routes
func (wr *WebhookRoutes) RegisterRoutes(app *fiber.App) {
	webhooks := app.Group("/webhooks/whatsapp")

	// Verification endpoint (GET)
	webhooks.Get("/:tenantId/:channelId", wr.handler.VerifyWebhook)

	// Receiving endpoint (POST) with chained handlers
	webhooks.Post("/:tenantId/:channelId",
		wr.handler.ReceiveWebhook, // Parse WhatsApp webhook
		wr.messageProcessHandler,  // Process generic message
	)
}

package instagram

import (
	"github.com/gofiber/fiber/v2"
)

// WebhookRoutes handles Instagram webhook route setup
// It configures the HTTP endpoints for Instagram webhook verification and message receiving
type WebhookRoutes struct {
	handler               *WebhookHandler
	messageProcessHandler fiber.Handler // Generic handler from channelapi
}

// NewWebhookRoutes creates a new webhook routes instance
//
// Parameters:
//   - handler: Instagram-specific webhook handler
//   - messageProcessHandler: Generic message processor that handles parsed messages
//
// Returns:
//   - *WebhookRoutes: Configured routes instance ready to register
func NewWebhookRoutes(
	handler *WebhookHandler,
	messageProcessHandler fiber.Handler,
) *WebhookRoutes {
	return &WebhookRoutes{
		handler:               handler,
		messageProcessHandler: messageProcessHandler,
	}
}

// RegisterRoutes configures Instagram webhook routes on the Fiber app
//
// Routes registered:
//   - GET  /webhooks/instagram/:tenantId/:channelId - Webhook verification
//   - POST /webhooks/instagram/:tenantId/:channelId - Webhook receiving
//
// The POST route uses chained handlers:
//  1. Instagram-specific parsing (handler.ReceiveWebhook)
//  2. Generic message processing (messageProcessHandler)
//
// This separation allows the Instagram adapter to focus on parsing Instagram's
// specific webhook format, while the generic handler manages business logic
// like storing messages, triggering workflows, etc.
//
// Parameters:
//   - app: Fiber application instance
func (wr *WebhookRoutes) RegisterRoutes(app *fiber.App) {
	// Create Instagram webhook group
	webhooks := app.Group("/webhooks/instagram")

	// Verification endpoint (GET) - Meta sends this during webhook setup
	webhooks.Get("/:tenantId/:channelId", wr.handler.VerifyWebhook)

	// Receiving endpoint (POST) with chained handlers
	// 1. Parse Instagram webhook format
	// 2. Process the parsed message generically
	webhooks.Post("/:tenantId/:channelId",
		wr.handler.ReceiveWebhook, // Parse Instagram webhook
		wr.messageProcessHandler,  // Process generic message
	)
}

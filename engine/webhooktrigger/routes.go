package webhooktrigger

import (
	"github.com/gofiber/fiber/v2"
)

type WebhookTriggerRoutes struct {
	handler *WebhookTriggerHandler
}

func NewWebhookTriggerRoutes(handler *WebhookTriggerHandler) *WebhookTriggerRoutes {
	return &WebhookTriggerRoutes{
		handler: handler,
	}
}

func (r *WebhookTriggerRoutes) RegisterRoutes(app *fiber.App) {
	webhooks := app.Group("/webhooks/trigger")

	// GET for verification (optional, for webhook providers that need verification)
	webhooks.Get("/:tenantId/:workflowId", r.handler.VerifyWebhook)

	// POST for triggering workflow
	webhooks.Post("/:tenantId/:workflowId", r.handler.HandleWebhook)

	// TEST endpoint (dry run)
	webhooks.Post("/:tenantId/:workflowId/test", r.handler.TestWebhook)

	// Support other HTTP methods if needed
	webhooks.Put("/:tenantId/:workflowId", r.handler.HandleWebhook)
	webhooks.Patch("/:tenantId/:workflowId", r.handler.HandleWebhook)
}

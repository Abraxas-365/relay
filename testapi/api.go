package testapi

import (
	"github.com/gofiber/fiber/v2"
)

// TestRoutes configura las rutas de testing
type TestRoutes struct {
	handler *TestHandler
}

// NewTestRoutes crea una nueva instancia
func NewTestRoutes(handler *TestHandler) *TestRoutes {
	return &TestRoutes{
		handler: handler,
	}
}

// Setup configura todas las rutas de testing
func (tr *TestRoutes) Setup(app *fiber.App) {
	// Grupo base de testing (público para desarrollo)
	test := app.Group("/test")

	// ==========================================
	// HEALTH & INFO ROUTES
	// ==========================================

	// Health check
	test.Get("/health", tr.handler.HealthCheck)

	// Instructions
	test.Get("/", tr.handler.GetTestInstructions)
	test.Get("/instructions", tr.handler.GetTestInstructions)

	// ==========================================
	// MESSAGE ROUTES
	// ==========================================

	// Enviar mensaje de prueba
	test.Post("/message", tr.handler.SendTestMessage)

	// Información del canal
	test.Get("/channel/:channelId", tr.handler.GetChannelInfo)

	// ==========================================
	// QUICK TEST ROUTES
	// ==========================================

	// Quick test - ejemplo rápido sin body
	test.Get("/quick/:text", func(c *fiber.Ctx) error {
		text := c.Params("text")
		return c.JSON(fiber.Map{
			"info": "This is a preview. Use POST /test/message for actual testing",
			"example_request": fiber.Map{
				"method": "POST",
				"url":    "/test/message",
				"body": fiber.Map{
					"channel_id": "test-ch-001",
					"sender_id":  "test-user",
					"text":       text,
				},
			},
			"curl": `curl -X POST http://localhost:8080/test/message \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "test-ch-1", "sender_id": "user1", "text": "` + text + `"}'`,
		})
	})
}

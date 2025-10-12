package testapi

import (
	"log"
	"time"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TestHandler maneja las peticiones HTTP para testing
type TestHandler struct {
	messageProcessor engine.MessageProcessor
}

// NewTestHandler crea un nuevo handler de test
func NewTestHandler(messageProcessor engine.MessageProcessor) *TestHandler {
	return &TestHandler{
		messageProcessor: messageProcessor,
	}
}

// SendTestMessage envía un mensaje de prueba
// POST /test/message
func (h *TestHandler) SendTestMessage(c *fiber.Ctx) error {
	var req struct {
		ChannelID string `json:"channel_id" validate:"required"`
		SenderID  string `json:"sender_id" validate:"required"`
		Text      string `json:"text" validate:"required"`
		TenantID  string `json:"tenant_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	log.Printf("📨 [TEST CHANNEL] Received message: '%s' from %s", req.Text, req.SenderID)

	// Create message
	msg := engine.Message{
		ID:        kernel.NewMessageID(uuid.NewString()),
		TenantID:  kernel.NewTenantID("tenant-test-001"),
		ChannelID: kernel.NewChannelID(req.ChannelID),
		SenderID:  req.SenderID,
		Content: engine.MessageContent{
			Type: "text",
			Text: req.Text,
		},
		Context:   nil,
		Status:    engine.MessageStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	log.Printf("🔍 Created message: %+v", msg)

	// Process message through the engine
	if err := h.messageProcessor.ProcessMessage(c.Context(), msg); err != nil {
		log.Printf("❌ Failed to process message: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to process message",
			"details": err.Error(),
		})
	}

	log.Printf("✅ Message processed successfully: %s", msg.ID.String())

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":    true,
		"message_id": msg.ID.String(),
		"channel_id": req.ChannelID,
		"sender_id":  req.SenderID,
		"text":       req.Text,
		"status":     "processed",
		"timestamp":  time.Now().Unix(),
	})
}

// GetChannelInfo obtiene información del canal de prueba
// GET /test/channel/:channelId
func (h *TestHandler) GetChannelInfo(c *fiber.Ctx) error {
	channelID := c.Params("channelId")
	if channelID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Channel ID is required",
		})
	}

	return c.JSON(fiber.Map{
		"channel_id":   channelID,
		"type":         "TEST_HTTP",
		"status":       "active",
		"description":  "Test channel for development and testing",
		"webhook_path": "/test/message",
		"examples": fiber.Map{
			"hi":    "Responds with 'hi'",
			"hello": "Responds with custom greeting",
		},
	})
}

// HealthCheck verifica el estado del sistema de testing
// GET /test/health
func (h *TestHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"service":   "test-channel",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}

// GetTestInstructions muestra instrucciones de uso
// GET /test/instructions
func (h *TestHandler) GetTestInstructions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"service": "Test Channel API",
		"endpoints": map[string]any{
			"POST /test/message": map[string]any{
				"description": "Send a test message",
				"body": map[string]string{
					"channel_id": "your-channel-id",
					"sender_id":  "test-user-123",
					"text":       "hi or hello",
					"tenant_id":  "optional-tenant-id",
				},
			},
			"GET /test/channel/:channelId": "Get channel info",
			"GET /test/health":             "Health check",
		},
		"workflow_patterns": map[string]string{
			"hi":    "Matches regex: ^hi$ (case insensitive)",
			"hello": "Matches regex: ^hello$ (case insensitive)",
		},
		"examples": map[string]string{
			"curl": `curl -X POST http://localhost:8080/test/message \
  -H "Content-Type: application/json" \
  -d '{"channel_id": "test-ch-1", "sender_id": "user1", "text": "hi"}'`,
		},
	})
}

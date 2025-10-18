package webhooktrigger

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/engine/triggerhandler"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/gofiber/fiber/v2"
)

// WebhookTriggerHandler handles generic webhook triggers for workflows
type WebhookTriggerHandler struct {
	workflowRepo   engine.WorkflowRepository
	triggerHandler *triggerhandler.TriggerHandler
}

func NewWebhookTriggerHandler(
	workflowRepo engine.WorkflowRepository,
	triggerHandler *triggerhandler.TriggerHandler,
) *WebhookTriggerHandler {
	return &WebhookTriggerHandler{
		workflowRepo:   workflowRepo,
		triggerHandler: triggerHandler,
	}
}

// HandleWebhook processes incoming webhook requests
// POST /webhooks/trigger/:tenantId/:workflowId
func (h *WebhookTriggerHandler) HandleWebhook(c *fiber.Ctx) error {
	tenantID := kernel.TenantID(c.Params("tenantId"))
	workflowID := kernel.NewWorkflowID(c.Params("workflowId"))

	log.Printf("📥 Received webhook trigger - Tenant: %s, Workflow: %s", tenantID, workflowID)

	// Get workflow (use c.Context() here - it's safe before goroutine)
	workflow, err := h.workflowRepo.FindByID(c.Context(), workflowID)
	if err != nil {
		log.Printf("❌ Workflow not found: %s", workflowID)
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// Verify tenant ownership
	if workflow.TenantID != tenantID {
		log.Printf("🚫 Tenant mismatch for workflow: %s", workflowID)
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Workflow does not belong to tenant",
		})
	}

	// Check if workflow is active
	if !workflow.IsActive {
		log.Printf("⚠️  Workflow is inactive: %s", workflowID)
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Workflow is not active",
		})
	}

	// Verify it's a webhook trigger
	if workflow.Trigger.Type != engine.TriggerTypeWebhook {
		log.Printf("⚠️  Workflow is not a webhook trigger: %s (type: %s)", workflowID, workflow.Trigger.Type)
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Workflow is not configured for webhook triggers",
		})
	}

	// Validate API key
	if !h.validateAPIKey(c, workflow) {
		log.Printf("🔐 API key validation failed for workflow: %s", workflowID)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or missing API key",
		})
	}

	// Parse request body
	var bodyData map[string]any
	if err := c.BodyParser(&bodyData); err != nil {
		bodyData = make(map[string]any)
	}

	// Extract query parameters
	queryParams := make(map[string]any)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		queryParams[string(key)] = string(value)
	})

	// Extract headers
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// Build trigger data
	triggerData := map[string]any{
		"body":        bodyData,
		"query":       queryParams,
		"headers":     headers,
		"method":      c.Method(),
		"path":        c.Path(),
		"workflow_id": workflowID.String(),
		"tenant_id":   tenantID.String(),
	}

	// Add custom fields from workflow config
	if customFields, ok := workflow.Trigger.Config["include_fields"].([]string); ok {
		for _, field := range customFields {
			if val, exists := bodyData[field]; exists {
				triggerData[field] = val
			}
		}
	}

	log.Printf("🚀 Triggering workflow: %s", workflow.Name)
	log.Printf("   📦 Payload keys: %v", getMapKeys(bodyData))

	// ✅ FIX: Use context.Background() for async execution
	go func() {
		// Create a new background context (not tied to the HTTP request)
		ctx := context.Background()

		if err := h.triggerHandler.HandleWebhookTrigger(
			ctx, // ← Use background context instead of c.Context()
			tenantID,
			triggerData,
		); err != nil {
			log.Printf("❌ Failed to trigger workflow: %v", err)
		}
	}()

	// Respond immediately with 202 Accepted
	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"status":      "accepted",
		"workflow_id": workflowID.String(),
		"message":     "Workflow triggered successfully",
		"timestamp":   time.Now().Unix(),
	})
}

// validateAPIKey validates the API key from request
func (h *WebhookTriggerHandler) validateAPIKey(c *fiber.Ctx, workflow *engine.Workflow) bool {
	// Get API key from workflow config
	configuredKey, hasKey := workflow.Trigger.Config["api_key"].(string)

	// If no API key is configured, allow access (open webhook)
	if !hasKey || configuredKey == "" {
		log.Printf("   ℹ️  No API key configured for workflow: %s (open access)", workflow.ID)
		return true
	}

	// Try to get API key from header (X-API-Key)
	providedKey := c.Get("X-API-Key")

	// If not in header, try Authorization header (Bearer token)
	if providedKey == "" {
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			providedKey = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// If not in headers, try query parameter
	if providedKey == "" {
		providedKey = c.Query("api_key")
	}

	// Validate
	isValid := providedKey != "" && providedKey == configuredKey

	if isValid {
		log.Printf("   ✅ API key validated for workflow: %s", workflow.ID)
	} else {
		log.Printf("   🔐 API key validation failed")
		log.Printf("      Expected length: %d, Provided length: %d", len(configuredKey), len(providedKey))
	}

	return isValid
}

// VerifyWebhook handles GET requests for webhook verification (optional)
// GET /webhooks/trigger/:tenantId/:workflowId
func (h *WebhookTriggerHandler) VerifyWebhook(c *fiber.Ctx) error {
	tenantID := kernel.TenantID(c.Params("tenantId"))
	workflowID := kernel.NewWorkflowID(c.Params("workflowId"))

	// Get workflow
	workflow, err := h.workflowRepo.FindByID(c.Context(), workflowID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// Verify tenant
	if workflow.TenantID != tenantID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Validate API key
	if !h.validateAPIKey(c, workflow) {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid API key",
		})
	}

	// If verification challenge is provided (similar to webhook providers like Facebook, Stripe)
	if challenge := c.Query("challenge"); challenge != "" {
		return c.SendString(challenge)
	}

	// Return webhook info
	return c.JSON(fiber.Map{
		"status":       "active",
		"workflow_id":  workflowID.String(),
		"workflow":     workflow.Name,
		"tenant_id":    tenantID.String(),
		"trigger_type": workflow.Trigger.Type,
		"is_active":    workflow.IsActive,
	})
}

// TestWebhook allows testing webhook without triggering (dry run)
// POST /webhooks/trigger/:tenantId/:workflowId/test
func (h *WebhookTriggerHandler) TestWebhook(c *fiber.Ctx) error {
	tenantID := kernel.TenantID(c.Params("tenantId"))
	workflowID := kernel.NewWorkflowID(c.Params("workflowId"))

	// Get workflow
	workflow, err := h.workflowRepo.FindByID(c.Context(), workflowID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Workflow not found",
		})
	}

	// Verify tenant
	if workflow.TenantID != tenantID {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	// Validate API key
	if !h.validateAPIKey(c, workflow) {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid API key",
		})
	}

	// Parse body
	var bodyData map[string]any
	if err := c.BodyParser(&bodyData); err != nil {
		bodyData = make(map[string]any)
	}

	// Return what would be triggered
	return c.JSON(fiber.Map{
		"status":        "test_successful",
		"workflow_id":   workflowID.String(),
		"workflow":      workflow.Name,
		"is_active":     workflow.IsActive,
		"would_trigger": workflow.IsActive,
		"received_data": bodyData,
		"note":          "This is a test request. The workflow was NOT executed.",
	})
}

// Helper function
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

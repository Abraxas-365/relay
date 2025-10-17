package instagram

import (
	"log"
	"net/http"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

// WebhookHandler handles Instagram-specific webhook operations
// It provides endpoints for Meta's webhook verification and incoming message processing
type WebhookHandler struct {
	channelRepo channels.ChannelRepository
	adapter     *InstagramAdapter
	redisClient *redis.Client
}

// NewWebhookHandler creates a new Instagram webhook handler
//
// Parameters:
//   - channelRepo: Repository for channel data access
//   - adapter: Instagram adapter instance (can be nil, will be created per-request)
//   - redisClient: Redis client for message buffering
//
// Returns:
//   - *WebhookHandler: Configured handler ready to process webhooks
func NewWebhookHandler(
	channelRepo channels.ChannelRepository,
	adapter *InstagramAdapter,
	redisClient *redis.Client,
) *WebhookHandler {
	return &WebhookHandler{
		channelRepo: channelRepo,
		adapter:     adapter,
		redisClient: redisClient,
	}
}

// VerifyWebhook handles Meta's webhook verification challenge
//
// Instagram/Meta sends a GET request with verification parameters when you
// configure a webhook in the Meta App Dashboard. This endpoint validates
// the verify token and returns the challenge to complete the verification.
//
// Endpoint: GET /webhooks/instagram/:tenantId/:channelId
//
// Query Parameters:
//   - hub.mode: Should be "subscribe"
//   - hub.verify_token: Token configured in the channel
//   - hub.challenge: Random string to echo back
//
// Returns:
//   - 200 with challenge string if verification successful
//   - 403 if verification fails
//   - 404 if channel not found
func (h *WebhookHandler) VerifyWebhook(c *fiber.Ctx) error {
	tenantID := kernel.TenantID(c.Params("tenantId"))
	channelID := kernel.NewChannelID(c.Params("channelId"))

	log.Printf("🔐 Verifying Instagram webhook - Tenant: %s, Channel: %s", tenantID, channelID)

	// Get channel to verify it exists and retrieve verify token
	channel, err := h.channelRepo.FindByID(c.Context(), channelID, tenantID)
	if err != nil {
		log.Printf("❌ Channel not found: %s (tenant: %s)", channelID, tenantID)
		return fiber.NewError(http.StatusNotFound, "Channel not found")
	}

	// Parse channel configuration
	config, err := channel.GetConfigStruct()
	if err != nil {
		log.Printf("❌ Invalid channel config: %v", err)
		return fiber.NewError(http.StatusInternalServerError, "Invalid channel config")
	}

	// Ensure it's an Instagram channel
	instagramConfig, ok := config.(channels.InstagramConfig)
	if !ok {
		log.Printf("❌ Not an Instagram channel: %s", channelID)
		return fiber.NewError(http.StatusBadRequest, "Not an Instagram channel")
	}

	// Extract verification parameters from query string
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	log.Printf("📝 Verification attempt - Mode: %s, Token matches: %t", mode, token == instagramConfig.VerifyToken)

	// Verify the token matches the configured token
	if mode == "subscribe" && token == instagramConfig.VerifyToken {
		log.Printf("✅ Instagram webhook verified successfully for channel: %s", channelID)
		// Return the challenge to complete verification
		return c.SendString(challenge)
	}

	log.Printf("❌ Instagram webhook verification failed - Invalid token for channel: %s", channelID)
	return fiber.NewError(http.StatusForbidden, "Verification failed")
}

// ReceiveWebhook handles incoming Instagram webhooks (parsing only)
//
// This endpoint receives webhook events from Instagram, parses them,
// and passes the extracted message to the next handler for processing.
//
// Endpoint: POST /webhooks/instagram/:tenantId/:channelId
//
// Event Types Handled:
//   - messages: Regular text and media messages
//   - messaging_postbacks: Button clicks
//   - messaging_reactions: Message reactions
//   - message_echoes: Sent message confirmations
//   - message_reads: Read receipts
//   - message_deliveries: Delivery confirmations
//
// Flow:
//  1. Validate channel exists and is active
//  2. Create adapter with channel-specific config
//  3. Verify webhook signature
//  4. Parse webhook payload
//  5. Extract incoming message
//  6. Store in context and pass to next handler
//
// Returns:
//   - 200 OK always (to prevent Meta from retrying)
//   - Calls c.Next() if message successfully parsed
func (h *WebhookHandler) ReceiveWebhook(c *fiber.Ctx) error {
	tenantID := kernel.TenantID(c.Params("tenantId"))
	channelID := kernel.NewChannelID(c.Params("channelId"))

	log.Printf("📥 Received Instagram webhook - Tenant: %s, Channel: %s", tenantID, channelID)

	// Get channel from repository
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

	// Get channel configuration
	config, err := channel.GetConfigStruct()
	if err != nil {
		log.Printf("❌ Invalid channel config: %v", err)
		return c.SendStatus(fiber.StatusOK)
	}

	// Ensure it's an Instagram channel
	instagramConfig, ok := config.(channels.InstagramConfig)
	if !ok {
		log.Printf("❌ Not an Instagram channel: %s", channelID)
		return c.SendStatus(fiber.StatusOK)
	}

	// Create adapter instance with this channel's specific config (with Redis for buffering)
	adapter := NewInstagramAdapter(instagramConfig, h.redisClient)

	// Read raw webhook payload
	body := c.Body()

	// Extract HTTP headers for signature verification
	headers := make(map[string]string)
	c.Request().Header.VisitAll(func(key, value []byte) {
		headers[string(key)] = string(value)
	})

	// Process webhook using adapter (Instagram-specific parsing)
	incomingMsg, err := adapter.ProcessWebhook(c.Context(), body, headers)
	if err != nil {
		log.Printf("❌ Failed to process Instagram webhook: %v", err)
		// Return 200 to prevent Meta from retrying
		return c.SendStatus(fiber.StatusOK)
	}

	// If message is nil, it means it's not a message event (status update, echo, etc.)
	if incomingMsg == nil {
		log.Printf("ℹ️  Instagram webhook contained no message (likely echo or status update) for channel: %s", channelID)
		return c.SendStatus(fiber.StatusOK)
	}

	log.Printf("✅ Instagram message parsed - From: %s, Type: %s, Content: %s",
		incomingMsg.SenderID,
		incomingMsg.Content.Type,
		incomingMsg.Content.Text,
	)

	// Store parsed message and channel in context for the next handler
	c.Locals("incoming_message", incomingMsg)
	c.Locals("channel", channel)

	// Continue to next handler (generic message processor from channelapi)
	return c.Next()
}

package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Example Usage: Instagram Channel Adapter
// ============================================================================
//
// This file demonstrates how to use the Instagram adapter for sending
// and receiving Instagram Direct Messages through Meta's Graph API.
//
// NOTE: This file is for documentation purposes only and should not be
// imported in production code. Copy the examples you need into your own code.
//
// ============================================================================

// ExampleSetup demonstrates how to create and configure an Instagram adapter
func ExampleSetup() {
	// Step 1: Create Instagram configuration
	config := channels.InstagramConfig{
		Provider:    "meta",
		PageID:      "123456789012345",                          // Your Facebook Page ID
		PageToken:   "EAAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", // Page Access Token
		AppSecret:   "abc123def456ghi789jkl012mno345pqr",        // App Secret for webhook verification
		VerifyToken: "my_secure_verify_token_12345",             // Custom verification token
	}

	// Step 2: Validate configuration
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Step 3: Create adapter instance (with Redis client for buffering)
	// For production, pass actual Redis client. For testing without buffering, can pass nil.
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	adapter := NewInstagramAdapter(config, redisClient)

	// Step 4: Test connection
	ctx := context.Background()
	if err := adapter.TestConnection(ctx, config); err != nil {
		log.Fatalf("Connection test failed: %v", err)
	}

	log.Println("✅ Instagram adapter configured successfully")
	_ = adapter
}

// ============================================================================
// Sending Messages
// ============================================================================

// ExampleSendTextMessage shows how to send a simple text message
func ExampleSendTextMessage(adapter *InstagramAdapter) {
	ctx := context.Background()

	message := channels.OutgoingMessage{
		RecipientID: "instagram_user_scoped_id",
		Content: channels.MessageContent{
			Type: "text",
			Text: "Hello! Thank you for contacting us. How can we help you today?",
		},
		Metadata: map[string]any{
			"agent_id":   "agent_123",
			"session_id": "session_456",
		},
	}

	if err := adapter.SendMessage(ctx, message); err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		return
	}

	log.Println("✅ Text message sent successfully")
}

// ExampleSendImageMessage shows how to send an image with optional caption
func ExampleSendImageMessage(adapter *InstagramAdapter) {
	ctx := context.Background()

	message := channels.OutgoingMessage{
		RecipientID: "instagram_user_scoped_id",
		Content: channels.MessageContent{
			Type:     "image",
			MediaURL: "https://example.com/images/product-photo.jpg",
			Caption:  "Check out our new product!", // Optional
		},
	}

	if err := adapter.SendMessage(ctx, message); err != nil {
		log.Printf("❌ Failed to send image: %v", err)
		return
	}

	log.Println("✅ Image sent successfully")
}

// ExampleSendVideoMessage shows how to send a video message
func ExampleSendVideoMessage(adapter *InstagramAdapter) {
	ctx := context.Background()

	message := channels.OutgoingMessage{
		RecipientID: "instagram_user_scoped_id",
		Content: channels.MessageContent{
			Type:     "video",
			MediaURL: "https://example.com/videos/tutorial.mp4",
			Caption:  "Watch this quick tutorial",
		},
	}

	if err := adapter.SendMessage(ctx, message); err != nil {
		log.Printf("❌ Failed to send video: %v", err)
		return
	}

	log.Println("✅ Video sent successfully")
}

// ExampleSendQuickReplies shows how to send a message with quick reply buttons
func ExampleSendQuickReplies(adapter *InstagramAdapter) {
	ctx := context.Background()

	message := channels.OutgoingMessage{
		RecipientID: "instagram_user_scoped_id",
		Content: channels.MessageContent{
			Type: "text",
			Text: "What would you like to know about our services?",
			Interactive: &channels.Interactive{
				Type: "button",
				Body: "Please select an option:",
				Buttons: []channels.Button{
					{
						ID:    "pricing",
						Title: "💰 Pricing",
						Type:  "reply",
					},
					{
						ID:    "features",
						Title: "✨ Features",
						Type:  "reply",
					},
					{
						ID:    "support",
						Title: "🆘 Support",
						Type:  "reply",
					},
				},
			},
		},
	}

	if err := adapter.SendMessage(ctx, message); err != nil {
		log.Printf("❌ Failed to send quick replies: %v", err)
		return
	}

	log.Println("✅ Quick replies sent successfully")
}

// ExampleSendGenericTemplate shows how to send a rich card with buttons
func ExampleSendGenericTemplate(adapter *InstagramAdapter) {
	ctx := context.Background()

	message := channels.OutgoingMessage{
		RecipientID: "instagram_user_scoped_id",
		Content: channels.MessageContent{
			Type:    "template",
			Text:    "New Product Launch! 🎉",
			Caption: "Limited time offer - 20% off",
			Interactive: &channels.Interactive{
				Type: "template",
				Buttons: []channels.Button{
					{
						ID:    "view_product",
						Title: "View Product",
						Type:  "url",
						URL:   "https://example.com/products/new-launch",
					},
					{
						ID:    "add_to_cart",
						Title: "Add to Cart",
						Type:  "reply",
					},
					{
						ID:    "learn_more",
						Title: "Learn More",
						Type:  "reply",
					},
				},
			},
		},
	}

	if err := adapter.SendMessage(ctx, message); err != nil {
		log.Printf("❌ Failed to send template: %v", err)
		return
	}

	log.Println("✅ Template message sent successfully")
}

// ============================================================================
// Processing Incoming Messages
// ============================================================================

// ExampleProcessWebhook demonstrates how to handle incoming Instagram webhooks
func ExampleProcessWebhook(adapter *InstagramAdapter) {
	// Simulated webhook payload from Instagram
	webhookPayload := []byte(`{
		"object": "instagram",
		"entry": [{
			"id": "page_id",
			"time": 1234567890,
			"messaging": [{
				"sender": {"id": "user_instagram_id"},
				"recipient": {"id": "page_id"},
				"timestamp": 1234567890,
				"message": {
					"mid": "msg_123456",
					"text": "Hello, I have a question about your product"
				}
			}]
		}]
	}`)

	// Headers from the webhook request
	headers := map[string]string{
		"X-Hub-Signature-256": "sha256=calculated_signature_here",
		"Content-Type":        "application/json",
	}

	ctx := context.Background()
	incomingMsg, err := adapter.ProcessWebhook(ctx, webhookPayload, headers)
	if err != nil {
		log.Printf("❌ Failed to process webhook: %v", err)
		return
	}

	if incomingMsg == nil {
		log.Println("ℹ️  No message to process (status update or echo)")
		return
	}

	// Process the incoming message
	log.Printf("✅ Received message from %s: %s", incomingMsg.SenderID, incomingMsg.Content.Text)

	// Route to appropriate handler based on content
	handleIncomingMessage(adapter, incomingMsg)
}

// handleIncomingMessage demonstrates message routing logic
func handleIncomingMessage(adapter *InstagramAdapter, msg *channels.IncomingMessage) {
	ctx := context.Background()

	switch msg.Content.Type {
	case "text":
		// Handle text message
		if msg.Content.Text == "hello" || msg.Content.Text == "hi" {
			respondWithGreeting(ctx, adapter, msg.SenderID)
		} else {
			respondWithEcho(ctx, adapter, msg.SenderID, msg.Content.Text)
		}

	case "image":
		// Handle image message
		log.Printf("Received image from %s: %s", msg.SenderID, msg.Content.MediaURL)
		respondWithAcknowledgment(ctx, adapter, msg.SenderID, "image")

	case "video":
		// Handle video message
		log.Printf("Received video from %s: %s", msg.SenderID, msg.Content.MediaURL)
		respondWithAcknowledgment(ctx, adapter, msg.SenderID, "video")

	case "postback":
		// Handle button click
		payload := msg.Metadata["postback_payload"].(string)
		handlePostback(ctx, adapter, msg.SenderID, payload)

	case "reaction":
		// Handle message reaction
		emoji := msg.Metadata["reaction_emoji"].(string)
		log.Printf("User %s reacted with %s", msg.SenderID, emoji)

	default:
		log.Printf("Unknown message type: %s", msg.Content.Type)
	}
}

// respondWithGreeting sends a greeting message
func respondWithGreeting(ctx context.Context, adapter *InstagramAdapter, recipientID string) {
	response := channels.OutgoingMessage{
		RecipientID: recipientID,
		Content: channels.MessageContent{
			Type: "text",
			Text: "Hello! 👋 Welcome to our support channel. How can I assist you today?",
			Interactive: &channels.Interactive{
				Type: "button",
				Buttons: []channels.Button{
					{ID: "new_order", Title: "New Order"},
					{ID: "track_order", Title: "Track Order"},
					{ID: "support", Title: "Get Support"},
				},
			},
		},
	}

	adapter.SendMessage(ctx, response)
}

// respondWithEcho echoes back the user's message
func respondWithEcho(ctx context.Context, adapter *InstagramAdapter, recipientID, text string) {
	response := channels.OutgoingMessage{
		RecipientID: recipientID,
		Content: channels.MessageContent{
			Type: "text",
			Text: fmt.Sprintf("You said: %s", text),
		},
	}

	adapter.SendMessage(ctx, response)
}

// respondWithAcknowledgment acknowledges media receipt
func respondWithAcknowledgment(ctx context.Context, adapter *InstagramAdapter, recipientID, mediaType string) {
	response := channels.OutgoingMessage{
		RecipientID: recipientID,
		Content: channels.MessageContent{
			Type: "text",
			Text: fmt.Sprintf("Thanks for sharing the %s! Our team will review it shortly.", mediaType),
		},
	}

	adapter.SendMessage(ctx, response)
}

// handlePostback processes button postback events
func handlePostback(ctx context.Context, adapter *InstagramAdapter, recipientID, payload string) {
	var response channels.OutgoingMessage

	switch payload {
	case "pricing":
		response = channels.OutgoingMessage{
			RecipientID: recipientID,
			Content: channels.MessageContent{
				Type: "text",
				Text: "Our pricing starts at $9.99/month. Visit our website for detailed plans!",
			},
		}

	case "features":
		response = channels.OutgoingMessage{
			RecipientID: recipientID,
			Content: channels.MessageContent{
				Type: "text",
				Text: "Here are our key features:\n• Feature 1\n• Feature 2\n• Feature 3",
			},
		}

	case "support":
		response = channels.OutgoingMessage{
			RecipientID: recipientID,
			Content: channels.MessageContent{
				Type: "text",
				Text: "Our support team is available 24/7. How can we help you?",
			},
		}

	default:
		response = channels.OutgoingMessage{
			RecipientID: recipientID,
			Content: channels.MessageContent{
				Type: "text",
				Text: "Thank you for your interest!",
			},
		}
	}

	adapter.SendMessage(ctx, response)
}

// ============================================================================
// Advanced Usage Examples
// ============================================================================

// ExampleConversationFlow demonstrates a multi-step conversation
func ExampleConversationFlow(adapter *InstagramAdapter, userID string) {
	ctx := context.Background()

	// Step 1: Welcome message
	step1 := channels.OutgoingMessage{
		RecipientID: userID,
		Content: channels.MessageContent{
			Type: "text",
			Text: "Welcome to our product catalog! 🛍️",
		},
	}
	adapter.SendMessage(ctx, step1)

	// Small delay for better UX
	time.Sleep(1 * time.Second)

	// Step 2: Show product image
	step2 := channels.OutgoingMessage{
		RecipientID: userID,
		Content: channels.MessageContent{
			Type:     "image",
			MediaURL: "https://example.com/products/featured.jpg",
			Caption:  "Check out our featured product!",
		},
	}
	adapter.SendMessage(ctx, step2)

	time.Sleep(1 * time.Second)

	// Step 3: Offer options
	step3 := channels.OutgoingMessage{
		RecipientID: userID,
		Content: channels.MessageContent{
			Type: "text",
			Text: "What would you like to do?",
			Interactive: &channels.Interactive{
				Type: "button",
				Buttons: []channels.Button{
					{ID: "buy_now", Title: "🛒 Buy Now"},
					{ID: "more_info", Title: "ℹ️ More Info"},
					{ID: "share", Title: "📤 Share"},
				},
			},
		},
	}
	adapter.SendMessage(ctx, step3)
}

// ExampleBatchMessaging demonstrates sending messages to multiple users
func ExampleBatchMessaging(adapter *InstagramAdapter, userIDs []string) {
	ctx := context.Background()

	message := channels.OutgoingMessage{
		Content: channels.MessageContent{
			Type: "text",
			Text: "🎉 Flash Sale! 50% off for the next 2 hours. Don't miss out!",
		},
	}

	for _, userID := range userIDs {
		message.RecipientID = userID

		if err := adapter.SendMessage(ctx, message); err != nil {
			log.Printf("❌ Failed to send to %s: %v", userID, err)
			continue
		}

		log.Printf("✅ Sent to %s", userID)

		// Rate limiting - be respectful of Instagram's limits
		time.Sleep(100 * time.Millisecond)
	}
}

// ExampleErrorHandling demonstrates proper error handling
func ExampleErrorHandling(adapter *InstagramAdapter) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message := channels.OutgoingMessage{
		RecipientID: "invalid_user_id",
		Content: channels.MessageContent{
			Type: "text",
			Text: "Test message",
		},
	}

	if err := adapter.SendMessage(ctx, message); err != nil {
		// Handle error appropriately based on error message
		errMsg := err.Error()

		// Check error patterns (errors are from errx library)
		if strings.Contains(errMsg, "API error") {
			log.Printf("Instagram API error: %v", err)
			// Handle API error (retry, log, alert, etc.)
		} else if strings.Contains(errMsg, "auth") || strings.Contains(errMsg, "authentication") {
			log.Printf("Authentication failed: %v", err)
			// Token may be expired - refresh it
		} else if strings.Contains(errMsg, "signature") {
			log.Printf("Security issue: %v", err)
			// Invalid webhook - potential security issue
		} else {
			log.Printf("Error sending message: %v", err)
		}
		return
	}

	log.Println("✅ Message sent successfully")
}

// ============================================================================
// Testing Utilities
// ============================================================================

// ExampleTestConfiguration validates configuration before use
func ExampleTestConfiguration() {
	config := channels.InstagramConfig{
		Provider:    "meta",
		PageID:      "test_page_id",
		PageToken:   "test_token",
		VerifyToken: "test_verify",
	}

	// Test validation
	if err := config.Validate(); err != nil {
		log.Fatalf("Configuration invalid: %v", err)
	}

	// Test features
	features := config.GetFeatures()
	log.Printf("Channel supports text: %t", features.SupportsText)
	log.Printf("Channel supports images: %t", features.SupportsImages)
	log.Printf("Max message length: %d", features.MaxMessageLength)
	log.Printf("Max attachment size: %d MB", features.MaxAttachmentSize/(1024*1024))

	// Create adapter and test connection
	// Pass nil for Redis if buffering is not needed for testing
	adapter := NewInstagramAdapter(config, nil)
	ctx := context.Background()

	if err := adapter.TestConnection(ctx, config); err != nil {
		log.Fatalf("Connection test failed: %v", err)
	}

	log.Println("✅ All tests passed")
}

// ExamplePrettyPrintWebhook helps debug webhook payloads
func ExamplePrettyPrintWebhook(webhookPayload []byte) {
	var webhook InstagramWebhook
	if err := json.Unmarshal(webhookPayload, &webhook); err != nil {
		log.Printf("Failed to parse webhook: %v", err)
		return
	}

	prettyJSON, _ := json.MarshalIndent(webhook, "", "  ")
	fmt.Println("Received webhook:")
	fmt.Println(string(prettyJSON))
}

// ============================================================================
// Integration Examples
// ============================================================================

// ExampleIntegrateWithChannelManager shows integration with the channel manager
func ExampleIntegrateWithChannelManager() {
	// This would typically be done during application startup

	// 1. Create channel configuration in database
	channel := channels.Channel{
		ID:          kernel.NewChannelID("instagram_channel_1"),
		TenantID:    kernel.TenantID("tenant_123"),
		Type:        channels.ChannelTypeInstagram,
		Name:        "Instagram Support Channel",
		Description: "Primary customer support channel for Instagram",
		IsActive:    true,
		WebhookURL:  "https://api.example.com/webhooks/instagram/tenant_123/instagram_channel_1",
	}

	// 2. Set channel config
	config := channels.InstagramConfig{
		Provider:    "meta",
		PageID:      "123456789012345",
		PageToken:   "EAAxxxxxxxxxxxxx",
		AppSecret:   "app_secret_here",
		VerifyToken: "verify_token_here",
	}

	if err := channel.UpdateConfig(config); err != nil {
		log.Fatalf("Failed to update config: %v", err)
	}

	// 3. Register with channel manager (handled by your application)
	log.Printf("Channel ready: %s", channel.ID)
	log.Printf("Webhook URL: %s", channel.WebhookURL)
	log.Printf("Configure this URL in Meta App Dashboard")
}

// ============================================================================
// Best Practices
// ============================================================================

/*
BEST PRACTICES FOR INSTAGRAM MESSAGING:

1. Message Timing
   - Only send messages within 24 hours of user's last message
   - Use message tags for notifications outside the 24-hour window
   - Respect user preferences and time zones

2. Rate Limiting
   - Instagram has rate limits on API calls
   - Implement exponential backoff for retries
   - Monitor your API usage in Meta App Dashboard

3. Error Handling
   - Always handle API errors gracefully
   - Log errors for debugging
   - Implement retry logic for transient failures
   - Don't retry on authentication errors

4. Security
   - Always verify webhook signatures
   - Keep your App Secret secure
   - Use HTTPS for all webhooks
   - Rotate tokens periodically

5. User Experience
   - Keep messages concise (< 1000 chars)
   - Use interactive elements (buttons, quick replies)
   - Provide clear calls-to-action
   - Don't spam users with too many messages

6. Testing
   - Test thoroughly in development before production
   - Use test accounts for development
   - Monitor logs for errors and issues
   - Set up alerts for failures

7. Compliance
   - Follow Instagram's Platform Policies
   - Respect user privacy (GDPR, CCPA, etc.)
   - Provide opt-out mechanisms
   - Don't send marketing messages without consent
*/

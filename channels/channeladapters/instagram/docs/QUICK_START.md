# Instagram Adapter - Quick Start Guide

Get your Instagram channel up and running in 5 minutes!

## 🚀 Quick Setup (5 minutes)

### Step 1: Meta App Setup (2 minutes)

1. Go to [Meta for Developers](https://developers.facebook.com/)
2. Create a new app (or use existing)
3. Add **Instagram** product
4. Connect your Facebook Page (must be linked to Instagram Business Account)

### Step 2: Get Credentials (1 minute)

```bash
# In Meta App Dashboard:
# 1. Go to Instagram > Basic Settings
# 2. Copy your App ID and App Secret
# 3. Go to Tools > Graph API Explorer
# 4. Select your Page and get Page Access Token
# 5. Your Page ID is in the token response
```

**Required Permissions:**
- `pages_messaging`
- `instagram_basic`
- `instagram_manage_messages`

### Step 3: Create Channel (1 minute)

```go
package main

import (
    "context"
    "github.com/Abraxas-365/relay/channels"
    instagram "github.com/Abraxas-365/relay/channels/channeladapters/instagram"
)

func main() {
    // Create configuration
    config := channels.InstagramConfig{
        Provider:    "meta",
        PageID:      "123456789012345",          // Your Page ID
        PageToken:   "EAAxxxxxxxxxxxxxxxxxx",    // Page Access Token
        AppSecret:   "your_app_secret_here",     // From App Dashboard
        VerifyToken: "my_custom_verify_123",     // Any string you choose
        
        // Optional: Enable message buffering (combines rapid messages)
        BufferEnabled:        true,              // Enable buffering
        BufferTimeSeconds:    5,                 // Buffer for 5 seconds
        BufferResetOnMessage: false,             // Don't reset timer on new messages
    }
    
    // Create Redis client for buffering (or nil to disable)
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // Create adapter with Redis client
    adapter := instagram.NewInstagramAdapter(config, redisClient)
    
    // Test connection
    ctx := context.Background()
    if err := adapter.TestConnection(ctx, config); err != nil {
        panic("Connection failed: " + err.Error())
    }
    
    println("✅ Instagram adapter ready!")
}
```

### Step 4: Configure Webhook (1 minute)

1. In Meta App Dashboard → Webhooks → Instagram
2. Set Callback URL: `https://your-domain.com/webhooks/instagram/{tenant_id}/{channel_id}`
3. Set Verify Token: Use the same token from your config (`my_custom_verify_123`)
4. Subscribe to fields:
   - ✅ `messages`
   - ✅ `messaging_postbacks`
   - ✅ `messaging_reactions`

5. Click **Verify and Save**

## 📤 Send Your First Message

```go
// Send text message
msg := channels.OutgoingMessage{
    RecipientID: "instagram_user_scoped_id",
    Content: channels.MessageContent{
        Type: "text",
        Text: "Hello! This is my first message via Instagram API! 🎉",
    },
}

err := adapter.SendMessage(ctx, msg)
if err != nil {
    log.Printf("Error: %v", err)
} else {
    log.Println("✅ Message sent!")
}
```

## 📥 Receive Messages

```go
// In your webhook handler
func handleInstagramWebhook(c *fiber.Ctx) error {
    // Get payload and headers
    payload := c.Body()
    headers := make(map[string]string)
    c.Request().Header.VisitAll(func(key, value []byte) {
        headers[string(key)] = string(value)
    })
    
    // Process webhook
    msg, err := adapter.ProcessWebhook(c.Context(), payload, headers)
    if err != nil {
        return c.SendStatus(200) // Always return 200 to prevent retries
    }
    
    if msg != nil {
        log.Printf("Received: %s from %s", msg.Content.Text, msg.SenderID)
        // Handle the message...
    }
    
    return c.SendStatus(200)
}
```

## 🎨 Common Message Types

### Text with Quick Replies

```go
msg := channels.OutgoingMessage{
    RecipientID: userID,
    Content: channels.MessageContent{
        Type: "text",
        Text: "What can I help you with?",
        Interactive: &channels.Interactive{
            Type: "button",
            Buttons: []channels.Button{
                {ID: "help", Title: "Help"},
                {ID: "pricing", Title: "Pricing"},
                {ID: "contact", Title: "Contact Us"},
            },
        },
    },
}
```

### Image Message

```go
msg := channels.OutgoingMessage{
    RecipientID: userID,
    Content: channels.MessageContent{
        Type:     "image",
        MediaURL: "https://example.com/image.jpg",
        Caption:  "Check this out!",
    },
}
```

### Template with Buttons

```go
msg := channels.OutgoingMessage{
    RecipientID: userID,
    Content: channels.MessageContent{
        Type:    "template",
        Text:    "New Product Launch! 🎉",
        Caption: "Limited time offer",
        Interactive: &channels.Interactive{
            Type: "template",
            Buttons: []channels.Button{
                {ID: "view", Title: "View Product", Type: "url", URL: "https://example.com/product"},
                {ID: "buy", Title: "Buy Now", Type: "reply"},
            },
        },
    },
}
```

## 🔧 Integration with Channel Manager

```go
// In your application setup
import (
    "github.com/Abraxas-365/relay/channels/channelmanager"
    instagram "github.com/Abraxas-365/relay/channels/channeladapters/instagram"
)

func setupChannels() {
    // Create channel manager
    manager := channelmanager.NewDefaultChannelManager(channelRepo, redisClient)
    
    // Create Instagram channel
    channel := channels.Channel{
        ID:          kernel.NewChannelID("instagram_1"),
        TenantID:    kernel.TenantID("tenant_123"),
        Type:        channels.ChannelTypeInstagram,
        Name:        "Instagram Support",
        Description: "Customer support channel",
        IsActive:    true,
    }
    
    // Set configuration
    config := channels.InstagramConfig{
        Provider:    "meta",
        PageID:      "your_page_id",
        PageToken:   "your_token",
        AppSecret:   "your_secret",
        VerifyToken: "your_verify_token",
    }
    channel.UpdateConfig(config)
    
    // Register channel
    manager.RegisterChannel(context.Background(), channel)
}
```

## 🐛 Troubleshooting

### Webhook Verification Fails

```bash
# Test your webhook verification endpoint
curl "http://localhost:3000/webhooks/instagram/tenant_123/channel_456?hub.mode=subscribe&hub.verify_token=my_custom_verify_123&hub.challenge=test_challenge"

# Should return: test_challenge
# If not, check:
# - Verify token matches exactly
# - Channel exists in database
# - Channel is active
# - Correct tenant_id and channel_id in URL
```

### Message Not Sending

```go
// Enable detailed logging
log.Printf("🔑 Token: %s...", config.PageToken[:20])
log.Printf("📱 Page ID: %s", config.PageID)
log.Printf("👤 Recipient: %s", msg.RecipientID)

err := adapter.SendMessage(ctx, msg)
if err != nil {
    log.Printf("❌ Error details: %+v", err)
    // Check:
    // - Token is valid and not expired
    // - User has messaged you in last 24 hours
    // - Page ID is correct
    // - Recipient ID is valid Instagram scoped ID
}
```

### Getting User's Instagram ID

When a user messages you, their Instagram scoped ID is in the webhook:

```json
{
  "sender": {
    "id": "1234567890"  // This is the Instagram scoped ID to use
  }
}
```

**Important**: You can ONLY message users who have messaged you first (24-hour window).

## 📚 Next Steps

1. ✅ **You're ready!** Start sending and receiving messages
2. 📖 Read [README.md](./README.md) for complete documentation
3. 🏗️ Check [ARCHITECTURE.md](./ARCHITECTURE.md) for technical details
4. 💡 See [example_usage.go](./example_usage.go) for more examples

## ⚡ Pro Tips

1. **24-Hour Rule**: You can only message users within 24 hours of their last message
2. **Rate Limits**: Instagram has rate limits - don't spam!
3. **User IDs**: Use the scoped ID from webhooks, not their Instagram username
4. **Testing**: Use a test Instagram Business Account for development
5. **Logging**: Check your logs - they have detailed info about what's happening

## 🔄 Optional: Message Buffering

Message buffering combines multiple rapid messages from the same user into one:

```go
config := channels.InstagramConfig{
    // ... other settings ...
    BufferEnabled:        true,  // Enable buffering
    BufferTimeSeconds:    5,     // Wait 5 seconds before processing
    BufferResetOnMessage: false, // Fixed time window
}
```

**Benefits:**
- Better conversation context (combines "Hey" + "Can you" + "help me?")
- Reduced processing overhead
- Better AI/chatbot understanding

**Requirements:**
- Redis server running
- Pass Redis client to `NewInstagramAdapter(config, redisClient)`

**When to use:**
- ✅ Users send multiple quick messages
- ✅ AI/NLP processing benefits from complete context
- ❌ Real-time response is critical

See [BUFFERING.md](./BUFFERING.md) for detailed documentation.

## 🆘 Need Help?

- **Documentation**: See [README.md](./README.md)
- **Buffering Guide**: See [BUFFERING.md](./BUFFERING.md)
- **Examples**: Check [example_usage.go](./example_usage.go)
- **Instagram Docs**: [Meta Developers](https://developers.facebook.com/docs/messenger-platform/instagram)
- **Common Issues**: See troubleshooting section above
</text>


---

**Time to first message**: ~5 minutes ⚡  
**Difficulty**: Easy 🟢  
**Prerequisites**: Instagram Business Account + Facebook Page

Happy messaging! 🚀
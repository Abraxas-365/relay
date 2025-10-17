# Instagram Channel Adapter

A robust, production-ready adapter for Instagram Direct Messaging through Meta's Graph API. This adapter enables seamless two-way communication with Instagram users via Direct Messages.

## 📋 Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Features](#features)
- [Configuration](#configuration)
- [Setup Guide](#setup-guide)
- [Usage](#usage)
- [Webhook Events](#webhook-events)
- [Message Types](#message-types)
- [Error Handling](#error-handling)
- [Security](#security)
- [Testing](#testing)
- [Limitations](#limitations)
- [Troubleshooting](#troubleshooting)

## 🎯 Overview

The Instagram adapter implements the `ChannelAdapter` interface to provide Instagram Direct Messaging capabilities. It handles:

- **Outgoing Messages**: Send text, images, videos, and interactive messages to Instagram users
- **Incoming Messages**: Process incoming messages, reactions, postbacks, and status updates
- **Webhook Security**: Verify webhook signatures using HMAC-SHA256
- **Error Handling**: Comprehensive error handling with detailed logging
- **Retry Logic**: Automatic retry for transient failures

## 🏗️ Architecture

The adapter follows a clean, modular architecture:

```
instagram/
├── ig_adapter.go      # Core adapter implementation (ChannelAdapter interface)
├── handler.go         # Webhook HTTP handlers (verification & receiving)
├── routes.go          # Route registration and middleware chaining
└── README.md          # This file
```

### Component Responsibilities

#### `ig_adapter.go`
- **Purpose**: Core adapter logic for Instagram API integration
- **Responsibilities**:
  - Send messages via Instagram Graph API
  - Parse incoming webhook payloads
  - Verify webhook signatures
  - Transform between internal and Instagram message formats
  - Handle API errors and retries

#### `handler.go`
- **Purpose**: HTTP request handling for webhooks
- **Responsibilities**:
  - Handle webhook verification (GET requests)
  - Receive and parse webhook events (POST requests)
  - Load channel configuration
  - Pass parsed messages to generic processor

#### `routes.go`
- **Purpose**: Route configuration and registration
- **Responsibilities**:
  - Register webhook endpoints
  - Chain handlers (Instagram-specific → Generic)
  - Configure middleware

## ✨ Features

### Supported Message Types

| Type | Direction | Description |
|------|-----------|-------------|
| Text | Both | Plain text messages (up to 1000 chars) |
| Images | Both | JPEG, PNG images (up to 8MB) |
| Videos | Both | MP4 videos (up to 8MB) |
| Quick Replies | Outgoing | Interactive button-like responses |
| Generic Templates | Outgoing | Structured messages with buttons |
| Reactions | Incoming | Emoji reactions to messages |
| Postbacks | Incoming | Button click events |
| Message Buffering | Both | Combines rapid messages (optional) |

### Channel Capabilities

```go
ChannelFeatures{
    SupportsText:                true,
    SupportsAttachments:         true,
    SupportsImages:              true,
    SupportsAudio:               false,
    SupportsVideo:               true,
    SupportsDocuments:           false,
    SupportsInteractiveMessages: true,
    SupportsButtons:             true,
    SupportsQuickReplies:        true,
    SupportsTemplates:           false,
    SupportsLocation:            false,
    SupportsContacts:            false,
    SupportsReactions:           true,
    SupportsThreads:             true,
    MaxMessageLength:            1000,
    MaxAttachmentSize:           8 * 1024 * 1024, // 8MB
    SupportedMimeTypes: []string{
        "image/jpeg", 
        "image/png",
        "video/mp4",
    },
}
```

### Message Buffering

The Instagram adapter supports optional message buffering, similar to the WhatsApp adapter:

```go
config := channels.InstagramConfig{
    Provider:             "meta",
    PageID:              "123456789012345",
    PageToken:           "EAAxxxxx...",
    BufferEnabled:        true,  // Enable buffering
    BufferTimeSeconds:    5,     // Buffer for 5 seconds
    BufferResetOnMessage: true,  // Reset timer on each message
}
```

**How it works:**
1. When a user sends a message, it's added to a buffer
2. A timer starts (e.g., 5 seconds)
3. If more messages arrive, they're added to the buffer
4. When timer expires, all buffered messages are combined and processed
5. If `BufferResetOnMessage` is true, timer resets with each new message

**Use cases:**
- Users typing multiple short messages quickly
- Reducing webhook processing overhead
- Better conversation context
- Handling "typing in progress" patterns

## ⚙️ Configuration

### InstagramConfig Structure

```go
type InstagramConfig struct {
    Provider    string `json:"provider"`     // Always "meta"
    PageID      string `json:"page_id"`      // Instagram-connected Facebook Page ID
    PageToken   string `json:"page_token"`   // Page Access Token
    AppSecret   string `json:"app_secret"`   // App Secret for webhook verification
    VerifyToken string `json:"verify_token"` // Custom token for webhook setup
    
    // Buffer configuration (optional)
    BufferEnabled        bool `json:"buffer_enabled"`          // Enable message buffering
    BufferTimeSeconds    int  `json:"buffer_time_seconds"`     // Time window to buffer messages (default: 5 seconds)
    BufferResetOnMessage bool `json:"buffer_reset_on_message"` // Reset timer on each new message
}
```

### Configuration Example

```json
{
    "provider": "meta",
    "page_id": "123456789012345",
    "page_token": "EAAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
    "app_secret": "abc123def456ghi789jkl012mno345pqr",
    "verify_token": "my_secure_verify_token_12345",
    "buffer_enabled": true,
    "buffer_time_seconds": 5,
    "buffer_reset_on_message": true
}
```

### Required Fields

- **`page_id`**: Facebook Page ID connected to your Instagram Business Account
- **`page_token`**: Page Access Token with `pages_messaging` and `instagram_basic` permissions
- **`verify_token`**: Custom string for webhook verification (you define this)
- **`app_secret`**: (Optional but recommended) App Secret for webhook signature verification

### Buffer Configuration (Optional)

- **`buffer_enabled`**: Enable message buffering (default: false)
- **`buffer_time_seconds`**: Time window to buffer messages in seconds (default: 5, max: 60)
- **`buffer_reset_on_message`**: Reset timer on each new message (default: false)

**What is Message Buffering?**

Message buffering combines multiple rapid messages from the same user into a single message. This is useful when users send multiple quick messages like:

```
User: "Hey"
User: "Can you"
User: "help me?"
```

With buffering enabled (5 seconds), these are combined into:
```
"Hey\nCan you\nhelp me?"
```

This improves conversation context and reduces processing overhead.

## 🚀 Setup Guide

### 1. Create Meta App

1. Go to [Meta for Developers](https://developers.facebook.com/)
2. Create a new app or use an existing one
3. Add **Instagram** product to your app
4. Add **Webhooks** product to your app

### 2. Connect Instagram Account

1. Go to your app's Instagram settings
2. Connect a Facebook Page that's linked to an Instagram Business Account
3. Instagram Professional accounts (Business or Creator) are required

### 3. Generate Access Token

```bash
# Get Page Access Token
# Go to: Graph API Explorer > Select your app > Get Token > Get Page Access Token
# Select your page and required permissions:
# - pages_messaging
# - instagram_basic
# - instagram_manage_messages
```

### 4. Subscribe to Webhooks

1. In App Dashboard → Webhooks → Instagram
2. Subscribe to these fields:
   - `messages`
   - `messaging_postbacks`
   - `messaging_reactions`
   - `message_echoes`
   - `message_reads`

### 5. Configure Webhook URL

Your webhook URL format:
```
https://your-domain.com/webhooks/instagram/{tenant_id}/{channel_id}
```

**Verification Token**: Use the `verify_token` from your config

### 6. Create Channel in Relay

```bash
curl -X POST https://your-api.com/channels \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "tenant_123",
    "type": "INSTAGRAM",
    "name": "Instagram Support",
    "description": "Instagram channel for customer support",
    "config": {
      "provider": "meta",
      "page_id": "123456789012345",
      "page_token": "EAAxxxxx...",
      "app_secret": "abc123...",
      "verify_token": "my_verify_token"
    },
    "is_active": true
  }'
```

## 💬 Usage

### Sending Messages

#### Text Message

```go
msg := channels.OutgoingMessage{
    RecipientID: "instagram_user_id",
    Content: channels.MessageContent{
        Type: "text",
        Text: "Hello! How can I help you today?",
    },
}

err := adapter.SendMessage(ctx, msg)
```

#### Image Message

```go
msg := channels.OutgoingMessage{
    RecipientID: "instagram_user_id",
    Content: channels.MessageContent{
        Type:     "image",
        MediaURL: "https://example.com/image.jpg",
    },
}

err := adapter.SendMessage(ctx, msg)
```

#### Text with Quick Replies

```go
msg := channels.OutgoingMessage{
    RecipientID: "instagram_user_id",
    Content: channels.MessageContent{
        Type: "text",
        Text: "What would you like to know?",
    },
    QuickReplies: []channels.QuickReply{
        {Title: "Pricing", Payload: "pricing_info"},
        {Title: "Features", Payload: "feature_list"},
        {Title: "Support", Payload: "contact_support"},
    },
}

err := adapter.SendMessage(ctx, msg)
```

#### Generic Template (Card with Buttons)

```go
msg := channels.OutgoingMessage{
    RecipientID: "instagram_user_id",
    Content: channels.MessageContent{
        Type:    "template",
        Text:    "Check out our latest product!",
        Caption: "Limited time offer",
    },
    Buttons: []channels.Button{
        {Title: "View Product", URL: "https://example.com/product"},
        {Title: "Learn More", Payload: "learn_more"},
    },
}

err := adapter.SendMessage(ctx, msg)
```

### Processing Incoming Messages

The adapter automatically processes incoming webhooks:

```go
// In your webhook handler
incomingMsg, err := adapter.ProcessWebhook(ctx, payload, headers)
if err != nil {
    log.Printf("Error processing webhook: %v", err)
    return
}

if incomingMsg != nil {
    // Process the message
    log.Printf("Received message from %s: %s", 
        incomingMsg.SenderID, 
        incomingMsg.Content.Text)
}
```

## 📨 Webhook Events

### Message Event

```json
{
  "object": "instagram",
  "entry": [{
    "id": "page_id",
    "time": 1234567890,
    "messaging": [{
      "sender": {"id": "user_id"},
      "recipient": {"id": "page_id"},
      "timestamp": 1234567890,
      "message": {
        "mid": "message_id",
        "text": "Hello!"
      }
    }]
  }]
}
```

### Postback Event (Button Click)

```json
{
  "object": "instagram",
  "entry": [{
    "messaging": [{
      "sender": {"id": "user_id"},
      "recipient": {"id": "page_id"},
      "timestamp": 1234567890,
      "postback": {
        "title": "Get Started",
        "payload": "get_started_payload"
      }
    }]
  }]
}
```

### Reaction Event

```json
{
  "object": "instagram",
  "entry": [{
    "messaging": [{
      "sender": {"id": "user_id"},
      "recipient": {"id": "page_id"},
      "timestamp": 1234567890,
      "reaction": {
        "mid": "message_id",
        "action": "react",
        "emoji": "❤️"
      }
    }]
  }]
}
```

## 🛡️ Security

### Webhook Signature Verification

The adapter verifies all incoming webhooks using HMAC-SHA256:

1. Meta signs webhook payloads with your App Secret
2. Signature is sent in `X-Hub-Signature-256` header
3. Adapter verifies signature matches expected value
4. Invalid signatures are rejected

```go
// Signature verification (automatic)
if err := adapter.verifySignature(payload, headers); err != nil {
    // Webhook rejected - potential security threat
    return err
}
```

### Best Practices

1. **Always set `app_secret`** - Required for signature verification
2. **Use HTTPS** - Webhooks must be delivered over HTTPS
3. **Rotate tokens** - Periodically rotate Page Access Tokens
4. **Validate inputs** - Always validate user inputs before processing
5. **Rate limiting** - Implement rate limiting on your webhook endpoint

## 🧪 Testing

### Test Connection

```go
config := channels.InstagramConfig{
    Provider:    "meta",
    PageID:      "your_page_id",
    PageToken:   "your_page_token",
    VerifyToken: "your_verify_token",
}

adapter := instagram.NewInstagramAdapter(config)

// Test API connectivity
err := adapter.TestConnection(ctx, config)
if err != nil {
    log.Fatal("Connection test failed:", err)
}
```

### Manual Webhook Testing

```bash
# Test webhook verification (GET)
curl "http://localhost:3000/webhooks/instagram/tenant_123/channel_456?hub.mode=subscribe&hub.verify_token=my_verify_token&hub.challenge=test_challenge"

# Should return: test_challenge
```

### Send Test Message

```bash
# Using the Graph API Explorer
curl -X POST "https://graph.facebook.com/v18.0/me/messages" \
  -H "Authorization: Bearer YOUR_PAGE_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "recipient": {"id": "USER_INSTAGRAM_ID"},
    "message": {"text": "Test message from API"}
  }'
```

## ⚠️ Limitations

### Instagram Platform Limits

1. **Message Window**: Can only send messages within 24 hours of last user message
2. **Message Length**: Text limited to 1000 characters
3. **Attachment Size**: Media files limited to 8MB
4. **Rate Limits**: Subject to Instagram's rate limiting (varies by app)
5. **No Audio**: Instagram doesn't support audio messages
6. **No Documents**: PDF and other document types not supported

### Adapter Limitations

1. **No Stories**: Does not support Instagram Stories
2. **No Comments**: Does not handle comment replies
3. **No Live Chat**: Real-time features require polling
4. **Single Page**: One channel per Facebook Page

## 🔧 Troubleshooting

### Common Issues

#### 1. Webhook Verification Fails

**Symptoms**: 
- GET request returns 403 Forbidden
- Meta shows "Failed to verify webhook"

**Solutions**:
```bash
# Check verify_token matches
# Check channel exists and is accessible
# Check URL format is correct
# Check logs for error details
```

#### 2. Messages Not Sending

**Symptoms**:
- API returns error code
- Messages don't appear in Instagram

**Solutions**:
```go
// Check Page Token is valid
// Verify token has correct permissions
// Check 24-hour message window
// Verify user hasn't blocked the page
// Check API error response details
```

#### 3. Webhooks Not Received

**Symptoms**:
- No webhook events received
- Messages sent but no notification

**Solutions**:
```bash
# Verify webhook subscriptions in Meta App Dashboard
# Check HTTPS certificate is valid
# Verify endpoint is publicly accessible
# Check webhook fields are subscribed
# Review Instagram settings for webhook fields
```

#### 4. Invalid Signature Errors

**Symptoms**:
- Webhooks rejected with signature error
- Error: "signature mismatch"

**Solutions**:
```go
// Verify app_secret is correct
// Check you're using the raw request body
// Ensure no body parsing middleware interferes
// Verify header name (case-insensitive)
```

### Debug Logging

Enable detailed logging:

```go
// Adapter logs key operations:
// 🌐 API URL being called
// 📦 Payload being sent
// ✅ Successful operations
// ❌ Errors with details
// 🔐 Verification attempts
```

### API Error Codes

| Code | Meaning | Solution |
|------|---------|----------|
| 190 | Invalid OAuth token | Regenerate Page Access Token |
| 200 | Permission denied | Add required permissions |
| 100 | Invalid parameter | Check message format |
| 368 | Temporarily blocked | Wait and retry |
| 10 | Permission denied | Check page roles |

## 📚 References

- [Instagram Messaging API Documentation](https://developers.facebook.com/docs/messenger-platform/instagram)
- [Graph API Reference](https://developers.facebook.com/docs/graph-api)
- [Webhook Reference](https://developers.facebook.com/docs/messenger-platform/webhooks)
- [Error Codes](https://developers.facebook.com/docs/graph-api/using-graph-api/error-handling)

## 📄 License

This adapter is part of the Relay project and follows the project's license terms.

## 🤝 Contributing

Contributions are welcome! Please ensure:

1. Code follows existing patterns
2. All functions are documented
3. Error handling is comprehensive
4. Logging is informative but not verbose
5. Tests pass (when test suite is available)

---

**Note**: This adapter requires an Instagram Business or Creator account connected to a Facebook Page. Personal Instagram accounts are not supported.
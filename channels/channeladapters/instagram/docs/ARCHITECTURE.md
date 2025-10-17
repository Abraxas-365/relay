# Instagram Adapter Architecture

## Overview

The Instagram adapter follows the same architectural patterns as the WhatsApp adapter, ensuring consistency and maintainability across all channel implementations in the Relay system.

## Table of Contents

- [Architecture Principles](#architecture-principles)
- [Component Structure](#component-structure)
- [Comparison with WhatsApp Adapter](#comparison-with-whatsapp-adapter)
- [Design Patterns](#design-patterns)
- [Data Flow](#data-flow)
- [Integration Points](#integration-points)
- [Extension Guide](#extension-guide)

## Architecture Principles

### 1. Separation of Concerns

Each component has a single, well-defined responsibility:

- **Adapter**: Business logic for Instagram API integration
- **Handler**: HTTP request/response handling
- **Routes**: Endpoint configuration and middleware chaining

### 2. Interface Compliance

Both adapters implement the `ChannelAdapter` interface:

```go
type ChannelAdapter interface {
    GetType() ChannelType
    SendMessage(ctx context.Context, msg OutgoingMessage) error
    ValidateConfig(config ChannelConfig) error
    ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*IncomingMessage, error)
    GetFeatures() ChannelFeatures
    TestConnection(ctx context.Context, config ChannelConfig) error
}
```

### 3. Configuration-Driven

Each channel type has its own configuration struct that implements `ChannelConfig`:

```go
type ChannelConfig interface {
    Validate() error
    GetProvider() string
    GetFeatures() ChannelFeatures
    GetType() ChannelType
}
```

### 4. Dependency Injection

External dependencies (Redis for WhatsApp, HTTP client for both) are injected at construction time, enabling testability and flexibility.

## Component Structure

### File Organization

Both adapters follow the same file structure:

```
channeladapters/
├── whatssapp/
│   ├── waa_adapter.go      # Core adapter implementation
│   ├── handler.go          # Webhook handlers
│   ├── routes.go           # Route configuration
│   ├── buffer.go           # WhatsApp-specific buffering
│   └── worker.go           # WhatsApp-specific worker
│
└── instagram/
    ├── ig_adapter.go       # Core adapter implementation
    ├── handler.go          # Webhook handlers
    ├── routes.go           # Route configuration
    ├── README.md           # Documentation
    ├── ARCHITECTURE.md     # This file
    └── example_usage.go    # Usage examples
```

### Component Responsibilities

#### 1. Adapter (`*_adapter.go`)

**Purpose**: Core business logic for API integration

**Responsibilities**:
- Implement `ChannelAdapter` interface
- Send messages via provider API
- Parse incoming webhooks
- Transform data between internal and provider formats
- Verify webhook signatures
- Handle API errors and retries

**Instagram Example**:
```go
type InstagramAdapter struct {
    config     channels.InstagramConfig
    httpClient *http.Client
    apiURL     string
}
```

**WhatsApp Example**:
```go
type WhatsAppAdapter struct {
    config        channels.WhatsAppConfig
    httpClient    *http.Client
    bufferService *BufferService
    apiURL        string
}
```

#### 2. Handler (`handler.go`)

**Purpose**: HTTP endpoint handlers for webhooks

**Responsibilities**:
- Handle webhook verification (GET requests)
- Receive webhook events (POST requests)
- Load channel configuration from repository
- Create adapter instance with channel-specific config
- Pass parsed messages to generic processor

**Pattern** (both adapters):
```go
type WebhookHandler struct {
    channelRepo channels.ChannelRepository
    adapter     *Adapter  // WhatsAppAdapter or InstagramAdapter
}

func (h *WebhookHandler) VerifyWebhook(c *fiber.Ctx) error
func (h *WebhookHandler) ReceiveWebhook(c *fiber.Ctx) error
```

#### 3. Routes (`routes.go`)

**Purpose**: Configure HTTP routes and middleware chains

**Responsibilities**:
- Register webhook endpoints
- Chain adapter-specific and generic handlers
- Configure middleware (if needed)

**Pattern** (both adapters):
```go
type WebhookRoutes struct {
    handler               *WebhookHandler
    messageProcessHandler fiber.Handler
}

func (wr *WebhookRoutes) RegisterRoutes(app *fiber.App)
```

## Comparison with WhatsApp Adapter

### Similarities

| Aspect | WhatsApp | Instagram | Notes |
|--------|----------|-----------|-------|
| **File Structure** | 5 files | 5 files | Same organization pattern |
| **Interface** | `ChannelAdapter` | `ChannelAdapter` | Both implement same interface |
| **Handler Pattern** | Verify + Receive | Verify + Receive | Identical webhook flow |
| **Route Pattern** | Group + Chain | Group + Chain | Same middleware chaining |
| **Config Pattern** | Struct + Validate | Struct + Validate | Same validation approach |
| **Signature Verification** | HMAC-SHA256 | HMAC-SHA256 | Same security mechanism |
| **Error Handling** | Typed errors | Typed errors | Same error pattern |
| **Logging** | Structured logs | Structured logs | Consistent logging |

### Differences

| Aspect | WhatsApp | Instagram | Reason |
|--------|----------|-----------|--------|
| **Buffering** | Yes (`buffer.go`) | No | WhatsApp-specific feature |
| **Worker** | Yes (`worker.go`) | No | Needed for buffer flush |
| **Dependencies** | Redis required | HTTP only | Buffering requires state |
| **API Base URL** | `graph.facebook.com` | `graph.facebook.com` | Both use Graph API |
| **Webhook Object** | `"whatsapp"` | `"instagram"` | Different webhook types |
| **Message Window** | No restriction | 24-hour limit | Platform difference |

### Architecture Comparison Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Channel Manager                           │
│  (manages all channel adapters and routes requests)         │
└─────────────────┬───────────────────────────┬───────────────┘
                  │                           │
        ┌─────────▼─────────┐       ┌────────▼──────────┐
        │  WhatsApp Adapter │       │ Instagram Adapter  │
        └─────────┬─────────┘       └────────┬───────────┘
                  │                           │
        ┌─────────▼─────────┐       ┌────────▼───────────┐
        │  WhatsApp Handler │       │ Instagram Handler   │
        └─────────┬─────────┘       └────────┬───────────┘
                  │                           │
        ┌─────────▼─────────┐       ┌────────▼───────────┐
        │  WhatsApp Routes  │       │ Instagram Routes    │
        └─────────┬─────────┘       └────────┬───────────┘
                  │                           │
        ┌─────────▼─────────┐       ┌────────▼───────────┐
        │   Buffer Service  │       │   (no buffer)       │
        │   Buffer Worker   │       │                     │
        └───────────────────┘       └─────────────────────┘
```

## Design Patterns

### 1. Adapter Pattern

Both implementations use the Adapter pattern to translate between:
- Internal Relay message format ↔ Provider-specific format
- Internal error types ↔ Provider error responses

```go
// Internal format
type OutgoingMessage struct {
    RecipientID string
    Content     MessageContent
    // ...
}

// Provider format (Instagram)
payload := map[string]any{
    "recipient": map[string]any{"id": msg.RecipientID},
    "message":   map[string]any{"text": msg.Content.Text},
}
```

### 2. Strategy Pattern

Different message types use different building strategies:

```go
func (a *InstagramAdapter) buildMessagePayload(msg channels.OutgoingMessage) map[string]any {
    switch msg.Content.Type {
    case "text":
        return a.buildTextMessage(msg)
    case "image":
        return a.buildImageMessage(msg)
    case "template":
        return a.buildTemplateMessage(msg)
    }
}
```

### 3. Chain of Responsibility

Webhook handling uses middleware chaining:

```go
webhooks.Post("/:tenantId/:channelId",
    handler.ReceiveWebhook,      // Parse provider-specific webhook
    messageProcessHandler,        // Process generic message
)
```

### 4. Factory Pattern

Channel Manager creates adapters based on configuration:

```go
func (cm *DefaultChannelManager) createAdapterForChannel(channel channels.Channel) (channels.ChannelAdapter, error) {
    switch channel.Type {
    case channels.ChannelTypeWhatsApp:
        config, _ := channel.GetConfigStruct()
        return whatsapp.NewWhatsAppAdapter(config.(channels.WhatsAppConfig), cm.redisClient)
    
    case channels.ChannelTypeInstagram:
        config, _ := channel.GetConfigStruct()
        return instagram.NewInstagramAdapter(config.(channels.InstagramConfig))
    }
}
```

### 5. Template Method Pattern

Webhook processing follows the same template:

```go
// Template
func ProcessWebhook(ctx, payload, headers) (*IncomingMessage, error) {
    // 1. Verify signature (security)
    verifySignature(payload, headers)
    
    // 2. Parse webhook (provider-specific)
    webhook := parseWebhook(payload)
    
    // 3. Extract message (provider-specific)
    message := extractMessage(webhook)
    
    // 4. Return standardized format
    return message, nil
}
```

## Data Flow

### Outgoing Message Flow

```
┌──────────────┐
│ Application  │
│   Code       │
└──────┬───────┘
       │ OutgoingMessage
       ▼
┌──────────────┐
│   Channel    │
│   Manager    │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Adapter    │ ◄── Transform to provider format
│  (Instagram) │
└──────┬───────┘
       │ HTTP POST
       ▼
┌──────────────┐
│  Instagram   │
│  Graph API   │
└──────────────┘
```

### Incoming Message Flow

```
┌──────────────┐
│  Instagram   │
│  Graph API   │
└──────┬───────┘
       │ Webhook POST
       ▼
┌──────────────┐
│   Routes     │ ◄── Route to handler
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Handler    │ ◄── Load channel config
│ ReceiveWebhook│
└──────┬───────┘
       │
       ▼
┌──────────────┐
│   Adapter    │ ◄── Verify & parse webhook
│ProcessWebhook│
└──────┬───────┘
       │ IncomingMessage
       ▼
┌──────────────┐
│   Generic    │ ◄── Store, route, trigger workflows
│   Processor  │
└──────────────┘
```

## Integration Points

### 1. Channel Manager Integration

```go
// Register adapter factory
func (cm *DefaultChannelManager) createAdapterForChannel(channel channels.Channel) {
    case channels.ChannelTypeInstagram:
        config, _ := channel.GetConfigStruct()
        return instagram.NewInstagramAdapter(config.(channels.InstagramConfig))
}
```

### 2. Route Registration

```go
// In main.go or router setup
instagramHandler := instagram.NewWebhookHandler(channelRepo, nil)
instagramRoutes := instagram.NewWebhookRoutes(
    instagramHandler,
    genericMessageHandler,
)
instagramRoutes.RegisterRoutes(app)
```

### 3. Configuration Integration

```go
// In channel.go
func (c *Channel) GetConfigStruct() (ChannelConfig, error) {
    switch c.Type {
    case ChannelTypeInstagram:
        var config InstagramConfig
        json.Unmarshal(c.Config, &config)
        return config, nil
    }
}
```

## Extension Guide

### Adding a New Channel Type

Follow this pattern based on WhatsApp and Instagram:

#### 1. Define Configuration

```go
// In relay/channels/channel.go
type NewChannelConfig struct {
    Provider string `json:"provider"`
    // ... provider-specific fields
}

func (c NewChannelConfig) Validate() error { /* ... */ }
func (c NewChannelConfig) GetProvider() string { /* ... */ }
func (c NewChannelConfig) GetType() ChannelType { /* ... */ }
func (c NewChannelConfig) GetFeatures() ChannelFeatures { /* ... */ }
```

#### 2. Create Adapter

```go
// In relay/channels/channeladapters/newchannel/adapter.go
type NewChannelAdapter struct {
    config     channels.NewChannelConfig
    httpClient *http.Client
}

func NewAdapter(config channels.NewChannelConfig) *NewChannelAdapter {
    return &NewChannelAdapter{
        config:     config,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

// Implement ChannelAdapter interface
func (a *NewChannelAdapter) GetType() channels.ChannelType { /* ... */ }
func (a *NewChannelAdapter) SendMessage(ctx context.Context, msg channels.OutgoingMessage) error { /* ... */ }
func (a *NewChannelAdapter) ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*channels.IncomingMessage, error) { /* ... */ }
func (a *NewChannelAdapter) ValidateConfig(config channels.ChannelConfig) error { /* ... */ }
func (a *NewChannelAdapter) GetFeatures() channels.ChannelFeatures { /* ... */ }
func (a *NewChannelAdapter) TestConnection(ctx context.Context, config channels.ChannelConfig) error { /* ... */ }
```

#### 3. Create Handler

```go
// In relay/channels/channeladapters/newchannel/handler.go
type WebhookHandler struct {
    channelRepo channels.ChannelRepository
    adapter     *NewChannelAdapter
}

func (h *WebhookHandler) VerifyWebhook(c *fiber.Ctx) error { /* ... */ }
func (h *WebhookHandler) ReceiveWebhook(c *fiber.Ctx) error { /* ... */ }
```

#### 4. Create Routes

```go
// In relay/channels/channeladapters/newchannel/routes.go
type WebhookRoutes struct {
    handler               *WebhookHandler
    messageProcessHandler fiber.Handler
}

func (wr *WebhookRoutes) RegisterRoutes(app *fiber.App) { /* ... */ }
```

#### 5. Register in Channel Manager

```go
// In relay/channels/channelmanager/manager.go
import newchannel "github.com/Abraxas-365/relay/channels/channeladapters/newchannel"

func (cm *DefaultChannelManager) createAdapterForChannel(channel channels.Channel) {
    // ...
    case channels.ChannelTypeNewChannel:
        config, _ := channel.GetConfigStruct()
        return newchannel.NewAdapter(config.(channels.NewChannelConfig))
}
```

### Checklist for New Adapters

- [ ] Configuration struct with validation
- [ ] Adapter implementing `ChannelAdapter` interface
- [ ] Webhook handler with verification and receiving
- [ ] Routes configuration
- [ ] Integration in channel manager
- [ ] Documentation (README.md)
- [ ] Usage examples
- [ ] Error handling
- [ ] Logging
- [ ] Security (signature verification)

## Best Practices

### 1. Error Handling

Use typed errors for consistency:

```go
// Good
return channels.ErrProviderAPIError().
    WithDetail("status", resp.StatusCode).
    WithDetail("error_message", apiError.Message)

// Avoid
return fmt.Errorf("API error: %d", resp.StatusCode)
```

### 2. Logging

Use structured logging with emojis for visibility:

```go
log.Printf("✅ Message sent successfully")
log.Printf("❌ Failed to send: %v", err)
log.Printf("🔐 Verifying webhook signature")
log.Printf("📥 Received webhook event")
```

### 3. Configuration

Always validate configuration early:

```go
func NewAdapter(config channels.Config) *Adapter {
    if err := config.Validate(); err != nil {
        panic("invalid configuration") // Or handle gracefully
    }
    // ...
}
```

### 4. Testing

Structure code to be testable:

```go
// Inject dependencies
func NewAdapter(config Config, httpClient *http.Client) *Adapter

// Use interfaces where appropriate
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}
```

### 5. Documentation

Document all public functions and structures:

```go
// SendMessage sends a message via Instagram Direct Message
//
// Supports:
//   - Text messages
//   - Images with optional captions
//   - Videos
//
// Parameters:
//   - ctx: Context for request cancellation
//   - msg: Outgoing message with recipient and content
//
// Returns:
//   - error: nil if successful
```

## Security Considerations

### 1. Webhook Signature Verification

Always verify webhook signatures:

```go
func (a *Adapter) verifySignature(payload []byte, headers map[string]string) error {
    signature := headers["X-Hub-Signature-256"]
    mac := hmac.New(sha256.New, []byte(a.config.AppSecret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    
    if !hmac.Equal([]byte(signature), []byte(expected)) {
        return channels.ErrInvalidWebhookSignature()
    }
    return nil
}
```

### 2. Token Security

- Never log access tokens or secrets
- Store tokens securely (encrypted in database)
- Rotate tokens periodically
- Use environment variables for sensitive config

### 3. Input Validation

Validate all incoming data:

```go
if err := config.Validate(); err != nil {
    return err
}

if msg.RecipientID == "" {
    return channels.ErrInvalidMessage()
}
```

## Performance Considerations

### 1. Connection Pooling

Reuse HTTP connections:

```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 100,
    },
}
```

### 2. Context Timeouts

Always use context with timeout:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
```

### 3. Rate Limiting

Implement rate limiting for API calls:

```go
// Example with rate limiter
limiter := rate.NewLimiter(rate.Limit(10), 1) // 10 requests per second
limiter.Wait(ctx)
adapter.SendMessage(ctx, msg)
```

## Conclusion

The Instagram adapter maintains architectural consistency with the WhatsApp adapter while accommodating platform-specific differences. This consistency:

1. **Reduces Learning Curve**: Developers familiar with one adapter can quickly work with others
2. **Improves Maintainability**: Common patterns make code easier to maintain
3. **Enables Scalability**: New channels can be added following the same pattern
4. **Ensures Quality**: Proven patterns reduce bugs and issues
5. **Facilitates Testing**: Consistent structure makes testing easier

By following these architectural patterns, the Relay system maintains a clean, scalable codebase that can easily accommodate new channel types while providing a consistent experience for developers.
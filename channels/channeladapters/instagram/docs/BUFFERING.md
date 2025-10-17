# Instagram Message Buffering

## 📋 Table of Contents

- [Overview](#overview)
- [Why Buffering?](#why-buffering)
- [How It Works](#how-it-works)
- [Configuration](#configuration)
- [Use Cases](#use-cases)
- [Architecture](#architecture)
- [Performance Impact](#performance-impact)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## 🎯 Overview

Message buffering is an optional feature that combines multiple rapid messages from the same Instagram user into a single message before processing. This improves conversation context and reduces processing overhead.

**Example:**

Without buffering:
```
User sends: "Hey"          → Process immediately
User sends: "Can you"      → Process immediately  
User sends: "help me?"     → Process immediately
```

With buffering (5 seconds):
```
User sends: "Hey"          → Start buffer timer (5s)
User sends: "Can you"      → Add to buffer
User sends: "help me?"     → Add to buffer
[5 seconds pass]
Combined: "Hey\nCan you\nhelp me?" → Process once
```

## 🤔 Why Buffering?

### Benefits

1. **Better Context**: AI/chatbots get complete thoughts instead of fragments
2. **Reduced Processing**: One webhook processing instead of three
3. **Cost Savings**: Fewer API calls to downstream services
4. **Improved UX**: More natural conversation flow
5. **Resource Efficiency**: Less server load from rapid messages

### When to Use

✅ **Use buffering when:**
- Users frequently send multiple quick messages
- You have AI/NLP processing that benefits from complete context
- You want to reduce processing overhead
- Conversation quality is more important than instant response

❌ **Don't use buffering when:**
- Real-time response is critical (e.g., emergency services)
- Every message needs immediate action
- Users expect instant acknowledgment
- Messages are typically complete thoughts

## ⚙️ Configuration

### Basic Configuration

```go
config := channels.InstagramConfig{
    Provider:    "meta",
    PageID:      "your_page_id",
    PageToken:   "your_token",
    
    // Buffering settings
    BufferEnabled:        true,  // Enable buffering
    BufferTimeSeconds:    5,     // Wait 5 seconds before processing
    BufferResetOnMessage: false, // Don't reset timer on new messages
}
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `BufferEnabled` | bool | `false` | Enable/disable buffering |
| `BufferTimeSeconds` | int | `5` | Time to wait before flushing (1-60 seconds) |
| `BufferResetOnMessage` | bool | `false` | Reset timer when new message arrives |

### Buffer Time Settings

```go
// Conservative (faster response, less combining)
BufferTimeSeconds: 2

// Balanced (good for most use cases)
BufferTimeSeconds: 5

// Aggressive (more combining, slower response)
BufferTimeSeconds: 10
```

### Reset Behavior

#### `BufferResetOnMessage: false` (Fixed Window)

```
Message 1 arrives at t=0    → Start 5s timer
Message 2 arrives at t=2    → Timer continues
Message 3 arrives at t=4    → Timer continues
Timer expires at t=5        → Flush all 3 messages
```

**Best for:** Predictable timing, consistent latency

#### `BufferResetOnMessage: true` (Sliding Window)

```
Message 1 arrives at t=0    → Start 5s timer
Message 2 arrives at t=2    → Reset timer (now expires at t=7)
Message 3 arrives at t=4    → Reset timer (now expires at t=9)
Timer expires at t=9        → Flush all 3 messages
```

**Best for:** Capturing complete conversations, users who type in bursts

## 🔄 How It Works

### Component Architecture

```
┌─────────────────────────────────────────────────┐
│          Instagram Webhook Received              │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│        Instagram Adapter (ProcessWebhook)        │
│  - Parse webhook payload                         │
│  - Extract incoming message                      │
└─────────────────┬───────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────┐
│         BufferService.AddMessage()               │
│  - Check if buffering enabled                    │
│  - Get/create buffer for user                    │
│  - Add message to buffer                         │
│  - Set/reset timer                               │
│  - Return nil (buffered) or message (process)    │
└─────────────────┬───────────────────────────────┘
                  │
        ┌─────────┴──────────┐
        │                    │
        ▼                    ▼
   [Buffered]           [Process Now]
        │                    │
        │                    ▼
        │           ┌──────────────────┐
        │           │ Message Processor │
        │           └──────────────────┘
        │
        ▼
┌─────────────────────────────────────────────────┐
│    BufferWorker (runs every 2 seconds)          │
│  - Scan all buffers                             │
│  - Check for expired timers                     │
│  - Flush expired buffers                        │
│  - Call message processor                       │
└─────────────────────────────────────────────────┘
```

### Data Flow

1. **Message Arrives**: Instagram sends webhook
2. **Parse**: Adapter extracts message data
3. **Buffer Check**: BufferService checks if should buffer
4. **Store**: Message stored in Redis with TTL
5. **Timer**: Timer key set in Redis
6. **Wait**: System waits for timer to expire
7. **Worker**: BufferWorker checks expired timers
8. **Combine**: All buffered messages combined
9. **Process**: Combined message sent to processor

### Redis Keys

```
# Buffer data
relay:instagram:buffer:{channelID}:{senderID}

# Timer flag
relay:instagram:buffer:timer:{channelID}:{senderID}
```

**Example:**
```
# Buffer for user 123 on channel abc
relay:instagram:buffer:abc123:user_123

# Timer for same user
relay:instagram:buffer:timer:abc123:user_123
```

## 💡 Use Cases

### Use Case 1: Customer Support Chatbot

**Scenario:** Customer asking multiple questions

**Without Buffering:**
```
Customer: "Hi"
Bot: "Hello! How can I help?"
Customer: "I have"
Bot: "Yes?"
Customer: "a problem"
Bot: "Please describe your problem"
Customer: "with my order"
Bot: "What's your order number?"
```

**With Buffering (5s):**
```
Customer: "Hi"
Customer: "I have"
Customer: "a problem"
Customer: "with my order"
[Buffer combines]
Bot receives: "Hi\nI have\na problem\nwith my order"
Bot: "I understand you have a problem with your order. What's your order number?"
```

### Use Case 2: AI-Powered Responses

**Problem:** AI needs full context to generate quality responses

```go
// Without buffering - AI gets fragments
"Can you" → AI confused, asks clarifying questions

// With buffering - AI gets complete question
"Can you recommend a product for sensitive skin?" → AI provides relevant answer
```

### Use Case 3: High-Volume Processing

**Benefits:**
- Reduce webhook processing from 1000 to 400 (60% reduction)
- Lower API costs to external services
- Better resource utilization
- Improved database write efficiency

## 🏗️ Architecture Details

### Buffer Service

```go
type BufferService struct {
    redis  *redis.Client
    config BufferConfig
}
```

**Responsibilities:**
- Store messages in Redis
- Manage buffer timers
- Combine messages on flush
- Provide buffer statistics

### Buffer Worker

```go
type BufferWorker struct {
    redis         *redis.Client
    bufferService *BufferService
    interval      time.Duration
    stopChan      chan struct{}
}
```

**Responsibilities:**
- Scan Redis for expired buffers
- Flush expired buffers
- Call message processor
- Handle graceful shutdown

### Message Combination Logic

```go
func combineMessages(buffer *MessageBuffer) *IncomingMessage {
    // Combine text with line breaks
    combinedText := strings.Join(allTexts, "\n")
    
    // Merge all attachments
    allAttachments := append(msg1.Attachments, msg2.Attachments...)
    
    // Add buffer metadata
    metadata["buffered"] = true
    metadata["message_count"] = len(buffer.Messages)
    metadata["buffer_duration_seconds"] = duration
    
    return combinedMessage
}
```

## 📊 Performance Impact

### Memory Usage

```
Per Buffer: ~2KB (typical)
1000 active buffers: ~2MB RAM
10000 active buffers: ~20MB RAM
```

**Redis Storage:**
- Buffer data: JSON serialized message array
- Timer key: Simple flag
- TTL: Automatic cleanup

### Latency Impact

| Configuration | Added Latency | Trade-off |
|--------------|---------------|-----------|
| Disabled | 0ms | No buffering benefit |
| 2 seconds | +2000ms | Minimal buffering |
| 5 seconds | +5000ms | Good balance |
| 10 seconds | +10000ms | Maximum buffering |

### Processing Reduction

Example with 5-second buffer:

```
Without Buffering:
- 1000 webhooks received
- 1000 processed individually
- Processing time: 1000 * 50ms = 50 seconds

With Buffering (60% reduction):
- 1000 webhooks received
- 400 combined messages processed
- Processing time: 400 * 50ms = 20 seconds
- Savings: 30 seconds (60%)
```

## 📈 Monitoring

### Key Metrics to Track

1. **Active Buffers**: Current number of buffers in Redis
2. **Buffer Hit Rate**: % of messages that get buffered vs processed immediately
3. **Average Messages Per Buffer**: How many messages typically combined
4. **Flush Latency**: Time from first message to flush
5. **Worker Performance**: How long buffer checks take

### Getting Stats

```go
// Get buffer statistics
stats, err := bufferService.GetBufferStats(ctx)

// Returns:
{
    "active_buffers": 42,
    "active_timers": 42,
    "buffer_enabled": true,
    "buffer_time_seconds": 5,
    "reset_on_message": false
}
```

### Worker Stats

```go
// Get worker statistics
stats := worker.GetStats(ctx)

// Returns:
{
    "is_running": true,
    "check_interval": "2s",
    "worker_type": "instagram_buffer_worker",
    "active_buffers": 42
}
```

### Logging

The system logs key events:

```
📦 Instagram message buffered for channel: abc123, sender: user_456
🔍 Instagram buffer check complete: checked=50, flushed=12
📤 Instagram buffer flushed: channel=abc123, sender=user_456, messages=3
✅ Instagram buffer worker: message processed successfully
```

## 🐛 Troubleshooting

### Problem: Messages Not Being Buffered

**Symptoms:** All messages processed immediately

**Checks:**
```go
// 1. Verify buffering is enabled
if !config.BufferEnabled {
    log.Println("Buffering is disabled")
}

// 2. Check Redis connection
err := redisClient.Ping(ctx).Err()
if err != nil {
    log.Printf("Redis not available: %v", err)
}

// 3. Check buffer service initialization
if adapter.bufferService == nil {
    log.Println("Buffer service not initialized")
}
```

### Problem: Messages Stuck in Buffer

**Symptoms:** Messages never processed, lost messages

**Checks:**
```bash
# Check if worker is running
# Should see periodic logs like:
🔍 Instagram buffer check complete: checked=X, flushed=Y

# If not running:
# 1. Verify worker is started
# 2. Check for worker errors
# 3. Verify Redis connectivity
```

**Manual Flush:**
```go
// Force flush all buffers
count, err := worker.FlushAll(ctx, processMessage)
log.Printf("Manually flushed %d buffers", count)
```

### Problem: Buffer Timer Not Expiring

**Symptoms:** Messages wait forever

**Checks:**
```bash
# Check Redis TTL on timer key
redis-cli TTL "relay:instagram:buffer:timer:channel_123:user_456"

# Should return seconds remaining, e.g., 3
# If returns -1: Key exists but no TTL (bug)
# If returns -2: Key doesn't exist
```

**Fix:**
```bash
# Delete stuck timer
redis-cli DEL "relay:instagram:buffer:timer:channel_123:user_456"
```

### Problem: High Memory Usage

**Symptoms:** Redis memory growing

**Checks:**
```bash
# Check number of buffer keys
redis-cli KEYS "relay:instagram:buffer:*" | wc -l

# Check Redis memory
redis-cli INFO memory
```

**Solutions:**
1. Reduce `BufferTimeSeconds`
2. Implement max buffer size limit
3. Add memory monitoring alerts
4. Clear old buffers: `worker.FlushAll()`

## ✅ Best Practices

### 1. Start Conservative

```go
// Start with minimal buffering
config := channels.InstagramConfig{
    BufferEnabled:        true,
    BufferTimeSeconds:    2,  // Start small
    BufferResetOnMessage: false,
}

// Monitor results, then adjust
```

### 2. Monitor and Adjust

```go
// Track metrics
stats, _ := bufferService.GetBufferStats(ctx)
log.Printf("Buffers: %v, Time: %v", 
    stats["active_buffers"], 
    stats["buffer_time_seconds"])

// Adjust based on patterns
if avgMessagesPerBuffer < 2 {
    // Increase buffer time
    config.BufferTimeSeconds = 7
}
```

### 3. Handle Edge Cases

```go
// Set maximum messages per buffer
config.MaxMessagesPerBuffer = 10

// Prevents memory issues from spam
```

### 4. Graceful Shutdown

```go
// On application shutdown
func shutdown() {
    // Stop worker
    worker.Stop()
    
    // Flush all pending buffers
    worker.FlushAll(ctx, processMessage)
    
    // Close Redis
    redisClient.Close()
}
```

### 5. Testing

```go
// Test with buffering disabled first
config.BufferEnabled = false

// Once working, enable buffering
config.BufferEnabled = true

// Test buffer behavior
sendTestMessages(3, 1*time.Second) // Send 3 messages, 1s apart
time.Sleep(6 * time.Second)        // Wait for flush
verifyMessagesCombined()           // Check result
```

### 6. User Experience

Consider these factors:

- **Critical Messages**: Disable buffering for urgent channels
- **User Expectations**: Longer buffers = slower response
- **Message Types**: Text benefits most, media less so
- **Peak Times**: More users = more buffers = more memory

### 7. Production Configuration

```go
// Recommended production settings
config := channels.InstagramConfig{
    BufferEnabled:        true,
    BufferTimeSeconds:    5,     // Sweet spot
    BufferResetOnMessage: false, // Predictable timing
}

// Worker configuration
worker := NewBufferWorker(redis, bufferService, 2*time.Second)
```

## 🔗 Related Documentation

- [README.md](./README.md) - Main documentation
- [ARCHITECTURE.md](./ARCHITECTURE.md) - Technical architecture
- [buffer.go](./buffer.go) - Buffer service implementation
- [worker.go](./worker.go) - Worker implementation

## 📞 Support

For issues or questions:
1. Check logs for error messages
2. Verify Redis connectivity
3. Check buffer stats
4. Review this documentation
5. Check example_usage.go for patterns

---

**Summary:** Message buffering is a powerful feature that improves conversation quality and reduces processing overhead. Start with conservative settings, monitor behavior, and adjust based on your specific use case.
package instagram

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
)

// BufferWorker periodically checks and flushes expired Instagram message buffers
//
// The worker runs in the background, scanning Redis for buffers whose timers
// have expired and flushing them for processing. This ensures messages don't
// get stuck in buffers indefinitely.
//
// Features:
//   - Periodic buffer checking (configurable interval)
//   - Graceful shutdown support
//   - Automatic flush of expired buffers
//   - Callback support for processed messages
//   - Error resilience (continues on errors)
//
// Usage:
//
//	worker := NewBufferWorker(redisClient, bufferService, 2*time.Second)
//	go worker.Start(ctx, func(ctx context.Context, msg *channels.IncomingMessage) error {
//	    // Process the flushed message
//	    return processMessage(ctx, msg)
//	})
type BufferWorker struct {
	redis         *redis.Client
	bufferService *BufferService
	interval      time.Duration
	stopChan      chan struct{}
	isRunning     bool
}

// NewBufferWorker creates a new Instagram buffer worker
//
// Parameters:
//   - redisClient: Redis client for scanning buffer keys
//   - bufferService: Buffer service for flushing operations
//   - interval: How often to check for expired buffers (e.g., 2*time.Second)
//
// Returns:
//   - *BufferWorker: Configured worker ready to start
//
// Example:
//
//	worker := NewBufferWorker(redisClient, bufferService, 2*time.Second)
func NewBufferWorker(
	redisClient *redis.Client,
	bufferService *BufferService,
	interval time.Duration,
) *BufferWorker {
	if interval <= 0 {
		interval = 2 * time.Second // Default to 2 seconds
	}

	return &BufferWorker{
		redis:         redisClient,
		bufferService: bufferService,
		interval:      interval,
		stopChan:      make(chan struct{}),
		isRunning:     false,
	}
}

// Start starts the buffer worker in the current goroutine
//
// This method blocks and should typically be called in a goroutine:
//
//	go worker.Start(ctx, handleMessage)
//
// The worker will:
//  1. Check for expired buffers every `interval`
//  2. Flush expired buffers
//  3. Call onFlush callback for each flushed message
//  4. Continue until context is cancelled or Stop() is called
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - onFlush: Callback function called for each flushed message
//
// Example:
//
//	ctx := context.Background()
//	go worker.Start(ctx, func(ctx context.Context, msg *channels.IncomingMessage) error {
//	    log.Printf("Processing buffered message from %s", msg.SenderID)
//	    return messageProcessor.Process(ctx, msg)
//	})
func (w *BufferWorker) Start(ctx context.Context, onFlush func(context.Context, *channels.IncomingMessage) error) {
	if w.isRunning {
		log.Println("⚠️  Instagram buffer worker already running")
		return
	}

	w.isRunning = true
	log.Printf("🚀 Instagram buffer worker started (interval: %v)", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Initial check on startup
	w.checkBuffers(ctx, onFlush)

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️  Instagram buffer worker stopped (context cancelled)")
			w.isRunning = false
			return

		case <-w.stopChan:
			log.Println("⏹️  Instagram buffer worker stopped (stop signal received)")
			w.isRunning = false
			return

		case <-ticker.C:
			// Periodic buffer check
			w.checkBuffers(ctx, onFlush)
		}
	}
}

// Stop gracefully stops the buffer worker
//
// This method signals the worker to stop and returns immediately.
// The worker will stop after the current check completes.
//
// Example:
//
//	worker.Stop()
//	time.Sleep(100 * time.Millisecond) // Wait for graceful shutdown
func (w *BufferWorker) Stop() {
	if !w.isRunning {
		log.Println("⚠️  Instagram buffer worker not running")
		return
	}

	log.Println("🛑 Stopping Instagram buffer worker...")
	close(w.stopChan)
}

// IsRunning returns whether the worker is currently running
func (w *BufferWorker) IsRunning() bool {
	return w.isRunning
}

// checkBuffers checks all Instagram buffers and flushes expired ones
//
// This method:
//  1. Scans Redis for all Instagram buffer keys
//  2. Checks each buffer's timer status
//  3. Flushes buffers with expired timers
//  4. Calls the onFlush callback for each flushed message
//
// The method is resilient to errors and will continue processing
// even if individual buffer flushes fail.
func (w *BufferWorker) checkBuffers(ctx context.Context, onFlush func(context.Context, *channels.IncomingMessage) error) {
	// Scan for all Instagram buffer keys (excluding timer keys)
	var cursor uint64
	pattern := "relay:instagram:buffer:*"
	checkedCount := 0
	flushedCount := 0

	for {
		keys, nextCursor, err := w.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			log.Printf("❌ Instagram buffer worker: failed to scan keys: %v", err)
			break
		}

		for _, key := range keys {
			// Skip timer keys - we only want buffer keys
			if strings.Contains(key, ":timer:") {
				continue
			}

			checkedCount++

			// Extract channel ID and sender ID from key
			// Format: relay:instagram:buffer:{channelID}:{senderID}
			parts := strings.Split(key, ":")
			if len(parts) < 5 {
				log.Printf("⚠️  Instagram buffer worker: invalid key format: %s", key)
				continue
			}

			channelID := kernel.NewChannelID(parts[3])
			senderID := parts[4]

			// Try to flush this buffer
			msg, err := w.bufferService.CheckAndFlush(ctx, channelID, senderID)
			if err != nil {
				log.Printf("❌ Instagram buffer worker: failed to flush buffer for channel=%s, sender=%s: %v",
					channelID, senderID, err)
				continue
			}

			// If no message, buffer wasn't ready to flush (timer not expired)
			if msg == nil {
				continue
			}

			// Buffer was flushed
			flushedCount++
			log.Printf("📤 Instagram buffer flushed: channel=%s, sender=%s, messages=%d",
				channelID,
				senderID,
				msg.Metadata["message_count"])

			// Call flush callback if provided
			if onFlush != nil {
				if err := onFlush(ctx, msg); err != nil {
					log.Printf("❌ Instagram buffer worker: onFlush callback failed for channel=%s, sender=%s: %v",
						channelID, senderID, err)
					// Continue processing other buffers even if callback fails
				} else {
					log.Printf("✅ Instagram buffer worker: message processed successfully for sender=%s", senderID)
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	// Log summary if any buffers were checked or flushed
	if checkedCount > 0 || flushedCount > 0 {
		log.Printf("🔍 Instagram buffer check complete: checked=%d, flushed=%d", checkedCount, flushedCount)
	}
}

// FlushAll immediately flushes all Instagram buffers regardless of timer status
//
// This is useful for:
//   - Graceful shutdown (flush all pending messages)
//   - Manual intervention (admin triggers flush)
//   - Testing purposes
//
// Parameters:
//   - ctx: Context for operations
//   - onFlush: Callback for each flushed message
//
// Returns:
//   - int: Number of buffers flushed
//   - error: Any error during flush operation
//
// Example:
//
//	count, err := worker.FlushAll(ctx, handleMessage)
//	log.Printf("Flushed %d buffers", count)
func (w *BufferWorker) FlushAll(ctx context.Context, onFlush func(context.Context, *channels.IncomingMessage) error) (int, error) {
	log.Println("🌊 Flushing all Instagram buffers...")

	var cursor uint64
	pattern := "relay:instagram:buffer:*"
	flushedCount := 0

	for {
		keys, nextCursor, err := w.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return flushedCount, err
		}

		for _, key := range keys {
			// Skip timer keys
			if strings.Contains(key, ":timer:") {
				continue
			}

			// Extract channel ID and sender ID
			parts := strings.Split(key, ":")
			if len(parts) < 5 {
				continue
			}

			channelID := kernel.NewChannelID(parts[3])
			senderID := parts[4]

			// Force flush
			msg, err := w.bufferService.FlushNow(ctx, channelID, senderID)
			if err != nil {
				log.Printf("❌ Failed to flush buffer: channel=%s, sender=%s: %v", channelID, senderID, err)
				continue
			}

			if msg == nil {
				continue
			}

			flushedCount++

			// Call callback
			if onFlush != nil {
				if err := onFlush(ctx, msg); err != nil {
					log.Printf("❌ onFlush callback failed: %v", err)
				}
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	log.Printf("✅ Flushed %d Instagram buffers", flushedCount)
	return flushedCount, nil
}

// GetStats returns statistics about the worker and buffers
//
// Returns information useful for monitoring and debugging
func (w *BufferWorker) GetStats(ctx context.Context) map[string]any {
	stats := map[string]any{
		"is_running":     w.isRunning,
		"check_interval": w.interval.String(),
		"worker_type":    "instagram_buffer_worker",
		"pattern":        "relay:instagram:buffer:*",
	}

	// Get buffer service stats
	if bufferStats, err := w.bufferService.GetBufferStats(ctx); err == nil {
		for k, v := range bufferStats {
			stats[k] = v
		}
	}

	return stats
}

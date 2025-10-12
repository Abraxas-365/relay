package whatsapp

import (
	"context"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
)

// BufferWorker periodically checks and flushes expired buffers
type BufferWorker struct {
	redis         *redis.Client
	bufferService *BufferService
	interval      time.Duration
	stopChan      chan struct{}
}

// NewBufferWorker creates a new buffer worker
func NewBufferWorker(
	redisClient *redis.Client,
	bufferService *BufferService,
	interval time.Duration,
) *BufferWorker {
	return &BufferWorker{
		redis:         redisClient,
		bufferService: bufferService,
		interval:      interval,
		stopChan:      make(chan struct{}),
	}
}

// Start starts the buffer worker
func (w *BufferWorker) Start(ctx context.Context, onFlush func(context.Context, *channels.IncomingMessage) error) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.checkBuffers(ctx, onFlush)
		}
	}
}

// Stop stops the buffer worker
func (w *BufferWorker) Stop() {
	close(w.stopChan)
}

// checkBuffers checks all buffers and flushes expired ones
func (w *BufferWorker) checkBuffers(ctx context.Context, onFlush func(context.Context, *channels.IncomingMessage) error) {
	// Scan for all buffer keys
	var cursor uint64
	pattern := "relay:buffer:*"

	for {
		keys, nextCursor, err := w.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}

		for _, key := range keys {
			// Skip timer keys
			if strings.Contains(key, ":timer:") {
				continue
			}

			// Extract channel ID and sender ID from key
			parts := strings.Split(key, ":")
			if len(parts) < 4 {
				continue
			}

			channelID := parts[2]
			senderID := parts[3]

			// Try to flush
			msg, err := w.bufferService.CheckAndFlush(ctx, kernel.NewChannelID(channelID), senderID)
			if err != nil || msg == nil {
				continue
			}

			// Call flush callback
			if onFlush != nil {
				onFlush(ctx, msg)
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

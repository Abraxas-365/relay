package delayscheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/relay/engine"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

const (
	delayedExecutionsKey = "relay:delayed_executions" // Sorted set
	continuationPrefix   = "relay:continuation:"      // Hash keys
	syncDelayThreshold   = 30 * time.Second
)

var _ engine.DelayScheduler = (*RedisDelayScheduler)(nil)

type RedisDelayScheduler struct {
	redis          *redis.Client
	syncThreshold  time.Duration
	onContinuation engine.ContinuationHandler
	workerRunning  bool
	stopChan       chan struct{}
}

func NewRedisDelayScheduler(
	redisClient *redis.Client,
	handler engine.ContinuationHandler,
) *RedisDelayScheduler {
	return &RedisDelayScheduler{
		redis:          redisClient,
		syncThreshold:  syncDelayThreshold,
		onContinuation: handler,
		stopChan:       make(chan struct{}),
	}
}

// Schedule schedules a workflow continuation
func (r *RedisDelayScheduler) Schedule(
	ctx context.Context,
	continuation *engine.WorkflowContinuation,
	delay time.Duration,
) error {
	if continuation.ID == "" {
		continuation.ID = uuid.New().String()
	}

	continuation.ScheduledFor = time.Now().Add(delay)
	continuation.CreatedAt = time.Now()

	// Serialize continuation
	data, err := json.Marshal(continuation)
	if err != nil {
		return fmt.Errorf("failed to marshal continuation: %w", err)
	}

	// Store continuation data
	key := fmt.Sprintf("%s%s", continuationPrefix, continuation.ID)
	if err := r.redis.Set(ctx, key, data, delay+time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to store continuation: %w", err)
	}

	// Add to sorted set with execution timestamp as score
	score := float64(continuation.ScheduledFor.Unix())
	if err := r.redis.ZAdd(ctx, delayedExecutionsKey, &redis.Z{
		Score:  score,
		Member: continuation.ID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to schedule continuation: %w", err)
	}

	log.Printf("⏰ Scheduled continuation %s for %v (delay: %v)",
		continuation.ID, continuation.ScheduledFor, delay)

	return nil
}

// ShouldUseAsync determines if delay should be async
func (r *RedisDelayScheduler) ShouldUseAsync(duration time.Duration) bool {
	return duration > r.syncThreshold
}

// StartWorker starts the background worker
func (r *RedisDelayScheduler) StartWorker(ctx context.Context) {
	if r.workerRunning {
		log.Println("⚠️  Delay scheduler worker already running")
		return
	}

	r.workerRunning = true
	log.Println("🚀 Starting delay scheduler worker...")

	go r.workerLoop(ctx)
}

// StopWorker stops the background worker
func (r *RedisDelayScheduler) StopWorker() {
	if !r.workerRunning {
		return
	}

	log.Println("🛑 Stopping delay scheduler worker...")
	close(r.stopChan)
	r.workerRunning = false
}

func (r *RedisDelayScheduler) workerLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️  Delay scheduler worker stopped (context done)")
			return
		case <-r.stopChan:
			log.Println("⏹️  Delay scheduler worker stopped")
			return
		case <-ticker.C:
			if err := r.processDueExecutions(ctx); err != nil {
				log.Printf("❌ Error processing due executions: %v", err)
			}
		}
	}
}

func (r *RedisDelayScheduler) processDueExecutions(ctx context.Context) error {
	now := float64(time.Now().Unix())

	// Get jobs due for execution
	jobs, err := r.redis.ZRangeByScore(ctx, delayedExecutionsKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%f", now),
		Count: 10,
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to fetch due executions: %w", err)
	}

	if len(jobs) == 0 {
		return nil
	}

	log.Printf("📋 Found %d due executions to process", len(jobs))

	for _, jobID := range jobs {
		// Try to claim the job atomically
		removed, err := r.redis.ZRem(ctx, delayedExecutionsKey, jobID).Result()
		if err != nil || removed == 0 {
			// Another worker claimed it or error occurred
			continue
		}

		// Execute the job
		go r.executeJob(context.Background(), jobID)
	}

	return nil
}

func (r *RedisDelayScheduler) executeJob(ctx context.Context, jobID string) {
	log.Printf("▶️  Executing delayed job: %s", jobID)

	// Retrieve continuation data
	key := fmt.Sprintf("%s%s", continuationPrefix, jobID)
	data, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		log.Printf("❌ Failed to retrieve continuation %s: %v", jobID, err)
		return
	}

	// Deserialize continuation
	var continuation engine.WorkflowContinuation
	if err := json.Unmarshal([]byte(data), &continuation); err != nil {
		log.Printf("❌ Failed to unmarshal continuation %s: %v", jobID, err)
		return
	}

	// Execute continuation handler
	if r.onContinuation != nil {
		if err := r.onContinuation(ctx, &continuation); err != nil {
			log.Printf("❌ Failed to execute continuation %s: %v", jobID, err)
			return
		}
	}

	// Clean up
	r.redis.Del(ctx, key)
	log.Printf("✅ Completed delayed job: %s", jobID)
}

// GetPendingCount returns the number of pending delayed executions
func (r *RedisDelayScheduler) GetPendingCount(ctx context.Context) (int64, error) {
	return r.redis.ZCard(ctx, delayedExecutionsKey).Result()
}

// GetContinuation retrieves a continuation by ID
func (r *RedisDelayScheduler) GetContinuation(ctx context.Context, id string) (*engine.WorkflowContinuation, error) {
	key := fmt.Sprintf("%s%s", continuationPrefix, id)
	data, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var continuation engine.WorkflowContinuation
	if err := json.Unmarshal([]byte(data), &continuation); err != nil {
		return nil, err
	}

	return &continuation, nil
}

// Cancel cancels a scheduled continuation
func (r *RedisDelayScheduler) Cancel(ctx context.Context, id string) error {
	// Remove from sorted set
	if err := r.redis.ZRem(ctx, delayedExecutionsKey, id).Err(); err != nil {
		return err
	}

	// Delete continuation data
	key := fmt.Sprintf("%s%s", continuationPrefix, id)
	return r.redis.Del(ctx, key).Err()
}


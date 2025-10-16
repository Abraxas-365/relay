package nodeexec

import (
	"context"
	"fmt"
	"time"

	"github.com/Abraxas-365/relay/engine"
)

type DelayExecutor struct {
	scheduler engine.DelayScheduler // ✅ Now using interface
}

func NewDelayExecutor(scheduler engine.DelayScheduler) *DelayExecutor {
	return &DelayExecutor{
		scheduler: scheduler,
	}
}

func (e *DelayExecutor) Execute(
	ctx context.Context,
	node engine.WorkflowNode,
	input map[string]any,
) (*engine.NodeResult, error) {
	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Success:   true,
		Output:    make(map[string]any),
		Timestamp: time.Now(),
	}

	duration, err := e.parseDuration(node.Config)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	// Validate duration
	if duration < 0 {
		err := fmt.Errorf("duration cannot be negative")
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	maxDelay := 24 * time.Hour
	if duration > maxDelay {
		err := fmt.Errorf("delay exceeds maximum allowed (%v)", maxDelay)
		result.Success = false
		result.Error = err.Error()
		return result, err
	}

	// Short delays: synchronous (blocking)
	if !e.scheduler.ShouldUseAsync(duration) {
		return e.executeSyncDelay(ctx, duration, result)
	}

	// Long delays: asynchronous (scheduled)
	return e.executeAsyncDelay(ctx, node, duration, input, result)
}

func (e *DelayExecutor) executeSyncDelay(
	ctx context.Context,
	duration time.Duration,
	result *engine.NodeResult,
) (*engine.NodeResult, error) {
	startTime := time.Now()
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		actualDuration := time.Since(startTime)
		result.Output["delayed_ms"] = actualDuration.Milliseconds()
		result.Output["requested_ms"] = duration.Milliseconds()
		result.Output["mode"] = "sync"
		result.Output["completed"] = true
		return result, nil
	case <-ctx.Done():
		result.Success = false
		result.Error = "delay cancelled"
		result.Output["completed"] = false
		result.Output["mode"] = "sync"
		return result, ctx.Err()
	}
}

func (e *DelayExecutor) executeAsyncDelay(
	ctx context.Context,
	node engine.WorkflowNode,
	duration time.Duration,
	input map[string]any,
	result *engine.NodeResult,
) (*engine.NodeResult, error) {
	// Extract workflow context from input
	continuation := &engine.WorkflowContinuation{
		WorkflowID:  extractString(input, "workflow_id"),
		NodeID:      node.ID,
		NextNodeID:  node.OnSuccess, // ✅ This is where to resume
		MessageID:   extractString(input, "message.id"),
		SessionID:   extractString(input, "session.id"),
		TenantID:    extractString(input, "tenant_id"),
		ChannelID:   extractString(input, "channel_id"),
		SenderID:    extractString(input, "sender_id"),
		NodeContext: input, // ✅ Save entire context to resume properly
	}

	// Schedule the continuation
	if err := e.scheduler.Schedule(ctx, continuation, duration); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to schedule delay: %v", err)
		return result, err
	}

	result.Output["delayed_ms"] = duration.Milliseconds()
	result.Output["mode"] = "async"
	result.Output["scheduled"] = true
	result.Output["continuation_id"] = continuation.ID
	result.Output["execute_at"] = continuation.ScheduledFor.Format(time.RFC3339)

	// Signal workflow to pause (not fail)
	result.Success = true // ✅ Mark as success!
	result.Output["__workflow_paused"] = true

	return result, nil
}
func (e *DelayExecutor) parseDuration(config map[string]any) (time.Duration, error) {
	// Try duration_ms
	if durationMs, ok := config["duration_ms"].(float64); ok {
		return time.Duration(durationMs) * time.Millisecond, nil
	}

	// Try duration string (e.g., "5s", "2m", "1h")
	if durationStr, ok := config["duration"].(string); ok {
		return time.ParseDuration(durationStr)
	}

	// Try duration_seconds
	if durationSec, ok := config["duration_seconds"].(float64); ok {
		return time.Duration(durationSec * float64(time.Second)), nil
	}

	return 0, fmt.Errorf("duration not found (try: duration_ms, duration, or duration_seconds)")
}

func (e *DelayExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeDelay
}

func (e *DelayExecutor) ValidateConfig(config map[string]any) error {
	_, err := e.parseDuration(config)
	return err
}

// Helper function
func extractString(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		// Handle nested maps (e.g., "message.id")
		if nested, ok := val.(map[string]any); ok {
			if id, ok := nested["id"].(string); ok {
				return id
			}
		}
	}
	return ""
}

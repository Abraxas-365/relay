package stepexec

import (
	"context"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
)

// DelayExecutor ejecuta delays
type DelayExecutor struct{}

var _ engine.StepExecutor = (*DelayExecutor)(nil)

func NewDelayExecutor() *DelayExecutor {
	return &DelayExecutor{}
}

func (de *DelayExecutor) Execute(ctx context.Context, step engine.WorkflowStep, input map[string]any) (*engine.StepResult, error) {
	startTime := time.Now()

	result := &engine.StepResult{
		StepID:    step.ID,
		StepName:  step.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	durationMs, ok := step.Config["duration_ms"].(float64)
	if !ok {
		if durationInt, ok := step.Config["duration_ms"].(int); ok {
			durationMs = float64(durationInt)
		} else {
			result.Success = false
			result.Error = "missing or invalid duration_ms"
			result.Duration = time.Since(startTime).Milliseconds()
			return result, errx.New("missing duration_ms", errx.TypeValidation)
		}
	}

	duration := time.Duration(durationMs) * time.Millisecond
	log.Printf("⏱️  Delaying for %v", duration)

	select {
	case <-time.After(duration):
		result.Success = true
		result.Output["delayed_ms"] = durationMs
		result.Duration = time.Since(startTime).Milliseconds()
		return result, nil
	case <-ctx.Done():
		result.Success = false
		result.Error = "delay cancelled"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, ctx.Err()
	}
}

func (de *DelayExecutor) SupportsType(stepType engine.StepType) bool {
	return stepType == engine.StepTypeDelay
}

func (de *DelayExecutor) ValidateConfig(config map[string]any) error {
	if _, ok := config["duration_ms"]; !ok {
		return errx.New("duration_ms is required for delay", errx.TypeValidation)
	}
	return nil
}

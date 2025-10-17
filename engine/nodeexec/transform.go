package nodeexec

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
)

type TransformExecutor struct {
	evaluator engine.ExpressionEvaluator
}

var _ engine.NodeExecutor = (*TransformExecutor)(nil)

func NewTransformExecutor(evaluator engine.ExpressionEvaluator) *TransformExecutor {
	return &TransformExecutor{evaluator: evaluator}
}

func (e *TransformExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()
	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract transform config
	transformConfig, err := engine.ExtractTransformConfig(node.Config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("invalid transform config: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	log.Printf("🔄 Transform: mapping %d fields", len(transformConfig.Mappings))

	// Transform each mapping
	transformed := make(map[string]any)
	errors := make([]string, 0)

	for targetKey, sourceExpr := range transformConfig.Mappings {
		log.Printf("   📍 Mapping '%s' from: %v", targetKey, sourceExpr)

		// Evaluate expression
		value, err := e.evaluator.Evaluate(ctx, sourceExpr, input)
		if err != nil {
			errMsg := fmt.Sprintf("failed to evaluate '%s': %v", targetKey, err)
			log.Printf("   ⚠️  %s", errMsg)
			errors = append(errors, errMsg)
			continue
		}

		transformed[targetKey] = value
		log.Printf("   ✅ '%s' = %v", targetKey, value)
	}

	// If all mappings failed, mark as failed
	if len(errors) > 0 && len(transformed) == 0 {
		result.Success = false
		result.Error = fmt.Sprintf("all transformations failed: %v", errors)
		result.Output["errors"] = errors
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New(result.Error, errx.TypeInternal)
	}

	result.Success = true
	result.Output = transformed

	if len(errors) > 0 {
		result.Output["errors"] = errors
		result.Output["partial_success"] = true
	}

	result.Duration = time.Since(startTime).Milliseconds()
	log.Printf("✅ Transform completed: %d fields mapped, %d errors", len(transformed), len(errors))

	return result, nil
}

func (e *TransformExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeTransform
}

func (e *TransformExecutor) ValidateConfig(config map[string]any) error {
	transformConfig, err := engine.ExtractTransformConfig(config)
	if err != nil {
		return err
	}
	return transformConfig.Validate()
}

package node

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
)

type LoopExecutor struct{}

var _ engine.NodeExecutor = (*LoopExecutor)(nil)

func NewLoopExecutor() *LoopExecutor {
	return &LoopExecutor{}
}

func (e *LoopExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()
	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract loop config
	loopConfig, err := engine.ExtractLoopConfig(node.Config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("invalid loop config: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	log.Printf("🔁 Loop: iterating over '%s'", loopConfig.IterateOver)

	// Get collection to iterate
	collectionValue := getNestedFieldValue(input, loopConfig.IterateOver)
	if collectionValue == nil {
		result.Success = false
		result.Error = fmt.Sprintf("field '%s' not found", loopConfig.IterateOver)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New(result.Error, errx.TypeValidation)
	}

	// Convert to slice
	var items []any
	switch v := collectionValue.(type) {
	case []any:
		items = v
	case []string:
		items = make([]any, len(v))
		for i, s := range v {
			items[i] = s
		}
	case []int:
		items = make([]any, len(v))
		for i, n := range v {
			items[i] = n
		}
	default:
		result.Success = false
		result.Error = fmt.Sprintf("iterate_over must be an array, got %T", collectionValue)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New(result.Error, errx.TypeValidation)
	}

	log.Printf("   📊 Found %d items to iterate", len(items))

	// Execute loop
	results := make([]map[string]any, 0, len(items))
	maxIterations := loopConfig.GetMaxIterations()

	for i, item := range items {
		if i >= maxIterations {
			log.Printf("   ⚠️  Max iterations reached: %d", maxIterations)
			break
		}

		log.Printf("   🔄 Iteration %d/%d", i+1, len(items))

		// Create iteration result
		iterResult := map[string]any{
			"index": i,
			"item":  item,
		}

		// TODO: In a real implementation, you would execute the body_node here
		// For now, we just collect the items
		// This would require recursive workflow execution

		results = append(results, iterResult)
	}

	result.Success = true
	result.Output["results"] = results
	result.Output["count"] = len(results)
	result.Output["total_items"] = len(items)

	// Set next node to body_node for first iteration
	// (This is a simplified implementation - real loops need more complex state management)
	if len(items) > 0 {
		result.Output["body_node"] = loopConfig.BodyNode
	}

	result.Duration = time.Since(startTime).Milliseconds()
	log.Printf("✅ Loop completed: %d iterations", len(results))

	return result, nil
}

func (e *LoopExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeLoop
}

func (e *LoopExecutor) ValidateConfig(config map[string]any) error {
	loopConfig, err := engine.ExtractLoopConfig(config)
	if err != nil {
		return err
	}
	return loopConfig.Validate()
}

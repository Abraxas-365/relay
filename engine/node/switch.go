package node

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/engine"
)

type SwitchExecutor struct{}

var _ engine.NodeExecutor = (*SwitchExecutor)(nil)

func NewSwitchExecutor() *SwitchExecutor {
	return &SwitchExecutor{}
}

func (e *SwitchExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()
	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract switch config
	switchConfig, err := engine.ExtractSwitchConfig(node.Config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("invalid switch config: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	log.Printf("🔀 Switch: evaluating field '%s'", switchConfig.Field)

	// Get field value from input using nested path
	fieldValue := getNestedFieldValue(input, switchConfig.Field)
	fieldValueStr := fmt.Sprint(fieldValue)

	log.Printf("   📊 Field value: %v (type: %T)", fieldValue, fieldValue)

	// Find matching case
	var matchedNodeID string
	var matchedCase string

	for caseValue, nodeID := range switchConfig.Cases {
		if caseValue == "default" {
			continue // Handle default separately
		}

		if fieldValueStr == caseValue {
			matchedNodeID = nodeID.(string)
			matchedCase = caseValue
			log.Printf("   ✅ Matched case: '%s' -> node '%s'", caseValue, matchedNodeID)
			break
		}
	}

	// Check for default case if no match
	if matchedNodeID == "" {
		if defaultNode, ok := switchConfig.Cases["default"]; ok {
			matchedNodeID = defaultNode.(string)
			matchedCase = "default"
			log.Printf("   📌 Using default case -> node '%s'", matchedNodeID)
		} else {
			log.Printf("   ⚠️  No matching case found and no default")
		}
	}

	result.Success = true
	result.Output["matched_case"] = matchedCase
	result.Output["field_value"] = fieldValue
	result.Output["field"] = switchConfig.Field

	// Set next node if matched
	if matchedNodeID != "" {
		result.Output["next_node"] = matchedNodeID
		// Store in context for workflow executor
		input["__next_node"] = matchedNodeID
	}

	result.Duration = time.Since(startTime).Milliseconds()
	return result, nil
}

func (e *SwitchExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeSwitch
}

func (e *SwitchExecutor) ValidateConfig(config map[string]any) error {
	switchConfig, err := engine.ExtractSwitchConfig(config)
	if err != nil {
		return err
	}
	return switchConfig.Validate()
}

// Helper to get nested field value (e.g., "trigger.message.text")
func getNestedFieldValue(data map[string]any, path string) any {
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			if val, ok := v[part]; ok {
				current = val
			} else {
				return nil
			}
		default:
			return nil
		}
	}

	return current
}

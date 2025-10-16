package nodeexec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
)

// ConditionExecutor ejecuta condiciones
type ConditionExecutor struct{}

var _ engine.NodeExecutor = (*ConditionExecutor)(nil)

func NewConditionExecutor() *ConditionExecutor {
	return &ConditionExecutor{}
}

func (ce *ConditionExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()

	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Obtener configuración
	conditionType, ok := node.Config["condition_type"].(string)
	if !ok {
		result.Success = false
		result.Error = "missing condition_type"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("missing condition_type", errx.TypeValidation)
	}

	var conditionMet bool
	var err error

	switch conditionType {
	case "contains":
		conditionMet, err = ce.evaluateContains(node.Config, input)
	case "equals":
		conditionMet, err = ce.evaluateEquals(node.Config, input)
	case "exists":
		conditionMet, err = ce.evaluateExists(node.Config, input)
	case "regex":
		conditionMet, err = ce.evaluateRegex(node.Config, input)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unknown condition type: %s", conditionType)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("unknown condition type", errx.TypeValidation)
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	result.Success = true
	result.Output["condition_met"] = conditionMet
	result.Duration = time.Since(startTime).Milliseconds()

	return result, nil
}

func (ce *ConditionExecutor) evaluateContains(config map[string]any, input map[string]any) (bool, error) {
	field, ok := config["field"].(string)
	if !ok {
		return false, errx.New("missing field", errx.TypeValidation)
	}

	value, ok := config["value"].(string)
	if !ok {
		return false, errx.New("missing value", errx.TypeValidation)
	}

	fieldValue, ok := input[field].(string)
	if !ok {
		return false, nil
	}

	caseInsensitive := config["case_insensitive"] == true
	if caseInsensitive {
		return strings.Contains(strings.ToLower(fieldValue), strings.ToLower(value)), nil
	}

	return strings.Contains(fieldValue, value), nil
}

func (ce *ConditionExecutor) evaluateEquals(config map[string]any, input map[string]any) (bool, error) {
	field, ok := config["field"].(string)
	if !ok {
		return false, errx.New("missing field", errx.TypeValidation)
	}

	expectedValue := config["value"]
	actualValue, exists := input[field]

	if !exists {
		return false, nil
	}

	return fmt.Sprint(actualValue) == fmt.Sprint(expectedValue), nil
}

func (ce *ConditionExecutor) evaluateExists(config map[string]any, input map[string]any) (bool, error) {
	field, ok := config["field"].(string)
	if !ok {
		return false, errx.New("missing field", errx.TypeValidation)
	}

	_, exists := input[field]
	return exists, nil
}

func (ce *ConditionExecutor) evaluateRegex(config map[string]any, input map[string]any) (bool, error) {
	// TODO: Implementar evaluación de regex
	return false, errx.New("regex evaluation not implemented", errx.TypeInternal)
}

func (ce *ConditionExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeCondition
}

func (ce *ConditionExecutor) ValidateConfig(config map[string]any) error {
	conditionType, ok := config["condition_type"].(string)
	if !ok {
		return errx.New("condition_type is required", errx.TypeValidation)
	}

	switch conditionType {
	case "contains", "equals", "exists":
		if _, ok := config["field"].(string); !ok {
			return errx.New("field is required", errx.TypeValidation)
		}
	case "regex":
		if _, ok := config["pattern"].(string); !ok {
			return errx.New("pattern is required for regex", errx.TypeValidation)
		}
	default:
		return errx.New("unknown condition type", errx.TypeValidation)
	}

	return nil
}

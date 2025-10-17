package nodeexec

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/Abraxas-365/relay/engine"
)

type ValidateExecutor struct{}

var _ engine.NodeExecutor = (*ValidateExecutor)(nil)

func NewValidateExecutor() *ValidateExecutor {
	return &ValidateExecutor{}
}

func (e *ValidateExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()
	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract validate config
	validateConfig, err := engine.ExtractValidateConfig(node.Config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("invalid validate config: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	log.Printf("✅ Validate: checking %d fields", len(validateConfig.Schema))

	validationErrors := make([]string, 0)
	validFields := make(map[string]bool)

	// Validate each field
	for field, rule := range validateConfig.Schema {
		ruleStr, _ := rule.(string)
		value := getNestedFieldValue(input, field)

		log.Printf("   🔍 Validating '%s' with rule '%s'", field, ruleStr)

		if err := e.validateField(field, value, ruleStr); err != nil {
			validationErrors = append(validationErrors, err.Error())
			validFields[field] = false
			log.Printf("   ❌ %s", err.Error())
		} else {
			validFields[field] = true
			log.Printf("   ✅ '%s' is valid", field)
		}
	}

	isValid := len(validationErrors) == 0

	// Set result based on fail_on_error setting
	result.Success = !validateConfig.ShouldFailOnError() || isValid
	result.Output["valid"] = isValid
	result.Output["errors"] = validationErrors
	result.Output["fields"] = validFields
	result.Output["error_count"] = len(validationErrors)

	if !isValid && validateConfig.ShouldFailOnError() {
		result.Error = fmt.Sprintf("validation failed: %v", validationErrors)
	}

	result.Duration = time.Since(startTime).Milliseconds()
	log.Printf("✅ Validation completed: %d/%d fields valid", len(validFields)-len(validationErrors), len(validFields))

	return result, nil
}

func (e *ValidateExecutor) validateField(field string, value any, rule string) error {
	// Parse rule (can be comma-separated: "required,email")
	rules := strings.Split(rule, ",")

	for _, r := range rules {
		r = strings.TrimSpace(r)

		switch r {
		case "required":
			if value == nil || value == "" {
				return fmt.Errorf("field '%s' is required", field)
			}

		case "email":
			str, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' must be a string for email validation", field)
			}
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(str) {
				return fmt.Errorf("field '%s' must be a valid email", field)
			}

		case "number":
			if !isNumeric(value) {
				return fmt.Errorf("field '%s' must be a number", field)
			}

		case "string":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("field '%s' must be a string", field)
			}

		case "url":
			str, ok := value.(string)
			if !ok {
				return fmt.Errorf("field '%s' must be a string for URL validation", field)
			}
			if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
				return fmt.Errorf("field '%s' must be a valid URL", field)
			}

		default:
			// Check for min/max rules
			if strings.HasPrefix(r, "min:") {
				// TODO: Implement min validation
			} else if strings.HasPrefix(r, "max:") {
				// TODO: Implement max validation
			} else {
				log.Printf("   ⚠️  Unknown validation rule: %s", r)
			}
		}
	}

	return nil
}

func isNumeric(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	default:
		return false
	}
}

func (e *ValidateExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeValidate
}

func (e *ValidateExecutor) ValidateConfig(config map[string]any) error {
	validateConfig, err := engine.ExtractValidateConfig(config)
	if err != nil {
		return err
	}
	return validateConfig.Validate()
}

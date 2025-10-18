package node

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"maps"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// FieldResolver resolves field values from different sources with priority:
// 1. Webhook data (trigger.body)
// 2. Node config
// 3. Previous node output
// 4. Default value
type FieldResolver struct {
	data      map[string]any
	config    map[string]any
	evaluator engine.ExpressionEvaluator
}

func NewFieldResolver(data map[string]any, config map[string]any, evaluator engine.ExpressionEvaluator) *FieldResolver {
	return &FieldResolver{
		data:      data,
		config:    config,
		evaluator: evaluator,
	}
}

// ============================================================================
// Field Resolution (Priority: webhook -> config -> previous -> default)
// ============================================================================

// GetString resolves a string field with fallback order
func (r *FieldResolver) GetString(fieldName string, defaultValue string) string {
	// 1. Try webhook data (trigger.body.fieldName)
	if val := r.getFromWebhook(fieldName); val != "" {
		return r.RenderTemplate(val)
	}

	// 2. Try config
	if val, ok := r.config[fieldName].(string); ok && val != "" {
		return r.RenderTemplate(val)
	}

	// 3. Try previous node output
	if val, ok := r.data[fieldName].(string); ok && val != "" {
		return r.RenderTemplate(val)
	}

	// 4. Use default
	return defaultValue
}

// GetInt resolves an integer field
func (r *FieldResolver) GetInt(fieldName string, defaultValue int) int {
	// 1. Try webhook
	if val := r.getFromWebhook(fieldName); val != "" {
		if num := toInt(val); num != 0 {
			return num
		}
	}

	// 2. Try config
	if val, ok := r.config[fieldName]; ok {
		if num := toInt(val); num != 0 {
			return num
		}
	}

	// 3. Try data
	if val, ok := r.data[fieldName]; ok {
		if num := toInt(val); num != 0 {
			return num
		}
	}

	// 4. Default
	return defaultValue
}

// GetFloat resolves a float field
func (r *FieldResolver) GetFloat(fieldName string, defaultValue float64) float64 {
	// 1. Try webhook
	if val := r.getFromWebhook(fieldName); val != "" {
		if num := toFloat64(val); num != 0 {
			return num
		}
	}

	// 2. Try config
	if val, ok := r.config[fieldName]; ok {
		if num := toFloat64(val); num != 0 {
			return num
		}
	}

	// 3. Try data
	if val, ok := r.data[fieldName]; ok {
		if num := toFloat64(val); num != 0 {
			return num
		}
	}

	// 4. Default
	return defaultValue
}

// GetBool resolves a boolean field
func (r *FieldResolver) GetBool(fieldName string, defaultValue bool) bool {
	// 1. Try webhook
	if val := r.getFromWebhook(fieldName); val != "" {
		if b := toBool(val); b {
			return true
		}
	}

	// 2. Try config
	if val, ok := r.config[fieldName].(bool); ok {
		return val
	}

	// 3. Try data
	if val, ok := r.data[fieldName].(bool); ok {
		return val
	}

	// 4. Default
	return defaultValue
}

// GetMap resolves a map field
func (r *FieldResolver) GetMap(fieldName string) map[string]any {
	// 1. Try webhook
	if val := r.getFromWebhookNested(fieldName); val != nil {
		if m, ok := val.(map[string]any); ok {
			return m
		}
	}

	// 2. Try config
	if val, ok := r.config[fieldName].(map[string]any); ok {
		return val
	}

	// 3. Try data
	if val, ok := r.data[fieldName].(map[string]any); ok {
		return val
	}

	// 4. Empty map
	return make(map[string]any)
}

// GetArray resolves an array field
func (r *FieldResolver) GetArray(fieldName string) []any {
	// 1. Try webhook
	if val := r.getFromWebhookNested(fieldName); val != nil {
		if arr, ok := val.([]any); ok {
			return arr
		}
	}

	// 2. Try config
	if val, ok := r.config[fieldName].([]any); ok {
		return val
	}

	// 3. Try data
	if val, ok := r.data[fieldName].([]any); ok {
		return val
	}

	// 4. Empty array
	return []any{}
}

// ============================================================================
// Source Extractors
// ============================================================================

// getFromWebhook gets string value from trigger.body using field name
func (r *FieldResolver) getFromWebhook(fieldName string) string {
	// Check trigger.body.fieldName
	if trigger, ok := r.data["trigger"].(map[string]any); ok {
		if body, ok := trigger["body"].(map[string]any); ok {
			if val, ok := body[fieldName]; ok {
				return toString(val)
			}
		}
	}

	// Also check trigger.query.fieldName
	if trigger, ok := r.data["trigger"].(map[string]any); ok {
		if query, ok := trigger["query"].(map[string]any); ok {
			if val, ok := query[fieldName]; ok {
				return toString(val)
			}
		}
	}

	return ""
}

// getFromWebhookNested gets any value (including nested objects/arrays)
func (r *FieldResolver) getFromWebhookNested(fieldName string) any {
	// Check trigger.body.fieldName
	if trigger, ok := r.data["trigger"].(map[string]any); ok {
		if body, ok := trigger["body"].(map[string]any); ok {
			if val, ok := body[fieldName]; ok {
				return val
			}
		}
	}

	return nil
}

// GetNestedValue gets nested value using path like "trigger.body.user.name"
func (r *FieldResolver) GetNestedValue(path string) any {
	parts := strings.Split(path, ".")
	current := any(r.data)

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

// ============================================================================
// Special Extractors
// ============================================================================

// GetTenantID extracts tenant ID from data
func (r *FieldResolver) GetTenantID() (kernel.TenantID, error) {
	// Try trigger.tenant_id
	if trigger, ok := r.data["trigger"].(map[string]any); ok {
		if tenantID, ok := trigger["tenant_id"].(string); ok && tenantID != "" {
			return kernel.TenantID(tenantID), nil
		}
	}

	// Try data.tenant_id
	if tenantID, ok := r.data["tenant_id"].(string); ok && tenantID != "" {
		return kernel.TenantID(tenantID), nil
	}

	// Try config.tenant_id
	if tenantID, ok := r.config["tenant_id"].(string); ok && tenantID != "" {
		return kernel.TenantID(tenantID), nil
	}

	return "", fmt.Errorf("tenant_id not found")
}

// GetChannelID extracts channel ID from data
func (r *FieldResolver) GetChannelID() (kernel.ChannelID, error) {
	channelIDStr := r.GetString("channel_id", "")
	if channelIDStr == "" {
		return "", fmt.Errorf("channel_id not found")
	}
	return kernel.ChannelID(channelIDStr), nil
}

// GetWorkflowID extracts workflow ID from data
func (r *FieldResolver) GetWorkflowID() (kernel.WorkflowID, error) {
	workflowIDStr := r.GetString("workflow_id", "")
	if workflowIDStr == "" {
		return "", fmt.Errorf("workflow_id not found")
	}
	return kernel.NewWorkflowID(workflowIDStr), nil
}

// ============================================================================
// Template Rendering
// ============================================================================

// RenderTemplate renders template strings like {{trigger.body.name}}
func (r *FieldResolver) RenderTemplate(template string) string {
	re := regexp.MustCompile(`\{\{(.+?)\}\}`)

	result := re.ReplaceAllStringFunc(template, func(match string) string {
		path := strings.TrimSpace(match[2 : len(match)-2])

		// Resolve the path
		value := r.GetNestedValue(path)
		if value != nil {
			return fmt.Sprintf("%v", value)
		}

		// Keep original if not found
		return match
	})

	return result
}

// RenderMap renders all string values in a map
func (r *FieldResolver) RenderMap(m map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range m {
		if str, ok := v.(string); ok {
			result[k] = r.RenderTemplate(str)
		} else {
			result[k] = v
		}
	}
	return result
}

// RenderArray renders all string values in an array
func (r *FieldResolver) RenderArray(arr []any) []any {
	result := make([]any, len(arr))
	for i, v := range arr {
		if str, ok := v.(string); ok {
			result[i] = r.RenderTemplate(str)
		} else {
			result[i] = v
		}
	}
	return result
}

// ============================================================================
// Expression Evaluation (CEL support)
// ============================================================================

// Evaluate evaluates an expression using the CEL evaluator
func (r *FieldResolver) Evaluate(ctx context.Context, expression string) (any, error) {
	if r.evaluator == nil {
		return nil, fmt.Errorf("expression evaluator not available")
	}

	return r.evaluator.Evaluate(ctx, expression, r.data)
}

// ============================================================================
// Type Conversion Helpers
// ============================================================================

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case int32:
		return int(val)
	case float64:
		return int(val)
	case float32:
		return int(val)
	case string:
		var i int
		fmt.Sscanf(val, "%d", &i)
		return i
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		lower := strings.ToLower(val)
		return lower == "true" || lower == "yes" || lower == "1"
	case int:
		return val != 0
	case float64:
		return val != 0
	default:
		return false
	}
}

// ============================================================================
// Validation & Checks
// ============================================================================

// HasField checks if a field exists in any source
func (r *FieldResolver) HasField(fieldName string) bool {
	// Check webhook
	if r.getFromWebhook(fieldName) != "" {
		return true
	}

	// Check config
	if _, ok := r.config[fieldName]; ok {
		return true
	}

	// Check data
	if _, ok := r.data[fieldName]; ok {
		return true
	}

	return false
}

// IsEmpty checks if a value is empty
func (r *FieldResolver) IsEmpty(v any) bool {
	if v == nil {
		return true
	}

	switch val := v.(type) {
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	default:
		return false
	}
}

// RequireField ensures a field exists or returns error
func (r *FieldResolver) RequireField(fieldName string) error {
	if !r.HasField(fieldName) {
		return fmt.Errorf("required field '%s' not found", fieldName)
	}
	return nil
}

// ============================================================================
// Debugging Helpers
// ============================================================================

// GetAllKeys returns all available keys from all sources
func (r *FieldResolver) GetAllKeys() []string {
	keys := make(map[string]bool)

	// From webhook
	if trigger, ok := r.data["trigger"].(map[string]any); ok {
		if body, ok := trigger["body"].(map[string]any); ok {
			for k := range body {
				keys[k] = true
			}
		}
	}

	// From config
	for k := range r.config {
		keys[k] = true
	}

	// From data
	for k := range r.data {
		keys[k] = true
	}

	// Convert to slice
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}

// Dump returns all data for debugging
func (r *FieldResolver) Dump() map[string]any {
	return map[string]any{
		"data":   r.data,
		"config": r.config,
		"keys":   r.GetAllKeys(),
	}
}

// ============================================================================
// Field Mappings Support
// ============================================================================

// GetWithMapping gets a field with custom mapping support
func (r *FieldResolver) GetWithMapping(fieldName string, defaultValue string) string {
	// Check if there's a field_mappings configuration
	if mappings, ok := r.config["field_mappings"].(map[string]any); ok {
		// Check if this field has a custom mapping
		if mappedField, ok := mappings[fieldName].(string); ok {
			// Use the mapped field name instead
			fieldName = mappedField
		}
	}

	// Now get the value normally
	return r.GetString(fieldName, defaultValue)
}

// ApplyMappings applies field mappings to the entire config
func (r *FieldResolver) ApplyMappings() map[string]any {
	result := make(map[string]any)

	// Copy config
	maps.Copy(result, r.config)

	// Apply mappings if they exist
	if mappings, ok := r.config["field_mappings"].(map[string]any); ok {
		for targetField, sourceField := range mappings {
			if sourceStr, ok := sourceField.(string); ok {
				// Get value from the source field
				value := r.GetString(sourceStr, "")
				if value != "" {
					result[targetField] = value
				}
			}
		}
	}

	return result
}

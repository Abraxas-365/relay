package engine

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types/ref"
)

// ExpressionEvaluator defines the interface for evaluating expressions within workflow data.
type ExpressionEvaluator interface {
	// Evaluate recursively traverses a data structure (like a step's config)
	// and replaces any expressions (e.g., {{step_1.output.userId}}) with their
	// evaluated values from the provided context.
	Evaluate(ctx context.Context, data any, context map[string]any) (any, error)
}

// celEvaluator is an implementation of ExpressionEvaluator using CEL-Go.
type celEvaluator struct {
	expressionRegex *regexp.Regexp
}

// NewCelEvaluator creates a new expression evaluator.
func NewCelEvaluator() ExpressionEvaluator {
	return &celEvaluator{
		// Regex to find expressions like {{ expression }}
		expressionRegex: regexp.MustCompile(`\{\{([^}]+)\}\}`),
	}
}

func (e *celEvaluator) Evaluate(ctx context.Context, data any, context map[string]any) (any, error) {
	return e.evaluateRecursive(reflect.ValueOf(data), context)
}

// evaluateRecursive is the core evaluation logic.
func (e *celEvaluator) evaluateRecursive(val reflect.Value, context map[string]any) (any, error) {
	// Handle pointers and interfaces
	if val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return nil, nil
		}
		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.String:
		// This is where we find and replace expressions
		return e.evaluateString(val.String(), context)

	case reflect.Map:
		newMap := make(map[string]any)
		for _, key := range val.MapKeys() {
			// Evaluate the value of each map entry
			evaluatedVal, err := e.evaluateRecursive(val.MapIndex(key), context)
			if err != nil {
				return nil, err
			}
			newMap[key.String()] = evaluatedVal
		}
		return newMap, nil

	case reflect.Slice:
		newSlice := make([]any, val.Len())
		for i := 0; i < val.Len(); i++ {
			// Evaluate each item in the slice
			evaluatedItem, err := e.evaluateRecursive(val.Index(i), context)
			if err != nil {
				return nil, err
			}
			newSlice[i] = evaluatedItem
		}
		return newSlice, nil

	default:
		// For other types (int, bool, etc.), return the original value
		return val.Interface(), nil
	}
}

// evaluateString finds and evaluates all expressions in a single string.
func (e *celEvaluator) evaluateString(s string, context map[string]any) (any, error) {
	matches := e.expressionRegex.FindStringSubmatch(s)

	// If the string is *only* an expression (e.g., "{{step_1.output}}"),
	// return the evaluated type directly (e.g., a map or a number).
	if len(matches) > 0 && s == matches[0] {
		expr := strings.TrimSpace(matches[1])

		// ✅ Try simple path lookup first before CEL
		if value, found := getNestedValue(context, expr); found {
			log.Printf("✅ Resolved '%s' via simple path lookup: %v", expr, value)
			return value, nil
		}

		return e.evaluateCEL(expr, context)
	}

	// Otherwise, replace all occurrences of expressions inside the string.
	var evalError error
	resultString := e.expressionRegex.ReplaceAllStringFunc(s, func(match string) string {
		expr := strings.TrimSpace(e.expressionRegex.FindStringSubmatch(match)[1])

		// Try simple path lookup first
		if value, found := getNestedValue(context, expr); found {
			return fmt.Sprintf("%v", value)
		}

		evaluatedVal, err := e.evaluateCEL(expr, context)
		if err != nil {
			evalError = err
			return match // Return original on error
		}
		return fmt.Sprintf("%v", evaluatedVal)
	})

	if evalError != nil {
		return nil, evalError
	}

	return resultString, nil
}

// evaluateCEL compiles and runs a single CEL expression.
func (e *celEvaluator) evaluateCEL(expression string, context map[string]any) (any, error) {
	log.Printf("🔍 Evaluating CEL expression: '%s'", expression)
	log.Printf("   Available context keys: %v", getContextKeys(context))

	// ✅ FIX: Declare all context variables to CEL
	var envOptions []cel.EnvOption

	// Declare each top-level context key as a CEL variable
	for key := range context {
		envOptions = append(envOptions, cel.Variable(key, cel.DynType))
	}

	env, err := cel.NewEnv(envOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	parsed, issues := env.Parse(expression)
	if issues != nil && issues.Err() != nil {
		log.Printf("❌ CEL parse error for '%s': %v", expression, issues.Err())
		return nil, fmt.Errorf("failed to parse expression '%s': %w", expression, issues.Err())
	}

	checked, issues := env.Check(parsed)
	if issues != nil && issues.Err() != nil {
		log.Printf("⚠️  CEL check warning for '%s': %v", expression, issues.Err())
		// Don't fail on check errors for dynamic data
	}

	prg, err := env.Program(checked)
	if err != nil {
		log.Printf("❌ CEL program error for '%s': %v", expression, err)
		return nil, fmt.Errorf("failed to create program for '%s': %w", expression, err)
	}

	out, _, err := prg.Eval(context)
	if err != nil {
		log.Printf("❌ CEL eval error for '%s': %v", expression, err)
		log.Printf("   Context: %+v", context)
		return nil, fmt.Errorf("failed to evaluate expression '%s': %w", expression, err)
	}

	// Convert CEL type to native Go type
	nativeValue, err := e.convertToNative(out)
	if err != nil {
		return nil, fmt.Errorf("failed to convert CEL result for '%s': %w", expression, err)
	}

	log.Printf("✅ CEL result for '%s': %v", expression, nativeValue)
	return nativeValue, nil
}

// convertToNative converts a CEL-Go `ref.Val` to a native Go type.
func (e *celEvaluator) convertToNative(val ref.Val) (any, error) {
	if val == nil || val.Value() == nil {
		return nil, nil
	}
	native, err := val.ConvertToNative(reflect.TypeOf(map[string]any{}))
	if err == nil {
		return native, nil // Successfully converted to map/slice/etc.
	}
	return val.Value(), nil // Fallback to the primitive value (int, string, bool)
}

// ✅ ADD: Simple nested value lookup (fallback if CEL fails)
func getNestedValue(data map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	current := any(data)

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			if val, ok := v[part]; ok {
				current = val
			} else {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	return current, true
}

// ✅ ADD: Helper to get context keys for logging
func getContextKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}


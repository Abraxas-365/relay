package stepexec

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
)

// ActionExecutor ejecuta acciones dentro de workflows
type ActionExecutor struct {
	// Puedes agregar dependencias aquí si necesitas
}

var _ engine.NodeExecutor = (*ActionExecutor)(nil)

// NewActionExecutor crea una nueva instancia del ejecutor de acciones
func NewActionExecutor() *ActionExecutor {
	return &ActionExecutor{}
}

// Execute ejecuta una acción según su tipo
func (ae *ActionExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()

	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
	}

	// Determinar tipo de acción desde config
	actionType, ok := node.Config["action_type"].(string)
	if !ok {
		result.Success = false
		result.Error = "missing action_type in config"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, engine.ErrInvalidWorkflowNode().WithDetail("reason", "missing action_type")
	}

	// Ejecutar según tipo
	var err error
	switch actionType {
	case "console_log":
		err = ae.executeConsoleLog(ctx, node, input, result)
	case "set_context":
		err = ae.executeSetContext(ctx, node, input, result)
	case "delay":
		err = ae.executeDelay(ctx, node, input, result)
	case "response":
		err = ae.executeResponse(ctx, node, input, result)
	default:
		result.Success = false
		result.Error = fmt.Sprintf("unknown action type: %s", actionType)
		err = engine.ErrInvalidWorkflowNode().WithDetail("action_type", actionType)
	}

	result.Duration = time.Since(startTime).Milliseconds()
	return result, err
}

// executeConsoleLog imprime mensaje en consola
func (ae *ActionExecutor) executeConsoleLog(ctx context.Context, node engine.WorkflowNode, input map[string]any, result *engine.NodeResult) error {
	message, ok := node.Config["message"].(string)
	if !ok {
		result.Success = false
		result.Error = "missing or invalid message"
		return errx.New("missing message in console_log action", errx.TypeValidation)
	}

	// Reemplazar variables del input en el mensaje
	formattedMessage := ae.interpolateVariables(message, input)

	// Imprimir en consola con formato
	log.Printf("🔹 [WORKFLOW ACTION] %s: %s", node.Name, formattedMessage)

	// También imprimir input si está configurado
	if printInput, ok := node.Config["print_input"].(bool); ok && printInput {
		log.Printf("   Input: %+v", input)
	}

	result.Success = true
	result.Output = map[string]any{
		"message":   formattedMessage,
		"logged_at": time.Now().Format(time.RFC3339),
	}
	return nil
}

// executeSetContext establece valores en el contexto
func (ae *ActionExecutor) executeSetContext(ctx context.Context, node engine.WorkflowNode, input map[string]any, result *engine.NodeResult) error {
	contextData, ok := node.Config["context"].(map[string]any)
	if !ok {
		result.Success = false
		result.Error = "missing or invalid context data"
		return errx.New("missing context in set_context action", errx.TypeValidation)
	}

	// Interpolar variables
	interpolatedContext := make(map[string]any)
	for key, value := range contextData {
		if strVal, ok := value.(string); ok {
			interpolatedContext[key] = ae.interpolateVariables(strVal, input)
		} else {
			interpolatedContext[key] = value
		}
	}

	log.Printf("🔹 [WORKFLOW ACTION] %s: Setting context keys: %v", node.Name, getKeys(interpolatedContext))

	result.Success = true
	result.Output = map[string]any{
		"context": interpolatedContext,
	}
	return nil
}

// executeDelay espera un tiempo determinado
func (ae *ActionExecutor) executeDelay(ctx context.Context, node engine.WorkflowNode, input map[string]any, result *engine.NodeResult) error {
	durationMs, ok := node.Config["duration_ms"].(float64)
	if !ok {
		// Intentar como int
		if durationInt, ok := node.Config["duration_ms"].(int); ok {
			durationMs = float64(durationInt)
		} else {
			result.Success = false
			result.Error = "missing or invalid duration_ms"
			return errx.New("missing duration_ms in delay action", errx.TypeValidation)
		}
	}

	duration := time.Duration(durationMs) * time.Millisecond
	log.Printf("🔹 [WORKFLOW ACTION] %s: Delaying for %v", node.Name, duration)

	select {
	case <-time.After(duration):
		log.Printf("   Delay completed")
	case <-ctx.Done():
		result.Success = false
		result.Error = "delay cancelled"
		return ctx.Err()
	}

	result.Success = true
	result.Output = map[string]any{
		"delayed_ms": durationMs,
	}
	return nil
}

// executeResponse genera una respuesta
func (ae *ActionExecutor) executeResponse(ctx context.Context, node engine.WorkflowNode, input map[string]any, result *engine.NodeResult) error {
	responseText, ok := node.Config["text"].(string)
	if !ok {
		result.Success = false
		result.Error = "missing or invalid response text"
		return errx.New("missing text in response action", errx.TypeValidation)
	}

	// Interpolar variables
	formattedResponse := ae.interpolateVariables(responseText, input)

	log.Printf("🔹 [WORKFLOW ACTION] %s: Response prepared: %s", node.Name, formattedResponse)

	result.Success = true
	result.Output = map[string]any{
		"response":       formattedResponse,
		"should_respond": true,
	}
	return nil
}

// interpolateVariables reemplaza variables tipo {{variable}} en el texto
func (ae *ActionExecutor) interpolateVariables(text string, variables map[string]any) string {
	result := text
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}
	return result
}

// getKeys obtiene las llaves de un map
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// SupportsType verifica si soporta un tipo de paso
func (ae *ActionExecutor) SupportsType(stepType engine.NodeType) bool {
	return stepType == engine.NodeTypeAction
}

// ValidateConfig valida la configuración de una acción
func (ae *ActionExecutor) ValidateConfig(config map[string]any) error {
	actionType, ok := config["action_type"].(string)
	if !ok {
		return errx.New("action_type is required", errx.TypeValidation)
	}

	switch actionType {
	case "console_log":
		if _, ok := config["message"].(string); !ok {
			return errx.New("message is required for console_log", errx.TypeValidation)
		}
	case "set_context":
		if _, ok := config["context"].(map[string]any); !ok {
			return errx.New("context is required for set_context", errx.TypeValidation)
		}
	case "delay":
		if _, ok := config["duration_ms"]; !ok {
			return errx.New("duration_ms is required for delay", errx.TypeValidation)
		}
	case "response":
		if _, ok := config["text"].(string); !ok {
			return errx.New("text is required for response", errx.TypeValidation)
		}
	default:
		return errx.New("unknown action type", errx.TypeValidation).
			WithDetail("action_type", actionType)
	}

	return nil
}

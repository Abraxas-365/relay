package nodeexec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
)

// ResponseExecutor ejecuta pasos de respuesta
type ResponseExecutor struct{}

var _ engine.NodeExecutor = (*ResponseExecutor)(nil)

func NewResponseExecutor() *ResponseExecutor {
	return &ResponseExecutor{}
}

func (re *ResponseExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()

	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	responseText, ok := node.Config["text"].(string)
	if !ok {
		result.Success = false
		result.Error = "missing response text"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("missing text in response", errx.TypeValidation)
	}

	// Interpolar variables
	formattedResponse := re.interpolateVariables(responseText, input)

	result.Success = true
	result.Output["response"] = formattedResponse
	result.Output["should_respond"] = true
	result.Duration = time.Since(startTime).Milliseconds()

	return result, nil
}

func (re *ResponseExecutor) interpolateVariables(text string, variables map[string]any) string {
	result := text
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprint(value))
	}
	return result
}

func (re *ResponseExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeResponse
}

func (re *ResponseExecutor) ValidateConfig(config map[string]any) error {
	if _, ok := config["text"].(string); !ok {
		return errx.New("text is required for response", errx.TypeValidation)
	}
	return nil
}

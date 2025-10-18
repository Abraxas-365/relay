package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Abraxas-365/relay/engine"
	"slices"
)

type HTTPExecutor struct {
	httpClient *http.Client
	evaluator  engine.ExpressionEvaluator
}

func NewHTTPExecutor(evaluator engine.ExpressionEvaluator) *HTTPExecutor {
	return &HTTPExecutor{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		evaluator:  evaluator,
	}
}

func (e *HTTPExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()
	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract HTTP config
	httpConfig, err := engine.ExtractHTTPConfig(node.Config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("invalid HTTP config: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// Create resolver for template rendering
	resolver := NewFieldResolver(input, node.Config, e.evaluator)

	// Render URL with templates
	url := resolver.RenderTemplate(httpConfig.URL)

	// Render headers
	headers := make(map[string]string)
	for k, v := range httpConfig.Headers {
		headers[k] = resolver.RenderTemplate(v)
	}

	// Render body
	body := resolver.RenderMap(httpConfig.Body)

	log.Printf("🌐 HTTP Request: %s %s", httpConfig.GetMethod(), url)

	// Build request
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to marshal body: %v", err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result, err
		}
		bodyReader = bytes.NewBuffer(bodyJSON)
	}

	req, err := http.NewRequestWithContext(ctx, httpConfig.GetMethod(), url, bodyReader)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := e.httpClient.Do(req)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("request failed: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}
	defer resp.Body.Close()

	// Read response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to read response: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// Check success
	successCodes := httpConfig.GetSuccessCodes()
	isSuccess := slices.Contains(successCodes, resp.StatusCode)

	result.Success = isSuccess
	result.Output["status_code"] = resp.StatusCode
	result.Output["body"] = string(bodyBytes)

	// Try parse JSON
	var jsonBody any
	if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
		result.Output["json"] = jsonBody
	}

	if !isSuccess {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	result.Duration = time.Since(startTime).Milliseconds()
	log.Printf("✅ HTTP Response: %d", resp.StatusCode)

	return result, nil
}

func (e *HTTPExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeHTTP
}

func (e *HTTPExecutor) ValidateConfig(config map[string]any) error {
	httpConfig, err := engine.ExtractHTTPConfig(config)
	if err != nil {
		return err
	}
	return httpConfig.Validate()
}

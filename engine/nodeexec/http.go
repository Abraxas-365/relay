package nodeexec

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
)

type HTTPExecutor struct {
	httpClient *http.Client
}

var _ engine.NodeExecutor = (*HTTPExecutor)(nil)

func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
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

	log.Printf("🌐 HTTP Request: %s %s", httpConfig.GetMethod(), httpConfig.URL)

	// Build request body
	var bodyReader io.Reader
	if httpConfig.Body != nil && len(httpConfig.Body) > 0 {
		bodyJSON, err := json.Marshal(httpConfig.Body)
		if err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to marshal body: %v", err)
			result.Duration = time.Since(startTime).Milliseconds()
			return result, err
		}
		bodyReader = bytes.NewBuffer(bodyJSON)
		log.Printf("   📤 Body: %s", string(bodyJSON))
	}

	// Create request with custom timeout
	reqCtx := ctx
	if httpConfig.Timeout != nil && *httpConfig.Timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, time.Duration(*httpConfig.Timeout)*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(reqCtx, httpConfig.GetMethod(), httpConfig.URL, bodyReader)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// Add headers
	for key, value := range httpConfig.Headers {
		req.Header.Set(key, value)
	}
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request with retry logic
	maxRetries := httpConfig.GetMaxRetries()
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("   🔄 Retry attempt %d/%d", attempt, maxRetries)
			time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
		}

		resp, lastErr = e.httpClient.Do(req)
		if lastErr == nil {
			break
		}

		if !httpConfig.RetryOnFailure {
			break
		}
	}

	if lastErr != nil {
		result.Success = false
		result.Error = fmt.Sprintf("request failed: %v", lastErr)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, lastErr
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

	// Check success codes
	successCodes := httpConfig.GetSuccessCodes()
	isSuccess := false
	for _, code := range successCodes {
		if resp.StatusCode == code {
			isSuccess = true
			break
		}
	}

	result.Success = isSuccess
	result.Output["status_code"] = resp.StatusCode
	result.Output["headers"] = resp.Header
	result.Output["body"] = string(bodyBytes)

	// Try to parse JSON response
	var jsonBody any
	if err := json.Unmarshal(bodyBytes, &jsonBody); err == nil {
		result.Output["json"] = jsonBody
		log.Printf("   ✅ Response: %d (JSON parsed)", resp.StatusCode)
	} else {
		log.Printf("   ✅ Response: %d (text)", resp.StatusCode)
	}

	if !isSuccess {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
		log.Printf("   ❌ Request failed: %s", result.Error)
	}

	result.Duration = time.Since(startTime).Milliseconds()
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

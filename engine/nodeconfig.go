package engine

import (
	"encoding/json"
	"fmt"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/ptrx"
)

// ============================================================================
// Node Config Interface
// ============================================================================

// NodeConfig interface that all node configs should implement
type NodeConfig interface {
	Validate() error
	GetType() NodeType
	GetTimeout() int
}

// ============================================================================
// AI Agent Config
// ============================================================================

type AIAgentConfig struct {
	Provider           string         `json:"provider"`
	Model              string         `json:"model"`
	SystemPrompt       string         `json:"system_prompt"`
	Prompt             string         `json:"prompt,omitempty"`
	Temperature        *float32       `json:"temperature,omitempty"`
	MaxTokens          *int           `json:"max_tokens,omitempty"`
	Timeout            *int           `json:"timeout,omitempty"`
	UseMemory          bool           `json:"use_memory,omitempty"`
	Tools              []string       `json:"tools,omitempty"`
	MaxAutoIterations  *int           `json:"max_auto_iterations,omitempty"`
	MaxTotalIterations *int           `json:"max_total_iterations,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

// Validate validates the AI agent configuration
func (c AIAgentConfig) Validate() error {
	if c.Provider == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "provider is required")
	}
	if c.Model == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "model is required")
	}
	if c.SystemPrompt == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "system_prompt is required")
	}

	// Validate temperature range
	if c.Temperature != nil && (*c.Temperature < 0 || *c.Temperature > 2) {
		return ErrInvalidWorkflowNode().WithDetail("reason", "temperature must be between 0 and 2")
	}

	// Validate max tokens
	if c.MaxTokens != nil && *c.MaxTokens <= 0 {
		return ErrInvalidWorkflowNode().WithDetail("reason", "max_tokens must be positive")
	}

	return nil
}

func (c AIAgentConfig) GetType() NodeType {
	return NodeTypeAIAgent
}

func (c AIAgentConfig) GetTimeout() int {
	if c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
	}
	return 60 // AI agents need more time
}

// GetLLMClient creates an LLM client based on provider
func (c AIAgentConfig) GetLLMClient() llm.Client {
	// TODO: Support multiple providers
	switch c.Provider {
	case "openai":
		provider := aiopenai.NewOpenAIProvider("") // API key from env
		return *llm.NewClient(provider)
	// case "anthropic":
	//     provider := anthropic.NewAnthropicProvider("")
	//     return *llm.NewClient(provider)
	default:
		// Default to OpenAI
		provider := aiopenai.NewOpenAIProvider("")
		return *llm.NewClient(provider)
	}
}

// GetLLMOptions returns LLM options for the client
func (c AIAgentConfig) GetLLMOptions() []llm.Option {
	return []llm.Option{
		llm.WithModel(c.Model),
		llm.WithTemperature(ptrx.Float32ValueOr(c.Temperature, 0.7)),
		llm.WithMaxTokens(ptrx.IntValueOr(c.MaxTokens, 1000)),
	}
}

// GetMaxAutoIterations returns max auto iterations with default
func (c AIAgentConfig) GetMaxAutoIterations() int {
	if c.MaxAutoIterations != nil && *c.MaxAutoIterations > 0 {
		return *c.MaxAutoIterations
	}
	return 3 // Default
}

// GetMaxTotalIterations returns max total iterations with default
func (c AIAgentConfig) GetMaxTotalIterations() int {
	if c.MaxTotalIterations != nil && *c.MaxTotalIterations > 0 {
		return *c.MaxTotalIterations
	}
	return 10 // Default
}

// ============================================================================
// HTTP Config
// ============================================================================

type HTTPConfig struct {
	Method         string            `json:"method"` // GET, POST, PUT, etc.
	URL            string            `json:"url"`    // Request URL
	Headers        map[string]string `json:"headers,omitempty"`
	Body           map[string]any    `json:"body,omitempty"`
	Timeout        *int              `json:"timeout,omitempty"`       // seconds
	SuccessCodes   []int             `json:"success_codes,omitempty"` // [200, 201, 204]
	RetryOnFailure bool              `json:"retry_on_failure,omitempty"`
	MaxRetries     *int              `json:"max_retries,omitempty"`
	Metadata       map[string]any    `json:"metadata,omitempty"`
}

func (c HTTPConfig) Validate() error {
	if c.URL == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "url is required")
	}

	// Validate HTTP method
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	method := c.Method
	if method == "" {
		method = "GET" // Default
	}

	isValid := false
	for _, vm := range validMethods {
		if method == vm {
			isValid = true
			break
		}
	}
	if !isValid {
		return ErrInvalidWorkflowNode().WithDetail("reason", "invalid HTTP method: "+method)
	}

	return nil
}

func (c HTTPConfig) GetType() NodeType {
	return NodeTypeHTTP
}

func (c HTTPConfig) GetTimeout() int {
	if c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
	}
	return 30
}

func (c HTTPConfig) GetMethod() string {
	if c.Method == "" {
		return "GET"
	}
	return c.Method
}

func (c HTTPConfig) GetSuccessCodes() []int {
	if len(c.SuccessCodes) == 0 {
		return []int{200, 201, 202, 204}
	}
	return c.SuccessCodes
}

func (c HTTPConfig) GetMaxRetries() int {
	if c.MaxRetries != nil && *c.MaxRetries > 0 {
		return *c.MaxRetries
	}
	return 0 // No retries by default
}

// ============================================================================
// Switch Config
// ============================================================================

type SwitchConfig struct {
	Field    string         `json:"field"` // Field to evaluate
	Cases    map[string]any `json:"cases"` // case_value -> node_id
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (c SwitchConfig) Validate() error {
	if c.Field == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "field is required")
	}
	if len(c.Cases) == 0 {
		return ErrInvalidWorkflowNode().WithDetail("reason", "cases cannot be empty")
	}

	// Validate that all cases map to strings (node IDs)
	for key, value := range c.Cases {
		if _, ok := value.(string); !ok {
			return ErrInvalidWorkflowNode().WithDetail("reason", fmt.Sprintf("case '%s' must map to a node ID (string)", key))
		}
	}

	return nil
}

func (c SwitchConfig) GetType() NodeType {
	return NodeTypeSwitch
}

func (c SwitchConfig) GetTimeout() int {
	return 5 // Fast operation
}

// ============================================================================
// Transform Config
// ============================================================================

type TransformConfig struct {
	Mappings map[string]any `json:"mappings"` // target_key -> source_expression
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (c TransformConfig) Validate() error {
	if len(c.Mappings) == 0 {
		return ErrInvalidWorkflowNode().WithDetail("reason", "mappings cannot be empty")
	}
	return nil
}

func (c TransformConfig) GetType() NodeType {
	return NodeTypeTransform
}

func (c TransformConfig) GetTimeout() int {
	return 5 // Fast operation
}

// ============================================================================
// Loop Config
// ============================================================================

type LoopConfig struct {
	IterateOver   string         `json:"iterate_over"`        // Collection to iterate
	ItemVar       string         `json:"item_var"`            // Variable name for item
	IndexVar      string         `json:"index_var,omitempty"` // Variable name for index
	BodyNode      string         `json:"body_node"`           // Node ID to execute for each item
	MaxIterations *int           `json:"max_iterations,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func (c LoopConfig) Validate() error {
	if c.IterateOver == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "iterate_over is required")
	}
	if c.ItemVar == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "item_var is required")
	}
	if c.BodyNode == "" {
		return ErrInvalidWorkflowNode().WithDetail("reason", "body_node is required")
	}

	// Validate max iterations
	if c.MaxIterations != nil && (*c.MaxIterations <= 0 || *c.MaxIterations > 10000) {
		return ErrInvalidWorkflowNode().WithDetail("reason", "max_iterations must be between 1 and 10000")
	}

	return nil
}

func (c LoopConfig) GetType() NodeType {
	return NodeTypeLoop
}

func (c LoopConfig) GetTimeout() int {
	return 300 // Loops can take time (5 minutes)
}

func (c LoopConfig) GetMaxIterations() int {
	if c.MaxIterations != nil && *c.MaxIterations > 0 {
		return *c.MaxIterations
	}
	return 1000 // Default
}

func (c LoopConfig) GetItemVar() string {
	if c.ItemVar == "" {
		return "item" // Default
	}
	return c.ItemVar
}

// ============================================================================
// Validate Config
// ============================================================================

type ValidateConfig struct {
	Schema      map[string]any `json:"schema"`                  // field -> validation_rule
	FailOnError bool           `json:"fail_on_error,omitempty"` // Stop workflow on validation failure
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (c ValidateConfig) Validate() error {
	if len(c.Schema) == 0 {
		return ErrInvalidWorkflowNode().WithDetail("reason", "schema cannot be empty")
	}

	// Validate that all schema values are strings (validation rules)
	for field, rule := range c.Schema {
		if _, ok := rule.(string); !ok {
			return ErrInvalidWorkflowNode().WithDetail("reason", fmt.Sprintf("validation rule for field '%s' must be a string", field))
		}
	}

	return nil
}

func (c ValidateConfig) GetType() NodeType {
	return NodeTypeValidate
}

func (c ValidateConfig) GetTimeout() int {
	return 5 // Fast operation
}

func (c ValidateConfig) ShouldFailOnError() bool {
	return c.FailOnError // Default is false (allow workflow to continue)
}

// ============================================================================
// Helper Functions for Config Extraction
// ============================================================================

// ExtractAIAgentConfig extracts and validates AI agent config from node config
func ExtractAIAgentConfig(config map[string]any) (*AIAgentConfig, error) {
	// Marshal and unmarshal to convert map to struct
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var aiConfig AIAgentConfig
	if err := json.Unmarshal(data, &aiConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AI agent config: %w", err)
	}

	if err := aiConfig.Validate(); err != nil {
		return nil, err
	}

	return &aiConfig, nil
}

// ExtractHTTPConfig extracts and validates HTTP config
func ExtractHTTPConfig(config map[string]any) (*HTTPConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var httpConfig HTTPConfig
	if err := json.Unmarshal(data, &httpConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal HTTP config: %w", err)
	}

	if err := httpConfig.Validate(); err != nil {
		return nil, err
	}

	return &httpConfig, nil
}

// ExtractSwitchConfig extracts and validates switch config
func ExtractSwitchConfig(config map[string]any) (*SwitchConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var switchConfig SwitchConfig
	if err := json.Unmarshal(data, &switchConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal switch config: %w", err)
	}

	if err := switchConfig.Validate(); err != nil {
		return nil, err
	}

	return &switchConfig, nil
}

// ExtractTransformConfig extracts and validates transform config
func ExtractTransformConfig(config map[string]any) (*TransformConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var transformConfig TransformConfig
	if err := json.Unmarshal(data, &transformConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transform config: %w", err)
	}

	if err := transformConfig.Validate(); err != nil {
		return nil, err
	}

	return &transformConfig, nil
}

// ExtractLoopConfig extracts and validates loop config
func ExtractLoopConfig(config map[string]any) (*LoopConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var loopConfig LoopConfig
	if err := json.Unmarshal(data, &loopConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal loop config: %w", err)
	}

	if err := loopConfig.Validate(); err != nil {
		return nil, err
	}

	return &loopConfig, nil
}

// ExtractValidateConfig extracts and validates validation config
func ExtractValidateConfig(config map[string]any) (*ValidateConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var validateConfig ValidateConfig
	if err := json.Unmarshal(data, &validateConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal validate config: %w", err)
	}

	if err := validateConfig.Validate(); err != nil {
		return nil, err
	}

	return &validateConfig, nil
}

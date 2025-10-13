package parser

import (
	"encoding/json"
	"time"

	"maps"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/ptrx"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Parser Entity (Single struct for DB)
// ============================================================================

// Parser representa un analizador de mensajes
type Parser struct {
	ID          kernel.ParserID `db:"id" json:"id"`
	TenantID    kernel.TenantID `db:"tenant_id" json:"tenant_id"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	Type        ParserType      `db:"type" json:"type"`
	Config      json.RawMessage `db:"config" json:"config"` // JSON que se deserializa según Type
	Priority    int             `db:"priority" json:"priority"`
	IsActive    bool            `db:"is_active" json:"is_active"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Parser Types & Enums
// ============================================================================

type ParserType string

const (
	ParserTypeRegex   ParserType = "REGEX"
	ParserTypeAI      ParserType = "AI"
	ParserTypeRule    ParserType = "RULE"
	ParserTypeKeyword ParserType = "KEYWORD"
	ParserTypeNLP     ParserType = "NLP"
)

// ============================================================================
// Config Interface
// ============================================================================

// ParserConfig interfaz que todos los configs deben implementar
type ParserConfig interface {
	Validate() error
	GetType() ParserType
	GetTimeout() int
}

// ============================================================================
// Regex Parser Config
// ============================================================================

type RegexParserConfig struct {
	Patterns       []RegexPattern `json:"patterns"`
	CacheResults   bool           `json:"cache_results,omitempty"`
	Timeout        *int           `json:"timeout,omitempty"`
	FallbackParser *string        `json:"fallback_parser,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

func (c RegexParserConfig) Validate() error {
	if len(c.Patterns) == 0 {
		return ErrInvalidParserConfig().WithDetail("reason", "at least one pattern is required")
	}
	for _, pattern := range c.Patterns {
		if pattern.Pattern == "" {
			return ErrInvalidParserConfig().WithDetail("reason", "pattern cannot be empty")
		}
	}
	return nil
}

func (c RegexParserConfig) GetType() ParserType {
	return ParserTypeRegex
}

func (c RegexParserConfig) GetTimeout() int {
	if c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
	}
	return 30
}

type RegexPattern struct {
	Name          string         `json:"name"`
	Pattern       string         `json:"pattern"`
	Description   string         `json:"description,omitempty"`
	Actions       []Action       `json:"actions"`
	Flags         string         `json:"flags,omitempty"`
	CaptureGroups map[string]int `json:"capture_groups,omitempty"`
}

// ============================================================================
// AI Parser Config
// ============================================================================

type AIParserConfig struct {
	Provider       string         `json:"provider"` // openai, anthropic, gemini
	Model          string         `json:"model"`
	Prompt         string         `json:"prompt"`
	Tools          []string       `json:"tools,omitempty"` // Tool IDs to use (for future implementation)
	Temperature    *float32       `json:"temperature,omitempty"`
	MaxTokens      *int           `json:"max_tokens,omitempty"`
	CacheResults   bool           `json:"cache_results,omitempty"`
	Timeout        *int           `json:"timeout,omitempty"`
	FallbackParser *string        `json:"fallback_parser,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`

	// Agent configuration
	UseAgent           bool   `json:"use_agent,omitempty"`            // Enable agent mode with memory
	MaxAutoIterations  *int   `json:"max_auto_iterations,omitempty"`  // Max iterations with "auto" tool choice
	MaxTotalIterations *int   `json:"max_total_iterations,omitempty"` // Hard limit to prevent infinite loops
	SystemPrompt       string `json:"system_prompt,omitempty"`        // System prompt for agent (overrides Prompt when in agent mode)
}

func (aipc AIParserConfig) GetLLMClient() llm.Client {
	provider := aiopenai.NewOpenAIProvider("")
	return *llm.NewClient(provider)
}

func (aipc AIParserConfig) GetLLMOptions() []llm.Option {
	llmOptions := []llm.Option{
		llm.WithModel(aipc.Model),
		llm.WithTemperature(ptrx.Float32ValueOr(aipc.Temperature, 0.7)),
		llm.WithMaxTokens(ptrx.IntValueOr(aipc.MaxTokens, 512)),
	}
	return llmOptions
}

func (aipc AIParserConfig) GetMaxAutoIterations() int {
	if aipc.MaxAutoIterations != nil && *aipc.MaxAutoIterations > 0 {
		return *aipc.MaxAutoIterations
	}
	return 3 // Default
}

func (aipc AIParserConfig) GetMaxTotalIterations() int {
	if aipc.MaxTotalIterations != nil && *aipc.MaxTotalIterations > 0 {
		return *aipc.MaxTotalIterations
	}
	return 10 // Default
}

func (aipc AIParserConfig) GetSystemPrompt() string {
	if aipc.UseAgent && aipc.SystemPrompt != "" {
		return aipc.SystemPrompt
	}
	return aipc.Prompt
}

func (c AIParserConfig) Validate() error {
	if c.Provider == "" {
		return ErrInvalidParserConfig().WithDetail("reason", "provider is required")
	}
	if c.Model == "" {
		return ErrInvalidParserConfig().WithDetail("reason", "model is required")
	}
	if c.Prompt == "" && !c.UseAgent {
		return ErrInvalidParserConfig().WithDetail("reason", "prompt is required")
	}
	if c.UseAgent && c.SystemPrompt == "" && c.Prompt == "" {
		return ErrInvalidParserConfig().WithDetail("reason", "system_prompt or prompt is required when use_agent is true")
	}
	return nil
}

func (c AIParserConfig) GetType() ParserType {
	return ParserTypeAI
}

func (c AIParserConfig) GetTimeout() int {
	if c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
	}
	return 60 // AI parsers need more time
}

// ============================================================================
// Rule Parser Config
// ============================================================================

type RuleParserConfig struct {
	Rules          []Rule         `json:"rules"`
	CacheResults   bool           `json:"cache_results,omitempty"`
	Timeout        *int           `json:"timeout,omitempty"`
	FallbackParser *string        `json:"fallback_parser,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

func (c RuleParserConfig) Validate() error {
	if len(c.Rules) == 0 {
		return ErrInvalidParserConfig().WithDetail("reason", "at least one rule is required")
	}
	for _, rule := range c.Rules {
		if !rule.IsValid() {
			return ErrInvalidParserConfig().WithDetail("reason", "invalid rule: "+rule.Name)
		}
	}
	return nil
}

func (c RuleParserConfig) GetType() ParserType {
	return ParserTypeRule
}

func (c RuleParserConfig) GetTimeout() int {
	if c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
	}
	return 30
}

type Rule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Conditions  []Condition `json:"conditions"`
	Operator    string      `json:"operator"` // AND, OR
	Actions     []Action    `json:"actions"`
	Priority    int         `json:"priority,omitempty"`
}

func (r *Rule) IsValid() bool {
	return r.Name != "" && len(r.Conditions) > 0 && len(r.Actions) > 0
}

func (r *Rule) IsAND() bool {
	return r.Operator == "AND" || r.Operator == ""
}

func (r *Rule) IsOR() bool {
	return r.Operator == "OR"
}

type Condition struct {
	Field         string `json:"field"`
	Operator      string `json:"operator"`
	Value         any    `json:"value"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
}

func (c *Condition) IsValid() bool {
	return c.Field != "" && c.Operator != "" && c.Value != nil
}

// ============================================================================
// Keyword Parser Config
// ============================================================================

type KeywordParserConfig struct {
	Keywords       []Keyword      `json:"keywords"`
	CacheResults   bool           `json:"cache_results,omitempty"`
	Timeout        *int           `json:"timeout,omitempty"`
	FallbackParser *string        `json:"fallback_parser,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

func (c KeywordParserConfig) Validate() error {
	if len(c.Keywords) == 0 {
		return ErrInvalidParserConfig().WithDetail("reason", "at least one keyword is required")
	}
	for _, kw := range c.Keywords {
		if kw.Word == "" {
			return ErrInvalidParserConfig().WithDetail("reason", "keyword word cannot be empty")
		}
	}
	return nil
}

func (c KeywordParserConfig) GetType() ParserType {
	return ParserTypeKeyword
}

func (c KeywordParserConfig) GetTimeout() int {
	if c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
	}
	return 10 // Keywords are fast
}

type Keyword struct {
	Word          string   `json:"word"`
	Aliases       []string `json:"aliases,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
	MatchWhole    bool     `json:"match_whole,omitempty"`
	Actions       []Action `json:"actions"`
	Weight        float64  `json:"weight,omitempty"`
}

// ============================================================================
// NLP Parser Config
// ============================================================================

type NLPParserConfig struct {
	NLPModel       string         `json:"nlp_model"`
	Intents        []Intent       `json:"intents"`
	Entities       []Entity       `json:"entities,omitempty"`
	MinConfidence  float64        `json:"min_confidence,omitempty"`
	CacheResults   bool           `json:"cache_results,omitempty"`
	Timeout        *int           `json:"timeout,omitempty"`
	FallbackParser *string        `json:"fallback_parser,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

func (c NLPParserConfig) Validate() error {
	if c.NLPModel == "" {
		return ErrInvalidParserConfig().WithDetail("reason", "nlp_model is required")
	}
	if len(c.Intents) == 0 {
		return ErrInvalidParserConfig().WithDetail("reason", "at least one intent is required")
	}
	return nil
}

func (c NLPParserConfig) GetType() ParserType {
	return ParserTypeNLP
}

func (c NLPParserConfig) GetTimeout() int {
	if c.Timeout != nil && *c.Timeout > 0 {
		return *c.Timeout
	}
	return 30
}

type Intent struct {
	Name             string   `json:"name"`
	Examples         []string `json:"examples"`
	Actions          []Action `json:"actions"`
	RequiredEntities []string `json:"required_entities,omitempty"`
}

type Entity struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Aliases []string `json:"aliases,omitempty"`
}

// ============================================================================
// Actions
// ============================================================================

type Action struct {
	Type   ActionType     `json:"type"`
	Config map[string]any `json:"config"`
}

type ActionType string

const (
	ActionTypeResponse        ActionType = "RESPONSE"
	ActionTypeTool            ActionType = "TOOL"
	ActionTypeRoute           ActionType = "ROUTE"
	ActionTypeSetContext      ActionType = "SET_CONTEXT"
	ActionTypeSetState        ActionType = "SET_STATE"
	ActionTypeTriggerWorkflow ActionType = "TRIGGER_WORKFLOW"
	ActionTypeWebhook         ActionType = "WEBHOOK"
	ActionTypeDelay           ActionType = "DELAY"
)

// ============================================================================
// Parse Result
// ============================================================================

type ParseResult struct {
	Success       bool             `json:"success"`
	ParserID      kernel.ParserID  `json:"parser_id"`
	ParserName    string           `json:"parser_name"`
	Response      string           `json:"response,omitempty"`
	ShouldRespond bool             `json:"should_respond"`
	Actions       []Action         `json:"actions,omitempty"`
	Context       map[string]any   `json:"context,omitempty"`
	ExtractedData map[string]any   `json:"extracted_data,omitempty"`
	Confidence    float64          `json:"confidence,omitempty"`
	NextParser    *kernel.ParserID `json:"next_parser,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
	Error         string           `json:"error,omitempty"`
	ProcessedAt   time.Time        `json:"processed_at"`
}

func (pr *ParseResult) IsSuccessful() bool {
	return pr.Success && pr.Error == ""
}

func (pr *ParseResult) HasActions() bool {
	return len(pr.Actions) > 0
}

func (pr *ParseResult) GetAction(actionType ActionType) *Action {
	for i := range pr.Actions {
		if pr.Actions[i].Type == actionType {
			return &pr.Actions[i]
		}
	}
	return nil
}

func (pr *ParseResult) GetActionsByType(actionType ActionType) []Action {
	var actions []Action
	for _, action := range pr.Actions {
		if action.Type == actionType {
			actions = append(actions, action)
		}
	}
	return actions
}

func (pr *ParseResult) HasExtractedData() bool {
	return len(pr.ExtractedData) > 0
}

func (pr *ParseResult) GetExtractedValue(key string) (any, bool) {
	if pr.ExtractedData == nil {
		return nil, false
	}
	val, ok := pr.ExtractedData[key]
	return val, ok
}

func (pr *ParseResult) SetExtractedValue(key string, value any) {
	if pr.ExtractedData == nil {
		pr.ExtractedData = make(map[string]any)
	}
	pr.ExtractedData[key] = value
}

func (pr *ParseResult) MergeContext(newContext map[string]any) {
	if pr.Context == nil {
		pr.Context = make(map[string]any)
	}
	maps.Copy(pr.Context, newContext)
}

func (pr *ParseResult) IsHighConfidence() bool {
	return pr.Confidence > 0.8
}

// ============================================================================
// Domain Methods - Parser
// ============================================================================

func (p *Parser) IsValid() bool {
	return p.Name != "" && p.Type != "" && !p.TenantID.IsEmpty()
}

func (p *Parser) Activate() {
	p.IsActive = true
	p.UpdatedAt = time.Now()
}

func (p *Parser) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now()
}

func (p *Parser) UpdateDetails(name, description string) {
	if name != "" {
		p.Name = name
	}
	if description != "" {
		p.Description = description
	}
	p.UpdatedAt = time.Now()
}

func (p *Parser) UpdatePriority(priority int) {
	p.Priority = priority
	p.UpdatedAt = time.Now()
}

// UpdateConfig actualiza la configuración
func (p *Parser) UpdateConfig(config ParserConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	p.Config = configJSON
	p.UpdatedAt = time.Now()
	return nil
}

// GetConfigStruct deserializa el config según el tipo
func (p *Parser) GetConfigStruct() (ParserConfig, error) {
	switch p.Type {
	case ParserTypeRegex:
		var config RegexParserConfig
		if err := json.Unmarshal(p.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ParserTypeAI:
		var config AIParserConfig
		if err := json.Unmarshal(p.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ParserTypeRule:
		var config RuleParserConfig
		if err := json.Unmarshal(p.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ParserTypeKeyword:
		var config KeywordParserConfig
		if err := json.Unmarshal(p.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ParserTypeNLP:
		var config NLPParserConfig
		if err := json.Unmarshal(p.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	default:
		return nil, ErrParserTypeNotSupported().WithDetail("type", string(p.Type))
	}
}

// GetTimeout obtiene el timeout configurado
func (p *Parser) GetTimeout() int {
	config, err := p.GetConfigStruct()
	if err != nil {
		return 30 // default
	}
	return config.GetTimeout()
}

// ============================================================================
// Helper Functions
// ============================================================================

// NewParserFromConfig crea un parser desde una config
func NewParserFromConfig(
	id kernel.ParserID,
	tenantID kernel.TenantID,
	name string,
	description string,
	config ParserConfig,
	priority int,
) (*Parser, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return &Parser{
		ID:          id,
		TenantID:    tenantID,
		Type:        config.GetType(),
		Name:        name,
		Description: description,
		Config:      configJSON,
		Priority:    priority,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func NewParseResult(parserID kernel.ParserID, parserName string) *ParseResult {
	return &ParseResult{
		ParserID:      parserID,
		ParserName:    parserName,
		Context:       make(map[string]any),
		ExtractedData: make(map[string]any),
		Metadata:      make(map[string]any),
		ProcessedAt:   time.Now(),
	}
}

func NewSuccessResult(parserID kernel.ParserID, parserName string) *ParseResult {
	result := NewParseResult(parserID, parserName)
	result.Success = true
	return result
}

func NewFailureResult(parserID kernel.ParserID, parserName string, err error) *ParseResult {
	result := NewParseResult(parserID, parserName)
	result.Success = false
	result.Error = err.Error()
	return result
}

// ============================================================================
// Selection Context
// ============================================================================

type SelectionContext struct {
	Message          engine.Message
	Session          *engine.Session
	AvailableParsers []*Parser
	PreviousResults  []*ParseResult
	Metadata         map[string]any
}

func NewSelectionContext(message engine.Message, session *engine.Session, parsers []*Parser) *SelectionContext {
	return &SelectionContext{
		Message:          message,
		Session:          session,
		AvailableParsers: parsers,
		PreviousResults:  make([]*ParseResult, 0),
		Metadata:         make(map[string]any),
	}
}

func (sc *SelectionContext) AddResult(result *ParseResult) {
	sc.PreviousResults = append(sc.PreviousResults, result)
}

func (sc *SelectionContext) GetLastResult() *ParseResult {
	if len(sc.PreviousResults) == 0 {
		return nil
	}
	return sc.PreviousResults[len(sc.PreviousResults)-1]
}

func (sc *SelectionContext) HasSuccessfulResult() bool {
	for _, result := range sc.PreviousResults {
		if result.IsSuccessful() {
			return true
		}
	}
	return false
}

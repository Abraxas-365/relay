package parser

import (
	"time"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Parser Entity
// ============================================================================

// Parser representa un analizador de mensajes
type Parser struct {
	ID          kernel.ParserID `db:"id" json:"id"`
	TenantID    kernel.TenantID `db:"tenant_id" json:"tenant_id"`
	Name        string          `db:"name" json:"name"`
	Description string          `db:"description" json:"description"`
	Type        ParserType      `db:"type" json:"type"`
	Config      ParserConfig    `db:"config" json:"config"`
	Priority    int             `db:"priority" json:"priority"` // Mayor número = mayor prioridad
	IsActive    bool            `db:"is_active" json:"is_active"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// ============================================================================
// Parser Types & Enums
// ============================================================================

// ParserType tipo de parser
type ParserType string

const (
	ParserTypeRegex   ParserType = "REGEX"
	ParserTypeAI      ParserType = "AI"
	ParserTypeRule    ParserType = "RULE"
	ParserTypeKeyword ParserType = "KEYWORD"
	ParserTypeNLP     ParserType = "NLP"
)

// ParserConfig configuración específica por tipo de parser
type ParserConfig struct {
	// Regex Parser
	Patterns []RegexPattern `json:"patterns,omitempty"`

	// AI Parser
	Provider    string   `json:"provider,omitempty"` // openai, anthropic, gemini
	Model       string   `json:"model,omitempty"`
	Prompt      string   `json:"prompt,omitempty"`
	Tools       []string `json:"tools,omitempty"` // IDs de tools disponibles
	Temperature *float64 `json:"temperature,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`

	// Rule Parser
	Rules []Rule `json:"rules,omitempty"`

	// Keyword Parser
	Keywords []Keyword `json:"keywords,omitempty"`

	// NLP Parser
	NLPModel      string   `json:"nlp_model,omitempty"`
	Intents       []Intent `json:"intents,omitempty"`
	Entities      []Entity `json:"entities,omitempty"`
	MinConfidence float64  `json:"min_confidence,omitempty"`

	// General
	Timeout        *int           `json:"timeout,omitempty"`         // seconds
	FallbackParser *string        `json:"fallback_parser,omitempty"` // Parser ID
	CacheResults   bool           `json:"cache_results,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// RegexPattern patrón regex con acciones
type RegexPattern struct {
	Name          string         `json:"name"`
	Pattern       string         `json:"pattern"`
	Description   string         `json:"description,omitempty"`
	Actions       []Action       `json:"actions"`
	Flags         string         `json:"flags,omitempty"`          // i, m, s, etc.
	CaptureGroups map[string]int `json:"capture_groups,omitempty"` // Nombre -> índice de grupo
}

// Rule regla lógica con condiciones y acciones
type Rule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Conditions  []Condition `json:"conditions"`
	Operator    string      `json:"operator"` // AND, OR
	Actions     []Action    `json:"actions"`
	Priority    int         `json:"priority,omitempty"`
}

// Condition condición para reglas
type Condition struct {
	Field         string `json:"field"`    // message.text, message.sender, context.key
	Operator      string `json:"operator"` // equals, contains, matches, gt, lt, in, etc.
	Value         any    `json:"value"`
	CaseSensitive bool   `json:"case_sensitive,omitempty"`
}

// Action acción a ejecutar cuando se cumple una condición
type Action struct {
	Type   ActionType     `json:"type"`
	Config map[string]any `json:"config"`
}

// ActionType tipo de acción
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

// Keyword palabra clave con acciones
type Keyword struct {
	Word          string   `json:"word"`
	Aliases       []string `json:"aliases,omitempty"`
	CaseSensitive bool     `json:"case_sensitive,omitempty"`
	MatchWhole    bool     `json:"match_whole,omitempty"` // Match palabra completa vs substring
	Actions       []Action `json:"actions"`
	Weight        float64  `json:"weight,omitempty"` // Para scoring múltiple
}

// Intent intención detectada por NLP
type Intent struct {
	Name             string   `json:"name"`
	Examples         []string `json:"examples"`
	Actions          []Action `json:"actions"`
	RequiredEntities []string `json:"required_entities,omitempty"`
}

// Entity entidad extraída por NLP
type Entity struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"` // person, location, date, custom
	Aliases []string `json:"aliases,omitempty"`
}

// ============================================================================
// Parse Result
// ============================================================================

// ParseResult resultado del parsing
type ParseResult struct {
	Success       bool             `json:"success"`
	ParserID      kernel.ParserID  `json:"parser_id"`
	ParserName    string           `json:"parser_name"`
	Response      string           `json:"response,omitempty"`
	ShouldRespond bool             `json:"should_respond"`
	Actions       []Action         `json:"actions,omitempty"`
	Context       map[string]any   `json:"context,omitempty"`
	ExtractedData map[string]any   `json:"extracted_data,omitempty"` // Datos extraídos (regex groups, entities, etc.)
	Confidence    float64          `json:"confidence,omitempty"`     // 0-1
	NextParser    *kernel.ParserID `json:"next_parser,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
	Error         string           `json:"error,omitempty"`
	ProcessedAt   time.Time        `json:"processed_at"`
}

// ============================================================================
// Domain Methods - Parser
// ============================================================================

// IsValid verifica si el parser es válido
func (p *Parser) IsValid() bool {
	return p.Name != "" && p.Type != "" && !p.TenantID.IsEmpty()
}

// Activate activa el parser
func (p *Parser) Activate() {
	p.IsActive = true
	p.UpdatedAt = time.Now()
}

// Deactivate desactiva el parser
func (p *Parser) Deactivate() {
	p.IsActive = false
	p.UpdatedAt = time.Now()
}

// UpdateConfig actualiza la configuración
func (p *Parser) UpdateConfig(config ParserConfig) {
	p.Config = config
	p.UpdatedAt = time.Now()
}

// UpdateDetails actualiza nombre y descripción
func (p *Parser) UpdateDetails(name, description string) {
	if name != "" {
		p.Name = name
	}
	if description != "" {
		p.Description = description
	}
	p.UpdatedAt = time.Now()
}

// UpdatePriority actualiza la prioridad
func (p *Parser) UpdatePriority(priority int) {
	p.Priority = priority
	p.UpdatedAt = time.Now()
}

// HasAIConfig verifica si tiene configuración AI
func (p *Parser) HasAIConfig() bool {
	return p.Type == ParserTypeAI && p.Config.Provider != ""
}

// HasRegexPatterns verifica si tiene patrones regex
func (p *Parser) HasRegexPatterns() bool {
	return p.Type == ParserTypeRegex && len(p.Config.Patterns) > 0
}

// HasRules verifica si tiene reglas
func (p *Parser) HasRules() bool {
	return p.Type == ParserTypeRule && len(p.Config.Rules) > 0
}

// GetTimeout obtiene el timeout configurado o default
func (p *Parser) GetTimeout() int {
	if p.Config.Timeout != nil && *p.Config.Timeout > 0 {
		return *p.Config.Timeout
	}
	return 30 // 30 segundos por defecto
}

// ============================================================================
// Domain Methods - ParseResult
// ============================================================================

// IsSuccessful verifica si el parsing fue exitoso
func (pr *ParseResult) IsSuccessful() bool {
	return pr.Success && pr.Error == ""
}

// HasActions verifica si hay acciones para ejecutar
func (pr *ParseResult) HasActions() bool {
	return len(pr.Actions) > 0
}

// GetAction obtiene una acción por tipo
func (pr *ParseResult) GetAction(actionType ActionType) *Action {
	for i := range pr.Actions {
		if pr.Actions[i].Type == actionType {
			return &pr.Actions[i]
		}
	}
	return nil
}

// GetActionsByType obtiene todas las acciones de un tipo
func (pr *ParseResult) GetActionsByType(actionType ActionType) []Action {
	var actions []Action
	for _, action := range pr.Actions {
		if action.Type == actionType {
			actions = append(actions, action)
		}
	}
	return actions
}

// HasExtractedData verifica si hay datos extraídos
func (pr *ParseResult) HasExtractedData() bool {
	return len(pr.ExtractedData) > 0
}

// GetExtractedValue obtiene un valor extraído
func (pr *ParseResult) GetExtractedValue(key string) (any, bool) {
	if pr.ExtractedData == nil {
		return nil, false
	}
	val, ok := pr.ExtractedData[key]
	return val, ok
}

// SetExtractedValue establece un valor extraído
func (pr *ParseResult) SetExtractedValue(key string, value any) {
	if pr.ExtractedData == nil {
		pr.ExtractedData = make(map[string]any)
	}
	pr.ExtractedData[key] = value
}

// MergeContext combina contexto existente con nuevo
func (pr *ParseResult) MergeContext(newContext map[string]any) {
	if pr.Context == nil {
		pr.Context = make(map[string]any)
	}
	for k, v := range newContext {
		pr.Context[k] = v
	}
}

// IsHighConfidence verifica si tiene alta confianza (> 0.8)
func (pr *ParseResult) IsHighConfidence() bool {
	return pr.Confidence > 0.8
}

// ============================================================================
// Domain Methods - Rule
// ============================================================================

// IsValid verifica si la regla es válida
func (r *Rule) IsValid() bool {
	return r.Name != "" && len(r.Conditions) > 0 && len(r.Actions) > 0
}

// HasOperator verifica el operador
func (r *Rule) IsAND() bool {
	return r.Operator == "AND" || r.Operator == ""
}

func (r *Rule) IsOR() bool {
	return r.Operator == "OR"
}

// ============================================================================
// Domain Methods - Condition
// ============================================================================

// IsValid verifica si la condición es válida
func (c *Condition) IsValid() bool {
	return c.Field != "" && c.Operator != "" && c.Value != nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// NewParseResult crea un nuevo resultado de parsing
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

// NewSuccessResult crea un resultado exitoso
func NewSuccessResult(parserID kernel.ParserID, parserName string) *ParseResult {
	result := NewParseResult(parserID, parserName)
	result.Success = true
	return result
}

// NewFailureResult crea un resultado fallido
func NewFailureResult(parserID kernel.ParserID, parserName string, err error) *ParseResult {
	result := NewParseResult(parserID, parserName)
	result.Success = false
	result.Error = err.Error()
	return result
}

// ============================================================================
// Parser Selection Context
// ============================================================================

// SelectionContext contexto para selección de parser
type SelectionContext struct {
	Message          engine.Message
	Session          *engine.Session
	AvailableParsers []*Parser
	PreviousResults  []*ParseResult
	Metadata         map[string]any
}

// NewSelectionContext crea un nuevo contexto de selección
func NewSelectionContext(message engine.Message, session *engine.Session, parsers []*Parser) *SelectionContext {
	return &SelectionContext{
		Message:          message,
		Session:          session,
		AvailableParsers: parsers,
		PreviousResults:  make([]*ParseResult, 0),
		Metadata:         make(map[string]any),
	}
}

// AddResult añade un resultado previo
func (sc *SelectionContext) AddResult(result *ParseResult) {
	sc.PreviousResults = append(sc.PreviousResults, result)
}

// GetLastResult obtiene el último resultado
func (sc *SelectionContext) GetLastResult() *ParseResult {
	if len(sc.PreviousResults) == 0 {
		return nil
	}
	return sc.PreviousResults[len(sc.PreviousResults)-1]
}

// HasSuccessfulResult verifica si hay algún resultado exitoso
func (sc *SelectionContext) HasSuccessfulResult() bool {
	for _, result := range sc.PreviousResults {
		if result.IsSuccessful() {
			return true
		}
	}
	return false
}

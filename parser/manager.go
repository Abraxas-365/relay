package parser

import (
	"context"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Parser Manager
// ============================================================================

// ParserManager orchestrates parser execution
type ParserManager struct {
	repo    ParserRepository
	engines map[ParserType]ParserEngine
}

func NewParserManager(repo ParserRepository) *ParserManager {
	return &ParserManager{
		repo:    repo,
		engines: make(map[ParserType]ParserEngine),
	}
}

// RegisterEngine registers a parser engine for a specific type
func (pm *ParserManager) RegisterEngine(parserType ParserType, engine ParserEngine) {
	pm.engines[parserType] = engine
}

// ExecuteParser executes a parser by ID
func (pm *ParserManager) ExecuteParser(
	ctx context.Context,
	parserID kernel.ParserID,
	tenantID kernel.TenantID,
	msg engine.Message,
	session *engine.Session,
) (*ParseResult, error) {
	// Get parser
	parser, err := pm.repo.FindByID(ctx, parserID, tenantID)
	if err != nil {
		return nil, ErrParserNotFound().WithDetail("parser_id", parserID.String())
	}

	// Check if active
	if !parser.IsActive {
		return nil, ErrParserInactive().WithDetail("parser_id", parserID.String())
	}

	// Get engine for this parser type
	engine, ok := pm.engines[parser.Type]
	if !ok {
		return nil, ErrParserEngineNotFound().WithDetail("type", string(parser.Type))
	}

	// Execute parser
	return engine.Parse(ctx, *parser, msg, session)
}

// ExecuteParserWithConfig executes a parser with custom config override
func (pm *ParserManager) ExecuteParserWithConfig(
	ctx context.Context,
	parserID kernel.ParserID,
	tenantID kernel.TenantID,
	msg engine.Message,
	session *engine.Session,
	configOverride map[string]any,
) (*ParseResult, error) {
	result, err := pm.ExecuteParser(ctx, parserID, tenantID, msg, session)
	if err != nil {
		return nil, err
	}

	// Apply config overrides (e.g., min_confidence)
	if minConf, ok := configOverride["min_confidence"].(float64); ok {
		if result.Confidence < minConf {
			result.Success = false
			result.Error = "confidence below threshold"
		}
	}

	return result, nil
}

// ValidateParser validates a parser configuration
func (pm *ParserManager) ValidateParser(parser *Parser) error {
	config, err := parser.GetConfigStruct()
	if err != nil {
		return err
	}

	engine, ok := pm.engines[parser.Type]
	if !ok {
		return ErrParserEngineNotFound().WithDetail("type", string(parser.Type))
	}

	return engine.ValidateConfig(config)
}

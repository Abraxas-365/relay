package parser

import (
	"context"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Repository Interface
// ============================================================================

// ParserRepository define el contrato para persistencia de parsers
type ParserRepository interface {
	// CRUD básico
	Save(ctx context.Context, parser Parser) error
	FindByID(ctx context.Context, id kernel.ParserID, tenantID kernel.TenantID) (*Parser, error)
	FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*Parser, error)
	Delete(ctx context.Context, id kernel.ParserID, tenantID kernel.TenantID) error
	ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error)

	// Búsquedas
	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*Parser, error)
	FindByType(ctx context.Context, parserType ParserType, tenantID kernel.TenantID) ([]*Parser, error)
	FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*Parser, error)
	FindByPriority(ctx context.Context, tenantID kernel.TenantID) ([]*Parser, error) // Ordenado por prioridad desc

	// List con paginación
	List(ctx context.Context, req ListParsersRequest) (ParserListResponse, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, ids []kernel.ParserID, tenantID kernel.TenantID, isActive bool) error
}

// ============================================================================
// Engine Interface
// ============================================================================

// ParserEngine ejecuta parsers según su tipo
type ParserEngine interface {
	// Parse procesa un mensaje con un parser específico
	Parse(ctx context.Context, parser Parser, message engine.Message, session *engine.Session) (*ParseResult, error)

	// Soporta el tipo de parser
	SupportsType(parserType ParserType) bool

	// Validar configuración del parser
	ValidateConfig(parserType ParserType, config ParserConfig) error
}

// ============================================================================
// Selector Interface
// ============================================================================

// ParserSelector selecciona el parser apropiado para un mensaje
type ParserSelector interface {
	// SelectParser selecciona el mejor parser para un mensaje
	SelectParser(ctx context.Context, selectionCtx *SelectionContext) (*Parser, error)

	// SelectParsers selecciona múltiples parsers (para cascada)
	SelectParsers(ctx context.Context, selectionCtx *SelectionContext, maxParsers int) ([]*Parser, error)

	// ShouldRetry determina si se debe intentar con otro parser
	ShouldRetry(ctx context.Context, result *ParseResult) bool
}

// ============================================================================
// Orchestrator Interface
// ============================================================================

// ParserOrchestrator orquesta la ejecución de múltiples parsers
type ParserOrchestrator interface {
	// Process procesa un mensaje con la cadena de parsers apropiada
	Process(ctx context.Context, message engine.Message, session *engine.Session) (*ParseResult, error)

	// ProcessWithParser procesa con un parser específico
	ProcessWithParser(ctx context.Context, parserID kernel.ParserID, message engine.Message, session *engine.Session) (*ParseResult, error)

	// ProcessCascade procesa en cascada hasta encontrar un resultado exitoso
	ProcessCascade(ctx context.Context, message engine.Message, session *engine.Session, maxAttempts int) (*ParseResult, error)
}

// ============================================================================
// Validator Interface
// ============================================================================

// ParserValidator valida parsers
type ParserValidator interface {
	// ValidateParser valida un parser completo
	ValidateParser(parser Parser) error

	// ValidateConfig valida configuración por tipo
	ValidateConfig(parserType ParserType, config ParserConfig) error

	// ValidateRegexPatterns valida patrones regex
	ValidateRegexPatterns(patterns []RegexPattern) error

	// ValidateRules valida reglas
	ValidateRules(rules []Rule) error

	// ValidateActions valida acciones
	ValidateActions(actions []Action) error
}

// ============================================================================
// Cache Interface
// ============================================================================

// ParserCache cachea resultados de parsing
type ParserCache interface {
	// Get obtiene resultado cacheado
	Get(ctx context.Context, cacheKey string) (*ParseResult, error)

	// Set guarda resultado en cache
	Set(ctx context.Context, cacheKey string, result *ParseResult, ttl int) error

	// Delete elimina del cache
	Delete(ctx context.Context, cacheKey string) error

	// Clear limpia cache de un tenant
	Clear(ctx context.Context, tenantID kernel.TenantID) error

	// GenerateKey genera una clave de cache
	GenerateKey(message engine.Message, parserID kernel.ParserID) string
}

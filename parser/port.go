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
	ValidateConfig(config ParserConfig) error
}

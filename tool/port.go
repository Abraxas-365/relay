package tool

import (
	"context"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Repository Interfaces
// ============================================================================

// ToolRepository define el contrato para persistencia de tools
type ToolRepository interface {
	// CRUD básico
	Save(ctx context.Context, tool Tool) error
	FindByID(ctx context.Context, id kernel.ToolID, tenantID kernel.TenantID) (*Tool, error)
	FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*Tool, error)
	Delete(ctx context.Context, id kernel.ToolID, tenantID kernel.TenantID) error
	ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error)

	// List con filtros y paginación
	List(ctx context.Context, req ListToolsRequest) (ToolListResponse, error)

	// Búsquedas específicas
	FindByType(ctx context.Context, toolType ToolType, tenantID kernel.TenantID) ([]*Tool, error)
	FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*Tool, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, ids []kernel.ToolID, tenantID kernel.TenantID, isActive bool) error
}

// ToolExecutionRepository define el contrato para persistencia de ejecuciones
type ToolExecutionRepository interface {
	// CRUD básico
	Save(ctx context.Context, execution ToolExecution) error
	FindByID(ctx context.Context, id string) (*ToolExecution, error)

	// List con filtros y paginación
	List(ctx context.Context, req ListExecutionsRequest) (ExecutionListResponse, error)

	// Estadísticas
	CountByTool(ctx context.Context, toolID kernel.ToolID) (int, error)
	CountByStatus(ctx context.Context, toolID kernel.ToolID, status ExecutionStatus) (int, error)
	GetAverageDuration(ctx context.Context, toolID kernel.ToolID) (float64, error)
}

// ============================================================================
// Executor Interfaces
// ============================================================================

// ToolExecutor ejecuta tools según su tipo
type ToolExecutor interface {
	// Execute ejecuta un tool con input dado
	Execute(ctx context.Context, tool *Tool, input map[string]any) (map[string]any, error)

	// ValidateInput valida input contra schema del tool
	ValidateInput(tool *Tool, input map[string]any) error

	// ValidateConfig valida configuración del tool
	ValidateConfig(toolType ToolType, config ToolConfig) error
}

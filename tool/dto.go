package tool

import (
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Request DTOs
// ============================================================================

// CreateToolRequest request para crear un tool
type CreateToolRequest struct {
	TenantID     kernel.TenantID `json:"tenant_id" validate:"required"`
	Name         string          `json:"name" validate:"required,min=2"`
	Description  string          `json:"description"`
	Type         ToolType        `json:"type" validate:"required"`
	Config       ToolConfig      `json:"config" validate:"required"`
	InputSchema  map[string]any  `json:"input_schema"`
	OutputSchema map[string]any  `json:"output_schema"`
}

// UpdateToolRequest request para actualizar un tool
type UpdateToolRequest struct {
	Name         *string         `json:"name,omitempty"`
	Description  *string         `json:"description,omitempty"`
	Config       *ToolConfig     `json:"config,omitempty"`
	InputSchema  *map[string]any `json:"input_schema,omitempty"`
	OutputSchema *map[string]any `json:"output_schema,omitempty"`
	IsActive     *bool           `json:"is_active,omitempty"`
}

// ExecuteToolRequest request para ejecutar un tool
type ExecuteToolRequest struct {
	ToolID kernel.ToolID  `json:"tool_id" validate:"required"`
	Input  map[string]any `json:"input" validate:"required"`
}

// ============================================================================
// List Request DTOs (con embedding de storex)
// ============================================================================

// ListToolsRequest request para listar tools con paginación y filtros
type ListToolsRequest struct {
	storex.PaginationOptions

	// Filtros tipados propios
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Type     *ToolType       `json:"type,omitempty"`
	IsActive *bool           `json:"is_active,omitempty"`
	Search   string          `json:"search,omitempty"`
}

// ListExecutionsRequest request para listar ejecuciones con filtros
type ListExecutionsRequest struct {
	storex.PaginationOptions

	// Filtros tipados propios
	TenantID kernel.TenantID  `json:"tenant_id" validate:"required"`
	ToolID   *kernel.ToolID   `json:"tool_id,omitempty"`
	Status   *ExecutionStatus `json:"status,omitempty"`
	From     *string          `json:"from,omitempty"` // ISO 8601 date
	To       *string          `json:"to,omitempty"`   // ISO 8601 date
}

// ============================================================================
// Response DTOs
// ============================================================================

// ToolResponse respuesta con tool y sus ejecuciones recientes
type ToolResponse struct {
	Tool             Tool            `json:"tool"`
	RecentExecutions []ToolExecution `json:"recent_executions,omitempty"`
}

// ExecutionResponse respuesta de una ejecución
type ExecutionResponse struct {
	Execution ToolExecution `json:"execution"`
}

// ToolListResponse lista paginada de tools (usa storex.Paginated)
type ToolListResponse = storex.Paginated[Tool]

// ExecutionListResponse lista paginada de executions (usa storex.Paginated)
type ExecutionListResponse = storex.Paginated[ToolExecution]

// ============================================================================
// Stats DTOs
// ============================================================================

// ToolStatsResponse estadísticas de un tool
type ToolStatsResponse struct {
	ToolID          kernel.ToolID `json:"tool_id"`
	ToolName        string        `json:"tool_name"`
	TotalExecutions int           `json:"total_executions"`
	SuccessCount    int           `json:"success_count"`
	FailureCount    int           `json:"failure_count"`
	AvgDuration     float64       `json:"avg_duration_ms"`
	LastExecutedAt  *string       `json:"last_executed_at,omitempty"`
}

// ToolUsageResponse uso de tools en un periodo
type ToolUsageResponse struct {
	TenantID        kernel.TenantID      `json:"tenant_id"`
	Period          string               `json:"period"` // day, week, month
	TotalExecutions int                  `json:"total_executions"`
	SuccessRate     float64              `json:"success_rate"`
	ToolBreakdown   []ToolUsageBreakdown `json:"tool_breakdown"`
}

type ToolUsageBreakdown struct {
	ToolID      kernel.ToolID `json:"tool_id"`
	ToolName    string        `json:"tool_name"`
	Executions  int           `json:"executions"`
	SuccessRate float64       `json:"success_rate"`
}

// ============================================================================
// Bulk Operation DTOs
// ============================================================================

// BulkToolOperationRequest request para operaciones masivas
type BulkToolOperationRequest struct {
	TenantID  kernel.TenantID `json:"tenant_id" validate:"required"`
	ToolIDs   []kernel.ToolID `json:"tool_ids" validate:"required,min=1"`
	Operation string          `json:"operation" validate:"required,oneof=activate deactivate delete"`
}

// BulkToolOperationResponse respuesta de operación masiva
type BulkToolOperationResponse struct {
	Successful []kernel.ToolID          `json:"successful"`
	Failed     map[kernel.ToolID]string `json:"failed"`
	Total      int                      `json:"total"`
}

// ============================================================================
// Simple DTOs
// ============================================================================

// ToolDetailsDTO DTO simplificado de tool
type ToolDetailsDTO struct {
	ID          kernel.ToolID `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Type        ToolType      `json:"type"`
	IsActive    bool          `json:"is_active"`
}

// ToDTO convierte Tool a ToolDetailsDTO
func (t *Tool) ToDTO() ToolDetailsDTO {
	return ToolDetailsDTO{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		Type:        t.Type,
		IsActive:    t.IsActive,
	}
}

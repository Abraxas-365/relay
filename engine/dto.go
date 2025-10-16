package engine

import (
	"time"

	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Message DTOs
// ============================================================================

// CreateMessageRequest request para crear un mensaje
type CreateMessageRequest struct {
	TenantID  kernel.TenantID  `json:"tenant_id" validate:"required"`
	ChannelID kernel.ChannelID `json:"channel_id" validate:"required"`
	SenderID  string           `json:"sender_id" validate:"required"`
	Content   MessageContent   `json:"content" validate:"required"`
	Context   map[string]any   `json:"context,omitempty"`
}

// MessageResponse respuesta con mensaje
type MessageResponse struct {
	Message Message `json:"message"`
}

// MessageListRequest request para listar mensajes
type MessageListRequest struct {
	storex.PaginationOptions

	TenantID  kernel.TenantID   `json:"tenant_id" validate:"required"`
	ChannelID *kernel.ChannelID `json:"channel_id,omitempty"`
	SenderID  *string           `json:"sender_id,omitempty"`
	Status    *MessageStatus    `json:"status,omitempty"`
	From      *string           `json:"from,omitempty"` // ISO 8601 date
	To        *string           `json:"to,omitempty"`   // ISO 8601 date
}

func (mlr MessageListRequest) GetOffset() int {
	page := mlr.Page
	size := mlr.PageSize
	return (page - 1) * size
}

// MessageListResponse lista paginada de mensajes
type MessageListResponse = storex.Paginated[Message]

// ============================================================================
// Workflow DTOs
// ============================================================================

// CreateWorkflowRequest request para crear un workflow
type CreateWorkflowRequest struct {
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Name     string          `json:"name" validate:"required,min=2"`
	Trigger  WorkflowTrigger `json:"trigger" validate:"required"`
	Nodes    []WorkflowNode  `json:"nodes" validate:"required,min=1"`
}

// UpdateWorkflowRequest request para actualizar un workflow
type UpdateWorkflowRequest struct {
	Name     *string          `json:"name,omitempty"`
	Trigger  *WorkflowTrigger `json:"trigger,omitempty"`
	Nodes    *[]WorkflowNode  `json:"nodes,omitempty"`
	IsActive *bool            `json:"is_active,omitempty"`
}

// ExecuteWorkflowRequest request para ejecutar un workflow manualmente
type ExecuteWorkflowRequest struct {
	WorkflowID kernel.WorkflowID `json:"workflow_id" validate:"required"`
	MessageID  kernel.MessageID  `json:"message_id" validate:"required"`
	Context    map[string]any    `json:"context,omitempty"`
}

// WorkflowResponse respuesta con workflow
type WorkflowResponse struct {
	Workflow Workflow `json:"workflow"`
}

// WorkflowListRequest request para listar workflows
type WorkflowListRequest struct {
	storex.PaginationOptions

	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	IsActive *bool           `json:"is_active,omitempty"`
	Search   string          `json:"search,omitempty"`
}

func (wlr WorkflowListRequest) GetOffset() int {
	page := wlr.Page
	size := wlr.PageSize
	return (page - 1) * size
}

// WorkflowListResponse lista paginada de workflows
type WorkflowListResponse = storex.Paginated[Workflow]

// WorkflowExecutionResponse respuesta de ejecución de workflow
type WorkflowExecutionResponse struct {
	WorkflowID    kernel.WorkflowID `json:"workflow_id"`
	MessageID     kernel.MessageID  `json:"message_id"`
	Success       bool              `json:"success"`
	Response      string            `json:"response,omitempty"`
	ShouldRespond bool              `json:"should_respond"`
	NextState     string            `json:"next_state,omitempty"`
	Context       map[string]any    `json:"context,omitempty"`
	Error         string            `json:"error,omitempty"`
}

// ============================================================================
// Session DTOs
// ============================================================================

// SessionResponse respuesta con sesión
type SessionResponse struct {
	Session Session `json:"session"`
}

// SessionListRequest request para listar sesiones
type SessionListRequest struct {
	storex.PaginationOptions

	IsActive     *bool             `json:"is_active,omitempty"`
	CurrentState *string           `json:"current_state,omitempty"`
	TenantID     kernel.TenantID   `json:"tenant_id" validate:"required"`
	ChannelID    *kernel.ChannelID `json:"channel_id,omitempty"`
	From         *time.Time        `json:"from,omitempty"`
	To           *time.Time        `json:"to,omitempty"`
	SenderID     *string           `json:"sender_id,omitempty"`
}

// SessionListResponse lista paginada de sesiones
type SessionListResponse = storex.Paginated[Session]

// UpdateSessionRequest request para actualizar sesión
type UpdateSessionRequest struct {
	Context      *map[string]any `json:"context,omitempty"`
	CurrentState *string         `json:"current_state,omitempty"`
}

// ============================================================================
// Stats DTOs
// ============================================================================

// MessageStatsResponse estadísticas de mensajes
type MessageStatsResponse struct {
	TenantID         kernel.TenantID         `json:"tenant_id"`
	Period           string                  `json:"period"` // day, week, month
	TotalMessages    int                     `json:"total_messages"`
	ProcessedCount   int                     `json:"processed_count"`
	FailedCount      int                     `json:"failed_count"`
	AvgProcessTime   float64                 `json:"avg_process_time_ms"`
	ChannelBreakdown []ChannelStatsBreakdown `json:"channel_breakdown"`
}

type ChannelStatsBreakdown struct {
	ChannelID      kernel.ChannelID `json:"channel_id"`
	ChannelName    string           `json:"channel_name"`
	MessageCount   int              `json:"message_count"`
	ProcessingRate float64          `json:"processing_rate"`
}

// WorkflowStatsResponse estadísticas de workflows
type WorkflowStatsResponse struct {
	WorkflowID       kernel.WorkflowID `json:"workflow_id"`
	WorkflowName     string            `json:"workflow_name"`
	TotalExecutions  int               `json:"total_executions"`
	SuccessCount     int               `json:"success_count"`
	FailureCount     int               `json:"failure_count"`
	AvgExecutionTime float64           `json:"avg_execution_time_ms"`
	LastExecutedAt   *string           `json:"last_executed_at,omitempty"`
}

// SessionStatsResponse estadísticas de sesiones
type SessionStatsResponse struct {
	TenantID       kernel.TenantID `json:"tenant_id"`
	ActiveSessions int             `json:"active_sessions"`
	TotalSessions  int             `json:"total_sessions"`
	AvgDuration    float64         `json:"avg_duration_minutes"`
}

// ============================================================================
// Bulk Operation DTOs
// ============================================================================

// BulkWorkflowOperationRequest request para operaciones masivas
type BulkWorkflowOperationRequest struct {
	TenantID    kernel.TenantID     `json:"tenant_id" validate:"required"`
	WorkflowIDs []kernel.WorkflowID `json:"workflow_ids" validate:"required,min=1"`
	Operation   string              `json:"operation" validate:"required,oneof=activate deactivate delete"`
}

// BulkWorkflowOperationResponse respuesta de operación masiva
type BulkWorkflowOperationResponse struct {
	Successful []kernel.WorkflowID          `json:"successful"`
	Failed     map[kernel.WorkflowID]string `json:"failed"`
	Total      int                          `json:"total"`
}

// ============================================================================
// Validation DTOs
// ============================================================================

// ValidateWorkflowRequest request para validar un workflow
type ValidateWorkflowRequest struct {
	Trigger WorkflowTrigger `json:"trigger" validate:"required"`
	Nodes   []WorkflowNode  `json:"nodes" validate:"required,min=1"`
}

// ValidateWorkflowResponse respuesta de validación
type ValidateWorkflowResponse struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ============================================================================
// Simple DTOs
// ============================================================================

// MessageDetailsDTO DTO simplificado de mensaje
type MessageDetailsDTO struct {
	ID        kernel.MessageID `json:"id"`
	ChannelID kernel.ChannelID `json:"channel_id"`
	SenderID  string           `json:"sender_id"`
	Content   MessageContent   `json:"content"`
	Status    MessageStatus    `json:"status"`
	CreatedAt string           `json:"created_at"`
}

// ToDTO convierte Message a MessageDetailsDTO
func (m *Message) ToDTO() MessageDetailsDTO {
	return MessageDetailsDTO{
		ID:        m.ID,
		ChannelID: m.ChannelID,
		SenderID:  m.SenderID,
		Content:   m.Content,
		Status:    m.Status,
		CreatedAt: m.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// WorkflowDetailsDTO DTO simplificado de workflow
type WorkflowDetailsDTO struct {
	ID        kernel.WorkflowID `json:"id"`
	Name      string            `json:"name"`
	IsActive  bool              `json:"is_active"`
	NodeCount int               `json:"node_count"`
}

// ToDTO convierte Workflow a WorkflowDetailsDTO
func (w *Workflow) ToDTO() WorkflowDetailsDTO {
	return WorkflowDetailsDTO{
		ID:        w.ID,
		Name:      w.Name,
		IsActive:  w.IsActive,
		NodeCount: len(w.Node),
	}
}

// GetOffset returns the offset for pagination
func (r SessionListRequest) GetOffset() int {
	return (r.Page - 1) * r.PageSize
}

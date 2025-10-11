package tool

import (
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Tool Entity
// ============================================================================

// Tool representa una herramienta/función ejecutable
type Tool struct {
	ID           kernel.ToolID   `db:"id" json:"id"`
	TenantID     kernel.TenantID `db:"tenant_id" json:"tenant_id"`
	Name         string          `db:"name" json:"name"`
	Description  string          `db:"description" json:"description"`
	Type         ToolType        `db:"type" json:"type"`
	Config       ToolConfig      `db:"config" json:"config"`
	InputSchema  map[string]any  `db:"input_schema" json:"input_schema"`
	OutputSchema map[string]any  `db:"output_schema" json:"output_schema"`
	IsActive     bool            `db:"is_active" json:"is_active"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at" json:"updated_at"`
}

// ToolType define los tipos de tools disponibles
type ToolType string

const (
	ToolTypeHTTP     ToolType = "HTTP"
	ToolTypeDatabase ToolType = "DATABASE"
	ToolTypeEmail    ToolType = "EMAIL"
	ToolTypeCustom   ToolType = "CUSTOM"
)

// ToolConfig configuración específica por tipo de tool
type ToolConfig struct {
	// HTTP
	Method  string            `json:"method,omitempty"`
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    map[string]any    `json:"body,omitempty"`
	Timeout int               `json:"timeout,omitempty"` // seconds

	// Database
	Query        string `json:"query,omitempty"`
	ConnectionID string `json:"connection_id,omitempty"`

	// Email
	Provider   string         `json:"provider,omitempty"`
	TemplateID string         `json:"template_id,omitempty"`
	From       string         `json:"from,omitempty"`
	Subject    string         `json:"subject,omitempty"`
	Variables  map[string]any `json:"variables,omitempty"`

	// Custom
	Runtime string `json:"runtime,omitempty"` // nodejs, python
	Code    string `json:"code,omitempty"`
	Memory  string `json:"memory,omitempty"` // 128mb, 256mb
}

// ============================================================================
// Tool Execution Entity
// ============================================================================

// ToolExecution representa una ejecución de un tool
type ToolExecution struct {
	ID        string          `db:"id" json:"id"`
	ToolID    kernel.ToolID   `db:"tool_id" json:"tool_id"`
	TenantID  kernel.TenantID `db:"tenant_id" json:"tenant_id"`
	Input     map[string]any  `db:"input" json:"input"`
	Output    map[string]any  `db:"output" json:"output"`
	Status    ExecutionStatus `db:"status" json:"status"`
	Error     string          `db:"error" json:"error,omitempty"`
	StartedAt time.Time       `db:"started_at" json:"started_at"`
	EndedAt   *time.Time      `db:"ended_at" json:"ended_at,omitempty"`
	Duration  int64           `db:"duration_ms" json:"duration_ms"` // milliseconds
}

// ExecutionStatus estado de ejecución
type ExecutionStatus string

const (
	ExecutionStatusPending ExecutionStatus = "PENDING"
	ExecutionStatusRunning ExecutionStatus = "RUNNING"
	ExecutionStatusSuccess ExecutionStatus = "SUCCESS"
	ExecutionStatusFailed  ExecutionStatus = "FAILED"
)

// ============================================================================
// Domain Methods - Tool
// ============================================================================

// IsValid verifica si el tool es válido
func (t *Tool) IsValid() bool {
	return t.Name != "" && t.Type != "" && !t.TenantID.IsEmpty()
}

// Activate activa el tool
func (t *Tool) Activate() {
	t.IsActive = true
	t.UpdatedAt = time.Now()
}

// Deactivate desactiva el tool
func (t *Tool) Deactivate() {
	t.IsActive = false
	t.UpdatedAt = time.Now()
}

// UpdateConfig actualiza la configuración del tool
func (t *Tool) UpdateConfig(config ToolConfig) {
	t.Config = config
	t.UpdatedAt = time.Now()
}

// UpdateDetails actualiza nombre y descripción
func (t *Tool) UpdateDetails(name, description string) {
	if name != "" {
		t.Name = name
	}
	if description != "" {
		t.Description = description
	}
	t.UpdatedAt = time.Now()
}

// ============================================================================
// Domain Methods - ToolExecution
// ============================================================================

// IsCompleted verifica si la ejecución terminó
func (e *ToolExecution) IsCompleted() bool {
	return e.Status == ExecutionStatusSuccess || e.Status == ExecutionStatusFailed
}

// IsSuccessful verifica si la ejecución fue exitosa
func (e *ToolExecution) IsSuccessful() bool {
	return e.Status == ExecutionStatusSuccess
}

// Complete marca la ejecución como completada
func (e *ToolExecution) Complete(output map[string]any) {
	now := time.Now()
	e.EndedAt = &now
	e.Status = ExecutionStatusSuccess
	e.Output = output
	e.Duration = now.Sub(e.StartedAt).Milliseconds()
}

// Fail marca la ejecución como fallida
func (e *ToolExecution) Fail(err error) {
	now := time.Now()
	e.EndedAt = &now
	e.Status = ExecutionStatusFailed
	e.Error = err.Error()
	e.Duration = now.Sub(e.StartedAt).Milliseconds()
}

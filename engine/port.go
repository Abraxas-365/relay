package engine

import (
	"context"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Repository Interfaces
// ============================================================================

// MessageRepository persistencia de mensajes
type MessageRepository interface {
	// CRUD básico
	Save(ctx context.Context, msg Message) error
	FindByID(ctx context.Context, id kernel.MessageID) (*Message, error)
	Delete(ctx context.Context, id kernel.MessageID) error

	// Búsquedas
	FindByChannel(ctx context.Context, channelID kernel.ChannelID) ([]*Message, error)
	FindBySender(ctx context.Context, senderID string, tenantID kernel.TenantID) ([]*Message, error)
	FindByStatus(ctx context.Context, status MessageStatus, tenantID kernel.TenantID) ([]*Message, error)

	// List con paginación
	List(ctx context.Context, req MessageListRequest) (MessageListResponse, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, ids []kernel.MessageID, status MessageStatus) error

	// Stats
	CountByStatus(ctx context.Context, status MessageStatus, tenantID kernel.TenantID) (int, error)
	CountByChannel(ctx context.Context, channelID kernel.ChannelID) (int, error)
}

// WorkflowRepository persistencia de workflows
type WorkflowRepository interface {
	// CRUD básico
	Save(ctx context.Context, wf Workflow) error
	FindByID(ctx context.Context, id kernel.WorkflowID) (*Workflow, error)
	FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*Workflow, error)
	Delete(ctx context.Context, id kernel.WorkflowID, tenantID kernel.TenantID) error
	ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error)

	// Búsquedas
	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*Workflow, error)
	FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*Workflow, error)
	FindByTriggerType(ctx context.Context, triggerType TriggerType, tenantID kernel.TenantID) ([]*Workflow, error)
	FindActiveByTrigger(ctx context.Context, trigger WorkflowTrigger, tenantID kernel.TenantID) ([]*Workflow, error)

	// List con paginación
	List(ctx context.Context, req WorkflowListRequest) (WorkflowListResponse, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, ids []kernel.WorkflowID, tenantID kernel.TenantID, isActive bool) error
}

// SessionRepository persistencia de sesiones
type SessionRepository interface {
	// CRUD básico
	Save(ctx context.Context, session Session) error
	FindByID(ctx context.Context, id kernel.SessionID) (*Session, error)
	Delete(ctx context.Context, id kernel.SessionID) error

	// Búsquedas
	FindByChannelAndSender(ctx context.Context, channelID kernel.ChannelID, senderID string) (*Session, error)
	FindByChannel(ctx context.Context, channelID kernel.ChannelID) ([]*Session, error)
	FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*Session, error)
	FindExpired(ctx context.Context) ([]*Session, error)

	// List con paginación
	List(ctx context.Context, req SessionListRequest) (SessionListResponse, error)

	// Mantenimiento
	CleanExpired(ctx context.Context) error
	ExtendExpiration(ctx context.Context, id kernel.SessionID, duration int64) error // duration en segundos

	// Stats
	CountActive(ctx context.Context, tenantID kernel.TenantID) (int, error)

	Close(ctx context.Context, id kernel.SessionID) error
	MarkExpired(ctx context.Context, id kernel.SessionID) error

	// Find only active sessions
	FindActiveByChannelAndSender(ctx context.Context, channelID kernel.ChannelID, senderID string) (*Session, error)
}

// ============================================================================
// Manager Interfaces
// ============================================================================

// SessionManager manejo de sesiones con lógica de negocio
type SessionManager interface {
	// Obtener o crear sesión
	GetOrCreate(ctx context.Context, channelID kernel.ChannelID, senderID string, tenantID kernel.TenantID) (*Session, error)

	// Actualizar sesión
	Update(ctx context.Context, session Session) error
	UpdateContext(ctx context.Context, sessionID kernel.SessionID, key string, value any) error
	UpdateState(ctx context.Context, sessionID kernel.SessionID, state string) error

	// Eliminar sesión
	Delete(ctx context.Context, sessionID kernel.SessionID) error

	// Obtener sesión
	Get(ctx context.Context, sessionID kernel.SessionID) (*Session, error)

	// Extender expiración
	ExtendSession(ctx context.Context, sessionID kernel.SessionID) error

	// Limpiar sesiones expiradas
	CleanExpiredSessions(ctx context.Context) error
}

// ============================================================================
// Executor Interfaces
// ============================================================================

// WorkflowExecutor ejecuta workflows
type WorkflowExecutor interface {
	// Ejecutar workflow completo
	Execute(ctx context.Context, workflow Workflow, message Message, session *Session) (*ExecutionResult, error)

	// Ejecutar paso específico
	ExecuteStep(ctx context.Context, step WorkflowStep, message Message, session *Session, stepContext map[string]any) (*StepResult, error)

	// Validar workflow
	ValidateWorkflow(ctx context.Context, workflow Workflow) error
}

// StepExecutor ejecuta pasos específicos de workflow
type StepExecutor interface {
	// Ejecutar paso
	Execute(ctx context.Context, step WorkflowStep, input map[string]any) (*StepResult, error)

	// Soporta el tipo de paso
	SupportsType(stepType StepType) bool

	// Validar configuración del paso
	ValidateConfig(config map[string]any) error
}

// ============================================================================
// Processor Interface
// ============================================================================

// MessageProcessor procesa mensajes entrantes
type MessageProcessor interface {
	// Procesar mensaje
	ProcessMessage(ctx context.Context, msg Message) error

	// Procesar mensaje con workflow específico
	ProcessWithWorkflow(ctx context.Context, msg Message, workflowID kernel.WorkflowID) error

	// Procesar respuesta
	ProcessResponse(ctx context.Context, msg Message, response string) error
}

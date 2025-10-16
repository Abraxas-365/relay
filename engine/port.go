package engine

import (
	"context"
	"time"

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

	ResumeFromNode(
		ctx context.Context,
		workflow Workflow,
		message Message,
		session *Session,
		startNodeID string,
		nodeContext map[string]any,
	) (*ExecutionResult, error)

	// Ejecutar paso específico
	ExecuteNode(ctx context.Context, node WorkflowNode, message Message, session *Session, nodeContext map[string]any) (*NodeResult, error)

	// Validar workflow
	ValidateWorkflow(ctx context.Context, workflow Workflow) error
}

// NodeExecutor ejecuta pasos específicos de workflow
type NodeExecutor interface {
	// Ejecutar paso
	Execute(ctx context.Context, node WorkflowNode, input map[string]any) (*NodeResult, error)

	// Soporta el tipo de paso
	SupportsType(nodeType NodeType) bool

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

// ============================================================================
// Delay Scheduler Interface
// ============================================================================

// WorkflowContinuation stores the state needed to resume workflow execution
type WorkflowContinuation struct {
	ID           string         `json:"id"`
	WorkflowID   string         `json:"workflow_id"`
	NodeID       string         `json:"node_id"`
	NextNodeID   string         `json:"next_node_id"`
	MessageID    string         `json:"message_id"`
	SessionID    string         `json:"session_id"`
	TenantID     string         `json:"tenant_id"`
	ChannelID    string         `json:"channel_id"`
	SenderID     string         `json:"sender_id"`
	NodeContext  map[string]any `json:"node_context"`
	ScheduledFor time.Time      `json:"scheduled_for"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ContinuationHandler is called when a delayed execution is ready
type ContinuationHandler func(ctx context.Context, continuation *WorkflowContinuation) error

// DelayScheduler manages delayed workflow executions
type DelayScheduler interface {
	// Schedule schedules a workflow continuation after a delay
	Schedule(ctx context.Context, continuation *WorkflowContinuation, delay time.Duration) error

	// ShouldUseAsync determines if a delay should be handled asynchronously
	ShouldUseAsync(duration time.Duration) bool

	// StartWorker starts the background worker that processes scheduled delays
	StartWorker(ctx context.Context)

	// StopWorker stops the background worker
	StopWorker()

	// GetPendingCount returns the number of pending delayed executions
	GetPendingCount(ctx context.Context) (int64, error)

	// GetContinuation retrieves a continuation by ID
	GetContinuation(ctx context.Context, id string) (*WorkflowContinuation, error)

	// Cancel cancels a scheduled continuation
	Cancel(ctx context.Context, id string) error
}

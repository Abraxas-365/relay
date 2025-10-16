package engine

import (
	"context"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Repository Interfaces
// ============================================================================

// WorkflowRepository persistence for workflows
type WorkflowRepository interface {
	Save(ctx context.Context, wf Workflow) error
	FindByID(ctx context.Context, id kernel.WorkflowID) (*Workflow, error)
	FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*Workflow, error)
	Delete(ctx context.Context, id kernel.WorkflowID, tenantID kernel.TenantID) error
	ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error)

	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*Workflow, error)
	FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*Workflow, error)
	FindByTriggerType(ctx context.Context, triggerType TriggerType, tenantID kernel.TenantID) ([]*Workflow, error)
	FindActiveByTrigger(ctx context.Context, trigger WorkflowTrigger, tenantID kernel.TenantID) ([]*Workflow, error)

	List(ctx context.Context, req WorkflowListRequest) (WorkflowListResponse, error)
	BulkUpdateStatus(ctx context.Context, ids []kernel.WorkflowID, tenantID kernel.TenantID, isActive bool) error
}

// ============================================================================
// Executor Interfaces
// ============================================================================

// WorkflowExecutor executes workflows
type WorkflowExecutor interface {
	// Execute workflow with generic input
	Execute(ctx context.Context, workflow Workflow, input WorkflowInput) (*ExecutionResult, error)

	// Resume workflow from specific node (for delay continuation)
	ResumeFromNode(
		ctx context.Context,
		workflow Workflow,
		input WorkflowInput,
		startNodeID string,
		nodeContext map[string]any,
	) (*ExecutionResult, error)

	// Validate workflow structure
	ValidateWorkflow(ctx context.Context, workflow Workflow) error
}

// NodeExecutor executes specific workflow nodes
type NodeExecutor interface {
	Execute(ctx context.Context, node WorkflowNode, input map[string]any) (*NodeResult, error)
	SupportsType(nodeType NodeType) bool
	ValidateConfig(config map[string]any) error
}

// ============================================================================
// Delay Scheduler Interface
// ============================================================================

// WorkflowContinuation stores state for resuming workflow execution
type WorkflowContinuation struct {
	ID           string         `json:"id"`
	WorkflowID   string         `json:"workflow_id"`
	TenantID     string         `json:"tenant_id"`
	NodeID       string         `json:"node_id"`
	NextNodeID   string         `json:"next_node_id"`
	NodeContext  map[string]any `json:"node_context"`
	ScheduledFor time.Time      `json:"scheduled_for"`
	CreatedAt    time.Time      `json:"created_at"`
}

// ContinuationHandler is called when delayed execution is ready
type ContinuationHandler func(ctx context.Context, continuation *WorkflowContinuation) error

// DelayScheduler manages delayed workflow executions
type DelayScheduler interface {
	Schedule(ctx context.Context, continuation *WorkflowContinuation, delay time.Duration) error
	ShouldUseAsync(duration time.Duration) bool
	StartWorker(ctx context.Context)
	StopWorker()
	GetPendingCount(ctx context.Context) (int64, error)
	GetContinuation(ctx context.Context, id string) (*WorkflowContinuation, error)
	Cancel(ctx context.Context, id string) error
}

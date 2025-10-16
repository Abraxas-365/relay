package engine

import (
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Workflow DTOs
// ============================================================================

type CreateWorkflowRequest struct {
	TenantID    kernel.TenantID `json:"tenant_id" validate:"required"`
	Name        string          `json:"name" validate:"required,min=2"`
	Description string          `json:"description,omitempty"`
	Trigger     WorkflowTrigger `json:"trigger" validate:"required"`
	Nodes       []WorkflowNode  `json:"nodes" validate:"required,min=1"`
}

type UpdateWorkflowRequest struct {
	Name        *string          `json:"name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Trigger     *WorkflowTrigger `json:"trigger,omitempty"`
	Nodes       *[]WorkflowNode  `json:"nodes,omitempty"`
	IsActive    *bool            `json:"is_active,omitempty"`
}

type ExecuteWorkflowRequest struct {
	WorkflowID  kernel.WorkflowID `json:"workflow_id" validate:"required"`
	TriggerData map[string]any    `json:"trigger_data,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
}

type WorkflowResponse struct {
	Workflow Workflow `json:"workflow"`
}

type WorkflowListRequest struct {
	storex.PaginationOptions
	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	IsActive *bool           `json:"is_active,omitempty"`
	Search   string          `json:"search,omitempty"`
}

func (wlr WorkflowListRequest) GetOffset() int {
	return (wlr.Page - 1) * wlr.PageSize
}

type WorkflowListResponse = storex.Paginated[Workflow]

type WorkflowExecutionResponse struct {
	WorkflowID    kernel.WorkflowID `json:"workflow_id"`
	Success       bool              `json:"success"`
	Output        map[string]any    `json:"output,omitempty"`
	Error         string            `json:"error,omitempty"`
	ExecutedNodes []NodeResult      `json:"executed_nodes,omitempty"`
}

// ============================================================================
// Validation DTOs
// ============================================================================

type ValidateWorkflowRequest struct {
	Trigger WorkflowTrigger `json:"trigger" validate:"required"`
	Nodes   []WorkflowNode  `json:"nodes" validate:"required,min=1"`
}

type ValidateWorkflowResponse struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ============================================================================
// Bulk Operation DTOs
// ============================================================================

type BulkWorkflowOperationRequest struct {
	TenantID    kernel.TenantID     `json:"tenant_id" validate:"required"`
	WorkflowIDs []kernel.WorkflowID `json:"workflow_ids" validate:"required,min=1"`
	Operation   string              `json:"operation" validate:"required,oneof=activate deactivate delete"`
}

type BulkWorkflowOperationResponse struct {
	Successful []kernel.WorkflowID          `json:"successful"`
	Failed     map[kernel.WorkflowID]string `json:"failed"`
	Total      int                          `json:"total"`
}

// ============================================================================
// Simple DTOs
// ============================================================================

type WorkflowDetailsDTO struct {
	ID        kernel.WorkflowID `json:"id"`
	Name      string            `json:"name"`
	IsActive  bool              `json:"is_active"`
	NodeCount int               `json:"node_count"`
}

func (w *Workflow) ToDTO() WorkflowDetailsDTO {
	return WorkflowDetailsDTO{
		ID:        w.ID,
		Name:      w.Name,
		IsActive:  w.IsActive,
		NodeCount: len(w.Nodes),
	}
}

package triggerhandler

import (
	"context"
	"fmt"
	"log"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// TriggerHandler handles workflow triggers
type TriggerHandler struct {
	workflowRepo     engine.WorkflowRepository
	workflowExecutor engine.WorkflowExecutor
}

func NewTriggerHandler(
	workflowRepo engine.WorkflowRepository,
	workflowExecutor engine.WorkflowExecutor,
) *TriggerHandler {
	return &TriggerHandler{
		workflowRepo:     workflowRepo,
		workflowExecutor: workflowExecutor,
	}
}

// HandleWebhookTrigger handles generic webhook triggers
func (h *TriggerHandler) HandleWebhookTrigger(
	ctx context.Context,
	tenantID kernel.TenantID,
	triggerData map[string]any,
) error {
	return h.executeTrigger(ctx, engine.TriggerTypeWebhook, tenantID, triggerData, nil)
}

// HandleChannelWebhookTrigger handles channel message triggers
func (h *TriggerHandler) HandleChannelWebhookTrigger(
	ctx context.Context,
	tenantID kernel.TenantID,
	channelID kernel.ChannelID,
	triggerData map[string]any,
) error {
	filters := map[string]any{
		"channel_ids": []string{channelID.String()},
	}
	return h.executeTrigger(ctx, engine.TriggerTypeChannelWebhook, tenantID, triggerData, filters)
}

// HandleScheduleTrigger handles scheduled triggers
func (h *TriggerHandler) HandleScheduleTrigger(
	ctx context.Context,
	tenantID kernel.TenantID,
	scheduleID string,
	triggerData map[string]any,
) error {
	filters := map[string]any{
		"schedule_id": scheduleID,
	}
	return h.executeTrigger(ctx, engine.TriggerTypeSchedule, tenantID, triggerData, filters)
}

// HandleManualTrigger handles manual workflow execution
func (h *TriggerHandler) HandleManualTrigger(
	ctx context.Context,
	workflowID kernel.WorkflowID,
	tenantID kernel.TenantID,
	triggerData map[string]any,
) error {
	workflow, err := h.workflowRepo.FindByID(ctx, workflowID)
	if err != nil {
		return fmt.Errorf("workflow not found: %w", err)
	}

	if workflow.TenantID != tenantID {
		return fmt.Errorf("workflow does not belong to tenant")
	}

	input := engine.WorkflowInput{
		TriggerData: triggerData,
		TenantID:    tenantID,
		Metadata: map[string]any{
			"trigger_type": engine.TriggerTypeManual,
		},
	}

	result, err := h.workflowExecutor.Execute(ctx, *workflow, input)
	if err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	log.Printf("✅ Manual workflow executed: %s (success=%v)", workflow.Name, result.Success)
	return nil
}

// executeTrigger is the core trigger execution logic
func (h *TriggerHandler) executeTrigger(
	ctx context.Context,
	triggerType engine.TriggerType,
	tenantID kernel.TenantID,
	triggerData map[string]any,
	filters map[string]any,
) error {
	log.Printf("🔔 Handling trigger: type=%s, tenant=%s", triggerType, tenantID.String())

	// Build trigger to match
	trigger := engine.WorkflowTrigger{
		Type:    triggerType,
		Filters: filters,
	}

	// Find matching workflows
	workflows, err := h.workflowRepo.FindActiveByTrigger(ctx, trigger, tenantID)
	if err != nil {
		return fmt.Errorf("failed to find workflows: %w", err)
	}

	if len(workflows) == 0 {
		log.Printf("ℹ️  No active workflows found for trigger type: %s", triggerType)
		return nil
	}

	log.Printf("📋 Found %d matching workflow(s)", len(workflows))

	// Execute each matching workflow (async to not block)
	for _, workflow := range workflows {
		go func(wf *engine.Workflow) {
			log.Printf("▶️  Executing workflow: %s", wf.Name)

			input := engine.WorkflowInput{
				TriggerData: triggerData,
				TenantID:    tenantID,
				Metadata: map[string]any{
					"trigger_type": triggerType,
					"workflow_id":  wf.ID.String(),
				},
			}

			result, err := h.workflowExecutor.Execute(ctx, *wf, input)
			if err != nil {
				log.Printf("❌ Workflow %s execution failed: %v", wf.Name, err)
				return
			}

			log.Printf("✅ Workflow %s executed (success=%v, nodes=%d)",
				wf.Name, result.Success, len(result.ExecutedNodes))
		}(workflow)
	}

	return nil
}

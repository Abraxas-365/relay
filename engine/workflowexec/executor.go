package workflowexec

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
)

type DefaultWorkflowExecutor struct {
	nodeExecutors       map[engine.NodeType]engine.NodeExecutor
	expressionEvaluator engine.ExpressionEvaluator
}

var _ engine.WorkflowExecutor = (*DefaultWorkflowExecutor)(nil)

func NewDefaultWorkflowExecutor(
	expressionEvaluator engine.ExpressionEvaluator,
	nodeExecutors ...engine.NodeExecutor,
) *DefaultWorkflowExecutor {
	executor := &DefaultWorkflowExecutor{
		nodeExecutors:       make(map[engine.NodeType]engine.NodeExecutor),
		expressionEvaluator: expressionEvaluator,
	}

	for _, nodeExec := range nodeExecutors {
		executor.RegisterNodeExecutor(nodeExec)
	}

	return executor
}

func (e *DefaultWorkflowExecutor) RegisterNodeExecutor(executor engine.NodeExecutor) {
	// Register for all supported types
	for _, nodeType := range []engine.NodeType{
		engine.NodeTypeCondition,
		engine.NodeTypeAction,
		engine.NodeTypeDelay,
		engine.NodeTypeAIAgent,
		engine.NodeTypeSendMessage,
		engine.NodeTypeHTTP,
		engine.NodeTypeTransform,
		engine.NodeTypeSwitch,
		engine.NodeTypeLoop,
		engine.NodeTypeValidate,
	} {
		if executor.SupportsType(nodeType) {
			e.nodeExecutors[nodeType] = executor
			log.Printf("✅ Registered executor for node type: %s", nodeType)
		}
	}
}

// ============================================================================
// Execute - Main workflow execution
// ============================================================================

func (e *DefaultWorkflowExecutor) Execute(
	ctx context.Context,
	workflow engine.Workflow,
	input engine.WorkflowInput,
) (*engine.ExecutionResult, error) {
	log.Printf("🚀 Starting workflow execution: %s", workflow.Name)

	startTime := time.Now()
	result := &engine.ExecutionResult{
		Success:       true,
		Output:        make(map[string]any),
		ExecutedNodes: []engine.NodeResult{},
	}

	if err := e.ValidateWorkflow(ctx, workflow); err != nil {
		return nil, errx.Wrap(err, "workflow validation failed", errx.TypeValidation)
	}

	// Prepare initial context from input
	nodeContext := e.prepareInitialContext(input)

	// Start from first node
	currentNodeID := ""
	if len(workflow.Nodes) > 0 {
		currentNodeID = workflow.Nodes[0].ID
	}

	visitedNodes := make(map[string]bool)
	maxNodes := len(workflow.Nodes) * 2

	for currentNodeID != "" && len(result.ExecutedNodes) < maxNodes {
		if visitedNodes[currentNodeID] {
			return nil, engine.ErrCyclicWorkflow().
				WithDetail("node_id", currentNodeID).
				WithDetail("workflow_id", workflow.ID.String())
		}
		visitedNodes[currentNodeID] = true

		node := workflow.GetNodeByID(currentNodeID)
		if node == nil {
			return nil, engine.ErrNodeNotFound().WithDetail("node_id", currentNodeID)
		}

		// Evaluate expressions in config
		evaluatedConfig, err := e.evaluateNodeConfig(ctx, node.Config, nodeContext)
		if err != nil {
			nodeResult := &engine.NodeResult{
				NodeID:    node.ID,
				NodeName:  node.Name,
				Success:   false,
				Error:     fmt.Sprintf("expression evaluation failed: %v", err),
				Timestamp: time.Now(),
			}
			result.ExecutedNodes = append(result.ExecutedNodes, *nodeResult)
			result.Success = false
			result.ErrorMessage = nodeResult.Error
			break
		}

		nodeForExecution := *node
		nodeForExecution.Config = evaluatedConfig

		// Execute node
		nodeResult, err := e.executeNodeInternal(ctx, nodeForExecution, nodeContext, result)
		if err != nil && nodeResult == nil {
			nodeResult = &engine.NodeResult{
				NodeID: node.ID, NodeName: node.Name, Success: false,
				Error: err.Error(), Timestamp: time.Now(),
			}
		}

		result.ExecutedNodes = append(result.ExecutedNodes, *nodeResult)

		// Check for workflow pause (async delay)
		if paused, ok := nodeResult.Output["__workflow_paused"].(bool); ok && paused {
			log.Printf("⏸️  Workflow paused for async delay")
			result.Success = true
			return result, nil
		}

		if !nodeResult.Success {
			result.Success = false
			result.Error = fmt.Errorf("node %s failed: %s", node.Name, nodeResult.Error)
			result.ErrorMessage = nodeResult.Error
			if node.OnFailure != "" {
				currentNodeID = node.OnFailure
				continue
			}
			break
		}

		// Update context with node output
		if nodeResult.Output != nil {
			nodeContext[node.ID] = map[string]any{
				"output":      nodeResult.Output,
				"success":     nodeResult.Success,
				"duration_ms": nodeResult.Duration,
			}

			for key, value := range nodeResult.Output {
				result.Output[key] = value
			}
		}

		// Determine next node
		if nextNodeOverride, ok := nodeContext["__next_node"].(string); ok {
			currentNodeID = nextNodeOverride
			delete(nodeContext, "__next_node")
		} else if node.OnSuccess != "" {
			currentNodeID = node.OnSuccess
		} else {
			currentNodeID = ""
		}
	}

	duration := time.Since(startTime)
	log.Printf("✅ Workflow execution completed: %s in %v", workflow.Name, duration)

	return result, nil
}

// ============================================================================
// ResumeFromNode - Resume workflow after delay
// ============================================================================

func (e *DefaultWorkflowExecutor) ResumeFromNode(
	ctx context.Context,
	workflow engine.Workflow,
	input engine.WorkflowInput,
	startNodeID string,
	savedNodeContext map[string]any,
) (*engine.ExecutionResult, error) {
	log.Printf("🔄 Resuming workflow: %s from node: %s", workflow.Name, startNodeID)

	startTime := time.Now()
	result := &engine.ExecutionResult{
		Success:       true,
		Output:        make(map[string]any),
		ExecutedNodes: []engine.NodeResult{},
	}

	if err := e.ValidateWorkflow(ctx, workflow); err != nil {
		return nil, errx.Wrap(err, "workflow validation failed", errx.TypeValidation)
	}

	startNode := workflow.GetNodeByID(startNodeID)
	if startNode == nil {
		return nil, engine.ErrNodeNotFound().WithDetail("node_id", startNodeID)
	}

	// Use saved context or create new
	nodeContext := savedNodeContext
	if nodeContext == nil {
		nodeContext = e.prepareInitialContext(input)
	}

	// Ensure trigger data is available
	if _, ok := nodeContext["trigger"]; !ok {
		nodeContext["trigger"] = input.TriggerData
	}

	currentNodeID := startNodeID
	visitedNodes := make(map[string]bool)
	maxNodes := len(workflow.Nodes) * 2

	for currentNodeID != "" && len(result.ExecutedNodes) < maxNodes {
		if visitedNodes[currentNodeID] {
			return nil, engine.ErrCyclicWorkflow().
				WithDetail("node_id", currentNodeID).
				WithDetail("workflow_id", workflow.ID.String())
		}
		visitedNodes[currentNodeID] = true

		node := workflow.GetNodeByID(currentNodeID)
		if node == nil {
			return nil, engine.ErrNodeNotFound().WithDetail("node_id", currentNodeID)
		}

		evaluatedConfig, err := e.evaluateNodeConfig(ctx, node.Config, nodeContext)
		if err != nil {
			nodeResult := &engine.NodeResult{
				NodeID: node.ID, NodeName: node.Name, Success: false,
				Error: fmt.Sprintf("expression evaluation failed: %v", err), Timestamp: time.Now(),
			}
			result.ExecutedNodes = append(result.ExecutedNodes, *nodeResult)
			result.Success = false
			result.ErrorMessage = nodeResult.Error
			break
		}

		nodeForExecution := *node
		nodeForExecution.Config = evaluatedConfig

		nodeResult, err := e.executeNodeInternal(ctx, nodeForExecution, nodeContext, result)
		if err != nil && nodeResult == nil {
			nodeResult = &engine.NodeResult{
				NodeID: node.ID, NodeName: node.Name, Success: false,
				Error: err.Error(), Timestamp: time.Now(),
			}
		}

		result.ExecutedNodes = append(result.ExecutedNodes, *nodeResult)

		if !nodeResult.Success {
			result.Success = false
			result.Error = fmt.Errorf("node %s failed: %s", node.Name, nodeResult.Error)
			result.ErrorMessage = nodeResult.Error
			if node.OnFailure != "" {
				currentNodeID = node.OnFailure
				continue
			}
			break
		}

		if nodeResult.Output != nil {
			nodeContext[node.ID] = map[string]any{
				"output":      nodeResult.Output,
				"success":     nodeResult.Success,
				"duration_ms": nodeResult.Duration,
			}

			for key, value := range nodeResult.Output {
				result.Output[key] = value
			}
		}

		if nextNodeOverride, ok := nodeContext["__next_node"].(string); ok {
			currentNodeID = nextNodeOverride
			delete(nodeContext, "__next_node")
		} else if node.OnSuccess != "" {
			currentNodeID = node.OnSuccess
		} else {
			currentNodeID = ""
		}
	}

	duration := time.Since(startTime)
	log.Printf("✅ Workflow resume completed: %s in %v", workflow.Name, duration)

	return result, nil
}

// ============================================================================
// Internal Execution
// ============================================================================

func (e *DefaultWorkflowExecutor) executeNodeInternal(
	ctx context.Context,
	node engine.WorkflowNode,
	nodeContext map[string]any,
	workflowResult *engine.ExecutionResult,
) (*engine.NodeResult, error) {
	log.Printf("⚡ Executing node: %s (type: %s)", node.Name, node.Type)
	startTime := time.Now()

	if node.Timeout != nil && *node.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*node.Timeout)*time.Second)
		defer cancel()
	}

	nodeResult := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Success:   true,
		Output:    make(map[string]any),
		Timestamp: startTime,
	}

	var err error

	// Check for registered executor
	if executor, ok := e.nodeExecutors[node.Type]; ok {
		input := nodeContext // Pass entire context as input
		nodeResult, err = executor.Execute(ctx, node, input)

		if nodeResult.NodeID == "" {
			nodeResult.NodeID = node.ID
		}
		if nodeResult.NodeName == "" {
			nodeResult.NodeName = node.Name
		}

		if err == nil && nodeResult.Output != nil {
			for key, value := range nodeResult.Output {
				workflowResult.Output[key] = value
			}
		}
	} else {
		err = engine.ErrInvalidWorkflowNode().
			WithDetail("node_type", string(node.Type)).
			WithDetail("reason", "no executor found for node type")
	}

	nodeResult.Duration = time.Since(startTime).Milliseconds()

	if err != nil {
		nodeResult.Success = false
		nodeResult.Error = err.Error()
		return nodeResult, err
	}

	return nodeResult, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func (e *DefaultWorkflowExecutor) prepareInitialContext(input engine.WorkflowInput) map[string]any {
	context := make(map[string]any)

	// Add trigger data
	context["trigger"] = input.TriggerData
	context["tenant_id"] = input.TenantID.String()

	// Add metadata
	if input.Metadata != nil {
		for key, value := range input.Metadata {
			context[key] = value
		}
	}

	return context
}

func (e *DefaultWorkflowExecutor) evaluateNodeConfig(
	ctx context.Context,
	config map[string]any,
	nodeContext map[string]any,
) (map[string]any, error) {
	evaluatedData, err := e.expressionEvaluator.Evaluate(ctx, config, nodeContext)
	if err != nil {
		return nil, err
	}

	evaluatedConfig, ok := evaluatedData.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expression evaluation did not return valid config map")
	}

	return evaluatedConfig, nil
}

// ============================================================================
// Validation
// ============================================================================

func (e *DefaultWorkflowExecutor) ValidateWorkflow(ctx context.Context, workflow engine.Workflow) error {
	if !workflow.IsValid() {
		return engine.ErrInvalidWorkflowConfig().WithDetail("reason", "workflow is not valid")
	}

	if len(workflow.Nodes) == 0 {
		return engine.ErrInvalidWorkflowConfig().WithDetail("reason", "workflow has no nodes")
	}

	nodeIDs := make(map[string]bool)
	for _, node := range workflow.Nodes {
		if node.ID == "" {
			return engine.ErrInvalidWorkflowNode().WithDetail("reason", "node has no ID")
		}
		if nodeIDs[node.ID] {
			return engine.ErrInvalidWorkflowNode().
				WithDetail("node_id", node.ID).
				WithDetail("reason", "duplicate node ID")
		}
		nodeIDs[node.ID] = true

		if executor, ok := e.nodeExecutors[node.Type]; ok {
			if err := executor.ValidateConfig(node.Config); err != nil {
				return errx.Wrap(err, "node config validation failed", errx.TypeValidation).
					WithDetail("node_id", node.ID).
					WithDetail("node_name", node.Name)
			}
		}
	}

	for _, node := range workflow.Nodes {
		if node.OnSuccess != "" && !nodeIDs[node.OnSuccess] {
			return engine.ErrInvalidWorkflowNode().
				WithDetail("node_id", node.ID).
				WithDetail("on_success", node.OnSuccess).
				WithDetail("reason", "on_success references non-existent node")
		}
		if node.OnFailure != "" && !nodeIDs[node.OnFailure] {
			return engine.ErrInvalidWorkflowNode().
				WithDetail("node_id", node.ID).
				WithDetail("on_failure", node.OnFailure).
				WithDetail("reason", "on_failure references non-existent node")
		}
	}

	return nil
}

// Utility functions
func compareNumeric(a, b any, operator string) bool {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)

	if !aOk || !bOk {
		return false
	}

	switch operator {
	case "gt":
		return aFloat > bFloat
	case "gte":
		return aFloat >= bFloat
	case "lt":
		return aFloat < bFloat
	case "lte":
		return aFloat <= bFloat
	default:
		return false
	}
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}


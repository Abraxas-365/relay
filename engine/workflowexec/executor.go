package workflowexec

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/parser"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type DefaultWorkflowExecutor struct {
	nodeExecutors       map[engine.NodeType]engine.NodeExecutor
	parserManager       *parser.ParserManager
	channelManager      channels.ChannelManager
	expressionEvaluator engine.ExpressionEvaluator
}

var _ engine.WorkflowExecutor = (*DefaultWorkflowExecutor)(nil)

// MODIFIED: Added expressionEvaluator
func NewDefaultWorkflowExecutor(
	parserManager *parser.ParserManager,
	channelManager channels.ChannelManager,
	expressionEvaluator engine.ExpressionEvaluator,
	nodeExecutors ...engine.NodeExecutor,
) *DefaultWorkflowExecutor {
	executor := &DefaultWorkflowExecutor{
		nodeExecutors:       make(map[engine.NodeType]engine.NodeExecutor),
		parserManager:       parserManager,
		channelManager:      channelManager,
		expressionEvaluator: expressionEvaluator,
	}

	for _, nodeExec := range nodeExecutors {
		executor.RegisterNodeExecutor(nodeExec)
	}

	return executor
}

// RegisterNodeExecutor registers a node executor
func (e *DefaultWorkflowExecutor) RegisterNodeExecutor(executor engine.NodeExecutor) {
	for _, nodeType := range []engine.NodeType{
		engine.NodeTypeCondition,
		engine.NodeTypeParser,
		engine.NodeTypeAction,
		engine.NodeTypeDelay,
		engine.NodeTypeResponse,
	} {
		if executor.SupportsType(nodeType) {
			e.nodeExecutors[nodeType] = executor
			log.Printf("✓ Registered executor for node type: %s", nodeType)
		}
	}
}

// Execute runs a full workflow.
func (e *DefaultWorkflowExecutor) Execute(
	ctx context.Context,
	workflow engine.Workflow,
	message engine.Message,
	session *engine.Session,
) (*engine.ExecutionResult, error) {
	log.Printf("🚀 Starting workflow execution: %s for message: %s", workflow.Name, message.ID.String())

	startTime := time.Now()
	result := &engine.ExecutionResult{
		Success:       true,
		ShouldRespond: false,
		Context:       make(map[string]any),
		Executedodes:  []engine.NodeResult{},
	}

	if err := e.ValidateWorkflow(ctx, workflow); err != nil {
		return nil, errx.Wrap(err, "workflow validation failed", errx.TypeValidation)
	}

	nodeContext := e.prepareInitialContext(message, session)

	currentNodeID := ""
	if len(workflow.Node) > 0 {
		currentNodeID = workflow.Node[0].ID
	}

	visitedNodes := make(map[string]bool)
	maxNodes := len(workflow.Node) * 2 // Prevent infinite loops

	for currentNodeID != "" && len(result.Executedodes) < maxNodes {
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

		configToEvaluate := make(map[string]any)
		for k, v := range node.Config {
			configToEvaluate[k] = v
		}

		evaluatedData, err := e.expressionEvaluator.Evaluate(ctx, configToEvaluate, nodeContext)
		if err != nil {
			nodeResult := &engine.NodeResult{
				NodeID:    node.ID,
				NodeName:  node.Name,
				Success:   false,
				Error:     fmt.Sprintf("expression evaluation failed: %v", err),
				Timestamp: time.Now(),
			}
			result.Executedodes = append(result.Executedodes, *nodeResult)
			result.Success = false
			result.ErrorMessage = nodeResult.Error
			break // Stop workflow execution
		}

		evaluatedConfig, ok := evaluatedData.(map[string]any)
		if !ok {
			nodeResult := &engine.NodeResult{
				NodeID:    node.ID,
				NodeName:  node.Name,
				Success:   false,
				Error:     "expression evaluation did not return a valid configuration map",
				Timestamp: time.Now(),
			}
			result.Executedodes = append(result.Executedodes, *nodeResult)
			result.Success = false
			result.ErrorMessage = nodeResult.Error
			break // Stop workflow execution
		}

		nodeForExecution := *node
		nodeForExecution.Config = evaluatedConfig
		// ========================================================================
		// END OF MODIFICATION
		// ========================================================================

		// Execute node with the *evaluated* config
		nodeResult, err := e.executeNodeInternal(ctx, nodeForExecution, message, session, nodeContext, result)
		if err != nil {
			if nodeResult == nil {
				nodeResult = &engine.NodeResult{
					NodeID: node.ID, NodeName: node.Name, Success: false,
					Error: err.Error(), Timestamp: time.Now(),
				}
			} else if nodeResult.Error == "" {
				nodeResult.Success = false
				nodeResult.Error = err.Error()
			}
		}

		result.Executedodes = append(result.Executedodes, *nodeResult)

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

		// ========================================================================
		// MODIFIED: STRUCTURED CONTEXT FOR DATA PIPELINING
		// ========================================================================
		if nodeResult.Output != nil {
			// Store the node's output in a structured way under its ID.
			// This allows expressions like {{node_1.output.userId}}.
			nodeContext[node.ID] = map[string]any{
				"output":      nodeResult.Output,
				"success":     nodeResult.Success,
				"duration_ms": nodeResult.Duration,
			}

			// Also merge into the top-level final result for convenience.
			for key, value := range nodeResult.Output {
				result.Context[key] = value
			}
		}
		// ========================================================================
		// END OF MODIFICATION
		// ========================================================================

		if responseText, ok := nodeResult.Output["response"].(string); ok && responseText != "" {
			result.Response = responseText
			result.ShouldRespond = true
		}
		if shouldRespond, ok := nodeResult.Output["should_respond"].(bool); ok {
			result.ShouldRespond = shouldRespond
		}
		if nextState, ok := nodeResult.Output["next_state"].(string); ok && nextState != "" {
			result.NextState = nextState
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
	log.Printf("✅ Workflow execution completed: %s in %v", workflow.Name, duration)

	return result, nil
}

// executeNodeInternal executes a single node with full integration
func (e *DefaultWorkflowExecutor) executeNodeInternal(
	ctx context.Context,
	node engine.WorkflowNode,
	message engine.Message,
	session *engine.Session,
	nodeContext map[string]any,
	workflowResult *engine.ExecutionResult,
) (*engine.NodeResult, error) {
	log.Printf("⚡ Executing node: %s (type: %s)", node.Name, node.Type)
	startTime := time.Now()

	// Apply timeout if configured
	if node.Timeout != nil && *node.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*node.Timeout)*time.Second)
		defer cancel()
	}

	// Create result base
	nodeResult := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Success:   true,
		Output:    make(map[string]any),
		Timestamp: startTime,
	}

	var err error

	// Execute according to node type
	switch node.Type {
	case engine.NodeTypeParser:
		err = e.executeParserNode(ctx, node, message, session, nodeResult, workflowResult, nodeContext)
	case engine.NodeTypeCondition:
		err = e.executeConditionNode(ctx, node, message, session, nodeResult, nodeContext)
	case engine.NodeTypeDelay:
		err = e.executeDelayNode(ctx, node, nodeResult)

	default:
		// Try with registered executors (THIS WILL NOW HANDLE RESPONSE TYPE)
		if executor, ok := e.nodeExecutors[node.Type]; ok {
			input := e.prepareNodeInput(message, session, nodeContext)
			// The original nodeResult is passed here, so the executor can populate it.
			var execErr error
			nodeResult, execErr = executor.Execute(ctx, node, input)
			if execErr != nil {
				err = execErr
			}

			// Ensure essential fields are set if the executor didn't set them
			if nodeResult.NodeID == "" {
				nodeResult.NodeID = node.ID
			}
			if nodeResult.NodeName == "" {
				nodeResult.NodeName = node.Name
			}

			// Merge node result output into workflow context
			if err == nil && nodeResult.Output != nil {
				for key, value := range nodeResult.Output {
					workflowResult.Context[key] = value
					// nodeContext is updated in the main loop after this function returns
				}
			}
		} else {
			err = engine.ErrInvalidWorkflowNode().
				WithDetail("node_type", string(node.Type)).
				WithDetail("reason", "no executor found for node type")
		}
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
// Parser Node Execution
// ============================================================================

func (e *DefaultWorkflowExecutor) executeParserNode(
	ctx context.Context,
	node engine.WorkflowNode,
	msg engine.Message,
	session *engine.Session,
	nodeResult *engine.NodeResult,
	workflowResult *engine.ExecutionResult,
	nodeContext map[string]any,
) error {
	// Get parser ID from config
	parserIDStr, ok := node.Config["parser_id"].(string)
	if !ok {
		return parser.ErrParserIDNotFound().WithDetail("node_id", node.ID)
	}

	parserID := kernel.ParserID(parserIDStr)

	// Execute parser
	parseResult, err := e.parserManager.ExecuteParserWithConfig(
		ctx,
		parserID,
		msg.TenantID,
		msg,
		session,
		node.Config,
	)
	if err != nil {
		return parser.ErrNodeExecutionFailed().
			WithDetail("node_id", node.ID).
			WithDetail("parser_id", parserIDStr).
			WithCause(err)
	}

	// Store parse result in node output
	nodeResult.Output["parse_result"] = parseResult
	nodeResult.Output["confidence"] = parseResult.Confidence
	nodeResult.Output["extracted_data"] = parseResult.ExtractedData
	nodeResult.Output["parser_success"] = parseResult.Success

	// Merge parser context into workflow context
	if parseResult.Context != nil {
		for k, v := range parseResult.Context {
			workflowResult.Context[k] = v
			nodeContext[k] = v
		}
	}

	// Merge extracted data into node context
	if parseResult.ExtractedData != nil {
		for k, v := range parseResult.ExtractedData {
			nodeContext[fmt.Sprintf("extracted_%s", k)] = v
		}
	}

	// Handle parser actions
	if parseResult.HasActions() {
		actionsExecuted, err := e.executeParserActions(ctx, parseResult, msg, session, workflowResult, nodeContext)
		nodeResult.Output["actions_executed"] = actionsExecuted
		if err != nil {
			return parser.ErrActionExecutionFailed().
				WithDetail("node_id", node.ID).
				WithCause(err)
		}
	}

	// Auto-respond if configured and parser has response
	autoRespond, _ := node.Config["auto_respond"].(bool)
	if autoRespond && parseResult.ShouldRespond && parseResult.Response != "" {
		nodeResult.Output["response"] = parseResult.Response
		nodeResult.Output["should_respond"] = true
	}

	// Check success based on parser result and optional min_confidence
	if !parseResult.Success {
		return parser.ErrParsingFailed().
			WithDetail("reason", parseResult.Error).
			WithDetail("parser_id", parserIDStr)
	}

	// Check min confidence if specified
	if minConf, ok := node.Config["min_confidence"].(float64); ok {
		if parseResult.Confidence < minConf {
			return parser.ErrLowConfidence().
				WithDetail("confidence", fmt.Sprintf("%.2f", parseResult.Confidence)).
				WithDetail("min_confidence", fmt.Sprintf("%.2f", minConf))
		}
	}

	return nil
}

// ============================================================================
// Parser Actions Execution
// ============================================================================

func (e *DefaultWorkflowExecutor) executeParserActions(
	ctx context.Context,
	parseResult *parser.ParseResult,
	msg engine.Message,
	session *engine.Session,
	workflowResult *engine.ExecutionResult,
	nodeContext map[string]any,
) ([]string, error) {
	executed := []string{}

	for i, action := range parseResult.Actions {
		switch action.Type {
		case parser.ActionTypeResponse:
			if message, ok := action.Config["message"].(string); ok {
				workflowResult.Response = message
				workflowResult.ShouldRespond = true
				executed = append(executed, string(action.Type))
			}

		case parser.ActionTypeSetContext:
			if key, ok := action.Config["key"].(string); ok {
				if value, ok := action.Config["value"]; ok {
					workflowResult.Context[key] = value
					nodeContext[key] = value
					if session != nil {
						session.SetContext(key, value)
					}
					executed = append(executed, string(action.Type))
				}
			}

		case parser.ActionTypeSetState:
			if state, ok := action.Config["state"].(string); ok {
				workflowResult.NextState = state
				if session != nil {
					session.UpdateState(state)
				}
				executed = append(executed, string(action.Type))
			}

		case parser.ActionTypeRoute:
			if nextNode, ok := action.Config["next_node"].(string); ok {
				// Store for routing logic
				nodeContext["__next_node"] = nextNode
				executed = append(executed, string(action.Type))
			}

		case parser.ActionTypeWebhook:
			if err := e.executeWebhookAction(ctx, action); err != nil {
				return executed, parser.ErrWebhookExecutionFailed().
					WithDetail("action_index", fmt.Sprintf("%d", i)).
					WithCause(err)
			}
			executed = append(executed, string(action.Type))

		case parser.ActionTypeDelay:
			if duration, ok := action.Config["duration"].(float64); ok {
				time.Sleep(time.Duration(duration) * time.Second)
				executed = append(executed, string(action.Type))
			}

		case parser.ActionTypeTriggerWorkflow:
			// Store workflow trigger request
			if workflowID, ok := action.Config["workflow_id"].(string); ok {
				nodeContext["__trigger_workflow"] = workflowID
				executed = append(executed, string(action.Type))
			}
		}
	}

	return executed, nil
}

func (e *DefaultWorkflowExecutor) executeWebhookAction(ctx context.Context, action parser.Action) error {
	webhookURL, ok := action.Config["url"].(string)
	if !ok {
		return parser.ErrInvalidActionConfig().WithDetail("reason", "webhook url not found")
	}

	log.Printf("📬 Webhook action: %s", webhookURL)
	// TODO: Implement actual webhook HTTP POST request here

	return nil
}

// ============================================================================
// Other Node Types
// ============================================================================

func (e *DefaultWorkflowExecutor) executeConditionNode(
	ctx context.Context,
	node engine.WorkflowNode,
	msg engine.Message,
	session *engine.Session,
	nodeResult *engine.NodeResult,
	nodeContext map[string]any,
) error {
	// conditionType, _ := node.Config["type"].(string)
	field, _ := node.Config["field"].(string)
	operator, _ := node.Config["operator"].(string)
	value := node.Config["value"]

	var actualValue any
	var conditionMet bool

	// Resolve the field to get the actual value from the context
	if val, ok := nodeContext[field]; ok {
		actualValue = val
	} else {
		// Fallback for direct message/session access for simplicity, though expressions are preferred
		switch field {
		case "message.text":
			actualValue = msg.Content.Text
		case "session.state":
			if session != nil {
				actualValue = session.CurrentState
			}
		}
	}

	switch operator {
	case "equals":
		conditionMet = fmt.Sprintf("%v", actualValue) == fmt.Sprintf("%v", value)
	case "contains":
		if str, ok := actualValue.(string); ok {
			if valStr, ok := value.(string); ok {
				conditionMet = strings.Contains(str, valStr)
			}
		}
	case "gt", "gte", "lt", "lte":
		conditionMet = compareNumeric(actualValue, value, operator)
	case "exists":
		_, exists := nodeContext[field]
		conditionMet = exists
	default:
		return engine.ErrInvalidWorkflowNode().
			WithDetail("reason", "unsupported operator").
			WithDetail("operator", operator)
	}

	nodeResult.Output["condition_met"] = conditionMet

	if !conditionMet {
		// This is not an execution error, but a logical failure.
		// The main loop will handle routing to OnFailure.
		nodeResult.Success = false
		nodeResult.Error = "condition not met"
		return nil
	}

	return nil
}

func (e *DefaultWorkflowExecutor) executeToolNode(
	ctx context.Context,
	node engine.WorkflowNode,
	msg engine.Message,
	session *engine.Session,
	nodeResult *engine.NodeResult,
	workflowResult *engine.ExecutionResult,
	nodeContext map[string]any,
) error {
	// TODO: Implement Tool Manager integration
	// toolID, ok := node.Config["tool_id"].(string)
	// ...
	log.Println("Tool node execution is not yet implemented.")
	nodeResult.Output["status"] = "skipped"
	return nil
}

func (e *DefaultWorkflowExecutor) executeDelayNode(
	ctx context.Context,
	node engine.WorkflowNode,
	nodeResult *engine.NodeResult,
) error {
	duration, ok := node.Config["duration_ms"].(float64)
	if !ok {
		return engine.ErrInvalidWorkflowNode().
			WithDetail("reason", "duration_ms not found or not a number in config").
			WithDetail("node_id", node.ID)
	}

	select {
	case <-time.After(time.Duration(duration) * time.Millisecond):
		nodeResult.Output["delayed_ms"] = duration
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *DefaultWorkflowExecutor) ExecuteNode(
	ctx context.Context,
	node engine.WorkflowNode,
	message engine.Message,
	session *engine.Session,
	nodeContext map[string]any,
) (*engine.NodeResult, error) {
	workflowResult := &engine.ExecutionResult{
		Context: make(map[string]any),
	}
	return e.executeNodeInternal(ctx, node, message, session, nodeContext, workflowResult)
}

// ============================================================================
// Validation
// ============================================================================

func (e *DefaultWorkflowExecutor) ValidateWorkflow(ctx context.Context, workflow engine.Workflow) error {
	if !workflow.IsValid() {
		return engine.ErrInvalidWorkflowConfig().WithDetail("reason", "workflow is not valid")
	}

	if len(workflow.Node) == 0 {
		return engine.ErrInvalidWorkflowConfig().WithDetail("reason", "workflow has no node")
	}

	nodeIDs := make(map[string]bool)
	for _, node := range workflow.Node {
		if node.ID == "" {
			return engine.ErrInvalidWorkflowNode().WithDetail("reason", "node has no ID")
		}
		if nodeIDs[node.ID] {
			return engine.ErrInvalidWorkflowNode().
				WithDetail("node_id", node.ID).
				WithDetail("reason", "duplicate node ID")
		}
		nodeIDs[node.ID] = true

		// Validate node config if an executor is registered for its type
		if executor, ok := e.nodeExecutors[node.Type]; ok {
			if err := executor.ValidateConfig(node.Config); err != nil {
				return errx.Wrap(err, "node config validation failed", errx.TypeValidation).
					WithDetail("node_id", node.ID).
					WithDetail("node_name", node.Name)
			}
		}
	}

	for _, node := range workflow.Node {
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

// ============================================================================
// Helper Functions
// ============================================================================

func (e *DefaultWorkflowExecutor) prepareInitialContext(message engine.Message, session *engine.Session) map[string]any {
	context := make(map[string]any)

	// Add message information in a structured way
	context["message"] = map[string]any{
		"id":      message.ID.String(),
		"text":    message.Content.Text,
		"type":    message.Content.Type,
		"sender":  message.SenderID,
		"channel": message.ChannelID.String(),
		"context": message.Context,
	}

	// Add session information in a structured way
	if session != nil {
		context["session"] = map[string]any{
			"id":      session.ID,
			"state":   session.CurrentState,
			"context": session.Context,
		}
	}

	return context
}

func (e *DefaultWorkflowExecutor) prepareNodeInput(
	message engine.Message,
	session *engine.Session,
	nodeContext map[string]any,
) map[string]any {
	input := make(map[string]any)

	// Copy the entire node context to be the input for the next node
	for key, value := range nodeContext {
		input[key] = value
	}

	return input
}

// This function is less critical now that CEL-go handles expressions, but can be kept for simple cases or legacy use.
func (e *DefaultWorkflowExecutor) replaceTemplateVariables(template string, context map[string]any) string {
	result := template
	// This simple replacement is not robust for nested maps.
	// The primary mechanism should be the CEL evaluator.
	// This can be left as a fallback or for very simple, non-nested variables.
	return result
}

// ============================================================================
// Utility Functions
// ============================================================================

func findSubstring(s, substr string) bool {
	// This is a simplified implementation for demonstration.
	// In a real application, `strings.Contains` is more efficient.
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

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
	case string: // Attempt to parse from string
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err == nil
	default:
		return 0, false
	}
}

func (e *DefaultWorkflowExecutor) ResumeFromNode(
	ctx context.Context,
	workflow engine.Workflow,
	message engine.Message,
	session *engine.Session,
	startNodeID string,
	savedNodeContext map[string]any,
) (*engine.ExecutionResult, error) {
	log.Printf("🔄 Resuming workflow execution: %s from node: %s", workflow.Name, startNodeID)

	startTime := time.Now()
	result := &engine.ExecutionResult{
		Success:       true,
		ShouldRespond: false,
		Context:       make(map[string]any),
		Executedodes:  []engine.NodeResult{},
	}

	// Validate workflow
	if err := e.ValidateWorkflow(ctx, workflow); err != nil {
		return nil, errx.Wrap(err, "workflow validation failed", errx.TypeValidation)
	}

	// Validate start node exists
	startNode := workflow.GetNodeByID(startNodeID)
	if startNode == nil {
		return nil, engine.ErrNodeNotFound().WithDetail("node_id", startNodeID)
	}

	// Use saved context as initial context
	nodeContext := savedNodeContext
	if nodeContext == nil {
		nodeContext = make(map[string]any)
	}

	// Ensure message and session are available in context
	if _, ok := nodeContext["message"]; !ok {
		nodeContext["message"] = map[string]any{
			"id":      message.ID.String(),
			"text":    message.Content.Text,
			"type":    message.Content.Type,
			"sender":  message.SenderID,
			"channel": message.ChannelID.String(),
			"context": message.Context,
		}
	}

	if _, ok := nodeContext["session"]; !ok && session != nil {
		nodeContext["session"] = map[string]any{
			"id":      session.ID,
			"state":   session.CurrentState,
			"context": session.Context,
		}
	}

	// Start from the specified node
	currentNodeID := startNodeID
	visitedNodes := make(map[string]bool)
	maxNode := len(workflow.Node) * 2 // Prevent infinite loops

	for currentNodeID != "" && len(result.Executedodes) < maxNode {
		if visitedNodes[currentNodeID] {
			return nil, engine.ErrCyclicWorkflow().
				WithDetail("node_id", currentNodeID).
				WithDetail("workflow_id", workflow.ID.String())
		}
		visitedNodes[currentNodeID] = true

		node := workflow.GetNodeByID(currentNodeID)
		if node == nil {
			return nil, engine.ErrNodeNotFound().WithDetail("node", currentNodeID)
		}

		// Evaluate expressions in config
		configToEvaluate := make(map[string]any)
		for k, v := range node.Config {
			configToEvaluate[k] = v
		}

		evaluatedData, err := e.expressionEvaluator.Evaluate(ctx, configToEvaluate, nodeContext)
		if err != nil {
			nodeResult := &engine.NodeResult{
				NodeID:    node.ID,
				NodeName:  node.Name,
				Success:   false,
				Error:     fmt.Sprintf("expression evaluation failed: %v", err),
				Timestamp: time.Now(),
			}
			result.Executedodes = append(result.Executedodes, *nodeResult)
			result.Success = false
			result.ErrorMessage = nodeResult.Error
			break
		}

		evaluatedConfig, ok := evaluatedData.(map[string]any)
		if !ok {
			nodeResult := &engine.NodeResult{
				NodeID:    node.ID,
				NodeName:  node.Name,
				Success:   false,
				Error:     "expression evaluation did not return a valid configuration map",
				Timestamp: time.Now(),
			}
			result.Executedodes = append(result.Executedodes, *nodeResult)
			result.Success = false
			result.ErrorMessage = nodeResult.Error
			break
		}

		nodeForExecution := *node
		nodeForExecution.Config = evaluatedConfig

		// Execute node
		nodeResult, err := e.executeNodeInternal(ctx, nodeForExecution, message, session, nodeContext, result)
		if err != nil {
			if nodeResult == nil {
				nodeResult = &engine.NodeResult{
					NodeID: node.ID, NodeName: node.Name, Success: false,
					Error: err.Error(), Timestamp: time.Now(),
				}
			} else if nodeResult.Error == "" {
				nodeResult.Success = false
				nodeResult.Error = err.Error()
			}
		}

		result.Executedodes = append(result.Executedodes, *nodeResult)

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
				result.Context[key] = value
			}
		}

		// Handle response
		if responseText, ok := nodeResult.Output["response"].(string); ok && responseText != "" {
			result.Response = responseText
			result.ShouldRespond = true
		}
		if shouldRespond, ok := nodeResult.Output["should_respond"].(bool); ok {
			result.ShouldRespond = shouldRespond
		}
		if nextState, ok := nodeResult.Output["next_state"].(string); ok && nextState != "" {
			result.NextState = nextState
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
	log.Printf("✅ Workflow resume completed: %s in %v (nodes executed: %d)",
		workflow.Name, duration, len(result.Executedodes))

	return result, nil
}

package workflowexec

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/parser"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// DefaultWorkflowExecutor implementa engine.WorkflowExecutor
type DefaultWorkflowExecutor struct {
	stepExecutors map[engine.StepType]engine.StepExecutor
	parserManager *parser.ParserManager
}

var _ engine.WorkflowExecutor = (*DefaultWorkflowExecutor)(nil)

// NewDefaultWorkflowExecutor crea una nueva instancia del ejecutor de workflows
func NewDefaultWorkflowExecutor(
	parserManager *parser.ParserManager,
	stepExecutors ...engine.StepExecutor,
) *DefaultWorkflowExecutor {
	executor := &DefaultWorkflowExecutor{
		stepExecutors: make(map[engine.StepType]engine.StepExecutor),
		parserManager: parserManager,
	}

	// Registrar todos los ejecutores proporcionados
	for _, stepExec := range stepExecutors {
		executor.RegisterStepExecutor(stepExec)
	}

	return executor
}

// RegisterStepExecutor registra un ejecutor de paso
func (e *DefaultWorkflowExecutor) RegisterStepExecutor(executor engine.StepExecutor) {
	// Registrar para todos los tipos que soporte
	for _, stepType := range []engine.StepType{
		engine.StepTypeCondition,
		engine.StepTypeParser,
		engine.StepTypeTool,
		engine.StepTypeAction,
		engine.StepTypeDelay,
		engine.StepTypeResponse,
	} {
		if executor.SupportsType(stepType) {
			e.stepExecutors[stepType] = executor
			log.Printf("✓ Registered executor for step type: %s", stepType)
		}
	}
}

// Execute ejecuta un workflow completo
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
		ExecutedSteps: []engine.StepResult{},
	}

	// Validar workflow
	if err := e.ValidateWorkflow(ctx, workflow); err != nil {
		return nil, errx.Wrap(err, "workflow validation failed", errx.TypeValidation)
	}

	// Preparar contexto inicial
	stepContext := e.prepareInitialContext(message, session)

	// Ejecutar pasos secuencialmente
	currentStepID := ""
	if len(workflow.Steps) > 0 {
		currentStepID = workflow.Steps[0].ID
	}

	visitedSteps := make(map[string]bool)
	maxSteps := len(workflow.Steps) * 2 // Prevenir ciclos infinitos

	for currentStepID != "" && len(result.ExecutedSteps) < maxSteps {
		// Prevenir ciclos
		if visitedSteps[currentStepID] {
			return nil, engine.ErrCyclicWorkflow().
				WithDetail("step_id", currentStepID).
				WithDetail("workflow_id", workflow.ID.String())
		}
		visitedSteps[currentStepID] = true

		// Buscar el paso actual
		step := workflow.GetStepByID(currentStepID)
		if step == nil {
			return nil, engine.ErrStepNotFound().WithDetail("step_id", currentStepID)
		}

		// Ejecutar paso
		stepResult, err := e.executeStepInternal(ctx, *step, message, session, stepContext, result)
		if err != nil {
			if stepResult == nil {
				stepResult = &engine.StepResult{
					StepID:    step.ID,
					StepName:  step.Name,
					Success:   false,
					Error:     err.Error(),
					Timestamp: time.Now(),
				}
			} else {
				stepResult.Success = false
				stepResult.Error = err.Error()
			}
		}

		result.ExecutedSteps = append(result.ExecutedSteps, *stepResult)

		// Si el paso falló
		if !stepResult.Success {
			result.Success = false
			result.Error = fmt.Errorf("step %s failed: %s", step.Name, stepResult.Error)
			result.ErrorMessage = stepResult.Error

			// Ir al paso de fallo si existe
			if step.OnFailure != "" {
				currentStepID = step.OnFailure
				continue
			}
			break
		}

		// Actualizar contexto con el output del paso
		if stepResult.Output != nil {
			for key, value := range stepResult.Output {
				stepContext[key] = value
				result.Context[key] = value
			}
		}

		// Manejar respuestas
		if responseText, ok := stepResult.Output["response"].(string); ok && responseText != "" {
			result.Response = responseText
			result.ShouldRespond = true
		}
		if shouldRespond, ok := stepResult.Output["should_respond"].(bool); ok {
			result.ShouldRespond = shouldRespond
		}

		// Manejar next_state
		if nextState, ok := stepResult.Output["next_state"].(string); ok && nextState != "" {
			result.NextState = nextState
		}

		// Siguiente paso (puede ser sobreescrito por acciones de parser)
		if nextStepOverride, ok := stepContext["__next_step"].(string); ok {
			currentStepID = nextStepOverride
			delete(stepContext, "__next_step") // Limpiar override
		} else if step.OnSuccess != "" {
			currentStepID = step.OnSuccess
		} else {
			// No hay siguiente paso, terminar
			currentStepID = ""
		}
	}

	duration := time.Since(startTime)
	log.Printf("✅ Workflow execution completed: %s in %v", workflow.Name, duration)

	return result, nil
}

// executeStepInternal ejecuta un paso con integración completa
func (e *DefaultWorkflowExecutor) executeStepInternal(
	ctx context.Context,
	step engine.WorkflowStep,
	message engine.Message,
	session *engine.Session,
	stepContext map[string]any,
	workflowResult *engine.ExecutionResult,
) (*engine.StepResult, error) {
	log.Printf("⚡ Executing step: %s (type: %s)", step.Name, step.Type)
	startTime := time.Now()

	// Apply timeout if configured
	if step.Timeout != nil && *step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*step.Timeout)*time.Second)
		defer cancel()
	}

	// Create result base
	stepResult := &engine.StepResult{
		StepID:    step.ID,
		StepName:  step.Name,
		Success:   true,
		Output:    make(map[string]any),
		Timestamp: startTime,
	}

	var err error

	// Execute according to step type
	switch step.Type {
	case engine.StepTypeParser:
		err = e.executeParserStep(ctx, step, message, session, stepResult, workflowResult, stepContext)
	case engine.StepTypeCondition:
		err = e.executeConditionStep(ctx, step, message, session, stepResult, stepContext)
	case engine.StepTypeTool:
		err = e.executeToolStep(ctx, step, message, session, stepResult, workflowResult, stepContext)
	case engine.StepTypeResponse:
		err = e.executeResponseStep(ctx, step, message, session, stepResult, stepContext)
	case engine.StepTypeDelay:
		err = e.executeDelayStep(ctx, step, stepResult)

	// REMOVE THIS CASE - Let registered executors handle ACTION steps
	// case engine.StepTypeAction:
	//     err = e.executeActionStep(ctx, step, message, session, stepResult, workflowResult, stepContext)

	default:
		// Try with registered executors (THIS WILL NOW HANDLE ACTION TYPE)
		if executor, ok := e.stepExecutors[step.Type]; ok {
			input := e.prepareStepInput(message, session, stepContext)
			stepResult, err = executor.Execute(ctx, step, input)

			// Merge step result output into workflow context
			if err == nil && stepResult.Output != nil {
				for key, value := range stepResult.Output {
					workflowResult.Context[key] = value
					stepContext[key] = value
				}
			}
		} else {
			err = engine.ErrInvalidWorkflowStep().
				WithDetail("step_type", string(step.Type)).
				WithDetail("reason", "no executor found for step type")
		}
	}

	stepResult.Duration = time.Since(startTime).Milliseconds()

	if err != nil {
		stepResult.Success = false
		stepResult.Error = err.Error()
		return stepResult, err
	}

	return stepResult, nil
}

// ============================================================================
// Parser Step Execution
// ============================================================================

func (e *DefaultWorkflowExecutor) executeParserStep(
	ctx context.Context,
	step engine.WorkflowStep,
	msg engine.Message,
	session *engine.Session,
	stepResult *engine.StepResult,
	workflowResult *engine.ExecutionResult,
	stepContext map[string]any,
) error {
	// Get parser ID from config
	parserIDStr, ok := step.Config["parser_id"].(string)
	if !ok {
		return parser.ErrParserIDNotFound().WithDetail("step_id", step.ID)
	}

	parserID := kernel.ParserID(parserIDStr)

	// Execute parser
	parseResult, err := e.parserManager.ExecuteParserWithConfig(
		ctx,
		parserID,
		msg.TenantID,
		msg,
		session,
		step.Config,
	)
	if err != nil {
		return parser.ErrStepExecutionFailed().
			WithDetail("step_id", step.ID).
			WithDetail("parser_id", parserIDStr).
			WithCause(err)
	}

	// Store parse result in step output
	stepResult.Output["parse_result"] = parseResult
	stepResult.Output["confidence"] = parseResult.Confidence
	stepResult.Output["extracted_data"] = parseResult.ExtractedData
	stepResult.Output["parser_success"] = parseResult.Success

	// Merge parser context into workflow context
	if parseResult.Context != nil {
		for k, v := range parseResult.Context {
			workflowResult.Context[k] = v
			stepContext[k] = v
		}
	}

	// Merge extracted data into step context
	if parseResult.ExtractedData != nil {
		for k, v := range parseResult.ExtractedData {
			stepContext[fmt.Sprintf("extracted_%s", k)] = v
		}
	}

	// Handle parser actions
	if parseResult.HasActions() {
		actionsExecuted, err := e.executeParserActions(ctx, parseResult, msg, session, workflowResult, stepContext)
		stepResult.Output["actions_executed"] = actionsExecuted
		if err != nil {
			return parser.ErrActionExecutionFailed().
				WithDetail("step_id", step.ID).
				WithCause(err)
		}
	}

	// Auto-respond if configured and parser has response
	autoRespond, _ := step.Config["auto_respond"].(bool)
	if autoRespond && parseResult.ShouldRespond && parseResult.Response != "" {
		stepResult.Output["response"] = parseResult.Response
		stepResult.Output["should_respond"] = true
	}

	// Check success based on parser result and optional min_confidence
	if !parseResult.Success {
		return parser.ErrParsingFailed().
			WithDetail("reason", parseResult.Error).
			WithDetail("parser_id", parserIDStr)
	}

	// Check min confidence if specified
	if minConf, ok := step.Config["min_confidence"].(float64); ok {
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
	stepContext map[string]any,
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

		case parser.ActionTypeTool:
			if err := e.executeToolAction(ctx, action, msg, session, workflowResult, stepContext); err != nil {
				return executed, parser.ErrToolExecutionFailed().
					WithDetail("action_index", fmt.Sprintf("%d", i)).
					WithCause(err)
			}
			executed = append(executed, string(action.Type))

		case parser.ActionTypeSetContext:
			if key, ok := action.Config["key"].(string); ok {
				if value, ok := action.Config["value"]; ok {
					workflowResult.Context[key] = value
					stepContext[key] = value
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
			if nextStep, ok := action.Config["next_step"].(string); ok {
				// Store for routing logic
				stepContext["__next_step"] = nextStep
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
				stepContext["__trigger_workflow"] = workflowID
				executed = append(executed, string(action.Type))
			}
		}
	}

	return executed, nil
}

func (e *DefaultWorkflowExecutor) executeToolAction(
	ctx context.Context,
	action parser.Action,
	msg engine.Message,
	session *engine.Session,
	workflowResult *engine.ExecutionResult,
	stepContext map[string]any,
) error {
	// if e.toolManager == nil {
	// 	return parser.ErrInvalidToolConfig().WithDetail("reason", "tool manager not configured")
	// }
	//
	// toolID, ok := action.Config["tool_id"].(string)
	// if !ok {
	// 	return parser.ErrInvalidToolConfig().WithDetail("reason", "tool_id not found")
	// }

	// params := action.Config["params"]

	// Execute tool
	// toolResult, err := e.toolManager.ExecuteTool(
	// 	ctx,
	// 	kernel.ToolID(toolID),
	// 	msg.TenantID,
	// 	params,
	// 	session,
	// )
	// if err != nil {
	// 	return err
	// }
	//
	// // Store tool result in workflow context
	// resultKey := fmt.Sprintf("tool_%s_result", toolID)
	// workflowResult.Context[resultKey] = toolResult
	// stepContext[resultKey] = toolResult

	return nil
}

func (e *DefaultWorkflowExecutor) executeWebhookAction(ctx context.Context, action parser.Action) error {
	webhookURL, ok := action.Config["url"].(string)
	if !ok {
		return parser.ErrInvalidActionConfig().WithDetail("reason", "webhook url not found")
	}

	// TODO: Implement actual webhook HTTP call
	log.Printf("📤 Webhook action: %s", webhookURL)

	// For now, just log
	// In production, you'd make an HTTP POST request here
	// using net/http client with the action.Config data

	return nil
}

// ============================================================================
// Other Step Types
// ============================================================================

func (e *DefaultWorkflowExecutor) executeConditionStep(
	ctx context.Context,
	step engine.WorkflowStep,
	msg engine.Message,
	session *engine.Session,
	stepResult *engine.StepResult,
	stepContext map[string]any,
) error {
	// Get condition config
	conditionType, _ := step.Config["type"].(string)
	field, _ := step.Config["field"].(string)
	operator, _ := step.Config["operator"].(string)
	value := step.Config["value"]

	// Evaluate condition based on type
	var actualValue any
	var conditionMet bool

	switch field {
	case "message.text":
		actualValue = msg.Content.Text
	case "message.type":
		actualValue = msg.Content.Type
	case "session.state":
		if session != nil {
			actualValue = session.CurrentState
		}
	default:
		// Check in step context
		if val, ok := stepContext[field]; ok {
			actualValue = val
		}
	}

	// Evaluate operator
	switch operator {
	case "equals":
		conditionMet = fmt.Sprintf("%v", actualValue) == fmt.Sprintf("%v", value)
	case "contains":
		if str, ok := actualValue.(string); ok {
			if valStr, ok := value.(string); ok {
				conditionMet = contains(str, valStr)
			}
		}
	case "gt", "gte", "lt", "lte":
		// Numeric comparisons
		conditionMet = compareNumeric(actualValue, value, operator)
	default:
		return engine.ErrInvalidWorkflowStep().
			WithDetail("reason", "unsupported operator").
			WithDetail("operator", operator)
	}

	stepResult.Output["condition_met"] = conditionMet
	stepResult.Output["condition_type"] = conditionType
	stepResult.Output["actual_value"] = actualValue
	stepResult.Output["expected_value"] = value

	if !conditionMet {
		return fmt.Errorf("condition not met")
	}

	return nil
}

func (e *DefaultWorkflowExecutor) executeToolStep(
	ctx context.Context,
	step engine.WorkflowStep,
	msg engine.Message,
	session *engine.Session,
	stepResult *engine.StepResult,
	workflowResult *engine.ExecutionResult,
	stepContext map[string]any,
) error {
	// if e.toolManager == nil {
	// 	return parser.ErrInvalidToolConfig().WithDetail("reason", "tool manager not configured")
	// }
	//
	// toolID, ok := step.Config["tool_id"].(string)
	// if !ok {
	// 	return parser.ErrInvalidToolConfig().WithDetail("reason", "tool_id not found in config")
	// }
	//
	// params := step.Config["params"]
	//
	// toolResult, err := e.toolManager.ExecuteTool(
	// 	ctx,
	// 	kernel.ToolID(toolID),
	// 	msg.TenantID,
	// 	params,
	// 	session,
	// )
	// if err != nil {
	// 	return parser.ErrToolExecutionFailed().
	// 		WithDetail("tool_id", toolID).
	// 		WithCause(err)
	// }
	//
	// stepResult.Output["tool_result"] = toolResult
	// workflowResult.Context[fmt.Sprintf("tool_%s_result", toolID)] = toolResult
	// stepContext[fmt.Sprintf("tool_%s_result", toolID)] = toolResult

	return nil
}

func (e *DefaultWorkflowExecutor) executeResponseStep(
	ctx context.Context,
	step engine.WorkflowStep,
	msg engine.Message,
	session *engine.Session,
	stepResult *engine.StepResult,
	stepContext map[string]any,
) error {
	message, ok := step.Config["message"].(string)
	if !ok {
		return engine.ErrInvalidWorkflowStep().
			WithDetail("reason", "message not found in config").
			WithDetail("step_id", step.ID)
	}

	// Template replacement (simple version)
	message = e.replaceTemplateVariables(message, stepContext)

	stepResult.Output["response"] = message
	stepResult.Output["should_respond"] = true

	return nil
}

func (e *DefaultWorkflowExecutor) executeDelayStep(
	ctx context.Context,
	step engine.WorkflowStep,
	stepResult *engine.StepResult,
) error {
	duration, ok := step.Config["duration"].(float64)
	if !ok {
		return engine.ErrInvalidWorkflowStep().
			WithDetail("reason", "duration not found in config").
			WithDetail("step_id", step.ID)
	}

	select {
	case <-time.After(time.Duration(duration) * time.Second):
		stepResult.Output["delayed_seconds"] = duration
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *DefaultWorkflowExecutor) executeActionStep(
	ctx context.Context,
	step engine.WorkflowStep,
	msg engine.Message,
	session *engine.Session,
	stepResult *engine.StepResult,
	workflowResult *engine.ExecutionResult,
	stepContext map[string]any,
) error {
	actionType, _ := step.Config["action_type"].(string)

	switch actionType {
	case "set_context":
		key, _ := step.Config["key"].(string)
		value := step.Config["value"]
		if key != "" && value != nil {
			workflowResult.Context[key] = value
			stepContext[key] = value
			if session != nil {
				session.SetContext(key, value)
			}
		}
	case "set_state":
		state, _ := step.Config["state"].(string)
		if state != "" {
			workflowResult.NextState = state
			if session != nil {
				session.UpdateState(state)
			}
		}
	default:
		return engine.ErrInvalidWorkflowStep().
			WithDetail("action_type", actionType).
			WithDetail("reason", "unsupported action type")
	}

	return nil
}

func (e *DefaultWorkflowExecutor) ExecuteStep(
	ctx context.Context,
	step engine.WorkflowStep,
	message engine.Message,
	session *engine.Session,
	stepContext map[string]any,
) (*engine.StepResult, error) {
	workflowResult := &engine.ExecutionResult{
		Context: make(map[string]any),
	}
	return e.executeStepInternal(ctx, step, message, session, stepContext, workflowResult)
}

// ============================================================================
// Validation
// ============================================================================

func (e *DefaultWorkflowExecutor) ValidateWorkflow(ctx context.Context, workflow engine.Workflow) error {
	if !workflow.IsValid() {
		return engine.ErrInvalidWorkflowConfig().WithDetail("reason", "workflow is not valid")
	}

	if len(workflow.Steps) == 0 {
		return engine.ErrInvalidWorkflowConfig().WithDetail("reason", "workflow has no steps")
	}

	// Validar que todos los steps tengan IDs únicos
	stepIDs := make(map[string]bool)
	for _, step := range workflow.Steps {
		if step.ID == "" {
			return engine.ErrInvalidWorkflowStep().WithDetail("reason", "step has no ID")
		}
		if stepIDs[step.ID] {
			return engine.ErrInvalidWorkflowStep().
				WithDetail("step_id", step.ID).
				WithDetail("reason", "duplicate step ID")
		}
		stepIDs[step.ID] = true

		// Validar configuración del paso
		if executor, ok := e.stepExecutors[step.Type]; ok {
			if err := executor.ValidateConfig(step.Config); err != nil {
				return errx.Wrap(err, "step config validation failed", errx.TypeValidation).
					WithDetail("step_id", step.ID).
					WithDetail("step_name", step.Name)
			}
		}
	}

	// Validar referencias de OnSuccess y OnFailure
	for _, step := range workflow.Steps {
		if step.OnSuccess != "" && !stepIDs[step.OnSuccess] {
			return engine.ErrInvalidWorkflowStep().
				WithDetail("step_id", step.ID).
				WithDetail("on_success", step.OnSuccess).
				WithDetail("reason", "on_success references non-existent step")
		}
		if step.OnFailure != "" && !stepIDs[step.OnFailure] {
			return engine.ErrInvalidWorkflowStep().
				WithDetail("step_id", step.ID).
				WithDetail("on_failure", step.OnFailure).
				WithDetail("reason", "on_failure references non-existent step")
		}
	}

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func (e *DefaultWorkflowExecutor) prepareInitialContext(message engine.Message, session *engine.Session) map[string]any {
	context := make(map[string]any)

	// Agregar información del mensaje
	context["message_id"] = message.ID.String()
	context["message_text"] = message.Content.Text
	context["message_type"] = message.Content.Type
	context["sender_id"] = message.SenderID
	context["channel_id"] = message.ChannelID.String()

	// Agregar contexto del mensaje si existe
	if message.Context != nil {
		for key, value := range message.Context {
			context["msg_"+key] = value
		}
	}

	// Agregar información de la sesión si existe
	if session != nil {
		context["session_id"] = session.ID
		context["session_state"] = session.CurrentState

		// Agregar contexto de la sesión
		if session.Context != nil {
			for key, value := range session.Context {
				context["session_"+key] = value
			}
		}
	}

	return context
}

func (e *DefaultWorkflowExecutor) prepareStepInput(
	message engine.Message,
	session *engine.Session,
	stepContext map[string]any,
) map[string]any {
	input := make(map[string]any)

	// Copiar contexto del paso
	for key, value := range stepContext {
		input[key] = value
	}

	return input
}

func (e *DefaultWorkflowExecutor) replaceTemplateVariables(template string, context map[string]any) string {
	// Simple template replacement
	// In production, use a proper template engine like text/template
	result := template
	for key, value := range context {
		placeholder := fmt.Sprintf("{{%s}}", key)
		replacement := fmt.Sprintf("%v", value)
		result = replaceAll(result, placeholder, replacement)
	}
	return result
}

// ============================================================================
// Utility Functions
// ============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func replaceAll(s, old, new string) string {
	if old == new || old == "" {
		return s
	}
	result := ""
	for {
		i := findIndex(s, old)
		if i == -1 {
			return result + s
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
}

func findIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func compareNumeric(a, b any, operator string) bool {
	// Simple numeric comparison
	// In production, use proper type conversion and comparison
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
	default:
		return 0, false
	}
}

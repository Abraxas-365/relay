package engine

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

var ErrRegistry = errx.NewRegistry("ENGINE")

var (
	// Workflow errors
	CodeWorkflowNotFound        = ErrRegistry.Register("WORKFLOW_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Workflow not found")
	CodeWorkflowAlreadyExists   = ErrRegistry.Register("WORKFLOW_ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Workflow already exists")
	CodeInvalidWorkflowConfig   = ErrRegistry.Register("INVALID_WORKFLOW_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Invalid workflow configuration")
	CodeWorkflowInactive        = ErrRegistry.Register("WORKFLOW_INACTIVE", errx.TypeBusiness, http.StatusForbidden, "Workflow is inactive")
	CodeWorkflowExecutionFailed = ErrRegistry.Register("WORKFLOW_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Workflow execution failed")
	CodeInvalidWorkflowNode     = ErrRegistry.Register("INVALID_WORKFLOW_NODE", errx.TypeValidation, http.StatusBadRequest, "Invalid workflow node")
	CodeNodeNotFound            = ErrRegistry.Register("NODE_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Node not found")
	CodeCyclicWorkflow          = ErrRegistry.Register("CYCLIC_WORKFLOW", errx.TypeValidation, http.StatusBadRequest, "Workflow has cycles")

	// Trigger errors
	CodeInvalidTrigger     = ErrRegistry.Register("INVALID_TRIGGER", errx.TypeValidation, http.StatusBadRequest, "Invalid trigger")
	CodeNoMatchingWorkflow = ErrRegistry.Register("NO_MATCHING_WORKFLOW", errx.TypeBusiness, http.StatusNotFound, "No matching workflow found")

	// Execution errors
	CodeExecutionTimeout    = ErrRegistry.Register("EXECUTION_TIMEOUT", errx.TypeInternal, http.StatusRequestTimeout, "Execution timeout")
	CodeNodeExecutionFailed = ErrRegistry.Register("NODE_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Node execution failed")

	// ✅ Schedule errors
	CodeScheduleNotFound        = ErrRegistry.Register("SCHEDULE_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Schedule not found")
	CodeScheduleAlreadyExists   = ErrRegistry.Register("SCHEDULE_ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Schedule already exists")
	CodeInvalidScheduleConfig   = ErrRegistry.Register("INVALID_SCHEDULE_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Invalid schedule configuration")
	CodeInvalidCronExpression   = ErrRegistry.Register("INVALID_CRON_EXPRESSION", errx.TypeValidation, http.StatusBadRequest, "Invalid cron expression")
	CodeInvalidInterval         = ErrRegistry.Register("INVALID_INTERVAL", errx.TypeValidation, http.StatusBadRequest, "Invalid interval")
	CodeScheduleInPast          = ErrRegistry.Register("SCHEDULE_IN_PAST", errx.TypeValidation, http.StatusBadRequest, "Scheduled time is in the past")
	CodeScheduleConflict        = ErrRegistry.Register("SCHEDULE_CONFLICT", errx.TypeConflict, http.StatusConflict, "Schedule conflicts with existing schedule")
	CodeScheduleExecutionFailed = ErrRegistry.Register("SCHEDULE_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Schedule execution failed")
	CodeScheduleNotActive       = ErrRegistry.Register("SCHEDULE_NOT_ACTIVE", errx.TypeBusiness, http.StatusForbidden, "Schedule is not active")
	CodeTooManySchedules        = ErrRegistry.Register("TOO_MANY_SCHEDULES", errx.TypeBusiness, http.StatusTooManyRequests, "Too many schedules for workflow")
)

// ============================================================================
// Workflow Error Constructors
// ============================================================================

func ErrWorkflowNotFound() *errx.Error {
	return ErrRegistry.New(CodeWorkflowNotFound)
}

func ErrWorkflowAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeWorkflowAlreadyExists)
}

func ErrInvalidWorkflowConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidWorkflowConfig)
}

func ErrWorkflowInactive() *errx.Error {
	return ErrRegistry.New(CodeWorkflowInactive)
}

func ErrWorkflowExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeWorkflowExecutionFailed)
}

func ErrInvalidWorkflowNode() *errx.Error {
	return ErrRegistry.New(CodeInvalidWorkflowNode)
}

func ErrNodeNotFound() *errx.Error {
	return ErrRegistry.New(CodeNodeNotFound)
}

func ErrCyclicWorkflow() *errx.Error {
	return ErrRegistry.New(CodeCyclicWorkflow)
}

// ============================================================================
// Trigger Error Constructors
// ============================================================================

func ErrInvalidTrigger() *errx.Error {
	return ErrRegistry.New(CodeInvalidTrigger)
}

func ErrNoMatchingWorkflow() *errx.Error {
	return ErrRegistry.New(CodeNoMatchingWorkflow)
}

// ============================================================================
// Execution Error Constructors
// ============================================================================

func ErrExecutionTimeout() *errx.Error {
	return ErrRegistry.New(CodeExecutionTimeout)
}

func ErrNodeExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeNodeExecutionFailed)
}

// ============================================================================
// ✅ Schedule Error Constructors
// ============================================================================

func ErrScheduleNotFound() *errx.Error {
	return ErrRegistry.New(CodeScheduleNotFound)
}

func ErrScheduleAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeScheduleAlreadyExists)
}

func ErrInvalidScheduleConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidScheduleConfig)
}

func ErrInvalidCronExpression() *errx.Error {
	return ErrRegistry.New(CodeInvalidCronExpression)
}

func ErrInvalidInterval() *errx.Error {
	return ErrRegistry.New(CodeInvalidInterval)
}

func ErrScheduleInPast() *errx.Error {
	return ErrRegistry.New(CodeScheduleInPast)
}

func ErrScheduleConflict() *errx.Error {
	return ErrRegistry.New(CodeScheduleConflict)
}

func ErrScheduleExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeScheduleExecutionFailed)
}

func ErrScheduleNotActive() *errx.Error {
	return ErrRegistry.New(CodeScheduleNotActive)
}

func ErrTooManySchedules() *errx.Error {
	return ErrRegistry.New(CodeTooManySchedules)
}

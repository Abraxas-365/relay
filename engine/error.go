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
)

// Error constructor functions
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

func ErrInvalidTrigger() *errx.Error {
	return ErrRegistry.New(CodeInvalidTrigger)
}

func ErrNoMatchingWorkflow() *errx.Error {
	return ErrRegistry.New(CodeNoMatchingWorkflow)
}

func ErrExecutionTimeout() *errx.Error {
	return ErrRegistry.New(CodeExecutionTimeout)
}

func ErrNodeExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeNodeExecutionFailed)
}


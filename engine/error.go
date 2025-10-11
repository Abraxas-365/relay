package engine

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

// ============================================================================
// Error Registry
// ============================================================================

var ErrRegistry = errx.NewRegistry("ENGINE")

// ============================================================================
// Error Codes
// ============================================================================

var (
	// Message errors
	CodeMessageNotFound         = ErrRegistry.Register("MESSAGE_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Mensaje no encontrado")
	CodeMessageAlreadyExists    = ErrRegistry.Register("MESSAGE_ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Mensaje ya existe")
	CodeInvalidMessageStatus    = ErrRegistry.Register("INVALID_MESSAGE_STATUS", errx.TypeValidation, http.StatusBadRequest, "Estado de mensaje inválido")
	CodeMessageProcessingFailed = ErrRegistry.Register("MESSAGE_PROCESSING_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al procesar mensaje")

	// Workflow errors
	CodeWorkflowNotFound        = ErrRegistry.Register("WORKFLOW_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Workflow no encontrado")
	CodeWorkflowAlreadyExists   = ErrRegistry.Register("WORKFLOW_ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Workflow ya existe")
	CodeInvalidWorkflowConfig   = ErrRegistry.Register("INVALID_WORKFLOW_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Configuración de workflow inválida")
	CodeWorkflowInactive        = ErrRegistry.Register("WORKFLOW_INACTIVE", errx.TypeBusiness, http.StatusForbidden, "Workflow está inactivo")
	CodeWorkflowExecutionFailed = ErrRegistry.Register("WORKFLOW_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Ejecución de workflow falló")
	CodeInvalidWorkflowStep     = ErrRegistry.Register("INVALID_WORKFLOW_STEP", errx.TypeValidation, http.StatusBadRequest, "Paso de workflow inválido")
	CodeStepNotFound            = ErrRegistry.Register("STEP_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Paso no encontrado")
	CodeCyclicWorkflow          = ErrRegistry.Register("CYCLIC_WORKFLOW", errx.TypeValidation, http.StatusBadRequest, "Workflow tiene ciclos")

	// Session errors
	CodeSessionNotFound     = ErrRegistry.Register("SESSION_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Sesión no encontrada")
	CodeSessionExpired      = ErrRegistry.Register("SESSION_EXPIRED", errx.TypeBusiness, http.StatusGone, "Sesión expirada")
	CodeInvalidSessionState = ErrRegistry.Register("INVALID_SESSION_STATE", errx.TypeValidation, http.StatusBadRequest, "Estado de sesión inválido")

	// Trigger errors
	CodeInvalidTrigger     = ErrRegistry.Register("INVALID_TRIGGER", errx.TypeValidation, http.StatusBadRequest, "Trigger inválido")
	CodeNoMatchingWorkflow = ErrRegistry.Register("NO_MATCHING_WORKFLOW", errx.TypeBusiness, http.StatusNotFound, "No hay workflow que coincida con el trigger")

	// Execution errors
	CodeExecutionTimeout    = ErrRegistry.Register("EXECUTION_TIMEOUT", errx.TypeInternal, http.StatusRequestTimeout, "Ejecución excedió timeout")
	CodeStepExecutionFailed = ErrRegistry.Register("STEP_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Ejecución de paso falló")
)

// ============================================================================
// Error Constructor Functions
// ============================================================================

// Message errors
func ErrMessageNotFound() *errx.Error {
	return ErrRegistry.New(CodeMessageNotFound)
}

func ErrMessageAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeMessageAlreadyExists)
}

func ErrInvalidMessageStatus() *errx.Error {
	return ErrRegistry.New(CodeInvalidMessageStatus)
}

func ErrMessageProcessingFailed() *errx.Error {
	return ErrRegistry.New(CodeMessageProcessingFailed)
}

// Workflow errors
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

func ErrInvalidWorkflowStep() *errx.Error {
	return ErrRegistry.New(CodeInvalidWorkflowStep)
}

func ErrStepNotFound() *errx.Error {
	return ErrRegistry.New(CodeStepNotFound)
}

func ErrCyclicWorkflow() *errx.Error {
	return ErrRegistry.New(CodeCyclicWorkflow)
}

// Session errors
func ErrSessionNotFound() *errx.Error {
	return ErrRegistry.New(CodeSessionNotFound)
}

func ErrSessionExpired() *errx.Error {
	return ErrRegistry.New(CodeSessionExpired)
}

func ErrInvalidSessionState() *errx.Error {
	return ErrRegistry.New(CodeInvalidSessionState)
}

// Trigger errors
func ErrInvalidTrigger() *errx.Error {
	return ErrRegistry.New(CodeInvalidTrigger)
}

func ErrNoMatchingWorkflow() *errx.Error {
	return ErrRegistry.New(CodeNoMatchingWorkflow)
}

// Execution errors
func ErrExecutionTimeout() *errx.Error {
	return ErrRegistry.New(CodeExecutionTimeout)
}

func ErrStepExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeStepExecutionFailed)
}

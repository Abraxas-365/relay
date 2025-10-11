package tool

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

// ============================================================================
// Error Registry
// ============================================================================

var ErrRegistry = errx.NewRegistry("TOOL")

// ============================================================================
// Error Codes
// ============================================================================

var (
	// Tool errors
	CodeToolNotFound      = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Tool no encontrado")
	CodeToolAlreadyExists = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Tool ya existe")
	CodeInvalidToolType   = ErrRegistry.Register("INVALID_TYPE", errx.TypeValidation, http.StatusBadRequest, "Tipo de tool inválido")
	CodeInvalidToolConfig = ErrRegistry.Register("INVALID_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Configuración de tool inválida")
	CodeToolInactive      = ErrRegistry.Register("TOOL_INACTIVE", errx.TypeBusiness, http.StatusForbidden, "Tool está inactivo")

	// Execution errors
	CodeExecutionFailed   = ErrRegistry.Register("EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Ejecución de tool falló")
	CodeInvalidInput      = ErrRegistry.Register("INVALID_INPUT", errx.TypeValidation, http.StatusBadRequest, "Input inválido para tool")
	CodeTimeoutExceeded   = ErrRegistry.Register("TIMEOUT_EXCEEDED", errx.TypeInternal, http.StatusRequestTimeout, "Timeout excedido")
	CodeExecutionNotFound = ErrRegistry.Register("EXECUTION_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Ejecución no encontrada")

	// HTTP Tool errors
	CodeHTTPRequestFailed = ErrRegistry.Register("HTTP_REQUEST_FAILED", errx.TypeExternal, http.StatusBadGateway, "HTTP request falló")
	CodeHTTPInvalidURL    = ErrRegistry.Register("HTTP_INVALID_URL", errx.TypeValidation, http.StatusBadRequest, "URL inválida")

	// Database Tool errors
	CodeDatabaseQueryFailed        = ErrRegistry.Register("DATABASE_QUERY_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Query de base de datos falló")
	CodeDatabaseConnectionNotFound = ErrRegistry.Register("DATABASE_CONNECTION_NOT_FOUND", errx.TypeValidation, http.StatusBadRequest, "Conexión de base de datos no encontrada")

	// Email Tool errors
	CodeEmailSendFailed = ErrRegistry.Register("EMAIL_SEND_FAILED", errx.TypeExternal, http.StatusBadGateway, "Envío de email falló")

	// Custom Tool errors
	CodeCustomCodeExecutionFailed = ErrRegistry.Register("CUSTOM_CODE_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Ejecución de código custom falló")
	CodeCustomCodeTimeout         = ErrRegistry.Register("CUSTOM_CODE_TIMEOUT", errx.TypeInternal, http.StatusRequestTimeout, "Código custom excedió timeout")
)

// ============================================================================
// Error Constructor Functions
// ============================================================================

// Tool errors
func ErrToolNotFound() *errx.Error {
	return ErrRegistry.New(CodeToolNotFound)
}

func ErrToolAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeToolAlreadyExists)
}

func ErrInvalidToolType() *errx.Error {
	return ErrRegistry.New(CodeInvalidToolType)
}

func ErrInvalidToolConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidToolConfig)
}

func ErrToolInactive() *errx.Error {
	return ErrRegistry.New(CodeToolInactive)
}

// Execution errors
func ErrExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeExecutionFailed)
}

func ErrInvalidInput() *errx.Error {
	return ErrRegistry.New(CodeInvalidInput)
}

func ErrTimeoutExceeded() *errx.Error {
	return ErrRegistry.New(CodeTimeoutExceeded)
}

func ErrExecutionNotFound() *errx.Error {
	return ErrRegistry.New(CodeExecutionNotFound)
}

// HTTP Tool errors
func ErrHTTPRequestFailed() *errx.Error {
	return ErrRegistry.New(CodeHTTPRequestFailed)
}

func ErrHTTPInvalidURL() *errx.Error {
	return ErrRegistry.New(CodeHTTPInvalidURL)
}

// Database Tool errors
func ErrDatabaseQueryFailed() *errx.Error {
	return ErrRegistry.New(CodeDatabaseQueryFailed)
}

func ErrDatabaseConnectionNotFound() *errx.Error {
	return ErrRegistry.New(CodeDatabaseConnectionNotFound)
}

// Email Tool errors
func ErrEmailSendFailed() *errx.Error {
	return ErrRegistry.New(CodeEmailSendFailed)
}

// Custom Tool errors
func ErrCustomCodeExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeCustomCodeExecutionFailed)
}

func ErrCustomCodeTimeout() *errx.Error {
	return ErrRegistry.New(CodeCustomCodeTimeout)
}

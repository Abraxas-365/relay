package channels

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

// ============================================================================
// Error Registry
// ============================================================================

var ErrRegistry = errx.NewRegistry("CHANNEL")

// ============================================================================
// Error Codes
// ============================================================================

var (
	// Channel errors
	CodeChannelNotFound      = ErrRegistry.Register("NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Canal no encontrado")
	CodeChannelAlreadyExists = ErrRegistry.Register("ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Canal ya existe")
	CodeInvalidChannelType   = ErrRegistry.Register("INVALID_TYPE", errx.TypeValidation, http.StatusBadRequest, "Tipo de canal inválido")
	CodeInvalidChannelConfig = ErrRegistry.Register("INVALID_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Configuración de canal inválida")
	CodeChannelInactive      = ErrRegistry.Register("CHANNEL_INACTIVE", errx.TypeBusiness, http.StatusForbidden, "Canal está inactivo")
	CodeChannelNotSupported  = ErrRegistry.Register("NOT_SUPPORTED", errx.TypeValidation, http.StatusBadRequest, "Tipo de canal no soportado")

	// Message sending errors
	CodeMessageSendFailed    = ErrRegistry.Register("MESSAGE_SEND_FAILED", errx.TypeExternal, http.StatusBadGateway, "Envío de mensaje falló")
	CodeInvalidRecipient     = ErrRegistry.Register("INVALID_RECIPIENT", errx.TypeValidation, http.StatusBadRequest, "Destinatario inválido")
	CodeInvalidMessageFormat = ErrRegistry.Register("INVALID_MESSAGE_FORMAT", errx.TypeValidation, http.StatusBadRequest, "Formato de mensaje inválido")
	CodeAttachmentTooLarge   = ErrRegistry.Register("ATTACHMENT_TOO_LARGE", errx.TypeValidation, http.StatusRequestEntityTooLarge, "Archivo adjunto muy grande")
	CodeUnsupportedMediaType = ErrRegistry.Register("UNSUPPORTED_MEDIA_TYPE", errx.TypeValidation, http.StatusUnsupportedMediaType, "Tipo de medio no soportado")

	// Provider errors
	CodeProviderNotConfigured = ErrRegistry.Register("PROVIDER_NOT_CONFIGURED", errx.TypeValidation, http.StatusBadRequest, "Proveedor no configurado")
	CodeProviderAuthFailed    = ErrRegistry.Register("PROVIDER_AUTH_FAILED", errx.TypeExternal, http.StatusUnauthorized, "Autenticación con proveedor falló")
	CodeProviderAPIError      = ErrRegistry.Register("PROVIDER_API_ERROR", errx.TypeExternal, http.StatusBadGateway, "Error en API del proveedor")
	CodeProviderRateLimited   = ErrRegistry.Register("PROVIDER_RATE_LIMITED", errx.TypeExternal, http.StatusTooManyRequests, "Proveedor limitó la tasa de requests")

	// Webhook errors
	CodeInvalidWebhookSignature = ErrRegistry.Register("INVALID_WEBHOOK_SIGNATURE", errx.TypeValidation, http.StatusUnauthorized, "Firma de webhook inválida")
	CodeWebhookProcessingFailed = ErrRegistry.Register("WEBHOOK_PROCESSING_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Procesamiento de webhook falló")

	// Feature errors
	CodeFeatureNotSupported = ErrRegistry.Register("FEATURE_NOT_SUPPORTED", errx.TypeBusiness, http.StatusNotImplemented, "Característica no soportada por el canal")
)

// ============================================================================
// Error Constructor Functions
// ============================================================================

// Channel errors
func ErrChannelNotFound() *errx.Error {
	return ErrRegistry.New(CodeChannelNotFound)
}

func ErrChannelAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeChannelAlreadyExists)
}

func ErrInvalidChannelType() *errx.Error {
	return ErrRegistry.New(CodeInvalidChannelType)
}

func ErrInvalidChannelConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidChannelConfig)
}

func ErrChannelInactive() *errx.Error {
	return ErrRegistry.New(CodeChannelInactive)
}

func ErrChannelNotSupported() *errx.Error {
	return ErrRegistry.New(CodeChannelNotSupported)
}

// Message sending errors
func ErrMessageSendFailed() *errx.Error {
	return ErrRegistry.New(CodeMessageSendFailed)
}

func ErrInvalidRecipient() *errx.Error {
	return ErrRegistry.New(CodeInvalidRecipient)
}

func ErrInvalidMessageFormat() *errx.Error {
	return ErrRegistry.New(CodeInvalidMessageFormat)
}

func ErrAttachmentTooLarge() *errx.Error {
	return ErrRegistry.New(CodeAttachmentTooLarge)
}

func ErrUnsupportedMediaType() *errx.Error {
	return ErrRegistry.New(CodeUnsupportedMediaType)
}

// Provider errors
func ErrProviderNotConfigured() *errx.Error {
	return ErrRegistry.New(CodeProviderNotConfigured)
}

func ErrProviderAuthFailed() *errx.Error {
	return ErrRegistry.New(CodeProviderAuthFailed)
}

func ErrProviderAPIError() *errx.Error {
	return ErrRegistry.New(CodeProviderAPIError)
}

func ErrProviderRateLimited() *errx.Error {
	return ErrRegistry.New(CodeProviderRateLimited)
}

// Webhook errors
func ErrInvalidWebhookSignature() *errx.Error {
	return ErrRegistry.New(CodeInvalidWebhookSignature)
}

func ErrWebhookProcessingFailed() *errx.Error {
	return ErrRegistry.New(CodeWebhookProcessingFailed)
}

// Feature errors
func ErrFeatureNotSupported() *errx.Error {
	return ErrRegistry.New(CodeFeatureNotSupported)
}

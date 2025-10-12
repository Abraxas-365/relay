package parser

import (
	"net/http"

	"github.com/Abraxas-365/craftable/errx"
)

// ============================================================================
// Error Registry
// ============================================================================

var ErrRegistry = errx.NewRegistry("PARSER")

// ============================================================================
// Error Codes - Parser
// ============================================================================

var (
	CodeParserNotFound         = ErrRegistry.Register("PARSER_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Parser no encontrado")
	CodeParserAlreadyExists    = ErrRegistry.Register("PARSER_ALREADY_EXISTS", errx.TypeConflict, http.StatusConflict, "Parser ya existe")
	CodeInvalidParserType      = ErrRegistry.Register("INVALID_PARSER_TYPE", errx.TypeValidation, http.StatusBadRequest, "Tipo de parser inválido")
	CodeInvalidParserConfig    = ErrRegistry.Register("INVALID_PARSER_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Configuración de parser inválida")
	CodeParserInactive         = ErrRegistry.Register("PARSER_INACTIVE", errx.TypeBusiness, http.StatusForbidden, "Parser está inactivo")
	CodeParserNotSupported     = ErrRegistry.Register("PARSER_NOT_SUPPORTED", errx.TypeValidation, http.StatusBadRequest, "Tipo de parser no soportado")
	CodeParserTypeNotSupported = ErrRegistry.Register("PARSER_TYPE_NOT_SUPPORTED", errx.TypeValidation, http.StatusBadRequest, "Tipo de parser no soportado")
)

// ============================================================================
// Error Codes - Parsing
// ============================================================================

var (
	CodeParsingFailed   = ErrRegistry.Register("PARSING_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al parsear mensaje")
	CodeNoMatchFound    = ErrRegistry.Register("NO_MATCH_FOUND", errx.TypeBusiness, http.StatusNotFound, "No se encontró coincidencia")
	CodeInvalidInput    = ErrRegistry.Register("INVALID_INPUT", errx.TypeValidation, http.StatusBadRequest, "Input inválido para parser")
	CodeParsingTimeout  = ErrRegistry.Register("PARSING_TIMEOUT", errx.TypeInternal, http.StatusRequestTimeout, "Parsing excedió timeout")
	CodeLowConfidence   = ErrRegistry.Register("LOW_CONFIDENCE", errx.TypeBusiness, http.StatusPartialContent, "Confianza de parsing baja")
	CodeAmbiguousResult = ErrRegistry.Register("AMBIGUOUS_RESULT", errx.TypeBusiness, http.StatusMultipleChoices, "Resultado ambiguo")
)

// ============================================================================
// Error Codes - Regex Parser
// ============================================================================

var (
	CodeInvalidRegexPattern = ErrRegistry.Register("INVALID_REGEX_PATTERN", errx.TypeValidation, http.StatusBadRequest, "Patrón regex inválido")
	CodeRegexCompileFailed  = ErrRegistry.Register("REGEX_COMPILE_FAILED", errx.TypeValidation, http.StatusBadRequest, "Fallo al compilar regex")
	CodeInvalidCaptureGroup = ErrRegistry.Register("INVALID_CAPTURE_GROUP", errx.TypeValidation, http.StatusBadRequest, "Grupo de captura inválido")
	CodeNoPatternsDefined   = ErrRegistry.Register("NO_PATTERNS_DEFINED", errx.TypeValidation, http.StatusBadRequest, "No hay patrones definidos")
)

// ============================================================================
// Error Codes - AI Parser
// ============================================================================

var (
	CodeAIProviderNotConfigured = ErrRegistry.Register("AI_PROVIDER_NOT_CONFIGURED", errx.TypeValidation, http.StatusBadRequest, "Proveedor AI no configurado")
	CodeAIRequestFailed         = ErrRegistry.Register("AI_REQUEST_FAILED", errx.TypeExternal, http.StatusBadGateway, "Request a AI falló")
	CodeAIInvalidResponse       = ErrRegistry.Register("AI_INVALID_RESPONSE", errx.TypeExternal, http.StatusBadGateway, "Respuesta de AI inválida")
	CodeAIQuotaExceeded         = ErrRegistry.Register("AI_QUOTA_EXCEEDED", errx.TypeExternal, http.StatusTooManyRequests, "Cuota de AI excedida")
	CodeInvalidPrompt           = ErrRegistry.Register("INVALID_PROMPT", errx.TypeValidation, http.StatusBadRequest, "Prompt inválido")
	CodeInvalidModel            = ErrRegistry.Register("INVALID_MODEL", errx.TypeValidation, http.StatusBadRequest, "Modelo de AI inválido")
)

// ============================================================================
// Error Codes - Rule Parser
// ============================================================================

var (
	CodeInvalidRule         = ErrRegistry.Register("INVALID_RULE", errx.TypeValidation, http.StatusBadRequest, "Regla inválida")
	CodeInvalidCondition    = ErrRegistry.Register("INVALID_CONDITION", errx.TypeValidation, http.StatusBadRequest, "Condición inválida")
	CodeConditionEvalFailed = ErrRegistry.Register("CONDITION_EVAL_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Evaluación de condición falló")
	CodeNoRulesDefined      = ErrRegistry.Register("NO_RULES_DEFINED", errx.TypeValidation, http.StatusBadRequest, "No hay reglas definidas")
	CodeInvalidRuleOperator = ErrRegistry.Register("INVALID_RULE_OPERATOR", errx.TypeValidation, http.StatusBadRequest, "Operador de regla inválido")
)

// ============================================================================
// Error Codes - Keyword Parser
// ============================================================================

var (
	CodeNoKeywordsDefined  = ErrRegistry.Register("NO_KEYWORDS_DEFINED", errx.TypeValidation, http.StatusBadRequest, "No hay keywords definidos")
	CodeInvalidKeyword     = ErrRegistry.Register("INVALID_KEYWORD", errx.TypeValidation, http.StatusBadRequest, "Keyword inválido")
	CodeKeywordMatchFailed = ErrRegistry.Register("KEYWORD_MATCH_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al buscar keywords")
)

// ============================================================================
// Error Codes - NLP Parser
// ============================================================================

var (
	CodeNLPModelNotFound       = ErrRegistry.Register("NLP_MODEL_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Modelo NLP no encontrado")
	CodeNLPModelLoadFailed     = ErrRegistry.Register("NLP_MODEL_LOAD_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al cargar modelo NLP")
	CodeIntentNotRecognized    = ErrRegistry.Register("INTENT_NOT_RECOGNIZED", errx.TypeBusiness, http.StatusNotFound, "Intención no reconocida")
	CodeEntityExtractionFailed = ErrRegistry.Register("ENTITY_EXTRACTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al extraer entidades")
	CodeInvalidIntent          = ErrRegistry.Register("INVALID_INTENT", errx.TypeValidation, http.StatusBadRequest, "Intención inválida")
	CodeInvalidEntity          = ErrRegistry.Register("INVALID_ENTITY", errx.TypeValidation, http.StatusBadRequest, "Entidad inválida")
)

// ============================================================================
// Error Codes - Actions
// ============================================================================

var (
	CodeInvalidAction         = ErrRegistry.Register("INVALID_ACTION", errx.TypeValidation, http.StatusBadRequest, "Acción inválida")
	CodeActionExecutionFailed = ErrRegistry.Register("ACTION_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Ejecución de acción falló")
	CodeInvalidActionConfig   = ErrRegistry.Register("INVALID_ACTION_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Configuración de acción inválida")
	CodeNoActionsDefined      = ErrRegistry.Register("NO_ACTIONS_DEFINED", errx.TypeValidation, http.StatusBadRequest, "No hay acciones definidas")
)

// ============================================================================
// Error Codes - Selection
// ============================================================================

var (
	CodeNoParserAvailable     = ErrRegistry.Register("NO_PARSER_AVAILABLE", errx.TypeBusiness, http.StatusNotFound, "No hay parser disponible")
	CodeParserSelectionFailed = ErrRegistry.Register("PARSER_SELECTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al seleccionar parser")
	CodeMaxRetriesExceeded    = ErrRegistry.Register("MAX_RETRIES_EXCEEDED", errx.TypeInternal, http.StatusInternalServerError, "Máximo de reintentos excedido")
)

// ============================================================================
// Error Codes - Cache
// ============================================================================

var (
	CodeCacheReadFailed  = ErrRegistry.Register("CACHE_READ_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al leer cache")
	CodeCacheWriteFailed = ErrRegistry.Register("CACHE_WRITE_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al escribir cache")
	CodeCacheClearFailed = ErrRegistry.Register("CACHE_CLEAR_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al limpiar cache")
)

// ============================================================================
// Error Codes - Parser Engine
// ============================================================================

var (
	CodeParserEngineNotFound      = ErrRegistry.Register("PARSER_ENGINE_NOT_FOUND", errx.TypeNotFound, http.StatusNotFound, "Motor de parser no encontrado")
	CodeParserEngineNotRegistered = ErrRegistry.Register("PARSER_ENGINE_NOT_REGISTERED", errx.TypeValidation, http.StatusBadRequest, "Motor de parser no registrado")
	CodeInvalidConfigType         = ErrRegistry.Register("INVALID_CONFIG_TYPE", errx.TypeValidation, http.StatusBadRequest, "Tipo de configuración inválido")
)

// ============================================================================
// Error Codes - Workflow Integration
// ============================================================================

var (
	CodeStepExecutionFailed    = ErrRegistry.Register("STEP_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al ejecutar paso")
	CodeInvalidStepConfig      = ErrRegistry.Register("INVALID_STEP_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Configuración de paso inválida")
	CodeParserIDNotFound       = ErrRegistry.Register("PARSER_ID_NOT_FOUND", errx.TypeValidation, http.StatusBadRequest, "ID de parser no encontrado en configuración")
	CodeToolExecutionFailed    = ErrRegistry.Register("TOOL_EXECUTION_FAILED", errx.TypeInternal, http.StatusInternalServerError, "Fallo al ejecutar herramienta")
	CodeWebhookExecutionFailed = ErrRegistry.Register("WEBHOOK_EXECUTION_FAILED", errx.TypeExternal, http.StatusBadGateway, "Fallo al ejecutar webhook")
	CodeInvalidToolConfig      = ErrRegistry.Register("INVALID_TOOL_CONFIG", errx.TypeValidation, http.StatusBadRequest, "Configuración de herramienta inválida")
)

// ============================================================================
// Error Constructor Functions - Parser Engine
// ============================================================================

func ErrParserEngineNotFound() *errx.Error {
	return ErrRegistry.New(CodeParserEngineNotFound)
}

func ErrParserEngineNotRegistered() *errx.Error {
	return ErrRegistry.New(CodeParserEngineNotRegistered)
}

func ErrInvalidConfigType() *errx.Error {
	return ErrRegistry.New(CodeInvalidConfigType)
}

// ============================================================================
// Error Constructor Functions - Workflow Integration
// ============================================================================

func ErrStepExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeStepExecutionFailed)
}

func ErrInvalidStepConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidStepConfig)
}

func ErrParserIDNotFound() *errx.Error {
	return ErrRegistry.New(CodeParserIDNotFound)
}

func ErrToolExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeToolExecutionFailed)
}

func ErrWebhookExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeWebhookExecutionFailed)
}

func ErrInvalidToolConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidToolConfig)
}

// ============================================================================
// Error Constructor Functions - Parser
// ============================================================================

func ErrParserNotFound() *errx.Error {
	return ErrRegistry.New(CodeParserNotFound)
}

func ErrParserAlreadyExists() *errx.Error {
	return ErrRegistry.New(CodeParserAlreadyExists)
}

func ErrInvalidParserType() *errx.Error {
	return ErrRegistry.New(CodeInvalidParserType)
}

func ErrInvalidParserConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidParserConfig)
}

func ErrParserInactive() *errx.Error {
	return ErrRegistry.New(CodeParserInactive)
}

func ErrParserNotSupported() *errx.Error {
	return ErrRegistry.New(CodeParserNotSupported)
}

// ============================================================================
// Error Constructor Functions - Parsing
// ============================================================================

func ErrParsingFailed() *errx.Error {
	return ErrRegistry.New(CodeParsingFailed)
}

func ErrNoMatchFound() *errx.Error {
	return ErrRegistry.New(CodeNoMatchFound)
}

func ErrInvalidInput() *errx.Error {
	return ErrRegistry.New(CodeInvalidInput)
}

func ErrParsingTimeout() *errx.Error {
	return ErrRegistry.New(CodeParsingTimeout)
}

func ErrLowConfidence() *errx.Error {
	return ErrRegistry.New(CodeLowConfidence)
}

func ErrAmbiguousResult() *errx.Error {
	return ErrRegistry.New(CodeAmbiguousResult)
}

// ============================================================================
// Error Constructor Functions - Regex Parser
// ============================================================================

func ErrInvalidRegexPattern() *errx.Error {
	return ErrRegistry.New(CodeInvalidRegexPattern)
}

func ErrRegexCompileFailed() *errx.Error {
	return ErrRegistry.New(CodeRegexCompileFailed)
}

func ErrInvalidCaptureGroup() *errx.Error {
	return ErrRegistry.New(CodeInvalidCaptureGroup)
}

func ErrNoPatternsDefined() *errx.Error {
	return ErrRegistry.New(CodeNoPatternsDefined)
}

// ============================================================================
// Error Constructor Functions - AI Parser
// ============================================================================

func ErrAIProviderNotConfigured() *errx.Error {
	return ErrRegistry.New(CodeAIProviderNotConfigured)
}

func ErrAIRequestFailed() *errx.Error {
	return ErrRegistry.New(CodeAIRequestFailed)
}

func ErrAIInvalidResponse() *errx.Error {
	return ErrRegistry.New(CodeAIInvalidResponse)
}

func ErrAIQuotaExceeded() *errx.Error {
	return ErrRegistry.New(CodeAIQuotaExceeded)
}

func ErrInvalidPrompt() *errx.Error {
	return ErrRegistry.New(CodeInvalidPrompt)
}

func ErrInvalidModel() *errx.Error {
	return ErrRegistry.New(CodeInvalidModel)
}

// ============================================================================
// Error Constructor Functions - Rule Parser
// ============================================================================

func ErrInvalidRule() *errx.Error {
	return ErrRegistry.New(CodeInvalidRule)
}

func ErrInvalidCondition() *errx.Error {
	return ErrRegistry.New(CodeInvalidCondition)
}

func ErrConditionEvalFailed() *errx.Error {
	return ErrRegistry.New(CodeConditionEvalFailed)
}

func ErrNoRulesDefined() *errx.Error {
	return ErrRegistry.New(CodeNoRulesDefined)
}

func ErrInvalidRuleOperator() *errx.Error {
	return ErrRegistry.New(CodeInvalidRuleOperator)
}

// ============================================================================
// Error Constructor Functions - Keyword Parser
// ============================================================================

func ErrNoKeywordsDefined() *errx.Error {
	return ErrRegistry.New(CodeNoKeywordsDefined)
}

func ErrInvalidKeyword() *errx.Error {
	return ErrRegistry.New(CodeInvalidKeyword)
}

func ErrKeywordMatchFailed() *errx.Error {
	return ErrRegistry.New(CodeKeywordMatchFailed)
}

// ============================================================================
// Error Constructor Functions - NLP Parser
// ============================================================================

func ErrNLPModelNotFound() *errx.Error {
	return ErrRegistry.New(CodeNLPModelNotFound)
}

func ErrNLPModelLoadFailed() *errx.Error {
	return ErrRegistry.New(CodeNLPModelLoadFailed)
}

func ErrIntentNotRecognized() *errx.Error {
	return ErrRegistry.New(CodeIntentNotRecognized)
}

func ErrEntityExtractionFailed() *errx.Error {
	return ErrRegistry.New(CodeEntityExtractionFailed)
}

func ErrInvalidIntent() *errx.Error {
	return ErrRegistry.New(CodeInvalidIntent)
}

func ErrInvalidEntity() *errx.Error {
	return ErrRegistry.New(CodeInvalidEntity)
}

// ============================================================================
// Error Constructor Functions - Actions
// ============================================================================

func ErrInvalidAction() *errx.Error {
	return ErrRegistry.New(CodeInvalidAction)
}

func ErrActionExecutionFailed() *errx.Error {
	return ErrRegistry.New(CodeActionExecutionFailed)
}

func ErrInvalidActionConfig() *errx.Error {
	return ErrRegistry.New(CodeInvalidActionConfig)
}

func ErrNoActionsDefined() *errx.Error {
	return ErrRegistry.New(CodeNoActionsDefined)
}

// ============================================================================
// Error Constructor Functions - Selection
// ============================================================================

func ErrNoParserAvailable() *errx.Error {
	return ErrRegistry.New(CodeNoParserAvailable)
}

func ErrParserSelectionFailed() *errx.Error {
	return ErrRegistry.New(CodeParserSelectionFailed)
}

func ErrMaxRetriesExceeded() *errx.Error {
	return ErrRegistry.New(CodeMaxRetriesExceeded)
}

// ============================================================================
// Error Constructor Functions - Cache
// ============================================================================

func ErrCacheReadFailed() *errx.Error {
	return ErrRegistry.New(CodeCacheReadFailed)
}

func ErrCacheWriteFailed() *errx.Error {
	return ErrRegistry.New(CodeCacheWriteFailed)
}

func ErrCacheClearFailed() *errx.Error {
	return ErrRegistry.New(CodeCacheClearFailed)
}

func ErrParserTypeNotSupported() *errx.Error {
	return ErrRegistry.New(CodeParserTypeNotSupported)
}

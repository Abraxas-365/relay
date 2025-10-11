package parser

import (
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// CreateParserRequest request para crear un parser
type CreateParserRequest struct {
	TenantID    kernel.TenantID `json:"tenant_id" validate:"required"`
	Name        string          `json:"name" validate:"required,min=2"`
	Description string          `json:"description"`
	Type        ParserType      `json:"type" validate:"required"`
	Config      ParserConfig    `json:"config" validate:"required"`
	Priority    int             `json:"priority"`
}

// UpdateParserRequest request para actualizar un parser
type UpdateParserRequest struct {
	Name        *string       `json:"name,omitempty"`
	Description *string       `json:"description,omitempty"`
	Config      *ParserConfig `json:"config,omitempty"`
	Priority    *int          `json:"priority,omitempty"`
	IsActive    *bool         `json:"is_active,omitempty"`
}

// ParseMessageRequest request para parsear un mensaje
type ParseMessageRequest struct {
	ParserID  *kernel.ParserID `json:"parser_id,omitempty"` // Si es nil, usa selector
	Message   engine.Message   `json:"message" validate:"required"`
	SessionID *string          `json:"session_id,omitempty"`
}

// ListParsersRequest request para listar parsers
type ListParsersRequest struct {
	storex.PaginationOptions

	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Type     *ParserType     `json:"type,omitempty"`
	IsActive *bool           `json:"is_active,omitempty"`
	Search   string          `json:"search,omitempty"`
}

// ParserListResponse lista paginada de parsers
type ParserListResponse = storex.Paginated[Parser]

// ParserResponse respuesta con parser
type ParserResponse struct {
	Parser Parser `json:"parser"`
}

// ParseResultResponse respuesta de parsing
type ParseResultResponse struct {
	Result ParseResult `json:"result"`
}

// ParserStatsResponse estadísticas de parser
type ParserStatsResponse struct {
	ParserID       kernel.ParserID `json:"parser_id"`
	ParserName     string          `json:"parser_name"`
	TotalParses    int             `json:"total_parses"`
	SuccessCount   int             `json:"success_count"`
	FailureCount   int             `json:"failure_count"`
	AvgConfidence  float64         `json:"avg_confidence"`
	AvgProcessTime float64         `json:"avg_process_time_ms"`
	LastUsedAt     *string         `json:"last_used_at,omitempty"`
}

// BulkParserOperationRequest request para operaciones masivas
type BulkParserOperationRequest struct {
	TenantID  kernel.TenantID   `json:"tenant_id" validate:"required"`
	ParserIDs []kernel.ParserID `json:"parser_ids" validate:"required,min=1"`
	Operation string            `json:"operation" validate:"required,oneof=activate deactivate delete"`
}

// BulkParserOperationResponse respuesta de operación masiva
type BulkParserOperationResponse struct {
	Successful []kernel.ParserID          `json:"successful"`
	Failed     map[kernel.ParserID]string `json:"failed"`
	Total      int                        `json:"total"`
}

// ValidateParserRequest request para validar parser
type ValidateParserRequest struct {
	Type   ParserType   `json:"type" validate:"required"`
	Config ParserConfig `json:"config" validate:"required"`
}

// ValidateParserResponse respuesta de validación
type ValidateParserResponse struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ParserDetailsDTO DTO simplificado de parser
type ParserDetailsDTO struct {
	ID       kernel.ParserID `json:"id"`
	Name     string          `json:"name"`
	Type     ParserType      `json:"type"`
	Priority int             `json:"priority"`
	IsActive bool            `json:"is_active"`
}

// ToDTO convierte Parser a ParserDetailsDTO
func (p *Parser) ToDTO() ParserDetailsDTO {
	return ParserDetailsDTO{
		ID:       p.ID,
		Name:     p.Name,
		Type:     p.Type,
		Priority: p.Priority,
		IsActive: p.IsActive,
	}
}

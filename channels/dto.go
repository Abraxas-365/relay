package channels

import (
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Message DTOs
// ============================================================================

// OutgoingMessage mensaje saliente a enviar por el canal
type OutgoingMessage struct {
	RecipientID string            `json:"recipient_id" validate:"required"`
	Content     MessageContent    `json:"content" validate:"required"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	ReplyToID   string            `json:"reply_to_id,omitempty"`
	TemplateID  string            `json:"template_id,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
}

// IncomingMessage mensaje entrante recibido del canal
type IncomingMessage struct {
	MessageID  string           `json:"message_id"`
	ChannelID  kernel.ChannelID `json:"channel_id"`
	SenderID   string           `json:"sender_id"`
	Content    MessageContent   `json:"content"`
	Timestamp  int64            `json:"timestamp"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
	RawPayload map[string]any   `json:"raw_payload,omitempty"`
}

// MessageContent contenido del mensaje
type MessageContent struct {
	Type        string         `json:"type"` // text, image, audio, video, document, location, contact
	Text        string         `json:"text,omitempty"`
	MediaURL    string         `json:"media_url,omitempty"`
	Caption     string         `json:"caption,omitempty"`
	MimeType    string         `json:"mime_type,omitempty"`
	Filename    string         `json:"filename,omitempty"`
	Attachments []Attachment   `json:"attachments,omitempty"`
	Location    *Location      `json:"location,omitempty"`
	Contact     *Contact       `json:"contact,omitempty"`
	Interactive *Interactive   `json:"interactive,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Attachment archivo adjunto
type Attachment struct {
	Type     string `json:"type"` // image, audio, video, document
	URL      string `json:"url"`
	MimeType string `json:"mime_type,omitempty"`
	Filename string `json:"filename,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

// Location ubicación geográfica
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

// Contact contacto compartido
type Contact struct {
	Name         string `json:"name"`
	PhoneNumber  string `json:"phone_number,omitempty"`
	Email        string `json:"email,omitempty"`
	Organization string `json:"organization,omitempty"`
}

// Interactive mensaje interactivo (botones, listas, etc)
type Interactive struct {
	Type    string   `json:"type"` // button, list, template
	Header  string   `json:"header,omitempty"`
	Body    string   `json:"body"`
	Footer  string   `json:"footer,omitempty"`
	Buttons []Button `json:"buttons,omitempty"`
	Items   []Item   `json:"items,omitempty"`
}

// Button botón interactivo
type Button struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Type  string `json:"type,omitempty"` // reply, url, call
	URL   string `json:"url,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// Item elemento de lista
type Item struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ============================================================================
// Request DTOs
// ============================================================================

// CreateChannelRequest request para crear un canal
type CreateChannelRequest struct {
	TenantID    kernel.TenantID `json:"tenant_id" validate:"required"`
	Name        string          `json:"name" validate:"required,min=2"`
	Description string          `json:"description"`
	Type        ChannelType     `json:"type" validate:"required"`
	Config      ChannelConfig   `json:"config" validate:"required"`
}

// UpdateChannelRequest request para actualizar un canal
type UpdateChannelRequest struct {
	Name        *string        `json:"name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Config      *ChannelConfig `json:"config,omitempty"`
	IsActive    *bool          `json:"is_active,omitempty"`
}

// SendMessageRequest request para enviar mensaje
type SendMessageRequest struct {
	ChannelID   kernel.ChannelID  `json:"channel_id" validate:"required"`
	RecipientID string            `json:"recipient_id" validate:"required"`
	Content     MessageContent    `json:"content" validate:"required"`
	ReplyToID   string            `json:"reply_to_id,omitempty"`
	TemplateID  string            `json:"template_id,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
}

// TestChannelRequest request para probar un canal
type TestChannelRequest struct {
	ChannelID   kernel.ChannelID `json:"channel_id" validate:"required"`
	RecipientID string           `json:"recipient_id" validate:"required"`
	Message     string           `json:"message"`
}

// ProcessWebhookRequest request para procesar webhook
type ProcessWebhookRequest struct {
	ChannelID kernel.ChannelID  `json:"channel_id" validate:"required"`
	Payload   map[string]any    `json:"payload" validate:"required"`
	Headers   map[string]string `json:"headers,omitempty"`
	Signature string            `json:"signature,omitempty"`
}

// ============================================================================
// List Request DTOs
// ============================================================================

// ListChannelsRequest request para listar canales
type ListChannelsRequest struct {
	storex.PaginationOptions

	TenantID kernel.TenantID `json:"tenant_id" validate:"required"`
	Type     *ChannelType    `json:"type,omitempty"`
	IsActive *bool           `json:"is_active,omitempty"`
	Provider *string         `json:"provider,omitempty"`
	Search   string          `json:"search,omitempty"`
}

// ============================================================================
// Response DTOs
// ============================================================================

// ChannelResponse respuesta con canal
type ChannelResponse struct {
	Channel  Channel         `json:"channel"`
	Features ChannelFeatures `json:"features"`
	Stats    *ChannelStats   `json:"stats,omitempty"`
}

// ChannelListResponse lista paginada de canales
type ChannelListResponse = storex.Paginated[Channel]

// SendMessageResponse respuesta de envío de mensaje
type SendMessageResponse struct {
	Success       bool           `json:"success"`
	MessageID     string         `json:"message_id,omitempty"`
	ProviderMsgID string         `json:"provider_message_id,omitempty"`
	Timestamp     int64          `json:"timestamp"`
	Error         string         `json:"error,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// TestChannelResponse respuesta de prueba de canal
type TestChannelResponse struct {
	Success      bool           `json:"success"`
	Message      string         `json:"message"`
	ResponseTime int64          `json:"response_time_ms"`
	ProviderInfo map[string]any `json:"provider_info,omitempty"`
	Error        string         `json:"error,omitempty"`
}

// ProcessWebhookResponse respuesta de procesamiento de webhook
type ProcessWebhookResponse struct {
	Success   bool            `json:"success"`
	Message   IncomingMessage `json:"message,omitempty"`
	Processed bool            `json:"processed"`
	Error     string          `json:"error,omitempty"`
}

// ============================================================================
// Stats DTOs
// ============================================================================

// ChannelStats estadísticas de un canal
type ChannelStats struct {
	ChannelID         kernel.ChannelID `json:"channel_id"`
	ChannelName       string           `json:"channel_name"`
	TotalMessagesSent int              `json:"total_messages_sent"`
	TotalMessagesRecv int              `json:"total_messages_received"`
	SuccessRate       float64          `json:"success_rate"`
	AvgResponseTime   float64          `json:"avg_response_time_ms"`
	LastMessageAt     *string          `json:"last_message_at,omitempty"`
	ErrorCount        int              `json:"error_count"`
}

// ChannelUsageResponse uso de canales en un periodo
type ChannelUsageResponse struct {
	TenantID         kernel.TenantID         `json:"tenant_id"`
	Period           string                  `json:"period"` // day, week, month
	TotalMessages    int                     `json:"total_messages"`
	MessagesSent     int                     `json:"messages_sent"`
	MessagesReceived int                     `json:"messages_received"`
	SuccessRate      float64                 `json:"success_rate"`
	ChannelBreakdown []ChannelUsageBreakdown `json:"channel_breakdown"`
}

type ChannelUsageBreakdown struct {
	ChannelID    kernel.ChannelID `json:"channel_id"`
	ChannelName  string           `json:"channel_name"`
	ChannelType  ChannelType      `json:"channel_type"`
	MessagesSent int              `json:"messages_sent"`
	MessagesRecv int              `json:"messages_received"`
	SuccessRate  float64          `json:"success_rate"`
}

// ============================================================================
// Bulk Operation DTOs
// ============================================================================

// BulkChannelOperationRequest request para operaciones masivas
type BulkChannelOperationRequest struct {
	TenantID   kernel.TenantID    `json:"tenant_id" validate:"required"`
	ChannelIDs []kernel.ChannelID `json:"channel_ids" validate:"required,min=1"`
	Operation  string             `json:"operation" validate:"required,oneof=activate deactivate delete test"`
}

// BulkChannelOperationResponse respuesta de operación masiva
type BulkChannelOperationResponse struct {
	Successful []kernel.ChannelID          `json:"successful"`
	Failed     map[kernel.ChannelID]string `json:"failed"`
	Total      int                         `json:"total"`
}

// ============================================================================
// Simple DTOs
// ============================================================================

// ChannelDetailsDTO DTO simplificado de canal
type ChannelDetailsDTO struct {
	ID         kernel.ChannelID `json:"id"`
	Name       string           `json:"name"`
	Type       ChannelType      `json:"type"`
	Provider   string           `json:"provider"`
	IsActive   bool             `json:"is_active"`
	WebhookURL string           `json:"webhook_url"`
}

// ToDTO convierte Channel a ChannelDetailsDTO
func (c *Channel) ToDTO() ChannelDetailsDTO {
	return ChannelDetailsDTO{
		ID:         c.ID,
		Name:       c.Name,
		Type:       c.Type,
		Provider:   c.GetProvider(),
		IsActive:   c.IsActive,
		WebhookURL: c.WebhookURL,
	}
}

// ============================================================================
// Validation DTOs
// ============================================================================

// ValidateChannelConfigRequest request para validar config de canal
type ValidateChannelConfigRequest struct {
	Type   ChannelType   `json:"type" validate:"required"`
	Config ChannelConfig `json:"config" validate:"required"`
}

// ValidateChannelConfigResponse respuesta de validación
type ValidateChannelConfigResponse struct {
	IsValid  bool     `json:"is_valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ChannelFeaturesResponse características de un tipo de canal
type ChannelFeaturesResponse struct {
	ChannelType ChannelType     `json:"channel_type"`
	Features    ChannelFeatures `json:"features"`
}

// AvailableChannelTypesResponse tipos de canales disponibles
type AvailableChannelTypesResponse struct {
	Types []ChannelTypeInfo `json:"types"`
}

type ChannelTypeInfo struct {
	Type        ChannelType     `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Providers   []string        `json:"providers"`
	Features    ChannelFeatures `json:"features"`
}

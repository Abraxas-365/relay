package channels

import (
	"encoding/json"
	"time"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Channel Entity (Single struct for DB)
// ============================================================================

// Channel estructura única que se guarda en la DB
type Channel struct {
	ID          kernel.ChannelID `db:"id" json:"id"`
	TenantID    kernel.TenantID  `db:"tenant_id" json:"tenant_id"`
	Type        ChannelType      `db:"type" json:"type"`
	Name        string           `db:"name" json:"name"`
	Description string           `db:"description" json:"description"`
	Config      json.RawMessage  `db:"config" json:"config"` // JSON que se deserializa según Type
	IsActive    bool             `db:"is_active" json:"is_active"`
	WebhookURL  string           `db:"webhook_url" json:"webhook_url"`
	CreatedAt   time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at" json:"updated_at"`
}

// ChannelType tipo de canal
type ChannelType string

const (
	ChannelTypeWhatsApp  ChannelType = "WHATSAPP"
	ChannelTypeInstagram ChannelType = "INSTAGRAM"
	ChannelTypeTelegram  ChannelType = "TELEGRAM"
	ChannelTypeInfobip   ChannelType = "INFOBIP"
	ChannelTypeEmail     ChannelType = "EMAIL"
	ChannelTypeSMS       ChannelType = "SMS"
	ChannelTypeWebChat   ChannelType = "WEBCHAT"
	ChannelTypeVoice     ChannelType = "VOICE"
	ChannelTypeTestHTTP  ChannelType = "TEST_HTTP"
)

// ============================================================================
// Channel Features
// ============================================================================

// ChannelFeatures características/capacidades de un canal
type ChannelFeatures struct {
	SupportsText                bool     `json:"supports_text"`
	SupportsAttachments         bool     `json:"supports_attachments"`
	SupportsImages              bool     `json:"supports_images"`
	SupportsAudio               bool     `json:"supports_audio"`
	SupportsVideo               bool     `json:"supports_video"`
	SupportsDocuments           bool     `json:"supports_documents"`
	SupportsInteractiveMessages bool     `json:"supports_interactive_messages"`
	SupportsButtons             bool     `json:"supports_buttons"`
	SupportsQuickReplies        bool     `json:"supports_quick_replies"`
	SupportsTemplates           bool     `json:"supports_templates"`
	SupportsLocation            bool     `json:"supports_location"`
	SupportsContacts            bool     `json:"supports_contacts"`
	SupportsReactions           bool     `json:"supports_reactions"`
	SupportsThreads             bool     `json:"supports_threads"`
	MaxMessageLength            int      `json:"max_message_length"`
	MaxAttachmentSize           int64    `json:"max_attachment_size_bytes"`
	SupportedMimeTypes          []string `json:"supported_mime_types,omitempty"`
}

// ============================================================================
// Config Interface
// ============================================================================

// ChannelConfig interfaz que todos los configs deben implementar
type ChannelConfig interface {
	Validate() error
	GetProvider() string
	GetFeatures() ChannelFeatures
	GetType() ChannelType
}

// ============================================================================
// WhatsApp Config
// ============================================================================

// WhatsAppConfig configuración para WhatsApp
type WhatsAppConfig struct {
	Provider           string `json:"provider"` // meta, twilio, infobip
	PhoneNumberID      string `json:"phone_number_id"`
	BusinessAccountID  string `json:"business_account_id"`
	AccessToken        string `json:"access_token"`
	AppSecret          string `json:"app_secret,omitempty"`
	WebhookVerifyToken string `json:"webhook_verify_token"`
	APIVersion         string `json:"api_version,omitempty"` // v17.0, v18.0

	// Buffer configuration
	BufferEnabled        bool `json:"buffer_enabled,omitempty"`          // Enable message buffering
	BufferTimeSeconds    int  `json:"buffer_time_seconds,omitempty"`     // Time window to buffer messages (e.g., 5 seconds)
	BufferResetOnMessage bool `json:"buffer_reset_on_message,omitempty"` // Reset timer on each new message
}

func (c WhatsAppConfig) Validate() error {
	if c.Provider == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "provider is required")
	}
	if c.PhoneNumberID == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "phone_number_id is required")
	}
	if c.AccessToken == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "access_token is required")
	}

	// Validate buffer config
	if c.BufferEnabled {
		if c.BufferTimeSeconds <= 0 {
			c.BufferTimeSeconds = 5 // Default 5 seconds
		}
		if c.BufferTimeSeconds > 60 {
			return ErrInvalidChannelConfig().WithDetail("reason", "buffer_time_seconds cannot exceed 60 seconds")
		}
	}

	return nil
}

func (c WhatsAppConfig) GetProvider() string {
	return c.Provider
}

func (c WhatsAppConfig) GetType() ChannelType {
	return ChannelTypeWhatsApp
}

func (c WhatsAppConfig) GetFeatures() ChannelFeatures {
	return ChannelFeatures{
		SupportsText:                true,
		SupportsAttachments:         true,
		SupportsImages:              true,
		SupportsAudio:               true,
		SupportsVideo:               true,
		SupportsDocuments:           true,
		SupportsInteractiveMessages: true,
		SupportsButtons:             true,
		SupportsQuickReplies:        true,
		SupportsTemplates:           true,
		SupportsLocation:            true,
		SupportsContacts:            true,
		SupportsReactions:           true,
		SupportsThreads:             false,
		MaxMessageLength:            4096,
		MaxAttachmentSize:           16 * 1024 * 1024, // 16MB
		SupportedMimeTypes: []string{
			"image/jpeg", "image/png",
			"video/mp4", "video/3gpp",
			"audio/aac", "audio/mp4", "audio/mpeg", "audio/amr", "audio/ogg",
			"application/pdf",
		},
	}
}

// ============================================================================
// Instagram Config
// ============================================================================

// InstagramConfig configuración para Instagram
type InstagramConfig struct {
	Provider    string `json:"provider"` // meta
	PageID      string `json:"page_id"`
	PageToken   string `json:"page_token"`
	AppSecret   string `json:"app_secret"`
	VerifyToken string `json:"verify_token"`
}

func (c InstagramConfig) Validate() error {
	if c.PageID == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "page_id is required")
	}
	if c.PageToken == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "page_token is required")
	}
	return nil
}

func (c InstagramConfig) GetProvider() string {
	return c.Provider
}

func (c InstagramConfig) GetType() ChannelType {
	return ChannelTypeInstagram
}

func (c InstagramConfig) GetFeatures() ChannelFeatures {
	return ChannelFeatures{
		SupportsText:                true,
		SupportsAttachments:         true,
		SupportsImages:              true,
		SupportsAudio:               false,
		SupportsVideo:               true,
		SupportsDocuments:           false,
		SupportsInteractiveMessages: true,
		SupportsButtons:             true,
		SupportsQuickReplies:        true,
		SupportsTemplates:           false,
		SupportsLocation:            false,
		SupportsContacts:            false,
		SupportsReactions:           true,
		SupportsThreads:             true,
		MaxMessageLength:            1000,
		MaxAttachmentSize:           8 * 1024 * 1024, // 8MB
		SupportedMimeTypes: []string{
			"image/jpeg", "image/png",
			"video/mp4",
		},
	}
}

// ============================================================================
// Telegram Config
// ============================================================================

// TelegramConfig configuración para Telegram
type TelegramConfig struct {
	Provider      string `json:"provider"` // telegram
	BotToken      string `json:"bot_token"`
	BotUsername   string `json:"bot_username,omitempty"`
	WebhookSecret string `json:"webhook_secret,omitempty"`
}

func (c TelegramConfig) Validate() error {
	if c.BotToken == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "bot_token is required")
	}
	return nil
}

func (c TelegramConfig) GetProvider() string {
	return c.Provider
}

func (c TelegramConfig) GetType() ChannelType {
	return ChannelTypeTelegram
}

func (c TelegramConfig) GetFeatures() ChannelFeatures {
	return ChannelFeatures{
		SupportsText:                true,
		SupportsAttachments:         true,
		SupportsImages:              true,
		SupportsAudio:               true,
		SupportsVideo:               true,
		SupportsDocuments:           true,
		SupportsInteractiveMessages: true,
		SupportsButtons:             true,
		SupportsQuickReplies:        false,
		SupportsTemplates:           false,
		SupportsLocation:            true,
		SupportsContacts:            true,
		SupportsReactions:           false,
		SupportsThreads:             true,
		MaxMessageLength:            4096,
		MaxAttachmentSize:           50 * 1024 * 1024, // 50MB
		SupportedMimeTypes: []string{
			"image/jpeg", "image/png", "image/gif",
			"video/mp4",
			"audio/mpeg", "audio/ogg",
			"application/pdf", "application/zip",
		},
	}
}

// ============================================================================
// Infobip Config
// ============================================================================

// InfobipConfig configuración para Infobip (multi-canal)
type InfobipConfig struct {
	Provider       string `json:"provider"` // infobip
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`         // https://api.infobip.com
	Sender         string `json:"sender"`           // número o ID de remitente
	SubChannelType string `json:"sub_channel_type"` // whatsapp, sms, email, viber
}

func (c InfobipConfig) Validate() error {
	if c.APIKey == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "api_key is required")
	}
	if c.BaseURL == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "base_url is required")
	}
	return nil
}

func (c InfobipConfig) GetProvider() string {
	return c.Provider
}

func (c InfobipConfig) GetType() ChannelType {
	return ChannelTypeInfobip
}

func (c InfobipConfig) GetFeatures() ChannelFeatures {
	// Features varían según SubChannelType, aquí las más comunes
	return ChannelFeatures{
		SupportsText:                true,
		SupportsAttachments:         true,
		SupportsImages:              true,
		SupportsAudio:               true,
		SupportsVideo:               true,
		SupportsDocuments:           true,
		SupportsInteractiveMessages: true,
		SupportsButtons:             true,
		SupportsQuickReplies:        true,
		SupportsTemplates:           true,
		SupportsLocation:            true,
		SupportsContacts:            false,
		SupportsReactions:           false,
		SupportsThreads:             false,
		MaxMessageLength:            4096,
		MaxAttachmentSize:           10 * 1024 * 1024, // 10MB
		SupportedMimeTypes: []string{
			"image/jpeg", "image/png",
			"video/mp4",
			"audio/mpeg",
			"application/pdf",
		},
	}
}

// ============================================================================
// Email Config
// ============================================================================

// EmailConfig configuración para Email
type EmailConfig struct {
	Provider  string `json:"provider"` // sendgrid, ses, smtp
	FromEmail string `json:"from_email"`
	FromName  string `json:"from_name"`
	APIKey    string `json:"api_key,omitempty"`

	// SMTP específico
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUsername string `json:"smtp_username,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	UseTLS       bool   `json:"use_tls,omitempty"`
}

func (c EmailConfig) Validate() error {
	if c.Provider == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "provider is required")
	}
	if c.FromEmail == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "from_email is required")
	}
	return nil
}

func (c EmailConfig) GetProvider() string {
	return c.Provider
}

func (c EmailConfig) GetType() ChannelType {
	return ChannelTypeEmail
}

func (c EmailConfig) GetFeatures() ChannelFeatures {
	return ChannelFeatures{
		SupportsText:                true,
		SupportsAttachments:         true,
		SupportsImages:              true,
		SupportsAudio:               false,
		SupportsVideo:               false,
		SupportsDocuments:           true,
		SupportsInteractiveMessages: false,
		SupportsButtons:             false,
		SupportsQuickReplies:        false,
		SupportsTemplates:           true,
		SupportsLocation:            false,
		SupportsContacts:            false,
		SupportsReactions:           false,
		SupportsThreads:             true,
		MaxMessageLength:            100000,
		MaxAttachmentSize:           25 * 1024 * 1024, // 25MB
		SupportedMimeTypes: []string{
			"image/jpeg", "image/png", "image/gif",
			"application/pdf",
			"application/msword",
			"application/vnd.ms-excel",
		},
	}
}

// ============================================================================
// SMS Config
// ============================================================================

// SMSConfig configuración para SMS
type SMSConfig struct {
	Provider  string `json:"provider"` // twilio, infobip, nexmo
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret,omitempty"`
	Sender    string `json:"sender"` // número de remitente
}

func (c SMSConfig) Validate() error {
	if c.Provider == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "provider is required")
	}
	if c.APIKey == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "api_key is required")
	}
	if c.Sender == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "sender is required")
	}
	return nil
}

func (c SMSConfig) GetProvider() string {
	return c.Provider
}

func (c SMSConfig) GetType() ChannelType {
	return ChannelTypeSMS
}

func (c SMSConfig) GetFeatures() ChannelFeatures {
	return ChannelFeatures{
		SupportsText:                true,
		SupportsAttachments:         false,
		SupportsImages:              false,
		SupportsAudio:               false,
		SupportsVideo:               false,
		SupportsDocuments:           false,
		SupportsInteractiveMessages: false,
		SupportsButtons:             false,
		SupportsQuickReplies:        false,
		SupportsTemplates:           false,
		SupportsLocation:            false,
		SupportsContacts:            false,
		SupportsReactions:           false,
		SupportsThreads:             false,
		MaxMessageLength:            160, // o 1600 para concatenados
		MaxAttachmentSize:           0,
		SupportedMimeTypes:          []string{},
	}
}

// ============================================================================
// WebChat Config
// ============================================================================

// WebChatConfig configuración para WebChat
type WebChatConfig struct {
	Provider   string            `json:"provider"` // custom, tawk, intercom
	WidgetID   string            `json:"widget_id"`
	APIKey     string            `json:"api_key,omitempty"`
	Settings   map[string]string `json:"settings,omitempty"`
	CustomCSS  string            `json:"custom_css,omitempty"`
	WelcomeMsg string            `json:"welcome_message,omitempty"`
}

func (c WebChatConfig) Validate() error {
	if c.WidgetID == "" {
		return ErrInvalidChannelConfig().WithDetail("reason", "widget_id is required")
	}
	return nil
}

func (c WebChatConfig) GetProvider() string {
	return c.Provider
}

func (c WebChatConfig) GetType() ChannelType {
	return ChannelTypeWebChat
}

func (c WebChatConfig) GetFeatures() ChannelFeatures {
	return ChannelFeatures{
		SupportsText:                true,
		SupportsAttachments:         true,
		SupportsImages:              true,
		SupportsAudio:               false,
		SupportsVideo:               false,
		SupportsDocuments:           true,
		SupportsInteractiveMessages: true,
		SupportsButtons:             true,
		SupportsQuickReplies:        true,
		SupportsTemplates:           false,
		SupportsLocation:            false,
		SupportsContacts:            false,
		SupportsReactions:           true,
		SupportsThreads:             true,
		MaxMessageLength:            10000,
		MaxAttachmentSize:           10 * 1024 * 1024, // 10MB
		SupportedMimeTypes: []string{
			"image/jpeg", "image/png", "image/gif",
			"application/pdf",
		},
	}
}

// ============================================================================
// Channel Domain Methods
// ============================================================================

// IsValid verifica si el canal es válido
func (c *Channel) IsValid() bool {
	return c.Name != "" && c.Type != "" && !c.TenantID.IsEmpty()
}

// Activate activa el canal
func (c *Channel) Activate() {
	c.IsActive = true
	c.UpdatedAt = time.Now()
}

// Deactivate desactiva el canal
func (c *Channel) Deactivate() {
	c.IsActive = false
	c.UpdatedAt = time.Now()
}

// UpdateDetails actualiza nombre y descripción
func (c *Channel) UpdateDetails(name, description string) {
	if name != "" {
		c.Name = name
	}
	if description != "" {
		c.Description = description
	}
	c.UpdatedAt = time.Now()
}

// UpdateConfig actualiza la configuración
func (c *Channel) UpdateConfig(config ChannelConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}
	c.Config = configJSON
	c.UpdatedAt = time.Now()
	return nil
}

// GetConfigStruct deserializa el config según el tipo
func (c *Channel) GetConfigStruct() (ChannelConfig, error) {
	switch c.Type {
	case ChannelTypeWhatsApp:
		var config WhatsAppConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ChannelTypeInstagram:
		var config InstagramConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ChannelTypeTelegram:
		var config TelegramConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ChannelTypeInfobip:
		var config InfobipConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ChannelTypeEmail:
		var config EmailConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ChannelTypeSMS:
		var config SMSConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	case ChannelTypeWebChat:
		var config WebChatConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil
	case ChannelTypeTestHTTP:
		var config TestHTTPConfig
		if err := json.Unmarshal(c.Config, &config); err != nil {
			return nil, err
		}
		return config, nil

	default:
		return nil, ErrChannelNotSupported().WithDetail("type", string(c.Type))
	}

}

// GetFeatures obtiene las features del canal
func (c *Channel) GetFeatures() (ChannelFeatures, error) {
	config, err := c.GetConfigStruct()
	if err != nil {
		return ChannelFeatures{}, err
	}
	return config.GetFeatures(), nil
}

// HasCredentials verifica si tiene credenciales configuradas
func (c *Channel) HasCredentials() bool {
	config, err := c.GetConfigStruct()
	if err != nil {
		return false
	}
	return config.GetProvider() != ""
}

// GetProvider retorna el proveedor
func (c *Channel) GetProvider() string {
	config, err := c.GetConfigStruct()
	if err != nil {
		return ""
	}
	return config.GetProvider()
}

// ============================================================================
// Helper Functions
// ============================================================================

// NewChannelFromConfig crea un canal desde una config
func NewChannelFromConfig(
	id kernel.ChannelID,
	tenantID kernel.TenantID,
	name string,
	description string,
	config ChannelConfig,
	webhookURL string,
) (*Channel, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	return &Channel{
		ID:          id,
		TenantID:    tenantID,
		Type:        config.GetType(),
		Name:        name,
		Description: description,
		Config:      configJSON,
		IsActive:    true,
		WebhookURL:  webhookURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

type TestHTTPConfig struct {
	Provider string `json:"provider"` // test
	Secret   string `json:"secret,omitempty"`
}

func (c TestHTTPConfig) Validate() error {
	return nil // No required fields for testing
}

func (c TestHTTPConfig) GetProvider() string {
	return "test"
}

func (c TestHTTPConfig) GetType() ChannelType {
	return ChannelTypeTestHTTP
}

func (c TestHTTPConfig) GetFeatures() ChannelFeatures {
	return ChannelFeatures{
		SupportsText:        true,
		SupportsAttachments: false,
		MaxMessageLength:    10000,
		SupportedMimeTypes:  []string{},
	}
}

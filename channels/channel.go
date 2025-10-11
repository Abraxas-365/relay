package channels

import (
	"github.com/Abraxas-365/relay/pkg/kernel"
	"time"
)

type Channel struct {
	ID         kernel.ChannelID `db:"id" json:"id"`
	TenantID   kernel.TenantID  `db:"tenant_id" json:"tenant_id"`
	Type       ChannelType      `db:"type" json:"type"`
	Name       string           `db:"name" json:"name"`
	Config     ChannelConfig    `db:"config" json:"config"`
	IsActive   bool             `db:"is_active" json:"is_active"`
	WebhookURL string           `db:"webhook_url" json:"webhook_url"`
	CreatedAt  time.Time        `db:"created_at" json:"created_at"`
}

type ChannelType string

const (
	ChannelTypeWhatsApp  ChannelType = "WHATSAPP"
	ChannelTypeEmail     ChannelType = "EMAIL"
	ChannelTypeInstagram ChannelType = "INSTAGRAM"
	ChannelTypeTelegram  ChannelType = "TELEGRAM"
)

type ChannelConfig struct {
	Provider    string         `json:"provider"` // twilio, sendgrid, meta
	Credentials map[string]any `json:"credentials"`
	Settings    map[string]any `json:"settings"`
}

// IncomingMessage mensaje que llega desde un canal
type IncomingMessage struct {
	ChannelID  kernel.ChannelID
	SenderID   string
	SenderName string
	Content    MessageContent
	Timestamp  time.Time
	RawPayload map[string]any
}

type MessageContent struct {
	Type        string
	Text        string
	Attachments []Attachment
}

type Attachment struct {
	Type string `json:"type"` // image, audio, video, document
	URL  string `json:"url"`
	Name string `json:"name,omitempty"`
}

// OutgoingMessage mensaje para enviar a un canal
type OutgoingMessage struct {
	RecipientID string
	Content     MessageContent
	Metadata    map[string]any
}

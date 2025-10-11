package channels

import (
	"context"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ChannelRepository persistencia de canales
type ChannelRepository interface {
	Save(ctx context.Context, ch Channel) error
	FindByID(ctx context.Context, id kernel.ChannelID) (*Channel, error)
	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*Channel, error)
	FindByWebhookURL(ctx context.Context, webhookURL string) (*Channel, error)
}

// ChannelAdapter adapta cada proveedor específico
type ChannelAdapter interface {
	GetType() ChannelType
	ParseIncoming(rawPayload map[string]any) (*IncomingMessage, error)
	SendMessage(ctx context.Context, msg OutgoingMessage) error
	ValidateConfig(config ChannelConfig) error
}

// ChannelManager gestiona todos los canales
type ChannelManager interface {
	RegisterChannel(ctx context.Context, ch Channel) error
	SendMessage(ctx context.Context, channelID kernel.ChannelID, msg OutgoingMessage) error
	GetAdapter(channelType ChannelType) (ChannelAdapter, error)
}

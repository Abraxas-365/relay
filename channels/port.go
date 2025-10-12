package channels

import (
	"context"

	"github.com/Abraxas-365/relay/pkg/kernel"
)

// ============================================================================
// Repository Interfaces
// ============================================================================

// ChannelRepository define el contrato para persistencia de canales
type ChannelRepository interface {
	// CRUD básico
	Save(ctx context.Context, channel Channel) error
	FindByID(ctx context.Context, id kernel.ChannelID, tenantID kernel.TenantID) (*Channel, error)
	FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*Channel, error)
	Delete(ctx context.Context, id kernel.ChannelID, tenantID kernel.TenantID) error
	ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error)

	// Búsquedas específicas
	FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*Channel, error)
	FindByType(ctx context.Context, channelType ChannelType, tenantID kernel.TenantID) ([]*Channel, error)
	FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*Channel, error)
	FindByProvider(ctx context.Context, provider string, tenantID kernel.TenantID) ([]*Channel, error)

	// List con paginación
	List(ctx context.Context, req ListChannelsRequest) (ChannelListResponse, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, ids []kernel.ChannelID, tenantID kernel.TenantID, isActive bool) error

	// Stats
	CountByType(ctx context.Context, channelType ChannelType, tenantID kernel.TenantID) (int, error)
	CountByTenant(ctx context.Context, tenantID kernel.TenantID) (int, error)
}

// ============================================================================
// Adapter Interfaces
// ============================================================================

// ChannelAdapter interfaz para adaptadores de canal específicos
type ChannelAdapter interface {
	// GetType retorna el tipo de canal que maneja
	GetType() ChannelType

	// SendMessage envía un mensaje a través del canal
	SendMessage(ctx context.Context, msg OutgoingMessage) error

	// ValidateConfig valida la configuración del canal
	ValidateConfig(config ChannelConfig) error

	// ProcessWebhook procesa webhooks entrantes del proveedor
	ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*IncomingMessage, error)

	// GetFeatures retorna las características soportadas
	GetFeatures() ChannelFeatures

	// TestConnection prueba la conexión con el proveedor
	TestConnection(ctx context.Context, config ChannelConfig) error
}

// ============================================================================
// Manager Interfaces
// ============================================================================

// ChannelManager gestiona operaciones de alto nivel con canales
type ChannelManager interface {
	// RegisterChannel registra un nuevo canal
	RegisterChannel(ctx context.Context, channel Channel) error

	// SendMessage envía un mensaje a través de un canal
	SendMessage(ctx context.Context, tenantID kernel.TenantID, channelID kernel.ChannelID, msg OutgoingMessage) error

	// ProcessIncomingMessage procesa un mensaje entrante
	ProcessIncomingMessage(ctx context.Context, tenantID kernel.TenantID, channelID kernel.ChannelID, msg IncomingMessage) error

	// GetAdapter obtiene el adapter para un tipo de canal
	GetAdapter(channelID kernel.ChannelID) (ChannelAdapter, error)
}

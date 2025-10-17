package channelmanager

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Abraxas-365/relay/channels"
	instagram "github.com/Abraxas-365/relay/channels/channeladapters/instagram"
	whatsapp "github.com/Abraxas-365/relay/channels/channeladapters/whatssapp"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/go-redis/redis/v8"
)

// DefaultChannelManager implementación del ChannelManager
type DefaultChannelManager struct {
	mu sync.RWMutex

	// ✅ Adapters registrados por CHANNEL ID (no por tipo)
	adapters map[kernel.ChannelID]channels.ChannelAdapter

	// Canales registrados por ID
	channels map[kernel.ChannelID]*channels.Channel

	// Channel repository para persistencia
	channelRepo channels.ChannelRepository

	// ✅ Redis client para crear adapters de WhatsApp
	redisClient *redis.Client
}

// NewDefaultChannelManager crea una nueva instancia
func NewDefaultChannelManager(
	channelRepo channels.ChannelRepository,
	redisClient *redis.Client,
) *DefaultChannelManager {
	return &DefaultChannelManager{
		adapters:    make(map[kernel.ChannelID]channels.ChannelAdapter),
		channels:    make(map[kernel.ChannelID]*channels.Channel),
		channelRepo: channelRepo,
		redisClient: redisClient,
	}
}

// RegisterChannel registra un canal en el manager y crea su adapter
func (cm *DefaultChannelManager) RegisterChannel(ctx context.Context, channel channels.Channel) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Verificar que el canal sea válido
	if !channel.IsValid() {
		return channels.ErrInvalidChannelConfig().WithDetail("reason", "channel is not valid")
	}

	// ✅ Crear adapter específico para este canal
	adapter, err := cm.createAdapterForChannel(channel)
	if err != nil {
		log.Printf("❌ Failed to create adapter for channel %s: %v", channel.ID.String(), err)
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Registrar canal y adapter en memoria
	cm.channels[channel.ID] = &channel
	cm.adapters[channel.ID] = adapter

	log.Printf("✅ Channel registered: %s (type: %s, id: %s)", channel.Name, channel.Type, channel.ID.String())

	return nil
}

// ✅ createAdapterForChannel crea un adapter con la config específica del canal
func (cm *DefaultChannelManager) createAdapterForChannel(channel channels.Channel) (channels.ChannelAdapter, error) {
	switch channel.Type {
	case channels.ChannelTypeWhatsApp:
		// Obtener config tipada
		config, err := channel.GetConfigStruct()
		if err != nil {
			return nil, fmt.Errorf("failed to get config struct: %w", err)
		}

		whatsappConfig, ok := config.(channels.WhatsAppConfig)
		if !ok {
			return nil, fmt.Errorf("invalid WhatsApp config type")
		}

		// Validar config
		if err := whatsappConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid WhatsApp config: %w", err)
		}

		// Log config details
		log.Printf("🔧 Creating WhatsApp adapter for channel: %s", channel.ID)
		log.Printf("   📱 Phone Number ID: %s", whatsappConfig.PhoneNumberID)
		log.Printf("   🌐 API Version: %s", whatsappConfig.APIVersion)
		log.Printf("   🏢 Business Account: %s", whatsappConfig.BusinessAccountID)
		log.Printf("   🔑 Access Token: %s... (%d chars)",
			safeSubstring(whatsappConfig.AccessToken, 20),
			len(whatsappConfig.AccessToken))

		// Crear adapter
		adapter := whatsapp.NewWhatsAppAdapter(whatsappConfig, cm.redisClient)
		if adapter == nil {
			return nil, fmt.Errorf("failed to create WhatsApp adapter")
		}

		return adapter, nil

	case channels.ChannelTypeInstagram:
		// Obtener config tipada
		config, err := channel.GetConfigStruct()
		if err != nil {
			return nil, fmt.Errorf("failed to get config struct: %w", err)
		}

		instagramConfig, ok := config.(channels.InstagramConfig)
		if !ok {
			return nil, fmt.Errorf("invalid Instagram config type")
		}

		// Validar config
		if err := instagramConfig.Validate(); err != nil {
			return nil, fmt.Errorf("invalid Instagram config: %w", err)
		}

		// Log config details
		log.Printf("🔧 Creating Instagram adapter for channel: %s", channel.ID)
		log.Printf("   📱 Page ID: %s", instagramConfig.PageID)
		log.Printf("   🏢 Provider: %s", instagramConfig.Provider)
		log.Printf("   🔑 Page Token: %s... (%d chars)",
			safeSubstring(instagramConfig.PageToken, 20),
			len(instagramConfig.PageToken))

		// Crear adapter with Redis client for buffering
		adapter := instagram.NewInstagramAdapter(instagramConfig, cm.redisClient)
		if adapter == nil {
			return nil, fmt.Errorf("failed to create Instagram adapter")
		}

		return adapter, nil

	// ✅ Agregar más tipos de canales aquí
	// case channels.ChannelTypeTelegram:
	//     ...
	// case channels.ChannelTypeSlack:
	//     ...

	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channel.Type)
	}
}

// SendMessage envía un mensaje a través de un canal
func (cm *DefaultChannelManager) SendMessage(
	ctx context.Context,
	tenantID kernel.TenantID,
	channelID kernel.ChannelID,
	msg channels.OutgoingMessage,
) error {
	// Obtener canal
	cm.mu.RLock()
	channel, channelExists := cm.channels[channelID]
	adapter, adapterExists := cm.adapters[channelID]
	cm.mu.RUnlock()

	// Si el canal no está en cache, cargarlo desde DB
	if !channelExists {
		log.Printf("⚠️  Channel %s not in cache, loading from database...", channelID)

		// Extraer tenantID del mensaje o del contexto
		// Nota: Necesitarás pasar tenantID como parámetro o extraerlo del contexto
		var err error
		channel, err = cm.channelRepo.FindByID(ctx, channelID, tenantID) // ⚠️ Fix tenantID
		if err != nil {
			return channels.ErrChannelNotFound().
				WithDetail("channel_id", channelID.String())
		}

		// Registrar el canal (esto creará el adapter)
		if err := cm.RegisterChannel(ctx, *channel); err != nil {
			return err
		}

		// Obtener el adapter recién creado
		cm.mu.RLock()
		adapter = cm.adapters[channelID]
		cm.mu.RUnlock()
	}

	// Si no hay adapter, intentar crearlo
	if !adapterExists {
		log.Printf("⚠️  Adapter not found for channel %s, creating...", channelID)

		newAdapter, err := cm.createAdapterForChannel(*channel)
		if err != nil {
			return err
		}

		cm.mu.Lock()
		cm.adapters[channelID] = newAdapter
		adapter = newAdapter
		cm.mu.Unlock()
	}

	// Verificar que el canal esté activo
	if !channel.IsActive {
		return channels.ErrChannelInactive().WithDetail("channel_id", channelID.String())
	}

	// Enviar mensaje usando el adapter específico del canal
	log.Printf("📤 Sending message via channel %s (type: %s) to %s",
		channel.Name, channel.Type, msg.RecipientID)

	if err := adapter.SendMessage(ctx, msg); err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		return channels.ErrMessageSendFailed().
			WithDetail("channel_id", channelID.String()).
			WithDetail("error", err.Error())
	}

	log.Printf("✅ Message sent successfully via %s", channel.Name)
	return nil
}

// ProcessIncomingMessage procesa un mensaje entrante
func (cm *DefaultChannelManager) ProcessIncomingMessage(
	ctx context.Context,
	tenantID kernel.TenantID,
	channelID kernel.ChannelID,
	msg channels.IncomingMessage,
) error {
	// Obtener canal
	channel, err := cm.getChannel(ctx, tenantID, channelID)
	if err != nil {
		return err
	}

	// Verificar que el canal esté activo
	if !channel.IsActive {
		return channels.ErrChannelInactive().WithDetail("channel_id", channelID.String())
	}

	log.Printf("📥 Processing incoming message from %s via channel %s", msg.SenderID, channel.Name)

	return nil
}

// getChannel obtiene un canal (primero de cache, luego de DB)
func (cm *DefaultChannelManager) getChannel(
	ctx context.Context,
	tenantID kernel.TenantID,
	channelID kernel.ChannelID,
) (*channels.Channel, error) {
	// Intentar obtener de cache primero
	cm.mu.RLock()
	channel, exists := cm.channels[channelID]
	cm.mu.RUnlock()

	if exists {
		return channel, nil
	}

	// Si no está en cache, buscar en DB
	channel, err := cm.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return nil, err
	}

	// Registrar en cache para futuras llamadas
	cm.mu.Lock()
	cm.channels[channelID] = channel
	cm.mu.Unlock()

	return channel, nil
}

// LoadChannels carga canales de un tenant en memoria
func (cm *DefaultChannelManager) LoadChannels(ctx context.Context, tenantID kernel.TenantID) error {
	if cm.channelRepo == nil {
		log.Println("⚠️  Channel repository not available, skipping channel loading")
		return nil
	}

	channels, err := cm.channelRepo.FindByTenant(ctx, tenantID)
	if err != nil {
		return err
	}

	// ✅ Registrar cada canal (esto crea los adapters)
	successCount := 0
	for _, ch := range channels {
		if err := cm.RegisterChannel(ctx, *ch); err != nil {
			log.Printf("⚠️  Failed to register channel %s: %v", ch.ID, err)
			continue
		}
		successCount++
	}

	log.Printf("✅ Loaded %d/%d channels for tenant %s", successCount, len(channels), tenantID.String())
	return nil
}

// GetAdapter obtiene el adapter para un canal específico
func (cm *DefaultChannelManager) GetAdapter(channelID kernel.ChannelID) (channels.ChannelAdapter, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	adapter, exists := cm.adapters[channelID]
	if !exists {
		return nil, channels.ErrChannelNotFound().
			WithDetail("channel_id", channelID.String())
	}

	return adapter, nil
}

// GetRegisteredChannels retorna los IDs de canales registrados
func (cm *DefaultChannelManager) GetRegisteredChannels() []kernel.ChannelID {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	ids := make([]kernel.ChannelID, 0, len(cm.channels))
	for channelID := range cm.channels {
		ids = append(ids, channelID)
	}

	return ids
}

// GetChannelsByType retorna canales de un tipo específico
func (cm *DefaultChannelManager) GetChannelsByType(channelType channels.ChannelType) []*channels.Channel {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []*channels.Channel
	for _, channel := range cm.channels {
		if channel.Type == channelType {
			result = append(result, channel)
		}
	}

	return result
}

// UnregisterChannel elimina un canal y su adapter
func (cm *DefaultChannelManager) UnregisterChannel(channelID kernel.ChannelID) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.channels, channelID)
	delete(cm.adapters, channelID)

	log.Printf("🗑️  Channel unregistered: %s", channelID)
}

// ReloadChannel recarga un canal (útil cuando cambia la config)
func (cm *DefaultChannelManager) ReloadChannel(ctx context.Context, channelID kernel.ChannelID, tenantID kernel.TenantID) error {
	// Cargar canal actualizado desde DB
	channel, err := cm.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return err
	}

	// Eliminar el anterior
	cm.UnregisterChannel(channelID)

	// Registrar el nuevo
	return cm.RegisterChannel(ctx, *channel)
}

// ============================================================================
// Helper Functions
// ============================================================================

// safeSubstring extrae substring de forma segura
func safeSubstring(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length]
}

package channelmanager

import (
	"context"
	"log"
	"sync"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// DefaultChannelManager implementación del ChannelManager
type DefaultChannelManager struct {
	mu sync.RWMutex

	// Adapters registrados por tipo de canal
	adapters map[channels.ChannelType]channels.ChannelAdapter

	// Canales registrados por ID
	channels map[kernel.ChannelID]*channels.Channel

	// Channel repository para persistencia
	channelRepo channels.ChannelRepository
}

// NewDefaultChannelManager crea una nueva instancia
func NewDefaultChannelManager(channelRepo channels.ChannelRepository) *DefaultChannelManager {
	return &DefaultChannelManager{
		adapters:    make(map[channels.ChannelType]channels.ChannelAdapter),
		channels:    make(map[kernel.ChannelID]*channels.Channel),
		channelRepo: channelRepo,
	}
}

// RegisterAdapter registra un adapter para un tipo de canal
func (cm *DefaultChannelManager) RegisterAdapter(adapter channels.ChannelAdapter) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	channelType := adapter.GetType()
	cm.adapters[channelType] = adapter

	log.Printf("📝 Registered adapter for channel type: %s", channelType)
}

// RegisterChannel registra un canal en el manager
func (cm *DefaultChannelManager) RegisterChannel(ctx context.Context, channel channels.Channel) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Verificar que el canal sea válido
	if !channel.IsValid() {
		return channels.ErrInvalidChannelConfig().WithDetail("reason", "channel is not valid")
	}

	// Verificar que exista adapter para este tipo
	if _, exists := cm.adapters[channel.Type]; !exists {
		log.Printf("⚠️  No adapter found for channel type: %s (channel: %s)", channel.Type, channel.ID.String())
		// No fallar, solo advertir - permite registro sin adapter
	}

	// Registrar canal en memoria
	cm.channels[channel.ID] = &channel

	log.Printf("✅ Channel registered: %s (type: %s, id: %s)", channel.Name, channel.Type, channel.ID.String())

	return nil
}

// GetAdapter obtiene el adapter para un tipo de canal
func (cm *DefaultChannelManager) GetAdapter(channelType channels.ChannelType) (channels.ChannelAdapter, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	adapter, exists := cm.adapters[channelType]
	if !exists {
		return nil, channels.ErrChannelNotSupported().WithDetail("type", string(channelType))
	}

	return adapter, nil
}

// SendMessage envía un mensaje a través de un canal
func (cm *DefaultChannelManager) SendMessage(ctx context.Context, channelID kernel.ChannelID, msg channels.OutgoingMessage) error {
	// Obtener canal
	channel, err := cm.getChannel(ctx, channelID)
	if err != nil {
		return err
	}

	// Verificar que el canal esté activo
	if !channel.IsActive {
		return channels.ErrChannelInactive().WithDetail("channel_id", channelID.String())
	}

	// Obtener adapter
	adapter, err := cm.GetAdapter(channel.Type)
	if err != nil {
		return err
	}

	// Enviar mensaje usando el adapter
	log.Printf("📤 Sending message via channel %s (type: %s) to %s", channel.Name, channel.Type, msg.RecipientID)

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
func (cm *DefaultChannelManager) ProcessIncomingMessage(ctx context.Context, channelID kernel.ChannelID, msg channels.IncomingMessage) error {
	// Obtener canal
	channel, err := cm.getChannel(ctx, channelID)
	if err != nil {
		return err
	}

	// Verificar que el canal esté activo
	if !channel.IsActive {
		return channels.ErrChannelInactive().WithDetail("channel_id", channelID.String())
	}

	log.Printf("📥 Processing incoming message from %s via channel %s", msg.SenderID, channel.Name)

	// TODO: Aquí puedes agregar lógica adicional de procesamiento
	// Por ejemplo, validación, transformación, etc.

	return nil
}

// TestChannel prueba la conexión de un canal
func (cm *DefaultChannelManager) TestChannel(ctx context.Context, channelID kernel.ChannelID) error {
	// Obtener canal
	channel, err := cm.getChannel(ctx, channelID)
	if err != nil {
		return err
	}

	// Obtener adapter
	adapter, err := cm.GetAdapter(channel.Type)
	if err != nil {
		return err
	}

	// Obtener configuración del canal
	config, err := channel.GetConfigStruct()
	if err != nil {
		return channels.ErrInvalidChannelConfig().
			WithDetail("channel_id", channelID.String()).
			WithDetail("error", err.Error())
	}

	// Probar conexión
	log.Printf("🧪 Testing channel: %s (type: %s)", channel.Name, channel.Type)

	if err := adapter.TestConnection(ctx, config); err != nil {
		log.Printf("❌ Channel test failed: %v", err)
		return err
	}

	log.Printf("✅ Channel test successful: %s", channel.Name)
	return nil
}

// getChannel obtiene un canal por ID (primero de cache, luego de DB)
func (cm *DefaultChannelManager) getChannel(ctx context.Context, channelID kernel.ChannelID) (*channels.Channel, error) {
	// Intentar obtener de cache primero
	cm.mu.RLock()
	channel, exists := cm.channels[channelID]
	cm.mu.RUnlock()

	if exists {
		return channel, nil
	}

	// Si no está en cache, intentar cargar de DB
	// Nota: necesitamos el tenantID, pero no lo tenemos aquí
	// Por ahora, retornamos error
	log.Printf("⚠️  Channel %s not found in cache", channelID.String())

	return nil, channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
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

	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, ch := range channels {
		cm.channels[ch.ID] = ch
		log.Printf("📥 Loaded channel: %s (type: %s)", ch.Name, ch.Type)
	}

	log.Printf("✅ Loaded %d channels for tenant %s", len(channels), tenantID.String())
	return nil
}

// GetRegisteredAdapters retorna los tipos de adaptadores registrados
func (cm *DefaultChannelManager) GetRegisteredAdapters() []channels.ChannelType {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	types := make([]channels.ChannelType, 0, len(cm.adapters))
	for channelType := range cm.adapters {
		types = append(types, channelType)
	}

	return types
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

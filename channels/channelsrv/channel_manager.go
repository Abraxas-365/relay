package channelsrv

import (
	"context"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/iam/tenant"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/google/uuid"
)

// ChannelService proporciona operaciones de negocio para canales
type ChannelService struct {
	channelRepo    channels.ChannelRepository
	tenantRepo     tenant.TenantRepository
	channelManager channels.ChannelManager
}

// NewChannelService crea una nueva instancia del servicio de canales
func NewChannelService(
	channelRepo channels.ChannelRepository,
	tenantRepo tenant.TenantRepository,
	channelManager channels.ChannelManager,
) *ChannelService {
	return &ChannelService{
		channelRepo:    channelRepo,
		tenantRepo:     tenantRepo,
		channelManager: channelManager,
	}
}

// ============================================================================
// CRUD Operations
// ============================================================================

// CreateChannel crea un nuevo canal
func (s *ChannelService) CreateChannel(ctx context.Context, req channels.CreateChannelRequest) (*channels.Channel, error) {
	// Verificar que el tenant exista y esté activo
	tenantEntity, err := s.tenantRepo.FindByID(ctx, req.TenantID)
	if err != nil {
		return nil, tenant.ErrTenantNotFound()
	}

	if !tenantEntity.IsActive() {
		return nil, tenant.ErrTenantSuspended()
	}

	// Verificar que no exista un canal con el mismo nombre
	exists, err := s.channelRepo.ExistsByName(ctx, req.Name, req.TenantID)
	if err != nil {
		return nil, errx.Wrap(err, "failed to check channel name existence", errx.TypeInternal)
	}
	if exists {
		return nil, channels.ErrChannelAlreadyExists().WithDetail("name", req.Name)
	}

	// Generar webhook URL
	webhookURL := s.generateWebhookURL(req.TenantID, req.Type)

	// Crear canal usando el helper
	newChannel, err := channels.NewChannelFromConfig(
		kernel.NewChannelID(uuid.NewString()),
		req.TenantID,
		req.Name,
		req.Description,
		req.Config,
		webhookURL,
	)
	if err != nil {
		return nil, errx.Wrap(err, "failed to create channel", errx.TypeInternal)
	}

	// Validar config usando el adapter si está disponible
	if adapter, err := s.channelManager.GetAdapter(newChannel.Type); err == nil {
		if err := adapter.ValidateConfig(req.Config); err != nil {
			return nil, channels.ErrInvalidChannelConfig().
				WithDetail("reason", err.Error())
		}
	}

	// Guardar canal
	if err := s.channelRepo.Save(ctx, *newChannel); err != nil {
		return nil, errx.Wrap(err, "failed to save channel", errx.TypeInternal)
	}

	// Registrar en el channel manager
	if err := s.channelManager.RegisterChannel(ctx, *newChannel); err != nil {
		// Log error but don't fail
		// logger.Error("Failed to register channel in manager", err)
	}

	return newChannel, nil
}

// GetChannelByID obtiene un canal por ID
func (s *ChannelService) GetChannelByID(ctx context.Context, channelID kernel.ChannelID, tenantID kernel.TenantID) (*channels.ChannelResponse, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return nil, channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
	}

	features, err := channel.GetFeatures()
	if err != nil {
		// Return channel without features
		features = channels.ChannelFeatures{}
	}

	return &channels.ChannelResponse{
		Channel:  *channel,
		Features: features,
	}, nil
}

// GetChannelByName obtiene un canal por nombre
func (s *ChannelService) GetChannelByName(ctx context.Context, name string, tenantID kernel.TenantID) (*channels.ChannelResponse, error) {
	channel, err := s.channelRepo.FindByName(ctx, name, tenantID)
	if err != nil {
		return nil, channels.ErrChannelNotFound().WithDetail("name", name)
	}

	features, _ := channel.GetFeatures()

	return &channels.ChannelResponse{
		Channel:  *channel,
		Features: features,
	}, nil
}

// GetChannelsByTenant obtiene todos los canales de un tenant
func (s *ChannelService) GetChannelsByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*channels.Channel, error) {
	return s.channelRepo.FindByTenant(ctx, tenantID)
}

// GetActiveChannels obtiene canales activos de un tenant
func (s *ChannelService) GetActiveChannels(ctx context.Context, tenantID kernel.TenantID) ([]*channels.Channel, error) {
	return s.channelRepo.FindActive(ctx, tenantID)
}

// GetChannelsByType obtiene canales por tipo
func (s *ChannelService) GetChannelsByType(ctx context.Context, channelType channels.ChannelType, tenantID kernel.TenantID) ([]*channels.Channel, error) {
	return s.channelRepo.FindByType(ctx, channelType, tenantID)
}

// UpdateChannel actualiza un canal
func (s *ChannelService) UpdateChannel(ctx context.Context, channelID kernel.ChannelID, req channels.UpdateChannelRequest, tenantID kernel.TenantID) (*channels.Channel, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return nil, channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
	}

	// Actualizar campos si se proporcionaron
	if req.Name != nil {
		// Verificar que no exista otro canal con el mismo nombre
		if *req.Name != channel.Name {
			exists, err := s.channelRepo.ExistsByName(ctx, *req.Name, tenantID)
			if err != nil {
				return nil, errx.Wrap(err, "failed to check channel name", errx.TypeInternal)
			}
			if exists {
				return nil, channels.ErrChannelAlreadyExists().WithDetail("name", *req.Name)
			}
		}
		channel.Name = *req.Name
	}

	if req.Description != nil {
		channel.Description = *req.Description
	}

	if req.Config != nil {
		// Validar config
		if adapter, err := s.channelManager.GetAdapter(channel.Type); err == nil {
			if err := adapter.ValidateConfig(*req.Config); err != nil {
				return nil, channels.ErrInvalidChannelConfig().WithDetail("reason", err.Error())
			}
		}

		if err := channel.UpdateConfig(*req.Config); err != nil {
			return nil, errx.Wrap(err, "failed to update config", errx.TypeInternal)
		}
	}

	if req.IsActive != nil {
		if *req.IsActive {
			channel.Activate()
		} else {
			channel.Deactivate()
		}
	}

	channel.UpdatedAt = time.Now()

	// Guardar cambios
	if err := s.channelRepo.Save(ctx, *channel); err != nil {
		return nil, errx.Wrap(err, "failed to update channel", errx.TypeInternal)
	}

	return channel, nil
}

// ActivateChannel activa un canal
func (s *ChannelService) ActivateChannel(ctx context.Context, channelID kernel.ChannelID, tenantID kernel.TenantID) error {
	channel, err := s.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
	}

	channel.Activate()
	return s.channelRepo.Save(ctx, *channel)
}

// DeactivateChannel desactiva un canal
func (s *ChannelService) DeactivateChannel(ctx context.Context, channelID kernel.ChannelID, tenantID kernel.TenantID) error {
	channel, err := s.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
	}

	channel.Deactivate()
	return s.channelRepo.Save(ctx, *channel)
}

// DeleteChannel elimina un canal
func (s *ChannelService) DeleteChannel(ctx context.Context, channelID kernel.ChannelID, tenantID kernel.TenantID) error {
	// Verificar que el canal existe
	_, err := s.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
	}

	// Eliminar canal
	return s.channelRepo.Delete(ctx, channelID, tenantID)
}

// ============================================================================
// Messaging Operations
// ============================================================================

// SendMessage envía un mensaje a través de un canal
func (s *ChannelService) SendMessage(ctx context.Context, channelID kernel.ChannelID, msg channels.OutgoingMessage) (*channels.SendMessageResponse, error) {
	// Verificar que el canal existe y está activo
	channel, err := s.channelRepo.FindByID(ctx, channelID, msg.Metadata["tenant_id"].(kernel.TenantID))
	if err != nil {
		return nil, channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
	}

	if !channel.IsActive {
		return nil, channels.ErrChannelInactive().WithDetail("channel_id", channelID.String())
	}

	// Enviar mensaje usando el channel manager
	startTime := time.Now()
	if err := s.channelManager.SendMessage(ctx, channelID, msg); err != nil {
		return &channels.SendMessageResponse{
			Success:   false,
			Timestamp: time.Now().Unix(),
			Error:     err.Error(),
		}, err
	}

	return &channels.SendMessageResponse{
		Success:   true,
		Timestamp: time.Now().Unix(),
		Metadata: map[string]any{
			"processing_time_ms": time.Since(startTime).Milliseconds(),
		},
	}, nil
}

// TestChannel prueba la conexión de un canal
func (s *ChannelService) TestChannel(ctx context.Context, channelID kernel.ChannelID, tenantID kernel.TenantID) (*channels.TestChannelResponse, error) {
	channel, err := s.channelRepo.FindByID(ctx, channelID, tenantID)
	if err != nil {
		return nil, channels.ErrChannelNotFound().WithDetail("channel_id", channelID.String())
	}

	// Obtener adapter
	adapter, err := s.channelManager.GetAdapter(channel.Type)
	if err != nil {
		return &channels.TestChannelResponse{
			Success: false,
			Message: "Adapter not available",
			Error:   err.Error(),
		}, err
	}

	// Probar conexión
	startTime := time.Now()
	config, err := channel.GetConfigStruct()
	if err != nil {
		return &channels.TestChannelResponse{
			Success: false,
			Message: "Invalid channel configuration",
			Error:   err.Error(),
		}, err
	}

	if err := adapter.TestConnection(ctx, config); err != nil {
		return &channels.TestChannelResponse{
			Success:      false,
			Message:      "Connection test failed",
			ResponseTime: time.Since(startTime).Milliseconds(),
			Error:        err.Error(),
		}, err
	}

	return &channels.TestChannelResponse{
		Success:      true,
		Message:      "Connection test successful",
		ResponseTime: time.Since(startTime).Milliseconds(),
	}, nil
}

// ============================================================================
// Bulk Operations
// ============================================================================

// BulkActivateChannels activa múltiples canales
func (s *ChannelService) BulkActivateChannels(ctx context.Context, channelIDs []kernel.ChannelID, tenantID kernel.TenantID) (*channels.BulkChannelOperationResponse, error) {
	result := &channels.BulkChannelOperationResponse{
		Successful: []kernel.ChannelID{},
		Failed:     make(map[kernel.ChannelID]string),
		Total:      len(channelIDs),
	}

	for _, channelID := range channelIDs {
		if err := s.ActivateChannel(ctx, channelID, tenantID); err != nil {
			result.Failed[channelID] = err.Error()
		} else {
			result.Successful = append(result.Successful, channelID)
		}
	}

	return result, nil
}

// BulkDeactivateChannels desactiva múltiples canales
func (s *ChannelService) BulkDeactivateChannels(ctx context.Context, channelIDs []kernel.ChannelID, tenantID kernel.TenantID) (*channels.BulkChannelOperationResponse, error) {
	result := &channels.BulkChannelOperationResponse{
		Successful: []kernel.ChannelID{},
		Failed:     make(map[kernel.ChannelID]string),
		Total:      len(channelIDs),
	}

	for _, channelID := range channelIDs {
		if err := s.DeactivateChannel(ctx, channelID, tenantID); err != nil {
			result.Failed[channelID] = err.Error()
		} else {
			result.Successful = append(result.Successful, channelID)
		}
	}

	return result, nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// generateWebhookURL genera una URL de webhook para el canal
func (s *ChannelService) generateWebhookURL(tenantID kernel.TenantID, channelType channels.ChannelType) string {
	// En producción, esto debería usar la configuración real del servidor
	baseURL := "https://api.yourdomain.com" // O desde config
	return baseURL + "/webhooks/channels/" + string(channelType) + "/" + tenantID.String()
}

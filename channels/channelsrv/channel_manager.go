package channelsrv

import (
	"context"
	"fmt"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type ChannelManagerService struct {
	channelRepo channels.ChannelRepository
	adapters    map[channels.ChannelType]channels.ChannelAdapter
}

func NewChannelManager(repo channels.ChannelRepository) *ChannelManagerService {
	return &ChannelManagerService{
		channelRepo: repo,
		adapters:    make(map[channels.ChannelType]channels.ChannelAdapter),
	}
}

func (cm *ChannelManagerService) RegisterAdapter(adapter channels.ChannelAdapter) {
	cm.adapters[adapter.GetType()] = adapter
}

func (cm *ChannelManagerService) RegisterChannel(ctx context.Context, ch channels.Channel) error {
	// Validar config con adapter
	adapter, err := cm.GetAdapter(ch.Type)
	if err != nil {
		return err
	}

	if err := adapter.ValidateConfig(ch.Config); err != nil {
		return fmt.Errorf("invalid channel config: %w", err)
	}

	return cm.channelRepo.Save(ctx, ch)
}

func (cm *ChannelManagerService) SendMessage(ctx context.Context, channelID kernel.ChannelID, msg channels.OutgoingMessage) error {
	// 1. Buscar canal
	channel, err := cm.channelRepo.FindByID(ctx, channelID)
	if err != nil {
		return err
	}

	// 2. Obtener adapter
	adapter, err := cm.GetAdapter(channel.Type)
	if err != nil {
		return err
	}

	// 3. Enviar mensaje
	return adapter.SendMessage(ctx, msg)
}

func (cm *ChannelManagerService) GetAdapter(channelType channels.ChannelType) (channels.ChannelAdapter, error) {
	adapter, exists := cm.adapters[channelType]
	if !exists {
		return nil, fmt.Errorf("adapter not found for channel type: %s", channelType)
	}
	return adapter, nil
}

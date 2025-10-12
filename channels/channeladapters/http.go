package channeladapters

import (
	"context"
	"log"

	"github.com/Abraxas-365/relay/channels"
)

// TestHTTPAdapter adapter para TEST_HTTP channel
type TestHTTPAdapter struct{}

// NewTestHTTPAdapter crea un nuevo adapter de prueba
func NewTestHTTPAdapter() *TestHTTPAdapter {
	return &TestHTTPAdapter{}
}

// GetType retorna el tipo de canal
func (t *TestHTTPAdapter) GetType() channels.ChannelType {
	return channels.ChannelTypeTestHTTP
}

// SendMessage envía un mensaje (simulado para testing)
func (t *TestHTTPAdapter) SendMessage(ctx context.Context, msg channels.OutgoingMessage) error {
	log.Printf("📤 [TEST HTTP ADAPTER] Sending message to: %s", msg.RecipientID)
	log.Printf("   Content Type: %s", msg.Content.Type)
	log.Printf("   Text: %s", msg.Content.Text)

	if msg.ReplyToID != "" {
		log.Printf("   Reply to: %s", msg.ReplyToID)
	}

	if len(msg.Metadata) > 0 {
		log.Printf("   Metadata: %+v", msg.Metadata)
	}

	// Simular envío exitoso
	log.Printf("✅ [TEST HTTP ADAPTER] Message sent successfully!")

	return nil
}

// ValidateConfig valida la configuración
func (t *TestHTTPAdapter) ValidateConfig(config channels.ChannelConfig) error {
	// TEST_HTTP no tiene requisitos especiales
	if config.GetType() != channels.ChannelTypeTestHTTP {
		return channels.ErrInvalidChannelType().
			WithDetail("expected", channels.ChannelTypeTestHTTP).
			WithDetail("got", config.GetType())
	}

	return nil
}

// ProcessWebhook procesa webhooks entrantes (no usado en TEST_HTTP)
func (t *TestHTTPAdapter) ProcessWebhook(ctx context.Context, payload []byte, headers map[string]string) (*channels.IncomingMessage, error) {
	log.Printf("📥 [TEST HTTP ADAPTER] Processing webhook (payload: %d bytes)", len(payload))

	// Para testing, retornamos nil (no procesamos webhooks reales)
	return nil, nil
}

// GetFeatures retorna las características del canal
func (t *TestHTTPAdapter) GetFeatures() channels.ChannelFeatures {
	return channels.ChannelFeatures{
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
		MaxMessageLength:            10000,
		MaxAttachmentSize:           0,
		SupportedMimeTypes:          []string{},
	}
}

// TestConnection prueba la conexión (siempre exitoso para TEST_HTTP)
func (t *TestHTTPAdapter) TestConnection(ctx context.Context, config channels.ChannelConfig) error {
	log.Printf("🧪 [TEST HTTP ADAPTER] Testing connection...")
	log.Printf("✅ [TEST HTTP ADAPTER] Connection test successful!")
	return nil
}

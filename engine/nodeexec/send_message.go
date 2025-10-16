package nodeexec

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// SendMessageExecutor sends messages to specific channels
type SendMessageExecutor struct {
	channelManager channels.ChannelManager
}

var _ engine.NodeExecutor = (*SendMessageExecutor)(nil)

func NewSendMessageExecutor(channelManager channels.ChannelManager) *SendMessageExecutor {
	return &SendMessageExecutor{
		channelManager: channelManager,
	}
}

func (e *SendMessageExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()

	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract tenant_id
	tenantIDStr, ok := getStringFromInput(input, "tenant_id")
	if !ok {
		result.Success = false
		result.Error = "tenant_id not found in input"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("tenant_id required", errx.TypeValidation)
	}
	tenantID := kernel.TenantID(tenantIDStr)

	// Extract channel_id - use trigger channel if not specified
	channelIDStr, ok := node.Config["channel_id"].(string)
	if !ok || channelIDStr == "" {
		// Use the channel that triggered the workflow
		channelIDStr, ok = getStringFromInput(input, "channel_id")
		if !ok {
			// Try to get from message context
			if msgMap, ok := input["message"].(map[string]any); ok {
				if ch, ok := msgMap["channel"].(string); ok {
					channelIDStr = ch
				}
			}
		}

		if channelIDStr == "" {
			result.Success = false
			result.Error = "channel_id not found in config or input"
			result.Duration = time.Since(startTime).Milliseconds()
			return result, errx.New("channel_id required", errx.TypeValidation)
		}
	}
	channelID := kernel.ChannelID(channelIDStr)

	// Extract recipient_id - required
	recipientID, ok := node.Config["recipient_id"].(string)
	if !ok || recipientID == "" {
		// Try to get from input (e.g., original sender)
		recipientID, ok = getStringFromInput(input, "sender_id")
		if !ok {
			// Try from message context
			if msgMap, ok := input["message"].(map[string]any); ok {
				if sender, ok := msgMap["sender"].(string); ok {
					recipientID = sender
				}
			}
		}

		if recipientID == "" {
			result.Success = false
			result.Error = "recipient_id not found in config or input"
			result.Duration = time.Since(startTime).Milliseconds()
			return result, errx.New("recipient_id required", errx.TypeValidation)
		}
	}

	// Extract message content
	text, ok := node.Config["text"].(string)
	if !ok || text == "" {
		result.Success = false
		result.Error = "text is required"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("text is required", errx.TypeValidation)
	}

	// Build message content
	messageContent := channels.MessageContent{
		Type: "text",
		Text: text,
	}

	// Handle attachments if provided
	if attachments, ok := node.Config["attachments"].([]any); ok {
		strAttachments := make([]string, 0, len(attachments))
		for _, att := range attachments {
			if attStr, ok := att.(string); ok {
				strAttachments = append(strAttachments, attStr)
			}
		}
		messageContent.Attachments = strAttachments
	}

	// Build metadata
	metadata := make(map[string]any)
	if meta, ok := node.Config["metadata"].(map[string]any); ok {
		metadata = meta
	}
	metadata["workflow_node_id"] = node.ID
	metadata["workflow_node_name"] = node.Name
	metadata["timestamp"] = time.Now().Unix()

	// Create outgoing message
	outgoingMsg := channels.OutgoingMessage{
		RecipientID: recipientID,
		Content:     messageContent,
		Metadata:    metadata,
	}

	// Send the message
	log.Printf("📤 Sending message to recipient %s via channel %s", recipientID, channelID.String())
	if err := e.channelManager.SendMessage(ctx, tenantID, channelID, outgoingMsg); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to send message: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.Wrap(err, "failed to send message", errx.TypeInternal)
	}

	// Success
	result.Success = true
	result.Output["sent"] = true
	result.Output["channel_id"] = channelID.String()
	result.Output["recipient_id"] = recipientID
	result.Output["message_text"] = text
	result.Duration = time.Since(startTime).Milliseconds()

	log.Printf("✅ Message sent successfully to %s", recipientID)
	return result, nil
}

func (e *SendMessageExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeSendMessage
}

func (e *SendMessageExecutor) ValidateConfig(config map[string]any) error {
	// text is required
	if _, ok := config["text"].(string); !ok {
		return errx.New("text is required for send_message node", errx.TypeValidation)
	}

	// channel_id and recipient_id are optional (can come from context)

	return nil
}

// Helper function to extract string from nested input
func getStringFromInput(input map[string]any, key string) (string, bool) {
	if val, ok := input[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

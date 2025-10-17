package nodeexec

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/logx"
	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

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
	logx.Info("Send Message Executor")
	startTime := time.Now()

	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract tenant_id
	tenantIDStr := extractStringFromInput(input, "tenant_id")
	if tenantIDStr == "" {
		result.Success = false
		result.Error = "tenant_id not found in input"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("tenant_id required", errx.TypeValidation)
	}
	tenantID := kernel.TenantID(tenantIDStr)

	// Extract channel_id
	channelIDStr, ok := node.Config["channel_id"].(string)
	if !ok || channelIDStr == "" {
		// Try from trigger data
		if trigger, ok := input["trigger"].(map[string]any); ok {
			if chID, ok := trigger["channel_id"].(string); ok {
				channelIDStr = chID
			}
		}
		if channelIDStr == "" {
			result.Success = false
			result.Error = "channel_id not found in config or trigger"
			result.Duration = time.Since(startTime).Milliseconds()
			return result, errx.New("channel_id required", errx.TypeValidation)
		}
	}
	channelID := kernel.ChannelID(channelIDStr)

	// Extract recipient_id
	recipientID, ok := node.Config["recipient_id"].(string)
	if !ok || recipientID == "" {
		// Try from trigger
		if trigger, ok := input["trigger"].(map[string]any); ok {
			if sender, ok := trigger["sender_id"].(string); ok {
				recipientID = sender
			}
		}
		if recipientID == "" {
			result.Success = false
			result.Error = "recipient_id not found"
			result.Duration = time.Since(startTime).Milliseconds()
			return result, errx.New("recipient_id required", errx.TypeValidation)
		}
	}

	// Extract message text
	text, ok := node.Config["text"].(string)
	if !ok || text == "" {
		result.Success = false
		result.Error = "text is required"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("text is required", errx.TypeValidation)
	}

	messageContent := channels.MessageContent{
		Type: "text",
		Text: text,
	}

	// ✅ FIX: Handle attachments properly
	if attachments, ok := node.Config["attachments"].([]any); ok {
		parsedAttachments := make([]channels.Attachment, 0, len(attachments))

		for _, att := range attachments {
			// Option 1: If attachment is a string (URL), convert to Attachment struct
			if attStr, ok := att.(string); ok {
				parsedAttachments = append(parsedAttachments, channels.Attachment{
					Type: "document", // or detect from URL/extension
					URL:  attStr,
				})
			}

			// Option 2: If attachment is already a map/struct
			if attMap, ok := att.(map[string]any); ok {
				attachment := channels.Attachment{}

				if attType, ok := attMap["type"].(string); ok {
					attachment.Type = attType
				}
				if url, ok := attMap["url"].(string); ok {
					attachment.URL = url
				}
				if mimeType, ok := attMap["mime_type"].(string); ok {
					attachment.MimeType = mimeType
				}
				if filename, ok := attMap["filename"].(string); ok {
					attachment.Filename = filename
				}
				if caption, ok := attMap["caption"].(string); ok {
					attachment.Caption = caption
				}

				parsedAttachments = append(parsedAttachments, attachment)
			}
		}

		messageContent.Attachments = parsedAttachments
	}

	metadata := make(map[string]any)
	if meta, ok := node.Config["metadata"].(map[string]any); ok {
		metadata = meta
	}
	metadata["workflow_node_id"] = node.ID
	metadata["workflow_node_name"] = node.Name
	metadata["timestamp"] = time.Now().Unix()

	outgoingMsg := channels.OutgoingMessage{
		RecipientID: recipientID,
		Content:     messageContent,
		Metadata:    metadata,
	}

	log.Printf("💬 Sending message to %s via channel %s", recipientID, channelID.String())
	if err := e.channelManager.SendMessage(ctx, tenantID, channelID, outgoingMsg); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to send message: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.Wrap(err, "failed to send message", errx.TypeInternal)
	}

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
	if _, ok := config["text"].(string); !ok {
		return errx.New("text is required for send_message node", errx.TypeValidation)
	}
	return nil
}

func extractStringFromInput(input map[string]any, key string) string {
	if val, ok := input[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

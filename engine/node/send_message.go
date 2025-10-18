package node

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type SendMessageExecutor struct {
	channelManager channels.ChannelManager
	evaluator      engine.ExpressionEvaluator
}

func NewSendMessageExecutor(
	channelManager channels.ChannelManager,
	evaluator engine.ExpressionEvaluator,
) *SendMessageExecutor {
	return &SendMessageExecutor{
		channelManager: channelManager,
		evaluator:      evaluator,
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

	// Create resolver
	resolver := NewFieldResolver(input, node.Config, e.evaluator)

	// Get tenant ID
	tenantID, err := resolver.GetTenantID()
	if err != nil {
		result.Success = false
		result.Error = "tenant_id not found"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// Resolve fields (priority: config -> webhook -> error)
	channelIDStr := resolver.GetString("channel_id", "")
	if channelIDStr == "" {
		result.Success = false
		result.Error = "channel_id is required"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, fmt.Errorf("channel_id required")
	}

	recipientID := resolver.GetString("recipient_id", "")
	if recipientID == "" {
		// Try sender_id as fallback
		recipientID = resolver.GetString("sender_id", "")
	}
	if recipientID == "" {
		result.Success = false
		result.Error = "recipient_id is required"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, fmt.Errorf("recipient_id required")
	}

	text := resolver.GetString("text", "")
	if text == "" {
		text = resolver.GetString("message", "") // Try 'message' as fallback
	}
	if text == "" {
		result.Success = false
		result.Error = "text is required"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, fmt.Errorf("text required")
	}

	messageType := resolver.GetString("message_type", "text")

	log.Printf("💬 Sending message to %s via channel %s", recipientID, channelIDStr)
	log.Printf("   📝 Text: %s", truncateString(text, 50))

	// Build message
	messageContent := channels.MessageContent{
		Type: messageType,
		Text: text,
	}

	// Handle attachments
	if attachments := resolver.GetArray("attachments"); len(attachments) > 0 {
		parsedAttachments := make([]channels.Attachment, 0, len(attachments))
		for _, att := range attachments {
			if attStr, ok := att.(string); ok {
				parsedAttachments = append(parsedAttachments, channels.Attachment{
					Type: "document",
					URL:  attStr,
				})
			} else if attMap, ok := att.(map[string]any); ok {
				attachment := channels.Attachment{
					Type:     getStringFromMap(attMap, "type", "document"),
					URL:      getStringFromMap(attMap, "url", ""),
					MimeType: getStringFromMap(attMap, "mime_type", ""),
					Filename: getStringFromMap(attMap, "filename", ""),
					Caption:  getStringFromMap(attMap, "caption", ""),
				}
				parsedAttachments = append(parsedAttachments, attachment)
			}
		}
		messageContent.Attachments = parsedAttachments
	}

	// Send message
	outgoingMsg := channels.OutgoingMessage{
		RecipientID: recipientID,
		Content:     messageContent,
		Metadata: map[string]any{
			"workflow_node_id":   node.ID,
			"workflow_node_name": node.Name,
			"timestamp":          time.Now().Unix(),
		},
	}

	if err := e.channelManager.SendMessage(ctx, tenantID, kernel.ChannelID(channelIDStr), outgoingMsg); err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("failed to send message: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	result.Success = true
	result.Output["sent"] = true
	result.Output["channel_id"] = channelIDStr
	result.Output["recipient_id"] = recipientID
	result.Output["message_text"] = text
	result.Duration = time.Since(startTime).Milliseconds()

	log.Printf("✅ Message sent successfully")
	return result, nil
}

func (e *SendMessageExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeSendMessage
}

func (e *SendMessageExecutor) ValidateConfig(config map[string]any) error {
	// Basic validation - text is required in config or will be from webhook
	return nil
}

func getStringFromMap(m map[string]any, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

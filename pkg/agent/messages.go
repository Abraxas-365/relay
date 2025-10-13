package agent

import (
	"encoding/json"
	"time"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// AgentMessage represents a message in a chat session
type AgentMessage struct {
	ID               string           `db:"id" json:"id"`
	SessionID        kernel.SessionID `db:"session_id" json:"session_id"`
	Role             string           `db:"role" json:"role"`
	Content          *string          `db:"content" json:"content,omitempty"`
	Name             *string          `db:"name" json:"name,omitempty"`
	FunctionCall     map[string]any   `db:"function_call" json:"function_call,omitempty"`
	ToolCalls        []map[string]any `db:"-" json:"tool_calls,omitempty"`
	ToolCallID       *string          `db:"tool_call_id" json:"tool_call_id,omitempty"`
	Metadata         map[string]any   `db:"metadata" json:"metadata"`
	MessageType      string           `db:"message_type" json:"message_type"`
	ProcessingTimeMs *int             `db:"processing_time_ms" json:"processing_time_ms,omitempty"`
	ModelUsed        *string          `db:"model_used" json:"model_used,omitempty"`
	TokensUsed       *int             `db:"tokens_used" json:"tokens_used,omitempty"`
	CreatedAt        time.Time        `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time        `db:"updated_at" json:"updated_at"`
}

// Message type constants
const (
	MessageTypeText     = "text"
	MessageTypeImage    = "image"
	MessageTypeDocument = "document"
	MessageTypeAudio    = "audio"
	MessageTypeVideo    = "video"
	MessageTypeTemplate = "template"
)

// ToLLMMessage converts AgentMessage to llm.Message
func (m *AgentMessage) ToLLMMessage() llm.Message {
	msg := llm.Message{
		Role: m.Role,
	}

	if m.Content != nil {
		msg.Content = *m.Content
	}

	if m.Name != nil {
		msg.Name = *m.Name
	}

	if m.Role == llm.RoleTool && m.ToolCallID != nil {
		msg.ToolCallID = *m.ToolCallID
	}

	// Convert function_call if present
	if m.FunctionCall != nil {
		// Since storexpostgres.JSONB is map[string]any, we need to convert it to the expected structure
		if name, ok := m.FunctionCall["name"].(string); ok {
			if arguments, ok := m.FunctionCall["arguments"].(string); ok {
				msg.FunctionCall = &llm.FunctionCall{
					Name:      name,
					Arguments: arguments,
				}
			}
		}
	}

	// Convert tool_calls if present
	if m.ToolCalls != nil {
		// Convert the JSONB to []llm.ToolCall
		// First, marshal it back to JSON bytes then unmarshal to the correct type
		if jsonBytes, err := json.Marshal(m.ToolCalls); err == nil {
			var toolCalls []llm.ToolCall
			if err := json.Unmarshal(jsonBytes, &toolCalls); err == nil {
				msg.ToolCalls = toolCalls
			}
		}
	}

	// Convert metadata if present
	if m.Metadata != nil {
		// Since it's already map[string]any, we can assign it directly
		msg.Metadata = map[string]any(m.Metadata)
	}

	return msg
}

// ToLLMMessages converts a slice of AgentMessage to []llm.Message
func ToLLMMessages(messages []AgentMessage) []llm.Message {
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMessages[i] = msg.ToLLMMessage()
	}
	return llmMessages
}

// CreateMessageRequest represents the request to create a message
type CreateMessageRequest struct {
	SessionID        kernel.SessionID `json:"session_id" validatex:"required,uuid"`
	Role             string           `json:"role" validatex:"required"`
	Content          *string          `json:"content,omitempty" validatex:"max=10000"`
	Name             *string          `json:"name,omitempty" validatex:"max=255"`
	FunctionCall     map[string]any   `json:"function_call,omitempty"`
	ToolCalls        []map[string]any `json:"tool_calls,omitempty"`
	ToolCallID       *string          `json:"tool_call_id,omitempty" validatex:"max=255"`
	Metadata         map[string]any   `json:"metadata,omitempty"`
	MessageType      *string          `json:"message_type,omitempty" validatex:"max=50"`
	ProcessingTimeMs *int             `json:"processing_time_ms,omitempty" validatex:"min=0"`
	ModelUsed        *string          `json:"model_used,omitempty" validatex:"max=100"`
	TokensUsed       *int             `json:"tokens_used,omitempty" validatex:"min=0"`
}

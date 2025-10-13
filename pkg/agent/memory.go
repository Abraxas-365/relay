package agent

import (
	"context"
	"fmt"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/memoryx"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

// sessionMemoryImpl implements the SessionMemory interface
type sessionMemoryImpl struct {
	sessionID       kernel.SessionID
	systemPrompt    string
	repository      AgentChatRepository
	ctx             context.Context
	contextMessages []llm.Message
}

// NewSessionMemory creates a new SessionMemory instance
func NewSessionMemory(
	ctx context.Context,
	sessionID kernel.SessionID,
	systemPrompt string,
	contexMessages []llm.Message,
	repository AgentChatRepository) memoryx.Memory {
	return &sessionMemoryImpl{
		sessionID:       sessionID,
		systemPrompt:    systemPrompt,
		repository:      repository,
		ctx:             ctx,
		contextMessages: contexMessages,
	}
}

func (sm *sessionMemoryImpl) Messages() ([]llm.Message, error) {
	// Get all messages for the session
	messages, err := sm.repository.GetAllMessagesBySession(sm.ctx, sm.sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}

	// Convert to LLM messages
	llmMessages := ToLLMMessages(messages)

	if len(sm.contextMessages) > 0 {
		llmMessages = append(llmMessages, sm.contextMessages...)
	}
	if sm.systemPrompt != "" {
		systemMsg := llm.NewSystemMessage(sm.systemPrompt)
		llmMessages = append([]llm.Message{systemMsg}, llmMessages...)
	}

	return llmMessages, nil
}

func (sm *sessionMemoryImpl) Add(message llm.Message) error {

	if message.Role == llm.RoleTool {

		// VALIDATE: Tool messages must have a ToolCallID
		if message.ToolCallID == "" {
			return fmt.Errorf("tool message must have a tool_call_id")
		}
	}

	req := CreateMessageRequest{
		SessionID: sm.sessionID,
		Role:      message.Role,
		Content:   &message.Content,
		Name:      &message.Name,
		Metadata:  message.Metadata,
	}

	// CRITICAL: Ensure ToolCallID is set for tool messages
	if message.Role == llm.RoleTool && message.ToolCallID != "" {
		req.ToolCallID = &message.ToolCallID
	}

	// Handle function call
	if message.FunctionCall != nil {
		req.FunctionCall = map[string]any{
			"name":      message.FunctionCall.Name,
			"arguments": message.FunctionCall.Arguments,
		}
	}

	// Handle tool calls (only for assistant messages)
	if len(message.ToolCalls) > 0 {

		toolCalls := make([]map[string]any, len(message.ToolCalls))
		for i, tc := range message.ToolCalls {
			toolCalls[i] = map[string]any{
				"id":   tc.ID,
				"type": tc.Type,
				"function": map[string]any{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			}
		}
		req.ToolCalls = toolCalls
	}

	// Create the message
	_, err := sm.repository.CreateMessage(sm.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to add message to session: %w", err)
	}

	return nil
}

// Clear resets the conversation but keeps the system prompt
func (sm *sessionMemoryImpl) Clear() error {
	// Clear all messages except system messages
	err := sm.repository.ClearSessionMessages(sm.ctx, sm.sessionID, true)
	if err != nil {
		return fmt.Errorf("failed to clear session messages: %w", err)
	}

	return nil
}

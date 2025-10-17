package agent

import (
	"context"
	"log"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type SessionMemory struct {
	ctx            context.Context
	tenantID       kernel.TenantID
	sessionID      kernel.SessionID
	systemPrompt   string
	contextMsgs    []llm.Message
	repo           AgentChatRepository
	cachedMessages []llm.Message
}

func NewSessionMemory(
	ctx context.Context,
	tenantID kernel.TenantID,
	sessionID kernel.SessionID,
	systemPrompt string,
	contextMessages []llm.Message,
	repo AgentChatRepository,
) *SessionMemory {
	return &SessionMemory{
		ctx:          ctx,
		tenantID:     tenantID,
		sessionID:    sessionID,
		systemPrompt: systemPrompt,
		contextMsgs:  contextMessages,
		repo:         repo,
	}
}

func (m *SessionMemory) Messages() ([]llm.Message, error) {
	if m.cachedMessages != nil {
		return m.cachedMessages, nil
	}

	messages := []llm.Message{}

	if m.systemPrompt != "" {
		messages = append(messages, llm.NewSystemMessage(m.systemPrompt))
	}

	if len(m.contextMsgs) > 0 {
		messages = append(messages, m.contextMsgs...)
	}

	storedMessages, err := m.repo.GetAllMessagesBySession(m.ctx, m.sessionID)
	if err != nil {
		log.Printf("⚠️  Failed to load stored messages: %v", err)
		m.cachedMessages = messages
		return messages, nil
	}

	for _, msg := range storedMessages {
		llmMsg := convertAgentMessageToLLM(&msg)
		if llmMsg != nil {
			messages = append(messages, *llmMsg)
		}
	}

	m.cachedMessages = messages
	return messages, nil
}

func (m *SessionMemory) Add(msg llm.Message) error {
	m.cachedMessages = nil

	req := CreateMessageRequest{
		TenantID:  m.tenantID,
		SessionID: m.sessionID,
		Role:      msg.Role,
		Content:   &msg.Content,
	}

	if msg.Name != "" {
		req.Name = &msg.Name
	}

	// ✅ FIX: Properly convert tool calls from llm.ToolCall to map[string]any
	if len(msg.ToolCalls) > 0 {
		toolCalls := make([]map[string]any, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
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

	// ✅ FIX: Properly convert function call from llm.FunctionCall to map[string]any
	if msg.FunctionCall != nil {
		req.FunctionCall = map[string]any{
			"name":      msg.FunctionCall.Name,
			"arguments": msg.FunctionCall.Arguments,
		}
	}

	if msg.ToolCallID != "" {
		req.ToolCallID = &msg.ToolCallID
	}

	_, err := m.repo.CreateMessage(m.ctx, req)
	return err
}

func (m *SessionMemory) Clear() error {
	m.cachedMessages = nil
	return m.repo.ClearSessionMessages(m.ctx, m.sessionID, true)
}

// ✅ FIX: Properly convert map[string]any to llm.FunctionCall and llm.ToolCall
func convertAgentMessageToLLM(msg *AgentMessage) *llm.Message {
	if msg == nil {
		return nil
	}

	llmMsg := &llm.Message{
		Role: msg.Role,
	}

	if msg.Content != nil {
		llmMsg.Content = *msg.Content
	}

	if msg.Name != nil {
		llmMsg.Name = *msg.Name
	}

	if msg.ToolCallID != nil {
		llmMsg.ToolCallID = *msg.ToolCallID
	}

	// ✅ FIX: Convert tool calls from map[string]any to llm.ToolCall
	if msg.ToolCalls != nil {
		toolCalls := make([]llm.ToolCall, 0, len(msg.ToolCalls))
		for _, tc := range msg.ToolCalls {
			toolCall := llm.ToolCall{}

			// Extract ID
			if id, ok := tc["id"].(string); ok {
				toolCall.ID = id
			}

			// Extract Type
			if tcType, ok := tc["type"].(string); ok {
				toolCall.Type = tcType
			}

			// ✅ Extract and convert function from map[string]any to llm.FunctionCall
			if fn, ok := tc["function"].(map[string]any); ok {
				functionCall := llm.FunctionCall{}

				if name, ok := fn["name"].(string); ok {
					functionCall.Name = name
				}

				if args, ok := fn["arguments"].(string); ok {
					functionCall.Arguments = args
				}

				toolCall.Function = functionCall
			}

			toolCalls = append(toolCalls, toolCall)
		}
		llmMsg.ToolCalls = toolCalls
	}

	// ✅ FIX: Convert function call from map[string]any to llm.FunctionCall
	if msg.FunctionCall != nil {
		functionCall := llm.FunctionCall{}

		if name, ok := msg.FunctionCall["name"].(string); ok {
			functionCall.Name = name
		}

		if args, ok := msg.FunctionCall["arguments"].(string); ok {
			functionCall.Arguments = args
		}

		llmMsg.FunctionCall = &functionCall
	}

	return llmMsg
}

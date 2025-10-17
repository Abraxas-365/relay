package nodeexec

import (
	"context"
	"fmt"
	"log"
	"maps"
	"time"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/agent"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type AIAgentExecutor struct {
	agentChatRepo agent.AgentChatRepository
}

var _ engine.NodeExecutor = (*AIAgentExecutor)(nil)

func NewAIAgentExecutor(
	agentChatRepo agent.AgentChatRepository,
) *AIAgentExecutor {
	return &AIAgentExecutor{
		agentChatRepo: agentChatRepo,
	}
}

func (e *AIAgentExecutor) Execute(ctx context.Context, node engine.WorkflowNode, input map[string]any) (*engine.NodeResult, error) {
	startTime := time.Now()

	result := &engine.NodeResult{
		NodeID:    node.ID,
		NodeName:  node.Name,
		Timestamp: startTime,
		Output:    make(map[string]any),
	}

	// Extract and validate AI agent config
	aiConfig, err := engine.ExtractAIAgentConfig(node.Config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("invalid AI agent config: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// ✅ NEW: Check for explicit prompt in config (for scheduled/generation workflows)
	var userMessage string
	if promptFromConfig, ok := node.Config["prompt"].(string); ok && promptFromConfig != "" {
		userMessage = promptFromConfig
		log.Printf("🤖 AI Agent using prompt from config: %s", userMessage)
	} else if userPrompt, ok := node.Config["user_prompt"].(string); ok && userPrompt != "" {
		userMessage = userPrompt
		log.Printf("🤖 AI Agent using user_prompt from config: %s", userMessage)
	} else {
		// Extract from trigger (existing behavior)
		userMessage = e.extractUserMessage(input)
	}

	// ✅ If still no message, use a default generation prompt
	if userMessage == "" {
		userMessage = "Generate a creative message." // Fallback
		log.Printf("⚠️  No user message or prompt found, using default: %s", userMessage)
	}

	// Extract tenant_id
	tenantID := e.extractTenantID(input)
	if tenantID == "" {
		// ✅ For scheduled workflows without explicit tenant_id, use from input
		if tid, ok := input["tenant_id"].(string); ok {
			tenantID = tid
		}
	}

	// Extract conversation ID (for memory persistence)
	conversationID := e.extractConversationID(input)

	log.Printf("🤖 AI Agent '%s' processing with model: %s (memory: %v, conversation_id: %s, tenant: %s)",
		node.Name, aiConfig.Model, aiConfig.UseMemory, conversationID, tenantID)

	var responseText string
	var metadata map[string]any

	// Decide execution mode based on UseMemory flag and conversation ID
	if aiConfig.UseMemory && conversationID != "" && tenantID != "" {
		responseText, metadata, err = e.executeWithAgent(ctx, aiConfig, userMessage, tenantID, conversationID, input)
	} else {
		responseText, metadata, err = e.executeWithLLM(ctx, aiConfig, userMessage, input)
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("AI execution failed: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// Store results
	result.Success = true
	result.Output["ai_response"] = responseText
	result.Output["response"] = responseText
	result.Output["model"] = aiConfig.Model
	result.Output["provider"] = aiConfig.Provider
	result.Output["use_memory"] = aiConfig.UseMemory
	result.Output["conversation_id"] = conversationID
	result.Output["tenant_id"] = tenantID

	// Add metadata
	if metadata != nil {
		maps.Copy(result.Output, metadata)
	}

	result.Duration = time.Since(startTime).Milliseconds()
	log.Printf("✅ AI Agent '%s' completed in %dms", node.Name, result.Duration)

	return result, nil
}

// executeWithLLM uses regular LLM without persistent memory
func (e *AIAgentExecutor) executeWithLLM(
	ctx context.Context,
	config *engine.AIAgentConfig,
	userMessage string,
	input map[string]any,
) (string, map[string]any, error) {
	// Get LLM client
	client := config.GetLLMClient()

	// Build messages
	messages := []llm.Message{
		llm.NewSystemMessage(config.SystemPrompt),
	}

	// Optionally add context messages from input
	contextMessages := e.buildContextMessagesFromInput(input)
	messages = append(messages, contextMessages...)

	// Add user message
	messages = append(messages, llm.NewUserMessage(userMessage))

	// Call LLM
	response, err := client.Chat(ctx, messages, config.GetLLMOptions()...)
	if err != nil {
		return "", nil, errx.Wrap(err, "LLM call failed", errx.TypeInternal)
	}

	// Build metadata
	metadata := map[string]any{
		"mode":          "llm",
		"finish_reason": response.Message.Content,
		"tokens_used": map[string]any{
			"prompt":     response.Usage.PromptTokens,
			"completion": response.Usage.CompletionTokens,
			"total":      response.Usage.TotalTokens,
		},
	}

	return response.Message.Content, metadata, nil
}

// ✅ FIX: executeWithAgent now accepts tenantID parameter
func (e *AIAgentExecutor) executeWithAgent(
	ctx context.Context,
	config *engine.AIAgentConfig,
	userMessage string,
	tenantID string, // ✅ ADDED
	conversationID string,
	input map[string]any,
) (string, map[string]any, error) {
	// Create LLM client
	llmClient := config.GetLLMClient()

	// Build context messages from input
	contextMessages := e.buildContextMessagesFromInput(input)

	// ✅ FIX: Create SessionMemory with tenant_id
	memory := agent.NewSessionMemory(
		ctx,
		kernel.TenantID(tenantID), // ✅ ADDED
		kernel.SessionID(conversationID),
		config.SystemPrompt,
		contextMessages,
		e.agentChatRepo,
	)

	// Create agent options
	agentOptions := []agentx.AgentOption{
		agentx.WithOptions(config.GetLLMOptions()...),
		agentx.WithMaxAutoIterations(config.GetMaxAutoIterations()),
		agentx.WithMaxTotalIterations(config.GetMaxTotalIterations()),
	}

	// TODO: Add tools support when implemented
	// if len(config.Tools) > 0 {
	//     toolxClient := e.createToolxClient(ctx, config.Tools)
	//     agentOptions = append(agentOptions, agentx.WithTools(toolxClient))
	// }

	// Create agent
	agentInstance := agentx.New(llmClient, memory, agentOptions...)

	// Run agent with user input
	response, err := agentInstance.Run(ctx, userMessage)
	if err != nil {
		return "", nil, errx.Wrap(err, "agent execution failed", errx.TypeInternal)
	}

	// Build metadata
	metadata := map[string]any{
		"mode":            "agent",
		"conversation_id": conversationID,
		"tenant_id":       tenantID,
		"has_memory":      true,
	}

	return response, metadata, nil
}

// ✅ NEW: extractTenantID gets tenant_id from input
func (e *AIAgentExecutor) extractTenantID(input map[string]any) string {
	// Try direct tenant_id field
	if tenantID, ok := input["tenant_id"].(string); ok && tenantID != "" {
		return tenantID
	}

	// Try from trigger data
	if trigger, ok := input["trigger"].(map[string]any); ok {
		if tenantID, ok := trigger["tenant_id"].(string); ok && tenantID != "" {
			return tenantID
		}
	}

	// Try from metadata
	if metadata, ok := input["metadata"].(map[string]any); ok {
		if tenantID, ok := metadata["tenant_id"].(string); ok && tenantID != "" {
			return tenantID
		}
	}

	return ""
}

// extractUserMessage gets the user message from input
func (e *AIAgentExecutor) extractUserMessage(input map[string]any) string {
	// Try trigger.text (n8n-style webhook trigger)
	if trigger, ok := input["trigger"].(map[string]any); ok {
		if text, ok := trigger["text"].(string); ok && text != "" {
			return text
		}
		if msg, ok := trigger["message"].(map[string]any); ok {
			if text, ok := msg["text"].(string); ok && text != "" {
				return text
			}
		}
	}

	// Try direct text field
	if text, ok := input["text"].(string); ok && text != "" {
		return text
	}

	// Try message_text
	if text, ok := input["message_text"].(string); ok && text != "" {
		return text
	}

	// Try input.body.message (from HTTP request)
	if body, ok := input["body"].(map[string]any); ok {
		if text, ok := body["message"].(string); ok && text != "" {
			return text
		}
	}

	return ""
}

// extractConversationID gets a unique conversation identifier for memory persistence
func (e *AIAgentExecutor) extractConversationID(input map[string]any) string {
	// Try conversation_id from config (highest priority)
	if convID, ok := input["conversation_id"].(string); ok && convID != "" {
		return convID
	}

	// Try from trigger data
	if trigger, ok := input["trigger"].(map[string]any); ok {
		if convID, ok := trigger["conversation_id"].(string); ok && convID != "" {
			return convID
		}

		// Try user_id as fallback (for user-specific conversations)
		if userID, ok := trigger["user_id"].(string); ok && userID != "" {
			return userID
		}

		// Try sender_id (for messaging platforms)
		if senderID, ok := trigger["sender_id"].(string); ok && senderID != "" {
			return senderID
		}

		// Try channel + sender combination
		if channelID, ok := trigger["channel_id"].(string); ok {
			if senderID, ok := trigger["sender_id"].(string); ok {
				return fmt.Sprintf("%s:%s", channelID, senderID)
			}
		}
	}

	// Try user_id from input
	if userID, ok := input["user_id"].(string); ok && userID != "" {
		return userID
	}

	// Try thread_id (for threaded conversations)
	if threadID, ok := input["thread_id"].(string); ok && threadID != "" {
		return threadID
	}

	// No conversation ID found - agent will run without memory
	return ""
}

// buildContextMessagesFromInput creates context messages from workflow input
func (e *AIAgentExecutor) buildContextMessagesFromInput(input map[string]any) []llm.Message {
	var contextMessages []llm.Message

	// Add context from previous nodes if available
	if context, ok := input["context"].(map[string]any); ok {
		// Example: Add user information
		if userInfo, ok := context["user_info"].(string); ok && userInfo != "" {
			contextMessages = append(contextMessages,
				llm.NewSystemMessage(fmt.Sprintf("User information: %s", userInfo)))
		}

		// Example: Add conversation history from previous execution
		if history, ok := context["history"].([]any); ok && len(history) > 0 {
			for _, item := range history {
				if historyMap, ok := item.(map[string]any); ok {
					role, _ := historyMap["role"].(string)
					content, _ := historyMap["content"].(string)

					switch role {
					case "user":
						contextMessages = append(contextMessages, llm.NewUserMessage(content))
					case "assistant":
						contextMessages = append(contextMessages, llm.NewAssistantMessage(content))
					case "system":
						contextMessages = append(contextMessages, llm.NewSystemMessage(content))
					}
				}
			}
		}
	}

	// Add metadata from trigger if available
	if trigger, ok := input["trigger"].(map[string]any); ok {
		if metadata, ok := trigger["metadata"].(map[string]any); ok {
			// Add relevant metadata as context
			if language, ok := metadata["language"].(string); ok && language != "" {
				contextMessages = append(contextMessages,
					llm.NewSystemMessage(fmt.Sprintf("User language preference: %s", language)))
			}
		}
	}

	return contextMessages
}

func (e *AIAgentExecutor) SupportsType(nodeType engine.NodeType) bool {
	return nodeType == engine.NodeTypeAIAgent
}

func (e *AIAgentExecutor) ValidateConfig(config map[string]any) error {
	aiConfig, err := engine.ExtractAIAgentConfig(config)
	if err != nil {
		return err
	}
	return aiConfig.Validate()
}

package nodeexec

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/agent"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"maps"
)

// AIAgentExecutor executes AI agent nodes with optional memory and tools
type AIAgentExecutor struct {
	agentChatRepo agent.AgentChatRepository
	// toolManager can be added here when tools are implemented
}

var _ engine.NodeExecutor = (*AIAgentExecutor)(nil)

func NewAIAgentExecutor(agentChatRepo agent.AgentChatRepository) *AIAgentExecutor {
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

	// Extract user message
	userMessage := e.extractUserMessage(input)
	if userMessage == "" {
		result.Success = false
		result.Error = "no user message found in input"
		result.Duration = time.Since(startTime).Milliseconds()
		return result, errx.New("no user message found", errx.TypeValidation)
	}

	// Get session if available
	session := e.extractSession(input)

	log.Printf("🤖 AI Agent '%s' processing with model: %s (memory: %v)",
		node.Name, aiConfig.Model, aiConfig.UseMemory)

	var responseText string
	var metadata map[string]any

	// Decide execution mode based on UseMemory flag
	if aiConfig.UseMemory && session != nil {
		responseText, metadata, err = e.executeWithAgent(ctx, aiConfig, userMessage, session, input)
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
	result.Output["response"] = responseText // For compatibility with SendMessage
	result.Output["model"] = aiConfig.Model
	result.Output["provider"] = aiConfig.Provider
	result.Output["use_memory"] = aiConfig.UseMemory

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

	// Optionally add context messages
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
	}

	metadata["tokens_used"] = map[string]any{
		"prompt":     response.Usage.PromptTokens,
		"completion": response.Usage.CompletionTokens,
		"total":      response.Usage.TotalTokens,
	}

	return response.Message.Content, metadata, nil
}

// executeWithAgent uses agentx with persistent SessionMemory
func (e *AIAgentExecutor) executeWithAgent(
	ctx context.Context,
	config *engine.AIAgentConfig,
	userMessage string,
	session *engine.Session,
	input map[string]any,
) (string, map[string]any, error) {
	// Create LLM client
	llmClient := config.GetLLMClient()

	// Build context messages from session
	contextMessages := e.buildContextMessagesFromSession(session, input)

	// Create SessionMemory with persistent storage
	memory := agent.NewSessionMemory(
		ctx,
		session.ID,
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
		"mode":       "agent",
		"session_id": session.ID.String(),
		"has_memory": true,
	}

	return response, metadata, nil
}

// extractUserMessage gets the user message from input
func (e *AIAgentExecutor) extractUserMessage(input map[string]any) string {
	// Try to get from message.text
	if msgMap, ok := input["message"].(map[string]any); ok {
		if text, ok := msgMap["text"].(string); ok && text != "" {
			return text
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

	return ""
}

// extractSession gets the session from input
func (e *AIAgentExecutor) extractSession(input map[string]any) *engine.Session {
	// This is a simplified extraction - in production, you might want to
	// reconstruct the full Session object from the input
	if sessionMap, ok := input["session"].(map[string]any); ok {
		// Extract session ID
		sessionID := ""
		if id, ok := sessionMap["id"].(string); ok {
			sessionID = id
		}

		if sessionID == "" {
			return nil
		}

		// Create a minimal session object
		// In production, you might want to fully reconstruct it
		return &engine.Session{
			ID:           kernel.SessionID(sessionID),
			Context:      make(map[string]any),
			CurrentState: "",
		}
	}

	return nil
}

// buildContextMessagesFromInput creates context messages from workflow input
func (e *AIAgentExecutor) buildContextMessagesFromInput(input map[string]any) []llm.Message {
	var contextMessages []llm.Message

	// Add any relevant context from input
	// Example: Add information about the current workflow state

	return contextMessages
}

// buildContextMessagesFromSession creates context messages from session data
func (e *AIAgentExecutor) buildContextMessagesFromSession(session *engine.Session, input map[string]any) []llm.Message {
	var contextMessages []llm.Message

	// Add session state context if available
	if session.CurrentState != "" {
		// You can optionally add session state as context
		// contextMsg := llm.NewSystemMessage(fmt.Sprintf("Current session state: %s", session.CurrentState))
		// contextMessages = append(contextMessages, contextMsg)
	}

	// Add any custom context from session.Context
	if session.Context != nil {
		// Example: If you store user preferences or other context
		// if userInfo, ok := session.Context["user_info"].(string); ok {
		//     contextMessages = append(contextMessages, llm.NewSystemMessage(userInfo))
		// }
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


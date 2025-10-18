package node

import (
	"context"
	"fmt"
	"log"
	"maps"
	"time"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/agent"
	"github.com/Abraxas-365/relay/pkg/kernel"
)

type AIAgentExecutor struct {
	agentChatRepo agent.AgentChatRepository
	evaluator     engine.ExpressionEvaluator
}

func NewAIAgentExecutor(
	agentChatRepo agent.AgentChatRepository,
	evaluator engine.ExpressionEvaluator,
) *AIAgentExecutor {
	return &AIAgentExecutor{
		agentChatRepo: agentChatRepo,
		evaluator:     evaluator,
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

	// Extract AI config
	aiConfig, err := engine.ExtractAIAgentConfig(node.Config)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("invalid AI agent config: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	// Create resolver
	resolver := NewFieldResolver(input, node.Config, e.evaluator)

	// Get user message (priority: config prompt -> webhook message -> default)
	userMessage := resolver.GetString("prompt", "")
	if userMessage == "" {
		userMessage = resolver.GetString("user_prompt", "")
	}
	if userMessage == "" {
		userMessage = resolver.GetString("message", "")
	}
	if userMessage == "" {
		userMessage = resolver.GetString("text", "")
	}
	if userMessage == "" {
		userMessage = "Generate a creative response."
	}

	// Get tenant ID
	tenantID, _ := resolver.GetTenantID()

	// Get conversation ID for memory
	conversationID := resolver.GetString("conversation_id", "")
	if conversationID == "" {
		conversationID = resolver.GetString("sender_id", "")
	}

	log.Printf("🤖 AI Agent '%s' - Model: %s, Memory: %v", node.Name, aiConfig.Model, aiConfig.UseMemory)

	var responseText string
	var metadata map[string]any

	// Execute with or without memory
	if aiConfig.UseMemory && conversationID != "" && tenantID != "" {
		responseText, metadata, err = e.executeWithAgent(ctx, aiConfig, userMessage, string(tenantID), conversationID, input)
	} else {
		responseText, metadata, err = e.executeWithLLM(ctx, aiConfig, userMessage, input)
	}

	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("AI execution failed: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result, err
	}

	result.Success = true
	result.Output["ai_response"] = responseText
	result.Output["response"] = responseText
	result.Output["model"] = aiConfig.Model
	result.Output["provider"] = aiConfig.Provider

	if metadata != nil {
		maps.Copy(result.Output, metadata)
	}

	result.Duration = time.Since(startTime).Milliseconds()
	log.Printf("✅ AI Agent completed in %dms", result.Duration)

	return result, nil
}

func (e *AIAgentExecutor) executeWithLLM(
	ctx context.Context,
	config *engine.AIAgentConfig,
	userMessage string,
	input map[string]any,
) (string, map[string]any, error) {
	client := config.GetLLMClient()

	messages := []llm.Message{
		llm.NewSystemMessage(config.SystemPrompt),
		llm.NewUserMessage(userMessage),
	}

	response, err := client.Chat(ctx, messages, config.GetLLMOptions()...)
	if err != nil {
		return "", nil, err
	}

	metadata := map[string]any{
		"mode": "llm",
		"tokens_used": map[string]any{
			"prompt":     response.Usage.PromptTokens,
			"completion": response.Usage.CompletionTokens,
			"total":      response.Usage.TotalTokens,
		},
	}

	return response.Message.Content, metadata, nil
}

func (e *AIAgentExecutor) executeWithAgent(
	ctx context.Context,
	config *engine.AIAgentConfig,
	userMessage string,
	tenantID string,
	conversationID string,
	input map[string]any,
) (string, map[string]any, error) {
	llmClient := config.GetLLMClient()

	memory := agent.NewSessionMemory(
		ctx,
		kernel.TenantID(tenantID),
		kernel.SessionID(conversationID),
		config.SystemPrompt,
		[]llm.Message{},
		e.agentChatRepo,
	)

	agentOptions := []agentx.AgentOption{
		agentx.WithOptions(config.GetLLMOptions()...),
		agentx.WithMaxAutoIterations(config.GetMaxAutoIterations()),
		agentx.WithMaxTotalIterations(config.GetMaxTotalIterations()),
	}

	agentInstance := agentx.New(llmClient, memory, agentOptions...)

	response, err := agentInstance.Run(ctx, userMessage)
	if err != nil {
		return "", nil, err
	}

	metadata := map[string]any{
		"mode":            "agent",
		"conversation_id": conversationID,
		"has_memory":      true,
	}

	return response, metadata, nil
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

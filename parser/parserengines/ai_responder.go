package parserengines

import (
	"context"
	"fmt"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/llm/agentx"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/parser"
	"github.com/Abraxas-365/relay/pkg/agent"
)

type AIParserEngine struct {
	agentChatRepo agent.AgentChatRepository
}

func NewAIParserEngine(agentChatRepo agent.AgentChatRepository) *AIParserEngine {
	return &AIParserEngine{
		agentChatRepo: agentChatRepo,
	}
}

func (rpe *AIParserEngine) Parse(ctx context.Context, p parser.Parser, msg engine.Message, session *engine.Session) (*parser.ParseResult, error) {
	// Get typed config
	config, err := p.GetConfigStruct()
	if err != nil {
		return parser.NewFailureResult(p.ID, p.Name, err), err
	}

	aiConfig, ok := config.(parser.AIParserConfig)
	if !ok {
		err := fmt.Errorf("invalid config type for AI parser")
		return parser.NewFailureResult(p.ID, p.Name, err), err
	}

	// Check if agent mode is enabled
	if aiConfig.UseAgent {
		return rpe.parseWithAgent(ctx, p, aiConfig, msg, session)
	}

	// Regular LLM mode (original implementation)
	return rpe.parseWithLLM(ctx, p, aiConfig, msg)
}

// parseWithLLM uses regular LLM without memory
func (rpe *AIParserEngine) parseWithLLM(ctx context.Context, p parser.Parser, aiConfig parser.AIParserConfig, msg engine.Message) (*parser.ParseResult, error) {
	result := parser.NewParseResult(p.ID, p.Name)
	llmClient := aiConfig.GetLLMClient()

	response, err := llmClient.Chat(
		ctx,
		[]llm.Message{
			llm.NewSystemMessage(aiConfig.Prompt),
			llm.NewUserMessage(msg.Content.Text),
		},
		aiConfig.GetLLMOptions()...,
	)
	if err != nil {
		return parser.NewFailureResult(p.ID, p.Name, err), err
	}

	result.Success = true
	result.ShouldRespond = true
	result.Response = response.Message.Content
	return result, nil
}

// parseWithAgent uses agentx with persistent SessionMemory
func (rpe *AIParserEngine) parseWithAgent(ctx context.Context, p parser.Parser, aiConfig parser.AIParserConfig, msg engine.Message, session *engine.Session) (*parser.ParseResult, error) {
	result := parser.NewParseResult(p.ID, p.Name)

	// Create LLM client
	llmClient := aiConfig.GetLLMClient()

	// Create context messages (optional - can include session context)
	contextMessages := rpe.buildContextMessages(session)

	// Create SessionMemory with persistent storage
	memory := agent.NewSessionMemory(
		ctx,
		session.ID,
		aiConfig.GetSystemPrompt(),
		contextMessages,
		rpe.agentChatRepo,
	)

	// Create agent options
	agentOptions := []agentx.AgentOption{
		agentx.WithOptions(aiConfig.GetLLMOptions()...),
		agentx.WithMaxAutoIterations(aiConfig.GetMaxAutoIterations()),
		agentx.WithMaxTotalIterations(aiConfig.GetMaxTotalIterations()),
	}

	// TODO: Add tools support when implemented
	// if len(aiConfig.Tools) > 0 {
	//     toolxClient := rpe.createToolxClient(ctx, aiConfig.Tools)
	//     agentOptions = append(agentOptions, agentx.WithTools(toolxClient))
	// }

	// Create agent
	agent := agentx.New(llmClient, memory, agentOptions...)

	// Run agent with user input
	response, err := agent.Run(ctx, msg.Content.Text)
	if err != nil {
		return parser.NewFailureResult(p.ID, p.Name, err), err
	}

	result.Success = true
	result.ShouldRespond = true
	result.Response = response

	// Optional: Add metadata about the conversation
	if result.Metadata == nil {
		result.Metadata = make(map[string]any)
	}
	result.Metadata["agent_mode"] = true
	result.Metadata["session_id"] = session.ID.String()

	return result, nil
}

// buildContextMessages creates context messages from session data
// This can include information about the current session state, user info, etc.
func (rpe *AIParserEngine) buildContextMessages(session *engine.Session) []llm.Message {
	var contextMessages []llm.Message

	// Example: Add session context as a system message
	if session.Context != nil {
		// You can format session context as additional context
		// For example, if you have user preferences, current state, etc.

		// contextMsg := llm.Message{
		// 	Role: llm.RoleSystem,
		// 	Content: fmt.Sprintf("Current session state: %v", session.CurrentState),
		// }
		// contextMessages = append(contextMessages, contextMsg)
	}

	return contextMessages
}

func (rpe *AIParserEngine) SupportsType(parserType parser.ParserType) bool {
	return parserType == parser.ParserTypeAI
}

func (rpe *AIParserEngine) ValidateConfig(config parser.ParserConfig) error {
	ai, ok := config.(parser.AIParserConfig)
	if !ok {
		return fmt.Errorf("invalid config type, expected AIParserConfig")
	}

	// Validate the config using its built-in method
	if err := ai.Validate(); err != nil {
		return err
	}

	return nil
}


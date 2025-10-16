package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/eventx"
	"github.com/Abraxas-365/craftable/eventx/providers/eventxmemory"

	"github.com/Abraxas-365/relay/channels"
	whatsapp "github.com/Abraxas-365/relay/channels/channeladapters/whatssapp"
	"github.com/Abraxas-365/relay/channels/channelapi"
	"github.com/Abraxas-365/relay/channels/channelmanager"
	"github.com/Abraxas-365/relay/channels/channelsinfra"
	"github.com/Abraxas-365/relay/channels/channelsrv"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/engine/delayscheduler"
	"github.com/Abraxas-365/relay/engine/engineinfra"
	"github.com/Abraxas-365/relay/engine/msgprocessor"
	"github.com/Abraxas-365/relay/engine/nodeexec"
	"github.com/Abraxas-365/relay/engine/sessmanager"
	"github.com/Abraxas-365/relay/engine/workflowexec"

	"github.com/Abraxas-365/relay/iam"
	"github.com/Abraxas-365/relay/iam/auth"
	"github.com/Abraxas-365/relay/iam/auth/authinfra"
	"github.com/Abraxas-365/relay/iam/role"
	"github.com/Abraxas-365/relay/iam/role/roleinfra"
	"github.com/Abraxas-365/relay/iam/role/rolesrv"
	"github.com/Abraxas-365/relay/iam/tenant"
	"github.com/Abraxas-365/relay/iam/tenant/tenantinfra"
	"github.com/Abraxas-365/relay/iam/tenant/tenantsrv"
	"github.com/Abraxas-365/relay/iam/user"
	"github.com/Abraxas-365/relay/iam/user/userinfra"
	"github.com/Abraxas-365/relay/iam/user/usersrv"

	"github.com/Abraxas-365/relay/parser"
	"github.com/Abraxas-365/relay/parser/parserengines"
	"github.com/Abraxas-365/relay/parser/parserinfra"

	"github.com/Abraxas-365/relay/tool"

	"github.com/Abraxas-365/relay/pkg/agent"
	"github.com/Abraxas-365/relay/pkg/agent/agentinfra"
	"github.com/Abraxas-365/relay/pkg/config"
	"github.com/Abraxas-365/relay/pkg/kernel"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
)

// Container contains all application dependencies
type Container struct {
	// =================================================================
	// CONFIGURATION & INFRASTRUCTURE
	// =================================================================
	Config      *config.Config
	DB          *sqlx.DB
	RedisClient *redis.Client

	// =================================================================
	// EVENT BUS ⚡
	// =================================================================
	EventBus eventx.EventBus

	// =================================================================
	// IAM - REPOSITORIES
	// =================================================================
	UserRepo         user.UserRepository
	UserRoleRepo     user.UserRoleRepository
	TenantRepo       tenant.TenantRepository
	TenantConfigRepo tenant.TenantConfigRepository
	RoleRepo         role.RoleRepository
	RolePermRepo     role.RolePermissionRepository

	// =================================================================
	// IAM - SERVICES
	// =================================================================
	PasswordService user.PasswordService
	UserService     *usersrv.UserService
	TenantService   *tenantsrv.TenantService
	RoleService     *rolesrv.RoleService

	// =================================================================
	// AUTH
	// =================================================================
	TokenRepo         auth.TokenRepository
	SessionRepo       auth.SessionRepository
	PasswordResetRepo auth.PasswordResetRepository
	StateManager      auth.StateManager
	TokenService      auth.TokenService
	OAuthServices     map[iam.OAuthProvider]auth.OAuthService
	AuthHandlers      *auth.AuthHandlers
	AuthMiddleware    *auth.AuthMiddleware

	// =================================================================
	// AGENT 🤖
	// =================================================================
	AgentChatRepo agent.AgentChatRepository

	// =================================================================
	// CHANNELS
	// =================================================================
	ChannelRepo    channels.ChannelRepository
	ChannelManager channels.ChannelManager
	ChannelService *channelsrv.ChannelService

	// Channel Adapters
	WhatsAppAdapter *whatsapp.WhatsAppAdapter

	// Channel API Handlers
	ChannelHandler         *channelapi.ChannelHandler
	WhatsAppWebhookHandler *whatsapp.WebhookHandler
	WhatsAppWebhookRoutes  *whatsapp.WebhookRoutes

	// =================================================================
	// ENGINE
	// =================================================================
	MessageRepo         engine.MessageRepository
	WorkflowRepo        engine.WorkflowRepository
	EngineSessionRepo   engine.SessionRepository
	SessionManager      engine.SessionManager
	WorkflowExecutor    engine.WorkflowExecutor
	MessageProcessor    engine.MessageProcessor
	ExpressionEvaluator engine.ExpressionEvaluator
	DelayScheduler      engine.DelayScheduler // ✅ NEW: Delay Scheduler

	// Step Executors
	ActionExecutor    engine.NodeExecutor
	ConditionExecutor engine.NodeExecutor
	ResponseExecutor  engine.NodeExecutor
	DelayExecutor     engine.NodeExecutor

	// =================================================================
	// PARSER 🔍
	// =================================================================
	ParserRepo    parser.ParserRepository
	ParserManager *parser.ParserManager

	// Parser Engines
	RegexParserEngine   parser.ParserEngine
	AIParserEngine      parser.ParserEngine
	RuleParserEngine    parser.ParserEngine
	KeywordParserEngine parser.ParserEngine
	NLPParserEngine     parser.ParserEngine

	// =================================================================
	// TOOL 🛠️
	// =================================================================
	ToolRepo     tool.ToolRepository
	ToolExecutor tool.ToolExecutor

	// =================================================================
	// AI/LLM 🤖
	// =================================================================
	LLMClient *llm.Client
}

// NewContainer creates a new dependency container
func NewContainer(cfg *config.Config, db *sqlx.DB, redisClient *redis.Client) *Container {
	c := &Container{
		Config:      cfg,
		DB:          db,
		RedisClient: redisClient,
	}

	// Initialize dependencies in the correct order
	log.Println("📦 Initializing dependency container...")

	c.initEventBus()
	c.initIAMRepositories()
	c.initIAMServices()
	c.initAuthServices()
	c.initAgentComponents()   // 🤖 Agent components (needed by AI parser)
	c.initLLMComponents()     // LLM (needed by AI parser)
	c.initParserComponents()  // Parser before engine
	c.initToolComponents()    // Tools before engine
	c.initChannelComponents() // ⚡ Channels BEFORE engine
	c.initEngineComponents()  // ⚙️ Engine AFTER channels (can use ChannelManager)

	log.Println("✅ Dependency container initialized successfully")

	return c
}

// =================================================================
// EVENT BUS INITIALIZATION ⚡
// =================================================================

func (c *Container) initEventBus() {
	log.Println("  ⚡ Initializing event bus...")

	busConfig := eventx.BusConfig{
		ConnectionName:    "relay-event-bus",
		EnableLogging:     true,
		EnableMetrics:     true,
		EnablePersistence: false,
		AutoAck:           true,
		MaxRetries:        3,
	}

	c.EventBus = eventxmemory.New(busConfig)

	ctx := context.Background()
	if err := c.EventBus.Connect(ctx); err != nil {
		log.Fatalf("❌ Failed to connect event bus: %v", err)
	}

	log.Println("  ✅ Event bus initialized and connected")
}

// =================================================================
// IAM INITIALIZATION
// =================================================================

func (c *Container) initIAMRepositories() {
	log.Println("  👥 Initializing IAM repositories...")
	c.UserRepo = userinfra.NewPostgresUserRepository(c.DB)
	c.UserRoleRepo = userinfra.NewPostgresUserRoleRepository(c.DB)
	c.TenantRepo = tenantinfra.NewPostgresTenantRepository(c.DB)
	c.TenantConfigRepo = tenantinfra.NewPostgresTenantConfigRepository(c.DB)
	c.RoleRepo = roleinfra.NewPostgresRoleRepository(c.DB)
	c.RolePermRepo = roleinfra.NewPostgresRolePermissionRepository(c.DB)
}

func (c *Container) initIAMServices() {
	log.Println("  👥 Initializing IAM services...")
	c.PasswordService = authinfra.NewBcryptPasswordService()

	c.UserService = usersrv.NewUserService(
		c.UserRepo,
		c.UserRoleRepo,
		c.TenantRepo,
		c.RoleRepo,
		c.PasswordService,
	)

	c.TenantService = tenantsrv.NewTenantService(
		c.TenantRepo,
		c.TenantConfigRepo,
		c.UserRepo,
	)

	c.RoleService = rolesrv.NewRoleService(
		c.RoleRepo,
		c.RolePermRepo,
		c.TenantRepo,
	)
}

func (c *Container) initAuthServices() {
	log.Println("  🔐 Initializing auth services...")

	c.TokenRepo = authinfra.NewPostgresTokenRepository(c.DB)
	c.SessionRepo = authinfra.NewPostgresSessionRepository(c.DB)
	c.PasswordResetRepo = authinfra.NewPostgresPasswordResetRepository(c.DB)
	c.StateManager = authinfra.NewRedisStateManager(c.RedisClient)

	c.TokenService = auth.NewJWTService(
		c.Config.Auth.JWT.SecretKey,
		c.Config.Auth.JWT.AccessTokenTTL,
		c.Config.Auth.JWT.RefreshTokenTTL,
		c.Config.Auth.JWT.Issuer,
	)

	c.OAuthServices = make(map[iam.OAuthProvider]auth.OAuthService)

	if c.Config.Auth.OAuth.Google.IsEnabled() {
		c.OAuthServices[iam.OAuthProviderGoogle] = auth.NewGoogleOAuthService(
			c.Config.Auth.OAuth.Google,
			c.StateManager,
		)
	}

	if c.Config.Auth.OAuth.Microsoft.IsEnabled() {
		c.OAuthServices[iam.OAuthProviderMicrosoft] = auth.NewMicrosoftOAuthService(
			c.Config.Auth.OAuth.Microsoft,
			c.StateManager,
		)
	}

	c.AuthHandlers = auth.NewAuthHandlers(
		c.OAuthServices,
		c.TokenService,
		c.UserRepo,
		c.TenantRepo,
		c.TokenRepo,
		c.SessionRepo,
		c.StateManager,
	)

	c.AuthMiddleware = auth.NewAuthMiddleware(c.TokenService)
}

// =================================================================
// AGENT INITIALIZATION 🤖
// =================================================================

func (c *Container) initAgentComponents() {
	log.Println("  🤖 Initializing agent components...")

	// Initialize agent chat repository
	c.AgentChatRepo = agentinfra.NewPostgresAgentChatRepository(c.DB)
	log.Println("    ✅ AgentChatRepo initialized")

	log.Println("  ✅ Agent components initialized")
}

// =================================================================
// LLM INITIALIZATION 🤖
// =================================================================

func (c *Container) initLLMComponents() {
	log.Println("  🤖 Initializing LLM components...")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("  ⚠️  OPENAI_API_KEY not set, AI parser will be disabled")
		return
	}

	client := aiopenai.NewOpenAIProvider(apiKey)
	c.LLMClient = llm.NewClient(client)

	log.Println("  ✅ LLM components initialized")
}

// =================================================================
// PARSER INITIALIZATION 🔍
// =================================================================

func (c *Container) initParserComponents() {
	log.Println("  🔍 Initializing parser components...")

	c.ParserRepo = parserinfra.NewPostgresParserRepository(c.DB)
	log.Println("    ✅ ParserRepo initialized")

	// Initialize parser engines
	c.RegexParserEngine = parserengines.NewRegexParserEngine()
	log.Println("    ✅ Regex parser engine initialized")

	// Initialize AI parser engine with AgentChatRepo
	c.AIParserEngine = parserengines.NewAIParserEngine(c.AgentChatRepo)
	log.Println("    ✅ AI parser engine initialized with agent support")

	// TODO: Initialize other parser engines
	// c.RuleParserEngine = parserengines.NewRuleParserEngine()
	// log.Println("    ✅ Rule parser engine initialized")
	//
	// c.KeywordParserEngine = parserengines.NewKeywordParserEngine()
	// log.Println("    ✅ Keyword parser engine initialized")

	// NLP parser requires additional dependencies
	log.Println("    ⚠️  NLP parser engine not initialized (pending implementation)")

	// Initialize ParserManager
	c.ParserManager = parser.NewParserManager(c.ParserRepo)

	// Register parser engines
	c.ParserManager.RegisterEngine(parser.ParserTypeRegex, c.RegexParserEngine)
	log.Println("    ✅ Registered Regex parser engine")

	if c.AIParserEngine != nil {
		c.ParserManager.RegisterEngine(parser.ParserTypeAI, c.AIParserEngine)
		log.Println("    ✅ Registered AI parser engine")
	}

	if c.RuleParserEngine != nil {
		c.ParserManager.RegisterEngine(parser.ParserTypeRule, c.RuleParserEngine)
		log.Println("    ✅ Registered Rule parser engine")
	}

	if c.KeywordParserEngine != nil {
		c.ParserManager.RegisterEngine(parser.ParserTypeKeyword, c.KeywordParserEngine)
		log.Println("    ✅ Registered Keyword parser engine")
	}

	if c.NLPParserEngine != nil {
		c.ParserManager.RegisterEngine(parser.ParserTypeNLP, c.NLPParserEngine)
		log.Println("    ✅ Registered NLP parser engine")
	}

	log.Println("  ✅ Parser components initialized")
}

// =================================================================
// TOOL INITIALIZATION 🛠️
// =================================================================

func (c *Container) initToolComponents() {
	log.Println("  🛠️ Initializing tool components...")

	// TODO: Initialize ToolRepo and ToolExecutor
	// c.ToolRepo = toolinfra.NewPostgresToolRepository(c.DB)
	// c.ToolExecutor = toolsrv.NewToolExecutor(...)

	log.Println("  ⚠️  Tool components not initialized (pending implementation)")
}

// =================================================================
// CHANNELS INITIALIZATION 📡 (BEFORE ENGINE)
// =================================================================

func (c *Container) initChannelComponents() {
	log.Println("  📡 Initializing channel components...")

	// Initialize channel repository
	c.ChannelRepo = channelsinfra.NewPostgresChannelRepository(c.DB)
	log.Println("    ✅ Channel repository initialized")

	// Initialize the channel manager FIRST
	c.ChannelManager = channelmanager.NewDefaultChannelManager(c.ChannelRepo, c.RedisClient)
	log.Println("    ✅ Channel manager initialized")

	// Initialize WhatsApp adapter (base instance)
	c.WhatsAppAdapter = whatsapp.NewWhatsAppAdapter(
		channels.WhatsAppConfig{}, // Empty config, overridden per channel
		c.RedisClient,
	)

	// Initialize channel service
	c.ChannelService = channelsrv.NewChannelService(
		c.ChannelRepo,
		c.TenantRepo,
		c.ChannelManager,
	)
	log.Println("    ✅ Channel service initialized")

	log.Println("  ✅ Channel components initialized")
}

// =================================================================
// ENGINE INITIALIZATION ⚙️ (AFTER CHANNELS)
// =================================================================

func (c *Container) initEngineComponents() {
	log.Println("  ⚙️  Initializing engine components...")

	// Initialize repositories
	c.MessageRepo = engineinfra.NewPostgresMessageRepository(c.DB)
	c.WorkflowRepo = engineinfra.NewPostgresWorkflowRepository(c.DB)
	c.EngineSessionRepo = engineinfra.NewPostgresSessionRepository(c.DB)

	// Initialize session manager
	sessionConfig := &sessmanager.SessionManagerConfig{
		DefaultExpirationTime: 10,
		MaxHistorySize:        100,
	}
	c.SessionManager = sessmanager.NewSessionManager(c.EngineSessionRepo, sessionConfig)
	log.Println("    ✅ Session manager initialized")

	// Initialize expression evaluator
	c.ExpressionEvaluator = engine.NewCelEvaluator()
	log.Println("    ✅ Expression evaluator initialized")

	// ✅ NEW: Initialize delay scheduler with continuation handler
	c.DelayScheduler = delayscheduler.NewRedisDelayScheduler(
		c.RedisClient,
		c.handleWorkflowContinuation,
	)
	log.Println("    ✅ Delay scheduler initialized")

	// Start delay scheduler worker
	ctx := context.Background()
	c.DelayScheduler.StartWorker(ctx)
	log.Println("    ✅ Delay scheduler worker started")

	// Initialize step executors
	c.ActionExecutor = nodeexec.NewActionExecutor()
	c.ConditionExecutor = nodeexec.NewConditionExecutor()
	c.ResponseExecutor = nodeexec.NewResponseExecutor()
	c.DelayExecutor = nodeexec.NewDelayExecutor(c.DelayScheduler) // ✅ Pass scheduler
	log.Println("    ✅ Step executors initialized")

	// Initialize workflow executor with ExpressionEvaluator
	c.WorkflowExecutor = workflowexec.NewDefaultWorkflowExecutor(
		c.ParserManager,
		c.ChannelManager,
		c.ExpressionEvaluator,
		c.ActionExecutor,
		c.ConditionExecutor,
		c.ResponseExecutor,
		c.DelayExecutor,
	)
	log.Println("    ✅ Workflow executor initialized with parser manager and expression evaluator")

	// ⚡ Initialize message processor WITH ChannelManager
	c.MessageProcessor = msgprocessor.NewMessageProcessor(
		c.MessageRepo,
		c.WorkflowRepo,
		c.SessionManager,
		c.WorkflowExecutor,
		c.ChannelManager, // ChannelManager already exists!
	)
	log.Println("    ✅ Message processor initialized with ChannelManager")

	// Initialize channel API handler (needs MessageProcessor)
	c.ChannelHandler = channelapi.NewChannelHandler(c.MessageProcessor)
	log.Println("    ✅ Channel API handler initialized")

	// Initialize WhatsApp webhook handler
	c.WhatsAppWebhookHandler = whatsapp.NewWebhookHandler(
		c.ChannelRepo,
		c.WhatsAppAdapter,
	)
	log.Println("    ✅ WhatsApp webhook handler initialized")

	// Initialize WhatsApp webhook routes (will be setup in main.go)
	c.WhatsAppWebhookRoutes = whatsapp.NewWebhookRoutes(
		c.WhatsAppWebhookHandler,
		c.ChannelHandler.ProcessIncomingMessage,
	)
	log.Println("    ✅ WhatsApp webhook routes initialized")

	log.Println("  ✅ Engine components initialized")
}

// =================================================================
// WORKFLOW CONTINUATION HANDLER
// =================================================================

// handleWorkflowContinuation is called when a delayed execution is ready
func (c *Container) handleWorkflowContinuation(
	ctx context.Context,
	continuation *engine.WorkflowContinuation,
) error {
	log.Printf("🔄 Resuming workflow %s from step %s",
		continuation.WorkflowID, continuation.NodeID)

	// Get session
	session, err := c.SessionManager.Get(ctx, kernel.SessionID(continuation.SessionID))
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Reconstruct message from continuation context
	message := engine.Message{
		ID:        kernel.MessageID(continuation.MessageID),
		TenantID:  kernel.TenantID(continuation.TenantID),
		ChannelID: kernel.ChannelID(continuation.ChannelID),
		SenderID:  continuation.SenderID,
		Content: engine.MessageContent{
			Text: fmt.Sprintf("Resuming after delay from step %s", continuation.NodeID),
			Type: engine.MessageTypeText,
		},
		Context: continuation.NodeContext,
	}

	// Get workflow
	workflow, err := c.WorkflowRepo.FindByID(ctx, kernel.WorkflowID(continuation.WorkflowID))
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// ✅ Resume from next step (not re-execute entire workflow!)
	var result *engine.ExecutionResult
	if continuation.NextNodeID != "" {
		// Resume from the next step with saved context
		result, err = c.WorkflowExecutor.ResumeFromNode(
			ctx,
			*workflow,
			message,
			session,
			continuation.NextNodeID,
			continuation.NodeContext,
		)
	} else {
		// No next step, workflow is complete
		log.Printf("✅ Workflow %s completed (no next step after delay)", workflow.ID.String())
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to resume workflow: %w", err)
	}

	log.Printf("✅ Workflow resumed successfully: success=%v, response=%s",
		result.Success, result.Response)

	// Update session with result
	if result.Context != nil {
		for key, value := range result.Context {
			session.SetContext(key, value)
		}
	}
	if result.NextState != "" {
		session.UpdateState(result.NextState)
	}
	session.ExtendExpiration(30 * time.Minute)
	if err := c.SessionManager.Update(ctx, *session); err != nil {
		log.Printf("⚠️  Failed to update session: %v", err)
	}

	// Send response if needed
	if result.ShouldRespond && result.Response != "" {
		outgoingMsg := channels.OutgoingMessage{
			RecipientID: message.SenderID,
			Content: channels.MessageContent{
				Type: "text",
				Text: result.Response,
			},
			Metadata: map[string]any{
				"workflow_id":        continuation.WorkflowID,
				"continuation_id":    continuation.ID,
				"delayed_from_step":  continuation.NodeID,
				"workflow_triggered": true,
				"timestamp":          time.Now().Unix(),
			},
		}

		if err := c.ChannelManager.SendMessage(
			ctx,
			message.TenantID,
			message.ChannelID,
			outgoingMsg,
		); err != nil {
			log.Printf("⚠️  Failed to send response: %v", err)
		} else {
			log.Printf("✅ Response sent successfully to %s", message.SenderID)
		}
	}

	return nil
}

// =================================================================
// UTILITY METHODS
// =================================================================

func (c *Container) GetAllRoutes() []RouteGroup {
	routes := []RouteGroup{
		{Name: "auth", Handler: c.AuthHandlers},
		{Name: "whatsapp_webhook", Handler: c.WhatsAppWebhookHandler},
		{Name: "channel_api", Handler: c.ChannelHandler},
	}
	return routes
}

type RouteGroup struct {
	Name    string
	Handler any
}

func (c *Container) Cleanup() {
	log.Println("🧹 Cleaning up container resources...")

	// ✅ NEW: Stop delay scheduler worker
	if c.DelayScheduler != nil {
		log.Println("  ⏰ Stopping delay scheduler...")
		c.DelayScheduler.StopWorker()
	}

	if c.EventBus != nil {
		log.Println("  ⚡ Disconnecting event bus...")
		ctx := context.Background()
		if err := c.EventBus.Disconnect(ctx); err != nil {
			log.Printf("  ⚠️  Failed to disconnect event bus: %v", err)
		}
	}

	if c.DB != nil {
		log.Println("  🗄️  Closing database connections...")
		c.DB.Close()
	}

	if c.RedisClient != nil {
		log.Println("  🔴 Closing Redis connections...")
		c.RedisClient.Close()
	}

	log.Println("✅ Container cleanup complete")
}

func (c *Container) HealthCheck() map[string]bool {
	health := make(map[string]bool)

	if c.DB != nil {
		err := c.DB.Ping()
		health["database"] = err == nil
	} else {
		health["database"] = false
	}

	if c.RedisClient != nil {
		err := c.RedisClient.Ping(c.RedisClient.Context()).Err()
		health["redis"] = err == nil
	} else {
		health["redis"] = false
	}

	if c.EventBus != nil {
		health["event_bus"] = c.EventBus.IsConnected()
	} else {
		health["event_bus"] = false
	}

	health["parser_manager"] = c.ParserManager != nil
	health["workflow_executor"] = c.WorkflowExecutor != nil
	health["message_processor"] = c.MessageProcessor != nil
	health["channel_manager"] = c.ChannelManager != nil
	health["whatsapp_adapter"] = c.WhatsAppAdapter != nil
	health["agent_chat_repo"] = c.AgentChatRepo != nil
	health["delay_scheduler"] = c.DelayScheduler != nil // ✅ NEW

	return health
}

func (c *Container) GetEventBusMetrics() eventx.BusMetrics {
	if metricsbus, ok := c.EventBus.(eventx.MetricsEventBus); ok {
		return metricsbus.GetMetrics()
	}
	return eventx.BusMetrics{}
}

func (c *Container) GetServiceNames() []string {
	return []string{
		"UserService",
		"TenantService",
		"RoleService",
		"ChannelService",
		"SessionManager",
		"WorkflowExecutor",
		"MessageProcessor",
		"ParserManager",
		"EventBus",
		"AgentChatRepo",
		"DelayScheduler", // ✅ NEW
	}
}

func (c *Container) GetRepositoryNames() []string {
	return []string{
		"UserRepo",
		"TenantRepo",
		"RoleRepo",
		"ChannelRepo",
		"MessageRepo",
		"WorkflowRepo",
		"EngineSessionRepo",
		"ParserRepo",
		"ToolRepo",
		"AgentChatRepo",
	}
}

func (c *Container) GetStepExecutorNames() []string {
	return []string{
		"ActionExecutor",
		"ConditionExecutor",
		"ResponseExecutor",
		"DelayExecutor",
	}
}

func (c *Container) GetParserEngineNames() []string {
	engines := []string{}
	if c.RegexParserEngine != nil {
		engines = append(engines, "RegexParserEngine")
	}
	if c.AIParserEngine != nil {
		engines = append(engines, "AIParserEngine")
	}
	if c.RuleParserEngine != nil {
		engines = append(engines, "RuleParserEngine")
	}
	if c.KeywordParserEngine != nil {
		engines = append(engines, "KeywordParserEngine")
	}
	if c.NLPParserEngine != nil {
		engines = append(engines, "NLPParserEngine")
	}
	return engines
}

func (c *Container) GetChannelAdapterNames() []string {
	adapters := []string{}
	if c.WhatsAppAdapter != nil {
		adapters = append(adapters, "WhatsAppAdapter")
	}
	// TODO: Add other adapters
	return adapters
}

// ✅ NEW: Get delay scheduler metrics
func (c *Container) GetDelaySchedulerMetrics(ctx context.Context) (int64, error) {
	if c.DelayScheduler != nil {
		return c.DelayScheduler.GetPendingCount(ctx)
	}
	return 0, fmt.Errorf("delay scheduler not initialized")
}

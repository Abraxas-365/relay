package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
	"github.com/Abraxas-365/relay/engine/nodeexec"
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
	// CHANNELS (Optional integration)
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
	// ENGINE (n8n-style)
	// =================================================================
	WorkflowRepo        engine.WorkflowRepository
	WorkflowExecutor    engine.WorkflowExecutor
	ExpressionEvaluator engine.ExpressionEvaluator
	DelayScheduler      engine.DelayScheduler

	// Node Executors
	ActionExecutor      engine.NodeExecutor
	ConditionExecutor   engine.NodeExecutor
	DelayExecutor       engine.NodeExecutor
	AIAgentExecutor     engine.NodeExecutor
	SendMessageExecutor engine.NodeExecutor
	HTTPExecutor        engine.NodeExecutor

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
	c.initAgentComponents()   // 🤖 Agent components (needed by AI executor)
	c.initLLMComponents()     // LLM (needed by AI executor)
	c.initChannelComponents() // ⚡ Channels (optional integration)
	c.initEngineComponents()  // ⚙️ Engine components

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
		log.Println("  ⚠️  OPENAI_API_KEY not set, AI features will be disabled")
		return
	}

	client := aiopenai.NewOpenAIProvider(apiKey)
	c.LLMClient = llm.NewClient(client)

	log.Println("  ✅ LLM components initialized")
}

// =================================================================
// CHANNELS INITIALIZATION 📡 (Optional Integration)
// =================================================================

func (c *Container) initChannelComponents() {
	log.Println("  📡 Initializing channel components (optional)...")

	// Initialize channel repository
	c.ChannelRepo = channelsinfra.NewPostgresChannelRepository(c.DB)
	log.Println("    ✅ Channel repository initialized")

	// Initialize the channel manager
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
// ENGINE INITIALIZATION ⚙️ (n8n-style)
// =================================================================

func (c *Container) initEngineComponents() {
	log.Println("  ⚙️  Initializing engine components (n8n-style)...")

	// Initialize workflow repository
	c.WorkflowRepo = engineinfra.NewPostgresWorkflowRepository(c.DB)
	log.Println("    ✅ Workflow repository initialized")

	// Initialize expression evaluator
	c.ExpressionEvaluator = engine.NewCelEvaluator()
	log.Println("    ✅ Expression evaluator initialized")

	// ⏰ Initialize delay scheduler with continuation handler
	c.DelayScheduler = delayscheduler.NewRedisDelayScheduler(
		c.RedisClient,
		c.handleWorkflowContinuation,
	)
	log.Println("    ✅ Delay scheduler initialized")

	// Start delay scheduler worker
	ctx := context.Background()
	c.DelayScheduler.StartWorker(ctx)
	log.Println("    ✅ Delay scheduler worker started")

	// Initialize node executors
	c.ActionExecutor = nodeexec.NewActionExecutor()
	c.ConditionExecutor = nodeexec.NewConditionExecutor()
	c.DelayExecutor = nodeexec.NewDelayExecutor(c.DelayScheduler)
	c.AIAgentExecutor = nodeexec.NewAIAgentExecutor(c.AgentChatRepo)
	c.SendMessageExecutor = nodeexec.NewSendMessageExecutor(c.ChannelManager)
	// c.HTTPExecutor = nodeexec.NewHTTPExecutor() // TODO: Add HTTP executor
	log.Println("    ✅ Node executors initialized")

	// Initialize workflow executor (n8n-style)
	c.WorkflowExecutor = workflowexec.NewDefaultWorkflowExecutor(
		c.ExpressionEvaluator,
		c.ActionExecutor,
		c.ConditionExecutor,
		c.DelayExecutor,
		c.AIAgentExecutor,
		c.SendMessageExecutor,
		// c.HTTPExecutor,
	)
	log.Println("    ✅ Workflow executor initialized (n8n-style)")

	// Initialize channel webhook handler (for channel trigger workflows)
	if c.ChannelRepo != nil && c.WhatsAppAdapter != nil {
		c.WhatsAppWebhookHandler = whatsapp.NewWebhookHandler(
			c.ChannelRepo,
			c.WhatsAppAdapter,
		)
		log.Println("    ✅ WhatsApp webhook handler initialized")

		// Initialize channel API handler
		// This would trigger workflows when messages arrive
		// c.ChannelHandler = channelapi.NewChannelHandler(c.WorkflowExecutor)
		// log.Println("    ✅ Channel API handler initialized")
	}

	log.Println("  ✅ Engine components initialized")
}

// =================================================================
// WORKFLOW CONTINUATION HANDLER ⏰
// =================================================================

// handleWorkflowContinuation is called when a delayed execution is ready
func (c *Container) handleWorkflowContinuation(
	ctx context.Context,
	continuation *engine.WorkflowContinuation,
) error {
	log.Printf("🔄 Resuming workflow %s from node %s",
		continuation.WorkflowID, continuation.NextNodeID)

	// Get workflow
	workflow, err := c.WorkflowRepo.FindByID(ctx, kernel.WorkflowID(continuation.WorkflowID))
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Prepare workflow input from saved context
	input := engine.WorkflowInput{
		TriggerData: continuation.NodeContext,
		TenantID:    kernel.TenantID(continuation.TenantID),
		Metadata: map[string]any{
			"resumed_from_delay": true,
			"original_node_id":   continuation.NodeID,
			"continuation_id":    continuation.ID,
		},
	}

	// Resume workflow from next node
	var result *engine.ExecutionResult
	if continuation.NextNodeID != "" {
		result, err = c.WorkflowExecutor.ResumeFromNode(
			ctx,
			*workflow,
			input,
			continuation.NextNodeID,
			continuation.NodeContext,
		)
	} else {
		log.Printf("✅ Workflow %s completed (no next node after delay)", workflow.ID.String())
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to resume workflow: %w", err)
	}

	log.Printf("✅ Workflow resumed successfully: success=%v", result.Success)

	return nil
}

// =================================================================
// UTILITY METHODS
// =================================================================

func (c *Container) GetAllRoutes() []RouteGroup {
	routes := []RouteGroup{
		{Name: "auth", Handler: c.AuthHandlers},
	}

	// Add channel routes if available
	if c.WhatsAppWebhookHandler != nil {
		routes = append(routes, RouteGroup{
			Name:    "whatsapp_webhook",
			Handler: c.WhatsAppWebhookHandler,
		})
	}

	if c.ChannelHandler != nil {
		routes = append(routes, RouteGroup{
			Name:    "channel_api",
			Handler: c.ChannelHandler,
		})
	}

	return routes
}

type RouteGroup struct {
	Name    string
	Handler any
}

func (c *Container) Cleanup() {
	log.Println("🧹 Cleaning up container resources...")

	// Stop delay scheduler worker
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

	health["workflow_executor"] = c.WorkflowExecutor != nil
	health["channel_manager"] = c.ChannelManager != nil
	health["whatsapp_adapter"] = c.WhatsAppAdapter != nil
	health["agent_chat_repo"] = c.AgentChatRepo != nil
	health["delay_scheduler"] = c.DelayScheduler != nil

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
		"WorkflowExecutor",
		"EventBus",
		"AgentChatRepo",
		"DelayScheduler",
	}
}

func (c *Container) GetRepositoryNames() []string {
	return []string{
		"UserRepo",
		"TenantRepo",
		"RoleRepo",
		"ChannelRepo",
		"WorkflowRepo",
		"AgentChatRepo",
	}
}

func (c *Container) GetNodeExecutorNames() []string {
	return []string{
		"ActionExecutor",
		"ConditionExecutor",
		"DelayExecutor",
		"AIAgentExecutor",
		"SendMessageExecutor",
		"HTTPExecutor",
	}
}

func (c *Container) GetChannelAdapterNames() []string {
	adapters := []string{}
	if c.WhatsAppAdapter != nil {
		adapters = append(adapters, "WhatsAppAdapter")
	}
	return adapters
}

// Get delay scheduler metrics
func (c *Container) GetDelaySchedulerMetrics(ctx context.Context) (int64, error) {
	if c.DelayScheduler != nil {
		return c.DelayScheduler.GetPendingCount(ctx)
	}
	return 0, fmt.Errorf("delay scheduler not initialized")
}

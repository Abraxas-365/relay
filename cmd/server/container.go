package main

import (
	"context"
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
	"github.com/Abraxas-365/relay/engine/engineinfra"
	"github.com/Abraxas-365/relay/engine/msgprocessor"
	"github.com/Abraxas-365/relay/engine/sessmanager"
	"github.com/Abraxas-365/relay/engine/stepexec"
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

	"github.com/Abraxas-365/relay/pkg/config"

	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
)

// Container contiene todas las dependencias de la aplicación
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
	MessageRepo       engine.MessageRepository
	WorkflowRepo      engine.WorkflowRepository
	EngineSessionRepo engine.SessionRepository
	SessionManager    engine.SessionManager
	WorkflowExecutor  engine.WorkflowExecutor
	MessageProcessor  engine.MessageProcessor

	// Step Executors
	ActionExecutor    engine.StepExecutor
	ConditionExecutor engine.StepExecutor
	ResponseExecutor  engine.StepExecutor
	DelayExecutor     engine.StepExecutor

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
	// TOOL 🔧
	// =================================================================
	ToolRepo     tool.ToolRepository
	ToolExecutor tool.ToolExecutor

	// =================================================================
	// AI/LLM 🤖
	// =================================================================
	LLMClient *llm.Client
}

// NewContainer crea un nuevo contenedor de dependencias
func NewContainer(cfg *config.Config, db *sqlx.DB, redisClient *redis.Client) *Container {
	c := &Container{
		Config:      cfg,
		DB:          db,
		RedisClient: redisClient,
	}

	// Inicializar dependencias en orden correcto
	log.Println("📦 Initializing dependency container...")

	c.initEventBus()
	c.initIAMRepositories()
	c.initIAMServices()
	c.initAuthServices()
	c.initLLMComponents()     // LLM first (needed by AI parser)
	c.initParserComponents()  // Parser before engine
	c.initToolComponents()    // Tools before engine
	c.initChannelComponents() // ✅ Channels BEFORE engine
	c.initEngineComponents()  // ✅ Engine AFTER channels (can use ChannelManager)

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

	// if c.LLMClient != nil {
	// 	c.AIParserEngine = parserengines.NewAIParserEngine(c.LLMClient)
	// 	log.Println("    ✅ AI parser engine initialized")
	// } else {
	// 	log.Println("    ⚠️  AI parser engine skipped (LLM not available)")
	// }
	//
	// c.RuleParserEngine = parserengines.NewRuleParserEngine()
	// log.Println("    ✅ Rule parser engine initialized")
	//
	// c.KeywordParserEngine = parserengines.NewKeywordParserEngine()
	// log.Println("    ✅ Keyword parser engine initialized")

	// NLP parser requires additional dependencies
	log.Println("    ⚠️  NLP parser engine not initialized (pending implementation)")

	// Initialize ParserManager
	c.ParserManager = parser.NewParserManager(c.ParserRepo)

	// Register all parser engines
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
// TOOL INITIALIZATION 🔧
// =================================================================

func (c *Container) initToolComponents() {
	log.Println("  🔧 Initializing tool components...")

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

	// Initialize step executors
	c.ActionExecutor = stepexec.NewActionExecutor()
	c.ConditionExecutor = stepexec.NewConditionExecutor()
	c.ResponseExecutor = stepexec.NewResponseExecutor()
	c.DelayExecutor = stepexec.NewDelayExecutor()
	log.Println("    ✅ Step executors initialized")

	// Initialize workflow executor with ParserManager and all step executors
	c.WorkflowExecutor = workflowexec.NewDefaultWorkflowExecutor(
		c.ParserManager, // 🔍 ParserManager is now the first parameter
		c.ChannelManager,
		c.ActionExecutor,
		c.ConditionExecutor,
		c.ResponseExecutor,
		c.DelayExecutor,
	)
	log.Println("    ✅ Workflow executor initialized with parser manager")

	// ✅ Initialize message processor WITH ChannelManager (no injection needed!)
	c.MessageProcessor = msgprocessor.NewMessageProcessor(
		c.MessageRepo,
		c.WorkflowRepo,
		c.SessionManager,
		c.WorkflowExecutor,
		c.ChannelManager, // ✅ ChannelManager already exists!
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

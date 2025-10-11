package main

import (
	"context"
	"log"
	"os"

	"github.com/Abraxas-365/craftable/ai/llm"
	"github.com/Abraxas-365/craftable/ai/providers/aiopenai"
	"github.com/Abraxas-365/craftable/eventx"
	"github.com/Abraxas-365/craftable/eventx/providers/eventxmemory"
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
	// EVENT BUS ✅
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

	c.initEventBus() // ✅ Initialize EventBus first
	c.initIAMRepositories()
	c.initIAMServices()
	c.initAuthServices()

	log.Println("✅ Dependency container initialized successfully")

	return c
}

// =================================================================
// EVENT BUS INITIALIZATION ✅
// =================================================================

func (c *Container) initEventBus() {
	log.Println("  📡 Initializing event bus...")

	// Create EventBus configuration
	busConfig := eventx.BusConfig{
		ConnectionName:    "invoice-event-bus",
		EnableLogging:     true,
		EnableMetrics:     true,
		EnablePersistence: false,
		AutoAck:           true,
		MaxRetries:        3,
	}

	// Create in-memory event bus
	c.EventBus = eventxmemory.New(busConfig)

	// Connect the bus
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
// LLM INITIALIZATION ✅
// =================================================================

func (c *Container) initLLMComponents() {
	log.Println("  🤖 Initializing LLM components...")

	// Initialize LLM client
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("  ⚠️  OPENAI_API_KEY not set, LLM features will be disabled")
		return
	}

	client := aiopenai.NewOpenAIProvider("")
	c.LLMClient = llm.NewClient(
		client,
	)

	log.Println("  ✅ LLM components initialized with analytics tools")
}

// =================================================================
// UTILITY METHODS
// =================================================================

func (c *Container) GetAllRoutes() []RouteGroup {
	routes := []RouteGroup{
		{Name: "auth", Handler: c.AuthHandlers},
	}

	return routes
}

type RouteGroup struct {
	Name    string
	Handler any
}

func (c *Container) Cleanup() {
	log.Println("🧹 Cleaning up container resources...")

	// ✅ Disconnect EventBus
	if c.EventBus != nil {
		log.Println("  📡 Disconnecting event bus...")
		ctx := context.Background()
		if err := c.EventBus.Disconnect(ctx); err != nil {
			log.Printf("  ⚠️  Failed to disconnect event bus: %v", err)
		}
	}

	// Close database
	if c.DB != nil {
		log.Println("  🗄️  Closing database connections...")
		c.DB.Close()
	}

	// Close Redis
	if c.RedisClient != nil {
		log.Println("  🔴 Closing Redis connections...")
		c.RedisClient.Close()
	}

	log.Println("✅ Container cleanup complete")
}

func (c *Container) HealthCheck() map[string]bool {
	health := make(map[string]bool)

	// Check database
	if c.DB != nil {
		err := c.DB.Ping()
		health["database"] = err == nil
	} else {
		health["database"] = false
	}

	// Check Redis
	if c.RedisClient != nil {
		err := c.RedisClient.Ping(c.RedisClient.Context()).Err()
		health["redis"] = err == nil
	} else {
		health["redis"] = false
	}

	// ✅ Check EventBus
	if c.EventBus != nil {
		health["event_bus"] = c.EventBus.IsConnected()
	} else {
		health["event_bus"] = false
	}

	return health
}

// ✅ GetEventBusMetrics returns event bus metrics
func (c *Container) GetEventBusMetrics() eventx.BusMetrics {
	if metricsbus, ok := c.EventBus.(eventx.MetricsEventBus); ok {
		return metricsbus.GetMetrics()
	}
	return eventx.BusMetrics{}
}

func (c *Container) GetServiceNames() []string {
	services := []string{
		"UserService",
		"TenantService",
		"RoleService",
		"EventBus",
	}

	return services
}

func (c *Container) GetRepositoryNames() []string {
	repos := []string{
		"UserRepo",
		"TenantRepo",
		"RoleRepo",
	}

	return repos
}

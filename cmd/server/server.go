package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Abraxas-365/craftable/errx/errxfiber"
	"github.com/Abraxas-365/relay/pkg/config"
	"github.com/Abraxas-365/relay/pkg/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

var startTime = time.Now()

func main() {
	// Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configurar logger
	setupLogger(cfg)

	log.Println("🚀 Starting Facturamelo API...")
	log.Printf("📍 Environment: %s", cfg.Server.Environment)

	// Conectar a PostgreSQL
	log.Println("🔌 Connecting to PostgreSQL...")
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.CloseDB(db)
	log.Println("✅ Connected to PostgreSQL")

	// Conectar a Redis
	log.Println("🔌 Connecting to Redis...")
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer database.CloseRedis(redisClient)
	log.Println("✅ Connected to Redis")

	// Inicializar contenedor de dependencias
	log.Println("📦 Initializing dependency container...")
	container := NewContainer(cfg, db, redisClient)
	defer container.Cleanup() // Cleanup on exit
	log.Println("✅ Dependencies initialized")

	// Verificar health de los servicios
	health := container.HealthCheck()
	log.Printf("🏥 Health check: Database=%v, Redis=%v, Scheduler=%v",
		health["database"], health["redis"], health["scheduler"])

	// =================================================================
	// INICIAR SCHEDULER
	// =================================================================
	log.Println("✅ Scheduler started")
	// =================================================================

	// Crear aplicación Fiber
	app := fiber.New(fiber.Config{
		AppName:      "Facturamelo API",
		ServerHeader: "Facturamelo",
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		ErrorHandler: errxfiber.FiberErrorHandler(),
	})

	// Configurar middleware global
	setupMiddleware(app, cfg)

	// Registrar rutas
	log.Println("🛣️  Setting up routes...")
	setupRoutes(app, container)
	log.Println("✅ Routes configured")

	// Log de servicios registrados
	log.Printf("📋 Registered services: %v", container.GetServiceNames())
	log.Printf("📋 Registered repositories: %v", container.GetRepositoryNames())

	// Iniciar servidor en goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.Server.Port)
		log.Printf("🚀 Server listening on %s", addr)
		log.Printf("🌍 Local: http://localhost%s", addr)
		if err := app.Listen(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("⏸️  Shutting down server...")

	// Apagado graceful del servidor Fiber
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Printf("❌ Error during server shutdown: %v", err)
	}

	log.Println("👋 Server stopped gracefully")
}

// setupLogger configura el logger
func setupLogger(cfg *config.Config) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if cfg.Server.Environment == "production" {
		// En producción podrías usar un logger más sofisticado
		log.SetFlags(log.LstdFlags)
	}
}

// setupMiddleware configura los middleware globales
func setupMiddleware(app *fiber.App, cfg *config.Config) {
	// Request ID
	app.Use(requestid.New())

	// Logger
	if cfg.Server.Environment != "test" {
		app.Use(logger.New(logger.Config{
			Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
		}))
	}

	// Recover de panics
	app.Use(recover.New())

	// CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     getCorsOrigins(cfg),
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	// Compression
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
}

// setupRoutes configura todas las rutas de la aplicación
func setupRoutes(app *fiber.App, c *Container) {

	log.Println("  ✓ Invoice routes registered")
	// Health check (enhanced)
	app.Get("/health", healthCheckHandler(c))

	// Root endpoint
	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{
			"message":  "Facturamelo API",
			"version":  "1.0.0",
			"status":   "running",
			"uptime":   time.Since(startTime).String(),
			"services": c.GetServiceNames(),
		})
	})

	// =================================================================
	// AUTH ROUTES
	// =================================================================
	c.AuthHandlers.RegisterRoutes(app)

	// =================================================================
	// BUSINESS ROUTES
	// =================================================================

	// Areas routes

	// =================================================================
	// DEBUG ROUTES (only in development)
	// =================================================================
	if c.Config.Server.Environment == "development" {
		app.Get("/debug/container", func(ctx *fiber.Ctx) error {
			return ctx.JSON(fiber.Map{
				"services":     c.GetServiceNames(),
				"repositories": c.GetRepositoryNames(),
				"health":       c.HealthCheck(),
			})
		})
	}

	// =================================================================
	// 404 HANDLER
	// =================================================================
	app.Use(func(ctx *fiber.Ctx) error {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Route not found",
			"path":  ctx.Path(),
		})
	})
}

// healthCheckHandler handler de health check mejorado
func healthCheckHandler(c *Container) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// Usar el método HealthCheck del container
		health := c.HealthCheck()

		// Determinar si todos los servicios están saludables
		allHealthy := true
		for _, healthy := range health {
			if !healthy {
				allHealthy = false
				break
			}
		}

		status := "healthy"
		statusCode := fiber.StatusOK

		if !allHealthy {
			status = "degraded"
			statusCode = fiber.StatusServiceUnavailable
		}

		return ctx.Status(statusCode).JSON(fiber.Map{
			"status":    status,
			"timestamp": time.Now(),
			"uptime":    time.Since(startTime).String(),
			"services":  health,
			"version":   "1.0.0",
		})
	}
}

// getCorsOrigins retorna los orígenes permitidos para CORS
func getCorsOrigins(cfg *config.Config) string {
	// Permite override via variable de entorno (lista separada por comas)
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		return origins
	}

	if cfg.Server.Environment == "production" {
		return "https://yourdomain.com"
	}

	// Evitar wildcard cuando AllowCredentials=true; usar orígenes comunes de desarrollo
	return "http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173,http://127.0.0.1:5173"
}

package config

import (
	"fmt"
	"os"
	"time"

	"github.com/Abraxas-365/relay/iam/auth"
)

// Config configuración principal de la aplicación
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     auth.Config
}

// ServerConfig configuración del servidor HTTP
type ServerConfig struct {
	Port            string
	Environment     string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig configuración de PostgreSQL
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig configuración de Redis
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// SIREConfig configuración de SIRE
type SIREConfig struct {
	BaseURL      string
	SecurityURL  string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
}

// Load carga la configuración desde variables de entorno
func Load() (*Config, error) {
	// Cargar .env si existe

	config := &Config{
		Server: ServerConfig{
			Port:            getEnv("PORT", "8080"),
			Environment:     getEnv("ENVIRONMENT", "development"),
			ReadTimeout:     getDurationEnv("READ_TIMEOUT", 10*time.Second),
			WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 10*time.Second),
			ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", getEnv("POSTGRES_HOST", "localhost")),
			Port:            getEnv("DB_PORT", getEnv("POSTGRES_PORT", "5432")),
			User:            getEnv("DB_USER", getEnv("POSTGRES_USER", "postgres")),
			Password:        getEnv("DB_PASSWORD", getEnv("POSTGRES_PASSWORD", "postgres")),
			DBName:          getEnv("DB_NAME", getEnv("POSTGRES_DB", "facturamelo")),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Auth: LoadAuthConfig(),
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// Validate valida la configuración
func (c *Config) Validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("DB_HOST is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USER is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("DB_NAME is required")
	}

	// Validar configuración de Auth
	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("invalid auth config: %w", err)
	}

	return nil
}

// GetDSN retorna el DSN de PostgreSQL
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// GetRedisAddr retorna la dirección de Redis
func (c *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// LoadAuthConfig carga la configuración desde variables de entorno
func LoadAuthConfig() auth.Config {
	return auth.Config{
		JWT: auth.JWTConfig{
			SecretKey:       getEnv("JWT_SECRET", "default-secret-change-in-production"),
			AccessTokenTTL:  getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
			Issuer:          getEnv("JWT_ISSUER", "facturamelo"),
		},
		OAuth: auth.OAuthConfigs{
			Google: auth.OAuthConfig{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/callback/google"),
				Scopes:       []string{"openid", "email", "profile"},
			},
			Microsoft: auth.OAuthConfig{
				ClientID:     getEnv("MICROSOFT_CLIENT_ID", ""),
				ClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("MICROSOFT_REDIRECT_URL", "http://localhost:8080/auth/callback/microsoft"),
				Scopes:       []string{"openid", "email", "profile", "User.Read"},
			},
		},
	}
}

export POSTGRES_DB = appdb
export POSTGRES_USER = appuser
export POSTGRES_PASSWORD = supersecret
export POSTGRES_HOST = localhost
export POSTGRES_PORT = 5449  # CHANGED FROM 5432


# Server
export PORT = 8080
export ENVIRONMENT = development
export READ_TIMEOUT = 10s
export WRITE_TIMEOUT = 10s
export SHUTDOWN_TIMEOUT = 30s
export CORS_ALLOWED_ORIGINS = http://localhost:3000,http://127.0.0.1:3000,http://localhost:5173,http://127.0.0.1:5173

# Database (DB_* map to POSTGRES_* by default)
export DB_HOST = $(POSTGRES_HOST)
export DB_PORT = $(POSTGRES_PORT)
export DB_USER = $(POSTGRES_USER)
export DB_PASSWORD = $(POSTGRES_PASSWORD)
export DB_NAME = $(POSTGRES_DB)
export DB_SSLMODE = disable
export DB_MAX_OPEN_CONNS = 25
export DB_MAX_IDLE_CONNS = 5
export DB_CONN_MAX_LIFETIME = 5m

# Redis
export REDIS_HOST = localhost
export REDIS_PORT = 6381
export REDIS_PASSWORD =
export REDIS_DB = 0

# Auth (JWT)
export JWT_SECRET = development-supersecret-32-characters-min-123456
export ACCESS_TOKEN_TTL = 15m
export REFRESH_TOKEN_TTL = 168h
export JWT_ISSUER = facturamelo

# OAuth (optional; leave CLIENT_ID/SECRET empty if unused)
export GOOGLE_CLIENT_ID = -
export GOOGLE_CLIENT_SECRET = -
export GOOGLE_REDIRECT_URL = http://localhost:5173/auth/callback/?provider=google
export MICROSOFT_CLIENT_ID =
export MICROSOFT_CLIENT_SECRET =
export MICROSOFT_REDIRECT_URL = http://localhost:8080/auth/callback/microsoft

# SIRE (optional)
export SIRE_BASE_URL = https://api-sire.sunat.gob.pe
export SIRE_SECURITY_URL = https://api-seguridad.sunat.gob.pe

# Build a standard PostgreSQL connection string
CONN_STRING = postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.PHONY: db-up db-down db-logs conn psql dev migrate seed clean

# Run the development server
dev:
	go mod tidy
	go run ./cmd/server

# Start database containers
db-up:
	docker compose up -d --remove-orphans

# Stop and remove database containers
db-down:
	docker compose down -v

# View database logs
db-logs:
	docker compose logs -f relay

# Show the connection string
conn:
	@echo $(CONN_STRING)

# Open psql in the container
psql:
	docker exec -it relay psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

# Run migrations (create tables)
migrate:
	@echo "Running migrations..."
	docker exec -i relay psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) < migrations/001_genesis.sql
	docker exec -i relay psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) < migrations/002_workflows.sql
	docker exec -i relay psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) < migrations/003_sire.sql
	@echo "✅ Migrations completed"

# Seed test data
seed:
	@echo "Seeding test data..."
	docker exec -i relay psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) < migrations/seed_test_data.sql
	@echo "✅ Test data seeded"

# Clean database (drop all tables)
clean:
	@echo "⚠️  Cleaning database..."
	docker exec -i relay psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "✅ Database cleaned"

# Full setup: up + migrate + seed
setup: db-up
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 3
	@make migrate
	@make seed
	@echo "✅ Database setup complete!"

# Full reset: clean + migrate + seed
reset:
	@make clean
	@make migrate
	@make seed
	@echo "✅ Database reset complete!"

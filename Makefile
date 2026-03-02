.PHONY: run build test generate db-up db-down clean help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ============================================================================
# SQLC
# ============================================================================

generate: ## Generate Go code from SQL (requires: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest)
	sqlc generate

verify: ## Verify sqlc configuration
	sqlc compile

# ============================================================================
# Development
# ============================================================================

run: ## Run the server
	go run ./cmd/server/main.go

build: ## Build binary
	go build -o bin/server ./cmd/server

test: ## Run tests
	go test -v -race ./...

# ============================================================================
# Database
# ============================================================================

db-up: ## Start PostgreSQL
	docker-compose up -d postgres

db-down: ## Stop PostgreSQL
	docker-compose down

db-reset: ## Reset database
	docker-compose down -v
	docker-compose up -d postgres
	sleep 3
	@echo "Database reset complete"

db-migrate: ## Run migrations
	docker exec -i postgres-db psql -U postgres -d postgres < migrations/001_schema.sql

db-shell: ## Open psql shell
	docker exec -it postgres-db psql -U postgres -d postgres

# ============================================================================
# Docker
# ============================================================================

docker-build: ## Build Docker image
	docker build -t postgres:latest .

docker-run: ## Run with Docker Compose
	docker-compose up --build

# ============================================================================
# Setup
# ============================================================================

setup: ## Install dependencies
	go mod download
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo "✓ Dependencies installed"
	@echo "✓ sqlc installed"
	@echo ""
	@echo "Next steps:"
	@echo "  1. make db-up      # Start PostgreSQL"
	@echo "  2. make db-migrate # Create tables"
	@echo "  3. make generate   # Generate sqlc code"
	@echo "  4. make run        # Start server"

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

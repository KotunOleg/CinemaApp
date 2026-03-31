.PHONY: run build test generate verify db-up db-down db-reset db-migrate db-shell docker-build docker-run setup frontend clean help

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ============================================================================
# SQLC
# ============================================================================

generate: ## Generate Go code from SQL
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

db-reset: ## Reset database (drops volumes)
	docker-compose down -v
	docker-compose up -d postgres
	sleep 3
	@echo "Database reset complete"

db-migrate: ## Run migrations
	go run ./cmd/migrate/main.go

db-shell: ## Open psql shell
	docker exec -it postgres-db psql -U postgres -d postgres

# ============================================================================
# Docker
# ============================================================================

docker-build: ## Build Docker image
	docker build -t cinema-app:latest .

docker-run: ## Run with Docker Compose
	docker-compose up --build

# ============================================================================
# Frontend
# ============================================================================

frontend: ## Build React frontend into static/
	cd frontend && npm install && npm run build

# ============================================================================
# Setup
# ============================================================================

setup: ## Install all dependencies
	go mod download
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	cd frontend && npm install
	@echo "Dependencies installed"
	@echo ""
	@echo "Next steps:"
	@echo "  1. make db-up      # Start PostgreSQL"
	@echo "  2. make db-migrate # Apply schema"
	@echo "  3. make frontend   # Build React app"
	@echo "  4. make run        # Start server on :8080"

clean: ## Clean build artifacts
	rm -rf bin/
	go clean

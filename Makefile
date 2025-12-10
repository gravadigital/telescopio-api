APP_NAME=telescopio-api
DOCKER_IMAGE=telescopio-api:latest
DOCKER_COMPOSE_FILE=docker-compose.yml
DOCKER_COMPOSE_DEV=docker-compose.dev.yml

.PHONY: help build run test clean docker-build docker-up docker-down docker-logs fmt lint mod-tidy dev-db-up dev-db-down dev-db-logs dev-db-connect dev-run wait-for-db

# Default target
.DEFAULT_GOAL := help

help: ## 📚 Show this help message
	@echo "Telescopio API - Available commands:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## 🚀 Build the application binary
	@echo "🚀 Building $(APP_NAME)..."
	@make check-go-version
	go build -o ./bin/$(APP_NAME) ./cmd/api/main.go

run: ## ▶️ Run the application locally
	@echo "▶️ Running $(APP_NAME)..."
	@make check-go-version
	go run ./cmd/api/main.go

dev-run: dev-db-up wait-for-db ## 🚀 Start PostgreSQL and run API (recommended for development)
	@echo "🚀 Starting API in development mode..."
	@make check-go-version
	go run ./cmd/api/main.go

test: ## ✅ Run all tests
	@echo "✅ Running tests..."
	go test -v ./...

test-coverage: ## 📊 Run tests with coverage report
	@echo "📊 Running tests with coverage..."
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "📈 Coverage report generated: coverage.html"

clean: ## 🧹 Clean build artifacts and coverage files
	@echo "🧹 Cleaning build artifacts..."
	rm -rf ./bin
	rm -f coverage.txt coverage.html

# Docker commands
docker-build: ## 🐳 Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-up: ## 🐳 Start all services with Docker Compose
	@echo "🐳 Starting services..."
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-up-build: ## 🐳 Build and start all services
	@echo "🐳 Building and starting services..."
	docker compose -f $(DOCKER_COMPOSE_FILE) up -d --build

docker-down: ## 🐳 Stop all services
	@echo "🐳 Stopping services..."
	docker compose -f $(DOCKER_COMPOSE_FILE) down

docker-down-volumes: ## ⚠️ Stop services and remove all volumes (data will be lost!)
	@echo "⚠️ Warning: This will remove all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) down -v; \
	fi

# Development Database Commands
dev-db-up: ## 🐳 Start PostgreSQL for development
	@echo "🐳 Starting PostgreSQL for development..."
	docker compose -f $(DOCKER_COMPOSE_DEV) up -d postgres
	@echo "✅ PostgreSQL started on localhost:5432"

dev-db-down: ## 🐳 Stop PostgreSQL development database
	@echo "🐳 Stopping PostgreSQL..."
	docker compose -f $(DOCKER_COMPOSE_DEV) stop postgres

dev-db-logs: ## 📄 Show PostgreSQL logs
	@echo "📄 Showing PostgreSQL logs..."
	docker compose -f $(DOCKER_COMPOSE_DEV) logs -f postgres

dev-db-connect: ## 🔗 Connect to development PostgreSQL database
	@echo "🔗 Connecting to development database..."
	docker compose -f $(DOCKER_COMPOSE_DEV) exec postgres psql -U telescopio -d telescopio_db

dev-db-reset: ## ⚠️ Reset development database (destroys all data!)
	@echo "⚠️ Warning: This will destroy all database data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker compose -f $(DOCKER_COMPOSE_DEV) down postgres; \
		docker volume rm telescopio-api_postgres_data_dev 2>/dev/null || true; \
		docker compose -f $(DOCKER_COMPOSE_DEV) up -d postgres; \
		sleep 5; \
		echo "✅ Database reset complete"; \
	fi

wait-for-db: ## ⏳ Wait for PostgreSQL to be ready
	@echo "⏳ Waiting for PostgreSQL to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker compose -f $(DOCKER_COMPOSE_DEV) exec -T postgres pg_isready -U telescopio -d telescopio_db > /dev/null 2>&1; then \
			echo "✅ PostgreSQL is ready!"; \
			exit 0; \
		fi; \
		echo "⏳ Waiting... ($$i/10)"; \
		sleep 2; \
	done; \
	echo "❌ PostgreSQL failed to start in time" && exit 1

docker-logs: ## 📄 Show logs from all services
	@echo "📄 Showing logs..."
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f

docker-logs-api: ## 📄 Show API service logs
	@echo "📄 Showing API logs..."
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f api

docker-logs-db: ## 📄 Show database service logs
	@echo "📄 Showing database logs..."
	docker compose -f $(DOCKER_COMPOSE_FILE) logs -f postgres

docker-restart: ## 🔄 Restart all services
	@echo "🔄 Restarting services..."
	docker compose -f $(DOCKER_COMPOSE_FILE) restart

docker-restart-api: ## 🔄 Restart only the API service
	@echo "🔄 Restarting API service..."
	docker compose -f $(DOCKER_COMPOSE_FILE) restart api

migrate: ## 💾 Run database migrations
	@echo "💾 Running database migrations..."
	go run ./cmd/migrate/main.go

migrate-rollback: ## ⏪ Rollback the last database migration
	@echo "⏪ Rolling back last migration..."
	go run ./cmd/migrate/main.go -rollback

db-connect: ## 🔗 Connect to the PostgreSQL database (production)
	@echo "🔗 Connecting to database..."
	docker compose -f $(DOCKER_COMPOSE_FILE) exec postgres psql -U telescopio -d telescopio_db

db-reset: ## ⚠️ Reset database (destroys all data!) - Production
	@echo "⚠️ Warning: This will destroy all database data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker compose -f $(DOCKER_COMPOSE_FILE) down postgres; \
		docker volume rm telescopio-api_postgres_data 2>/dev/null || true; \
		docker compose -f $(DOCKER_COMPOSE_FILE) up -d postgres; \
		sleep 5; \
		make migrate; \
	fi

fmt: ## ✨ Format Go code
	@echo "✨ Formatting code..."
	go fmt ./...

lint: ## 🔍 Run Go linter
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️ golangci-lint not found. Install it with:"; \
		echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

mod-tidy: ## 🧹 Tidy Go modules
	@echo "🧹 Tidying Go modules..."
	go mod tidy

mod-vendor: ## 📦 Vendor Go modules
	@echo "📦 Vendoring Go modules..."
	go mod vendor

# Development setup
setup: ## 🛠️ Setup development environment
	@echo "🛠️ Setting up development environment..."
	@if [ ! -f .env.example ]; then \
		echo "❌ .env.example file not found!"; \
		exit 1; \
	fi
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ Created .env file from .env.example"; \
	else \
		echo "⚠️ .env file already exists, skipping copy"; \
	fi
	@echo "✏️ Please edit the .env file with your configuration."

dev: setup dev-db-up ## 🚀 Full development setup (database only)
	@echo "✅ Development database is ready!"
	@echo "➡️ Database: localhost:5432"
	@echo "➡️ Run 'make dev-run' to start the API"
	@echo "➡️ Run 'make dev-full' to start all services including pgAdmin"

dev-full: setup ## 🚀 Full development environment with all services
	@echo "🚀 Starting all development services..."
	docker compose -f $(DOCKER_COMPOSE_DEV) up -d
	@echo "✅ Development environment is ready!"
	@echo "➡️ Database: localhost:5432"
	@echo "➡️ pgAdmin: http://localhost:5050"
	@echo "➡️ Redis: localhost:6379"
	@echo "➡️ Run 'make dev-run' to start the API"

# Production commands
prod-build: ## 🏭 Build for production
	@echo "🏭 Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ./bin/$(APP_NAME) ./cmd/api/main.go

prod-deploy: prod-build docker-build ## 🚀 Build and deploy for production
	@echo "🚀 Ready for production deployment."

# Health checks
health: ## 🩺 Check API health
	@echo "🩺 Checking API health..."
	@if command -v curl >/dev/null 2>&1; then \
		curl -f http://localhost:8080/health || echo "❌ API is not responding"; \
	else \
		echo "❌ curl not found. Install curl to use this command."; \
	fi

status: ## 📊 Show status of all services
	@echo "📊 Services status:"
	docker compose -f $(DOCKER_COMPOSE_FILE) ps

dev-status: ## 📊 Show status of development services
	@echo "📊 Development services status:"
	docker compose -f $(DOCKER_COMPOSE_DEV) ps

# Additional useful commands
check-deps: ## 🔍 Check if required dependencies are installed
	@echo "🔍 Checking dependencies..."
	@echo -n "Go: "; go version 2>/dev/null || echo "❌ Not found"
	@echo -n "Docker: "; docker --version 2>/dev/null || echo "❌ Not found"
	@echo -n "Docker Compose: "; docker compose version 2>/dev/null || echo "❌ Not found"
	@echo "📋 Expected Go version: 1.25.x"
	@echo -n "Current Go version: "; go version 2>/dev/null | grep -o 'go1\.[0-9]*\.[0-9]*' || echo "❌ Not found"

check-go-version: ## 🔍 Verify Go version consistency
	@echo "🔍 Checking Go version consistency..."
	@GO_VERSION=$$(go version 2>/dev/null | grep -o 'go1\.[0-9]*' | sed 's/go//'); \
	MOD_VERSION=$$(grep '^go ' go.mod | awk '{print $$2}' | grep -o '^1\.[0-9]*'); \
	if [ "$$GO_VERSION" != "$$MOD_VERSION" ]; then \
		echo "⚠️ Version mismatch detected:"; \
		echo "   Local Go: $$GO_VERSION"; \
		echo "   go.mod: $$MOD_VERSION"; \
		echo "   Run 'make fix-go-version' to update go.mod"; \
	else \
		echo "✅ Go versions are consistent: $$GO_VERSION"; \
	fi

fix-go-version: ## 🔧 Update go.mod to match local Go version
	@echo "🔧 Updating go.mod to match local Go version..."
	@GO_VERSION=$$(go version 2>/dev/null | grep -o 'go1\.[0-9]*\.[0-9]*' | sed 's/go//'); \
	if [ -n "$$GO_VERSION" ]; then \
		sed -i "s/^go .*/go $$GO_VERSION/" go.mod; \
		echo "✅ Updated go.mod to Go $$GO_VERSION"; \
		go mod tidy; \
	else \
		echo "❌ Could not detect Go version"; \
	fi

logs-follow: ## 📄 Follow logs from all services (alias for docker-logs)
	@make docker-logs

restart: ## 🔄 Restart all services (alias for docker-restart)
	@make docker-restart
APP_NAME=telescopio-api
DOCKER_IMAGE=telescopio-api:latest
DOCKER_COMPOSE_FILE=docker-compose.yml

.PHONY: help build run test clean docker-build docker-up docker-down docker-logs fmt lint mod-tidy

# Default target
.DEFAULT_GOAL := help

help:
	@echo "Telescopio API - Available commands:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*?##/ { printf "  %-20s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build:
	@echo "🚀 Building $(APP_NAME)..."
	go build -o ./bin/$(APP_NAME) ./cmd/api/main.go

run:
	@echo "▶️ Running $(APP_NAME)..."
	go run ./cmd/api/main.go

test:
	@echo "✅ Running tests..."
	go test -v ./...

test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "📈 Coverage report generated: coverage.html"

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf ./bin
	rm -f coverage.txt coverage.html

# Docker commands
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

docker-up:
	@echo "🐳 Starting services..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-up-build:
	@echo "🐳 Building and starting services..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) up -d --build

docker-down:
	@echo "🐳 Stopping services..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) down

docker-down-volumes:
	@echo "⚠️ Warning: This will remove all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f $(DOCKER_COMPOSE_FILE) down -v; \
	fi

docker-logs:
	@echo "📄 Showing logs..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

docker-logs-api:
	@echo "📄 Showing API logs..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f api

docker-logs-db:
	@echo "📄 Showing database logs..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f postgres

docker-restart:
	@echo "🔄 Restarting services..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) restart

docker-restart-api:
	@echo "🔄 Restarting API service..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) restart api

migrate:
	@echo "💾 Running database migrations..."
	go run ./cmd/migrate/main.go

migrate-rollback:
	@echo "⏪ Rolling back last migration..."
	go run ./cmd/migrate/main.go -rollback

db-connect:
	@echo "🔗 Connecting to database..."
	docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgres psql -U telescopio -d telescopio_db

db-reset:
	@echo "⚠️ Warning: This will destroy all database data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		docker-compose -f $(DOCKER_COMPOSE_FILE) down; \
		docker volume rm $$(docker volume ls -q | grep postgres) 2>/dev/null || true; \
		docker-compose -f $(DOCKER_COMPOSE_FILE) up -d postgres; \
		sleep 5; \
		make migrate; \
	fi

fmt:
	@echo "✨ Formatting code..."
	go fmt ./...

lint:
	@echo "🔍 Running linter..."
	golangci-lint run

mod-tidy:
	@echo "🧹 Tidying Go modules..."
	go mod tidy

mod-vendor:
	@echo "📦 Vendoring Go modules..."
	go mod vendor

# Development setup
setup: ## 🛠️ Setup development environment
	@echo "🛠️ Setting up development environment..."
	cp .env.example .env
	@echo "✏️ Please edit the .env file with your configuration."

dev: setup docker-up ## 🚀 Full development setup
	@echo "✅ Development environment is ready!"
	@echo "➡️ API will be available at: http://localhost:8080"
	@echo "➡️ pgAdmin will be available at: http://localhost:5050"

# Production commands
prod-build: ## 🏭 Build for production
	@echo "🏭 Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o ./bin/$(APP_NAME) ./cmd/api/main.go

prod-deploy: prod-build docker-build ## 🚀 Build and deploy for production
	@echo "🚀 Ready for production deployment."

# Health checks
health: ## 🩺 Check API health
	@echo "🩺 Checking API health..."
	@curl -f http://localhost:8080/health || echo "❌ API is not responding"

status: ## 📊 Show status of all services
	@echo "📊 Services status:"
	docker-compose -f $(DOCKER_COMPOSE_FILE) ps
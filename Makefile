.PHONY: help dev-up dev-down dev-restart migrate-up migrate-down seed-dev test build clean

# Colors for terminal output
COLOR_RESET = \033[0m
COLOR_BOLD = \033[1m
COLOR_GREEN = \033[32m
COLOR_YELLOW = \033[33m
COLOR_BLUE = \033[34m

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "$(COLOR_BOLD)Unity Collaboration Platform - Development Commands$(COLOR_RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'
	@echo ""

# Development Environment
dev-up: ## Start all development services with Docker Compose
	@echo "$(COLOR_BLUE)Starting development environment...$(COLOR_RESET)"
	docker-compose -f docker-compose.dev.yml up -d
	@echo "$(COLOR_GREEN)Development environment started!$(COLOR_RESET)"
	@echo ""
	@echo "Services available at:"
	@echo "  PostgreSQL:    localhost:5432"
	@echo "  Redis:         localhost:6379"
	@echo "  MongoDB:       localhost:27017"
	@echo "  RabbitMQ:      localhost:5672 (UI: localhost:15672)"
	@echo "  MinIO:         localhost:9000 (Console: localhost:9001)"
	@echo "  Prometheus:    localhost:9090"
	@echo "  Grafana:       localhost:3000 (admin/admin)"
	@echo "  Jaeger:        localhost:16686"

dev-down: ## Stop all development services
	@echo "$(COLOR_YELLOW)Stopping development environment...$(COLOR_RESET)"
	docker-compose -f docker-compose.dev.yml down
	@echo "$(COLOR_GREEN)Development environment stopped!$(COLOR_RESET)"

dev-restart: dev-down dev-up ## Restart all development services

dev-logs: ## Show logs from all services
	docker-compose -f docker-compose.dev.yml logs -f

dev-ps: ## Show status of all services
	docker-compose -f docker-compose.dev.yml ps

# Database Management
migrate-up: ## Run database migrations
	@echo "$(COLOR_BLUE)Running database migrations...$(COLOR_RESET)"
	@if [ -d "migrations" ]; then \
		docker exec -i collab_postgres psql -U dev -d collab_dev < migrations/001_initial_schema.sql || true; \
	fi
	@echo "$(COLOR_GREEN)Migrations completed!$(COLOR_RESET)"

migrate-down: ## Rollback database migrations
	@echo "$(COLOR_YELLOW)Rolling back migrations...$(COLOR_RESET)"
	docker exec -i collab_postgres psql -U dev -d collab_dev -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	@echo "$(COLOR_GREEN)Migrations rolled back!$(COLOR_RESET)"

seed-dev: ## Seed database with development data
	@echo "$(COLOR_BLUE)Seeding development data...$(COLOR_RESET)"
	@if [ -f "migrations/seed.sql" ]; then \
		docker exec -i collab_postgres psql -U dev -d collab_dev < migrations/seed.sql; \
	fi
	@echo "$(COLOR_GREEN)Database seeded!$(COLOR_RESET)"

db-shell: ## Open PostgreSQL shell
	docker exec -it collab_postgres psql -U dev -d collab_dev

redis-cli: ## Open Redis CLI
	docker exec -it collab_redis redis-cli -a devpass

mongo-shell: ## Open MongoDB shell
	docker exec -it collab_mongodb mongosh -u dev -p devpass

# Build Services
build-session: ## Build Session Service
	@echo "$(COLOR_BLUE)Building Session Service...$(COLOR_RESET)"
	cd services/session && go build -o ../../bin/session cmd/server/main.go
	@echo "$(COLOR_GREEN)Session Service built!$(COLOR_RESET)"

build-asset: ## Build Asset Service
	@echo "$(COLOR_BLUE)Building Asset Service...$(COLOR_RESET)"
	cd services/asset && go build -o ../../bin/asset cmd/server/main.go
	@echo "$(COLOR_GREEN)Asset Service built!$(COLOR_RESET)"

build-auth: ## Build Auth Service
	@echo "$(COLOR_BLUE)Building Auth Service...$(COLOR_RESET)"
	cd services/auth && go build -o ../../bin/auth cmd/server/main.go
	@echo "$(COLOR_GREEN)Auth Service built!$(COLOR_RESET)"

build-sync: ## Build Sync Service
	@echo "$(COLOR_BLUE)Building Sync Service...$(COLOR_RESET)"
	cd services/sync && npm run build
	@echo "$(COLOR_GREEN)Sync Service built!$(COLOR_RESET)"

build-all: build-session build-asset build-auth build-sync ## Build all services

# Run Services
run-session: ## Run Session Service
	cd services/session && go run cmd/server/main.go

run-asset: ## Run Asset Service
	cd services/asset && go run cmd/server/main.go

run-auth: ## Run Auth Service
	cd services/auth && go run cmd/server/main.go

run-sync: ## Run Sync Service
	cd services/sync && npm run dev

run-gateway: ## Run API Gateway
	cd gateway && npm run dev

# Testing
test-session: ## Run Session Service tests
	cd services/session && go test ./... -v -cover

test-asset: ## Run Asset Service tests
	cd services/asset && go test ./... -v -cover

test-auth: ## Run Auth Service tests
	cd services/auth && go test ./... -v -cover

test-sync: ## Run Sync Service tests
	cd services/sync && npm test

test-all: ## Run all tests
	@echo "$(COLOR_BLUE)Running all tests...$(COLOR_RESET)"
	@$(MAKE) test-session
	@$(MAKE) test-asset
	@$(MAKE) test-auth
	@$(MAKE) test-sync
	@echo "$(COLOR_GREEN)All tests completed!$(COLOR_RESET)"

# Code Quality
lint-go: ## Run Go linter
	@echo "$(COLOR_BLUE)Running Go linter...$(COLOR_RESET)"
	@for service in session asset auth; do \
		echo "Linting $$service service..."; \
		cd services/$$service && golangci-lint run ./... && cd ../..; \
	done

lint-ts: ## Run TypeScript linter
	@echo "$(COLOR_BLUE)Running TypeScript linter...$(COLOR_RESET)"
	cd services/sync && npm run lint
	cd gateway && npm run lint

format-go: ## Format Go code
	@echo "$(COLOR_BLUE)Formatting Go code...$(COLOR_RESET)"
	@for service in session asset auth; do \
		cd services/$$service && gofmt -w . && cd ../..; \
	done

format-ts: ## Format TypeScript code
	@echo "$(COLOR_BLUE)Formatting TypeScript code...$(COLOR_RESET)"
	cd services/sync && npm run format
	cd gateway && npm run format

# Clean
clean: ## Clean build artifacts and dependencies
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	rm -rf bin/
	rm -rf dist/
	@for service in session asset auth; do \
		rm -rf services/$$service/bin; \
	done
	cd services/sync && rm -rf dist/ node_modules/
	cd gateway && rm -rf dist/ node_modules/
	@echo "$(COLOR_GREEN)Cleanup completed!$(COLOR_RESET)"

clean-all: clean dev-down ## Clean everything including Docker volumes
	@echo "$(COLOR_YELLOW)Removing Docker volumes...$(COLOR_RESET)"
	docker-compose -f docker-compose.dev.yml down -v
	@echo "$(COLOR_GREEN)Full cleanup completed!$(COLOR_RESET)"

# Installation
install-deps: ## Install all dependencies
	@echo "$(COLOR_BLUE)Installing dependencies...$(COLOR_RESET)"
	@for service in session asset auth; do \
		echo "Installing Go dependencies for $$service..."; \
		cd services/$$service && go mod download && cd ../..; \
	done
	cd services/sync && npm install
	cd gateway && npm install
	@echo "$(COLOR_GREEN)Dependencies installed!$(COLOR_RESET)"

# Proto generation
proto-gen: ## Generate code from Protocol Buffers
	@echo "$(COLOR_BLUE)Generating protobuf code...$(COLOR_RESET)"
	@for service in session asset auth; do \
		if [ -d "services/$$service/proto" ]; then \
			cd services/$$service && protoc --go_out=. --go_opt=paths=source_relative \
				--go-grpc_out=. --go-grpc_opt=paths=source_relative \
				proto/*.proto && cd ../..; \
		fi; \
	done
	@echo "$(COLOR_GREEN)Protobuf code generated!$(COLOR_RESET)"

# Quick Start
setup: dev-up migrate-up seed-dev install-deps ## Complete setup (Docker + DB + Dependencies)
	@echo "$(COLOR_GREEN)$(COLOR_BOLD)Setup completed! You can now start developing.$(COLOR_RESET)"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Run 'make run-session' to start Session Service"
	@echo "  2. Run 'make run-sync' to start Sync Service"
	@echo "  3. Run 'make run-asset' to start Asset Service"
	@echo "  4. Run 'make run-auth' to start Auth Service"
	@echo "  5. Run 'make run-gateway' to start API Gateway"

# Docker build
docker-build-session: ## Build Session Service Docker image
	docker build -t collab/session:dev -f services/session/Dockerfile services/session

docker-build-asset: ## Build Asset Service Docker image
	docker build -t collab/asset:dev -f services/asset/Dockerfile services/asset

docker-build-auth: ## Build Auth Service Docker image
	docker build -t collab/auth:dev -f services/auth/Dockerfile services/auth

docker-build-sync: ## Build Sync Service Docker image
	docker build -t collab/sync:dev -f services/sync/Dockerfile services/sync

docker-build-all: docker-build-session docker-build-asset docker-build-auth docker-build-sync ## Build all Docker images

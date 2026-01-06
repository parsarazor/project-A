# URL Shortener Makefile
# Provides convenient commands for development, testing, and deployment

.PHONY: help build run dev test test-cover lint clean docker-build docker-up docker-down docker-logs migrate seed-admin

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Binary names
BINARY_NAME=url-shortener
MIGRATE_BINARY=migrate

# Directories
CMD_DIR=./cmd/api
MIGRATE_DIR=./cmd/migrate
BUILD_DIR=./build

# Docker
DOCKER_COMPOSE=docker-compose
DOCKER_COMPOSE_PROD=docker-compose -f docker-compose.prod.yml

# Colors for terminal output
GREEN=\033[0;32m
YELLOW=\033[0;33m
RED=\033[0;31m
NC=\033[0m # No Color

## help: Display this help message
help:
	@echo "URL Shortener - Available Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Development:"
	@echo "  dev           Start development server with hot reload (requires air)"
	@echo "  run           Run the application directly"
	@echo "  build         Build the application binary"
	@echo "  clean         Remove build artifacts"
	@echo ""
	@echo "Database:"
	@echo "  migrate       Run database migrations"
	@echo "  seed-admin    Create an admin user"
	@echo ""
	@echo "Testing:"
	@echo "  test          Run all tests"
	@echo "  test-cover    Run tests with coverage report"
	@echo "  test-verbose  Run tests with verbose output"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint          Run linter (requires golangci-lint)"
	@echo "  fmt           Format code"
	@echo "  vet           Run go vet"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build  Build Docker images"
	@echo "  docker-up     Start containers (development)"
	@echo "  docker-down   Stop containers"
	@echo "  docker-logs   View container logs"
	@echo "  docker-shell  Open shell in app container"
	@echo ""
	@echo "Production:"
	@echo "  docker-prod-build  Build production Docker images"
	@echo "  docker-prod-up     Start production containers"
	@echo "  docker-prod-down   Stop production containers"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps          Download dependencies"
	@echo "  deps-update   Update dependencies"
	@echo "  deps-tidy     Tidy dependencies"

## dev: Start development server with hot reload
dev:
	@echo "$(GREEN)Starting development server with hot reload...$(NC)"
	@air -c .air.toml

## run: Run the application directly
run:
	@echo "$(GREEN)Starting application...$(NC)"
	$(GORUN) $(CMD_DIR)/main.go

## build: Build the application binary
build:
	@echo "$(GREEN)Building application...$(NC)"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	$(GOBUILD) -o $(BUILD_DIR)/$(MIGRATE_BINARY) $(MIGRATE_DIR)/main.go
	@echo "$(GREEN)Build complete: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

## build-linux: Build for Linux (useful for Docker)
build-linux:
	@echo "$(GREEN)Building for Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(CMD_DIR)/main.go

## clean: Remove build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Clean complete$(NC)"

## migrate: Run database migrations
migrate:
	@echo "$(GREEN)Running database migrations...$(NC)"
	$(GORUN) $(MIGRATE_DIR)/main.go

## seed-admin: Create an admin user
seed-admin:
	@echo "$(GREEN)Creating admin user...$(NC)"
	$(GORUN) $(MIGRATE_DIR)/main.go -seed-admin

## seed-admin-custom: Create admin with custom credentials
seed-admin-custom:
	@echo "$(GREEN)Creating admin user with custom credentials...$(NC)"
	@read -p "Enter admin email: " email; \
	read -p "Enter admin password: " password; \
	read -p "Enter admin name: " name; \
	$(GORUN) $(MIGRATE_DIR)/main.go -seed-admin -admin-email="$$email" -admin-password="$$password" -admin-name="$$name"

## test: Run all tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GOTEST) ./...

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "$(GREEN)Running tests (verbose)...$(NC)"
	$(GOTEST) -v ./...

## test-cover: Run tests with coverage
test-cover:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## test-unit: Run only unit tests
test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	$(GOTEST) ./internal/...

## test-integration: Run integration tests
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	$(GOTEST) ./tests/...

## lint: Run linter
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	$(GOLINT) run ./...

## fmt: Format code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GOFMT) -s -w .

## vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	$(GOCMD) vet ./...

## deps: Download dependencies
deps:
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	$(GOMOD) download

## deps-update: Update dependencies
deps-update:
	@echo "$(GREEN)Updating dependencies...$(NC)"
	$(GOGET) -u ./...
	$(GOMOD) tidy

## deps-tidy: Tidy dependencies
deps-tidy:
	@echo "$(GREEN)Tidying dependencies...$(NC)"
	$(GOMOD) tidy

## docker-build: Build Docker images for development
docker-build:
	@echo "$(GREEN)Building Docker images...$(NC)"
	$(DOCKER_COMPOSE) build

## docker-up: Start development containers
docker-up:
	@echo "$(GREEN)Starting containers...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Containers started. API available at http://localhost:8080$(NC)"
	@echo "$(GREEN)Adminer available at http://localhost:8081$(NC)"

## docker-down: Stop containers
docker-down:
	@echo "$(YELLOW)Stopping containers...$(NC)"
	$(DOCKER_COMPOSE) down

## docker-logs: View container logs
docker-logs:
	$(DOCKER_COMPOSE) logs -f app

## docker-shell: Open shell in app container
docker-shell:
	$(DOCKER_COMPOSE) exec app sh

## docker-db-shell: Open psql shell in database container
docker-db-shell:
	$(DOCKER_COMPOSE) exec db psql -U postgres -d urlshortener

## docker-restart: Restart containers
docker-restart: docker-down docker-up

## docker-clean: Remove containers, volumes, and images
docker-clean:
	@echo "$(RED)Removing containers, volumes, and images...$(NC)"
	$(DOCKER_COMPOSE) down -v --rmi local
	@echo "$(GREEN)Cleanup complete$(NC)"

## docker-prod-build: Build production Docker images
docker-prod-build:
	@echo "$(GREEN)Building production Docker images...$(NC)"
	$(DOCKER_COMPOSE_PROD) build

## docker-prod-up: Start production containers
docker-prod-up:
	@echo "$(GREEN)Starting production containers...$(NC)"
	$(DOCKER_COMPOSE_PROD) up -d

## docker-prod-down: Stop production containers
docker-prod-down:
	@echo "$(YELLOW)Stopping production containers...$(NC)"
	$(DOCKER_COMPOSE_PROD) down

## docker-prod-logs: View production logs
docker-prod-logs:
	$(DOCKER_COMPOSE_PROD) logs -f

## setup: Initial project setup
setup: deps
	@echo "$(GREEN)Setting up project...$(NC)"
	@cp -n .env.example .env 2>/dev/null || true
	@echo "$(GREEN)Project setup complete!$(NC)"
	@echo "$(YELLOW)Don't forget to update .env with your settings$(NC)"

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "$(GREEN)All checks passed!$(NC)"

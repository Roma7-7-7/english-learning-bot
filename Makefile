.PHONY: help deps lint fix-nolint test vet clean
.PHONY: build build-local build-bot build-backend build-web
.PHONY: build-bot-local run-web
.PHONY: docker-build docker-up docker-down

# ==========================================
# Configuration
# ==========================================

# Version info (can be overridden by CI)
VERSION ?= dev
BUILD_TIME ?= $(shell date -u +%Y%m%d-%H%M%S)

# Target architecture
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Build settings
BIN_DIR := ./bin
BOT_BIN := $(BIN_DIR)/english-learning-bot

# CGO was required for go-sqlite3 but we switched to modernc.org/sqlite
CGO_ENABLED := 0

# Build flags
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# Go commands
GO := go
GOTEST := $(GO) test
GOVET := $(GO) vet
GOBUILD := $(GO) build

# ==========================================
# Help
# ==========================================

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# ==========================================
# Dependencies
# ==========================================

deps: ## Download Go dependencies
	$(GO) mod download

# ==========================================
# Testing & Linting
# ==========================================

test: ## Run tests
	$(GOTEST) -v ./...

vet: ## Run go vet
	$(GOVET) ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

fix-nolint: ## Fix nolint comments (remove space after //)
	find . -type f -name "*.go" -exec sed -i '' 's|// nolint|//nolint|g' {} +

# ==========================================
# Local Development Builds
# ==========================================

build-bot-local: deps ## Build bot for local development (native OS/arch)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BOT_BIN) ./cmd/bot

# Aliases for convenience
build-bot: build-bot-local
build-backend: build-bot-local

# ==========================================
# Web Frontend
# ==========================================

build-web: ## Build web frontend
	cd web && npm install && npm run build

# ==========================================
# Combined Builds
# ==========================================

build: build-backend build-web ## Build everything (backend + frontend)

build-local: build-backend build-web ## Build everything for local development

# ==========================================
# Clean
# ==========================================

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
	@echo "Cleaned build directory"

# ==========================================
# Development
# ==========================================

run-web: ## Run web development server
	cd web && npm run dev

# ==========================================
# Docker
# ==========================================

docker-build: ## Build Docker images
	VERSION=$(VERSION) BUILD_TIME=$(BUILD_TIME) docker-compose build

# Usage: BOT_TELEGRAM_TOKEN=<token> BOT_TELEGRAM_ALLOWED_CHAT_IDS=<ids> make docker-up
# Web UI: http://localhost:3000 (nginx proxies API requests to bot)
# Bot API: http://localhost:8080 (direct access for debugging)
docker-up: ## Build and run with Docker Compose
	BOT_TELEGRAM_TOKEN=$(BOT_TELEGRAM_TOKEN) BOT_TELEGRAM_ALLOWED_CHAT_IDS=$(BOT_TELEGRAM_ALLOWED_CHAT_IDS) \
	VERSION=$(VERSION) BUILD_TIME=$(BUILD_TIME) \
	docker-compose up --build -d

docker-down: ## Stop Docker Compose services
	docker-compose down

# ==========================================
# CI/CD Targets
# ==========================================

ci-test: test vet ## Run all CI tests

# ==========================================
# Info
# ==========================================

info: ## Show build configuration
	@echo "Build Configuration:"
	@echo "  VERSION:     $(VERSION)"
	@echo "  BUILD_TIME:  $(BUILD_TIME)"
	@echo "  GOOS:        $(GOOS)"
	@echo "  GOARCH:      $(GOARCH)"
	@echo "  CGO_ENABLED: $(CGO_ENABLED)"
	@echo "  BIN_DIR:     $(BIN_DIR)"
	@echo "  LDFLAGS:     $(LDFLAGS)"

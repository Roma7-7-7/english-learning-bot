.PHONY: help deps lint fix-nolint test vet clean
.PHONY: build build-local build-release build-api build-bot build-backend build-web
.PHONY: build-api-local build-bot-local build-api-release build-bot-release
.PHONY: version-file docker-build docker-run docker-compose run-web

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
API_BIN := $(BIN_DIR)/english-learning-api
BOT_BIN := $(BIN_DIR)/english-learning-bot

# CGO was required for go-sqlite3 but we switched to modernc.org/sqlite
CGO_ENABLED := 0

# Build flags
LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)
# Release flags: strip debug info and symbol table for smaller binaries
LDFLAGS_RELEASE := $(LDFLAGS) -w -s

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

build-api-local: deps ## Build API for local development (native OS/arch)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(API_BIN) ./cmd/api

build-bot-local: deps ## Build bot for local development (native OS/arch)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BOT_BIN) ./cmd/bot

build-backend-local: build-api-local build-bot-local ## Build both API and bot for local development

# Aliases for convenience
build-api: build-api-local
build-bot: build-bot-local
build-backend: build-backend-local

# ==========================================
# Release Builds (Multi-Architecture)
# ==========================================

build-api-release: deps ## Build API for production (AMD64 and ARM64)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 \
	$(GOBUILD) -ldflags="$(LDFLAGS_RELEASE)" -o $(API_BIN)-amd64 ./cmd/api
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 \
	$(GOBUILD) -ldflags="$(LDFLAGS_RELEASE)" -o $(API_BIN)-arm64 ./cmd/api
	@echo "Built API binaries:"
	@echo "  $(API_BIN)-amd64 (linux/amd64)"
	@echo "  $(API_BIN)-arm64 (linux/arm64)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"

build-bot-release: deps ## Build bot for production (AMD64 and ARM64)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=amd64 \
	$(GOBUILD) -ldflags="$(LDFLAGS_RELEASE)" -o $(BOT_BIN)-amd64 ./cmd/bot
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=arm64 \
	$(GOBUILD) -ldflags="$(LDFLAGS_RELEASE)" -o $(BOT_BIN)-arm64 ./cmd/bot
	@echo "Built Bot binaries:"
	@echo "  $(BOT_BIN)-amd64 (linux/amd64)"
	@echo "  $(BOT_BIN)-arm64 (linux/arm64)"
	@echo "  Version: $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"

build-release: build-api-release build-bot-release version-file ## Build all binaries for production
	@chmod +x $(BIN_DIR)/*
	@ls -lh $(BIN_DIR)

# ==========================================
# Version File
# ==========================================

version-file: ## Create VERSION file with build metadata
	@mkdir -p $(BIN_DIR)
	@echo "$(VERSION)" > $(BIN_DIR)/VERSION
	@echo "Built at: $(BUILD_TIME)" >> $(BIN_DIR)/VERSION
	@echo "Commit: $(shell git rev-parse HEAD 2>/dev/null || echo 'unknown')" >> $(BIN_DIR)/VERSION
	@echo "Created $(BIN_DIR)/VERSION"

# ==========================================
# Web Frontend
# ==========================================

build-web: ## Build web frontend
	cd web && npm install && npm run build

# ==========================================
# Combined Builds
# ==========================================

build: build-backend build-web ## Build everything (backend + frontend)

build-local: build-backend-local build-web ## Build everything for local development

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
# CI/CD Targets
# ==========================================

ci-test: test vet ## Run all CI tests

ci-build: build-release ## Build release binaries for CI

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

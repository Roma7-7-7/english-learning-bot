
# Default target architecture for local development
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

lint:
	golangci-lint run ./...

# Goland automatically adds a space after "//" when adding a comment, so we need to remove it (yes, I'm too lazy to fix it in Goland configuration)
fix-nolint:
	find . -type f -name "*.go" -exec sed -i '' 's|// nolint|//nolint|g' {} +

deps:
	go mod download

# Local builds (uses your current OS/arch)
build-bot: deps
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ./bin/english-learning-bot ./cmd/bot/main.go

build-api: deps
	CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ./bin/english-learning-api ./cmd/api/main.go

build-web:
	cd web && npm install && npm run build

build-backend: build-bot build-api

build: build-backend build-web

# Clean build artifacts
clean:
	rm -rf ./bin/

# Run
run-web:
	cd web && npm run dev


# Default target architecture for local development
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# AWS EC2
AWS_GOOS = linux
AWS_GOARCH = amd64

lint:
	golangci-lint run ./...

# Goland automatically adds a space after "//" when adding a comment, so we need to remove it (yes, I'm too lazy to fix it in Goland configuration)
fix-nolint:
	find . -type f -name "*.go" -exec sed -i '' 's|// nolint|//nolint|g' {} +

deps:
	go mod download

# Local builds (uses your current OS/arch)
build-bot: deps
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ./bin/english-learning-bot ./cmd/bot/main.go

build-api: deps
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o ./bin/english-learning-api ./cmd/api/main.go

build-web:
	cd web && npm install && npm run build

build: build-bot build-api build-web

# Linux x86_64 builds
build-bot-linux: deps
	CGO_ENABLED=0 GOOS=$(AWS_GOOS) GOARCH=$(AWS_GOARCH) go build -o ./bin/english-learning-bot-linux ./cmd/bot/main.go

build-api-linux: deps
	CGO_ENABLED=0 GOOS=$(AWS_GOOS) GOARCH=$(AWS_GOARCH) go build -o ./bin/english-learning-api-linux ./cmd/api/main.go

build-linux: build-web build-bot-linux build-linux-aws

# Clean build artifacts
clean:
	rm -rf ./bin/

# Run
run-web:
	cd web && npm run dev

docker-build:
	docker build -t english-learning-bot .

docker-run:
	docker run --network shared -e ENV="${ENV}" -e TELEGRAM_TOKEN="${TELEGRAM_TOKEN}" -e ALLOWED_CHAT_IDS="${ALLOWED_CHAT_IDS}" -e DB_URL="${DB_URL}" -e PUBLISH_INTERVAL="${PUBLISH_INTERVAL}" --restart always -d --name english-learning-bot english-learning-bot

docker-compose:
	docker-compose down
	docker-compose up -d

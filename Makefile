include .env

lint:
	golangci-lint run ./...

build:
	go mod download
	CGO_ENABLED=0 go build -o ./bin/english-learning-bot ./cmd/bot/main.go

docker-build:
	docker build -t english-learning-bot .

docker-run:
	docker run --network shared -e ENV="${ENV}" -e TELEGRAM_TOKEN="${TELEGRAM_TOKEN}" -e ALLOWED_CHAT_IDS="${ALLOWED_CHAT_IDS}" -e DB_URL="${DB_URL}" -e PUBLISH_INTERVAL="${PUBLISH_INTERVAL}" --restart always -d --name english-learning-bot english-learning-bot

docker-compose:
	docker-compose down
	docker-compose up -d

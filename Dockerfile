# ==========================================
# Bot (Go backend) - Multi-stage build
# ==========================================

# --- Build stage ---
FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG BUILD_TIME=unknown

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/

RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o /english-learning-bot ./cmd/bot

# --- Runtime stage ---
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1000 appuser
WORKDIR /app

COPY --from=builder /english-learning-bot .
COPY schema/ schema/

RUN mkdir -p /app/data && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["./english-learning-bot"]

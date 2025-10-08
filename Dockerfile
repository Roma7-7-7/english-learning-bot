# build
FROM golang:1.24.1-alpine3.20 AS build

RUN apk add --no-cache make

COPY . /app
WORKDIR /app
RUN make build

# run
FROM alpine:3.22

ENV ENV="prod"
ENV TELEGRAM_TOKEN=""
ENV ALLOWED_CHAT_IDS=""
ENV DB_URL=""
ENV PUBLISH_INTERVAL="1h"

RUN apk add --no-cache tzdata

COPY --from=build /app/bin/english-learning-bot /app/english-learning-bot

WORKDIR /app

ENTRYPOINT ["/app/english-learning-bot"]

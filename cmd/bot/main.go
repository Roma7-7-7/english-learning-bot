package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/schedule"
	"github.com/Roma7-7-7/english-learning-bot/internal/telegram"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

var (
	envEnvVar             = os.Getenv("ENV")
	telegramTokenEnvVar   = os.Getenv("TELEGRAM_TOKEN")
	allowedChatIDsEnvVar  = os.Getenv("ALLOWED_CHAT_IDS")
	dbURLEnvVar           = os.Getenv("DB_URL")
	publishIntervalEnvVar = os.Getenv("PUBLISH_INTERVAL")
	allowedChatIDs        []int64
	publishInterval       time.Duration
)

func main() {
	ctx := context.Background()
	log := mustLogger()
	repo, err := dal.NewPostgresqlRepository(ctx, dbURLEnvVar)
	if err != nil {
		log.Error("failed to create repository", "error", err)
		return
	}
	defer repo.Close()

	bot, err := telegram.NewBot(telegramTokenEnvVar, repo, log, telegram.Recover(log), telegram.LogErrors(log), telegram.AllowedChats(allowedChatIDs))
	if err != nil {
		log.Error("failed to create bot", "error", err)
		return
	}

	go func() {
		err := schedule.StartWordCheckSchedule(ctx, allowedChatIDs, publishInterval, bot, log)
		if err != nil {
			log.Error("failed to start word check schedule", "error", err)
		}
	}()

	bot.Start()
}

func mustLogger() *slog.Logger {
	var handler slog.Handler
	if envEnvVar == envDev {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}
	return slog.New(handler)
}

func init() {
	if envEnvVar == "" {
		envEnvVar = envProd
	}
	if envEnvVar != envDev && envEnvVar != envProd {
		panic("invalid ENV value, must be 'dev' or 'prod'")
	}
	if telegramTokenEnvVar == "" {
		panic("TELEGRAM_TOKEN is required")
	}
	if allowedChatIDsEnvVar != "" {
		chatIDStrings := strings.Split(allowedChatIDsEnvVar, ",")
		for _, chatIDString := range chatIDStrings {
			chatID, err := strconv.ParseInt(chatIDString, 10, 64)
			if err != nil {
				panic("invalid chat ID " + chatIDString)
			}
			allowedChatIDs = append(allowedChatIDs, chatID)
		}
	}
	if dbURLEnvVar == "" {
		panic("DB_URL is required")
	}
	if publishIntervalEnvVar == "" {
		publishIntervalEnvVar = "1h"
	}
	var err error
	publishInterval, err = time.ParseDuration(publishIntervalEnvVar)
	if err != nil {
		panic("invalid PUBLISH_INTERVAL value")
	}
}

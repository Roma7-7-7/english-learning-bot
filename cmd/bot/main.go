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
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	envDev  = "dev"
	envProd = "prod"

	batchSize          = 50
	guessedStreakLimit = 15
)

// nolint: gochecknoglobals // configuration in main file
var (
	envEnvVar             = os.Getenv("ENV")
	telegramTokenEnvVar   = os.Getenv("TELEGRAM_TOKEN")
	allowedChatIDsEnvVar  = os.Getenv("ALLOWED_CHAT_IDS")
	dbURLEnvVar           = os.Getenv("DB_URL")
	publishIntervalEnvVar = os.Getenv("PUBLISH_INTERVAL")
	allowedChatIDs        []int64
	publishInterval       time.Duration
	kyivLocation          *time.Location
)

func main() {
	ctx := context.Background()
	log := mustLogger()
	log.InfoContext(ctx, "initialized", "kyiv_time", time.Now().In(kyivLocation).Format(time.RFC3339))
	db, err := pgxpool.New(ctx, dbURLEnvVar)
	if err != nil {
		log.Error("failed to create database connection pool", "error", err)
		return
	}
	defer db.Close()
	repo := dal.NewPostgreSQLRepository(ctx, db, log)

	bot, err := telegram.NewBot(telegramTokenEnvVar, repo, log, telegram.Recover(log), telegram.LogErrors(log), telegram.AllowedChats(allowedChatIDs))
	if err != nil {
		log.Error("failed to create bot", "error", err)
		return
	}

	go schedule.StartWordCheckSchedule(ctx, allowedChatIDs, publishInterval, bot, kyivLocation, log)
	go schedule.StartUpdateBatchSchedule(ctx, allowedChatIDs, batchSize, guessedStreakLimit, repo, log)

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

//nolint:gochecknoinits // init main
func init() {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		panic(err)
	}
	kyivLocation = loc
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
			var chatID int64
			chatID, err = strconv.ParseInt(chatIDString, 10, 64)
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
	publishInterval, err = time.ParseDuration(publishIntervalEnvVar)
	if err != nil {
		panic("invalid PUBLISH_INTERVAL value")
	}
}

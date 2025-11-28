package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	sqlrepo "github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/schedule"
	"github.com/Roma7-7-7/english-learning-bot/internal/telegram"
)

var (
	// Version is set via -ldflags at build time
	Version = "dev" //nolint:gochecknoglobals // must be global to be replaced at build time
	// BuildTime is set via -ldflags at build time
	BuildTime = "unknown" //nolint:gochecknoglobals // must be global to be replaced at build time
)

const (
	batchSize          = 50
	guessedStreakLimit = 15

	exitCodeOK int = iota
	exitCodeConfigParse
	exitCodeDBConnect
	exitCodeBotCreate
)

func main() {
	os.Exit(run(context.Background()))
}

func run(ctx context.Context) int {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	go func() {
		<-sigs
		cancel()
	}()
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	conf, err := config.GetBot(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get config", "error", err) //nolint:sloglint // app logger is not configured yet
		return exitCodeConfigParse
	}

	log := mustLogger(conf.Dev)
	loc := conf.Schedule.MustTimeLocation()

	log.InfoContext(ctx, "starting bot",
		"version", Version,
		"build_time", BuildTime,
		"config", loggableConfig(conf),
		"current_time_in_location", time.Now().In(loc),
	)
	defer log.InfoContext(ctx, "bot is stopped")

	db, err := sql.Open("sqlite", conf.DBPath)
	if err != nil {
		log.ErrorContext(ctx, "create database connection", "error", err)
		return exitCodeDBConnect
	}
	defer db.Close()
	repo := sqlrepo.NewSQLiteRepository(ctx, db, log)

	bot, err := telegram.NewBot(conf.TelegramToken, repo, log, telegram.Recover(log), telegram.LogErrors(log), telegram.AllowedChats(conf.AllowedChatIDs))
	if err != nil {
		log.ErrorContext(ctx, "failed to create bot", "error", err)
		return exitCodeBotCreate
	}

	go schedule.StartWordCheckSchedule(ctx, schedule.WordCheckConfig{
		ChatIDs:  conf.AllowedChatIDs,
		Interval: conf.Schedule.PublishInterval,
		HourFrom: conf.Schedule.HourFrom,
		HourTo:   conf.Schedule.HourTo,
		Location: loc,
	}, bot, log)
	go schedule.StartUpdateBatchSchedule(ctx, conf.AllowedChatIDs, batchSize, guessedStreakLimit, repo, log)

	log.InfoContext(ctx, "starting bot")
	bot.Start(ctx)

	return exitCodeOK
}

func mustLogger(dev bool) *slog.Logger {
	var handler slog.Handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	if dev {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	return slog.New(handler)
}

func loggableConfig(conf *config.Bot) map[string]any {
	return map[string]any{
		"dev":              conf.Dev,
		"allowed-chat-ids": conf.AllowedChatIDs,
		"word-check-schedule": map[string]any{
			"publish-interval": fmt.Sprintf("%v", conf.Schedule.PublishInterval),
			"hour-from":        conf.Schedule.HourFrom,
			"hour-to":          conf.Schedule.HourTo,
		},
	}
}

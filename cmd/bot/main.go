package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/schedule"
	"github.com/Roma7-7-7/english-learning-bot/internal/telegram"
)

const (
	batchSize          = 50
	guessedStreakLimit = 15
)

const (
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

	conf, err := config.GetConfig()
	if err != nil {
		slog.ErrorContext(ctx, "failed to get config", "error", err) //nolint:sloglint // app logger is not configured yet
		return exitCodeConfigParse
	}

	log := mustLogger(conf.Env)

	log.InfoContext(ctx, "starting bot", "env", conf.Env, "interval", conf.PublishInterval, "current_time_in_location", time.Now().In(conf.Location))
	defer log.InfoContext(ctx, "bot is stopped")

	db, err := pgxpool.New(ctx, conf.DBURL)
	if err != nil {
		log.ErrorContext(ctx, "failed to create database connection pool", "error", err)
		return exitCodeDBConnect
	}
	defer db.Close()
	repo := dal.NewPostgreSQLRepository(ctx, db, log)

	bot, err := telegram.NewBot(conf.TelegramToken, repo, log, telegram.Recover(log), telegram.LogErrors(log), telegram.AllowedChats(conf.AllowedChatIDs))
	if err != nil {
		log.ErrorContext(ctx, "failed to create bot", "error", err)
		return exitCodeBotCreate
	}

	go schedule.StartWordCheckSchedule(ctx, conf.AllowedChatIDs, conf.PublishInterval, bot, conf.Location, log)
	go schedule.StartUpdateBatchSchedule(ctx, conf.AllowedChatIDs, batchSize, guessedStreakLimit, repo, log)

	log.InfoContext(ctx, "starting bot")
	bot.Start(ctx)

	return exitCodeOK
}

func mustLogger(env string) *slog.Logger {
	var handler slog.Handler
	if env == config.EnvProd {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	}
	return slog.New(handler)
}

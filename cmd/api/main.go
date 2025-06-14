package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Roma7-7-7/english-learning-bot/internal/api"
	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal/postgres"
	"github.com/Roma7-7-7/english-learning-bot/internal/telegram"
)

const (
	exitCodeOK int = iota
	exitCodeConfigParse
	exitCodeDBConnect
	exitCodeServerStart
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

	conf, err := config.NewAPI()
	if err != nil {
		slog.ErrorContext(ctx, "failed to get config", "error", err) //nolint:sloglint // ignore
		return exitCodeConfigParse
	}
	log := mustLogger(conf.Dev)

	db, err := pgxpool.New(ctx, conf.DB.URL)
	if err != nil {
		log.ErrorContext(ctx, "failed to create database connection pool", "error", err)
		return exitCodeDBConnect
	}
	defer db.Close()

	deps := dependencies(ctx, conf, db, log)
	router := api.NewRouter(ctx, conf, deps)
	log.InfoContext(ctx, "starting api server")

	server := &http.Server{
		ReadHeaderTimeout: conf.Server.ReadHeaderTimeout,
		Addr:              conf.Server.Addr,
		Handler:           router,
	}

	go func() {
		<-ctx.Done()
		cCtx, cCancel := context.WithTimeout(context.Background(), 15*time.Second) //nolint:mnd // ignore mnd
		defer cCancel()

		if sErr := server.Shutdown(cCtx); sErr != nil {
			log.ErrorContext(cCtx, "failed to shutdown api server", "error", sErr)
		}
	}()

	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.ErrorContext(ctx, "failed to start api server", "error", err)
		return exitCodeServerStart
	}

	log.InfoContext(ctx, "api server is stopped")

	return exitCodeOK
}

func dependencies(ctx context.Context, conf *config.API, db *pgxpool.Pool, log *slog.Logger) api.Dependencies {
	return api.Dependencies{
		Repo:           postgres.NewRepository(ctx, db, log),
		TelegramClient: telegram.NewClient(conf.Telegram.Token, log),
		Logger:         log,
	}
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

package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	sqlrepo "github.com/Roma7-7-7/english-learning-bot/internal/dal"
	_ "github.com/mattn/go-sqlite3"

	"github.com/Roma7-7-7/english-learning-bot/internal/api"
	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	"github.com/Roma7-7-7/english-learning-bot/internal/telegram"
)

var (
	// Version is set via -ldflags at build time
	Version = "dev" //nolint:gochecknoglobals // must be global to be replaced at build time
	// BuildTime is set via -ldflags at build time
	BuildTime = "unknown" //nolint:gochecknoglobals // must be global to be replaced at build time
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

	conf, err := config.NewAPI(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get config", "error", err) //nolint:sloglint // ignore
		return exitCodeConfigParse
	}
	log := mustLogger(conf.Dev)

	db, err := sql.Open("sqlite3", conf.DB.URL)
	if err != nil {
		log.ErrorContext(ctx, "failed to create database connection pool", "error", err)
		return exitCodeDBConnect
	}
	defer db.Close()

	deps := dependencies(ctx, conf, db, log)
	conf.BuildInfo.Version = Version
	conf.BuildInfo.BuildTime = BuildTime
	router := api.NewRouter(ctx, conf, deps)
	log.InfoContext(ctx, "starting api server",
		"version", Version,
		"build_time", BuildTime,
		"address", conf.Server.Addr,
	)

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

func dependencies(ctx context.Context, conf *config.API, db *sql.DB, log *slog.Logger) api.Dependencies {
	return api.Dependencies{
		Repo:           sqlrepo.NewSQLiteRepository(ctx, db, log),
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

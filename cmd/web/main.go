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

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/telegram"
	"github.com/Roma7-7-7/english-learning-bot/internal/web"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	exitCodeOK int = iota
	exitCodeConfigParse
	exitCodeDBConnect
	exitCodeDependenciesCreate
	exitCodeServerStart
)

func main() {
	os.Exit(run(config.GetEnv(), context.Background()))
}

func run(env config.Env, ctx context.Context) int {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigs := make(chan os.Signal, 1)
	go func() {
		<-sigs
		cancel()
	}()
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	log := mustLogger(env)

	conf, err := config.NewWeb(env)
	if err != nil {
		log.ErrorContext(ctx, "failed to get config", "error", err)
		return exitCodeConfigParse
	}

	db, err := pgxpool.New(ctx, conf.DB.URL)
	if err != nil {
		log.ErrorContext(ctx, "failed to create database connection pool", "error", err)
		return exitCodeDBConnect
	}
	defer db.Close()

	deps, err := dependencies(ctx, conf, db, log)
	if err != nil {
		log.ErrorContext(ctx, "failed to create dependencies", "error", err)
		return exitCodeDependenciesCreate
	}
	router := web.NewRouter(ctx, conf, deps)
	log.InfoContext(ctx, "starting web server", "config", conf)

	server := &http.Server{
		Addr:    conf.Server.Addr,
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		cCtx, cCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cCancel()

		if sErr := server.Shutdown(cCtx); sErr != nil {
			log.ErrorContext(cCtx, "failed to shutdown web server", "error", sErr)
		}
	}()

	if err = server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.ErrorContext(ctx, "failed to start web server", "error", err)
		return exitCodeServerStart
	} else {
		log.InfoContext(ctx, "web server is stopped")
	}

	return exitCodeOK
}

func dependencies(ctx context.Context, conf config.Web, db *pgxpool.Pool, log *slog.Logger) (web.Dependencies, error) {
	return web.Dependencies{
		Repo:           dal.NewPostgreSQLRepository(ctx, db, log),
		TelegramClient: telegram.NewClient(conf.Telegram.Token, log),
		Logger:         log,
	}, nil
}

func mustLogger(env config.Env) *slog.Logger {
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

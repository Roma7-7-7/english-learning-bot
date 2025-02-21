package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
)

type Dependencies struct {
	Repo           dal.Repository
	TelegramClient TelegramClient
	Logger         *slog.Logger
}

func NewRouter(ctx context.Context, conf config.Web, deps Dependencies) http.Handler {
	e := echo.New()

	e.Use(middleware.RequestID())
	e.Use(loggingMiddleware(ctx, deps.Logger))
	e.Use(middleware.Recover())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(conf.API.RateLimit))))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: conf.API.CORS.AllowOrigins,
	}))
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: conf.API.Timeout,
	}))
	e.Use(middleware.Secure())

	jwtProcessor := NewJWTProcessor(conf.API.JWT, conf.API.Cookie.AuthExpiresIn, conf.API.Cookie.AccessExpiresIn)
	cookiesProcessor := NewCookiesProcessor(conf.API.Cookie)

	auth := NewAuth(deps.Repo, jwtProcessor, cookiesProcessor, deps.TelegramClient, deps.Logger)
	e.GET("/login", auth.Login)
	e.POST("/login", auth.SubmitChatID)
	e.GET("/login/status", auth.LoginStatus)

	return e
}

func loggingMiddleware(ctx context.Context, log *slog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true, // forwards error to the global error handler, so it can decide appropriate status code
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				log.LogAttrs(ctx, slog.LevelInfo, "REQUEST",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
				)
			} else {
				log.LogAttrs(ctx, slog.LevelError, "REQUEST_ERROR",
					slog.String("uri", v.URI),
					slog.Int("status", v.Status),
					slog.String("err", v.Error.Error()),
				)
			}
			return nil
		},
	})
}

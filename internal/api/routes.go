package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type (
	Dependencies struct {
		Repo           dal.Repository
		TelegramClient TelegramClient
		Logger         *slog.Logger
	}
)

func NewRouter(ctx context.Context, conf *config.API, deps Dependencies) http.Handler {
	e := echo.New()
	e.Validator = NewCustomValidator()

	e.Use(middleware.RequestID())
	e.Use(loggingMiddleware(ctx, deps.Logger))
	e.Use(middleware.Recover())

	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(conf.HTTP.RateLimit),
				Burst:     int(conf.HTTP.RateLimit * 2), //nolint:mnd // burst is twice the rate limit
				ExpiresIn: time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, _ error) error {
			return context.JSON(http.StatusTooManyRequests, ErrorResponse{
				Message: "Too many requests",
			})
		},
		DenyHandler: func(context echo.Context, _ string, _ error) error {
			return context.JSON(http.StatusTooManyRequests, ErrorResponse{
				Message: "Too many requests",
			})
		},
	}))

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     conf.HTTP.CORS.AllowOrigins,
		AllowMethods:     []string{http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderXRequestedWith},
		AllowCredentials: true,
		MaxAge:           3600, //nolint:mnd // 1 hour
	}))

	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: conf.HTTP.ProcessTimeout,
	}))

	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000, //nolint:mnd // 1 year
		HSTSExcludeSubdomains: false,
		HSTSPreloadEnabled:    true,
		ContentSecurityPolicy: "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
	}))

	e.Use(middleware.BodyLimit("1M"))

	e.HTTPErrorHandler = HTTPErrorHandler(deps.Logger)

	jwtProcessor := NewJWTProcessor(conf.HTTP.JWT, conf.HTTP.Cookie.AuthExpiresIn, conf.HTTP.Cookie.AccessExpiresIn)
	cookiesProcessor := NewCookiesProcessor(conf.HTTP.Cookie)

	authMiddleware := AuthMiddleware(cookiesProcessor, jwtProcessor, deps.Logger)
	auth := NewAuthHandler(AuthDependencies{
		Repo:             deps.Repo,
		JWTProcessor:     jwtProcessor,
		CookiesProcessor: cookiesProcessor,
		TelegramClient:   deps.TelegramClient,
		AllowedChatIDs:   conf.Telegram.AllowedChatIDs,
		Logger:           deps.Logger,
	})

	e.POST("/auth/login", auth.Login)
	e.GET("/auth/status", auth.Status)
	e.POST("/auth/logout", auth.LogOut)

	securedGroup := e.Group("", authMiddleware)
	securedGroup.GET("/auth/info", auth.Info)

	words := NewWordsHandler(deps.Repo, deps.Logger)
	securedGroup.GET("/words", words.FindWords)
	securedGroup.POST("/words", words.CreateWord)
	securedGroup.PUT("/words", words.UpdateWord)
	securedGroup.PUT("/words/review", words.MarkToReview)
	securedGroup.DELETE("/words", words.DeleteWord)

	stats := NewStatsHandler(deps.Repo, deps.Logger)
	securedGroup.GET("/stats/total", stats.TotalStats)
	securedGroup.GET("/stats", stats.GetStats)
	securedGroup.GET("/stats/range", stats.GetStatsRange)

	return e
}

func loggingMiddleware(ctx context.Context, log *slog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
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

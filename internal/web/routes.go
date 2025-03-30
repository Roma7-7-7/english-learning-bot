package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Roma7-7-7/english-learning-bot/internal/config"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

const wordsPage = "/words"

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

	e.HTTPErrorHandler = HTTPErrorHandler(deps.Logger)

	jwtProcessor := NewJWTProcessor(conf.API.JWT, conf.API.Cookie.AuthExpiresIn, conf.API.Cookie.AccessExpiresIn)
	cookiesProcessor := NewCookiesProcessor(conf.API.Cookie)

	auth := NewAuthHandler(deps.Repo, jwtProcessor, cookiesProcessor, deps.TelegramClient, deps.Logger)

	e.GET("/login", auth.Login)
	e.POST("/login", auth.SubmitChatID)
	e.GET("/login/status", auth.LoginStatus)
	e.GET("/logout", auth.LogOut)

	words := NewWordsHandler(deps.Repo, deps.Logger)
	securedGroup := e.Group("", AuthMiddleware(cookiesProcessor, jwtProcessor, deps.Logger))
	securedGroup.GET("/", redirectHandleFunc(http.StatusFound, wordsPage))
	securedGroup.GET(wordsPage, words.ListWordsPage)
	securedGroup.DELETE("/words/:word", words.DeleteWord)

	securedGroup.GET("/error", ErrorPage)

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

func redirectHandleFunc(status int, to string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return redirect(c, status, to)
	}
}

func redirect(c echo.Context, status int, to string) error {
	c.Response().Header().Set("HX-Redirect", to)
	return c.Redirect(status, to)
}

func redirectError(c echo.Context, status int, err string) error {
	return c.Redirect(status, "/error?error="+err)
}

func redirectToLogin(c echo.Context, status int) error {
	return redirect(c, status, "/login")
}

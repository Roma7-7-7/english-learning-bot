package web

import (
	"log/slog"
	"net/http"

	"github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/labstack/echo/v4"
)

func AuthMiddleware(cookieProc *CookiesProcessor, jwtProc *JWTProcessor, log *slog.Logger) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, ok := cookieProc.GetAccessToken(c)
			if !ok {
				return redirectToLogin(c, http.StatusFound)
			}

			chatID, err := jwtProc.ParseAccessToken(token)
			if err != nil {
				log.WarnContext(c.Request().Context(), "parse access token", "error", err)
				return redirectToLogin(c, http.StatusFound)
			}

			c.Set("chatID", chatID)
			c.SetRequest(c.Request().WithContext(context.WithChatID(c.Request().Context(), chatID)))

			return next(c)
		}
	}
}

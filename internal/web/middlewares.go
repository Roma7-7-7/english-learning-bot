package web

import (
	"log/slog"
	"net/http"

	"github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/labstack/echo/v4"
)

var unauthorizedResponse = ErrorResponse{"Unauthorized"} //nolint:gochecknoglobals // this is a constant response for unauthorized access

func AuthMiddleware(cookieProc *CookiesProcessor, jwtProc *JWTProcessor, log *slog.Logger) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token, ok := cookieProc.GetAccessToken(c)
			if !ok {
				return c.JSON(http.StatusUnauthorized, unauthorizedResponse)
			}

			chatID, err := jwtProc.ParseAccessToken(token)
			if err != nil {
				log.WarnContext(c.Request().Context(), "parse access token", "error", err)
				return c.JSON(http.StatusUnauthorized, unauthorizedResponse)
			}

			c.Set("chatID", chatID)
			c.SetRequest(c.Request().WithContext(context.WithChatID(c.Request().Context(), chatID)))

			return next(c)
		}
	}
}

package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Roma7-7-7/english-learning-bot/internal/web/views"
	"github.com/labstack/echo/v4"
)

const pageGlobalErrorAlertSelector = "#pageGlobalErrorAlert"

func HTTPErrorHandler(log *slog.Logger) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		log.ErrorContext(c.Request().Context(), "failed to process request", "error", err)
		if err == nil {
			// already handled
			return
		}

		var echoError *echo.HTTPError
		if errors.As(err, &echoError) {
			if echoError.Code == http.StatusTooManyRequests {
				c.Response().WriteHeader(http.StatusTooManyRequests)
				err = views.ErrorPage("Too many requests").Render(c.Request().Context(), c.Response().Writer)
				if err != nil {
					log.ErrorContext(c.Request().Context(), "failed to render error page", "error", err)
				}
				return
			}

			err = redirect(c, http.StatusFound, "/error?error="+http.StatusText(echoError.Code))
			if err != nil {
				log.ErrorContext(c.Request().Context(), "failed to redirect echo error", "error", err)
			}
			return
		}

		err = redirect(c, http.StatusFound, "/error?error=Something went wrong")
		if err != nil {
			log.ErrorContext(c.Request().Context(), "failed to redirect", "error", err)
		}
	}
}

func ErrorPage(c echo.Context) error {
	err := c.QueryParam("error")
	if err == "" {
		err = "Something went wrong"
	}
	return views.ErrorPage(err).Render(c.Request().Context(), c.Response().Writer)
}

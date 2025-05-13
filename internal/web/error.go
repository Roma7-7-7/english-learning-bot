package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type ErrorResponse struct {
	Message string `json:"error"`
}

var InternalServerError = ErrorResponse{"Internal server error"}

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
				err = c.JSON(http.StatusTooManyRequests, echoError)
				if err != nil {
					log.ErrorContext(c.Request().Context(), "failed to respond with error", "error", err)
				}
				return
			}

			// todo test this
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

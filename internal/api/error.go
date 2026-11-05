package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

type ErrorResponse struct {
	Message string `json:"error"`
}

var (
	InternalServerError = ErrorResponse{"Internal server error"} //nolint:gochecknoglobals // this is a constant response for internal server error
	BadRequestError     = ErrorResponse{"Bad request"}           //nolint:gochecknoglobals // this is a constant response for bad request
	NotFoundError       = ErrorResponse{"Not found"}             //nolint:gochecknoglobals // this is a constant response for not found
)

func HTTPErrorHandler(log *slog.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		ctx := c.Request().Context()
		log.ErrorContext(ctx, "failed to process request", "error", err)

		if resp, uerr := echo.UnwrapResponse(c.Response()); uerr == nil && resp.Committed {
			return
		}

		code := echo.StatusCode(err)
		if code == 0 {
			code = http.StatusInternalServerError
		}

		message := InternalServerError.Message
		var echoError *echo.HTTPError
		if code != http.StatusInternalServerError && errors.As(err, &echoError) && echoError.Message != "" {
			message = echoError.Message
		} else if code != http.StatusInternalServerError {
			message = http.StatusText(code)
		}

		if err := c.JSON(code, ErrorResponse{Message: message}); err != nil {
			log.ErrorContext(ctx, "failed to write error response", "error", err)
		}
	}
}

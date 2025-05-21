package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

type ErrorResponse struct {
	Message string `json:"error"`
}

var (
	InternalServerError = ErrorResponse{"Internal server error"} //nolint:gochecknoglobals // this is a constant response for internal server error
	BadRequestError     = ErrorResponse{"Bad request"}           //nolint:gochecknoglobals // this is a constant response for bad request
)

//nolint:gocognit // no more changes are needed
func HTTPErrorHandler(log *slog.Logger) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		log.ErrorContext(c.Request().Context(), "failed to process request", "error", err)

		var echoError *echo.HTTPError
		if !errors.As(err, &echoError) {
			if err := c.JSON(http.StatusInternalServerError, InternalServerError); err != nil { //nolint:govet // ignore shadow declaration
				log.ErrorContext(c.Request().Context(), "failed to write error response", "error", err)
			}
			return
		}

		if message, ok := echoError.Message.(string); ok {
			if message == "" {
				message = "Internal server error"
			}
			if echoError.Code == http.StatusInternalServerError {
				message = InternalServerError.Message
			}
			if err := c.JSON(echoError.Code, ErrorResponse{Message: message}); err != nil { //nolint:govet // ignore shadow declaration
				log.ErrorContext(c.Request().Context(), "failed to write error response", "error", err)
			}

			return
		}

		if bytes, err := json.Marshal(echoError.Message); err != nil { //nolint:govet // ignore shadow declaration
			log.ErrorContext(c.Request().Context(), "failed to marshal error message", "error", err)
			if err := c.JSON(echoError.Code, InternalServerError); err != nil { //nolint:govet // ignore shadow declaration
				log.ErrorContext(c.Request().Context(), "failed to write error response", "error", err)
			}
		} else {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if err := c.String(echoError.Code, string(bytes)); err != nil { //nolint:govet // ignore shadow declaration
				log.ErrorContext(c.Request().Context(), "failed to write error response", "error", err)
			}
		}
	}
}

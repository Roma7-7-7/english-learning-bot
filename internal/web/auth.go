package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
)

type (
	TelegramClient interface {
		AskAuthConfirmation(ctx context.Context, chatID int64, token string) error
	}

	SubmitChatIDRequest struct {
		ChatID int64 `json:"chat_id"`
	}

	AuthHandler struct {
		repo             dal.AuthConfirmationRepository
		teleClient       TelegramClient
		jwtProcessor     *JWTProcessor
		cookiesProcessor *CookiesProcessor

		log *slog.Logger
	}
)

func NewAuthHandler(repo dal.AuthConfirmationRepository, jwtProc *JWTProcessor, cookiesProc *CookiesProcessor, teleClient TelegramClient, log *slog.Logger) *AuthHandler {
	return &AuthHandler{
		repo:             repo,
		teleClient:       teleClient,
		jwtProcessor:     jwtProc,
		cookiesProcessor: cookiesProc,

		log: log,
	}
}

func (h *AuthHandler) LogOut(c echo.Context) error {
	c.SetCookie(h.cookiesProcessor.ExpireAccessTokenCookie())
	return redirectToLogin(c, http.StatusFound)
}

func (h *AuthHandler) SubmitChatID(c echo.Context) error {
	var req SubmitChatIDRequest
	var err error
	if err = c.Bind(&req); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	chatID := req.ChatID
	key := uuid.NewString()
	if err = h.repo.InsertAuthConfirmation(c.Request().Context(), chatID, key, h.cookiesProcessor.authExpiresIn); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to insert auth confirmation", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	if err = h.teleClient.AskAuthConfirmation(c.Request().Context(), chatID, key); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to ask auth confirmation", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	token, err := h.jwtProcessor.ToAuthToken(chatID, key)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to create auth token", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}
	c.SetCookie(h.cookiesProcessor.NewAuthTokenCookie(token))

	return c.JSON(http.StatusAccepted, nil)
}

func (h *AuthHandler) Status(c echo.Context) error {
	res := echo.Map{
		"authenticated": false,
	}

	token, ok := h.cookiesProcessor.GetAuthToken(c)
	if !ok {
		h.log.DebugContext(c.Request().Context(), "auth token not found")
		return c.JSON(http.StatusUnauthorized, res)
	}
	chatID, key, err := h.jwtProcessor.ParseAuthToken(token)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to parse auth token", "error", err)
		return c.JSON(http.StatusUnauthorized, res)
	}

	res["chatID"] = chatID

	confirmed, err := h.repo.IsConfirmed(c.Request().Context(), chatID, key)
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			return c.JSON(http.StatusOK, res)
		}

		h.log.ErrorContext(c.Request().Context(), "failed to check auth confirmation", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	if !confirmed {
		return c.JSON(http.StatusOK, res)
	}

	res["authenticated"] = true

	accessToken, err := h.jwtProcessor.ToAccessToken(chatID)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to create access token", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	c.SetCookie(h.cookiesProcessor.NewAccessTokenCookie(accessToken))
	c.SetCookie(h.cookiesProcessor.ExpireAuthTokenCookie())
	return c.JSON(http.StatusOK, res)
}

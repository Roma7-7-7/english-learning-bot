package web

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/web/views"
	"github.com/labstack/echo/v4"
)

type (
	TelegramClient interface {
		AskAuthConfirmation(ctx context.Context, chatID int64, token string) error
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

func (h *AuthHandler) Login(c echo.Context) error {
	return views.LoginPage(c.QueryParam("error")).Render(c.Request().Context(), c.Response().Writer)
}

func (h *AuthHandler) LogOut(c echo.Context) error {
	c.SetCookie(h.cookiesProcessor.ExpireAccessTokenCookie())
	return redirectToLogin(c, http.StatusFound)
}

func (h *AuthHandler) SubmitChatID(c echo.Context) error {
	chatIDStr := c.FormValue("chatID")
	if chatIDStr == "" {
		return views.LoginForm(chatIDStr, "chatID is required").Render(c.Request().Context(), c.Response().Writer)
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return views.LoginForm(chatIDStr, "chatID must be a number").Render(c.Request().Context(), c.Response().Writer)
	}

	key := uuid.NewString()
	if err = h.repo.InsertAuthConfirmation(c.Request().Context(), chatID, key, h.cookiesProcessor.authExpiresIn); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to insert auth confirmation", "error", err)
		return views.LoginForm(chatIDStr, "something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}

	if err = h.teleClient.AskAuthConfirmation(c.Request().Context(), chatID, key); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to ask auth confirmation", "error", err)
		return views.LoginForm(chatIDStr, "something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}

	token, err := h.jwtProcessor.ToAuthToken(chatID, key)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to create auth token", "error", err)
		return views.LoginForm(chatIDStr, "something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}
	c.SetCookie(h.cookiesProcessor.NewAuthTokenCookie(token))

	return views.AuthAwaiting(key).Render(c.Request().Context(), c.Response().Writer)
}

func (h *AuthHandler) LoginStatus(c echo.Context) error {
	token, ok := h.cookiesProcessor.GetAuthToken(c)
	if !ok {
		h.log.DebugContext(c.Request().Context(), "auth token not found")
		return c.String(401, "Unauthorized")
	}
	chatID, key, err := h.jwtProcessor.ParseAuthToken(token)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to parse auth token", "error", err)
		return c.String(401, "Unauthorized")
	}

	confirmed, err := h.repo.IsConfirmed(c.Request().Context(), chatID, key)
	if err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			return c.String(http.StatusOK, "DECLINED")
		}

		h.log.ErrorContext(c.Request().Context(), "failed to check auth confirmation", "error", err)
		return c.String(500, "Internal Server Error")
	}

	if !confirmed {
		return c.String(http.StatusOK, "NOT CONFIRMED")
	}

	accessToken, err := h.jwtProcessor.ToAccessToken(chatID)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to create access token", "error", err)
		return c.String(500, "Internal Server Error")
	}

	c.SetCookie(h.cookiesProcessor.NewAccessTokenCookie(accessToken))
	c.SetCookie(h.cookiesProcessor.ExpireAuthTokenCookie())
	return c.String(http.StatusOK, "CONFIRMED")
}

package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	appctx "github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
)

type (
	TelegramClient interface {
		AskAuthConfirmation(ctx context.Context, chatID int64, token string) error
	}

	AuthDependencies struct {
		Repo             dal.AuthConfirmationRepository
		JWTProcessor     *JWTProcessor
		CookiesProcessor *CookiesProcessor
		TelegramClient   TelegramClient
		AllowedChatIDs   []int64
		Logger           *slog.Logger
	}

	AuthHandler struct {
		repo             dal.AuthConfirmationRepository
		teleClient       TelegramClient
		jwtProcessor     *JWTProcessor
		cookiesProcessor *CookiesProcessor
		allowedChatIDs   map[int64]bool

		log *slog.Logger
	}

	submitChatIDRequest struct {
		ChatID int64 `json:"chat_id"`
	}

	statusResponse struct {
		Authenticated bool  `json:"authenticated"`
		ChatID        int64 `json:"chat_id"`
	}
)

func NewAuthHandler(deps AuthDependencies) *AuthHandler {
	allowedChatIDs := make(map[int64]bool, len(deps.AllowedChatIDs))
	for _, chatID := range deps.AllowedChatIDs {
		allowedChatIDs[chatID] = true
	}
	return &AuthHandler{
		repo:             deps.Repo,
		teleClient:       deps.TelegramClient,
		jwtProcessor:     deps.JWTProcessor,
		cookiesProcessor: deps.CookiesProcessor,
		allowedChatIDs:   allowedChatIDs,

		log: deps.Logger,
	}
}

func (h *AuthHandler) Info(c echo.Context) error {
	chatID := appctx.MustChatIDFromContext(c.Request().Context())
	return c.JSON(http.StatusOK, echo.Map{
		"chat_id": chatID,
	})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req submitChatIDRequest
	var err error
	if err = c.Bind(&req); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	chatID := req.ChatID
	if _, ok := h.allowedChatIDs[chatID]; !ok {
		h.log.DebugContext(c.Request().Context(), "chat ID not allowed", "chat_id", chatID)
		return c.JSON(http.StatusForbidden, ErrorResponse{
			Message: "chat ID not allowed",
		})
	}
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
	var res statusResponse

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

	res.ChatID = chatID

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

	res.Authenticated = true

	accessToken, err := h.jwtProcessor.ToAccessToken(chatID)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to create access token", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	c.SetCookie(h.cookiesProcessor.NewAccessTokenCookie(accessToken))
	c.SetCookie(h.cookiesProcessor.ExpireAuthTokenCookie())
	return c.JSON(http.StatusOK, res)
}

func (h *AuthHandler) LogOut(c echo.Context) error {
	c.SetCookie(h.cookiesProcessor.ExpireAccessTokenCookie())
	return c.JSON(http.StatusOK, nil)
}

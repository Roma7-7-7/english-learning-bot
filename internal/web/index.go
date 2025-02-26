package web

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/web/views"
	"github.com/labstack/echo/v4"
)

type IndexHandler struct {
	repo dal.WordTranslationsRepository
	log  *slog.Logger
}

func NewIndexHandler(repo dal.WordTranslationsRepository, log *slog.Logger) *IndexHandler {
	return &IndexHandler{
		repo: repo,
		log:  log,
	}
}

func (h *IndexHandler) IndexPage(c echo.Context) error {
	chatID, ok := context.ChatIDFromContext(c.Request().Context())
	if !ok {
		return redirectToLogin(c, http.StatusFound)
	}
	stats, err := h.repo.GetStats(c.Request().Context(), chatID)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to get stats", "error", err)
		return views.IndexPage(stats.GreaterThanOrEqual15, stats.Total, nil, "Something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}

	offset, err := strconv.Atoi(defString(c.QueryParam("offset"), "0"))
	if err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to parse offset", "error", err)
		return views.IndexPage(stats.GreaterThanOrEqual15, stats.Total, nil, "Something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}
	limit, err := strconv.Atoi(defString(c.QueryParam("limit"), "25"))
	if err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to parse limit", "error", err)
		return views.IndexPage(stats.GreaterThanOrEqual15, stats.Total, nil, "Something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}
	filter := dal.WordTranslationsFilter{
		Word:     "",
		ToReview: false,
		Offset:   offset,
		Limit:    limit,
	}
	words, err := h.repo.FindWordTranslations(c.Request().Context(), chatID, filter)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to find word translations", "error", err)
		return views.IndexPage(stats.GreaterThanOrEqual15, stats.Total, nil, "Something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}

	viewWords := make([]views.WordTranslation, len(words))
	for i, word := range words {
		viewWords[i] = views.WordTranslation{
			Word:        word.Word,
			Translation: word.Translation,
			ToReview:    word.ToReview,
		}
	}

	return views.IndexPage(stats.GreaterThanOrEqual15, stats.Total, viewWords, "").Render(c.Request().Context(), c.Response().Writer)
}

func defString(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

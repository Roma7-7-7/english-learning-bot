package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
)

type (
	DailyStatsHandler struct {
		repo dal.DailyStatsRepository
		log  *slog.Logger
	}

	DailyStatsQueryParams struct {
		From time.Time `query:"from" validate:"required"`
		To   time.Time `query:"to" validate:"required"`
	}
)

func NewDailyStatsHandler(repo dal.DailyStatsRepository, log *slog.Logger) *DailyStatsHandler {
	return &DailyStatsHandler{
		repo: repo,
		log:  log,
	}
}

func (h *DailyStatsHandler) GetDailyStats(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	stats, err := h.repo.GetDailyStats(c.Request().Context(), chatID, time.Now())
	if err != nil && err != dal.ErrNotFound {
		h.log.ErrorContext(c.Request().Context(), "failed to get daily stats", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	if err == dal.ErrNotFound {
		return c.JSON(http.StatusOK, echo.Map{
			"words_guessed":       0,
			"words_missed":        0,
			"total_words_learned": 0,
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"words_guessed":       stats.WordsGuessed,
		"words_missed":        stats.WordsMissed,
		"total_words_learned": stats.TotalWordsLearned,
	})
}

func (h *DailyStatsHandler) GetStatsRange(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var qp DailyStatsQueryParams
	if err := c.Bind(&qp); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	if err := c.Validate(&qp); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to validate request", "error", err)
		return err
	}

	stats, err := h.repo.GetStatsRange(c.Request().Context(), chatID, qp.From, qp.To)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to get stats range", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	result := make([]echo.Map, len(stats))
	for i, stat := range stats {
		result[i] = echo.Map{
			"date":                stat.Date,
			"words_guessed":       stat.WordsGuessed,
			"words_missed":        stat.WordsMissed,
			"total_words_learned": stat.TotalWordsLearned,
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"items": result,
	})
}

package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/Roma7-7-7/english-learning-bot/internal/web/views"
	"github.com/labstack/echo/v4"
)

type WordsHandler struct {
	repo dal.WordTranslationsRepository
	log  *slog.Logger
}

func NewWordsHandler(repo dal.WordTranslationsRepository, log *slog.Logger) *WordsHandler {
	return &WordsHandler{
		repo: repo,
		log:  log,
	}
}

func (h *WordsHandler) ListWordsPage(c echo.Context) error {
	chatID, ok := context.ChatIDFromContext(c.Request().Context())
	if !ok {
		return redirectToLogin(c, http.StatusFound)
	}

	var stats views.Stats
	wStats, err := h.repo.GetStats(c.Request().Context(), chatID)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to get stats", "error", err)
		return views.ErrorPage("Something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}
	stats.Learned = wStats.GreaterThanOrEqual15
	stats.Total = wStats.Total

	qp, err := parseWordsPageQueryParams(c)
	if err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to parse query params", "error", err)
		return views.ListWordsPage(stats, qp, "Something went wrong").Render(c.Request().Context(), c.Response().Writer)
	}

	return views.ListWordsPage(stats, qp, "").Render(c.Request().Context(), c.Response().Writer)
}

func (h *WordsHandler) ListWords(c echo.Context) error {
	chatID, ok := context.ChatIDFromContext(c.Request().Context())
	if !ok {
		return redirectToLogin(c, http.StatusFound)
	}

	qp, err := parseWordsPageQueryParams(c)
	if err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to parse query params", "error", err)
		return retargetErrorDiv(c, http.StatusBadRequest, "Something went wrong", pageGlobalErrorAlertSelector)
	}

	qp, words, err := h.listWords(c, chatID, qp)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to list words", "error", err)
		return retargetErrorDiv(c, http.StatusInternalServerError, "Something went wrong", pageGlobalErrorAlertSelector)
	}

	return views.ListWords(qp, words, "").Render(c.Request().Context(), c.Response().Writer)
}

func (h *WordsHandler) DeleteWord(c echo.Context) error {
	chatID, ok := context.ChatIDFromContext(c.Request().Context())
	if !ok {
		return redirectToLogin(c, http.StatusFound)
	}

	word := c.Param("word")
	if word == "" {
		return c.Redirect(http.StatusFound, "/?error=word not found")
	}

	if err := h.repo.DeleteWordTranslation(c.Request().Context(), chatID, word); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to delete word translation", "error", err)
		return retargetErrorDiv(c, http.StatusInternalServerError, "Something went wrong", pageGlobalErrorAlertSelector)
	}

	return c.String(http.StatusOK, "OK")
}

func (h *WordsHandler) listWords(c echo.Context, chatID int64, qp views.WordsQueryParams) (views.WordsQueryParams, []views.WordTranslation, error) {
	filter := dal.WordTranslationsFilter{
		Word:     qp.Search,
		ToReview: false,
		Offset:   qp.Paginate.Offset(),
		Limit:    qp.Paginate.Limit,
	}
	words, totalWords, err := h.repo.FindWordTranslations(c.Request().Context(), chatID, filter)
	if err != nil {
		return views.WordsQueryParams{}, nil, fmt.Errorf("find word translations: %w", err)
	}

	viewWords := make([]views.WordTranslation, len(words))
	for i, word := range words {
		viewWords[i] = views.WordTranslation{
			Word:        word.Word,
			Translation: word.Translation,
			ToReview:    word.ToReview,
		}
	}
	qp.Paginate.TotalPages = qp.Paginate.CalcTotalPages(totalWords)

	return qp, viewWords, nil
}

func parseWordsPageQueryParams(c echo.Context) (views.WordsQueryParams, error) {
	limit, err := strconv.Atoi(defString(c.QueryParam("limit"), "15"))
	if err != nil {
		return views.WordsQueryParams{}, fmt.Errorf("parse limit: %w", err)
	}
	page, err := strconv.Atoi(defString(c.QueryParam("page"), "1"))
	if err != nil {
		return views.WordsQueryParams{}, fmt.Errorf("parse page: %w", err)
	}

	return views.WordsQueryParams{
		Search: defString(c.QueryParam("search"), ""),
		Paginate: views.Pagination{
			Limit: limit,
			Page:  page,
		},
	}, nil
}

func defString(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

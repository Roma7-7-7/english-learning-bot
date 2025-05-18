package web

import (
	"log/slog"
	"net/http"

	"github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
)

type (
	WordTranslation struct {
		Word        string `json:"word"`
		Translation string `json:"translation"`
		Description string `json:"description"`
		ToReview    bool   `json:"to_review"`
	}

	WordsQueryParams struct {
		Search   string `query:"search"`
		ToReview bool   `query:"to_review"`
		Offset   int    `query:"offset"`
		Limit    int    `query:"limit"`
	}

	WordsHandler struct {
		repo dal.WordTranslationsRepository
		log  *slog.Logger
	}
)

func NewWordsHandler(repo dal.WordTranslationsRepository, log *slog.Logger) *WordsHandler {
	return &WordsHandler{
		repo: repo,
		log:  log,
	}
}

func (h *WordsHandler) Stats(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	stats, err := h.repo.GetStats(c.Request().Context(), chatID)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to get stats", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"learned": stats.GreaterThanOrEqual15,
		"total":   stats.Total,
	})
}

func (h *WordsHandler) FindWords(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var qp WordsQueryParams
	if err := c.Bind(&qp); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	filter := dal.WordTranslationsFilter{
		Word:     qp.Search,
		ToReview: qp.ToReview,
		Offset:   qp.Offset,
		Limit:    qp.Limit,
	}
	words, totalWords, err := h.repo.FindWordTranslations(c.Request().Context(), chatID, filter)
	if err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to find word translations", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	viewWords := make([]WordTranslation, len(words))
	for i, word := range words {
		viewWords[i] = WordTranslation{
			Word:        word.Word,
			Translation: word.Translation,
			Description: word.Description,
			ToReview:    word.ToReview,
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"items": viewWords,
		"total": totalWords,
	})
}

//func (h *WordsHandler) WordPage(c echo.Context) error {
//	chatID, ok := context.ChatIDFromContext(c.Request().Context())
//	if !ok {
//		return redirectToLogin(c, http.StatusFound)
//	}
//
//	qp, err := parseWordsPageQueryParams(c)
//	if err != nil {
//		h.log.DebugContext(c.Request().Context(), "failed to parse query params", "error", err)
//		return redirectError(c, http.StatusFound, "Something went wrong")
//	}
//
//	var stats views.Stats
//	wStats, err := h.repo.GetStats(c.Request().Context(), chatID)
//	if err != nil {
//		h.log.ErrorContext(c.Request().Context(), "failed to get stats", "error", err)
//		return views.WordPage(stats, views.WordTranslation{}, qp.PageToHref(qp.Pagination.Page), "Something went wrong").Render(c.Request().Context(), c.Response().Writer)
//	}
//	stats.Learned = wStats.GreaterThanOrEqual15
//	stats.Total = wStats.Total
//
//	var wt views.WordTranslation
//	word := c.QueryParam("word")
//	if word != "" {
//		w, err := h.repo.FindWordTranslation(c.Request().Context(), chatID, word)
//		if err != nil {
//			h.log.ErrorContext(c.Request().Context(), "failed to get word translation", "error", err)
//			return redirectError(c, http.StatusFound, "Something went wrong")
//		}
//		wt = views.WordTranslation{
//			Word:        w.Word,
//			Translation: w.Translation,
//			Description: w.Description,
//			ToReview:    w.ToReview,
//		}
//	}
//
//	return views.WordPage(stats, wt, qp.PageToHref(qp.Pagination.Page), "").Render(c.Request().Context(), c.Response().Writer)
//}

//func (h *WordsHandler) DeleteWord(c echo.Context) error {
//	chatID, ok := context.ChatIDFromContext(c.Request().Context())
//	if !ok {
//		return redirectToLogin(c, http.StatusFound)
//	}
//
//	word := c.Param("word")
//	if word == "" {
//		return c.Redirect(http.StatusFound, "/?error=word not found")
//	}
//
//	if err := h.repo.DeleteWordTranslation(c.Request().Context(), chatID, word); err != nil {
//		h.log.ErrorContext(c.Request().Context(), "failed to delete word translation", "error", err)
//		return redirectError(c, http.StatusFound, "Something went wrong")
//	}
//
//	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "word deleted"})
//}
//
//func defString(val, def string) string {
//	if val == "" {
//		return def
//	}
//	return val
//}

package api

import (
	"log/slog"
	"net/http"

	"github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
	"github.com/labstack/echo/v4"
)

type (
	WordTranslation struct {
		Word          string `json:"word" validate:"required,min=1"`
		NewWord       string `json:"new_word,omitempty" validate:"omitempty,min=1"`
		Translation   string `json:"translation" validate:"required,min=1"`
		Description   string `json:"description"`
		ToReview      bool   `json:"to_review"`
		GuessedStreak int    `json:"guessed_streak,omitempty"`
	}

	Guessed string

	WordsQueryParams struct {
		Search   string  `query:"search"`
		Guessed  Guessed `query:"guessed" validate:"omitempty,oneof=all learned batched to_learn"`
		ToReview bool    `query:"to_review"`
		Offset   uint64  `query:"offset" validate:"min=0"`
		Limit    uint64  `query:"limit" validate:"required,min=1,max=100"`
	}

	WordsHandler struct {
		repo dal.WordTranslationsRepository
		log  *slog.Logger
	}
)

const (
	GuessedAll     Guessed = "all"
	GuessedLearned Guessed = "learned"
	GuessedBatched Guessed = "batched"
	GuessedToLearn Guessed = "to_learn"
)

func NewWordsHandler(repo dal.WordTranslationsRepository, log *slog.Logger) *WordsHandler {
	return &WordsHandler{
		repo: repo,
		log:  log,
	}
}

func (h *WordsHandler) FindWords(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var qp WordsQueryParams
	if err := c.Bind(&qp); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	if err := c.Validate(&qp); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to validate request", "error", err)
		return err
	}

	filter := dal.WordTranslationsFilter{
		Word:     qp.Search,
		Guessed:  toDALGuessed(qp.Guessed),
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
			Word:          word.Word,
			Translation:   word.Translation,
			Description:   word.Description,
			ToReview:      word.ToReview,
			GuessedStreak: word.GuessedStreak,
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"items": viewWords,
		"total": totalWords,
	})
}

func (h *WordsHandler) CreateWord(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var wt WordTranslation
	if err := c.Bind(&wt); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	if err := c.Validate(&wt); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to validate request", "error", err)
		return err
	}

	if err := h.repo.AddWordTranslation(c.Request().Context(), chatID, wt.Word, wt.Translation, wt.Description); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to create word translation", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "word created"})
}

func (h *WordsHandler) UpdateWord(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var wt WordTranslation
	if err := c.Bind(&wt); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	if err := c.Validate(&wt); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to validate request", "error", err)
		return err
	}

	if err := h.repo.UpdateWordTranslation(c.Request().Context(), chatID, wt.Word, wt.NewWord, wt.Translation, wt.Description); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to update word translation", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "word updated"})
}

type DeleteWordRequest struct {
	Word string `json:"word" validate:"required,min=1"`
}

func (h *WordsHandler) DeleteWord(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var req DeleteWordRequest
	if err := c.Bind(&req); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	if err := c.Validate(&req); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to validate request", "error", err)
		return err
	}

	if err := h.repo.DeleteWordTranslation(c.Request().Context(), chatID, req.Word); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to delete word translation", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "word deleted"})
}

type MarkToReviewRequest struct {
	Word     string `json:"word" validate:"required,min=1"`
	ToReview bool   `json:"to_review"`
}

func (h *WordsHandler) MarkToReview(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var r MarkToReviewRequest
	if err := c.Bind(&r); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	if err := c.Validate(&r); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to validate request", "error", err)
		return err
	}

	if err := h.repo.MarkToReview(c.Request().Context(), chatID, r.Word, r.ToReview); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to mark word to review", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "word marked"})
}

func toDALGuessed(g Guessed) dal.Guessed {
	switch g {
	case GuessedAll:
		return dal.GuessedAll
	case GuessedLearned:
		return dal.GuessedLearned
	case GuessedBatched:
		return dal.GuessedBatched
	case GuessedToLearn:
		return dal.GuessedToLearn
	default:
		return dal.GuessedAll
	}
}

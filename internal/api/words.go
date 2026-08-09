package api

import (
	"errors"
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
		// OnConflict is only meaningful on create. Left empty, adding a word that already exists is
		// refused with 409 and the existing entry, so the caller can ask the user what to do; set,
		// it applies that answer.
		OnConflict string `json:"on_conflict,omitempty" validate:"omitempty,oneof=reset_and_batch reset_only update_only"`
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

// createAttempts bounds the create/read-the-conflict loop in CreateWord. One retry is enough for a
// word that is deleted concurrently; more than that is a client fighting itself.
const createAttempts = 2

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

	ctx := c.Request().Context()

	// The caller has already been told about the existing entry and said what to do about it.
	// ResolveWordConflict writes the word either way, so the decision still stands if the word was
	// deleted between the 409 and this request.
	if wt.OnConflict != "" {
		err := h.repo.ResolveWordConflict(ctx, chatID,
			wt.Word, wt.Translation, wt.Description, dal.ConflictResolution(wt.OnConflict))
		if err != nil {
			h.log.ErrorContext(ctx, "failed to resolve word conflict", "error", err)
			return c.JSON(http.StatusInternalServerError, InternalServerError)
		}
		return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "word resolved"})
	}

	// Without a decision the create must not overwrite anything, so the repository refuses the
	// insert instead of the handler checking first — a lookup here would let two concurrent creates
	// both see nothing and have the second discard the first. Reading the existing entry afterwards
	// can lose the race the other way round, hence the retry: the word was deleted in between and
	// there is nothing to report a conflict about anymore.
	for range createAttempts {
		err := h.repo.CreateWordTranslation(ctx, chatID, wt.Word, wt.Translation, wt.Description)
		switch {
		case err == nil:
			return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "word created"})
		case !errors.Is(err, dal.ErrAlreadyExists):
			h.log.ErrorContext(ctx, "failed to create word translation", "error", err)
			return c.JSON(http.StatusInternalServerError, InternalServerError)
		}

		existing, err := h.repo.FindWordTranslation(ctx, chatID, wt.Word)
		if err != nil {
			if errors.Is(err, dal.ErrNotFound) {
				continue
			}
			h.log.ErrorContext(ctx, "failed to find word translation", "error", err)
			return c.JSON(http.StatusInternalServerError, InternalServerError)
		}

		// Report what is already stored so the caller can show the translation and the streak that
		// an overwrite would discard.
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "word already exists",
			"existing": WordTranslation{
				Word:          existing.Word,
				Translation:   existing.Translation,
				Description:   existing.Description,
				ToReview:      existing.ToReview,
				GuessedStreak: existing.GuessedStreak,
			},
		})
	}

	h.log.ErrorContext(ctx, "gave up creating word translation", "word", wt.Word)
	return c.JSON(http.StatusInternalServerError, InternalServerError)
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

type ResetStreakRequest struct {
	Word string `json:"word" validate:"required,min=1"`
	// AddToBatch also puts the word back into the active learning batch, so it starts coming up in
	// word checks again. Without it the reset word is neither batched nor eligible for review — only
	// learned words are reviewed — so it waits for a refill to pick it out of the whole vocabulary.
	AddToBatch bool `json:"add_to_batch"`
}

func (h *WordsHandler) ResetStreak(c echo.Context) error {
	chatID := context.MustChatIDFromContext(c.Request().Context())

	var req ResetStreakRequest
	if err := c.Bind(&req); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to bind request", "error", err)
		return c.JSON(http.StatusBadRequest, BadRequestError)
	}

	if err := c.Validate(&req); err != nil {
		h.log.DebugContext(c.Request().Context(), "failed to validate request", "error", err)
		return err
	}

	if _, err := h.repo.FindWordTranslation(c.Request().Context(), chatID, req.Word); err != nil {
		if errors.Is(err, dal.ErrNotFound) {
			return c.JSON(http.StatusNotFound, NotFoundError)
		}
		h.log.ErrorContext(c.Request().Context(), "failed to find word translation", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	if err := h.repo.ResetStreak(c.Request().Context(), chatID, req.Word, req.AddToBatch); err != nil {
		h.log.ErrorContext(c.Request().Context(), "failed to reset streak", "error", err)
		return c.JSON(http.StatusInternalServerError, InternalServerError)
	}

	return c.JSON(http.StatusOK, echo.Map{"status": "ok", "message": "streak reset"})
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

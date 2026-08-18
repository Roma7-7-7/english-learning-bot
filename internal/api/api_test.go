package api_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/Roma7-7-7/english-learning-bot/internal/api"
	appctx "github.com/Roma7-7-7/english-learning-bot/internal/context"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

const testChatID int64 = 42

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newRequest wires up just enough of echo to exercise one handler: the custom validator and the
// chat ID that AuthMiddleware would normally have injected. Booting NewRouter would need a full
// config (JWT, CORS, cookies) for no extra coverage. Every handler covered here takes a POST.
func newRequest(t *testing.T, path, body string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()

	e := echo.New()
	e.Validator = api.NewCustomValidator()

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req = req.WithContext(appctx.WithChatID(req.Context(), testChatID))

	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// stubWordsRepo implements dal.WordTranslationsRepository, recording the calls each test cares about
// and returning canned results. Only the methods a test exercises need to be configured.
type stubWordsRepo struct {
	findWord     func(word string) (*dal.WordTranslation, error)
	resetCalls   []resetCall
	resetErr     error
	createCalls  []createCall
	createErr    error
	resolveCalls []resolveCall
	resolveErr   error
}

type resetCall struct {
	word       string
	addToBatch bool
}

type createCall struct {
	word, translation, description string
}

type resolveCall struct {
	word, translation, description string
	resolution                     dal.ConflictResolution
}

func (s *stubWordsRepo) FindWordTranslation(_ context.Context, _ int64, word string) (*dal.WordTranslation, error) {
	if s.findWord == nil {
		return nil, dal.ErrNotFound
	}
	return s.findWord(word)
}

// CreateWordTranslation mirrors the real insert-only semantics: a word findWord already serves is
// refused rather than overwritten.
func (s *stubWordsRepo) CreateWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error {
	if _, err := s.FindWordTranslation(ctx, chatID, word); err == nil {
		return dal.ErrAlreadyExists
	}
	s.createCalls = append(s.createCalls, createCall{word, translation, description})
	return s.createErr
}

func (s *stubWordsRepo) ResetStreak(_ context.Context, _ int64, word string, addToBatch bool) error {
	s.resetCalls = append(s.resetCalls, resetCall{word, addToBatch})
	return s.resetErr
}

func (s *stubWordsRepo) ResolveWordConflict(
	_ context.Context, _ int64, word, translation, description string, resolution dal.ConflictResolution,
) error {
	s.resolveCalls = append(s.resolveCalls, resolveCall{word, translation, description, resolution})
	return s.resolveErr
}

func (s *stubWordsRepo) FindWordTranslations(
	_ context.Context, _ int64, _ dal.WordTranslationsFilter,
) ([]dal.WordTranslation, int, error) {
	return nil, 0, nil
}

func (s *stubWordsRepo) FindRandomWordTranslation(
	_ context.Context, _ int64, _ dal.FindRandomWordFilter,
) (*dal.WordTranslation, error) {
	return nil, dal.ErrNotFound
}

func (s *stubWordsRepo) UpdateWordTranslation(_ context.Context, _ int64, _, _, _, _ string) error {
	return nil
}
func (s *stubWordsRepo) DeleteWordTranslation(_ context.Context, _ int64, _ string) error { return nil }
func (s *stubWordsRepo) RegisterGuess(_ context.Context, _ int64, _ string) error         { return nil }
func (s *stubWordsRepo) RegisterMiss(_ context.Context, _ int64, _ string) error          { return nil }
func (s *stubWordsRepo) MarkToReview(_ context.Context, _ int64, _ string, _ bool) error  { return nil }
func (s *stubWordsRepo) MarkWordReviewed(_ context.Context, _ int64, _ string) error      { return nil }

func (s *stubWordsRepo) RefillLearningBatch(_ context.Context, _ int64) (int, int, error) {
	return 0, 0, nil
}

// existingWord builds a findWord stub that serves wt for its own word and reports every other word
// as missing.
func existingWord(wt dal.WordTranslation) func(string) (*dal.WordTranslation, error) {
	wt.ChatID = testChatID
	return func(got string) (*dal.WordTranslation, error) {
		if got != wt.Word {
			return nil, dal.ErrNotFound
		}
		return &wt, nil
	}
}

func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}

var _ dal.WordTranslationsRepository = (*stubWordsRepo)(nil)

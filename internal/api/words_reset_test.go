package api_test

import (
	"net/http"
	"testing"

	"github.com/Roma7-7-7/english-learning-bot/internal/api"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func TestResetStreak(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantAddToBatch bool
	}{
		{name: "reset only", body: `{"word":"apple","add_to_batch":false}`, wantAddToBatch: false},
		{name: "reset and add to batch", body: `{"word":"apple","add_to_batch":true}`, wantAddToBatch: true},
		{name: "add_to_batch defaults to false", body: `{"word":"apple"}`, wantAddToBatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubWordsRepo{findWord: existingWord(dal.WordTranslation{
				Word: "apple", Translation: "яблуко", GuessedStreak: 18,
			})}
			h := api.NewWordsHandler(repo, testLogger())

			c, rec := newRequest(t, "/words/reset", tt.body)
			if err := h.ResetStreak(c); err != nil {
				t.Fatalf("ResetStreak: %v", err)
			}

			assertStatus(t, rec, http.StatusOK)
			if len(repo.resetCalls) != 1 {
				t.Fatalf("reset calls = %d, want 1", len(repo.resetCalls))
			}
			got := repo.resetCalls[0]
			if got.word != "apple" {
				t.Errorf("reset word = %q, want apple", got.word)
			}
			if got.addToBatch != tt.wantAddToBatch {
				t.Errorf("addToBatch = %v, want %v", got.addToBatch, tt.wantAddToBatch)
			}
		})
	}
}

func TestResetStreakUnknownWord(t *testing.T) {
	repo := &stubWordsRepo{} // findWord nil => always ErrNotFound
	h := api.NewWordsHandler(repo, testLogger())

	c, rec := newRequest(t, "/words/reset", `{"word":"missing"}`)
	if err := h.ResetStreak(c); err != nil {
		t.Fatalf("ResetStreak: %v", err)
	}

	assertStatus(t, rec, http.StatusNotFound)
	if len(repo.resetCalls) != 0 {
		t.Errorf("reset was attempted on a word that does not exist: %+v", repo.resetCalls)
	}
}

func TestResetStreakRejectsEmptyWord(t *testing.T) {
	repo := &stubWordsRepo{}
	h := api.NewWordsHandler(repo, testLogger())

	c, _ := newRequest(t, "/words/reset", `{"word":""}`)
	if err := h.ResetStreak(c); err == nil {
		t.Fatal("ResetStreak accepted an empty word, want a validation error")
	}
	if len(repo.resetCalls) != 0 {
		t.Errorf("reset was attempted despite validation failing: %+v", repo.resetCalls)
	}
}

package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Roma7-7-7/english-learning-bot/internal/api"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func TestCreateWordNew(t *testing.T) {
	repo := &stubWordsRepo{} // nothing exists
	h := api.NewWordsHandler(repo, testLogger())

	c, rec := newRequest(t, "/words",
		`{"word":"apple","translation":"яблуко","description":"a fruit"}`)
	if err := h.CreateWord(c); err != nil {
		t.Fatalf("CreateWord: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
	if len(repo.addCalls) != 1 {
		t.Fatalf("add calls = %d, want 1", len(repo.addCalls))
	}
	if got := repo.addCalls[0]; got.word != "apple" || got.translation != "яблуко" || got.description != "a fruit" {
		t.Errorf("added %+v, want apple/яблуко/a fruit", got)
	}
	if len(repo.resolveCalls) != 0 {
		t.Errorf("conflict resolution ran for a brand new word: %+v", repo.resolveCalls)
	}
}

// Adding an existing word used to silently overwrite it and return 200, quietly discarding whatever
// the user might have wanted to know about the existing entry.
func TestCreateWordDuplicateReportsConflict(t *testing.T) {
	repo := &stubWordsRepo{findWord: existingWord(dal.WordTranslation{
		Word: "apple", Translation: "яблуко", Description: "a fruit", GuessedStreak: 18,
	})}
	h := api.NewWordsHandler(repo, testLogger())

	c, rec := newRequest(t, "/words",
		`{"word":"apple","translation":"різновид яблука"}`)
	if err := h.CreateWord(c); err != nil {
		t.Fatalf("CreateWord: %v", err)
	}

	assertStatus(t, rec, http.StatusConflict)
	if len(repo.addCalls) != 0 {
		t.Errorf("existing word was overwritten: %+v", repo.addCalls)
	}
	if len(repo.resolveCalls) != 0 {
		t.Errorf("resolution ran without the caller asking for one: %+v", repo.resolveCalls)
	}

	var body struct {
		Error    string              `json:"error"`
		Existing api.WordTranslation `json:"existing"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error == "" {
		t.Error(`response has no "error" message`)
	}
	// The dialog needs all of this to explain the choice.
	if body.Existing.Translation != "яблуко" {
		t.Errorf("existing translation = %q, want яблуко", body.Existing.Translation)
	}
	if body.Existing.Description != "a fruit" {
		t.Errorf("existing description = %q, want 'a fruit'", body.Existing.Description)
	}
	if body.Existing.GuessedStreak != 18 {
		t.Errorf("existing streak = %d, want 18", body.Existing.GuessedStreak)
	}
}

func TestCreateWordAppliesConflictResolution(t *testing.T) {
	tests := []struct {
		name       string
		onConflict string
		want       dal.ConflictResolution
	}{
		{name: "reset and batch", onConflict: "reset_and_batch", want: dal.ResolveResetAndBatch},
		{name: "reset only", onConflict: "reset_only", want: dal.ResolveResetOnly},
		{name: "update only", onConflict: "update_only", want: dal.ResolveUpdateOnly},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stubWordsRepo{findWord: existingWord(dal.WordTranslation{
				Word: "apple", Translation: "яблуко", Description: "a fruit", GuessedStreak: 18,
			})}
			h := api.NewWordsHandler(repo, testLogger())

			c, rec := newRequest(t, "/words",
				`{"word":"apple","translation":"нове","description":"new desc","on_conflict":"`+tt.onConflict+`"}`)
			if err := h.CreateWord(c); err != nil {
				t.Fatalf("CreateWord: %v", err)
			}

			assertStatus(t, rec, http.StatusOK)
			if len(repo.resolveCalls) != 1 {
				t.Fatalf("resolve calls = %d, want 1", len(repo.resolveCalls))
			}
			got := repo.resolveCalls[0]
			if got.resolution != tt.want {
				t.Errorf("resolution = %q, want %q", got.resolution, tt.want)
			}
			// Whichever text the user picked is what gets written.
			if got.word != "apple" || got.translation != "нове" || got.description != "new desc" {
				t.Errorf("resolved with %+v, want apple/нове/new desc", got)
			}
		})
	}
}

// Sending back the existing translation is how the UI says "keep what is already there".
func TestCreateWordConflictCanKeepExistingTranslation(t *testing.T) {
	repo := &stubWordsRepo{findWord: existingWord(dal.WordTranslation{
		Word: "apple", Translation: "яблуко", Description: "a fruit", GuessedStreak: 18,
	})}
	h := api.NewWordsHandler(repo, testLogger())

	c, rec := newRequest(t, "/words",
		`{"word":"apple","translation":"яблуко","description":"a fruit","on_conflict":"reset_only"}`)
	if err := h.CreateWord(c); err != nil {
		t.Fatalf("CreateWord: %v", err)
	}

	assertStatus(t, rec, http.StatusOK)
	if len(repo.resolveCalls) != 1 {
		t.Fatalf("resolve calls = %d, want 1", len(repo.resolveCalls))
	}
	if got := repo.resolveCalls[0]; got.translation != "яблуко" {
		t.Errorf("translation = %q, want the existing яблуко", got.translation)
	}
}

func TestCreateWordRejectsUnknownConflictResolution(t *testing.T) {
	repo := &stubWordsRepo{findWord: existingWord(dal.WordTranslation{
		Word: "apple", Translation: "яблуко", GuessedStreak: 18,
	})}
	h := api.NewWordsHandler(repo, testLogger())

	c, _ := newRequest(t, "/words",
		`{"word":"apple","translation":"x","on_conflict":"delete_everything"}`)
	if err := h.CreateWord(c); err == nil {
		t.Fatal("CreateWord accepted an unknown on_conflict value, want a validation error")
	}
	if len(repo.resolveCalls) != 0 {
		t.Errorf("resolution ran despite validation failing: %+v", repo.resolveCalls)
	}
}

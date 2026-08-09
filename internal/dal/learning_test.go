package dal_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func TestRegisterGuess(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("word", 14)

	if err := r.RegisterGuess(ctx, dal.TestChatID, "word"); err != nil {
		t.Fatalf("RegisterGuess: %v", err)
	}

	if got := r.StreakOf("word"); got != 15 {
		t.Errorf("streak = %d, want 15", got)
	}
	guessed, missed, totalLearned := r.TodayStats()
	if guessed != 1 || missed != 0 {
		t.Errorf("guessed/missed = %d/%d, want 1/0", guessed, missed)
	}
	if totalLearned != 1 {
		t.Errorf("total_words_learned = %d, want 1", totalLearned)
	}
}

// A missed word must land back in the learning batch, otherwise a word forgotten during review
// silently disappears again.
func TestRegisterMissDemotesLearnedWordIntoBatch(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("word", 20)

	if err := r.RegisterMiss(ctx, dal.TestChatID, "word"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := r.StreakOf("word"); got != 0 {
		t.Errorf("streak = %d, want 0", got)
	}
	if !r.IsBatched("word") {
		t.Error("missed word is not in the learning batch")
	}
	guessed, missed, totalLearned := r.TodayStats()
	if guessed != 0 || missed != 1 {
		t.Errorf("guessed/missed = %d/%d, want 0/1", guessed, missed)
	}
	if totalLearned != 0 {
		t.Errorf("total_words_learned = %d, want 0", totalLearned)
	}
}

func TestRegisterMissOnBatchedWordIsIdempotent(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("word", 3)
	r.SeedBatch("word")

	if err := r.RegisterMiss(ctx, dal.TestChatID, "word"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := len(r.BatchWords()); got != 1 {
		t.Errorf("batch size = %d, want 1 (no duplicate row)", got)
	}
}

// The batch is allowed to grow past its configured size when words are demoted back into it.
func TestRegisterMissMayPushBatchOverItsLimit(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	for i := range 3 {
		r.AddWord(fmt.Sprintf("learning-%d", i), 1)
	}
	if _, _, err := r.RefillLearningBatch(ctx, dal.TestChatID, 3, 15); err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	r.AddWord("forgotten", 20)

	if err := r.RegisterMiss(ctx, dal.TestChatID, "forgotten"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := len(r.BatchWords()); got != 4 {
		t.Errorf("batch size = %d, want 4 (over the limit of 3)", got)
	}

	// The next refill must accept the overflow rather than trimming it.
	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID, 3, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	if evicted != 0 || added != 0 {
		t.Errorf("evicted/added = %d/%d, want 0/0", evicted, added)
	}
	if got := len(r.BatchWords()); got != 4 {
		t.Errorf("batch size = %d after refill, want 4", got)
	}
}

func TestResetStreak(t *testing.T) {
	tests := []struct {
		name       string
		addToBatch bool
		wantBatch  bool
	}{
		{name: "reset only", addToBatch: false, wantBatch: false},
		{name: "reset and add to batch", addToBatch: true, wantBatch: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := dal.NewTestRepo(t)
			r.AddWord("word", 20)

			if err := r.ResetStreak(ctx, dal.TestChatID, "word", tt.addToBatch); err != nil {
				t.Fatalf("ResetStreak: %v", err)
			}

			if got := r.StreakOf("word"); got != 0 {
				t.Errorf("streak = %d, want 0", got)
			}
			if got := r.IsBatched("word"); got != tt.wantBatch {
				t.Errorf("batched = %v, want %v", got, tt.wantBatch)
			}
		})
	}
}

// A deliberate reset is a correction, not a wrong answer, so it must not pollute the daily
// guessed/missed counters the way RegisterMiss does.
func TestResetStreakDoesNotCountAsMiss(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("word", 20)
	r.AddWord("other", 20)
	// Create today's statistics row via a real answer.
	if err := r.RegisterGuess(ctx, dal.TestChatID, "other"); err != nil {
		t.Fatalf("RegisterGuess: %v", err)
	}

	if err := r.ResetStreak(ctx, dal.TestChatID, "word", true); err != nil {
		t.Fatalf("ResetStreak: %v", err)
	}

	guessed, missed, totalLearned := r.TodayStats()
	if guessed != 1 || missed != 0 {
		t.Errorf("guessed/missed = %d/%d, want 1/0", guessed, missed)
	}
	if totalLearned != 1 {
		t.Errorf("total_words_learned = %d, want 1 (only 'other' is still learned)", totalLearned)
	}
}

func TestResolveWordConflict(t *testing.T) {
	tests := []struct {
		name       string
		resolution dal.ConflictResolution
		wantStreak int
		wantBatch  bool
	}{
		{name: "reset and batch", resolution: dal.ResolveResetAndBatch, wantStreak: 0, wantBatch: true},
		{name: "reset only", resolution: dal.ResolveResetOnly, wantStreak: 0, wantBatch: false},
		{name: "update only keeps progress", resolution: dal.ResolveUpdateOnly, wantStreak: 18, wantBatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			r := dal.NewTestRepo(t)
			r.AddWord("apple", 18)

			err := r.ResolveWordConflict(ctx, dal.TestChatID, "apple", "new-translation", "new-description", tt.resolution)
			if err != nil {
				t.Fatalf("ResolveWordConflict: %v", err)
			}

			// The chosen text always wins, whatever happens to the streak.
			got, err := r.FindWordTranslation(ctx, dal.TestChatID, "apple")
			if err != nil {
				t.Fatalf("FindWordTranslation: %v", err)
			}
			if got.Translation != "new-translation" {
				t.Errorf("translation = %q, want new-translation", got.Translation)
			}
			if got.Description != "new-description" {
				t.Errorf("description = %q, want new-description", got.Description)
			}
			if got.GuessedStreak != tt.wantStreak {
				t.Errorf("streak = %d, want %d", got.GuessedStreak, tt.wantStreak)
			}
			if batched := r.IsBatched("apple"); batched != tt.wantBatch {
				t.Errorf("batched = %v, want %v", batched, tt.wantBatch)
			}
		})
	}
}

func TestResolveWordConflictRejectsUnknownResolution(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("apple", 18)

	if err := r.ResolveWordConflict(ctx, dal.TestChatID, "apple", "x", "", dal.ConflictResolution("nonsense")); err == nil {
		t.Fatal("ResolveWordConflict accepted an unknown resolution")
	}
	// Nothing may have been written.
	got, err := r.FindWordTranslation(ctx, dal.TestChatID, "apple")
	if err != nil {
		t.Fatalf("FindWordTranslation: %v", err)
	}
	if got.Translation != "apple-translation" || got.GuessedStreak != 18 {
		t.Errorf("word was modified by a rejected resolution: %+v", got)
	}
}

func TestRefillLearningBatchEvictsAndFills(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	r.AddWord("learned", 15)
	r.AddWord("beyond", 20)
	for i := range 5 {
		r.AddWord(fmt.Sprintf("learning-%d", i), i)
	}
	// Both finished words start out batched and must be evicted.
	r.SeedBatch("learned", "beyond")

	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID, 3, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if evicted != 2 {
		t.Errorf("evicted = %d, want 2", evicted)
	}
	if added != 3 {
		t.Errorf("added = %d, want 3", added)
	}

	got := r.BatchWords()
	if len(got) != 3 {
		t.Fatalf("batch = %v (%d words), want 3", got, len(got))
	}
	for _, w := range got {
		if w == "learned" || w == "beyond" {
			t.Errorf("batch contains finished word %q", w)
		}
		if r.StreakOf(w) >= 15 {
			t.Errorf("batch contains %q with streak >= 15", w)
		}
	}
}

func TestRefillLearningBatchStopsWhenOverFull(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	for i := range 5 {
		word := fmt.Sprintf("learning-%d", i)
		r.AddWord(word, 1)
		r.SeedBatch(word)
	}
	r.AddWord("spare", 1)

	// Batch already holds 5 against a limit of 3: nothing may be added, nothing evicted.
	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID, 3, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if evicted != 0 {
		t.Errorf("evicted = %d, want 0", evicted)
	}
	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if got := len(r.BatchWords()); got != 5 {
		t.Errorf("batch size = %d, want 5 (unchanged)", got)
	}
	if r.IsBatched("spare") {
		t.Error("spare word was added to an over-full batch")
	}
}

func TestRefillLearningBatchSkipsAlreadyBatched(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	for i := range 4 {
		r.AddWord(fmt.Sprintf("learning-%d", i), 1)
	}
	r.SeedBatch("learning-0")

	_, added, err := r.RefillLearningBatch(ctx, dal.TestChatID, 4, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if added != 3 {
		t.Errorf("added = %d, want 3 (the one already batched must not be re-added)", added)
	}
	if got := len(r.BatchWords()); got != 4 {
		t.Errorf("batch size = %d, want 4", got)
	}
}

func TestRefillLearningBatchWithNothingToLearn(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("learned", 20)

	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID, 10, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	if evicted != 0 || added != 0 {
		t.Errorf("evicted/added = %d/%d, want 0/0", evicted, added)
	}
}

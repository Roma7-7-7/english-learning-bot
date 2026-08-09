package dal

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

// TestInTxRollsBackEveryStatement guards the reason inTx exists: statements handed to it must run
// inside the transaction it opens, so that a later failure undoes the earlier writes.
func TestInTxRollsBackEveryStatement(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addWord(t, r, "word", 7)

	sentinel := errors.New("boom")
	err := r.inTx(ctx, func(e execer) error {
		if err := resetGuessedStreak(ctx, e, testChatID, "word"); err != nil {
			return err
		}
		if err := addToLearningBatch(ctx, e, testChatID, "word"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("inTx error = %v, want %v", err, sentinel)
	}

	if got := streakOf(t, r, "word"); got != 7 {
		t.Errorf("streak = %d after rollback, want 7", got)
	}
	if isBatched(t, r, "word") {
		t.Error("word is in the learning batch after rollback, want it absent")
	}
}

func TestInTxCommits(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addWord(t, r, "word", 7)

	err := r.inTx(ctx, func(e execer) error {
		return resetGuessedStreak(ctx, e, testChatID, "word")
	})
	if err != nil {
		t.Fatalf("inTx: %v", err)
	}

	if got := streakOf(t, r, "word"); got != 0 {
		t.Errorf("streak = %d after commit, want 0", got)
	}
}

func TestRegisterGuess(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addWord(t, r, "word", 14)

	if err := r.RegisterGuess(ctx, testChatID, "word"); err != nil {
		t.Fatalf("RegisterGuess: %v", err)
	}

	if got := streakOf(t, r, "word"); got != 15 {
		t.Errorf("streak = %d, want 15", got)
	}
	guessed, missed, totalLearned := todayStats(t, r)
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
	r := newTestRepo(t)
	addWord(t, r, "word", 20)

	if err := r.RegisterMiss(ctx, testChatID, "word"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := streakOf(t, r, "word"); got != 0 {
		t.Errorf("streak = %d, want 0", got)
	}
	if !isBatched(t, r, "word") {
		t.Error("missed word is not in the learning batch")
	}
	guessed, missed, totalLearned := todayStats(t, r)
	if guessed != 0 || missed != 1 {
		t.Errorf("guessed/missed = %d/%d, want 0/1", guessed, missed)
	}
	if totalLearned != 0 {
		t.Errorf("total_words_learned = %d, want 0", totalLearned)
	}
}

func TestRegisterMissOnBatchedWordIsIdempotent(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addWord(t, r, "word", 3)
	if err := r.inTx(ctx, func(e execer) error { return addToLearningBatch(ctx, e, testChatID, "word") }); err != nil {
		t.Fatalf("seed batch: %v", err)
	}

	if err := r.RegisterMiss(ctx, testChatID, "word"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := len(batchWords(t, r)); got != 1 {
		t.Errorf("batch size = %d, want 1 (no duplicate row)", got)
	}
}

// The batch is allowed to grow past its configured size when words are demoted back into it.
func TestRegisterMissMayPushBatchOverItsLimit(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	for i := range 3 {
		addWord(t, r, fmt.Sprintf("learning-%d", i), 1)
	}
	if _, _, err := r.RefillLearningBatch(ctx, testChatID, 3, 15); err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	addWord(t, r, "forgotten", 20)

	if err := r.RegisterMiss(ctx, testChatID, "forgotten"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := len(batchWords(t, r)); got != 4 {
		t.Errorf("batch size = %d, want 4 (over the limit of 3)", got)
	}

	// The next refill must accept the overflow rather than trimming it.
	evicted, added, err := r.RefillLearningBatch(ctx, testChatID, 3, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	if evicted != 0 || added != 0 {
		t.Errorf("evicted/added = %d/%d, want 0/0", evicted, added)
	}
	if got := len(batchWords(t, r)); got != 4 {
		t.Errorf("batch size = %d after refill, want 4", got)
	}
}

func TestRefillLearningBatchEvictsAndFills(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	addWord(t, r, "learned", 15)
	addWord(t, r, "beyond", 20)
	for i := range 5 {
		addWord(t, r, fmt.Sprintf("learning-%d", i), i)
	}
	// Both finished words start out batched and must be evicted.
	for _, w := range []string{"learned", "beyond"} {
		if err := r.inTx(ctx, func(e execer) error { return addToLearningBatch(ctx, e, testChatID, w) }); err != nil {
			t.Fatalf("seed batch: %v", err)
		}
	}

	evicted, added, err := r.RefillLearningBatch(ctx, testChatID, 3, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if evicted != 2 {
		t.Errorf("evicted = %d, want 2", evicted)
	}
	if added != 3 {
		t.Errorf("added = %d, want 3", added)
	}

	got := batchWords(t, r)
	if len(got) != 3 {
		t.Fatalf("batch = %v (%d words), want 3", got, len(got))
	}
	for _, w := range got {
		if w == "learned" || w == "beyond" {
			t.Errorf("batch contains finished word %q", w)
		}
		if streakOf(t, r, w) >= 15 {
			t.Errorf("batch contains %q with streak >= 15", w)
		}
	}
}

func TestRefillLearningBatchStopsWhenOverFull(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	for i := range 5 {
		word := fmt.Sprintf("learning-%d", i)
		addWord(t, r, word, 1)
		if err := r.inTx(ctx, func(e execer) error { return addToLearningBatch(ctx, e, testChatID, word) }); err != nil {
			t.Fatalf("seed batch: %v", err)
		}
	}
	addWord(t, r, "spare", 1)

	// Batch already holds 5 against a limit of 3: nothing may be added, nothing evicted.
	evicted, added, err := r.RefillLearningBatch(ctx, testChatID, 3, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if evicted != 0 {
		t.Errorf("evicted = %d, want 0", evicted)
	}
	if added != 0 {
		t.Errorf("added = %d, want 0", added)
	}
	if got := len(batchWords(t, r)); got != 5 {
		t.Errorf("batch size = %d, want 5 (unchanged)", got)
	}
	if isBatched(t, r, "spare") {
		t.Error("spare word was added to an over-full batch")
	}
}

func TestRefillLearningBatchSkipsAlreadyBatched(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)

	for i := range 4 {
		addWord(t, r, fmt.Sprintf("learning-%d", i), 1)
	}
	if err := r.inTx(ctx, func(e execer) error { return addToLearningBatch(ctx, e, testChatID, "learning-0") }); err != nil {
		t.Fatalf("seed batch: %v", err)
	}

	_, added, err := r.RefillLearningBatch(ctx, testChatID, 4, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if added != 3 {
		t.Errorf("added = %d, want 3 (the one already batched must not be re-added)", added)
	}
	if got := len(batchWords(t, r)); got != 4 {
		t.Errorf("batch size = %d, want 4", got)
	}
}

func TestRefillLearningBatchWithNothingToLearn(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	addWord(t, r, "learned", 20)

	evicted, added, err := r.RefillLearningBatch(ctx, testChatID, 10, 15)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	if evicted != 0 || added != 0 {
		t.Errorf("evicted/added = %d/%d, want 0/0", evicted, added)
	}
}

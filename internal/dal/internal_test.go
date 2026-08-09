package dal

import (
	"context"
	"errors"
	"testing"
)

// TestInTxRollsBackEveryStatement guards the reason inTx exists: statements handed to it must run
// inside the transaction it opens, so that a later failure undoes the earlier writes.
func TestInTxRollsBackEveryStatement(t *testing.T) {
	ctx := context.Background()
	r := NewTestRepo(t)
	r.AddWord("word", 7)

	sentinel := errors.New("boom")
	err := r.inTx(ctx, func(e execer) error {
		if err := resetGuessedStreak(ctx, e, TestChatID, "word"); err != nil {
			return err
		}
		if err := addToLearningBatch(ctx, e, TestChatID, "word"); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("inTx error = %v, want %v", err, sentinel)
	}

	if got := r.StreakOf("word"); got != 7 {
		t.Errorf("streak = %d after rollback, want 7", got)
	}
	if r.IsBatched("word") {
		t.Error("word is in the learning batch after rollback, want it absent")
	}
}

func TestInTxCommits(t *testing.T) {
	ctx := context.Background()
	r := NewTestRepo(t)
	r.AddWord("word", 7)

	err := r.inTx(ctx, func(e execer) error {
		return resetGuessedStreak(ctx, e, TestChatID, "word")
	})
	if err != nil {
		t.Fatalf("inTx: %v", err)
	}

	if got := r.StreakOf("word"); got != 0 {
		t.Errorf("streak = %d after commit, want 0", got)
	}
}

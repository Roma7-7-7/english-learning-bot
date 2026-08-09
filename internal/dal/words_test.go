package dal_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func reviewFilter() dal.FindRandomWordFilter {
	return dal.FindRandomWordFilter{
		StreakLimitDirection: dal.LimitDirectionGreaterThanOrEqual,
		StreakLimit:          dal.TestStreakLimit,
		Order:                dal.OrderLeastRecentlyReviewed,
	}
}

func TestFindRandomWordTranslationReviewPicksOnlyLearnedWords(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	r.AddWord("still-learning", 14)
	r.AddWord("learned", 15)

	for range 5 {
		wt, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
		if err != nil {
			t.Fatalf("FindRandomWordTranslation: %v", err)
		}
		if wt.Word != "learned" {
			t.Fatalf("picked %q, want the only learned word", wt.Word)
		}
	}
}

func TestFindRandomWordTranslationReviewSkipsBatchedWords(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	// A learned word that was demoted back into the batch is being actively learned again and must
	// not also be offered as a review.
	r.AddWord("demoted", 20)
	r.SeedBatch("demoted")

	_, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
	if !errors.Is(err, dal.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestFindRandomWordTranslationReviewWithoutLearnedWords(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("still-learning", 3)

	_, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
	if !errors.Is(err, dal.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

// The whole point of ordering by last_reviewed_at: every learned word must come up once before any
// of them comes up twice.
func TestFindRandomWordTranslationReviewRotatesThroughAllWords(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	const total = 6
	for i := range total {
		r.AddWord(fmt.Sprintf("learned-%d", i), 20)
	}

	seen := make(map[string]int, total)
	for range total {
		wt, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
		if err != nil {
			t.Fatalf("FindRandomWordTranslation: %v", err)
		}
		seen[wt.Word]++
		// Stamping the review is what advances the rotation.
		if err = r.MarkWordReviewed(ctx, dal.TestChatID, wt.Word); err != nil {
			t.Fatalf("MarkWordReviewed: %v", err)
		}
	}

	if len(seen) != total {
		t.Errorf("covered %d distinct words in %d picks, want %d: %v", len(seen), total, total, seen)
	}
	for word, count := range seen {
		if count != 1 {
			t.Errorf("%q was reviewed %d times before the rotation completed", word, count)
		}
	}

	// Second lap: every word now carries a timestamp, so coverage depends on those timestamps
	// being distinct rather than on NULLs sorting first.
	clear(seen)
	for range total {
		wt, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
		if err != nil {
			t.Fatalf("FindRandomWordTranslation: %v", err)
		}
		seen[wt.Word]++
		if err = r.MarkWordReviewed(ctx, dal.TestChatID, wt.Word); err != nil {
			t.Fatalf("MarkWordReviewed: %v", err)
		}
	}
	if len(seen) != total {
		t.Errorf("second lap covered %d distinct words, want %d: %v", len(seen), total, seen)
	}
}

// An unanswered review must not pin the rotation to the same word.
func TestFindRandomWordTranslationReviewAdvancesOnIgnoredMessage(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("first", 20)
	r.AddWord("second", 20)

	first, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
	if err != nil {
		t.Fatalf("FindRandomWordTranslation: %v", err)
	}
	// Sent but never answered - only MarkWordReviewed runs.
	if err = r.MarkWordReviewed(ctx, dal.TestChatID, first.Word); err != nil {
		t.Fatalf("MarkWordReviewed: %v", err)
	}

	second, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
	if err != nil {
		t.Fatalf("FindRandomWordTranslation: %v", err)
	}
	if second.Word == first.Word {
		t.Errorf("picked %q twice in a row; the ignored review stalled the rotation", first.Word)
	}
}

// Editing a word must not reorder the review queue - that is why last_reviewed_at is separate from
// the trigger-maintained updated_at.
func TestMarkWordReviewedIsIndependentOfEdits(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("reviewed", 20)
	r.AddWord("untouched", 20)

	if err := r.MarkWordReviewed(ctx, dal.TestChatID, "reviewed"); err != nil {
		t.Fatalf("MarkWordReviewed: %v", err)
	}
	// Edit the word that has never been reviewed; it must still be picked first.
	if err := r.UpdateWordTranslation(ctx, dal.TestChatID, "untouched", "untouched", "new-translation", ""); err != nil {
		t.Fatalf("UpdateWordTranslation: %v", err)
	}

	got, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
	if err != nil {
		t.Fatalf("FindRandomWordTranslation: %v", err)
	}
	if got.Word != "untouched" {
		t.Errorf("picked %q, want the never-reviewed word", got.Word)
	}
}

func TestMarkWordReviewedIsStrictlyMonotonic(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	for i := range 4 {
		r.AddWord(fmt.Sprintf("learned-%d", i), 20)
	}

	seen := map[int]bool{}
	for i := range 4 {
		word := fmt.Sprintf("learned-%d", i)
		if err := r.MarkWordReviewed(ctx, dal.TestChatID, word); err != nil {
			t.Fatalf("MarkWordReviewed: %v", err)
		}

		seq := r.ReviewSeq(word)
		if seq != i+1 {
			t.Errorf("cursor for %q = %d, want %d", word, seq, i+1)
		}
		if seen[seq] {
			t.Errorf("cursor %d handed out twice", seq)
		}
		seen[seq] = true
	}
}

// A word forgotten during review goes back into the batch; once it is learned again it must rejoin
// the review rotation rather than being stuck at the front of it.
func TestReviewRotationAfterDemotionAndRelearning(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("forgotten", 20)
	r.AddWord("other", 20)

	first, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter())
	if err != nil {
		t.Fatalf("FindRandomWordTranslation: %v", err)
	}
	if err = r.MarkWordReviewed(ctx, dal.TestChatID, first.Word); err != nil {
		t.Fatalf("MarkWordReviewed: %v", err)
	}

	// Answered wrong: streak resets and the word is demoted into the batch.
	if err = r.RegisterMiss(ctx, dal.TestChatID, first.Word); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}
	if got, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, reviewFilter()); err != nil {
		t.Fatalf("FindRandomWordTranslation: %v", err)
	} else if got.Word == first.Word {
		t.Errorf("demoted word %q is still offered for review", first.Word)
	}
}

func TestFindRandomWordTranslationBatchedIgnoresOrder(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("batched", 2)
	r.SeedBatch("batched")

	got, err := r.FindRandomWordTranslation(ctx, dal.TestChatID, dal.FindRandomWordFilter{Batched: true})
	if err != nil {
		t.Fatalf("FindRandomWordTranslation: %v", err)
	}
	if got.Word != "batched" {
		t.Errorf("picked %q, want batched", got.Word)
	}
}

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

// The batch is now a hard cap: a miss that would otherwise overflow it queues the word instead.
func TestRegisterMissQueuesWhenBatchIsFull(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(3)

	for i := range 3 {
		word := fmt.Sprintf("learning-%d", i)
		r.AddWord(word, 1)
		r.SeedBatch(word)
	}
	r.AddWord("forgotten", 20)

	if err := r.RegisterMiss(ctx, dal.TestChatID, "forgotten"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := len(r.BatchWords()); got != 3 {
		t.Errorf("batch size = %d, want 3 (unchanged - the cap held)", got)
	}
	if r.IsBatched("forgotten") {
		t.Error("forgotten word was admitted into a full batch")
	}
	if !r.IsQueued("forgotten") {
		t.Error("forgotten word was not queued")
	}

	// The next refill must drain it once room appears.
	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	if evicted != 0 || added != 0 {
		t.Errorf("evicted/added = %d/%d, want 0/0 (still no room)", evicted, added)
	}
	if !r.IsQueued("forgotten") {
		t.Error("forgotten word should still be queued while the batch stays full")
	}
}

// A word that misses twice while already queued must not be duplicated or reordered.
func TestRegisterMissOnQueuedWordIsIdempotent(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(1)

	r.AddWord("resident", 1)
	r.SeedBatch("resident")
	r.AddWord("a", 20)
	r.AddWord("b", 20)
	r.SeedQueue("a", "b")

	if err := r.RegisterMiss(ctx, dal.TestChatID, "a"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}
	if err := r.RegisterMiss(ctx, dal.TestChatID, "a"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}

	if got := r.QueueWords(); len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("queue = %v, want [a b] unchanged (no duplicate, no reordering)", got)
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
	r.SetBatchSize(3)

	r.AddWord("learned", 15)
	r.AddWord("beyond", 20)
	for i := range 5 {
		r.AddWord(fmt.Sprintf("learning-%d", i), i)
	}
	// Both finished words start out batched and must be evicted.
	r.SeedBatch("learned", "beyond")

	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
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
	r.SetBatchSize(3)

	for i := range 5 {
		word := fmt.Sprintf("learning-%d", i)
		r.AddWord(word, 1)
		r.SeedBatch(word)
	}
	r.AddWord("spare", 1)

	// Batch already holds 5 against a limit of 3: nothing may be added, nothing evicted.
	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
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
	r.SetBatchSize(4)

	for i := range 4 {
		r.AddWord(fmt.Sprintf("learning-%d", i), 1)
	}
	r.SeedBatch("learning-0")

	_, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
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
	r.SetBatchSize(10)
	r.AddWord("learned", 20)

	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}
	if evicted != 0 || added != 0 {
		t.Errorf("evicted/added = %d/%d, want 0/0", evicted, added)
	}
}

// Resetting a streak can be the first thing that happens on a given day, before any answer has
// created today's statistics row. The learned count still has to be recorded, or the dashboard keeps
// reporting the word as learned.
func TestResetStreakRecordsLearnedCountOnAQuietDay(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.AddWord("word", 20)
	r.AddWord("other", 20)

	if err := r.ResetStreak(ctx, dal.TestChatID, "word", true); err != nil {
		t.Fatalf("ResetStreak: %v", err)
	}

	guessed, missed, totalLearned := r.TodayStats()
	if guessed != 0 || missed != 0 {
		t.Errorf("guessed/missed = %d/%d, want 0/0", guessed, missed)
	}
	if totalLearned != 1 {
		t.Errorf("total_words_learned = %d, want 1 (only 'other' is still learned)", totalLearned)
	}
}

func TestResetStreakQueuesWhenBatchIsFull(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(1)
	r.AddWord("resident", 1)
	r.SeedBatch("resident")
	r.AddWord("word", 20)

	if err := r.ResetStreak(ctx, dal.TestChatID, "word", true); err != nil {
		t.Fatalf("ResetStreak: %v", err)
	}

	if r.IsBatched("word") {
		t.Error("word was admitted into a full batch")
	}
	if !r.IsQueued("word") {
		t.Error("word was not queued")
	}
}

func TestResolveWordConflictResetAndBatchQueuesWhenBatchIsFull(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(1)
	r.AddWord("resident", 1)
	r.SeedBatch("resident")
	r.AddWord("apple", 18)

	err := r.ResolveWordConflict(ctx, dal.TestChatID, "apple", "new-translation", "", dal.ResolveResetAndBatch)
	if err != nil {
		t.Fatalf("ResolveWordConflict: %v", err)
	}

	if r.IsBatched("apple") {
		t.Error("apple was admitted into a full batch")
	}
	if !r.IsQueued("apple") {
		t.Error("apple was not queued")
	}
}

// A brand-new word wants practicing exactly like a missed or reset one: it must not have to wait for
// the next hourly refill when there is room right now.
func TestCreateWordTranslationEntersBatchImmediatelyWhenRoom(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)

	if err := r.CreateWordTranslation(ctx, dal.TestChatID, "word", "translation", ""); err != nil {
		t.Fatalf("CreateWordTranslation: %v", err)
	}

	if !r.IsBatched("word") {
		t.Error("new word was not admitted into the batch immediately")
	}
	if r.IsQueued("word") {
		t.Error("new word was queued despite room being available")
	}
}

func TestCreateWordTranslationQueuesWhenBatchIsFull(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(1)
	r.AddWord("resident", 1)
	r.SeedBatch("resident")

	if err := r.CreateWordTranslation(ctx, dal.TestChatID, "word", "translation", ""); err != nil {
		t.Fatalf("CreateWordTranslation: %v", err)
	}

	if r.IsBatched("word") {
		t.Error("new word was admitted into a full batch")
	}
	if !r.IsQueued("word") {
		t.Error("new word was not queued")
	}
}

// The queue must be drained in FIFO order, ahead of the random fallback: a never-queued but equally
// eligible word must not jump the line in front of words that explicitly asked to get back in first.
func TestRefillLearningBatchDrainsQueueBeforeRandomFill(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(2)

	r.AddWord("never-queued", 1)
	r.AddWord("a", 1)
	r.AddWord("b", 1)
	r.AddWord("c", 1)
	r.SeedQueue("a", "b", "c")

	_, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if added != 2 {
		t.Fatalf("added = %d, want 2", added)
	}
	got := r.BatchWords()
	if len(got) != 2 {
		t.Fatalf("batch = %v (%d words), want 2", got, len(got))
	}
	for _, w := range got {
		if w == "never-queued" {
			t.Error("never-queued word jumped ahead of the FIFO queue")
		}
		if w == "c" {
			t.Error("queue was not drained oldest-first")
		}
	}
	if r.QueueWords()[0] != "c" {
		t.Errorf("remaining queue = %v, want [c] to still be waiting", r.QueueWords())
	}
}

// Once the queue is exhausted, remaining room must still be filled from the wider pool of eligible
// words that never explicitly asked to be re-admitted.
func TestRefillLearningBatchFallsBackToRandomFillWhenQueueExhausted(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(3)

	r.AddWord("never-queued", 1)
	r.AddWord("queued", 1)
	r.SeedQueue("queued")

	_, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if added != 2 {
		t.Fatalf("added = %d, want 2 (one drained, one from the fallback)", added)
	}
	if !r.IsBatched("queued") || !r.IsBatched("never-queued") {
		t.Errorf("batch = %v, want both words admitted", r.BatchWords())
	}
	if len(r.QueueWords()) != 0 {
		t.Errorf("queue = %v, want empty", r.QueueWords())
	}
}

// A word graduating out of the batch and a word waiting in the queue must both be resolved by one
// refill call: eviction frees the room the drain then uses.
func TestRefillLearningBatchDrainOpensRoomAfterEviction(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(1)

	r.AddWord("graduated", 20)
	r.SeedBatch("graduated")
	r.AddWord("waiting", 1)
	r.SeedQueue("waiting")

	evicted, added, err := r.RefillLearningBatch(ctx, dal.TestChatID)
	if err != nil {
		t.Fatalf("RefillLearningBatch: %v", err)
	}

	if evicted != 1 {
		t.Errorf("evicted = %d, want 1", evicted)
	}
	if added != 1 {
		t.Errorf("added = %d, want 1", added)
	}
	if r.IsBatched("graduated") {
		t.Error("graduated word is still batched")
	}
	if !r.IsBatched("waiting") {
		t.Error("waiting word was not admitted")
	}
	if len(r.QueueWords()) != 0 {
		t.Errorf("queue = %v, want empty", r.QueueWords())
	}
}

// The whole design rests on a word never being tracked in both the batch and the queue at once.
func TestBatchAndQueueAreMutuallyExclusive(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetBatchSize(2)

	r.AddWord("resident-1", 1)
	r.AddWord("resident-2", 1)
	r.SeedBatch("resident-1", "resident-2")

	// All of these arrive while the batch is already full, so they must all end up queued, not
	// batched - and calling CreateWordTranslation, RegisterMiss and ResolveWordConflict again on the
	// same words afterward must not create any overlap either.
	if err := r.CreateWordTranslation(ctx, dal.TestChatID, "new", "translation", ""); err != nil {
		t.Fatalf("CreateWordTranslation: %v", err)
	}
	r.AddWord("missed", 20)
	if err := r.RegisterMiss(ctx, dal.TestChatID, "missed"); err != nil {
		t.Fatalf("RegisterMiss: %v", err)
	}
	r.AddWord("conflicted", 20)
	err := r.ResolveWordConflict(ctx, dal.TestChatID, "conflicted", "t", "", dal.ResolveResetAndBatch)
	if err != nil {
		t.Fatalf("ResolveWordConflict: %v", err)
	}
	if err := r.RegisterMiss(ctx, dal.TestChatID, "missed"); err != nil {
		t.Fatalf("RegisterMiss (again): %v", err)
	}

	batched := r.BatchWords()
	queued := r.QueueWords()
	seen := make(map[string]bool, len(batched))
	for _, w := range batched {
		seen[w] = true
	}
	for _, w := range queued {
		if seen[w] {
			t.Errorf("%q is in both the batch and the queue", w)
		}
	}
	for _, w := range []string{"new", "missed", "conflicted"} {
		if seen[w] {
			t.Errorf("%q was admitted into a full batch", w)
		}
	}
}

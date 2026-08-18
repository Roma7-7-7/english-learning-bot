package dal_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func TestGetTotalStatsBuckets(t *testing.T) {
	tests := []struct {
		name        string
		streakLimit int
		// streaks to seed, one word each
		streaks                       []int
		batched                       int // how many of the seeded words, from the front, go into the batch
		queued                        int // how many of the seeded words, after those, go into the queue
		wantLearned, wantNearly       int
		wantEarly, wantTotal          int
		wantStreakLimit, wantNearlyFr int
		wantBatched                   int
		wantQueued                    int
	}{
		{
			name:        "default limit keeps the historical 15+/10-14/1-9 split",
			streakLimit: 15,
			streaks:     []int{0, 1, 9, 10, 14, 15, 20},
			batched:     2,
			wantLearned: 2, wantNearly: 2, wantEarly: 2, wantTotal: 7,
			wantStreakLimit: 15, wantNearlyFr: 10, wantBatched: 2,
		},
		{
			name:        "retuned limit moves every bucket together",
			streakLimit: 8,
			// limit 8 => learned >= 8, nearly 3-7, early 1-2
			streaks:     []int{0, 1, 2, 3, 7, 8, 12},
			wantLearned: 2, wantNearly: 2, wantEarly: 2, wantTotal: 7,
			wantStreakLimit: 8, wantNearlyFr: 3, wantBatched: 0,
		},
		{
			name:        "limit below the nearly width collapses the early bucket",
			streakLimit: 3,
			streaks:     []int{0, 1, 2, 3},
			batched:     1,
			queued:      1,
			wantLearned: 1, wantNearly: 2, wantEarly: 0, wantTotal: 4,
			wantStreakLimit: 3, wantNearlyFr: 1, wantBatched: 1, wantQueued: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := dal.NewTestRepo(t)
			r.SetStreakLimit(tt.streakLimit)
			var toBatch, toQueue []string
			for i, streak := range tt.streaks {
				word := fmt.Sprintf("word-%d", i)
				r.AddWord(word, streak)
				switch {
				case i < tt.batched:
					toBatch = append(toBatch, word)
				case i < tt.batched+tt.queued:
					toQueue = append(toQueue, word)
				}
			}
			if len(toBatch) > 0 {
				r.SeedBatch(toBatch...)
			}
			if len(toQueue) > 0 {
				r.SeedQueue(toQueue...)
			}

			got, err := r.GetTotalStats(context.Background(), dal.TestChatID)
			if err != nil {
				t.Fatalf("GetTotalStats: %v", err)
			}

			if got.Learned != tt.wantLearned {
				t.Errorf("Learned = %d, want %d", got.Learned, tt.wantLearned)
			}
			if got.Nearly != tt.wantNearly {
				t.Errorf("Nearly = %d, want %d", got.Nearly, tt.wantNearly)
			}
			if got.Early != tt.wantEarly {
				t.Errorf("Early = %d, want %d", got.Early, tt.wantEarly)
			}
			if got.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", got.Total, tt.wantTotal)
			}
			if got.StreakLimit != tt.wantStreakLimit {
				t.Errorf("StreakLimit = %d, want %d", got.StreakLimit, tt.wantStreakLimit)
			}
			if got.NearlyFrom != tt.wantNearlyFr {
				t.Errorf("NearlyFrom = %d, want %d", got.NearlyFrom, tt.wantNearlyFr)
			}
			if got.Batched != tt.wantBatched {
				t.Errorf("Batched = %d, want %d", got.Batched, tt.wantBatched)
			}
			if got.Queued != tt.wantQueued {
				t.Errorf("Queued = %d, want %d", got.Queued, tt.wantQueued)
			}
		})
	}
}

func TestGetTotalStatsNoWords(t *testing.T) {
	r := dal.NewTestRepo(t)
	r.SetStreakLimit(8)

	got, err := r.GetTotalStats(context.Background(), dal.TestChatID)
	if err != nil {
		t.Fatalf("GetTotalStats: %v", err)
	}
	if got.Total != 0 || got.Learned != 0 {
		t.Errorf("got %+v, want zeroed stats", got)
	}
	// The counts are empty but the thresholds are not: a streak limit of 0 would make every caller
	// treat every word as learned.
	if got.StreakLimit != 8 {
		t.Errorf("StreakLimit = %d, want 8", got.StreakLimit)
	}
	if got.NearlyFrom != 3 {
		t.Errorf("NearlyFrom = %d, want 3", got.NearlyFrom)
	}
	if got.Batched != 0 {
		t.Errorf("Batched = %d, want 0", got.Batched)
	}
	if got.Queued != 0 {
		t.Errorf("Queued = %d, want 0", got.Queued)
	}
}

func TestFindWordTranslationsLearnedFilterUsesStreakLimit(t *testing.T) {
	ctx := context.Background()
	r := dal.NewTestRepo(t)
	r.SetStreakLimit(5)

	r.AddWord("below", 4)
	r.AddWord("at", 5)
	r.AddWord("above", 9)

	got, total, err := r.FindWordTranslations(ctx, dal.TestChatID, dal.WordTranslationsFilter{
		Guessed: dal.GuessedLearned,
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("FindWordTranslations: %v", err)
	}

	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	for _, wt := range got {
		if wt.Word == "below" {
			t.Error("word below the streak limit was reported as learned")
		}
	}
}

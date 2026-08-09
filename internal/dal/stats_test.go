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
		wantLearned, wantNearly       int
		wantEarly, wantTotal          int
		wantStreakLimit, wantNearlyFr int
	}{
		{
			name:        "default limit keeps the historical 15+/10-14/1-9 split",
			streakLimit: 15,
			streaks:     []int{0, 1, 9, 10, 14, 15, 20},
			wantLearned: 2, wantNearly: 2, wantEarly: 2, wantTotal: 7,
			wantStreakLimit: 15, wantNearlyFr: 10,
		},
		{
			name:        "retuned limit moves every bucket together",
			streakLimit: 8,
			// limit 8 => learned >= 8, nearly 3-7, early 1-2
			streaks:     []int{0, 1, 2, 3, 7, 8, 12},
			wantLearned: 2, wantNearly: 2, wantEarly: 2, wantTotal: 7,
			wantStreakLimit: 8, wantNearlyFr: 3,
		},
		{
			name:        "limit below the nearly width collapses the early bucket",
			streakLimit: 3,
			streaks:     []int{0, 1, 2, 3},
			wantLearned: 1, wantNearly: 2, wantEarly: 0, wantTotal: 4,
			wantStreakLimit: 3, wantNearlyFr: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := dal.NewTestRepo(t)
			r.SetStreakLimit(tt.streakLimit)
			for i, streak := range tt.streaks {
				r.AddWord(fmt.Sprintf("word-%d", i), streak)
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
		})
	}
}

func TestGetTotalStatsNoWords(t *testing.T) {
	r := dal.NewTestRepo(t)

	got, err := r.GetTotalStats(context.Background(), dal.TestChatID)
	if err != nil {
		t.Fatalf("GetTotalStats: %v", err)
	}
	if got.Total != 0 || got.Learned != 0 {
		t.Errorf("got %+v, want zeroed stats", got)
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

// This file stays in package telegram: totalStatsMessage is unexported, and the alternative —
// exercising it through HandleStats — would need a real telebot context for no extra coverage.

package telegram

import (
	"testing"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

// The bucket labels are derived from the configured streak limit, so a small limit used to squeeze
// the early band into the impossible range "1-0" — a line that could never report anything but 0.
func TestTotalStatsMessage(t *testing.T) {
	tests := []struct {
		name  string
		stats dal.TotalStats
		want  string
	}{
		{
			name: "default limit shows every bucket",
			stats: dal.TotalStats{
				Learned: 3, Nearly: 2, Early: 5, Total: 12,
				StreakLimit: 15, NearlyFrom: 10, Batched: 4, Queued: 6,
			},
			want: "Overall Progress:\n15+: 3\n10-14: 2\n1-9: 5\nTotal: 12\nIn learning batch: 4\nWaiting in queue: 6",
		},
		{
			name: "small limit drops the early bucket instead of labelling it 1-0",
			stats: dal.TotalStats{
				Learned: 3, Nearly: 2, Early: 0, Total: 6,
				StreakLimit: 5, NearlyFrom: 1, Batched: 2, Queued: 1,
			},
			want: "Overall Progress:\n5+: 3\n1-4: 2\nTotal: 6\nIn learning batch: 2\nWaiting in queue: 1",
		},
		{
			name: "a limit of 1 leaves nothing between learned and nothing",
			stats: dal.TotalStats{
				Learned: 4, Total: 6,
				StreakLimit: 1, NearlyFrom: 1, Batched: 0, Queued: 0,
			},
			want: "Overall Progress:\n1+: 4\nTotal: 6\nIn learning batch: 0\nWaiting in queue: 0",
		},
		{
			name: "an empty vocabulary still labels the buckets",
			stats: dal.TotalStats{
				StreakLimit: 15, NearlyFrom: 10,
			},
			want: "Overall Progress:\n15+: 0\n10-14: 0\n1-9: 0\nTotal: 0\nIn learning batch: 0\nWaiting in queue: 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := totalStatsMessage(&tt.stats); got != tt.want {
				t.Errorf("totalStatsMessage() =\n%s\n\nwant\n%s", got, tt.want)
			}
		})
	}
}

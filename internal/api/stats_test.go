package api_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/api"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

// stubStatsRepo implements dal.StatsRepository, returning canned results.
type stubStatsRepo struct {
	totalStats *dal.TotalStats
}

func (s *stubStatsRepo) GetTotalStats(_ context.Context, _ int64) (*dal.TotalStats, error) {
	return s.totalStats, nil
}

func (s *stubStatsRepo) GetStats(_ context.Context, _ int64, _ time.Time) (*dal.Stats, error) {
	return nil, dal.ErrNotFound
}

func (s *stubStatsRepo) GetStatsRange(_ context.Context, _ int64, _, _ time.Time) ([]dal.Stats, error) {
	return nil, nil
}

var _ dal.StatsRepository = (*stubStatsRepo)(nil)

func TestTotalStatsIncludesBatched(t *testing.T) {
	repo := &stubStatsRepo{totalStats: &dal.TotalStats{
		Learned:     3,
		Total:       12,
		StreakLimit: 15,
		Batched:     4,
	}}
	h := api.NewStatsHandler(repo, testLogger())

	c, rec := newRequest(t, "/stats/total", "")
	if err := h.TotalStats(c); err != nil {
		t.Fatalf("TotalStats: %v", err)
	}

	assertStatus(t, rec, 200)

	var body map[string]int
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	want := map[string]int{"learned": 3, "total": 12, "streak_limit": 15, "batched": 4}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("body[%q] = %d, want %d", k, body[k], v)
		}
	}
}

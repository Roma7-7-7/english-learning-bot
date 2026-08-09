package schedule

import (
	"context"
	"log/slog"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

const (
	processTimeout = 10 * time.Second
)

func StartUpdateBatchSchedule(ctx context.Context, chatIDs []int64, batchSize, guessedStreakLimit int, repo dal.Repository, log *slog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			log.ErrorContext(ctx, "panic", "error", r)
		}
	}()

	log.InfoContext(ctx, "update learning batch schedule started")
	defer log.InfoContext(ctx, "update learning batch schedule stopped")
	runIn := time.After(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-runIn:
			runIn = time.After(1 * time.Hour)

			log.DebugContext(ctx, "update learning batch execution started")
			for _, chatID := range chatIDs {
				ctx, cancel := context.WithTimeout(ctx, processTimeout)

				evicted, added, err := repo.RefillLearningBatch(ctx, chatID, batchSize, guessedStreakLimit)
				if err != nil {
					log.ErrorContext(ctx, "failed to refill learning batch", "error", err, "chat_id", chatID)
				} else {
					log.DebugContext(ctx, "learning batch refilled", "chat_id", chatID, "evicted", evicted, "added", added)
				}
				cancel()
			}
			log.DebugContext(ctx, "update learning batch execution finished")
		}
	}
}

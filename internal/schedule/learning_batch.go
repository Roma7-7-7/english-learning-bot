package schedule

import (
	"context"
	"errors"
	"fmt"
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

				err := repo.Transact(ctx, func(repo dal.Repository) error {
					return updateLearningBatch(ctx, chatID, guessedStreakLimit, repo, log, batchSize)
				})
				if err != nil {
					log.ErrorContext(ctx, "failed to delete from learning batch", "error", err, "chat_id", chatID)
				}
				cancel()
			}
			log.DebugContext(ctx, "update learning batch execution finished")
		}
	}
}

func updateLearningBatch(ctx context.Context, chatID int64, guessedStreakLimit int, repo dal.Repository, log *slog.Logger, batchSize int) error {
	deleted, err := repo.DeleteFromLearningBatchGtGuessedStreak(ctx, chatID, guessedStreakLimit)
	if err != nil {
		return fmt.Errorf("delete from learning batch: %w", err)
	}
	log.DebugContext(ctx, "deleted from learning batch", "chat_id", chatID, "deleted", deleted)

	batched, err := repo.GetBatchedWordTranslationsCount(ctx, chatID)
	if err != nil {
		return fmt.Errorf("get batched word translations count: %w", err)
	}

	for range batchSize - batched {
		word, err := repo.FindRandomWordTranslation(ctx, chatID, dal.FindRandomWordFilter{
			StreakLimitDirection: dal.LimitDirectionLessThan,
			StreakLimit:          guessedStreakLimit,
		})
		if err != nil {
			if errors.Is(err, dal.ErrNotFound) {
				log.DebugContext(ctx, "no words to add to learning batch", "chat_id", chatID)
				return nil
			}
			return fmt.Errorf("get random not batched word translation: %w", err)
		}
		if err = repo.AddToLearningBatch(ctx, chatID, word.Word); err != nil {
			return fmt.Errorf("add to learning batch: %w", err)
		}
	}
	log.DebugContext(ctx, "added to learning batch", "chat_id", chatID, "added", batchSize-batched)

	return nil
}

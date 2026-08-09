package dal

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
)

// RegisterGuess records a correct answer: the word's streak grows and today's counters follow it.
func (r *SQLiteRepository) RegisterGuess(ctx context.Context, chatID int64, word string) error {
	return r.inTx(ctx, func(e execer) error {
		if err := increaseGuessedStreak(ctx, e, chatID, word); err != nil {
			return fmt.Errorf("increase guessed streak: %w", err)
		}
		if err := incrementWordGuessed(ctx, e, chatID); err != nil {
			return fmt.Errorf("increment word guessed: %w", err)
		}
		if err := updateTotalWordsLearned(ctx, e, chatID, r.streakLimit); err != nil {
			return fmt.Errorf("update total words learned: %w", err)
		}
		return nil
	})
}

// RegisterMiss records a wrong answer: the word's streak drops back to zero and today's counters
// follow it.
func (r *SQLiteRepository) RegisterMiss(ctx context.Context, chatID int64, word string) error {
	return r.inTx(ctx, func(e execer) error {
		if err := resetGuessedStreak(ctx, e, chatID, word); err != nil {
			return fmt.Errorf("reset guessed streak: %w", err)
		}
		if err := incrementWordMissed(ctx, e, chatID); err != nil {
			return fmt.Errorf("increment word missed: %w", err)
		}
		if err := updateTotalWordsLearned(ctx, e, chatID, r.streakLimit); err != nil {
			return fmt.Errorf("update total words learned: %w", err)
		}
		return nil
	})
}

// RefillLearningBatch evicts words that reached guessedStreakLimit from the learning batch and tops
// it back up to batchSize with randomly chosen words that are still being learned.
//
// The batch is allowed to sit above batchSize (words can be pushed back into it out of band); in that
// case nothing is added until eviction frees up room again.
func (r *SQLiteRepository) RefillLearningBatch(ctx context.Context, chatID int64, batchSize, guessedStreakLimit int) (int, int, error) {
	var evicted, added int

	err := r.inTx(ctx, func(e execer) error {
		var err error
		if evicted, err = deleteFromLearningBatchGeGuessedStreak(ctx, e, chatID, guessedStreakLimit); err != nil {
			return fmt.Errorf("delete from learning batch: %w", err)
		}

		batched, err := batchedWordTranslationsCount(ctx, e, chatID)
		if err != nil {
			return fmt.Errorf("get batched word translations count: %w", err)
		}

		room := batchSize - batched
		if room <= 0 {
			return nil
		}

		if added, err = fillLearningBatch(ctx, e, chatID, guessedStreakLimit, room); err != nil {
			return fmt.Errorf("fill learning batch: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}

	return evicted, added, nil
}

func fillLearningBatch(ctx context.Context, e execer, chatID int64, guessedStreakLimit, limit int) (int, error) {
	// The nested select is built with the package-level builder (":?" placeholders) so that the
	// outer builder's Dollar format is applied exactly once, over the whole statement.
	query := qb.Insert("learning_batches").
		Columns("chat_id", "word").
		Select(squirrel.Select("chat_id", "word").
			From("word_translations").
			Where("chat_id = ? AND guessed_streak < ?", chatID, guessedStreakLimit).
			Where("word NOT IN (SELECT word FROM learning_batches WHERE chat_id = ?)", chatID).
			OrderBy("random()").
			Limit(uint64(limit))). //nolint:gosec // limit is bounded by batchSize
		Suffix("ON CONFLICT DO NOTHING")

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build insert query: %w", err)
	}

	res, err := e.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("add to learning batch: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return int(affected), nil
}

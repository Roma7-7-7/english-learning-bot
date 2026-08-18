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

// RegisterMiss records a wrong answer: the word's streak drops back to zero, the word requests batch
// membership again, and today's counters follow.
//
// Requesting membership is what stops a forgotten word from disappearing again. It matters most for
// words that had been learned and were only being reviewed; for words already in the batch, or
// already queued behind it, the request is a no-op. BOT_LEARNING_BATCH_SIZE is a hard cap: if the
// batch is full the word is appended to learning_batch_queue instead of being lost, and is drained
// oldest-first the next time RefillLearningBatch runs.
func (r *SQLiteRepository) RegisterMiss(ctx context.Context, chatID int64, word string) error {
	return r.inTx(ctx, func(e execer) error {
		if err := resetGuessedStreak(ctx, e, chatID, word); err != nil {
			return fmt.Errorf("reset guessed streak: %w", err)
		}
		if err := requestBatchMembership(ctx, e, chatID, word, r.batchSize); err != nil {
			return fmt.Errorf("request batch membership: %w", err)
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

// ResetStreak drops a word's streak back to zero on purpose, optionally requesting batch membership
// so that it is asked again soon.
//
// Unlike RegisterMiss this is a deliberate correction rather than a wrong answer, so it does not
// touch the daily guessed/missed counters. Requesting membership when the batch is full queues the
// word instead of overflowing the configured size; RefillLearningBatch drains it oldest-first once
// there is room again.
func (r *SQLiteRepository) ResetStreak(ctx context.Context, chatID int64, word string, addToBatch bool) error {
	return r.inTx(ctx, func(e execer) error {
		if err := resetGuessedStreak(ctx, e, chatID, word); err != nil {
			return fmt.Errorf("reset guessed streak: %w", err)
		}
		if addToBatch {
			if err := requestBatchMembership(ctx, e, chatID, word, r.batchSize); err != nil {
				return fmt.Errorf("request batch membership: %w", err)
			}
		}
		if err := updateTotalWordsLearned(ctx, e, chatID, r.streakLimit); err != nil {
			return fmt.Errorf("update total words learned: %w", err)
		}
		return nil
	})
}

// ResolveWordConflict applies the user's decision about a word they tried to add that already
// exists: the given translation and description always win, and resolution decides what happens to
// the existing learning progress.
//
// Which translation to keep is not encoded here — the caller passes whichever text the user chose.
func (r *SQLiteRepository) ResolveWordConflict(
	ctx context.Context, chatID int64, word, translation, description string, resolution ConflictResolution,
) error {
	switch resolution {
	case ResolveResetAndBatch, ResolveResetOnly, ResolveUpdateOnly:
	default:
		return fmt.Errorf("unknown conflict resolution: %q", resolution)
	}

	return r.inTx(ctx, func(e execer) error {
		if err := upsertWordTranslation(ctx, e, chatID, word, translation, description); err != nil {
			return fmt.Errorf("upsert word translation: %w", err)
		}

		if resolution != ResolveUpdateOnly {
			if err := resetGuessedStreak(ctx, e, chatID, word); err != nil {
				return fmt.Errorf("reset guessed streak: %w", err)
			}
		}
		if resolution == ResolveResetAndBatch {
			if err := requestBatchMembership(ctx, e, chatID, word, r.batchSize); err != nil {
				return fmt.Errorf("request batch membership: %w", err)
			}
		}

		if err := updateTotalWordsLearned(ctx, e, chatID, r.streakLimit); err != nil {
			return fmt.Errorf("update total words learned: %w", err)
		}
		return nil
	})
}

// MarkWordReviewed records that a word has just been offered for review.
//
// It is called when the review is sent, not when it is answered, so that an ignored message still
// advances the rotation instead of pinning it to the same word forever.
func (r *SQLiteRepository) MarkWordReviewed(ctx context.Context, chatID int64, word string) error {
	// The cursor is one past the chat's current maximum. The subquery sees the pre-update state, so
	// the new value is always strictly greater than every other row's and the rotation can never
	// stall on a tie.
	query := qb.Update("word_translations").
		Set("last_reviewed_seq", squirrel.Expr(
			"COALESCE((SELECT MAX(last_reviewed_seq) FROM word_translations WHERE chat_id = ?), 0) + 1", chatID)).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	if _, err = r.db.ExecContext(ctx, sql, args...); err != nil {
		return fmt.Errorf("mark word reviewed: %w", err)
	}
	return nil
}

// RefillLearningBatch evicts words that reached the streak limit from the learning batch, then tops
// it back up to batchSize: first draining learning_batch_queue oldest-first (words that explicitly
// requested membership - a miss, a deliberate reset, a conflict resolution, or a new word - while the
// batch was full), and only once the queue is exhausted falling back to a random pick from the wider
// pool of still-eligible words nobody has explicitly asked to re-admit.
//
// The drain always runs before the fallback, so a still-queued word can never be skipped by the
// random pick while it waits its turn.
func (r *SQLiteRepository) RefillLearningBatch(ctx context.Context, chatID int64) (int, int, error) {
	var evicted, added int

	err := r.inTx(ctx, func(e execer) error {
		var err error
		if evicted, err = deleteFromLearningBatchGeGuessedStreak(ctx, e, chatID, r.streakLimit); err != nil {
			return fmt.Errorf("delete from learning batch: %w", err)
		}

		batched, err := batchedWordTranslationsCount(ctx, e, chatID)
		if err != nil {
			return fmt.Errorf("get batched word translations count: %w", err)
		}

		room := r.batchSize - batched
		if room <= 0 {
			return nil
		}

		drained, err := drainLearningBatchQueue(ctx, e, chatID, r.streakLimit, room)
		if err != nil {
			return fmt.Errorf("drain learning batch queue: %w", err)
		}
		added += drained
		room -= drained
		if room <= 0 {
			return nil
		}

		filled, err := fillLearningBatch(ctx, e, chatID, r.streakLimit, room)
		if err != nil {
			return fmt.Errorf("fill learning batch: %w", err)
		}
		added += filled
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

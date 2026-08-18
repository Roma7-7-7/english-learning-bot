package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"golang.org/x/sync/errgroup"
)

// CreateWordTranslation stores a word that is not supposed to exist yet, reporting ErrAlreadyExists
// instead of overwriting one that does, and requests batch membership for it - a brand-new word wants
// practicing exactly like a missed or deliberately reset one, so it goes through the same admission
// gate rather than waiting for the next hourly refill's random pick.
//
// The check is the insert itself rather than a preceding lookup: two concurrent creates would both
// find nothing and the second would silently discard the first, along with its learning progress.
// Overwriting on purpose goes through ResolveWordConflict.
func (r *SQLiteRepository) CreateWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error {
	return r.inTx(ctx, func(e execer) error {
		query := qb.Insert("word_translations").
			Columns("chat_id", "word", "translation", "description").
			Values(chatID, word, translation, description).
			Suffix("ON CONFLICT (chat_id, word) DO NOTHING")

		sql, args, err := query.ToSql()
		if err != nil {
			return fmt.Errorf("build insert query: %w", err)
		}

		res, err := e.ExecContext(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("create translation: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("get rows affected: %w", err)
		}
		if affected == 0 {
			return ErrAlreadyExists
		}

		if err := requestBatchMembership(ctx, e, chatID, word, r.batchSize); err != nil {
			return fmt.Errorf("request batch membership: %w", err)
		}
		return nil
	})
}

func upsertWordTranslation(ctx context.Context, e execer, chatID int64, word, translation, description string) error {
	query := qb.Insert("word_translations").
		Columns("chat_id", "word", "translation", "description").
		Values(chatID, word, translation, description).
		Suffix("ON CONFLICT (chat_id, word) DO UPDATE SET translation = EXCLUDED.translation, description = EXCLUDED.description")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = e.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("add translation: %w", err)
	}
	return nil
}

// FindWordTranslations runs its page and count queries concurrently, so it must never be called from
// inside inTx: *sql.Tx is not safe for concurrent use and the two queries would deadlock.
func (r *SQLiteRepository) FindWordTranslations(ctx context.Context, chatID int64, filter WordTranslationsFilter) ([]WordTranslation, int, error) {
	baseQuery := qb.Select().
		From("word_translations wt").
		Where(squirrel.Eq{"wt.chat_id": chatID})

	if filter.Word != "" {
		// Search in both word and translation fields for SQLite compatibility
		searchTerm := fmt.Sprintf("%%%s%%", strings.ToLower(filter.Word))
		baseQuery = baseQuery.Where(
			squirrel.Or{
				squirrel.Expr("LOWER(wt.word) LIKE ?", searchTerm),
				squirrel.Expr("LOWER(wt.translation) LIKE ?", searchTerm),
			},
		)
	}

	if filter.ToReview {
		baseQuery = baseQuery.Where(squirrel.Eq{"wt.to_review": filter.ToReview})
	}

	switch filter.Guessed {
	case "", GuessedAll:
	case GuessedLearned:
		baseQuery = baseQuery.Where(squirrel.Expr("wt.guessed_streak >= ?", r.streakLimit))
	case GuessedBatched:
		baseQuery = baseQuery.Where("EXISTS (SELECT 1 FROM learning_batches lb WHERE lb.chat_id = wt.chat_id AND lb.word = wt.word)")
	case GuessedToLearn:
		baseQuery = baseQuery.Where("wt.guessed_streak = 0")
	}

	selectQuery2 := baseQuery.
		Columns(wordTranslationColumns()...).
		OrderBy("wt.word").
		Offset(filter.Offset).
		Limit(filter.Limit)

	countQuery2 := baseQuery.Columns("COUNT(*)")
	selectQuery, countQuery := selectQuery2, countQuery2

	eg, ctx := errgroup.WithContext(ctx)
	res := make([]WordTranslation, 0, 25) //nolint:mnd // let it be default limit
	total := 0

	eg.Go(func() error {
		sql, args, err := selectQuery.ToSql()
		if err != nil {
			return fmt.Errorf("build select query: %w", err)
		}

		rows, err := r.db.QueryContext(ctx, sql, args...)
		if err != nil {
			return fmt.Errorf("find translations: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			wt, err := hydrateWordTranslation(rows)
			if err != nil {
				return fmt.Errorf("scan word translation: %w", err)
			}
			res = append(res, *wt)
		}

		if rows.Err() != nil {
			return fmt.Errorf("iterate word translations: %w", rows.Err())
		}

		return nil
	})

	eg.Go(func() error {
		sql, args, err := countQuery.ToSql()
		if err != nil {
			return fmt.Errorf("build count query: %w", err)
		}

		if err := r.db.QueryRowContext(ctx, sql, args...).Scan(&total); err != nil {
			return fmt.Errorf("get total: %w", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, 0, fmt.Errorf("find word_translations: %w", err)
	}

	return res, total, nil
}

func (r *SQLiteRepository) DeleteWordTranslation(ctx context.Context, chatID int64, word string) error {
	query := qb.Delete("word_translations").
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete translation: %w", err)
	}
	return nil
}

func addToLearningBatch(ctx context.Context, e execer, chatID int64, word string) error {
	query := qb.Insert("learning_batches").
		Columns("chat_id", "word").
		Values(chatID, word).
		Suffix("ON CONFLICT DO NOTHING")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = e.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("add to learning batch: %w", err)
	}
	return nil
}

// wordInBatchOrQueue reports whether requesting batch membership for word would have anything to do:
// true if it is already in the active batch or already waiting in the admission queue behind it. A
// word is never in both at once - see requestBatchMembership.
func wordInBatchOrQueue(ctx context.Context, e execer, chatID int64, word string) (bool, error) {
	query := qb.Select().Column(
		"EXISTS(SELECT 1 FROM learning_batches WHERE chat_id = ? AND word = ?) OR "+
			"EXISTS(SELECT 1 FROM learning_batch_queue WHERE chat_id = ? AND word = ?)",
		chatID, word, chatID, word,
	)

	sql, args, err := query.ToSql()
	if err != nil {
		return false, fmt.Errorf("build select query: %w", err)
	}

	var awaiting bool
	if err := e.QueryRowContext(ctx, sql, args...).Scan(&awaiting); err != nil {
		return false, fmt.Errorf("check batch admission state: %w", err)
	}
	return awaiting, nil
}

// enqueueForLearningBatch appends word to the FIFO admission queue. ON CONFLICT DO NOTHING means a
// word that somehow gets queued twice keeps its original position rather than being pushed to the
// back - callers are expected to have already checked wordInBatchOrQueue, so this should only ever
// insert a fresh row in practice.
func enqueueForLearningBatch(ctx context.Context, e execer, chatID int64, word string) error {
	// Same "one past the chat's current maximum" pattern as MarkWordReviewed's last_reviewed_seq:
	// the subquery sees the pre-insert state, so the new value is always strictly greater than every
	// other queued word's and two words queued in the same instant never tie.
	query := qb.Insert("learning_batch_queue").
		Columns("chat_id", "word", "queued_seq").
		Values(chatID, word, squirrel.Expr(
			"COALESCE((SELECT MAX(queued_seq) FROM learning_batch_queue WHERE chat_id = ?), 0) + 1", chatID)).
		Suffix("ON CONFLICT DO NOTHING")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}
	if _, err = e.ExecContext(ctx, sql, args...); err != nil {
		return fmt.Errorf("enqueue for learning batch: %w", err)
	}
	return nil
}

// requestBatchMembership is the single admission gate every producer of "this word should be
// practiced" goes through: RegisterMiss, ResetStreak, ResolveWordConflict's reset_and_batch, and
// CreateWordTranslation for brand-new words.
//
// It admits immediately if there is room (today's "instant" UX for a miss or a deliberate reset), or
// appends to the FIFO queue otherwise so the word is delayed, never lost. It is idempotent - a word
// already in the batch or already queued is left exactly where it is, so two producers racing on the
// same word (or the same producer firing twice) never duplicates a row or reorders the queue.
func requestBatchMembership(ctx context.Context, e execer, chatID int64, word string, batchSize int) error {
	awaiting, err := wordInBatchOrQueue(ctx, e, chatID, word)
	if err != nil {
		return fmt.Errorf("check batch admission state: %w", err)
	}
	if awaiting {
		return nil
	}

	batched, err := batchedWordTranslationsCount(ctx, e, chatID)
	if err != nil {
		return fmt.Errorf("get batched word translations count: %w", err)
	}

	if batched < batchSize {
		if err := addToLearningBatch(ctx, e, chatID, word); err != nil {
			return fmt.Errorf("add to learning batch: %w", err)
		}
		return nil
	}

	if err := enqueueForLearningBatch(ctx, e, chatID, word); err != nil {
		return fmt.Errorf("enqueue for learning batch: %w", err)
	}
	return nil
}

// drainLearningBatchQueue admits up to limit of the oldest eligible queued words into
// learning_batches, then removes them from the queue: insert first, delete second, so a crash
// between the two never loses a word (worst case it briefly sits in both, never in neither).
//
// The guessed_streak filter is defensive: nothing should ever queue a word above guessedStreakLimit,
// since every producer resets the streak to 0 before requesting membership, but a stale row must not
// be promoted if that invariant is ever violated.
//
// The delete is safe precisely because of the batch/queue mutual-exclusivity invariant: within this
// transaction, the only rows that can be in both tables at this point are the ones the insert above
// just moved.
func drainLearningBatchQueue(ctx context.Context, e execer, chatID int64, guessedStreakLimit, limit int) (int, error) {
	insert := qb.Insert("learning_batches").
		Columns("chat_id", "word").
		Select(squirrel.Select("lbq.chat_id", "lbq.word").
			From("learning_batch_queue lbq").
			Join("word_translations wt ON wt.chat_id = lbq.chat_id AND wt.word = lbq.word").
			Where("lbq.chat_id = ? AND wt.guessed_streak < ?", chatID, guessedStreakLimit).
			OrderBy("lbq.queued_seq ASC").
			Limit(uint64(limit))). //nolint:gosec // limit is room, itself bounded by batchSize
		Suffix("ON CONFLICT DO NOTHING")

	sql, args, err := insert.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build insert query: %w", err)
	}
	res, err := e.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("drain learning batch queue: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	del := qb.Delete("learning_batch_queue").
		Where("chat_id = ? AND word IN (SELECT word FROM learning_batches WHERE chat_id = ?)", chatID, chatID)

	delSQL, delArgs, err := del.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete query: %w", err)
	}
	if _, err = e.ExecContext(ctx, delSQL, delArgs...); err != nil {
		return 0, fmt.Errorf("remove drained words from queue: %w", err)
	}

	return int(affected), nil
}

func increaseGuessedStreak(ctx context.Context, e execer, chatID int64, word string) error {
	query := qb.Update("word_translations").
		Set("guessed_streak", squirrel.Expr("guessed_streak + 1")).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = e.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("increase guessed streak: %w", err)
	}
	return nil
}

func resetGuessedStreak(ctx context.Context, e execer, chatID int64, word string) error {
	query := qb.Update("word_translations").
		Set("guessed_streak", 0).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = e.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("reset guessed streak: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) MarkToReview(ctx context.Context, chatID int64, word string, toReview bool) error {
	query := qb.Update("word_translations").
		Set("to_review", toReview).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("mark to review: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, updatedTranslation, description string) error {
	query := qb.Update("word_translations").
		Set("word", updatedWord).
		Set("translation", updatedTranslation).
		Set("description", description).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update translation: %w", err)
	}
	return nil
}

func batchedWordTranslationsCount(ctx context.Context, e execer, chatID int64) (int, error) {
	query := qb.Select("COUNT(*)").
		From("word_translations wt").
		Join("learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word").
		Where(squirrel.Eq{"wt.chat_id": chatID})

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	err = e.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get batched word translations count: %w", err)
	}
	return count, nil
}

func queuedWordCount(ctx context.Context, e execer, chatID int64) (int, error) {
	query := qb.Select("COUNT(*)").
		From("learning_batch_queue").
		Where(squirrel.Eq{"chat_id": chatID})

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	err = e.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get queued word count: %w", err)
	}
	return count, nil
}

func (r *SQLiteRepository) FindWordTranslation(ctx context.Context, chatID int64, word string) (*WordTranslation, error) {
	query := qb.Select(wordTranslationColumns()...).
		From("word_translations wt").
		Where(squirrel.Eq{"wt.chat_id": chatID, "wt.word": word})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	row := r.db.QueryRowContext(ctx, sqlQuery, args...)
	wt, err := hydrateWordTranslation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find word translation: %w", err)
	}
	return wt, nil
}

func (r *SQLiteRepository) FindRandomWordTranslation(ctx context.Context, chatID int64, filter FindRandomWordFilter) (*WordTranslation, error) {
	var query2 squirrel.SelectBuilder

	if filter.Batched {
		query2 = qb.Select(wordTranslationColumns()...).
			From("word_translations wt").
			Join("learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word").
			Where(squirrel.Eq{"wt.chat_id": chatID}).
			OrderBy("random()").
			Limit(1)
	} else {
		// NULL sorts first in SQLite's ASC, so words that have never been reviewed come out ahead
		// of any that have.
		orderBy := "random()"
		if filter.Order == OrderLeastRecentlyReviewed {
			orderBy = "wt.last_reviewed_seq ASC"
		}

		query2 = qb.Select(wordTranslationColumns()...).
			From("word_translations wt").
			Where(squirrel.Eq{"wt.chat_id": chatID}).
			Where(squirrel.Expr("wt.guessed_streak "+filter.StreakLimitDirection.String()+" ?", filter.StreakLimit)).
			Where("wt.word NOT IN (SELECT word FROM learning_batches WHERE chat_id = ?)", chatID).
			OrderBy(orderBy).
			Limit(1)
	}

	var r2 squirrel.Sqlizer = query2
	query := r2

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	row := r.db.QueryRowContext(ctx, sqlQuery, args...)
	wt, err := hydrateWordTranslation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get random word translation: %w", err)
	}
	return wt, nil
}

func deleteFromLearningBatchGeGuessedStreak(ctx context.Context, e execer, chatID int64, guessedStreakLimit int) (int, error) {
	query := qb.Delete("learning_batches").
		Where("chat_id = ? AND word IN (SELECT word FROM word_translations WHERE chat_id = ? AND guessed_streak >= ?)",
			chatID, chatID, guessedStreakLimit)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete query: %w", err)
	}

	res, err := e.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("delete from learning batch: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return int(affected), nil
}

// wordTranslationColumns returns the column list hydrateWordTranslation scans, in order. Every query
// that uses it must select from `word_translations wt`, since the columns are alias-qualified.
//
// Keep the two in sync: a query that hand-rolls its own column list will scan into the wrong fields
// the next time one is added here. It is built per call rather than kept in a package variable so
// that no caller can mutate the list every query shares.
func wordTranslationColumns() []string {
	return []string{
		"wt.chat_id", "wt.word", "wt.translation",
		"COALESCE(wt.description, '')", "wt.guessed_streak",
		"wt.to_review", "wt.created_at", "wt.updated_at",
		// Folds in the admission queue: requesting membership again is a no-op whether the word is
		// sitting in the batch or waiting behind it, so the conflict-resolution "would this change
		// anything?" question (see api.WordTranslation.InBatch) should get the same answer either
		// way. The word-list "batched" filter (GuessedBatched, above) deliberately does NOT do this -
		// a queued word is not actively being asked about right now and should not show up there.
		"EXISTS (SELECT 1 FROM learning_batches blb WHERE blb.chat_id = wt.chat_id AND blb.word = wt.word) OR " +
			"EXISTS (SELECT 1 FROM learning_batch_queue blq WHERE blq.chat_id = wt.chat_id AND blq.word = wt.word)",
	}
}

func hydrateWordTranslation(row interface {
	Scan(dest ...interface{}) error
}) (*WordTranslation, error) {
	var wt WordTranslation
	err := row.Scan(
		&wt.ChatID,
		&wt.Word,
		&wt.Translation,
		&wt.Description,
		&wt.GuessedStreak,
		&wt.ToReview,
		&wt.CreatedAt,
		&wt.UpdatedAt,
		&wt.InBatch,
	)
	if err != nil {
		return nil, fmt.Errorf("scan word translation: %w", err)
	}
	return &wt, nil
}

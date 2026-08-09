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
// instead of overwriting one that does.
//
// The check is the insert itself rather than a preceding lookup: two concurrent creates would both
// find nothing and the second would silently discard the first, along with its learning progress.
// Overwriting on purpose goes through ResolveWordConflict.
func (r *SQLiteRepository) CreateWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error {
	query := qb.Insert("word_translations").
		Columns("chat_id", "word", "translation", "description").
		Values(chatID, word, translation, description).
		Suffix("ON CONFLICT (chat_id, word) DO NOTHING")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	res, err := r.db.ExecContext(ctx, sql, args...)
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
	return nil
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
		"EXISTS (SELECT 1 FROM learning_batches blb WHERE blb.chat_id = wt.chat_id AND blb.word = wt.word)",
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

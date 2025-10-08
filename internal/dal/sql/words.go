package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"golang.org/x/sync/errgroup"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func (r *SQLiteRepository) AddWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error {
	query := r.qb.Insert("word_translations").
		Columns("chat_id", "word", "translation", "description").
		Values(chatID, word, translation, description).
		Suffix("ON CONFLICT (chat_id, word) DO UPDATE SET translation = EXCLUDED.translation, description = EXCLUDED.description")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("add translation: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) FindWordTranslations(ctx context.Context, chatID int64, filter dal.WordTranslationsFilter) ([]dal.WordTranslation, int, error) {
	baseQuery := r.qb.Select().
		From("word_translations").
		Where(squirrel.Eq{"chat_id": chatID})

	if filter.Word != "" {
		// Search in both word and translation fields for SQLite compatibility
		searchTerm := fmt.Sprintf("%%%s%%", strings.ToLower(filter.Word))
		baseQuery = baseQuery.Where(
			squirrel.Or{
				squirrel.Expr("LOWER(word) LIKE ?", searchTerm),
				squirrel.Expr("LOWER(translation) LIKE ?", searchTerm),
			},
		)
	}

	if filter.ToReview {
		baseQuery = baseQuery.Where(squirrel.Eq{"to_review": filter.ToReview})
	}

	switch filter.Guessed {
	case "", dal.GuessedAll:
	case dal.GuessedLearned:
		baseQuery = baseQuery.Where("guessed_streak >= 15")
	case dal.GuessedBatched:
		baseQuery = baseQuery.Where("EXISTS (SELECT 1 FROM learning_batches lb WHERE lb.chat_id = word_translations.chat_id AND lb.word = word_translations.word)")
	case dal.GuessedToLearn:
		baseQuery = baseQuery.Where("guessed_streak = 0")
	}

	selectQuery2 := baseQuery.
		Columns("chat_id", "word", "translation", "COALESCE(description, '')", "guessed_streak", "to_review", "created_at", "updated_at").
		OrderBy("word").
		Offset(filter.Offset).
		Limit(filter.Limit)

	countQuery2 := baseQuery.Columns("COUNT(*)")
	selectQuery, countQuery := selectQuery2, countQuery2

	eg, ctx := errgroup.WithContext(ctx)
	res := make([]dal.WordTranslation, 0, filter.Limit)
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
		return nil, 0, err
	}

	return res, total, nil
}

func (r *SQLiteRepository) DeleteWordTranslation(ctx context.Context, chatID int64, word string) error {
	query := r.qb.Delete("word_translations").
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

func (r *SQLiteRepository) AddToLearningBatch(ctx context.Context, chatID int64, word string) error {
	query := r.qb.Insert("learning_batches").
		Columns("chat_id", "word").
		Values(chatID, word).
		Suffix("ON CONFLICT DO NOTHING")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("add to learning batch: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error {
	query := r.qb.Update("word_translations").
		Set("guessed_streak", squirrel.Expr("guessed_streak + 1")).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("increase guessed streak: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ResetGuessedStreak(ctx context.Context, chatID int64, word string) error {
	query := r.qb.Update("word_translations").
		Set("guessed_streak", 0).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("reset guessed streak: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) MarkToReview(ctx context.Context, chatID int64, word string, toReview bool) error {
	query := r.qb.Update("word_translations").
		Set("to_review", toReview).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("mark review and reset streak: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, updatedTranslation, description string) error {
	query := r.qb.Update("word_translations").
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

func (r *SQLiteRepository) ResetToReview(ctx context.Context, chatID int64) error {
	query := r.qb.Update("word_translations").
		Set("to_review", false).
		Where(squirrel.Eq{"chat_id": chatID})

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("reset to review: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error) {
	query := r.qb.Select("COUNT(*)").
		From("word_translations wt").
		Join("learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word").
		Where(squirrel.Eq{"wt.chat_id": chatID})

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	err = r.db.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get batched word translations count: %w", err)
	}
	return count, nil
}

func (r *SQLiteRepository) FindWordTranslation(ctx context.Context, chatID int64, word string) (*dal.WordTranslation, error) {
	query := r.qb.Select(
		"wt.chat_id", "wt.word", "wt.translation",
		"COALESCE(wt.description, '')", "wt.guessed_streak",
		"wt.to_review", "wt.created_at", "wt.updated_at",
	).
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
			return nil, dal.ErrNotFound
		}
		return nil, fmt.Errorf("find word translation: %w", err)
	}
	return wt, nil
}

func (r *SQLiteRepository) FindRandomWordTranslation(ctx context.Context, chatID int64, filter dal.FindRandomWordFilter) (*dal.WordTranslation, error) {
	var query2 squirrel.SelectBuilder

	if filter.Batched {
		query2 = r.qb.Select(
			"wt.chat_id", "wt.word", "wt.translation",
			"COALESCE(wt.description, '')", "wt.guessed_streak",
			"wt.to_review", "wt.created_at", "wt.updated_at",
		).
			From("word_translations wt").
			Join("learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word").
			Where(squirrel.Eq{"wt.chat_id": chatID}).
			OrderBy("random()").
			Limit(1)
	} else {
		query2 = r.qb.Select(
			"wt.chat_id", "wt.word", "wt.translation",
			"COALESCE(wt.description, '')", "wt.guessed_streak",
			"wt.to_review", "wt.created_at", "wt.updated_at",
		).
			From("word_translations wt").
			Where(squirrel.Eq{"wt.chat_id": chatID}).
			Where(squirrel.Expr("wt.guessed_streak "+filter.StreakLimitDirection.String()+" ?", filter.StreakLimit)).
			Where("wt.word NOT IN (SELECT word FROM learning_batches WHERE chat_id = ?)", chatID).
			OrderBy("random()").
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
			return nil, dal.ErrNotFound
		}
		return nil, fmt.Errorf("get random word translation: %w", err)
	}
	return wt, nil
}

func (r *SQLiteRepository) DeleteFromLearningBatchGtGuessedStreak(ctx context.Context, chatID int64, guessedStreakLimit int) (int, error) {
	query := r.qb.Delete("learning_batches").
		Where("chat_id = ? AND word IN (SELECT word FROM word_translations WHERE chat_id = ? AND guessed_streak >= ?)",
			chatID, chatID, guessedStreakLimit)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete query: %w", err)
	}

	res, err := r.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("delete from learning batch: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return int(affected), nil
}

func hydrateWordTranslation(row interface {
	Scan(dest ...interface{}) error
}) (*dal.WordTranslation, error) {
	var wt dal.WordTranslation
	err := row.Scan(
		&wt.ChatID,
		&wt.Word,
		&wt.Translation,
		&wt.Description,
		&wt.GuessedStreak,
		&wt.ToReview,
		&wt.CreatedAt,
		&wt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan word translation: %w", err)
	}
	return &wt, nil
}

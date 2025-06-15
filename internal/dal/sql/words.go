package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func (r *Repository) AddWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error {
	query := r.queries.AddWordTranslationQuery(chatID, word, translation, description)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("add translation: %w", err)
	}
	return nil
}

func (r *Repository) FindWordTranslations(ctx context.Context, chatID int64, filter dal.WordTranslationsFilter) ([]dal.WordTranslation, int, error) {
	selectQuery, countQuery := r.queries.FindWordTranslationsQuery(chatID, filter)

	eg, ctx := errgroup.WithContext(ctx)
	res := make([]dal.WordTranslation, 0, filter.Limit)
	total := 0

	eg.Go(func() error {
		sql, args, err := selectQuery.ToSql()
		if err != nil {
			return fmt.Errorf("build select query: %w", err)
		}

		rows, err := r.client.QueryContext(ctx, sql, args...)
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

		if err := r.client.QueryRowContext(ctx, sql, args...).Scan(&total); err != nil {
			return fmt.Errorf("get total: %w", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}

	return res, total, nil
}

func (r *Repository) DeleteWordTranslation(ctx context.Context, chatID int64, word string) error {
	query := r.queries.DeleteWordTranslationQuery(chatID, word)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build delete query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete translation: %w", err)
	}
	return nil
}

func (r *Repository) AddToLearningBatch(ctx context.Context, chatID int64, word string) error {
	query := r.queries.AddToLearningBatchQuery(chatID, word)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build insert query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("add to learning batch: %w", err)
	}
	return nil
}

func (r *Repository) IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error {
	query := r.queries.IncreaseGuessedStreakQuery(chatID, word)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("increase guessed streak: %w", err)
	}
	return nil
}

func (r *Repository) ResetGuessedStreak(ctx context.Context, chatID int64, word string) error {
	query := r.queries.ResetGuessedStreakQuery(chatID, word)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("reset guessed streak: %w", err)
	}
	return nil
}

func (r *Repository) MarkToReview(ctx context.Context, chatID int64, word string, toReview bool) error {
	query := r.queries.MarkToReviewQuery(chatID, word, toReview)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("mark review and reset streak: %w", err)
	}
	return nil
}

func (r *Repository) UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, updatedTranslation, description string) error {
	query := r.queries.UpdateWordTranslationQuery(chatID, word, updatedWord, updatedTranslation, description)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update translation: %w", err)
	}
	return nil
}

func (r *Repository) ResetToReview(ctx context.Context, chatID int64) error {
	query := r.queries.ResetToReviewQuery(chatID)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build update query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("reset to review: %w", err)
	}
	return nil
}

func (r *Repository) GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error) {
	query := r.queries.GetBatchedWordTranslationsCountQuery(chatID)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count query: %w", err)
	}

	var count int
	err = r.client.QueryRowContext(ctx, sql, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get batched word translations count: %w", err)
	}
	return count, nil
}

func (r *Repository) FindWordTranslation(ctx context.Context, chatID int64, word string) (*dal.WordTranslation, error) {
	query := r.queries.FindWordTranslationQuery(chatID, word)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	row := r.client.QueryRowContext(ctx, sqlQuery, args...)
	wt, err := hydrateWordTranslation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dal.ErrNotFound
		}
		return nil, fmt.Errorf("find word translation: %w", err)
	}
	return wt, nil
}

func (r *Repository) FindRandomWordTranslation(ctx context.Context, chatID int64, filter dal.FindRandomWordFilter) (*dal.WordTranslation, error) {
	query := r.queries.FindRandomWordTranslationQuery(chatID, filter)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select query: %w", err)
	}

	row := r.client.QueryRowContext(ctx, sqlQuery, args...)
	wt, err := hydrateWordTranslation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dal.ErrNotFound
		}
		return nil, fmt.Errorf("get random word translation: %w", err)
	}
	return wt, nil
}

func (r *Repository) DeleteFromLearningBatchGtGuessedStreak(ctx context.Context, chatID int64, guessedStreakLimit int) (int, error) {
	query := r.queries.DeleteFromLearningBatchGtGuessedStreakQuery(chatID, guessedStreakLimit)

	sql, args, err := query.ToSql()
	if err != nil {
		return 0, fmt.Errorf("build delete query: %w", err)
	}

	res, err := r.client.ExecContext(ctx, sql, args...)
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

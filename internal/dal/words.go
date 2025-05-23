package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"golang.org/x/sync/errgroup"
)

const (
	LimitDirectionLessThan StreakLimitDirection = iota
	LimitDirectionGreaterThanOrEqual

	GuessedAll     Guessed = "all"
	GuessedLearned Guessed = "learned"
	GuessedBatched Guessed = "batched"
	GuessedToLearn Guessed = "to_learn"
)

var (
	ErrNotFound = errors.New("not found")
)

type (
	Guessed string

	WordTranslationsFilter struct {
		Word     string
		Guessed  Guessed
		ToReview bool
		Offset   uint64
		Limit    uint64
	}

	WordTranslationsRepository interface {
		WordTransactionsOperationsRepository
		GetStats(ctx context.Context, chatID int64) (*WordTranslationStats, error)
		FindWordTranslation(ctx context.Context, chatID int64, word string) (*WordTranslation, error)
		FindWordTranslations(ctx context.Context, chatID int64, filter WordTranslationsFilter) ([]WordTranslation, int, error)
		FindRandomWordTranslation(ctx context.Context, chatID int64, filter FindRandomWordFilter) (*WordTranslation, error)
		AddWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error
		UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, translation, description string) error
		DeleteWordTranslation(ctx context.Context, chatID int64, word string) error
	}

	WordTransactionsOperationsRepository interface {
		GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error)
		AddToLearningBatch(ctx context.Context, chatID int64, word string) error
		IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error
		ResetGuessedStreak(ctx context.Context, chatID int64, word string) error
		ResetToReview(ctx context.Context, chatID int64) error
		MarkToReview(ctx context.Context, chatID int64, word string, toReview bool) error
		DeleteFromLearningBatchGtGuessedStreak(ctx context.Context, chatID int64, guessedStreakLimit int) (int, error)
	}

	StreakLimitDirection int

	FindRandomWordFilter struct {
		Batched              bool
		StreakLimitDirection StreakLimitDirection // ignored if Batched = true
		StreakLimit          int                  // ignored if Batched = true
	}
)

func (r *PostgreSQLRepository) GetStats(ctx context.Context, chatID int64) (*WordTranslationStats, error) {
	row := r.client.QueryRow(ctx, `
SELECT 
    chat_id,
    SUM(CASE WHEN guessed_streak >= 15 THEN 1 ELSE 0 END) AS streak_15_plus,
    SUM(CASE WHEN guessed_streak BETWEEN 10 AND 14 THEN 1 ELSE 0 END) AS streak_10_to_14,
    SUM(CASE WHEN guessed_streak BETWEEN 1 AND 9 THEN 1 ELSE 0 END) AS streak_1_to_9,
    COUNT(*) AS total_words
FROM 
    word_translations
WHERE
	chat_id = $1
GROUP BY
	chat_id
`, chatID)

	var stats WordTranslationStats
	err := row.Scan(
		&stats.ChatID,
		&stats.GreaterThanOrEqual15,
		&stats.Between10And14,
		&stats.Between1And9,
		&stats.Total,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &WordTranslationStats{
				ChatID: chatID,
			}, nil
		}
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return &stats, nil
}

func (r *PostgreSQLRepository) AddWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO word_translations (chat_id, word, translation, description)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (chat_id, word) DO UPDATE SET translation = $3, description = $4
	`, chatID, word, translation, description)
	if err != nil {
		return fmt.Errorf("add translation: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) FindWordTranslations(ctx context.Context, chatID int64, filter WordTranslationsFilter) ([]WordTranslation, int, error) {
	// Base query builder
	baseQuery := squirrel.Select().
		From("word_translations").
		Where(squirrel.Eq{"chat_id": chatID}).
		PlaceholderFormat(squirrel.Dollar)

	// Apply filters
	if filter.Word != "" {
		baseQuery = baseQuery.Where("LOWER(word) SIMILAR TO ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Word)))
	}

	if filter.ToReview {
		baseQuery = baseQuery.Where(squirrel.Eq{"to_review": filter.ToReview})
	}

	switch filter.Guessed {
	case "", GuessedAll:
	case GuessedLearned:
		baseQuery = baseQuery.Where("guessed_streak >= 15")
	case GuessedBatched:
		baseQuery = baseQuery.Where("guessed_streak < 15")
	case GuessedToLearn:
		baseQuery = baseQuery.Where("guessed_streak = 0")
	default:
		return nil, 0, fmt.Errorf("invalid guessed filter: %s", filter.Guessed)
	}

	eg, ctx := errgroup.WithContext(ctx)
	res := make([]WordTranslation, 0, filter.Limit)
	total := 0

	eg.Go(func() error {
		// Build select query
		selectQuery := baseQuery.
			Columns("chat_id", "word", "translation", "COALESCE(description, '')", "guessed_streak", "to_review", "created_at", "updated_at").
			OrderBy("word").
			Offset(filter.Offset).
			Limit(filter.Limit)

		query, args, err := selectQuery.ToSql()
		if err != nil {
			return fmt.Errorf("build select query: %w", err)
		}

		rows, err := r.client.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("find translations: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			wt, err := hydrateWordTranslation(rows) //nolint:govet // ignore shadow declaration
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
		// Build count query
		countQuery := baseQuery.
			Columns("COUNT(*)")

		query, args, err := countQuery.ToSql()
		if err != nil {
			return fmt.Errorf("build count query: %w", err)
		}

		if err := r.client.QueryRow(ctx, query, args...).Scan(&total); err != nil { //nolint:govet // ignore shadow declaration
			return fmt.Errorf("get total: %w", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}

	return res, total, nil
}
func (r *PostgreSQLRepository) DeleteWordTranslation(ctx context.Context, chatID int64, word string) error {
	_, err := r.client.Exec(ctx, `
		DELETE FROM word_translations
		WHERE chat_id = $1 AND word = $2
	`, chatID, word)
	if err != nil {
		return fmt.Errorf("delete translation: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) AddToLearningBatch(ctx context.Context, chatID int64, word string) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO learning_batches (chat_id, word)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, chatID, word)
	if err != nil {
		return fmt.Errorf("add to learning batch: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error {
	_, err := r.client.Exec(ctx, `
		UPDATE word_translations
		SET guessed_streak = guessed_streak + 1
		WHERE chat_id = $1 AND word = $2
	`, chatID, word)
	if err != nil {
		return fmt.Errorf("increase guessed streak: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) ResetGuessedStreak(ctx context.Context, chatID int64, word string) error {
	_, err := r.client.Exec(ctx, `
		UPDATE word_translations
		SET guessed_streak = 0
		WHERE chat_id = $1 AND word = $2
	`, chatID, word)
	if err != nil {
		return fmt.Errorf("reset guessed streak: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) MarkToReview(ctx context.Context, chatID int64, word string, toReview bool) error {
	_, err := r.client.Exec(ctx, `
		UPDATE word_translations
		SET to_review = $1
		WHERE chat_id = $2 AND word = $3
	`, toReview, chatID, word)
	if err != nil {
		return fmt.Errorf("mark review and reset streak: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, updatedTranslation, description string) error {
	_, err := r.client.Exec(ctx, `
		UPDATE word_translations
		SET word = $3, translation = $4, description = $5
		WHERE chat_id = $1 AND word = $2
	`, chatID, word, updatedWord, updatedTranslation, description)
	if err != nil {
		return fmt.Errorf("update translation: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) ResetToReview(ctx context.Context, chatID int64) error {
	_, err := r.client.Exec(ctx, `
		UPDATE word_translations
		SET to_review = false
		WHERE chat_id = $1
	`, chatID)
	if err != nil {
		return fmt.Errorf("reset to review: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error) {
	row := r.client.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM word_translations wt
		INNER JOIN learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word
		WHERE wt.chat_id = $1
	`, chatID)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("get batched word translations count: %w", err)
	}
	return count, nil
}

func (r *PostgreSQLRepository) FindWordTranslation(ctx context.Context, chatID int64, word string) (*WordTranslation, error) {
	row := r.client.QueryRow(ctx, `
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.to_review, wt.created_at, wt.updated_at
		FROM word_translations wt
		WHERE wt.chat_id = $1 AND wt.word = $2
	`, chatID, word)

	wt, err := hydrateWordTranslation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("find word translation: %w", err)
	}
	return wt, nil
}

func (r *PostgreSQLRepository) FindRandomWordTranslation(ctx context.Context, chatID int64, filter FindRandomWordFilter) (*WordTranslation, error) {
	var (
		query string
		args  []any
	)
	if filter.Batched {
		query = `
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.to_review, wt.created_at, wt.updated_at
		FROM word_translations wt
		INNER JOIN learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word
		WHERE wt.chat_id = $1
		ORDER BY random()
		LIMIT 1
	`
		args = []any{chatID}
	} else {
		query = fmt.Sprintf(`
		WITH batched_words AS (
			SELECT lb.word
			FROM learning_batches lb
			WHERE lb.chat_id = $1
		)
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.to_review, wt.created_at, wt.updated_at
		FROM word_translations wt
		WHERE wt.chat_id = $1 AND wt.guessed_streak %s $2 AND wt.word NOT IN (SELECT word FROM batched_words)
		ORDER BY random()
		LIMIT 1
	`, filter.StreakLimitDirection.String())
		args = []any{chatID, filter.StreakLimit}
	}

	row := r.client.QueryRow(ctx, query, args...)
	wt, err := hydrateWordTranslation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get random word translation: %w", err)
	}

	return wt, nil
}

func (r *PostgreSQLRepository) DeleteFromLearningBatchGtGuessedStreak(ctx context.Context, chatID int64, guessedStreakLimit int) (int, error) {
	res, err := r.client.Exec(ctx, `
		WITH known_words AS (
			SELECT wt.word
			FROM word_translations wt
			WHERE wt.chat_id = $1 AND wt.guessed_streak >= $2
		)
		DELETE FROM learning_batches lb
		WHERE lb.chat_id = $1 AND lb.word IN (SELECT word FROM known_words)
	`, chatID, guessedStreakLimit)

	if err != nil {
		return 0, fmt.Errorf("delete from learning batch: %w", err)
	}

	return int(res.RowsAffected()), nil
}

func (d StreakLimitDirection) String() string {
	return [...]string{"<", ">="}[d]
}

func hydrateWordTranslation(row pgx.Row) (*WordTranslation, error) {
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
	)
	if err != nil {
		return nil, fmt.Errorf("scan word translation: %w", err)
	}
	return &wt, nil
}

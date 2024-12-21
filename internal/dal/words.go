package dal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const (
	LimitDirectionLessThan StreakLimitDirection = iota
	LimitDirectionGreaterThanOrEqual
)

var (
	ErrNotFound = errors.New("not found")
)

type (
	WordTranslationsRepository interface {
		GetStats(ctx context.Context, chatID int64) (*WordTranslationStats, error)
		AddWordTranslation(ctx context.Context, chatID int64, word, translation string) error
		UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, translation, description string) error
		AddToLearningBatch(ctx context.Context, chatID int64, word string) error
		GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error)
		FindWordTranslation(ctx context.Context, chatID int64, word string) (*WordTranslation, error)
		FindRandomWordTranslation(ctx context.Context, chatID int64, filter FindRandomWordFilter) (*WordTranslation, error)
		FindWordsToReview(ctx context.Context, chatID int64) ([]WordTranslation, error)
		IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error
		ResetGuessedStreak(ctx context.Context, chatID int64, word string) error
		ResetToReview(ctx context.Context, chatID int64) error
		MarkToReviewAndResetStreak(ctx context.Context, chatID int64, word string) error
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
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return &stats, nil
}

func (r *PostgreSQLRepository) AddWordTranslation(ctx context.Context, chatID int64, word, translation string) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO word_translations (chat_id, word, translation)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, word) DO UPDATE SET translation = $3, guessed_streak = 0
	`, chatID, word, translation)
	if err != nil {
		return fmt.Errorf("add translation: %w", err)
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

func (r *PostgreSQLRepository) MarkToReviewAndResetStreak(ctx context.Context, chatID int64, word string) error {
	_, err := r.client.Exec(ctx, `
		UPDATE word_translations
		SET guessed_streak = 0, to_review = true
		WHERE chat_id = $1 AND word = $2
	`, chatID, word)
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
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.created_at, wt.updated_at
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
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.created_at, wt.updated_at
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
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.created_at, wt.updated_at
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

func (r *PostgreSQLRepository) FindWordsToReview(ctx context.Context, chatID int64) ([]WordTranslation, error) {
	rows, err := r.client.Query(ctx, `
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.created_at, wt.updated_at
		FROM word_translations wt
		WHERE wt.chat_id = $1 AND wt.to_review = true
	`, chatID)
	if err != nil {
		return nil, fmt.Errorf("get words to review: %w", err)
	}
	defer rows.Close()

	var words []WordTranslation
	for rows.Next() {
		wt, err := hydrateWordTranslation(rows) //nolint:govet // it is supposed to be used in a loop
		if err != nil {
			return nil, fmt.Errorf("scan word translation: %w", err)
		}
		words = append(words, *wt)
	}

	return words, nil
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
		&wt.CreatedAt,
		&wt.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan word translation: %w", err)
	}
	return &wt, nil
}

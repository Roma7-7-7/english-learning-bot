package dal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrNotFound = errors.New("not found")

	columns = []string{
		"chat_id",
		"word",
		"translation",
		"guessed_streak",
		"created_at",
		"updated_at",
	}
)

type (
	Client interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
		QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	}

	Repository interface {
		Transact(ctx context.Context, txFunc func(r Repository) error) error
		GetStats(ctx context.Context, chatID int64) (*WordTranslationStats, error)
		AddWordTranslation(ctx context.Context, chatID int64, word, translation string) error
		AddToLearningBatch(ctx context.Context, chatID int64, word string) error
		GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error)
		GetRandomBatchedWordTranslation(ctx context.Context, chatID int64) (*WordTranslation, error)
		GetRandomNotBatchedWordTranslation(ctx context.Context, chatID int64, streakLimit int) (*WordTranslation, error)
		IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error
		ResetGuessedStreak(ctx context.Context, chatID int64, word string) error
		DeleteFromLearningBatchGtGuessedStreak(ctx context.Context, chatID int64, guessedStreakLimit int) (int, error)
	}

	PostgresqlRepository struct {
		client Client
	}
)

func NewPostgresqlRepository(client Client) *PostgresqlRepository {
	return &PostgresqlRepository{client: client}
}

func (r *PostgresqlRepository) Transact(ctx context.Context, txFunc func(r Repository) error) error {
	tx, err := r.client.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := txFunc(NewPostgresqlRepository(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresqlRepository) GetStats(ctx context.Context, chatID int64) (*WordTranslationStats, error) {
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

func (r *PostgresqlRepository) AddWordTranslation(ctx context.Context, chatID int64, word, translation string) error {
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

func (r *PostgresqlRepository) AddToLearningBatch(ctx context.Context, chatID int64, word string) error {
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

func (r *PostgresqlRepository) IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error {
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

func (r *PostgresqlRepository) ResetGuessedStreak(ctx context.Context, chatID int64, word string) error {
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

func (r *PostgresqlRepository) GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error) {
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

func (r *PostgresqlRepository) GetRandomBatchedWordTranslation(ctx context.Context, chatID int64) (*WordTranslation, error) {
	row := r.client.QueryRow(ctx, `
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.created_at, wt.updated_at
		FROM word_translations wt
		INNER JOIN learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word
		WHERE wt.chat_id = $1
		ORDER BY random()
		LIMIT 1
	`, chatID)

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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get random word translation: %w", err)
	}
	return &wt, nil
}

func (r *PostgresqlRepository) GetRandomNotBatchedWordTranslation(ctx context.Context, chatID int64, streakLimit int) (*WordTranslation, error) {
	row := r.client.QueryRow(ctx, `
		WITH batched_words AS (
			SELECT lb.word
			FROM learning_batches lb
			WHERE lb.chat_id = $1
		)
		SELECT wt.chat_id, wt.word, wt.translation, COALESCE(wt.description, ''), wt.guessed_streak, wt.created_at, wt.updated_at
		FROM word_translations wt
		WHERE wt.chat_id = $1 AND wt.guessed_streak < $2 AND wt.word NOT IN (SELECT word FROM batched_words)
		ORDER BY random()
		LIMIT 1
	`, chatID, streakLimit)

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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get random word translation: %w", err)
	}
	return &wt, nil
}

func (r *PostgresqlRepository) DeleteFromLearningBatchGtGuessedStreak(ctx context.Context, chatID int64, guessedStreakLimit int) (int, error) {
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

package dal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

type PostgresqlRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresqlRepository(ctx context.Context, dbURL string) (*PostgresqlRepository, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return &PostgresqlRepository{pool: pool}, nil
}

func (r *PostgresqlRepository) AddWordTranslation(ctx context.Context, chatID int64, word, translation string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO word_translations (chat_id, word, translation)
		VALUES ($1, $2, $3)
		ON CONFLICT (chat_id, word) DO UPDATE SET translation = $3, guessed_streak = 0
	`, chatID, word, translation)
	if err != nil {
		return fmt.Errorf("add translation: %w", err)
	}
	return nil
}

func (r *PostgresqlRepository) IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error {
	_, err := r.pool.Exec(ctx, `
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
	_, err := r.pool.Exec(ctx, `
		UPDATE word_translations
		SET guessed_streak = 0
		WHERE chat_id = $1 AND word = $2
	`, chatID, word)
	if err != nil {
		return fmt.Errorf("reset guessed streak: %w", err)
	}

	return nil
}

func (r *PostgresqlRepository) GetRandomWordTranslation(ctx context.Context, chatID int64, streakLimit int) (*WordTranslation, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT chat_id, word, translation, guessed_streak, created_at, updated_at
		FROM word_translations
		WHERE chat_id = $1 AND guessed_streak < $2
		ORDER BY random()
		LIMIT 1
	`, chatID, streakLimit)

	var wt WordTranslation
	err := row.Scan(
		&wt.ChatID,
		&wt.Word,
		&wt.Translation,
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

func (r *PostgresqlRepository) Close() {
	r.pool.Close()
}

package dal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type (
	AuthConfirmation struct {
		ChatID    int
		Token     string
		ExpiresAt time.Time
		Confirmed bool
	}

	AuthConfirmationRepository interface {
		InsertAuthConfirmation(ctx context.Context, chatID int64, token string, expiresIn time.Duration) error
		IsConfirmed(ctx context.Context, chatID int64, token string) (bool, error)
		ConfirmAuthConfirmation(ctx context.Context, chatID int64, token string) error
		DeleteAuthConfirmation(ctx context.Context, chatID int64, token string) error
	}
)

func (r *PostgreSQLRepository) InsertAuthConfirmation(ctx context.Context, chatID int64, token string, expiresIn time.Duration) error {
	if chatID == 0 {
		return errors.New("chat id is required")
	}
	if expiresIn <= 0 {
		return errors.New("expires in is required")
	}

	_, err := r.client.Exec(ctx, `
		INSERT INTO auth_confirmations(chat_id, token, expires_at)
		VALUES ($1, $2, $3)
	`, chatID, token, time.Now().Add(expiresIn))
	if err != nil {
		return fmt.Errorf("insert auth confirmation: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) IsConfirmed(ctx context.Context, chatID int64, token string) (bool, error) {
	var confirmed bool
	err := r.client.QueryRow(ctx, `
			SELECT confirmed
			FROM auth_confirmations
			WHERE chat_id = $1 AND token = $2 AND expires_at > NOW()
	`, chatID, token).Scan(&confirmed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrNotFound
		}
		return false, fmt.Errorf("is confirmed: %w", err)
	}

	return confirmed, nil
}

func (r *PostgreSQLRepository) ConfirmAuthConfirmation(ctx context.Context, chatID int64, token string) error {
	_, err := r.client.Exec(ctx, `
		UPDATE auth_confirmations
		SET confirmed = true
		WHERE chat_id = $1 AND token = $2 AND expires_at > NOW()
	`, chatID, token)
	if err != nil {
		return fmt.Errorf("confirm auth confirmation: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) DeleteAuthConfirmation(ctx context.Context, chatID int64, token string) error {
	_, err := r.client.Exec(ctx, `
		DELETE FROM auth_confirmations
		WHERE chat_id = $1 AND token = $2
	`, chatID, token)
	if err != nil {
		return fmt.Errorf("delete auth confirmation: %w", err)
	}

	return nil
}

func (r *PostgreSQLRepository) cleanupAuthConfirmations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
			_, err := r.client.Exec(ctx, `
				DELETE FROM auth_confirmations
				WHERE expires_at < NOW()
			`)
			if err != nil {
				r.log.ErrorContext(ctx, "failed to cleanup auth confirmations", "error", err)
			}
		}
	}
}

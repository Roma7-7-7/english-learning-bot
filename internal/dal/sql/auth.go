package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func (r *Repository) InsertAuthConfirmation(ctx context.Context, chatID int64, token string, expiresIn time.Duration) error {
	if chatID == 0 {
		return errors.New("chat id is required")
	}
	if expiresIn <= 0 {
		return errors.New("expires in is required")
	}

	query := r.queries.InsertAuthConfirmationQuery(chatID, token, time.Now().Add(expiresIn))

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("insert auth confirmation: %w", err)
	}

	return nil
}

func (r *Repository) IsConfirmed(ctx context.Context, chatID int64, token string) (bool, error) {
	query := r.queries.IsConfirmedQuery(chatID, token)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return false, fmt.Errorf("build query: %w", err)
	}

	var confirmed bool
	err = r.client.QueryRowContext(ctx, sqlQuery, args...).Scan(&confirmed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, dal.ErrNotFound
		}
		return false, fmt.Errorf("is confirmed: %w", err)
	}

	return confirmed, nil
}

func (r *Repository) ConfirmAuthConfirmation(ctx context.Context, chatID int64, token string) error {
	query := r.queries.ConfirmAuthConfirmationQuery(chatID, token)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("confirm auth confirmation: %w", err)
	}

	return nil
}

func (r *Repository) DeleteAuthConfirmation(ctx context.Context, chatID int64, token string) error {
	query := r.queries.DeleteAuthConfirmationQuery(chatID, token)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("delete auth confirmation: %w", err)
	}

	return nil
}

func (r *Repository) cleanupAuthConfirmations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
			query := r.queries.CleanupAuthConfirmationsQuery()

			sql, args, err := query.ToSql()
			if err != nil {
				r.log.ErrorContext(ctx, "failed to build cleanup query", "error", err)
				continue
			}

			_, err = r.client.ExecContext(ctx, sql, args...)
			if err != nil {
				r.log.ErrorContext(ctx, "failed to cleanup auth confirmations", "error", err)
			}
		}
	}
}

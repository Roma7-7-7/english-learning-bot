package dal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type CallbacksRepository interface {
	InsertCallback(ctx context.Context, data CallbackData) (string, error)
	FindCallback(ctx context.Context, chatID int64, uuid string) (*CallbackData, error)
}

func (r *PostgreSQLRepository) InsertCallback(ctx context.Context, data CallbackData) (string, error) {
	if data.ChatID == 0 {
		return "", fmt.Errorf("chat id is required")
	}
	if data.ExpiresAt.IsZero() {
		return "", fmt.Errorf("expires at is required")
	}

	row := r.client.QueryRow(ctx, `
		INSERT INTO callback_data(uuid, chat_id, data, expires_at)
		VALUES (gen_random_uuid(), $1, $2, $3)
		ON CONFLICT (uuid, chat_id) DO UPDATE SET data = $2
		RETURNING uuid
	`, data.ChatID, data, data.ExpiresAt)
	err := row.Scan(&data.ID)
	if err != nil {
		return "", fmt.Errorf("insert callback: %w", err)
	}

	return data.ID, nil
}

func (r *PostgreSQLRepository) FindCallback(ctx context.Context, chatID int64, uuid string) (*CallbackData, error) {
	var (
		data      CallbackData
		expiresAt time.Time
	)

	err := r.client.QueryRow(ctx, `
		SELECT data, expires_at
		FROM callback_data	
		WHERE chat_id = $1 AND uuid = $2 
	`, chatID, uuid).Scan(&data, &expiresAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("find callback: %w", err)
	}
	data.ChatID = chatID
	data.ID = uuid
	data.ExpiresAt = expiresAt

	return &data, nil
}

func (r *PostgreSQLRepository) cleanupJob(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
			r.log.Info("running cleanup job")
			_, err := r.client.Exec(ctx, `DELETE FROM callback_data WHERE expires_at < now()`)
			if err != nil {
				r.log.Error("failed to run cleanup job", "error", err)
			}
		}
	}
}

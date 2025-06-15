package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func (r *Repository) InsertCallback(ctx context.Context, data dal.CallbackData) (string, error) {
	if data.ChatID == 0 {
		return "", errors.New("chat id is required")
	}
	if data.ExpiresAt.IsZero() {
		return "", errors.New("expires at is required")
	}

	query, err := r.queries.InsertCallbackQuery(data.ChatID, data, data.ExpiresAt)
	if err != nil {
		return "", fmt.Errorf("build query: %w", err)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return "", fmt.Errorf("build query: %w", err)
	}

	row := r.client.QueryRowContext(ctx, sql, args...)
	err = row.Scan(&data.ID)
	if err != nil {
		return "", fmt.Errorf("insert callback: %w", err)
	}

	return data.ID, nil
}

func (r *Repository) FindCallback(ctx context.Context, chatID int64, uuid string) (*dal.CallbackData, error) {
	query := r.queries.FindCallbackQuery(chatID, uuid)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var (
		rawData   any
		expiresAt time.Time
	)

	err = r.client.QueryRowContext(ctx, sqlQuery, args...).Scan(&rawData, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dal.ErrNotFound
		}
		return nil, fmt.Errorf("find callback: %w", err)
	}

	data, err := r.queries.DeserializeCallbackData(rawData)
	if err != nil {
		return nil, fmt.Errorf("deserialize callback data: %w", err)
	}

	data.ChatID = chatID
	data.ID = uuid
	data.ExpiresAt = expiresAt

	return &data, nil
}

func (r *Repository) cleanupCallbacksJob(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
			r.log.InfoContext(ctx, "running cleanup job")

			query := r.queries.CleanupCallbacksQuery()

			sql, args, err := query.ToSql()
			if err != nil {
				r.log.ErrorContext(ctx, "failed to build cleanup query", "error", err)
				continue
			}

			_, err = r.client.ExecContext(ctx, sql, args...)
			if err != nil {
				r.log.ErrorContext(ctx, "failed to run cleanup job", "error", err)
			}
		}
	}
}

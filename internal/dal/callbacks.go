package dal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

func (r *SQLiteRepository) InsertCallback(ctx context.Context, data CallbackData) (string, error) {
	if data.ChatID == 0 {
		return "", errors.New("chat id is required")
	}
	if data.ExpiresAt.IsZero() {
		return "", errors.New("expires at is required")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal callback data: %w", err)
	}
	serializedData := string(jsonData)

	query := qb.Insert("callback_data").
		Columns("uuid", "chat_id", "data", "expires_at").
		Values(squirrel.Expr("hex(randomblob(4))"), data.ChatID, serializedData, data.ExpiresAt).
		Suffix("ON CONFLICT (uuid, chat_id) DO UPDATE SET data = EXCLUDED.data").
		Suffix("RETURNING uuid")

	sql, args, err := query.ToSql()
	if err != nil {
		return "", fmt.Errorf("build query: %w", err)
	}

	row := r.db.QueryRowContext(ctx, sql, args...)
	err = row.Scan(&data.ID)
	if err != nil {
		return "", fmt.Errorf("insert callback: %w", err)
	}

	return data.ID, nil
}

func (r *SQLiteRepository) FindCallback(ctx context.Context, chatID int64, uuid string) (*CallbackData, error) {
	query := qb.Select("data", "expires_at").
		From("callback_data").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"uuid":    uuid,
		})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var (
		rawData   any
		expiresAt time.Time
	)

	err = r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(&rawData, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find callback: %w", err)
	}

	// For SQLite, we need to deserialize from JSON string
	strData, ok := rawData.(string)
	if !ok {
		return nil, fmt.Errorf("expected string data for SQLite, got %T", rawData)
	}
	var res CallbackData
	if err := json.Unmarshal([]byte(strData), &res); err != nil {
		return nil, fmt.Errorf("unmarshal callback data: %w", err)
	}

	res.ChatID = chatID
	res.ID = uuid
	res.ExpiresAt = expiresAt

	return &res, nil
}

func (r *SQLiteRepository) cleanupCallbacksJob(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Hour):
			r.log.InfoContext(ctx, "running cleanup job")

			query := qb.Delete("callback_data").
				Where(squirrel.Expr("expires_at < " + ("datetime('now', 'localtime')")))

			sql, args, err := query.ToSql()
			if err != nil {
				r.log.ErrorContext(ctx, "failed to build cleanup query", "error", err)
				continue
			}

			_, err = r.db.ExecContext(ctx, sql, args...)
			if err != nil {
				r.log.ErrorContext(ctx, "failed to run cleanup job", "error", err)
			}
		}
	}
}

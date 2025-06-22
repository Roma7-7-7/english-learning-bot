package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func (r *Repository) GetTotalStats(ctx context.Context, chatID int64) (*dal.TotalStats, error) {
	query := r.queries.GetTotalStatsQuery(chatID)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.client.QueryRowContext(ctx, sqlQuery, args...)

	var stats dal.TotalStats
	err = row.Scan(
		&stats.ChatID,
		&stats.GreaterThanOrEqual15,
		&stats.Between10And14,
		&stats.Between1And9,
		&stats.Total,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &dal.TotalStats{
				ChatID: chatID,
			}, nil
		}
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return &stats, nil
}

func (r *Repository) GetStats(ctx context.Context, chatID int64, date time.Time) (*dal.Stats, error) {
	query := r.queries.GetStatsQuery(chatID, date)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.client.QueryRowContext(ctx, sqlQuery, args...)

	var stats dal.Stats
	var strDate string
	err = row.Scan(
		&stats.ChatID,
		&strDate,
		&stats.WordsGuessed,
		&stats.WordsMissed,
		&stats.TotalWordsLearned,
		&stats.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, dal.ErrNotFound
		}
		return nil, fmt.Errorf("get stats: %w", err)
	}
	stats.Date, err = time.Parse("2006-01-02", strDate)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	return &stats, nil
}

func (r *Repository) GetStatsRange(ctx context.Context, chatID int64, from, to time.Time) ([]dal.Stats, error) {
	query := r.queries.GetStatsRangeQuery(chatID, from, to)

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.client.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("get stats range: %w", err)
	}
	defer rows.Close()

	var stats []dal.Stats
	var dateStr string
	for rows.Next() {
		var stat dal.Stats
		err := rows.Scan(
			&stat.ChatID,
			&dateStr,
			&stat.WordsGuessed,
			&stat.WordsMissed,
			&stat.TotalWordsLearned,
			&stat.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stat.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parse date: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stats: %w", err)
	}

	return stats, nil
}

func (r *Repository) IncrementWordGuessed(ctx context.Context, chatID int64) error {
	query := r.queries.IncrementWordGuessedQuery(chatID)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("increment word guessed: %w", err)
	}
	return nil
}

func (r *Repository) IncrementWordMissed(ctx context.Context, chatID int64) error {
	query := r.queries.IncrementWordMissedQuery(chatID)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("increment word missed: %w", err)
	}
	return nil
}

func (r *Repository) UpdateTotalWordsLearned(ctx context.Context, chatID int64) error {
	query := r.queries.UpdateTotalWordsLearnedQuery(chatID)

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.client.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update total words learned: %w", err)
	}
	return nil
}

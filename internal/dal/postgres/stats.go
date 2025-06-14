package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

func (r *Repository) GetTotalStats(ctx context.Context, chatID int64) (*dal.TotalStats, error) {
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

	var stats dal.TotalStats
	err := row.Scan(
		&stats.ChatID,
		&stats.GreaterThanOrEqual15,
		&stats.Between10And14,
		&stats.Between1And9,
		&stats.Total,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &dal.TotalStats{
				ChatID: chatID,
			}, nil
		}
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return &stats, nil
}

func (r *Repository) GetStats(ctx context.Context, chatID int64, date time.Time) (*dal.Stats, error) {
	row := r.client.QueryRow(ctx, `
		SELECT 
			chat_id, date, words_guessed, words_missed, 
			total_words_learned, created_at
		FROM statistics
		WHERE chat_id = $1 AND date = $2
	`, chatID, date)

	var stats dal.Stats
	err := row.Scan(
		&stats.ChatID,
		&stats.Date,
		&stats.WordsGuessed,
		&stats.WordsMissed,
		&stats.TotalWordsLearned,
		&stats.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, dal.ErrNotFound
		}
		return nil, fmt.Errorf("get stats: %w", err)
	}
	return &stats, nil
}

func (r *Repository) GetStatsRange(ctx context.Context, chatID int64, from, to time.Time) ([]dal.Stats, error) {
	rows, err := r.client.Query(ctx, `
		SELECT 
			chat_id, date, words_guessed, words_missed, 
			total_words_learned, created_at
		FROM statistics
		WHERE chat_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date
	`, chatID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get stats range: %w", err)
	}
	defer rows.Close()

	var stats []dal.Stats
	for rows.Next() {
		var stat dal.Stats
		err := rows.Scan(
			&stat.ChatID,
			&stat.Date,
			&stat.WordsGuessed,
			&stat.WordsMissed,
			&stat.TotalWordsLearned,
			&stat.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stats: %w", err)
	}

	return stats, nil
}

func (r *Repository) IncrementWordGuessed(ctx context.Context, chatID int64) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO statistics (chat_id, date, words_guessed)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (chat_id, date) DO UPDATE 
		SET words_guessed = statistics.words_guessed + 1
	`, chatID)
	if err != nil {
		return fmt.Errorf("increment word guessed: %w", err)
	}
	return nil
}

func (r *Repository) IncrementWordMissed(ctx context.Context, chatID int64) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO statistics (chat_id, date, words_missed)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (chat_id, date) DO UPDATE 
		SET words_missed = statistics.words_missed + 1
	`, chatID)
	if err != nil {
		return fmt.Errorf("increment word missed: %w", err)
	}
	return nil
}

func (r *Repository) UpdateTotalWordsLearned(ctx context.Context, chatID int64) error {
	_, err := r.client.Exec(ctx, `
		UPDATE statistics
		SET total_words_learned = (
			SELECT COUNT(*)
			FROM word_translations
			WHERE chat_id = $1 AND guessed_streak >= 15
		)
		WHERE chat_id = $1 AND date = CURRENT_DATE
	`, chatID)
	if err != nil {
		return fmt.Errorf("update total words learned: %w", err)
	}
	return nil
}

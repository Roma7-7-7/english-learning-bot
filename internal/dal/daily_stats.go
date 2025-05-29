package dal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgreSQLRepository) IncrementWordGuessed(ctx context.Context, chatID int64) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO daily_word_statistics (chat_id, date, words_guessed, total_words_guessed)
		VALUES ($1, CURRENT_DATE, 1, (
			SELECT COALESCE(MAX(total_words_guessed), 0) + 1 
			FROM daily_word_statistics 
			WHERE chat_id = $1
		))
		ON CONFLICT (chat_id, date) DO UPDATE 
		SET words_guessed = daily_word_statistics.words_guessed + 1,
			total_words_guessed = daily_word_statistics.total_words_guessed + 1
	`, chatID)
	if err != nil {
		return fmt.Errorf("increment word guessed: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) IncrementWordMissed(ctx context.Context, chatID int64) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO daily_word_statistics (chat_id, date, words_missed)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (chat_id, date) DO UPDATE 
		SET words_missed = daily_word_statistics.words_missed + 1
	`, chatID)
	if err != nil {
		return fmt.Errorf("increment word missed: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) IncrementWordToReview(ctx context.Context, chatID int64) error {
	_, err := r.client.Exec(ctx, `
		INSERT INTO daily_word_statistics (chat_id, date, words_to_review)
		VALUES ($1, CURRENT_DATE, 1)
		ON CONFLICT (chat_id, date) DO UPDATE 
		SET words_to_review = daily_word_statistics.words_to_review + 1
	`, chatID)
	if err != nil {
		return fmt.Errorf("increment word to review: %w", err)
	}
	return nil
}

func (r *PostgreSQLRepository) GetDailyStats(ctx context.Context, chatID int64, date time.Time) (*DailyWordStats, error) {
	row := r.client.QueryRow(ctx, `
		SELECT 
			chat_id, date, words_guessed, words_missed, words_to_review, 
			total_words_guessed, avg_guesses_to_success, longest_streak, created_at
		FROM daily_word_statistics
		WHERE chat_id = $1 AND date = $2
	`, chatID, date)

	var stats DailyWordStats
	err := row.Scan(
		&stats.ChatID,
		&stats.Date,
		&stats.WordsGuessed,
		&stats.WordsMissed,
		&stats.WordsToReview,
		&stats.TotalWordsGuessed,
		&stats.AvgGuessesToSuccess,
		&stats.LongestStreak,
		&stats.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get daily stats: %w", err)
	}
	return &stats, nil
}

func (r *PostgreSQLRepository) GetStatsRange(ctx context.Context, chatID int64, from, to time.Time) ([]DailyWordStats, error) {
	rows, err := r.client.Query(ctx, `
		SELECT 
			chat_id, date, words_guessed, words_missed, words_to_review, 
			total_words_guessed, avg_guesses_to_success, longest_streak, created_at
		FROM daily_word_statistics
		WHERE chat_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date
	`, chatID, from, to)
	if err != nil {
		return nil, fmt.Errorf("get stats range: %w", err)
	}
	defer rows.Close()

	var stats []DailyWordStats
	for rows.Next() {
		var stat DailyWordStats
		err := rows.Scan(
			&stat.ChatID,
			&stat.Date,
			&stat.WordsGuessed,
			&stat.WordsMissed,
			&stat.WordsToReview,
			&stat.TotalWordsGuessed,
			&stat.AvgGuessesToSuccess,
			&stat.LongestStreak,
			&stat.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan daily stats: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily stats: %w", err)
	}

	return stats, nil
}

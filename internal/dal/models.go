package dal

import (
	"context"
	"time"
)

type (
	WordTranslationStats struct {
		ChatID               int64
		GreaterThanOrEqual15 int
		Between10And14       int
		Between1And9         int
		Total                int
	}

	WordTranslation struct {
		ChatID        int64
		Word          string
		Translation   string
		Description   string
		GuessedStreak int
		ToReview      bool
		CreatedAt     time.Time
		UpdatedAt     time.Time
	}

	Cache struct {
		Key       string
		Value     string
		ExpiresAt time.Time
	}

	CallbackData struct {
		ChatID    int64     `json:"-"`
		ID        string    `json:"-"`
		Word      string    `json:"word"`
		ExpiresAt time.Time `json:"-"`
	}

	DailyWordStats struct {
		ChatID              int64
		Date                time.Time
		WordsGuessed        int
		WordsMissed         int
		WordsToReview       int
		TotalWordsGuessed   int
		AvgGuessesToSuccess float64
		LongestStreak       int
		CreatedAt           time.Time
	}

	DailyStatsRepository interface {
		IncrementWordGuessed(ctx context.Context, chatID int64) error
		IncrementWordMissed(ctx context.Context, chatID int64) error
		IncrementWordToReview(ctx context.Context, chatID int64) error
		GetDailyStats(ctx context.Context, chatID int64, date time.Time) (*DailyWordStats, error)
		GetStatsRange(ctx context.Context, chatID int64, from, to time.Time) ([]DailyWordStats, error)
	}
)

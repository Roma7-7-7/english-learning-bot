package dal

import "time"

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
)

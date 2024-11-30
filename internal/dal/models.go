package dal

import "time"

type (
	WordTranslationStats struct {
		ChatID               int64
		GreaterThanOrEqual15 int
		Between10And14       int
		Between1An9          int
		Total                int
	}

	WordTranslation struct {
		ChatID        int64
		Word          string
		Translation   string
		Description   string
		GuessedStreak int
		CreatedAt     time.Time
		UpdatedAt     time.Time
	}
)

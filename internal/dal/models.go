package dal

import "time"

type WordTranslation struct {
	ChatID        int64
	Word          string
	Translation   string
	Description   string
	GuessedStreak int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

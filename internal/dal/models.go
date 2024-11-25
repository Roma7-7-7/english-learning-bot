package dal

import "time"

type WordTranslation struct {
	ChatID        int64
	Word          string
	Translation   string
	GuessedStreak int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

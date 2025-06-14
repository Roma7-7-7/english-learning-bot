package dal

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
)

type (
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

	Stats struct {
		ChatID            int64
		Date              time.Time
		WordsGuessed      int
		WordsMissed       int
		TotalWordsLearned int
		CreatedAt         time.Time
	}

	AuthConfirmation struct {
		ChatID    int
		Token     string
		ExpiresAt time.Time
		Confirmed bool
	}

	CallbackData struct {
		ChatID    int64     `json:"-"`
		ID        string    `json:"-"`
		Word      string    `json:"word"`
		ExpiresAt time.Time `json:"-"`
	}
)

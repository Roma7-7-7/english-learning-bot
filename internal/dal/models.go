package dal

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned by inserts that refuse to overwrite what is already stored.
	ErrAlreadyExists = errors.New("already exists")
)

type (
	WordTranslation struct {
		ChatID        int64
		Word          string
		Translation   string
		Description   string
		GuessedStreak int
		ToReview      bool
		// InBatch reports whether the word has already requested batch membership: it is either in
		// the active learning batch (one of the words being asked about right now) or waiting in
		// learning_batch_queue behind it. Either way, requesting membership again is a no-op.
		InBatch   bool
		CreatedAt time.Time
		UpdatedAt time.Time
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

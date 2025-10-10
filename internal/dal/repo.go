package dal

import (
	"context"
	"time"
)

const (
	LimitDirectionLessThan StreakLimitDirection = iota
	LimitDirectionGreaterThanOrEqual

	GuessedAll     Guessed = "all"
	GuessedLearned Guessed = "learned"
	GuessedBatched Guessed = "batched"
	GuessedToLearn Guessed = "to_learn"
)

type (
	Guessed              string
	StreakLimitDirection int

	WordTranslationsFilter struct {
		Word     string
		Guessed  Guessed
		ToReview bool
		Offset   uint64
		Limit    uint64
	}

	FindRandomWordFilter struct {
		Batched              bool
		StreakLimitDirection StreakLimitDirection // ignored if Batched = true
		StreakLimit          int                  // ignored if Batched = true
	}

	TotalStats struct {
		ChatID               int64
		GreaterThanOrEqual15 int
		Between10And14       int
		Between1And9         int
		Total                int
	}

	WordTranslationsRepository interface {
		WordTransactionsOperationsRepository
		FindWordTranslation(ctx context.Context, chatID int64, word string) (*WordTranslation, error)
		FindWordTranslations(ctx context.Context, chatID int64, filter WordTranslationsFilter) ([]WordTranslation, int, error)
		FindRandomWordTranslation(ctx context.Context, chatID int64, filter FindRandomWordFilter) (*WordTranslation, error)
		AddWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error
		UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, translation, description string) error
		DeleteWordTranslation(ctx context.Context, chatID int64, word string) error
	}

	WordTransactionsOperationsRepository interface {
		GetBatchedWordTranslationsCount(ctx context.Context, chatID int64) (int, error)
		AddToLearningBatch(ctx context.Context, chatID int64, word string) error
		IncreaseGuessedStreak(ctx context.Context, chatID int64, word string) error
		ResetGuessedStreak(ctx context.Context, chatID int64, word string) error
		MarkToReview(ctx context.Context, chatID int64, word string, toReview bool) error
		DeleteFromLearningBatchGtGuessedStreak(ctx context.Context, chatID int64, guessedStreakLimit int) (int, error)
	}

	StatsRepository interface {
		GetTotalStats(ctx context.Context, chatID int64) (*TotalStats, error)
		GetStats(ctx context.Context, chatID int64, date time.Time) (*Stats, error)
		GetStatsRange(ctx context.Context, chatID int64, from, to time.Time) ([]Stats, error)
		IncrementWordGuessed(ctx context.Context, chatID int64) error
		IncrementWordMissed(ctx context.Context, chatID int64) error
		UpdateTotalWordsLearned(ctx context.Context, chatID int64) error
	}

	AuthConfirmationRepository interface {
		InsertAuthConfirmation(ctx context.Context, chatID int64, token string, expiresIn time.Duration) error
		IsConfirmed(ctx context.Context, chatID int64, token string) (bool, error)
		ConfirmAuthConfirmation(ctx context.Context, chatID int64, token string) error
		DeleteAuthConfirmation(ctx context.Context, chatID int64, token string) error
	}

	CallbacksRepository interface {
		InsertCallback(ctx context.Context, data CallbackData) (string, error)
		FindCallback(ctx context.Context, chatID int64, uuid string) (*CallbackData, error)
	}

	Repository interface {
		Transact(ctx context.Context, txFunc func(r Repository) error) error
		WordTranslationsRepository
		CallbacksRepository
		AuthConfirmationRepository
		StatsRepository
	}
)

func (d StreakLimitDirection) String() string {
	return [...]string{"<", ">="}[d]
}

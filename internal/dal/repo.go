package dal

import (
	"context"
	"time"
)

const (
	LimitDirectionLessThan StreakLimitDirection = iota
	LimitDirectionGreaterThanOrEqual
)

const (
	// OrderRandom picks any matching word with equal probability. It is the zero value, so an
	// unset Order keeps the previous behaviour.
	OrderRandom RandomOrder = iota
	// OrderLeastRecentlyReviewed picks the word whose last review is furthest in the past, so that
	// repeated picks rotate through the whole set before revisiting anything.
	OrderLeastRecentlyReviewed
)

const (
	GuessedAll     Guessed = "all"
	GuessedLearned Guessed = "learned"
	GuessedBatched Guessed = "batched"
	GuessedToLearn Guessed = "to_learn"
)

type (
	Guessed              string
	StreakLimitDirection int
	RandomOrder          int

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
		Order                RandomOrder          // ignored if Batched = true
	}

	TotalStats struct {
		ChatID int64
		// Learned counts words at or above StreakLimit, Nearly those in [NearlyFrom, StreakLimit),
		// Early those in [1, NearlyFrom).
		Learned int
		Nearly  int
		Early   int
		Total   int
		// StreakLimit and NearlyFrom are echoed back so that callers can label the buckets without
		// hardcoding the configured threshold.
		StreakLimit int
		NearlyFrom  int
	}

	WordTranslationsRepository interface {
		LearningRepository
		FindWordTranslation(ctx context.Context, chatID int64, word string) (*WordTranslation, error)
		FindWordTranslations(ctx context.Context, chatID int64, filter WordTranslationsFilter) ([]WordTranslation, int, error)
		FindRandomWordTranslation(ctx context.Context, chatID int64, filter FindRandomWordFilter) (*WordTranslation, error)
		AddWordTranslation(ctx context.Context, chatID int64, word, translation, description string) error
		UpdateWordTranslation(ctx context.Context, chatID int64, word, updatedWord, translation, description string) error
		DeleteWordTranslation(ctx context.Context, chatID int64, word string) error
	}

	// LearningRepository exposes learning progress as whole operations rather than as the individual
	// statements they are made of. Anything that has to touch more than one table runs in a single
	// transaction owned by the implementation, so callers cannot compose a half-applied update.
	LearningRepository interface {
		RegisterGuess(ctx context.Context, chatID int64, word string) error
		RegisterMiss(ctx context.Context, chatID int64, word string) error
		MarkToReview(ctx context.Context, chatID int64, word string, toReview bool) error
		MarkWordReviewed(ctx context.Context, chatID int64, word string) error
		RefillLearningBatch(ctx context.Context, chatID int64, batchSize, guessedStreakLimit int) (evicted, added int, err error)
	}

	StatsRepository interface {
		GetTotalStats(ctx context.Context, chatID int64) (*TotalStats, error)
		GetStats(ctx context.Context, chatID int64, date time.Time) (*Stats, error)
		GetStatsRange(ctx context.Context, chatID int64, from, to time.Time) ([]Stats, error)
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
		WordTranslationsRepository
		CallbacksRepository
		AuthConfirmationRepository
		StatsRepository
	}
)

func (d StreakLimitDirection) String() string {
	return [...]string{"<", ">="}[d]
}

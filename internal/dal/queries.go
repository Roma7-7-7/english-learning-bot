package dal

import (
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
)

// AddWordTranslationQuery builds a query to add or update a word translation
func AddWordTranslationQuery(chatID int64, word, translation, description string) squirrel.Sqlizer {
	return squirrel.Insert("word_translations").
		Columns("chat_id", "word", "translation", "description").
		Values(chatID, word, translation, description).
		Suffix("ON CONFLICT (chat_id, word) DO UPDATE SET translation = EXCLUDED.translation, description = EXCLUDED.description").
		PlaceholderFormat(squirrel.Dollar)
}

// FindWordTranslationsQuery builds a query to find word translations with filters
func FindWordTranslationsQuery(chatID int64, filter WordTranslationsFilter) (selectQuery, countQuery squirrel.Sqlizer) {
	baseQuery := squirrel.Select().
		From("word_translations").
		Where(squirrel.Eq{"chat_id": chatID}).
		PlaceholderFormat(squirrel.Dollar)

	if filter.Word != "" {
		baseQuery = baseQuery.Where("LOWER(word) SIMILAR TO ?", fmt.Sprintf("%%%s%%", strings.ToLower(filter.Word)))
	}

	if filter.ToReview {
		baseQuery = baseQuery.Where(squirrel.Eq{"to_review": filter.ToReview})
	}

	switch filter.Guessed {
	case "", GuessedAll:
	case GuessedLearned:
		baseQuery = baseQuery.Where("guessed_streak >= 15")
	case GuessedBatched:
		baseQuery = baseQuery.Where("guessed_streak < 15")
	case GuessedToLearn:
		baseQuery = baseQuery.Where("guessed_streak = 0")
	}

	selectQuery = baseQuery.
		Columns("chat_id", "word", "translation", "COALESCE(description, '')", "guessed_streak", "to_review", "created_at", "updated_at").
		OrderBy("word").
		Offset(filter.Offset).
		Limit(filter.Limit)

	countQuery = baseQuery.Columns("COUNT(*)")

	return selectQuery, countQuery
}

// DeleteWordTranslationQuery builds a query to delete a word translation
func DeleteWordTranslationQuery(chatID int64, word string) squirrel.Sqlizer {
	return squirrel.Delete("word_translations").
		Where(squirrel.Eq{"chat_id": chatID, "word": word}).
		PlaceholderFormat(squirrel.Dollar)
}

// AddToLearningBatchQuery builds a query to add a word to learning batch
func AddToLearningBatchQuery(chatID int64, word string) squirrel.Sqlizer {
	return squirrel.Insert("learning_batches").
		Columns("chat_id", "word").
		Values(chatID, word).
		Suffix("ON CONFLICT DO NOTHING").
		PlaceholderFormat(squirrel.Dollar)
}

// IncreaseGuessedStreakQuery builds a query to increase guessed streak
func IncreaseGuessedStreakQuery(chatID int64, word string) squirrel.Sqlizer {
	return squirrel.Update("word_translations").
		Set("guessed_streak", squirrel.Expr("guessed_streak + 1")).
		Where(squirrel.Eq{"chat_id": chatID, "word": word}).
		PlaceholderFormat(squirrel.Dollar)
}

// ResetGuessedStreakQuery builds a query to reset guessed streak
func ResetGuessedStreakQuery(chatID int64, word string) squirrel.Sqlizer {
	return squirrel.Update("word_translations").
		Set("guessed_streak", 0).
		Where(squirrel.Eq{"chat_id": chatID, "word": word}).
		PlaceholderFormat(squirrel.Dollar)
}

// MarkToReviewQuery builds a query to mark a word for review
func MarkToReviewQuery(chatID int64, word string, toReview bool) squirrel.Sqlizer {
	return squirrel.Update("word_translations").
		Set("to_review", toReview).
		Where(squirrel.Eq{"chat_id": chatID, "word": word}).
		PlaceholderFormat(squirrel.Dollar)
}

// UpdateWordTranslationQuery builds a query to update a word translation
func UpdateWordTranslationQuery(chatID int64, word, updatedWord, updatedTranslation, description string) squirrel.Sqlizer {
	return squirrel.Update("word_translations").
		Set("word", updatedWord).
		Set("translation", updatedTranslation).
		Set("description", description).
		Where(squirrel.Eq{"chat_id": chatID, "word": word}).
		PlaceholderFormat(squirrel.Dollar)
}

// ResetToReviewQuery builds a query to reset all words to not review
func ResetToReviewQuery(chatID int64) squirrel.Sqlizer {
	return squirrel.Update("word_translations").
		Set("to_review", false).
		Where(squirrel.Eq{"chat_id": chatID}).
		PlaceholderFormat(squirrel.Dollar)
}

// GetBatchedWordTranslationsCountQuery builds a query to get count of batched words
func GetBatchedWordTranslationsCountQuery(chatID int64) squirrel.Sqlizer {
	return squirrel.Select("COUNT(*)").
		From("word_translations wt").
		Join("learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word").
		Where(squirrel.Eq{"wt.chat_id": chatID}).
		PlaceholderFormat(squirrel.Dollar)
}

// FindWordTranslationQuery builds a query to find a specific word translation
func FindWordTranslationQuery(chatID int64, word string) squirrel.Sqlizer {
	return squirrel.Select(
		"wt.chat_id", "wt.word", "wt.translation",
		"COALESCE(wt.description, '')", "wt.guessed_streak",
		"wt.to_review", "wt.created_at", "wt.updated_at",
	).
		From("word_translations wt").
		Where(squirrel.Eq{"wt.chat_id": chatID, "wt.word": word}).
		PlaceholderFormat(squirrel.Dollar)
}

// FindRandomWordTranslationQuery builds a query to find a random word translation
func FindRandomWordTranslationQuery(chatID int64, filter FindRandomWordFilter) squirrel.Sqlizer {
	var query squirrel.SelectBuilder

	if filter.Batched {
		query = squirrel.Select(
			"wt.chat_id", "wt.word", "wt.translation",
			"COALESCE(wt.description, '')", "wt.guessed_streak",
			"wt.to_review", "wt.created_at", "wt.updated_at",
		).
			From("word_translations wt").
			Join("learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word").
			Where(squirrel.Eq{"wt.chat_id": chatID}).
			OrderBy("random()").
			Limit(1)
	} else {
		query = squirrel.Select(
			"wt.chat_id", "wt.word", "wt.translation",
			"COALESCE(wt.description, '')", "wt.guessed_streak",
			"wt.to_review", "wt.created_at", "wt.updated_at",
		).
			From("word_translations wt").
			Where(squirrel.Eq{"wt.chat_id": chatID}).
			Where(squirrel.Expr("wt.guessed_streak "+filter.StreakLimitDirection.String()+" ?", filter.StreakLimit)).
			Where("wt.word NOT IN (SELECT word FROM learning_batches WHERE chat_id = ?)", chatID).
			OrderBy("random()").
			Limit(1)
	}

	return query.PlaceholderFormat(squirrel.Dollar)
}

// DeleteFromLearningBatchGtGuessedStreakQuery builds a query to delete words from learning batch
func DeleteFromLearningBatchGtGuessedStreakQuery(chatID int64, guessedStreakLimit int) squirrel.Sqlizer {
	return squirrel.Delete("learning_batches lb").
		Where("lb.chat_id = ? AND lb.word IN (SELECT word FROM word_translations WHERE chat_id = ? AND guessed_streak >= ?)",
			chatID, chatID, guessedStreakLimit).
		PlaceholderFormat(squirrel.Dollar)
}

// GetTotalStatsQuery builds a query to get total statistics
func GetTotalStatsQuery(chatID int64) squirrel.Sqlizer {
	return squirrel.Select(
		"chat_id",
		"SUM(CASE WHEN guessed_streak >= 15 THEN 1 ELSE 0 END) AS streak_15_plus",
		"SUM(CASE WHEN guessed_streak BETWEEN 10 AND 14 THEN 1 ELSE 0 END) AS streak_10_to_14",
		"SUM(CASE WHEN guessed_streak BETWEEN 1 AND 9 THEN 1 ELSE 0 END) AS streak_1_to_9",
		"COUNT(*) AS total_words",
	).
		From("word_translations").
		Where(squirrel.Eq{"chat_id": chatID}).
		GroupBy("chat_id").
		PlaceholderFormat(squirrel.Dollar)
}

// GetStatsQuery builds a query to get statistics for a specific date
func GetStatsQuery(chatID int64, date time.Time) squirrel.Sqlizer {
	return squirrel.Select(
		"chat_id", "date", "words_guessed", "words_missed",
		"total_words_learned", "created_at",
	).
		From("statistics").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"date":    date,
		}).
		PlaceholderFormat(squirrel.Dollar)
}

// GetStatsRangeQuery builds a query to get statistics for a date range
func GetStatsRangeQuery(chatID int64, from, to time.Time) squirrel.Sqlizer {
	return squirrel.Select(
		"chat_id", "date", "words_guessed", "words_missed",
		"total_words_learned", "created_at",
	).
		From("statistics").
		Where(squirrel.Eq{"chat_id": chatID}).
		Where(squirrel.Expr("date BETWEEN ? AND ?", from, to)).
		OrderBy("date").
		PlaceholderFormat(squirrel.Dollar)
}

// IncrementWordGuessedQuery builds a query to increment words guessed count
func IncrementWordGuessedQuery(chatID int64) squirrel.Sqlizer {
	return squirrel.Insert("statistics").
		Columns("chat_id", "date", "words_guessed").
		Values(chatID, squirrel.Expr("CURRENT_DATE"), 1).
		Suffix("ON CONFLICT (chat_id, date) DO UPDATE SET words_guessed = statistics.words_guessed + 1").
		PlaceholderFormat(squirrel.Dollar)
}

// IncrementWordMissedQuery builds a query to increment words missed count
func IncrementWordMissedQuery(chatID int64) squirrel.Sqlizer {
	return squirrel.Insert("statistics").
		Columns("chat_id", "date", "words_missed").
		Values(chatID, squirrel.Expr("CURRENT_DATE"), 1).
		Suffix("ON CONFLICT (chat_id, date) DO UPDATE SET words_missed = statistics.words_missed + 1").
		PlaceholderFormat(squirrel.Dollar)
}

// UpdateTotalWordsLearnedQuery builds a query to update total words learned count
func UpdateTotalWordsLearnedQuery(chatID int64) squirrel.Sqlizer {
	return squirrel.Update("statistics").
		Set("total_words_learned", squirrel.Select("COUNT(*)").
			From("word_translations").
			Where(squirrel.Eq{"chat_id": chatID}).
			Where("guessed_streak >= 15")).
		Where(squirrel.Eq{
			"chat_id": chatID,
			"date":    squirrel.Expr("CURRENT_DATE"),
		}).
		PlaceholderFormat(squirrel.Dollar)
}

// InsertAuthConfirmationQuery builds a query to insert a new auth confirmation
func InsertAuthConfirmationQuery(chatID int64, token string, expiresAt time.Time) squirrel.Sqlizer {
	return squirrel.Insert("auth_confirmations").
		Columns("chat_id", "token", "expires_at").
		Values(chatID, token, expiresAt).
		PlaceholderFormat(squirrel.Dollar)
}

// IsConfirmedQuery builds a query to check if auth confirmation is confirmed
func IsConfirmedQuery(chatID int64, token string) squirrel.Sqlizer {
	return squirrel.Select("confirmed").
		From("auth_confirmations").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"token":   token,
		}).
		Where("expires_at > NOW()").
		PlaceholderFormat(squirrel.Dollar)
}

// ConfirmAuthConfirmationQuery builds a query to confirm auth confirmation
func ConfirmAuthConfirmationQuery(chatID int64, token string) squirrel.Sqlizer {
	return squirrel.Update("auth_confirmations").
		Set("confirmed", true).
		Where(squirrel.Eq{
			"chat_id": chatID,
			"token":   token,
		}).
		Where("expires_at > NOW()").
		PlaceholderFormat(squirrel.Dollar)
}

// DeleteAuthConfirmationQuery builds a query to delete auth confirmation
func DeleteAuthConfirmationQuery(chatID int64, token string) squirrel.Sqlizer {
	return squirrel.Delete("auth_confirmations").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"token":   token,
		}).
		PlaceholderFormat(squirrel.Dollar)
}

// CleanupAuthConfirmationsQuery builds a query to cleanup expired auth confirmations
func CleanupAuthConfirmationsQuery() squirrel.Sqlizer {
	return squirrel.Delete("auth_confirmations").
		Where("expires_at < NOW()").
		PlaceholderFormat(squirrel.Dollar)
}

// InsertCallbackQuery builds a query to insert a new callback data
func InsertCallbackQuery(chatID int64, data CallbackData, expiresAt time.Time) squirrel.Sqlizer {
	return squirrel.Insert("callback_data").
		Columns("uuid", "chat_id", "data", "expires_at").
		Values(squirrel.Expr("gen_random_uuid()"), chatID, data, expiresAt).
		Suffix("ON CONFLICT (uuid, chat_id) DO UPDATE SET data = EXCLUDED.data").
		Suffix("RETURNING uuid").
		PlaceholderFormat(squirrel.Dollar)
}

// FindCallbackQuery builds a query to find callback data
func FindCallbackQuery(chatID int64, uuid string) squirrel.Sqlizer {
	return squirrel.Select("data", "expires_at").
		From("callback_data").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"uuid":    uuid,
		}).
		PlaceholderFormat(squirrel.Dollar)
}

// CleanupCallbacksQuery builds a query to cleanup expired callbacks
func CleanupCallbacksQuery() squirrel.Sqlizer {
	return squirrel.Delete("callback_data").
		Where("expires_at < NOW()").
		PlaceholderFormat(squirrel.Dollar)
}

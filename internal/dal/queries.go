package dal

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
)

type DBType string

const (
	PostgreSQL DBType = "postgres"
	SQLite     DBType = "sqlite"
)

type Queries struct {
	dbType DBType
	qb     squirrel.StatementBuilderType
}

func NewQueries(dbType DBType) *Queries {
	return &Queries{
		dbType: dbType,
		qb:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (q *Queries) Clone() *Queries {
	return &Queries{
		dbType: q.dbType,
		qb:     q.qb,
	}
}

func (q *Queries) getUUIDFunction() string {
	switch q.dbType {
	case PostgreSQL:
		return "gen_random_uuid()"
	case SQLite:
		return "hex(randomblob(4))"
	default:
		return "gen_random_uuid()"
	}
}

func (q *Queries) getCurrentTimestampFunction() string {
	switch q.dbType {
	case PostgreSQL:
		return "NOW()"
	case SQLite:
		return "datetime('now', 'localtime')"
	default:
		return "NOW()"
	}
}

func (q *Queries) getCurrentDateFunction() string {
	switch q.dbType {
	case PostgreSQL:
		return "CURRENT_DATE"
	case SQLite:
		return "date('now', 'localtime')"
	default:
		return "CURRENT_DATE"
	}
}

func (q *Queries) toSquirrelDate(date time.Time) any {
	switch q.dbType {
	case PostgreSQL:
		return date
	case SQLite:
		return date.Format("2006-01-02")
	default:
		return date
	}
}

func (q *Queries) AddWordTranslationQuery(chatID int64, word, translation, description string) squirrel.Sqlizer {
	return q.qb.Insert("word_translations").
		Columns("chat_id", "word", "translation", "description").
		Values(chatID, word, translation, description).
		Suffix("ON CONFLICT (chat_id, word) DO UPDATE SET translation = EXCLUDED.translation, description = EXCLUDED.description")
}

func (q *Queries) FindWordTranslationsQuery(chatID int64, filter WordTranslationsFilter) (squirrel.Sqlizer, squirrel.Sqlizer) {
	baseQuery := q.qb.Select().
		From("word_translations").
		Where(squirrel.Eq{"chat_id": chatID})

	if filter.Word != "" {
		// Search in both word and translation fields for SQLite compatibility
		searchTerm := fmt.Sprintf("%%%s%%", strings.ToLower(filter.Word))
		baseQuery = baseQuery.Where(
			squirrel.Or{
				squirrel.Expr("LOWER(word) LIKE ?", searchTerm),
				squirrel.Expr("LOWER(translation) LIKE ?", searchTerm),
			},
		)
	}

	if filter.ToReview {
		baseQuery = baseQuery.Where(squirrel.Eq{"to_review": filter.ToReview})
	}

	switch filter.Guessed {
	case "", GuessedAll:
	case GuessedLearned:
		baseQuery = baseQuery.Where("guessed_streak >= 15")
	case GuessedBatched:
		baseQuery = baseQuery.Where("EXISTS (SELECT 1 FROM learning_batches lb WHERE lb.chat_id = word_translations.chat_id AND lb.word = word_translations.word)")
	case GuessedToLearn:
		baseQuery = baseQuery.Where("guessed_streak = 0")
	}

	selectQuery := baseQuery.
		Columns("chat_id", "word", "translation", "COALESCE(description, '')", "guessed_streak", "to_review", "created_at", "updated_at").
		OrderBy("word").
		Offset(filter.Offset).
		Limit(filter.Limit)

	countQuery := baseQuery.Columns("COUNT(*)")

	return selectQuery, countQuery
}

func (q *Queries) DeleteWordTranslationQuery(chatID int64, word string) squirrel.Sqlizer {
	return q.qb.Delete("word_translations").
		Where(squirrel.Eq{"chat_id": chatID, "word": word})
}

func (q *Queries) AddToLearningBatchQuery(chatID int64, word string) squirrel.Sqlizer {
	return q.qb.Insert("learning_batches").
		Columns("chat_id", "word").
		Values(chatID, word).
		Suffix("ON CONFLICT DO NOTHING")
}

func (q *Queries) IncreaseGuessedStreakQuery(chatID int64, word string) squirrel.Sqlizer {
	return q.qb.Update("word_translations").
		Set("guessed_streak", squirrel.Expr("guessed_streak + 1")).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})
}

func (q *Queries) ResetGuessedStreakQuery(chatID int64, word string) squirrel.Sqlizer {
	return q.qb.Update("word_translations").
		Set("guessed_streak", 0).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})
}

func (q *Queries) MarkToReviewQuery(chatID int64, word string, toReview bool) squirrel.Sqlizer {
	return q.qb.Update("word_translations").
		Set("to_review", toReview).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})
}

func (q *Queries) UpdateWordTranslationQuery(chatID int64, word, updatedWord, updatedTranslation, description string) squirrel.Sqlizer {
	return q.qb.Update("word_translations").
		Set("word", updatedWord).
		Set("translation", updatedTranslation).
		Set("description", description).
		Where(squirrel.Eq{"chat_id": chatID, "word": word})
}

func (q *Queries) ResetToReviewQuery(chatID int64) squirrel.Sqlizer {
	return q.qb.Update("word_translations").
		Set("to_review", false).
		Where(squirrel.Eq{"chat_id": chatID})
}

func (q *Queries) GetBatchedWordTranslationsCountQuery(chatID int64) squirrel.Sqlizer {
	return q.qb.Select("COUNT(*)").
		From("word_translations wt").
		Join("learning_batches lb ON wt.chat_id = lb.chat_id AND wt.word = lb.word").
		Where(squirrel.Eq{"wt.chat_id": chatID})
}

func (q *Queries) FindWordTranslationQuery(chatID int64, word string) squirrel.Sqlizer {
	return q.qb.Select(
		"wt.chat_id", "wt.word", "wt.translation",
		"COALESCE(wt.description, '')", "wt.guessed_streak",
		"wt.to_review", "wt.created_at", "wt.updated_at",
	).
		From("word_translations wt").
		Where(squirrel.Eq{"wt.chat_id": chatID, "wt.word": word})
}

func (q *Queries) FindRandomWordTranslationQuery(chatID int64, filter FindRandomWordFilter) squirrel.Sqlizer {
	var query squirrel.SelectBuilder

	if filter.Batched {
		query = q.qb.Select(
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
		query = q.qb.Select(
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

	return query
}

func (q *Queries) DeleteFromLearningBatchGtGuessedStreakQuery(chatID int64, guessedStreakLimit int) squirrel.Sqlizer {
	return q.qb.Delete("learning_batches").
		Where("chat_id = ? AND word IN (SELECT word FROM word_translations WHERE chat_id = ? AND guessed_streak >= ?)",
			chatID, chatID, guessedStreakLimit)
}

func (q *Queries) GetTotalStatsQuery(chatID int64) squirrel.Sqlizer {
	return q.qb.Select(
		"chat_id",
		"SUM(CASE WHEN guessed_streak >= 15 THEN 1 ELSE 0 END) AS streak_15_plus",
		"SUM(CASE WHEN guessed_streak BETWEEN 10 AND 14 THEN 1 ELSE 0 END) AS streak_10_to_14",
		"SUM(CASE WHEN guessed_streak BETWEEN 1 AND 9 THEN 1 ELSE 0 END) AS streak_1_to_9",
		"COUNT(*) AS total_words",
	).
		From("word_translations").
		Where(squirrel.Eq{"chat_id": chatID}).
		GroupBy("chat_id")
}

func (q *Queries) GetStatsQuery(chatID int64, date time.Time) squirrel.Sqlizer {
	return q.qb.Select(
		"chat_id", "date", "words_guessed", "words_missed",
		"total_words_learned", "created_at",
	).
		From("statistics").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"date":    q.toSquirrelDate(date),
		})
}

func (q *Queries) GetStatsRangeQuery(chatID int64, from, to time.Time) squirrel.Sqlizer {
	return q.qb.Select(
		"chat_id", "date", "words_guessed", "words_missed",
		"total_words_learned", "created_at",
	).
		From("statistics").
		Where(squirrel.Eq{"chat_id": chatID}).
		Where(squirrel.Expr("date BETWEEN ? AND ?", from, to)).
		OrderBy("date")
}

func (q *Queries) IncrementWordGuessedQuery(chatID int64) squirrel.Sqlizer {
	return q.qb.Insert("statistics").
		Columns("chat_id", "date", "words_guessed").
		Values(chatID, squirrel.Expr(q.getCurrentDateFunction()), 1).
		Suffix("ON CONFLICT (chat_id, date) DO UPDATE SET words_guessed = statistics.words_guessed + 1")
}

func (q *Queries) IncrementWordMissedQuery(chatID int64) squirrel.Sqlizer {
	return q.qb.Insert("statistics").
		Columns("chat_id", "date", "words_missed").
		Values(chatID, squirrel.Expr(q.getCurrentDateFunction()), 1).
		Suffix("ON CONFLICT (chat_id, date) DO UPDATE SET words_missed = statistics.words_missed + 1")
}

func (q *Queries) UpdateTotalWordsLearnedQuery(chatID int64) squirrel.Sqlizer {
	return q.qb.Update("statistics").
		Set("total_words_learned", squirrel.Select("COUNT(*)").
			From("word_translations").
			Where(squirrel.Eq{"chat_id": chatID}).
			Where("guessed_streak >= 15")).
		Where(squirrel.And{
			squirrel.Eq{
				"chat_id": chatID,
			},
			squirrel.Expr(fmt.Sprintf("date = %s", q.getCurrentDateFunction())),
		})
}

func (q *Queries) InsertAuthConfirmationQuery(chatID int64, token string, expiresAt time.Time) squirrel.Sqlizer {
	return q.qb.Insert("auth_confirmations").
		Columns("chat_id", "token", "expires_at").
		Values(chatID, token, expiresAt)
}

func (q *Queries) InsertCallbackQuery(chatID int64, data CallbackData, expiresAt time.Time) (squirrel.Sqlizer, error) {
	serializedData, err := q.serializeCallbackData(data)
	if err != nil {
		return nil, fmt.Errorf("serialize callback data: %w", err)
	}

	return q.qb.Insert("callback_data").
		Columns("uuid", "chat_id", "data", "expires_at").
		Values(squirrel.Expr(q.getUUIDFunction()), chatID, serializedData, expiresAt).
		Suffix("ON CONFLICT (uuid, chat_id) DO UPDATE SET data = EXCLUDED.data").
		Suffix("RETURNING uuid"), nil
}

func (q *Queries) IsConfirmedQuery(chatID int64, token string) squirrel.Sqlizer {
	return q.qb.Select("confirmed").
		From("auth_confirmations").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"token":   token,
		}).
		Where(squirrel.Expr("expires_at > " + q.getCurrentTimestampFunction()))
}

func (q *Queries) ConfirmAuthConfirmationQuery(chatID int64, token string) squirrel.Sqlizer {
	return q.qb.Update("auth_confirmations").
		Set("confirmed", true).
		Where(squirrel.Eq{
			"chat_id": chatID,
			"token":   token,
		}).
		Where(squirrel.Expr("expires_at > " + q.getCurrentTimestampFunction()))
}

func (q *Queries) DeleteAuthConfirmationQuery(chatID int64, token string) squirrel.Sqlizer {
	return q.qb.Delete("auth_confirmations").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"token":   token,
		})
}

func (q *Queries) CleanupAuthConfirmationsQuery() squirrel.Sqlizer {
	return q.qb.Delete("auth_confirmations").
		Where(squirrel.Expr("expires_at < " + q.getCurrentTimestampFunction()))
}

func (q *Queries) FindCallbackQuery(chatID int64, uuid string) squirrel.Sqlizer {
	return q.qb.Select("data", "expires_at").
		From("callback_data").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"uuid":    uuid,
		})
}

func (q *Queries) CleanupCallbacksQuery() squirrel.Sqlizer {
	return q.qb.Delete("callback_data").
		Where(squirrel.Expr("expires_at < " + q.getCurrentTimestampFunction()))
}

func (q *Queries) serializeCallbackData(data CallbackData) (interface{}, error) {
	if q.dbType == PostgreSQL {
		return data, nil
	}

	// For SQLite, we need to serialize to JSON string
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal callback data: %w", err)
	}
	return string(jsonData), nil
}

func (q *Queries) DeserializeCallbackData(data interface{}) (*CallbackData, error) {
	if q.dbType == PostgreSQL {
		cast, ok := data.(CallbackData)
		if !ok {
			return nil, fmt.Errorf("expected CallbackData type, got %T", data)
		}
		return &cast, nil
	}

	// For SQLite, we need to deserialize from JSON string
	strData, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("expected string data for SQLite, got %T", data)
	}
	var res CallbackData
	if err := json.Unmarshal([]byte(strData), &res); err != nil {
		return nil, fmt.Errorf("unmarshal callback data: %w", err)
	}
	return &res, nil
}

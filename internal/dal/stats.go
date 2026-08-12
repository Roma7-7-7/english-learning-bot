package dal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
)

// nearlyLearnedWidth is how many streaks below the learned threshold count as "nearly there" in the
// overall progress breakdown.
const nearlyLearnedWidth = 5

func (r *SQLiteRepository) GetTotalStats(ctx context.Context, chatID int64) (*TotalStats, error) {
	// Buckets are derived from the streak limit so that they keep meaning something when it is
	// retuned: [limit, ∞) is learned, the five below it are nearly there, the rest are early.
	nearlyFrom := max(1, r.streakLimit-nearlyLearnedWidth)
	query := qb.Select("chat_id").
		Column("SUM(CASE WHEN guessed_streak >= ? THEN 1 ELSE 0 END) AS learned", r.streakLimit).
		Column("SUM(CASE WHEN guessed_streak BETWEEN ? AND ? THEN 1 ELSE 0 END) AS nearly", nearlyFrom, r.streakLimit-1).
		Column("SUM(CASE WHEN guessed_streak BETWEEN 1 AND ? THEN 1 ELSE 0 END) AS early", nearlyFrom-1).
		Column("COUNT(*) AS total_words").
		From("word_translations").
		Where(squirrel.Eq{"chat_id": chatID}).
		GroupBy("chat_id")

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.db.QueryRowContext(ctx, sqlQuery, args...)

	var stats TotalStats
	err = row.Scan(
		&stats.ChatID,
		&stats.Learned,
		&stats.Nearly,
		&stats.Early,
		&stats.Total,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// An empty vocabulary still has to report the thresholds: they label the buckets, and a
			// zero would tell every caller that all words are learned.
			stats = TotalStats{ChatID: chatID}
		} else {
			return nil, fmt.Errorf("get stats: %w", err)
		}
	}
	stats.StreakLimit = r.streakLimit
	stats.NearlyFrom = nearlyFrom

	batched, err := batchedWordTranslationsCount(ctx, r.db, chatID)
	if err != nil {
		return nil, fmt.Errorf("get batched word translations count: %w", err)
	}
	stats.Batched = batched

	return &stats, nil
}

func (r *SQLiteRepository) GetStats(ctx context.Context, chatID int64, date time.Time) (*Stats, error) {
	var r2 any = date.Format("2006-01-02")
	query := qb.Select(
		"chat_id", "date", "words_guessed", "words_missed",
		"total_words_learned", "created_at",
	).
		From("statistics").
		Where(squirrel.Eq{
			"chat_id": chatID,
			"date":    r2,
		})

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.db.QueryRowContext(ctx, sqlQuery, args...)

	var stats Stats
	var strDate string
	err = row.Scan(
		&stats.ChatID,
		&strDate,
		&stats.WordsGuessed,
		&stats.WordsMissed,
		&stats.TotalWordsLearned,
		&stats.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get stats: %w", err)
	}
	stats.Date, err = time.Parse("2006-01-02", strDate)
	if err != nil {
		return nil, fmt.Errorf("parse date: %w", err)
	}
	return &stats, nil
}

func (r *SQLiteRepository) GetStatsRange(ctx context.Context, chatID int64, from, to time.Time) ([]Stats, error) {
	query := qb.Select(
		"chat_id", "date", "words_guessed", "words_missed",
		"total_words_learned", "created_at",
	).
		From("statistics").
		Where(squirrel.Eq{"chat_id": chatID}).
		Where(squirrel.Expr("date BETWEEN ? AND ?", from, to)).
		OrderBy("date")

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("get stats range: %w", err)
	}
	defer rows.Close()

	var stats []Stats
	var dateStr string
	for rows.Next() {
		var stat Stats
		err := rows.Scan(
			&stat.ChatID,
			&dateStr,
			&stat.WordsGuessed,
			&stat.WordsMissed,
			&stat.TotalWordsLearned,
			&stat.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan stats: %w", err)
		}
		stat.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, fmt.Errorf("parse date: %w", err)
		}
		stats = append(stats, stat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stats: %w", err)
	}

	return stats, nil
}

func incrementWordGuessed(ctx context.Context, e execer, chatID int64) error {
	query := qb.Insert("statistics").
		Columns("chat_id", "date", "words_guessed").
		Values(chatID, squirrel.Expr("date('now', 'localtime')"), 1).
		Suffix("ON CONFLICT (chat_id, date) DO UPDATE SET words_guessed = statistics.words_guessed + 1")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = e.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("increment word guessed: %w", err)
	}
	return nil
}

func incrementWordMissed(ctx context.Context, e execer, chatID int64) error {
	query := qb.Insert("statistics").
		Columns("chat_id", "date", "words_missed").
		Values(chatID, squirrel.Expr("date('now', 'localtime')"), 1).
		Suffix("ON CONFLICT (chat_id, date) DO UPDATE SET words_missed = statistics.words_missed + 1")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = e.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("increment word missed: %w", err)
	}
	return nil
}

// updateTotalWordsLearned recomputes today's learned count from the vocabulary itself.
//
// It inserts today's row as well as updating it: a streak reset from the UI can be the first thing
// that happens on a given day, and a plain UPDATE would match nothing and leave the dashboard
// showing yesterday's number until an answer in Telegram created the row.
func updateTotalWordsLearned(ctx context.Context, e execer, chatID int64, streakLimit int) error {
	// The subquery is inlined with "?" placeholders so that the outer builder's Dollar format is
	// applied exactly once, over the whole statement.
	learned := squirrel.Expr(
		"(SELECT COUNT(*) FROM word_translations WHERE chat_id = ? AND guessed_streak >= ?)",
		chatID, streakLimit)

	query := qb.Insert("statistics").
		Columns("chat_id", "date", "total_words_learned").
		Values(chatID, squirrel.Expr("date('now', 'localtime')"), learned).
		Suffix("ON CONFLICT (chat_id, date) DO UPDATE SET total_words_learned = EXCLUDED.total_words_learned")

	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = e.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("update total words learned: %w", err)
	}
	return nil
}

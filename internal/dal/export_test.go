package dal

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// This file is compiled into package dal, so it is the one place where the external dal_test package
// can reach the unexported repository internals: the private constructor, the raw *sql.DB used to
// assert on stored rows, and the statement helpers used to seed state.

const TestChatID int64 = 42

const TestStreakLimit int = 15

// TestRepo is a repository backed by a fresh in-memory database with the production schema applied,
// plus the assertion helpers the tests need. It embeds *SQLiteRepository, so every repository method
// is available on it directly.
type TestRepo struct {
	*SQLiteRepository

	t *testing.T
}

// NewTestRepo builds a TestRepo through the unexported constructor, so that no background cleanup
// goroutines are started.
func NewTestRepo(t *testing.T) *TestRepo {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// A shared in-memory database is dropped when its last connection closes, so keep the pool at
	// one connection for the lifetime of the test.
	db.SetMaxOpenConns(1)

	schema, err := os.ReadFile(filepath.Join("..", "..", "schema", "schema_sqlite.sql"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	if _, err = db.Exec(string(schema)); err != nil {
		t.Fatalf("apply schema: %v", err)
	}

	repo := newSQLRepository(db, TestStreakLimit, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return &TestRepo{SQLiteRepository: repo, t: t}
}

// SetStreakLimit retunes the "is this learned?" threshold the repository queries with.
func (r *TestRepo) SetStreakLimit(limit int) {
	r.streakLimit = limit
}

func (r *TestRepo) AddWord(word string, streak int) {
	r.t.Helper()

	ctx := context.Background()
	if err := r.CreateWordTranslation(ctx, TestChatID, word, word+"-translation", ""); err != nil {
		r.t.Fatalf("add word %q: %v", word, err)
	}
	if streak == 0 {
		return
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE word_translations SET guessed_streak = ? WHERE chat_id = ? AND word = ?", streak, TestChatID, word)
	if err != nil {
		r.t.Fatalf("set streak for %q: %v", word, err)
	}
}

// SeedBatch puts words into the learning batch directly, bypassing the refill rules.
func (r *TestRepo) SeedBatch(words ...string) {
	r.t.Helper()

	ctx := context.Background()
	for _, word := range words {
		if err := r.inTx(ctx, func(e execer) error { return addToLearningBatch(ctx, e, TestChatID, word) }); err != nil {
			r.t.Fatalf("seed batch with %q: %v", word, err)
		}
	}
}

func (r *TestRepo) StreakOf(word string) int {
	r.t.Helper()

	var streak int
	err := r.db.QueryRowContext(context.Background(),
		"SELECT guessed_streak FROM word_translations WHERE chat_id = ? AND word = ?", TestChatID, word).Scan(&streak)
	if err != nil {
		r.t.Fatalf("get streak for %q: %v", word, err)
	}
	return streak
}

// ReviewSeq is the rotation cursor stamped by MarkWordReviewed.
func (r *TestRepo) ReviewSeq(word string) int {
	r.t.Helper()

	var seq int
	err := r.db.QueryRowContext(context.Background(),
		"SELECT last_reviewed_seq FROM word_translations WHERE chat_id = ? AND word = ?", TestChatID, word).Scan(&seq)
	if err != nil {
		r.t.Fatalf("read cursor for %q: %v", word, err)
	}
	return seq
}

func (r *TestRepo) IsBatched(word string) bool {
	r.t.Helper()

	var count int
	err := r.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM learning_batches WHERE chat_id = ? AND word = ?", TestChatID, word).Scan(&count)
	if err != nil {
		r.t.Fatalf("check batch membership for %q: %v", word, err)
	}
	return count > 0
}

func (r *TestRepo) BatchWords() []string {
	r.t.Helper()

	rows, err := r.db.QueryContext(context.Background(),
		"SELECT word FROM learning_batches WHERE chat_id = ? ORDER BY word", TestChatID)
	if err != nil {
		r.t.Fatalf("list batch: %v", err)
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			r.t.Fatalf("scan batch word: %v", err)
		}
		words = append(words, word)
	}
	if err := rows.Err(); err != nil {
		r.t.Fatalf("iterate batch: %v", err)
	}
	return words
}

// TodayStats returns today's guessed, missed and total-learned counters.
func (r *TestRepo) TodayStats() (int, int, int) {
	r.t.Helper()

	var guessed, missed, totalLearned int
	err := r.db.QueryRowContext(context.Background(),
		`SELECT words_guessed, words_missed, total_words_learned FROM statistics
		 WHERE chat_id = ? AND date = date('now', 'localtime')`, TestChatID).
		Scan(&guessed, &missed, &totalLearned)
	if err != nil {
		r.t.Fatalf("get today's stats: %v", err)
	}
	return guessed, missed, totalLearned
}

// HasColumn reports whether the applied schema defines table.column.
func (r *TestRepo) HasColumn(table, column string) bool {
	r.t.Helper()

	// PRAGMA does not accept bound parameters for the table name; it comes from a repo file, not
	// from user input.
	rows, err := r.db.QueryContext(context.Background(), "SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		r.t.Fatalf("read columns of %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			r.t.Fatalf("scan column name: %v", err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		r.t.Fatalf("iterate columns: %v", err)
	}
	return false
}

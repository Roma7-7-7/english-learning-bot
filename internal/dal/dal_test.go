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

const testChatID int64 = 42

// newTestRepo returns a repository backed by a fresh in-memory database with the production schema
// applied. It uses the unexported constructor so that no background cleanup goroutines are started.
func newTestRepo(t *testing.T) *SQLiteRepository {
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

	return newSQLRepository(db, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func addWord(t *testing.T, r *SQLiteRepository, word string, streak int) {
	t.Helper()

	ctx := context.Background()
	if err := r.AddWordTranslation(ctx, testChatID, word, word+"-translation", ""); err != nil {
		t.Fatalf("add word %q: %v", word, err)
	}
	if streak == 0 {
		return
	}
	_, err := r.db.ExecContext(ctx,
		"UPDATE word_translations SET guessed_streak = ? WHERE chat_id = ? AND word = ?", streak, testChatID, word)
	if err != nil {
		t.Fatalf("set streak for %q: %v", word, err)
	}
}

func streakOf(t *testing.T, r *SQLiteRepository, word string) int {
	t.Helper()

	var streak int
	err := r.db.QueryRowContext(context.Background(),
		"SELECT guessed_streak FROM word_translations WHERE chat_id = ? AND word = ?", testChatID, word).Scan(&streak)
	if err != nil {
		t.Fatalf("get streak for %q: %v", word, err)
	}
	return streak
}

func isBatched(t *testing.T, r *SQLiteRepository, word string) bool {
	t.Helper()

	var count int
	err := r.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM learning_batches WHERE chat_id = ? AND word = ?", testChatID, word).Scan(&count)
	if err != nil {
		t.Fatalf("check batch membership for %q: %v", word, err)
	}
	return count > 0
}

func batchWords(t *testing.T, r *SQLiteRepository) []string {
	t.Helper()

	rows, err := r.db.QueryContext(context.Background(),
		"SELECT word FROM learning_batches WHERE chat_id = ? ORDER BY word", testChatID)
	if err != nil {
		t.Fatalf("list batch: %v", err)
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var word string
		if err := rows.Scan(&word); err != nil {
			t.Fatalf("scan batch word: %v", err)
		}
		words = append(words, word)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate batch: %v", err)
	}
	return words
}

func todayStats(t *testing.T, r *SQLiteRepository) (guessed, missed, totalLearned int) {
	t.Helper()

	err := r.db.QueryRowContext(context.Background(),
		`SELECT words_guessed, words_missed, total_words_learned FROM statistics
		 WHERE chat_id = ? AND date = date('now', 'localtime')`, testChatID).
		Scan(&guessed, &missed, &totalLearned)
	if err != nil {
		t.Fatalf("get today's stats: %v", err)
	}
	return guessed, missed, totalLearned
}

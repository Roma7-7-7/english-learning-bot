package dal

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var alterAddColumnRe = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(\w+)\s+ADD\s+COLUMN\s+(\w+)`)

// There is no migration runner: a fresh database is built from schema_sqlite.sql while a live one is
// patched by the files under schema/migrations. Nothing keeps the two in step, and the failure is
// silent - the app works locally and breaks on the deployed database, or vice versa.
//
// This asserts every column added by a migration also exists in the base schema.
func TestMigrationsMatchBaseSchema(t *testing.T) {
	r := newTestRepo(t) // applies schema_sqlite.sql

	entries, err := os.ReadDir(filepath.Join("..", "..", "schema", "migrations"))
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}

	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		body, rErr := os.ReadFile(filepath.Join("..", "..", "schema", "migrations", entry.Name()))
		if rErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), rErr)
		}

		for _, m := range alterAddColumnRe.FindAllStringSubmatch(string(body), -1) {
			table, column := m[1], m[2]
			if !hasColumn(t, r, table, column) {
				t.Errorf("%s adds %s.%s, but schema_sqlite.sql does not define it", entry.Name(), table, column)
			}
			checked++
		}
	}

	if checked == 0 {
		t.Error("no ALTER TABLE ... ADD COLUMN found in schema/migrations; did the parser stop matching?")
	}
}

func hasColumn(t *testing.T, r *SQLiteRepository, table, column string) bool {
	t.Helper()

	// PRAGMA does not accept bound parameters for the table name; it comes from a repo file, not
	// from user input.
	rows, err := r.db.QueryContext(context.Background(), "SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		t.Fatalf("read columns of %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan column name: %v", err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate columns: %v", err)
	}
	return false
}

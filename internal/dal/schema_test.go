package dal_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

var alterAddColumnRe = regexp.MustCompile(`(?i)ALTER\s+TABLE\s+(\w+)\s+ADD\s+COLUMN\s+(\w+)`)

// There is no migration runner: a fresh database is built from schema_sqlite.sql while a live one is
// patched by the files under schema/migrations. Nothing keeps the two in step, and the failure is
// silent - the app works locally and breaks on the deployed database, or vice versa.
//
// This asserts every column added by a migration also exists in the base schema.
func TestMigrationsMatchBaseSchema(t *testing.T) {
	r := dal.NewTestRepo(t) // applies schema_sqlite.sql

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
			if !r.HasColumn(table, column) {
				t.Errorf("%s adds %s.%s, but schema_sqlite.sql does not define it", entry.Name(), table, column)
			}
			checked++
		}
	}

	if checked == 0 {
		t.Error("no ALTER TABLE ... ADD COLUMN found in schema/migrations; did the parser stop matching?")
	}
}

package dal

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
)

//nolint:gochecknoglobals // this is query builder
var qb = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type (
	// execer is the subset of database/sql shared by *sql.DB and *sql.Tx. Statement helpers take
	// one so that they can run either standalone or as part of a transaction opened by inTx.
	execer interface {
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	}

	SQLiteRepository struct {
		db  *sql.DB
		log *slog.Logger
	}
)

func NewSQLiteRepository(ctx context.Context, client *sql.DB, log *slog.Logger) *SQLiteRepository {
	res := newSQLRepository(client, log)
	go res.cleanupCallbacksJob(ctx)
	go res.cleanupAuthConfirmations(ctx)
	return res
}

// inTx runs fn inside a transaction, committing when it returns nil and rolling back otherwise.
//
// The transaction never leaves the package: fn receives an execer, not a Repository. That keeps
// composite writes atomic by construction and makes it impossible to accidentally run a statement
// against the pool while a transaction is open. Callers outside dal express multi-statement work as
// a single repository method instead (see learning.go).
//
// Do not call query helpers that fan out concurrently (FindWordTranslations) from fn: *sql.Tx is not
// safe for concurrent use.
func (r *SQLiteRepository) inTx(ctx context.Context, fn func(e execer) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op once the transaction is committed

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func newSQLRepository(db *sql.DB, log *slog.Logger) *SQLiteRepository {
	return &SQLiteRepository{db: db, log: log}
}

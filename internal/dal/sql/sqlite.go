package sql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

var qb = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

type (
	Client interface {
		BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
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

func (r *SQLiteRepository) Transact(ctx context.Context, txFunc func(r dal.Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // ignore rollback errors

	if err = txFunc(newSQLRepository(r.db, r.log)); err != nil {
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

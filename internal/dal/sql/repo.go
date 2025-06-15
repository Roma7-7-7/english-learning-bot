package sql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/Roma7-7-7/english-learning-bot/internal/dal"
)

type (
	Client interface {
		BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
		ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
		QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
		QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	}

	Repository struct {
		client  Client
		queries *dal.Queries
		log     *slog.Logger
	}
)

func NewRepository(ctx context.Context, client Client, dbType dal.DBType, log *slog.Logger) *Repository {
	res := newSQLRepository(client, dal.NewQueries(dbType), log)
	go res.cleanupCallbacksJob(ctx)
	go res.cleanupAuthConfirmations(ctx)
	return res
}

func (r *Repository) Transact(ctx context.Context, txFunc func(r dal.Repository) error) error {
	tx, err := r.client.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck // ignore rollback errors

	if err = txFunc(newSQLRepository(r.client, r.queries.Clone(), r.log)); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func newSQLRepository(client Client, queries *dal.Queries, log *slog.Logger) *Repository {
	return &Repository{client: client, queries: queries, log: log}
}

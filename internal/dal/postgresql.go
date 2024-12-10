package dal

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type (
	Client interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
		QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
		Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	}

	PostgreSQLRepository struct {
		client Client
		log    *slog.Logger
	}
)

func NewPostgreSQLRepository(ctx context.Context, client Client, log *slog.Logger) *PostgreSQLRepository {
	res := newPostgreSQLRepository(client, log)
	go res.cleanupJob(ctx)
	return res
}

func (r *PostgreSQLRepository) Transact(ctx context.Context, txFunc func(r Repository) error) error {
	tx, err := r.client.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // ignore rollback errors

	if err = txFunc(newPostgreSQLRepository(r.client, r.log)); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func newPostgreSQLRepository(client Client, log *slog.Logger) *PostgreSQLRepository {
	return &PostgreSQLRepository{client: client, log: log}
}

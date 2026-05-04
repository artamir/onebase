package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

// WithTx runs fn inside a PostgreSQL transaction. On fn error the transaction
// is rolled back; on success it is committed.
func (db *DB) WithTx(ctx context.Context, fn func(context.Context) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}
	txCtx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(txCtx); err != nil {
		tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// ContextWithTx embeds tx into ctx so that exec/q will use it.
func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// BeginTx starts a new PostgreSQL transaction and returns it together with a
// context that has the transaction embedded for use by exec/q.
func (db *DB) BeginTx(ctx context.Context) (pgx.Tx, context.Context, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, ctx, err
	}
	return tx, ContextWithTx(ctx, tx), nil
}

// exec uses the transaction from ctx if present, otherwise the pool.
func (db *DB) exec(ctx context.Context, sql string, args ...any) error {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		_, err := tx.Exec(ctx, sql, args...)
		return err
	}
	_, err := db.pool.Exec(ctx, sql, args...)
	return err
}

// querier returns a query executor that respects the transaction in ctx.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (db *DB) q(ctx context.Context) querier {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db.pool
}

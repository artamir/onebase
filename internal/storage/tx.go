package storage

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

// WithTx runs fn inside a transaction. On fn error the transaction is rolled
// back; on success it is committed.
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

// ContextWithTx embeds a storage.Tx into ctx so that exec/q/Exec/Query use it.
func ContextWithTx(ctx context.Context, tx Tx) context.Context {
	if pt, ok := tx.(*pgxTx); ok {
		return context.WithValue(ctx, txKey{}, pt.tx)
	}
	return context.WithValue(ctx, txKey{}, tx)
}

// BeginTx starts a new transaction and returns it together with a context
// that has the transaction embedded for use by Exec/Query/QueryRow.
func (db *DB) BeginTx(ctx context.Context) (Tx, context.Context, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, ctx, err
	}
	storTx := &pgxTx{tx: tx}
	return storTx, context.WithValue(ctx, txKey{}, tx), nil
}

// Exec runs a non-query SQL statement, respecting any transaction in ctx.
func (db *DB) Exec(ctx context.Context, sql string, args ...any) (CommandTag, error) {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return cmdTag(tx.Exec(ctx, sql, args...))
	}
	return cmdTag(db.pool.Exec(ctx, sql, args...))
}

// Query runs a SQL query and returns multiple rows, respecting any transaction in ctx.
func (db *DB) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		rows, err := tx.Query(ctx, sql, args...)
		if err != nil {
			return nil, err
		}
		return &pgxRows{r: rows}, nil
	}
	rows, err := db.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRows{r: rows}, nil
}

// QueryRow runs a SQL query expected to return at most one row, respecting any
// transaction in ctx.
func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) Row {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return pgxRow{r: tx.QueryRow(ctx, sql, args...)}
	}
	return pgxRow{r: db.pool.QueryRow(ctx, sql, args...)}
}

// exec is the internal helper retained for existing internal callers.
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

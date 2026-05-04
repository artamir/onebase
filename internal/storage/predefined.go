package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/ivantit66/onebase/internal/metadata"
)

// EnsurePredefinedColumns adds _predefined_name and _is_predefined columns to
// catalog tables that declare predefined items. Safe to call repeatedly (IF NOT EXISTS).
func (db *DB) EnsurePredefinedColumns(ctx context.Context, entities []*metadata.Entity) error {
	for _, e := range entities {
		if e.Kind != metadata.KindCatalog || len(e.Predefined) == 0 {
			continue
		}
		table := metadata.TableName(e.Name)
		if _, err := db.pool.Exec(ctx, AddColumnSQL(table, "_predefined_name", "TEXT")); err != nil {
			return fmt.Errorf("ensure predefined cols %s._predefined_name: %w", e.Name, err)
		}
		if _, err := db.pool.Exec(ctx, AddColumnSQL(table, "_is_predefined", "BOOLEAN NOT NULL DEFAULT FALSE")); err != nil {
			return fmt.Errorf("ensure predefined cols %s._is_predefined: %w", e.Name, err)
		}
		// Unique index on _predefined_name for predefined rows (partial index)
		idxName := "idx_" + strings.ToLower(e.Name) + "_predefined"
		idxSQL := fmt.Sprintf(
			`CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (_predefined_name) WHERE _is_predefined = TRUE`,
			idxName, table)
		if _, err := db.pool.Exec(ctx, idxSQL); err != nil {
			return fmt.Errorf("ensure predefined index %s: %w", e.Name, err)
		}
	}
	return nil
}

// SyncPredefined upserts all predefined items declared in the entity YAML into
// the database. The _predefined_name is used as the conflict target so the UUID
// never changes on subsequent syncs — only field values are updated.
func (db *DB) SyncPredefined(ctx context.Context, e *metadata.Entity) error {
	if len(e.Predefined) == 0 {
		return nil
	}
	table := metadata.TableName(e.Name)
	for _, item := range e.Predefined {
		cols := []string{"id", "_predefined_name", "_is_predefined"}
		phs := []string{"gen_random_uuid()", "$1", "TRUE"}
		args := []any{item.Name}
		updates := []string{"_is_predefined = TRUE"}
		argIdx := 2

		for _, f := range e.Fields {
			col := metadata.ColumnName(f)
			val, ok := item.Fields[f.Name]
			if !ok {
				continue
			}
			cols = append(cols, col)
			phs = append(phs, fmt.Sprintf("$%d", argIdx))
			args = append(args, val)
			updates = append(updates, fmt.Sprintf("%s = EXCLUDED.%s", col, col))
			argIdx++
		}

		sql := fmt.Sprintf(
			`INSERT INTO %s (%s) VALUES (%s)
			 ON CONFLICT (_predefined_name) WHERE _is_predefined = TRUE
			 DO UPDATE SET %s`,
			table,
			strings.Join(cols, ", "),
			strings.Join(phs, ", "),
			strings.Join(updates, ", "),
		)
		if _, err := db.pool.Exec(ctx, sql, args...); err != nil {
			return fmt.Errorf("sync predefined %s.%s: %w", e.Name, item.Name, err)
		}
	}
	return nil
}

// GetPredefinedID returns the UUID of a predefined item by its name.
func (db *DB) GetPredefinedID(ctx context.Context, entityName, predefinedName string) (uuid.UUID, error) {
	table := metadata.TableName(entityName)
	var id uuid.UUID
	err := db.pool.QueryRow(ctx,
		fmt.Sprintf(`SELECT id FROM %s WHERE _predefined_name = $1 AND _is_predefined = TRUE`, table),
		predefinedName,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("predefined %s.%s not found: %w", entityName, predefinedName, err)
	}
	return id, nil
}

// GetPredefinedIDStr is a string-returning variant of GetPredefinedID for the
// DSL interpreter interface (avoids uuid dependency in the interpreter package).
func (db *DB) GetPredefinedIDStr(ctx context.Context, entityName, predefinedName string) (string, error) {
	id, err := db.GetPredefinedID(ctx, entityName, predefinedName)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

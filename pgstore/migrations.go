package pgstore

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations.sql
var migrationSQL string

// MigrationSQL returns the SQL used by the reference pgstore schema.
func MigrationSQL() string {
	return migrationSQL
}

// ApplyMigrations executes the embedded migration SQL.
func ApplyMigrations(ctx context.Context, db *pgxpool.Pool) error {
	if db == nil {
		return fmt.Errorf("nil database pool")
	}
	if _, err := db.Exec(ctx, migrationSQL); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

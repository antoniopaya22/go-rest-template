package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"

	"github.com/antoniopaya22/go-rest-template/v3/migrations"
)

// MigrationStatus describes one migration and whether it has been applied.
type MigrationStatus struct {
	Version int64
	Name    string
	Applied bool
}

func newProvider(dialect goose.Dialect, db *sql.DB, dir string, options ...goose.ProviderOption) (*goose.Provider, error) {
	files, err := fs.Sub(migrations.FS, dir)
	if err != nil {
		return nil, fmt.Errorf("open embedded migrations: %w", err)
	}
	return goose.NewProvider(dialect, db, files, options...)
}

// Migrate applies every pending migration.
func (db *DB) Migrate(ctx context.Context) error {
	provider, err := db.driver.migrator(db.pool)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// MigrateDown rolls back the most recent migration.
func (db *DB) MigrateDown(ctx context.Context) error {
	provider, err := db.driver.migrator(db.pool)
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	if _, err := provider.Down(ctx); err != nil {
		return fmt.Errorf("roll back migration: %w", err)
	}
	return nil
}

// MigrationStatus lists every known migration.
func (db *DB) MigrationStatus(ctx context.Context) ([]MigrationStatus, error) {
	provider, err := db.driver.migrator(db.pool)
	if err != nil {
		return nil, fmt.Errorf("create migration provider: %w", err)
	}

	statuses, err := provider.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("read migration status: %w", err)
	}

	result := make([]MigrationStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, MigrationStatus{
			Version: status.Source.Version,
			Name:    status.Source.Path,
			Applied: status.State == goose.StateApplied,
		})
	}
	return result, nil
}

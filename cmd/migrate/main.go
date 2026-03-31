// Command migrate applies pending SQL migrations.
package main

import (
	"context"
	"database/sql"
	"log"

	"github.com/antonioalfa22/go-rest-template/internal/app/api/store/postgres"
	"github.com/antonioalfa22/go-rest-template/internal/app/api/store/sqlite"
	"github.com/antonioalfa22/go-rest-template/internal/pkg/config"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	var dbOpener func(context.Context, config.DatabaseConfig) (*sql.DB, error)
	var migrationRunner func(context.Context, *sql.DB, string) error

	switch cfg.Database.Driver {
	case config.DatabaseDriverSQLite:
		dbOpener = sqlite.Open
		migrationRunner = sqlite.ApplyMigrations
	case config.DatabaseDriverPostgres:
		dbOpener = postgres.Open
		migrationRunner = postgres.ApplyMigrations
	default:
		log.Fatalf("unsupported database driver: %s", cfg.Database.Driver)
	}

	db, err := dbOpener(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("close database: %v", closeErr)
		}
	}()

	if err := migrationRunner(ctx, db, cfg.Database.MigrationDir()); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
}

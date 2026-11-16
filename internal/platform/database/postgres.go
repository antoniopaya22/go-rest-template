package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // Registers the pgx database/sql driver.
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"

	"github.com/antoniopaya22/go-rest-template/internal/config"
)

func init() {
	register(config.DriverPostgres, driver{
		open:                openPostgres,
		numbered:            true,
		uniqueViolation:     func(err error) bool { return postgresCode(err) == "23505" },
		foreignKeyViolation: func(err error) bool { return postgresCode(err) == "23503" },
		migrator: func(db *sql.DB) (*goose.Provider, error) {
			// The advisory lock keeps replicas that start together from
			// applying the same migration twice.
			locker, err := lock.NewPostgresSessionLocker()
			if err != nil {
				return nil, err
			}
			return newProvider(goose.DialectPostgres, db, config.DriverPostgres, goose.WithSessionLocker(locker))
		},
	})
}

func openPostgres(_ context.Context, cfg config.Database) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	return db, nil
}

func postgresCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

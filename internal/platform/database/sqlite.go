package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/antoniopaya22/go-rest-template/v3/internal/config"
)

func init() {
	register(config.DriverSQLite, driver{
		open:                openSQLite,
		uniqueViolation:     func(err error) bool { return sqliteCode(err) == sqlite3.SQLITE_CONSTRAINT_UNIQUE },
		foreignKeyViolation: func(err error) bool { return sqliteCode(err) == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY },
		migrator: func(db *sql.DB) (*goose.Provider, error) {
			return newProvider(goose.DialectSQLite3, db, config.DriverSQLite)
		},
	})
}

func openSQLite(_ context.Context, cfg config.Database) (*sql.DB, error) {
	path := filepath.Clean(cfg.SQLitePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	// Pragmas go in the DSN so they apply to every connection of the pool.
	options := url.Values{}
	options.Add("_pragma", "foreign_keys(1)")
	options.Add("_pragma", "busy_timeout(5000)")
	options.Add("_pragma", "journal_mode(WAL)")
	options.Set("_time_format", "sqlite")

	db, err := sql.Open("sqlite", "file:"+path+"?"+options.Encode())
	if err != nil {
		return nil, err
	}

	// SQLite allows a single writer; one connection avoids SQLITE_BUSY errors.
	db.SetMaxOpenConns(1)
	return db, nil
}

func sqliteCode(err error) int {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		return sqliteErr.Code()
	}
	return 0
}

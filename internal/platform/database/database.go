// Package database opens the relational database, runs migrations and gives
// repositories a small, driver-independent query API.
//
// Queries are written once with "?" placeholders; DB rewrites them for the
// active driver. Each driver lives in its own file and registers itself, so
// removing a backend means deleting one file and its migrations directory.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pressly/goose/v3"

	"github.com/antoniopaya22/go-rest-template/internal/config"
	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
)

// driver describes one database backend.
type driver struct {
	open func(ctx context.Context, cfg config.Database) (*sql.DB, error)
	// numbered reports whether placeholders are written $1, $2... instead of ?.
	numbered bool
	// uniqueViolation and foreignKeyViolation classify driver errors.
	uniqueViolation     func(error) bool
	foreignKeyViolation func(error) bool
	// migrator builds the goose provider for this backend.
	migrator func(db *sql.DB) (*goose.Provider, error)
}

var drivers = map[string]driver{}

func register(name string, d driver) { drivers[name] = d }

// Drivers returns the names of the compiled-in database drivers.
func Drivers() []string {
	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DB is a connection pool bound to one driver.
type DB struct {
	pool   *sql.DB
	driver driver
	name   string
}

// Open connects to the configured database and verifies the connection.
func Open(ctx context.Context, cfg config.Database) (*DB, error) {
	d, ok := drivers[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("database driver %q is not compiled in (available: %s)", cfg.Driver, strings.Join(Drivers(), ", "))
	}

	pool, err := d.open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.Driver, err)
	}

	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping %s database: %w", cfg.Driver, err)
	}

	return &DB{pool: pool, driver: d, name: cfg.Driver}, nil
}

// Driver returns the name of the active driver.
func (db *DB) Driver() string { return db.name }

// Ping verifies that the database is reachable.
func (db *DB) Ping(ctx context.Context) error { return db.pool.PingContext(ctx) }

// Close releases the connection pool.
func (db *DB) Close() error { return db.pool.Close() }

type txKey struct{}

// queryer is the subset of *sql.DB and *sql.Tx used by repositories.
type queryer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (db *DB) conn(ctx context.Context) queryer {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return db.pool
}

// InTx runs fn inside a transaction. Queries issued through db with the
// context passed to fn join the transaction, so repositories need no special
// transactional variants. Nested calls reuse the outer transaction.
func (db *DB) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return fn(ctx)
	}

	tx, err := db.pool.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// Exec runs a statement that returns no rows and reports the affected row count.
func (db *DB) Exec(ctx context.Context, query string, args ...any) (int64, error) {
	result, err := db.conn(ctx).ExecContext(ctx, db.Rebind(query), args...)
	if err != nil {
		return 0, db.translate(err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read affected rows: %w", err)
	}
	return affected, nil
}

// Query runs a statement that returns rows.
func (db *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := db.conn(ctx).QueryContext(ctx, db.Rebind(query), args...)
	if err != nil {
		return nil, db.translate(err)
	}
	return rows, nil
}

// QueryRow runs a statement expected to return at most one row.
// Pass the Scan error to Translate to map it to an application error.
func (db *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return db.conn(ctx).QueryRowContext(ctx, db.Rebind(query), args...)
}

// Rebind rewrites "?" placeholders into the form the active driver expects.
func (db *DB) Rebind(query string) string {
	if !db.driver.numbered {
		return query
	}

	var b strings.Builder
	b.Grow(len(query) + 8)
	position := 0
	for _, r := range query {
		if r == '?' {
			position++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(position))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Translate maps driver errors to application errors: a missing row becomes
// not_found, a unique violation becomes conflict and a foreign key violation
// becomes invalid_argument. Other errors are returned unchanged.
func (db *DB) Translate(err error, resource string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sql.ErrNoRows):
		return errs.Wrap(errs.CodeNotFound, resource+" not found", err)
	case db.driver.uniqueViolation(err):
		return errs.Wrap(errs.CodeConflict, resource+" already exists", err)
	case db.driver.foreignKeyViolation(err):
		return errs.Wrap(errs.CodeInvalidArgument, resource+" references a record that does not exist", err)
	default:
		return err
	}
}

func (db *DB) translate(err error) error {
	return db.Translate(err, "record")
}

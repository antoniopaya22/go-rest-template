package database

import (
	"context"
	"errors"        // init:sqlite
	"path/filepath" // init:sqlite
	"testing"

	"github.com/antoniopaya22/go-rest-template/v3/internal/config"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs" // init:sqlite
)

func TestRebind(t *testing.T) {
	t.Parallel()

	const query = `SELECT id FROM users WHERE a = ? AND b = ? LIMIT ?`

	numbered := &DB{driver: driver{numbered: true}}
	if got, want := numbered.Rebind(query), `SELECT id FROM users WHERE a = $1 AND b = $2 LIMIT $3`; got != want {
		t.Errorf("numbered = %q, want %q", got, want)
	}

	positional := &DB{driver: driver{}}
	if got := positional.Rebind(query); got != query {
		t.Errorf("positional = %q, want it unchanged", got)
	}
}

func TestOpenUnknownDriver(t *testing.T) {
	t.Parallel()

	if _, err := Open(context.Background(), config.Database{Driver: "oracle"}); err == nil {
		t.Fatal("an unknown driver must be rejected")
	}
}

// init:sqlite:begin

func openSQLiteForTest(t *testing.T) *DB {
	t.Helper()

	db, err := Open(context.Background(), config.Database{
		Driver:     config.DriverSQLite,
		SQLitePath: filepath.Join(t.TempDir(), "nested", "test.db"),
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigrations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openSQLiteForTest(t)

	pending := func() int {
		statuses, err := db.MigrationStatus(ctx)
		if err != nil {
			t.Fatalf("MigrationStatus: %v", err)
		}
		count := 0
		for _, status := range statuses {
			if !status.Applied {
				count++
			}
		}
		return count
	}

	total := pending()
	if total == 0 {
		t.Fatal("no embedded migrations found")
	}

	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate must be idempotent: %v", err)
	}
	if got := pending(); got != 0 {
		t.Fatalf("%d migrations still pending after Migrate", got)
	}

	// Every migration must be reversible.
	for range total {
		if err := db.MigrateDown(ctx); err != nil {
			t.Fatalf("MigrateDown: %v", err)
		}
	}
	if got := pending(); got != total {
		t.Fatalf("%d migrations pending after rolling everything back, want %d", got, total)
	}
}

func TestTransactionsAndErrorTranslation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	db := openSQLiteForTest(t)
	if err := db.Migrate(ctx); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	insert := func(ctx context.Context, username string) error {
		_, err := db.Exec(ctx,
			`INSERT INTO users (username, first_name, last_name, password_hash, role, created_at, updated_at)
			 VALUES (?, 'a', 'b', 'hash', 'user', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, username)
		return err
	}
	count := func() int {
		var n int
		if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
			t.Fatalf("count users: %v", err)
		}
		return n
	}

	boom := errors.New("boom")
	err := db.InTx(ctx, func(ctx context.Context) error {
		if err := insert(ctx, "rolled-back"); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) || count() != 0 {
		t.Fatalf("a failed transaction must roll back: err = %v, rows = %d", err, count())
	}

	err = db.InTx(ctx, func(ctx context.Context) error {
		// A nested call joins the outer transaction.
		return db.InTx(ctx, func(ctx context.Context) error { return insert(ctx, "committed") })
	})
	if err != nil || count() != 1 {
		t.Fatalf("a successful transaction must commit: err = %v, rows = %d", err, count())
	}

	if err := insert(ctx, "committed"); !errs.Is(err, errs.CodeConflict) {
		t.Errorf("duplicate key: error = %v, want conflict", err)
	}

	_, err = db.Exec(ctx, `INSERT INTO refresh_tokens (user_id, family_id, token_hash, expires_at, created_at)
		VALUES (999, 'f', 'h', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	if !errs.Is(err, errs.CodeInvalidArgument) {
		t.Errorf("foreign key violation: error = %v, want invalid_argument", err)
	}

	var id int64
	err = db.Translate(db.QueryRow(ctx, `SELECT id FROM users WHERE username = ?`, "nobody").Scan(&id), "user")
	if e, ok := errs.As(err); !ok || e.Code != errs.CodeNotFound || e.Message != "user not found" {
		t.Errorf("missing row: error = %v, want not_found", err)
	}
}

// init:sqlite:end

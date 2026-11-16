# Database

`internal/platform/database` gives repositories one small API that works with PostgreSQL
and SQLite. Queries are written once.

## Drivers

| Driver | Library | Selected with |
|---|---|---|
| `postgres` | [pgx](https://github.com/jackc/pgx) through `database/sql` | `DATABASE_DRIVER=postgres` and `DATABASE_URL` |
| `sqlite` | [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), pure Go, no cgo | `DATABASE_DRIVER=sqlite` and `SQLITE_PATH` |

When both are compiled in, the default is `sqlite` in development and `postgres` in
production.

Each driver is one file, `postgres.go` or `sqlite.go`, that registers itself:

```go
func init() {
	register(config.DriverPostgres, driver{
		open:                openPostgres,
		numbered:            true,
		uniqueViolation:     func(err error) bool { return postgresCode(err) == "23505" },
		foreignKeyViolation: func(err error) bool { return postgresCode(err) == "23503" },
		migrator:            // ...
	})
}
```

Removing a backend is deleting its file and its migrations directory, which is what the
init tool does for `-db postgres` or `-db sqlite`.

### SQLite specifics

- The pool is limited to **one connection**. SQLite allows a single writer, and one
  connection avoids `SQLITE_BUSY` errors.
- Foreign keys, a 5 second busy timeout and WAL journaling are set in the connection
  string, so they apply to every connection.
- The directory of `SQLITE_PATH` is created when missing.

### PostgreSQL specifics

The pool is sized with `DATABASE_MAX_OPEN_CONNS`, `DATABASE_MAX_IDLE_CONNS`,
`DATABASE_CONN_MAX_IDLE_TIME` and `DATABASE_CONN_MAX_LIFETIME`.

## Writing a repository

A repository holds a `*database.DB` and uses four methods:

| Method | Use |
|---|---|
| `Query(ctx, sql, args...)` | Statements that return rows |
| `QueryRow(ctx, sql, args...)` | Statements that return at most one row |
| `Exec(ctx, sql, args...)` | Statements that return none; reports the affected row count |
| `Translate(err, "task")` | Maps a driver error to an application error |

```go
const columns = `id, title, description, status, user_id, version, created_at, updated_at`

func (r *Repository) GetByID(ctx context.Context, id int64) (Task, error) {
	task, err := scan(r.db.QueryRow(ctx, `SELECT `+columns+` FROM tasks WHERE id = ?`, id))
	return task, r.db.Translate(err, "task")
}
```

### Placeholders

Write `?` for every parameter. For PostgreSQL the query is rewritten to `$1`, `$2`... just
before it runs (`DB.Rebind`). The rewrite replaces every `?` in the text, so do not use a
literal question mark inside a query, for example in a string constant or a JSON operator.

### Portable SQL

The two databases share enough SQL for typical CRUD. The template relies on:

- `RETURNING`, to read generated columns after `INSERT` and `UPDATE`.
- `LOWER(column) LIKE ?` for case-insensitive matching, instead of `ILIKE`.
- Timestamps generated in Go, in UTC, truncated to microseconds, and passed as parameters.

When you need something specific to one database, branch on `db.Driver()`.

### Error translation

`Translate(err, resource)` turns driver errors into application errors:

| Driver error | Application error |
|---|---|
| `sql.ErrNoRows` | `not_found`: "task not found" |
| Unique violation | `conflict`: "task already exists" |
| Foreign key violation | `invalid_argument`: "task references a record that does not exist" |
| Anything else | Returned unchanged, which the HTTP layer reports as `internal` |

`Query` and `Exec` apply the same translation with the generic name `record`. A service can
replace the message when it knows more, as the users service does for a duplicate username.

## Transactions

`InTx` runs a function inside a transaction:

```go
err := s.db.InTx(ctx, func(ctx context.Context) error {
	stored, err := s.tokens.findByHash(ctx, hash)
	if err != nil {
		return err
	}
	if err := s.tokens.revoke(ctx, stored.ID, now); err != nil {
		return err
	}
	return s.tokens.create(ctx, /* ... */)
})
```

The transaction travels in the **context** passed to the function. Any query issued through
the same `DB` with that context joins it, so repositories have no separate transactional
methods and do not know whether they run inside one.

- Returning an error rolls back; returning `nil` commits.
- A nested `InTx` joins the outer transaction.
- Use the context you were given. A query made with the outer context runs outside the
  transaction, and with SQLite's single connection it would wait for a connection the
  transaction is holding.

## Optimistic concurrency

Each table has a `version` column. Updates increment it and require the version that was
read:

```sql
UPDATE tasks
SET title = ?, description = ?, status = ?, version = version + 1, updated_at = ?
WHERE id = ? AND version = ?
RETURNING ...
```

When no row matches, the repository returns `conflict`. See
[API conventions](api-conventions.md#concurrent-updates).

## Migrations

Migrations are plain SQL files run by [goose](https://github.com/pressly/goose), one
directory per driver:

```text
migrations/
├── migrations.go          embeds the files
├── postgres/
│   ├── 00001_users.sql
│   └── 00002_tasks.sql
└── sqlite/
    ├── 00001_users.sql
    └── 00002_tasks.sql
```

They are **embedded in the binaries**, so there is no directory to ship or mount.

### Writing one

Create a file with the next number in each driver directory. A file has an `Up` and a
`Down` section:

```sql
-- +goose Up
CREATE TABLE projects (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT        NOT NULL,
    version    BIGINT      NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE projects;
```

The SQLite version of the same table:

```sql
-- +goose Up
CREATE TABLE projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT      NOT NULL,
    version    INTEGER   NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE projects;
```

Declare time columns as `TIMESTAMP` in SQLite: the driver uses the declared type to return
them as `time.Time`.

A unit test applies every migration and then rolls every one back, so a missing or broken
`Down` section fails the build.

### Running them

| Command | Effect |
|---|---|
| `make migrate-up` | Apply every pending migration |
| `make migrate-down` | Roll back the most recent migration |
| `make migrate-status` | List migrations and whether they are applied |

These run `cmd/migrate`, which reads the same configuration as the server. In the
container image the binary is `/app/migrate`:

```sh
docker run --rm --entrypoint /app/migrate \
  -e DATABASE_URL=... -e AUTH_SECRET=... go-rest-template:dev up
```

!!! note

    The default entrypoint of the image is the server, hence `--entrypoint`. The image
    sets `APP_ENV=production` and `cmd/migrate` loads the whole configuration, so it needs
    a valid `AUTH_SECRET` even though it does not use it.

### At startup

With `DATABASE_AUTO_MIGRATE=true`, the default, the server applies pending migrations
before it starts listening. On PostgreSQL goose takes a session-level **advisory lock**
while it migrates, so several replicas starting at the same time do not apply a migration
twice: one migrates, the others wait and find nothing to do.

Set it to `false` to make migrations an explicit step of your deployment.

### Applied versions

goose records applied migrations in the `goose_db_version` table.

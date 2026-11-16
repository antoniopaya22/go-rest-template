# Testing

The template has two kinds of tests: unit tests next to the code they cover, and
end-to-end tests that start the whole application and call it over HTTP.

```sh
make test         # everything, on SQLite
make test-race    # the same with the race detector
make cover        # race detector plus coverage, written to coverage.out
```

## End-to-end tests

`internal/app/apptest` starts the real application: configuration, database, migrations,
every middleware, contract validation and all routes, behind an `httptest.Server`.

```go
package tasks_test

func TestTasksAPI(t *testing.T) {
	t.Parallel()

	apptest.Each(t, func(t *testing.T, env *apptest.Env) {
		_, alice := env.User("alice")

		response := env.Do(http.StatusCreated, apptest.Request{
			Method: http.MethodPost,
			Path:   "/api/v1/tasks",
			Token:  alice,
			Body:   map[string]any{"title": "Write docs", "description": "All of them"},
		})

		var task struct {
			ID     int64  `json:"id"`
			Status string `json:"status"`
		}
		response.Decode(t, &task)
		if task.Status != "pending" {
			t.Errorf("status = %q, want pending", task.Status)
		}
	})
}
```

### The harness

| Function | Does |
|---|---|
| `apptest.Each(t, fn, settings...)` | Runs `fn` once per available database, as parallel subtests named `sqlite` and `postgres`. |
| `apptest.New(t, settings...)` | Starts one application on the first available database. |
| `env.Do(wantStatus, request)` | Performs a request and fails the test when the status differs, printing the body. |
| `env.Login(username, password)` | Returns the token pair of a user. |
| `env.AdminToken()` | Returns an access token of the bootstrap administrator. |
| `env.User(username)` | Registers a regular user and returns its identifier and an access token. |
| `response.Decode(t, &v)` | Unmarshals the body. |
| `response.Code(t)` | Returns the `code` of a problem response. |

`settings` are configuration overrides, for tests that need a different setup:

```go
env := apptest.New(t, map[string]string{
	"RATE_LIMIT_ENABLED":        "true",
	"RATE_LIMIT_LOGIN_REQUESTS": "2",
})
```

A test in a feature package uses an external test package (`package tasks_test`), because
`apptest` imports the application, which imports the feature.

### What the harness changes

Each application starts from the defaults with a few overrides:

| Setting | Value | Why |
|---|---|---|
| `OPENAPI_VALIDATE_RESPONSES` | `true` | Every response is checked against the contract. |
| `RATE_LIMIT_ENABLED` | `false` | Tests issue many requests from one address. |
| `AUTH_BCRYPT_COST` | `10` | The lowest allowed cost, to keep tests fast. |
| `ADMIN_PORT` | `0` | No second listener per test. |
| `BOOTSTRAP_ADMIN_*` | `admin` / `admin-password-123` | Every environment starts with one administrator. |

Logs are discarded.

### Databases

Each test gets its own database, so tests run in parallel and need no cleanup:

- **SQLite:** a file in `t.TempDir()`.
- **PostgreSQL:** a schema created for the test and dropped afterwards. The connection
  string of the application points its `search_path` at that schema.

PostgreSQL is used when `TEST_POSTGRES_DSN` is set, in URL form:

```sh
docker run -d --name test-postgres -p 5432:5432 \
  -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=app_test postgres:17

export TEST_POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/app_test?sslmode=disable'
make test
```

Without the variable, tests run on SQLite only. CI sets it, so every change is tested on
both databases.

!!! note "PostgreSQL-only projects"

    A project generated with `-db postgres` has no SQLite to fall back on. Without
    `TEST_POSTGRES_DSN` its end-to-end tests are **skipped**, not failed. Set the variable
    locally to run them.

## Unit tests

Unit tests cover code whose behaviour does not need the whole application:

| Package | Covered |
|---|---|
| `internal/config` | Defaults, overrides, every validation rule, the generated files |
| `internal/auth` | Hashing, password policy, signing and verifying with HS256 and EdDSA, forged and expired tokens, the middleware |
| `internal/platform/database` | Placeholder rewriting, migrations up and down, transactions, error translation |
| `internal/platform/contract` | Request and response validation, invalid documents |
| `internal/platform/httpx` | Problem responses, JSON decoding, pagination, `If-Match` |
| `internal/platform/middleware` | Request ID, recovery, security headers |
| `internal/platform/errs` | Codes and wrapping |

They are table-driven where there are several cases:

```go
tests := []struct {
	header  string
	wantErr bool
}{
	{"", false},
	{"*", false},
	{`"3"`, false},
	{`"2"`, true},
}
for _, tt := range tests {
	// ...
}
```

Services depend on a `Store` interface declared next to them, so a service can be tested
with an in-memory fake when a rule deserves a focused test. The template itself covers
the services through the end-to-end tests.

## Tests that guard the structure

A few tests exist to catch drift rather than bugs:

| Test | Fails when |
|---|---|
| `TestRoutesMatchContract` | A route is not in the OpenAPI document, or the reverse. |
| `TestGeneratedFilesAreCurrent` | A configuration variable changed and `make generate` was not run. |
| `TestMigrations` | A migration cannot be applied, applied twice, or rolled back. |
| `TestReferenceListsEveryVariableOnce` | A variable is declared twice or has no description. |

## Coverage

```sh
make cover
```

runs `go test -race -covermode=atomic -coverpkg=./internal/... -coverprofile=coverage.out ./...`
and prints the total. `-coverpkg` makes the end-to-end tests count towards the packages
they exercise, not only towards the package they sit in. CI uploads the profile to
Codecov from the job that runs the stable Go release.

```sh
go tool cover -html=coverage.out
```

## In CI

The `ci` workflow runs, on every push and pull request:

| Job | What |
|---|---|
| `lint` | `go mod tidy` and `make generate` leave no diff, `go vet`, golangci-lint |
| `test` | `make cover` on the stable and the previous Go release, with PostgreSQL |
| `vuln` | `govulncheck` |
| `init` | `make test-init`: generates and tests nine projects |
| `image` | Builds the container image and waits for `/health/ready` |
| `meta` | Lints the workflows, the Dockerfile and the Helm chart |

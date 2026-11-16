# Getting started

go-rest-template is a Go REST API that already runs: users, token authentication, an
example resource, migrations, tests and deployment files. You turn it into your own
project with a tool that renames it and deletes the parts you do not want.

## What you need

| Tool | Used for | Required |
|---|---|---|
| Go 1.26 or newer | Building and running everything | Yes |
| `make` | The developer commands | Yes |
| Docker | The container image, the Compose stack, PostgreSQL for tests | No |
| Python 3 | Building this documentation site | No |

Linters and scanners are not installed: the Makefile runs them with
`go run package@version`, so everyone uses the same release.

## The shortest path

```sh
git clone https://github.com/antoniopaya22/go-rest-template billing
cd billing
make init     # answer the questions
make check    # tidy, vet, lint and test what is left
make run      # http://localhost:8080, reference at /docs
```

## Where to go next

- [Create your project](create-your-project.md): every option of the init tool and what it removes.
- [Quickstart](quickstart.md): run the API and call it, from login to an optimistic update.
- [Architecture](../architecture/index.md): how the code is organised and why.
- [Add a resource](../guides/add-a-resource.md): generate your first resource.

## What the template decides for you

| Concern | Decision |
|---|---|
| Router | [chi](https://github.com/go-chi/chi), on top of `net/http` |
| Contract | OpenAPI 3.1 in `api/openapi.yaml`, embedded and enforced with [kin-openapi](https://github.com/getkin/kin-openapi) |
| Errors | RFC 9457 problem responses |
| Lists | Cursor pagination, ascending by identifier |
| Concurrent updates | A `version` column, `ETag` and `If-Match` |
| Authentication | JWT access tokens (HS256 or EdDSA) and rotating refresh tokens |
| Passwords | bcrypt |
| Database | PostgreSQL ([pgx](https://github.com/jackc/pgx)) and SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), pure Go) through `database/sql` |
| Migrations | Embedded SQL files run by [goose](https://github.com/pressly/goose) |
| Configuration | Environment variables, with an optional `.env` file |
| Logging | `log/slog`, text in development and JSON in production |
| Metrics and tracing | Prometheus and OpenTelemetry, both removable |

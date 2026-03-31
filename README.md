# go-rest-template

Modern Go 1.26 REST API template with a small but production-oriented foundation:

- Standard-library-first bootstrap with graceful shutdown.
- `chi` HTTP router with request IDs, timeouts, structured logging and CORS.
- `slog` logging.
- JWT auth with `golang-jwt/jwt/v5`.
- OpenAPI 3.1 contract with request and response validation.
- PostgreSQL-first persistence for production with SQLite retained for local development.
- First-admin bootstrap flow for clean environment provisioning.
- Service and repository layers with explicit `context.Context` propagation.
- Health endpoints, Prometheus metrics and optional pprof.
- Table-driven tests, `go vet`, `golangci-lint` and GitHub Actions CI.

The legacy implementation has been preserved in [old](old).

## Quick start

1. Copy [configs/env.example](configs/env.example) to `.env` or export the variables in your shell.
2. Ensure Go 1.26 is installed and available in your terminal.
3. Run the application:

```powershell
go mod tidy
go run ./cmd/api
```

The API starts on `http://localhost:8080` by default.

Documentation is available at `http://localhost:8080/docs`, and the raw contract at `http://localhost:8080/openapi/openapi.yaml`.

## Main endpoints

- `GET /health/live`
- `GET /health/ready`
- `GET /metrics`
- `POST /api/v1/auth/login`
- `POST /api/v1/users`
- `GET /api/v1/users/me`
- `GET /api/v1/users/{userID}`
- `PATCH /api/v1/users/{userID}`
- `DELETE /api/v1/users/{userID}`
- `GET /api/v1/tasks`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks/{taskID}`
- `PATCH /api/v1/tasks/{taskID}`
- `DELETE /api/v1/tasks/{taskID}`

## Project layout

```text
api/
	openapi.yaml # OpenAPI contract and source of truth for the HTTP surface
build/
	package/     # Container packaging assets
configs/
	env.example  # Environment template for local/dev/prod configuration
cmd/
	api/         # HTTP server entrypoint
	migrate/     # migration command
internal/
	app/         # application runtime wiring
		api/       # API-specific auth, contract, transport, domain and storage packages
	pkg/         # shared internal support packages such as config and error model
migrations/    # backend-specific versioned SQL migrations
old/           # legacy project preserved for reference
test/          # external integration tests and fixtures
```

This repo intentionally uses the community Go layout patterns that make sense for an application template: `cmd`, `internal`, `api`, `configs`, `build/package`, `docs` and `test`. Some ecosystem tools still require fixed locations, so GitHub Actions remains in `.github/workflows` even though CI assets conceptually belong under `build/ci`.

## Configuration

The template reads its configuration from environment variables.

- `APP_ENV`: `development` or `production`
- `HTTP_HOST`: default `0.0.0.0`
- `HTTP_PORT`: default `8080`
- `AUTH_SECRET`: JWT signing key
- `AUTH_TOKEN_TTL`: token lifetime, default `24h`
- `AUTH_BCRYPT_COST`: bcrypt cost, default `12`
- `DATABASE_DRIVER`: `sqlite` or `postgres`
- `DATABASE_URL`: PostgreSQL DSN when `DATABASE_DRIVER=postgres`
- `DATABASE_MIGRATIONS_DIR`: optional custom path for migration files
- `SQLITE_PATH`: SQLite database path when `DATABASE_DRIVER=sqlite`
- `DATABASE_MAX_OPEN_CONNS`, `DATABASE_MAX_IDLE_CONNS`, `DATABASE_CONN_MAX_IDLE_TIME`, `DATABASE_CONN_MAX_LIFETIME`: pool settings for PostgreSQL
- `AUTO_MIGRATE`: applies migrations on startup when `true`
- `BOOTSTRAP_ADMIN_*`: creates the first admin automatically when enabled
- `OPENAPI_SPEC_PATH`: path to the OpenAPI document, default `api/openapi.yaml`
- `OPENAPI_VALIDATE_REQUESTS`, `OPENAPI_VALIDATE_RESPONSES`: toggle contract enforcement
- `OPENAPI_SERVE_DOCUMENTATION`: enables `/docs`
- `LOG_LEVEL`: `DEBUG`, `INFO`, `WARN`, `ERROR`
- `ENABLE_PPROF`: exposes `/debug` when `true`

## Development workflow

```powershell
go mod tidy
go fmt ./...
go vet ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run
go test -race ./...
```

Makefile targets are also available:

```powershell
make tidy
make fmt
make vet
make lint
make integration
make test-race
make build
```

## Docker

```powershell
docker build -f build/package/Dockerfile -t go-rest-template .
docker run -p 8080:8080 -e AUTH_SECRET=super-secret -e DATABASE_URL=postgres://postgres:postgres@host.docker.internal:5432/go_rest_template?sslmode=disable go-rest-template
```

## Migration strategy

Migrations are plain SQL files split by backend under [migrations](migrations). They run automatically at startup when `AUTO_MIGRATE=true`, or manually with:

```powershell
go run ./cmd/migrate
```

## Integration coverage

`go test ./...` now exercises the API end-to-end with SQLite by default. When `TEST_POSTGRES_DSN` is set, the same integration flow is executed against PostgreSQL as well, which is how CI validates both persistence backends.

## Future work

- Split auth, users and tasks into versioned modules if the API grows.
- Add domain events and an outbox if asynchronous workflows appear.
- Add stricter authorization policies if the permission model becomes more complex.


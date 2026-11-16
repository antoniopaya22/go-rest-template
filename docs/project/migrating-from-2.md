# Migrating from 2.x

Version 3 is a rewrite of the core. The HTTP API, the configuration variables, the
database schema handling and the project layout all changed, and nothing in it is backward
compatible with 2.x.

**There is no automatic upgrade.** A project created from 2.x is your own code; it does not
follow the template. This page lists what changed, so that you can decide what to port,
and what you need to do if you deploy version 3 on top of a 2.x database.

## HTTP API

### Errors

Errors moved from a private format to RFC 9457 problem responses.

=== "2.x"

    ```http
    HTTP/1.1 404 Not Found
    Content-Type: application/json

    {"code":"not_found","error":"task not found"}
    ```

=== "3.x"

    ```http
    HTTP/1.1 404 Not Found
    Content-Type: application/problem+json

    {"type":"about:blank","title":"Not Found","status":404,"detail":"task not found","code":"not_found","request_id":"..."}
    ```

- The message moved from `error` to `detail`.
- `code` keeps its meaning. New codes: `precondition_failed` (412) and `rate_limited` (429).
- Validation failures add an `errors` array with the field and the reason.
- The content type is `application/problem+json`.

### Login and tokens

=== "2.x"

    ```json
    {
      "token": {
        "access_token": "...",
        "token_type": "Bearer",
        "expires_at": "2026-10-14T11:20:49Z"
      },
      "user": { "id": 1, "username": "admin", "role": "admin" }
    }
    ```

=== "3.x"

    ```json
    {
      "access_token": "...",
      "refresh_token": "...",
      "token_type": "Bearer",
      "expires_in": 900
    }
    ```

- The response is flat and no longer includes the user. Call `GET /api/v1/users/me`.
- `expires_at` (a timestamp) became `expires_in` (seconds).
- Access tokens last 15 minutes by default instead of 24 hours. Clients must use the new
  `POST /api/v1/auth/refresh` endpoint, and `POST /api/v1/auth/logout` to end a session.
- The claims changed: the user identifier is in `sub`; `user_id` and `username` are gone.
  Tokens issued by 2.x are not accepted.
- A deleted user, or one whose role changed, is affected on the next request, not when the
  token expires.

### Lists

=== "2.x"

    ```json
    [ { "id": 1, "title": "..." }, { "id": 2, "title": "..." } ]
    ```

=== "3.x"

    ```json
    { "data": [ { "id": 1, "title": "..." } ], "next_cursor": "MQ" }
    ```

Lists are wrapped in an envelope and paginated: 20 items by default, 100 at most, with the
`limit` and `cursor` query parameters. A client that read the whole array must now follow
`next_cursor`.

### Other behaviour

| | 2.x | 3.x |
|---|---|---|
| Reading another user's task | 403 | 404 |
| `PATCH` and `DELETE` | Unconditional | Honour `If-Match`; a stale value is 412 |
| Concurrent updates of one row | Last write wins | The second write gets 409 |
| Responses of a single resource | No `ETag` | `ETag` with the row version |
| `POST /api/v1/users` | Always public | Public or administrators only, by `AUTH_REGISTRATION_ENABLED` |
| Unknown path or wrong method | Plain text from the router | Problem response |
| Passwords | Surrounding spaces trimmed before hashing | Used as typed; limited to 72 bytes |
| Rate limiting | None | Per client IP, 429 when exceeded |

!!! warning "Passwords with surrounding spaces"

    2.x trimmed a password before hashing it and compared the untrimmed one at login. 3.x
    does neither. An account whose password was typed with leading or trailing spaces must
    log in without them.

### Operational endpoints

| 2.x | 3.x |
|---|---|
| `GET /openapi/openapi.yaml` | `GET /openapi.yaml` |
| `GET /metrics` on the API port | `GET /metrics` on the admin port, 9090 |
| `/debug/...` on the API port, with `ENABLE_PPROF` | `/debug/pprof/...` on the admin port, with `ADMIN_PPROF_ENABLED` |
| `GET /docs` (ReDoc) | `GET /docs` (Scalar) |
| none | `GET /version`, `GET /.well-known/jwks.json` |

Update your scrape configuration and anything that fetches the OpenAPI document.

## Configuration

### Renamed

| 2.x | 3.x | Note |
|---|---|---|
| `AUTH_TOKEN_TTL` | `AUTH_ACCESS_TOKEN_TTL` | Default 24h became 15m |
| `AUTO_MIGRATE` | `DATABASE_AUTO_MIGRATE` | |
| `OPENAPI_SERVE_DOCUMENTATION` | `OPENAPI_DOCS_ENABLED` | |
| `ENABLE_PPROF` | `ADMIN_PPROF_ENABLED` | Served on the admin port |

### Removed

| 2.x | Why |
|---|---|
| `DATABASE_MIGRATIONS_DIR` | Migrations are embedded in the binary |
| `OPENAPI_SPEC_PATH` | The OpenAPI document is embedded in the binary |

The old names are not read. A leftover variable is ignored silently, so check your
deployment for them.

### Changed

| Variable | Change |
|---|---|
| `OPENAPI_VALIDATE_RESPONSES` | Default was `true`; now `true` in development and `false` in production |
| `AUTH_SECRET` | In production it must also be at least 32 characters |
| `APP_ENV` | Only `development` and `production` are accepted |
| Any invalid value | Reported together with the others at startup; 2.x panicked on the first |

### New

`HTTP_MAX_BODY_BYTES`, `HTTP_CORS_ALLOWED_ORIGINS`, `ADMIN_HOST`, `ADMIN_PORT`,
`AUTH_ALGORITHM`, `AUTH_PRIVATE_KEY_FILE`, `AUTH_REFRESH_TOKEN_TTL`,
`AUTH_REGISTRATION_ENABLED`, `RATE_LIMIT_ENABLED`, `RATE_LIMIT_REQUESTS`,
`RATE_LIMIT_WINDOW`, `RATE_LIMIT_LOGIN_REQUESTS`, `LOG_FORMAT`, `METRICS_ENABLED`,
`OTEL_ENABLED` and `OTEL_SERVICE_NAME`. All have defaults; see the
[configuration reference](../reference/configuration.md).

A `.env` file in the working directory is now read. 2.x documented one but did not load it.

## Database

2.x ran migrations with its own code and recorded them in a `schema_migrations` table.
3.x uses goose, which records them in `goose_db_version`, and its first migration creates
the `users` table. The schema also changed:

- `users` and `tasks` have a new `version` column.
- There is a new `refresh_tokens` table.

**Version 3 cannot start on a 2.x database as it is**: goose finds no record of applied
migrations, tries to create `users` and fails because the table exists. The template does
not ship an upgrade migration. You have two options.

### Option 1: a new database

Start version 3 on an empty database and import the rows of `users` and `tasks`. Password
hashes are bcrypt in both versions and can be copied. This is the simpler path when
downtime is acceptable.

### Option 2: bring the schema forward by hand

Apply the schema changes yourself and tell goose that the first two migrations are already
in place. For PostgreSQL, with the unmodified 2.0.1 schema:

```sql
BEGIN;

ALTER TABLE users ADD COLUMN version BIGINT NOT NULL DEFAULT 1;
ALTER TABLE tasks ADD COLUMN version BIGINT NOT NULL DEFAULT 1;

CREATE TABLE refresh_tokens (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    family_id  TEXT        NOT NULL,
    token_hash TEXT        NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens (family_id);
CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

-- Tell goose that migrations 1 and 2 are already in place.
CREATE TABLE goose_db_version (
    id         INTEGER PRIMARY KEY GENERATED BY DEFAULT AS IDENTITY,
    version_id BIGINT    NOT NULL,
    is_applied BOOLEAN   NOT NULL,
    tstamp     TIMESTAMP NOT NULL DEFAULT now()
);
INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, true), (1, true), (2, true);

DROP TABLE schema_migrations;

COMMIT;
```

This script was run against a database created by the 2.0.1 migration; version 3 then
started on it, reported both migrations as applied and served requests. It has not been
tested with schemas you changed, and there is no equivalent script for SQLite, where
adding the tables to an existing file is rarely worth it over starting fresh.

!!! warning

    Take a backup first, and try the script on a copy. If you added your own migrations to
    a 2.x project, they are not known to version 3 either: port them to the goose format
    with numbers after the ones of the template.

## Project layout

The code moved from layers to feature packages.

| 2.x | 3.x |
|---|---|
| `internal/app/api/domain/users`, `.../tasks` | `internal/users`, `internal/tasks` |
| `internal/app/api/httpapi` (all handlers and the router) | `handler.go` in each feature; the router in `internal/app/routes.go` |
| `internal/app/api/store/postgres`, `.../sqlite` (one repository per database) | `repository.go` in each feature, one implementation for both databases |
| `internal/app/api/auth` | `internal/auth` |
| `internal/app/api/apicontract` | `internal/platform/contract` |
| `internal/pkg/config` | `internal/config` |
| `internal/pkg/errs` | `internal/platform/errs` |
| `build/package/Dockerfile` | `Dockerfile` |
| `deployments/docker-compose.yml` | `deployments/compose.yaml` |
| `scripts/tasks.sh`, `scripts/tasks.ps1` | The `Makefile` runs the commands directly |
| `test/api_integration_test.go` | End-to-end tests in `internal/app` and in each feature, on `internal/app/apptest` |

Other changes that affect how you build and run it:

- Go 1.27 is required.
- The container image contains only the two binaries; `api/` and `migrations/` are no
  longer copied into it.
- `cmd/migrate` takes a subcommand: `up`, `down` or `status`.
- The default branch of the repository is `main`.

## Porting your code

If you have a 2.x project with your own resources, the least painful path is usually to
create a new project from version 3 and move each resource into it:

1. [Create the project](../getting-started/create-your-project.md).
2. For each resource, [generate it](../guides/add-a-resource.md) and move your fields,
   rules and queries into the generated files. Your service logic carries over almost
   unchanged; the two repositories collapse into one, with `?` placeholders.
3. Rewrite your migrations in the goose format.
4. Move your OpenAPI paths into the new document, using the shared `Problem` response and
   the page envelope.

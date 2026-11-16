<div align="center">

# go-rest-template

**A Go REST API template you can cut down to what you need.**

[![CI](https://github.com/antoniopaya22/go-rest-template/actions/workflows/ci.yml/badge.svg)](https://github.com/antoniopaya22/go-rest-template/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/antoniopaya22/go-rest-template?sort=semver)](https://github.com/antoniopaya22/go-rest-template/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/antoniopaya22/go-rest-template?logo=go&logoColor=white&color=00ADD8)](go.mod)
[![codecov](https://codecov.io/gh/antoniopaya22/go-rest-template/graph/badge.svg)](https://codecov.io/gh/antoniopaya22/go-rest-template)
[![CodeQL](https://github.com/antoniopaya22/go-rest-template/actions/workflows/codeql.yml/badge.svg)](https://github.com/antoniopaya22/go-rest-template/actions/workflows/codeql.yml)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/antoniopaya22/go-rest-template/badge)](https://scorecard.dev/viewer/?uri=github.com/antoniopaya22/go-rest-template)
[![License: MIT](https://img.shields.io/github/license/antoniopaya22/go-rest-template)](LICENSE)

[Documentation](https://antoniopaya22.github.io/go-rest-template/) ·
[Quickstart](https://antoniopaya22.github.io/go-rest-template/getting-started/quickstart/) ·
[Changelog](CHANGELOG.md)

</div>

A working service, not a skeleton: authentication, users and an example
resource on PostgreSQL or SQLite, described by an OpenAPI 3.1 contract that is
enforced at runtime. An init tool renames it and removes the parts you do not
want, and CI checks that every combination of options still builds and passes
its tests.

## Start a project

Create a repository with **Use this template** (or clone this one), then:

```sh
make init
```

It asks for your module path and what to keep. The same without questions:

```sh
go run ./tools/init -yes -module github.com/acme/billing -db postgres -example=false
```

| Option | Default | Removes when turned off |
|---|---|---|
| `-db both\|postgres\|sqlite` | `both` | The other driver, its migrations and its dependency |
| `-example` | on | The `tasks` resource and the resource generator |
| `-metrics` | on | Prometheus metrics |
| `-tracing` | on | OpenTelemetry tracing and its dependencies |
| `-ratelimit` | on | Per-IP rate limiting |
| `-observability` | on | The Prometheus, Grafana and Tempo compose profile |
| `-helm` | on | The Helm chart |
| `-devcontainer` | on | The dev container |

The website, the changelog and the init tool itself belong to the template and
are always removed. What is left is your project.

## Run it

```sh
make run
```

The API listens on <http://localhost:8080> with a SQLite file under `data/`.
The interactive reference is at `/docs`. To have an account to sign in with,
create the first administrator at startup:

```sh
BOOTSTRAP_ADMIN_ENABLED=true BOOTSTRAP_ADMIN_USERNAME=admin \
BOOTSTRAP_ADMIN_FIRST_NAME=Ada BOOTSTRAP_ADMIN_LAST_NAME=Lovelace \
BOOTSTRAP_ADMIN_PASSWORD=change-me-now-123 make run
```

```sh
curl -s localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"change-me-now-123"}'
```

`make compose-up` runs it with PostgreSQL instead, and `make obs-up` adds
Prometheus, Grafana and Tempo.

## What is in it

**API**

- OpenAPI 3.1 contract embedded in the binary. Requests that do not match it are rejected before reaching a handler; a test fails when a route and the contract disagree.
- Errors as RFC 9457 problem responses with a stable `code`, per-field details and the request ID.
- Cursor pagination, and optimistic concurrency with `ETag` and `If-Match`.

**Authentication**

- Short-lived access tokens and rotating refresh tokens. Reusing a refresh token revokes every token of that login.
- HS256, or EdDSA with the public key published as a JWKS.
- The caller is reloaded on every request, so deleting a user or changing a role takes effect immediately.

**Data**

- One repository implementation for PostgreSQL and SQLite, written in plain SQL.
- Embedded migrations run by goose, with down migrations and an advisory lock so replicas do not race.
- Transactions that repositories join through the context.

**Operations**

- Health and readiness probes, build information, graceful shutdown that fails readiness first.
- Prometheus metrics and pprof on a separate admin port; optional OpenTelemetry tracing.
- Structured logs with `slog`, security headers, configurable CORS, rate limiting.
- Static distroless image with its own health check, a Helm chart and a compose stack.

**Project**

- Configuration declared once: it validates everything at startup, reads `.env` files and generates its own reference.
- A generator for new resources: `make resource name=project`.
- End-to-end tests against both databases, linting, vulnerability scanning, and releases with signed images, SBOMs and build provenance.

## Layout

```text
api/                OpenAPI contract
cmd/api             the server
cmd/migrate         schema migrations: up, down, status
internal/app        composition root: wiring and routes
internal/auth       tokens, passwords, authentication middleware
internal/users      the users feature: model, rules, storage, endpoints
internal/tasks      the example feature
internal/platform   database, HTTP conventions, middleware, tracing
internal/config     environment configuration
migrations/         SQL migrations per driver
deployments/        compose stack, observability and Helm chart
tools/              init and the resource generator
```

A feature owns everything about itself, depends only on `platform` and `auth`,
and is mounted in one place. Removing one is deleting its directory and two
lines in `internal/app`.

## Development

```sh
make help     # every target
make check    # tidy, vet, lint and tests with the race detector
make dev      # run with hot reload
```

See [CONTRIBUTING.md](CONTRIBUTING.md). The earlier versions are available as
the [`2.0.1`](https://github.com/antoniopaya22/go-rest-template/tree/2.0.1) and
[`1.0.0`](https://github.com/antoniopaya22/go-rest-template/tree/1.0.0) tags;
the documentation covers [migrating from 2.x](https://antoniopaya22.github.io/go-rest-template/project/migrating/).

## License

[MIT](LICENSE) © Antonio Payá González

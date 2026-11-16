# Project layout

```text
.
├── api/
│   ├── api.go                  embeds the document
│   └── openapi.yaml            the HTTP contract
├── cmd/
│   ├── api/                    the server
│   └── migrate/                up, down, status
├── configs/
│   └── env.example             generated: every variable with its default
├── deployments/
│   ├── compose.yaml            API and PostgreSQL, plus an observability profile
│   ├── helm/go-rest-template/  Helm chart
│   └── observability/          Prometheus, Tempo and Grafana configuration
├── docs/                       this site
├── internal/
│   ├── app/                    composition root
│   │   ├── app.go              construction, run, shutdown
│   │   ├── routes.go           middleware and routes
│   │   └── apptest/            end-to-end test harness
│   ├── auth/                   authentication
│   ├── config/                 configuration
│   ├── platform/               shared infrastructure
│   │   ├── contract/
│   │   ├── database/
│   │   ├── errs/
│   │   ├── httpx/
│   │   ├── logging/
│   │   ├── middleware/
│   │   ├── tracing/
│   │   └── version/
│   ├── tasks/                  example feature
│   └── users/                  users feature
├── migrations/
│   ├── migrations.go           embeds the SQL files
│   ├── postgres/
│   └── sqlite/
├── scripts/
│   └── test-init.sh            generates and tests projects
├── tools/
│   ├── init/                   turns the template into your project
│   └── scaffold/               generates a resource
├── Dockerfile
├── Makefile
└── mkdocs.yml
```

## Conventions

**`internal/` holds all application code.** Nothing is meant to be imported by other
modules, so there is no `pkg/` directory.

**One package per feature, four files per package.** `task.go` (model and inputs),
`service.go`, `repository.go` and `handler.go`. The example and every generated resource
also carry their end-to-end test in `api_test.go`; the tests of `users` and `auth` live in
`internal/app`, because every other test depends on them.

**`platform` is for code with no business meaning.** If a package would still make sense
in a different service, it belongs there. If it mentions users or tasks, it does not.

**Embedded, not shipped next to the binary.** The OpenAPI document and the migrations are
compiled in. The container image contains two binaries and nothing else.

**Generated files are committed and checked.** `configs/env.example` and
`docs/reference/configuration.md` are produced by `make generate` from the declarations in
`internal/config/config.go`. A test fails when they are out of date.

## Files at the root

| File | Purpose |
|---|---|
| `Makefile` | Developer commands. `make help` lists them. See [Make targets](../reference/make-targets.md). |
| `Dockerfile` | Multi-stage build of a static, distroless image. |
| `.golangci.yml` | Linter configuration. |
| `lefthook.yml` | Optional git hooks: formatting, vet and lint before a commit, Conventional Commits on the message. |
| `.goreleaser.yaml` | Release binaries and their SBOM. |
| `release-please-config.json`, `.release-please-manifest.json` | Versioning and changelog automation. |
| `.editorconfig` | Indentation and line endings. |

## What the init tool removes

A generated project has no `docs/`, `overrides/`, `mkdocs.yml`, `scripts/` or `tools/init`,
and loses whatever belongs to the options that were turned off. See
[Create your project](../getting-started/create-your-project.md).

# Make targets

`make help` prints this list. Linters and scanners run through `go run package@version`
with the versions pinned at the top of the `Makefile`, so nothing has to be installed.

## Development

| Target | Does |
|---|---|
| `make run` | Runs the API with `go run ./cmd/api`. |
| `make dev` | Runs the API and restarts it when a file changes, using [air](https://github.com/air-verse/air). |
| `make build` | Builds `bin/api` and `bin/migrate`, static, with the version, commit and date linked in. |
| `make generate` | Regenerates `configs/env.example` and `docs/reference/configuration.md` from the configuration declarations. |
| `make resource name=<singular>` | Generates a resource from the tasks example. Optional `plural=<plural>`. See [Add a resource](../guides/add-a-resource.md). |

## Quality

| Target | Does |
|---|---|
| `make fmt` | Formats the code with the golangci-lint formatters (`gofmt`, `goimports`). |
| `make vet` | Runs `go vet ./...`. |
| `make lint` | Runs golangci-lint with `.golangci.yml`. |
| `make vuln` | Runs `govulncheck ./...`. |
| `make tidy` | Runs `go mod tidy`. |
| `make test` | Runs the tests. |
| `make test-race` | Runs the tests with the race detector. |
| `make cover` | Runs the tests with the race detector and coverage across `./internal/...`, writes `coverage.out` and prints the total. |
| `make check` | Runs `tidy`, `vet`, `lint` and `test-race`. Use it before pushing. |

Set `TEST_POSTGRES_DSN` to make the test targets cover PostgreSQL too. See
[Testing](../guides/testing.md).

## Database

| Target | Does |
|---|---|
| `make migrate-up` | Applies pending migrations. |
| `make migrate-down` | Rolls back the most recent migration. |
| `make migrate-status` | Shows which migrations are applied. |

They use the database of the current configuration, which in development is the SQLite
file under `data/`.

## Containers

| Target | Does |
|---|---|
| `make docker-build` | Builds the image as `go-rest-template:dev`. Override with `IMAGE=`. |
| `make compose-up` | Builds and starts the API with PostgreSQL. |
| `make obs-up` | The same plus Prometheus, Grafana and Tempo. |
| `make compose-down` | Stops and removes the stack, observability services included. |

## Template

These exist in the template only; the init tool removes them from a generated project.

| Target | Does |
|---|---|
| `make init` | Turns the template into your project, interactively. See [Create your project](../getting-started/create-your-project.md). |
| `make test-init` | Generates nine projects with different options and builds and tests each. `INIT_LINT=1` lints them too. |
| `make docs-serve` | Serves this site at <http://localhost:8000>. Needs `pip install -r requirements-docs.txt`. |
| `make docs-build` | Builds this site into `site/` in strict mode. |

## Variables

| Variable | Default | Used by |
|---|---|---|
| `GOLANGCI_LINT_VERSION` | `v2.15.0` | `fmt`, `lint` |
| `GOVULNCHECK_VERSION` | `v1.9.0` | `vuln` |
| `AIR_VERSION` | `v1.67.4` | `dev` |
| `VERSION`, `COMMIT`, `DATE` | From git and the clock | `build`, `docker-build` |
| `IMAGE` | `go-rest-template:dev` | `docker-build` |

## Git hooks

`lefthook.yml` defines optional hooks: formatting, `go vet` and lint before a commit, and
a Conventional Commits check on the message. Install them with
[lefthook](https://lefthook.dev):

```sh
lefthook install
```

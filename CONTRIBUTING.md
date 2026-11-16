# Contributing

Thanks for taking the time to contribute. This document explains how to set up
the project, the conventions the repository follows and how changes get merged.

## Ground rules

- Be respectful. This project follows the [Code of Conduct](CODE_OF_CONDUCT.md).
- Report security problems privately, as described in [SECURITY.md](SECURITY.md).
  Do not open a public issue for them.
- For anything larger than a small fix, open an issue first so the approach can
  be agreed before you write the code.

## Development setup

Requirements:

- Go 1.27 or newer
- Docker with the Compose plugin (only for the container targets and for running
  the test suite against PostgreSQL)
- `make`

A [dev container](.devcontainer/devcontainer.json) with all of this is included.

```sh
git clone https://github.com/antoniopaya22/go-rest-template
cd go-rest-template
make help     # list every target
make run      # start the API on http://localhost:8080 with SQLite
```

`make dev` starts the API with hot reload. The admin listener (metrics and
pprof) is on port 9090.

## Everyday commands

| Command | What it does |
|---|---|
| `make check` | What CI runs on the code: tidy, vet, lint and tests with the race detector |
| `make test` | Unit and integration tests against SQLite |
| `make test-race` | The same with the race detector |
| `make cover` | Tests with coverage, written to `coverage.out` |
| `make lint` | `golangci-lint` with the repository configuration |
| `make fmt` | Format the code |
| `make vuln` | `govulncheck` |
| `make generate` | Regenerate `configs/env.example` and the configuration reference |
| `make compose-up` / `make compose-down` | API and PostgreSQL in containers |
| `make obs-up` | The same plus Prometheus, Grafana and Tempo |

To run the tests against PostgreSQL as well, point `TEST_POSTGRES_DSN` at a
database:

```sh
export TEST_POSTGRES_DSN='postgres://postgres:postgres@localhost:5432/go_rest_template_test?sslmode=disable'
make test
```

### Generated files

`configs/env.example` and `docs/reference/configuration.md` are generated from
the configuration code. After adding or changing a configuration variable, run
`make generate` and commit the result. CI fails when they are out of date.

### Git hooks

The repository ships a [lefthook](https://lefthook.dev) configuration that
checks formatting, runs `go vet` and the linter on staged files, and validates
the commit message. It is optional but recommended:

```sh
go install github.com/evilmartians/lefthook@latest
lefthook install
```

## Adding a resource

A resource is a domain package, its repositories, its HTTP handlers, a
migration and its OpenAPI paths. The scaffolding is generated:

```sh
make resource name=project
```

Then:

1. Fill in the fields in the generated model, migration and OpenAPI schema.
2. Adjust the business rules in the service and the authorization of each route.
3. Run `make check`.

The OpenAPI document under `api/` is the source of truth for the HTTP surface.
Requests are validated against it, and a test fails when a route exists in the
router but not in the contract or the other way round.

## Template initialization

`make init` turns a copy of this repository into a new project (module path,
application name, database backends, optional modules). When you change the
layout, or add an optional module, update the init tool as well and run:

```sh
make test-init
```

It generates projects for several combinations of options and builds and tests
each one.

## Commit messages

Commits follow [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<optional scope>): <summary in the imperative>

<optional body explaining why>
```

Types in use: `feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`,
`chore`, `revert`. A breaking change is marked with `!` after the type or scope
and a `BREAKING CHANGE:` footer.

The release notes and the version number are derived from these messages, so
the type matters: `feat` produces a minor release, `fix` and `perf` a patch
release, and a breaking change a major one.

## Pull requests

1. Fork the repository and create a branch from `main`.
2. Make the change, with tests for new behaviour.
3. Run `make check`.
4. Open the pull request against `main` and fill in the template.

Pull requests are squash-merged, so the pull request title must itself be a
valid Conventional Commit message. Keep each pull request focused on one
change; unrelated cleanups are easier to review on their own.

## Releases

Releases are automated. Merging to `main` updates a release pull request that
collects the changelog; merging that pull request tags the version, publishes
the binaries and pushes the signed container image to GHCR.

## License

By contributing you agree that your contributions are licensed under the
[MIT License](LICENSE).

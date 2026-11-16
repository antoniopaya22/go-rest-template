# Create your project

The repository is a GitHub template and a working application at the same time. Creating
a project takes two steps: copy it, then run the init tool.

## 1. Copy the template

=== "GitHub"

    Press **Use this template** on the
    [repository page](https://github.com/antoniopaya22/go-rest-template), create your
    repository and clone it. Its history starts empty.

=== "Clone"

    ```sh
    git clone https://github.com/antoniopaya22/go-rest-template billing
    cd billing
    ```

    The clone carries the history of the template. Start a new one after the init tool
    has run:

    ```sh
    rm -rf .git && git init
    ```

## 2. Run the init tool

```sh
make init
```

`make init` runs `go run ./tools/init`, which asks for the module path, the application
name and each option, with the default in brackets. To run it without questions, pass
`-yes` and the flags:

```sh
go run ./tools/init -yes -module github.com/acme/billing -db postgres -tracing=false
```

```text
billing is ready.

  module   github.com/acme/billing
  kept     dev container, example resource, helm chart, metrics, postgres, rate limiting
  removed  observability stack, sqlite, tracing
```

Then check the result:

```sh
make check
```

### Options

| Flag | Default | Effect |
|---|---|---|
| `-module` | none, required | Go module path of the new project, such as `github.com/acme/billing`. |
| `-name` | last element of the module path | Application name: lowercase letters, digits and dashes. Used for the service name, the image and the chart. |
| `-db` | `both` | `both`, `postgres` or `sqlite`. The other driver, its migrations and its Go dependency are deleted. |
| `-example` | `true` | Keep the `tasks` resource and the resource generator in `tools/scaffold`. |
| `-metrics` | `true` | Keep Prometheus metrics. |
| `-tracing` | `true` | Keep OpenTelemetry tracing. |
| `-ratelimit` | `true` | Keep rate limiting. |
| `-observability` | `true` | Keep the Prometheus, Grafana and Tempo profile of the Compose file. |
| `-helm` | `true` | Keep the Helm chart. |
| `-devcontainer` | `true` | Keep the dev container. |
| `-dir` | `.` | Directory of the project to initialize. |
| `-yes` | `false` | Do not ask questions. |

Some options depend on others. The tool applies these rules without asking:

- `-db sqlite` removes the Compose file and the Helm chart, because both assume PostgreSQL.
- The observability profile is kept only with PostgreSQL, metrics and tracing all kept.

!!! note "The generator needs the example"

    `make resource` clones the `tasks` package, so `-example=false` removes the generator
    together with the example. If you want both a clean project and the generator, keep
    the example, generate your first resource and then delete `internal/tasks` yourself.

### What it does

1. Deletes the files owned by the options you turned off.
2. Deletes what belongs to the template itself (see below).
3. Applies the marker comments to every text file.
4. Replaces the module path, the repository name and the application name everywhere,
   and renames the chart directory.
5. Resets the release configuration to version `0.0.0`.
6. Writes a new `README.md`.
7. Runs `go mod tidy`, which drops the dependencies of removed options.
8. Regenerates `configs/env.example`, which then lists only the variables that are left.

It refuses to run twice: once the module path is no longer the one of the template, the
project counts as initialized.

### What is always removed

These describe or serve the template, not your project:

- `tools/init`, `scripts/` and the `init` job of the CI workflow
- `docs/`, `overrides/`, `mkdocs.yml`, `requirements-docs.txt` and the docs workflow
- `CHANGELOG.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`
- `.github/ISSUE_TEMPLATE`, `.github/CODEOWNERS`, the Scorecard workflow
- `deployments/README.md`

Kept: the CI, CodeQL and release workflows (without the jobs that test the template),
Dependabot, the pull request template, `lefthook.yml`, `.editorconfig`, `.golangci.yml`
and `.goreleaser.yaml`.

!!! warning "License"

    `LICENSE` is kept. It is the MIT license of the template, with its copyright line.
    Add your own copyright line or replace the file with the license of your project.

## Marker comments

The init tool knows nothing about the code it removes. Optional code is delimited with
comments, and the tool only reads those. A marker at the end of a line covers that line;
a `begin`/`end` pair covers a block. This is `internal/app/routes.go` in the template:

```go
r.Use(middleware.Logger(a.logger))
r.Use(middleware.Recover)
// init:metrics:begin
if a.cfg.Metrics.Enabled {
	r.Use(middleware.Metrics(a.registry))
}
// init:metrics:end
r.Use(middleware.SecurityHeaders)
```

```go
a.auth.Routes(r, loginLimiter)
a.users.Routes(r, a.authn)
a.tasks.Routes(r, a.authn) // init:example
```

With `-metrics=false` the block disappears. With metrics kept, only the two marker lines
disappear. Either way the generated project contains no markers. The same syntax works
with `#` comments in YAML and Makefiles.

| Marker | Removed when |
|---|---|
| `init:example` | `-example=false` |
| `init:metrics` | `-metrics=false` |
| `init:otel` | `-tracing=false` |
| `init:ratelimit` | `-ratelimit=false` |
| `init:sqlite`, `init:postgres` | the driver is not selected by `-db` |
| `init:compose` | `-db sqlite` |
| `init:observability` | the observability profile is not kept |
| `init:helm` | the chart is not kept |
| `init:template`, `init:site` | always |

The `scaffold:` comments in `api/openapi.yaml` and `internal/app` belong to the resource
generator. They stay while the example stays and are removed with it.

## How this is tested

`make test-init`, which CI runs on every change, generates nine projects:

| Case | Flags |
|---|---|
| defaults | none |
| postgres-only | `-db postgres` |
| sqlite-only | `-db sqlite` |
| bare | `-db sqlite -example=false -metrics=false -tracing=false -ratelimit=false -devcontainer=false` |
| no-example | `-example=false` |
| no-metrics | `-metrics=false` |
| no-tracing | `-tracing=false` |
| no-ratelimit | `-ratelimit=false` |
| no-extras | `-helm=false -observability=false -devcontainer=false` |

For each one it checks that no file still mentions the template repository or carries an
`init:` marker, that `gofmt` has nothing to change, and that `go build`, `go vet` and
`go test` pass. Projects that keep the example also run the generator and test the new
resource. Set `INIT_LINT=1` to lint each generated project as well; this is off by
default because it is slow.

!!! note "PostgreSQL-only projects"

    Without SQLite the end-to-end tests need a database. They are skipped unless
    `TEST_POSTGRES_DSN` is set. See [Testing](../guides/testing.md).

## After the init tool

- Add `CONTRIBUTING.md` and `SECURITY.md` that describe your project.
- Review `LICENSE`.
- The release workflow pushes the image to `ghcr.io/<owner>/<repository>` of the GitHub
  repository it runs in. Check `.github/workflows/release.yml` and `.goreleaser.yaml` if
  you release somewhere else.
- A generated project is independent of the template. There is no mechanism to pull in
  later changes; read the [changelog](../project/changelog.md) and port what you want.

# Configuration

The application is configured with environment variables. The complete list is in the
[configuration reference](../reference/configuration.md), which is generated from the
code.

## Where values come from

1. The process environment.
2. A `.env` file in the working directory, if there is one.
3. The default declared in the code.

A variable set in the real environment wins over the file. A variable that is set but
empty counts as not set. Copy `configs/env.example` to `.env` to start from every variable
and its default; `.env` is ignored by git.

## Errors are reported together

Loading never panics and never stops at the first problem. Every unparsable value and
every broken rule is collected, and the process exits with the whole list:

```sh
HTTP_PORT=eighty LOG_LEVEL=loud AUTH_BCRYPT_COST=4 make run
```

```text
error: invalid configuration:
  - HTTP_PORT: "eighty" is not an integer
  - LOG_LEVEL: "loud" is not a log level
  - AUTH_BCRYPT_COST: must be between 10 and 16
```

## Development and production

`APP_ENV` is `development` or `production`. It changes defaults and tightens validation:

| | development | production |
|---|---|---|
| `DATABASE_DRIVER` default | `sqlite` | `postgres` |
| `LOG_FORMAT=auto` resolves to | `text` | `json` |
| `OPENAPI_VALIDATE_RESPONSES` default | `true` | `false` |
| Default `AUTH_SECRET` | Accepted | Refused |
| `AUTH_SECRET` shorter than 32 characters | Accepted | Refused |

The container image sets `APP_ENV=production`.

## How variables are declared

Every variable is declared once, in `load` in `internal/config/config.go`, with its name,
default and description:

```go
l.section("HTTP server")
cfg.HTTP = HTTP{
	Host: l.str("HTTP_HOST", "0.0.0.0", "Interface the API listens on."),
	Port: l.integer("HTTP_PORT", 8080, "Port the API listens on."),
	ReadTimeout: l.duration("HTTP_READ_TIMEOUT", 5*time.Second, "Maximum time to read a request, including the body."),
	// ...
}
```

The loader has one method per type:

| Method | Accepts |
|---|---|
| `str` | Any text |
| `integer` | A base-10 integer |
| `boolean` | Whatever `strconv.ParseBool` accepts: `true`, `false`, `1`, `0`... |
| `duration` | A Go duration such as `30s`, `5m` or `1h30m` |
| `enum` | One of a fixed list, case-insensitive |
| `list` | Comma-separated values |
| `level` | `DEBUG`, `INFO`, `WARN` or `ERROR` |

Because the declaration carries the description, the same code produces the documentation.
`config.Reference()` runs `load` with an empty environment and returns every variable it
asked for.

Rules that involve more than one variable are in `validate`, in the same file.

## Adding a variable

**1. Add a field** to the relevant struct, or a new struct to `Config`:

```go
type HTTP struct {
	// ...
	MaxHeaderBytes int
}
```

**2. Declare it** in `load`:

```go
MaxHeaderBytes: l.integer("HTTP_MAX_HEADER_BYTES", 1<<20, "Maximum size of the request headers in bytes."),
```

**3. Validate it** in `validate`, if it has constraints:

```go
if cfg.HTTP.MaxHeaderBytes < 1 {
	fail("HTTP_MAX_HEADER_BYTES: must be greater than zero")
}
```

**4. Regenerate** the example file and the reference:

```sh
make generate
```

**5. Use it** where the component is built, in `internal/app/app.go`.

`TestGeneratedFilesAreCurrent` compares `configs/env.example` and
`docs/reference/configuration.md` with what the code would generate, and CI runs
`make generate` and fails on a diff. Forgetting step 4 does not go unnoticed.

## Configuration in tests

Tests build a configuration from a map, with defaults for everything else:

```go
cfg, err := config.FromMap(map[string]string{
	"HTTP_PORT":    "9000",
	"DATABASE_URL": "postgres://localhost/app",
})
```

`config.FromMap` ignores the process environment and any `.env` file, so tests do not
depend on the machine they run on.

## Secrets

`AUTH_SECRET`, `DATABASE_URL` and `BOOTSTRAP_ADMIN_PASSWORD` are secrets. Provide them
through your platform's secret mechanism and keep them out of images and of `.env` files
that get committed. For EdDSA, `AUTH_PRIVATE_KEY_FILE` points at a mounted file. See the
[production checklist](../deployment/production-checklist.md).

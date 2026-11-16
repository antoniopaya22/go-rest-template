# Reference

| Page | Content |
|---|---|
| [Configuration](configuration.md) | Every environment variable, its default and what it does. Generated from the code. |
| [HTTP API](api.md) | The OpenAPI document of the template, rendered. |
| [Make targets](make-targets.md) | The developer commands. |

## Command line

`cmd/api`, built as `bin/api` and shipped as `/app/server`:

| Flag | Effect |
|---|---|
| none | Run the server |
| `-version` | Print the version, commit, build date and Go version, then exit |
| `-healthcheck` | Request `/health/live` on `127.0.0.1:$HTTP_PORT`; exit 0 when it answers 200 |
| `-print-config env` | Print an example environment file listing every variable |
| `-print-config markdown` | Print the configuration reference as Markdown |

`cmd/migrate`, built as `bin/migrate` and shipped as `/app/migrate`:

| Argument | Effect |
|---|---|
| `up` (default) | Apply every pending migration |
| `down` | Roll back the most recent migration |
| `status` | List migrations and whether they are applied |

Both read the same configuration.

## Endpoints outside the API

| Endpoint | Port | Description |
|---|---|---|
| `GET /health/live` | 8080, 9090 | Liveness |
| `GET /health/ready` | 8080 | Readiness |
| `GET /version` | 8080 | Build metadata |
| `GET /openapi.yaml` | 8080 | The OpenAPI document |
| `GET /docs` | 8080 | Interactive reference, unless `OPENAPI_DOCS_ENABLED=false` |
| `GET /.well-known/jwks.json` | 8080 | Public signing key, with `AUTH_ALGORITHM=EdDSA` only |
| `GET /metrics` | 9090 | Prometheus metrics, when `METRICS_ENABLED=true` |
| `GET /debug/pprof/` | 9090 | Profiling, when `ADMIN_PPROF_ENABLED=true` |

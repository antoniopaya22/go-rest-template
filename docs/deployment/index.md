# Deployment

The service is two static binaries, `api` and `migrate`, with the OpenAPI document and the
migrations embedded. It needs a database and environment variables, nothing else.

| Page | Covers |
|---|---|
| [Docker](docker.md) | The container image: what is in it, how to build and run it. |
| [Compose](compose.md) | The API with PostgreSQL on one machine, with an optional observability stack. |
| [Kubernetes](kubernetes.md) | The Helm chart. |
| [Production checklist](production-checklist.md) | What to set and verify before real traffic. |

## Ports

| Port | Serves | Expose |
|---|---|---|
| 8080 | The API, `/health/*`, `/version`, `/docs`, `/openapi.yaml` | Publicly |
| 9090 | `/metrics`, `/debug/pprof/` and `/health/live` | Inside your network only |

## Health

| Endpoint | Returns 200 when | Use as |
|---|---|---|
| `GET /health/live` | The process is running | Liveness probe |
| `GET /health/ready` | The database answers a ping within 2 seconds and the process is not shutting down | Readiness probe |

On `SIGTERM` readiness fails first, then the servers stop accepting connections and wait
for in-flight requests up to `APP_SHUTDOWN_TIMEOUT` (10 seconds by default).

## Releases of the template

The repository publishes its own releases: binaries with checksums and an SBOM on GitHub,
and a multi-architecture image at `ghcr.io/antoniopaya22/go-rest-template`, signed with
cosign and with a build provenance attestation. A generated project keeps the same
workflow and publishes under its own repository.

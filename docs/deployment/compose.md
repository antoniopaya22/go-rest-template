# Compose

`deployments/compose.yaml` runs the API with PostgreSQL. It is meant for local use and for
trying the production image; its default credentials are not secrets.

```sh
make compose-up      # build the image, start PostgreSQL and the API
make compose-down    # stop and remove everything, observability services included
```

| Service | Address |
|---|---|
| API | <http://localhost:8080> |
| Admin (metrics) | <http://localhost:9090/metrics> |
| PostgreSQL | `localhost:5432`, database `go_rest_template`, user and password `postgres` |

The API waits for PostgreSQL to be healthy, applies the migrations and creates an
administrator: `admin` with the password `change-me-now-123`.

```sh
curl -s localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"change-me-now-123"}'
```

## Settings

Every value in the file has a default and can be overridden from the shell or from a
`.env` file placed next to `compose.yaml`:

| Variable | Default | |
|---|---|---|
| `API_PORT` | `8080` | Host port of the API |
| `ADMIN_PORT` | `9090` | Host port of the admin server |
| `POSTGRES_PORT` | `5432` | Host port of PostgreSQL |
| `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` | `go_rest_template`, `postgres`, `postgres` | Database credentials |
| `APP_ENV` | `development` | |
| `AUTH_SECRET` | `local-dev-secret-change-me` | |
| `BOOTSTRAP_ADMIN_*` | `admin` / `change-me-now-123` | The first administrator |
| `LOG_LEVEL`, `LOG_FORMAT` | `INFO`, `json` | |

```sh
API_PORT=8081 LOG_LEVEL=DEBUG make compose-up
```

To set a variable the file does not list, add it under `services.api.environment`.

## Observability profile

```sh
make obs-up                      # adds Prometheus, Grafana and Tempo
OTEL_ENABLED=true make obs-up    # and sends traces to Tempo
```

| Service | Address |
|---|---|
| Grafana | <http://localhost:3000>, anonymous viewer |
| Prometheus | <http://localhost:9091> |
| Tempo | OTLP/HTTP on `localhost:4318` |

Host ports can be changed with `GRAFANA_PORT`, `PROMETHEUS_PORT` and `TEMPO_OTLP_PORT`.
See [Observability](../guides/observability.md#the-local-stack).

## Data

PostgreSQL, Prometheus and Tempo keep their data in named volumes, which survive
`make compose-down`. To start from an empty database:

```sh
docker compose -f deployments/compose.yaml --profile "*" down --volumes
```

## Using it beyond a laptop

The file is a reasonable starting point for a single-host deployment, after these changes:

- Set `APP_ENV=production`, a real `AUTH_SECRET` of at least 32 characters and real
  database credentials.
- Stop publishing the PostgreSQL and admin ports.
- Put a reverse proxy with TLS in front of the API and set `HTTP_TRUSTED_PROXIES` to the
  number of proxies.
- Replace the bootstrap administrator password, or disable the bootstrap once the
  administrator exists.

See the [production checklist](production-checklist.md).

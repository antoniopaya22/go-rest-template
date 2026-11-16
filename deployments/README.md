# Deployments

Runnable deployment assets for the API.

| Path | What it is |
|---|---|
| [`compose.yaml`](compose.yaml) | Local stack: API and PostgreSQL, plus an optional observability profile |
| [`observability/`](observability) | Prometheus, Tempo and Grafana configuration used by that profile |
| [`helm/go-rest-template`](helm/go-rest-template) | Helm chart for Kubernetes |

## Docker Compose

```sh
make compose-up     # API on :8080, admin listener on :9090, PostgreSQL on :5432
make compose-down
```

The API image is built from the [`Dockerfile`](../Dockerfile) at the repository
root. The stack creates a first administrator (`admin` / `change-me-now-123`)
so the API is usable straight away; these defaults are for local use only.

Every setting has a default and can be overridden from the shell or from a
`.env` file next to `compose.yaml`:

```sh
API_PORT=8081 LOG_LEVEL=DEBUG make compose-up
```

The container health check runs the server binary itself
(`/app/server -healthcheck`), because the distroless image has no shell or
`curl`.

### Observability profile

```sh
make obs-up
```

This starts the same stack with three more services:

| Service | URL | Purpose |
|---|---|---|
| Grafana | <http://localhost:3000> | Dashboards, no login required (anonymous viewer) |
| Prometheus | <http://localhost:9091> | Scrapes `api:9090/metrics` every 15 seconds |
| Tempo | OTLP/HTTP on `localhost:4318` | Trace storage, queried from Grafana |

Grafana opens on the provisioned **go-rest-template API** dashboard: request
rate by route, responses by status, error rate, p50/p95/p99 latency, in-flight
requests, goroutines and memory. The dashboard is a file,
[`observability/grafana/dashboards/api.json`](observability/grafana/dashboards/api.json);
edit it there to keep changes.

Tracing is off by default. To send traces to Tempo, start the stack with
`OTEL_ENABLED=true`:

```sh
OTEL_ENABLED=true make obs-up
```

Traces then appear in Grafana under **Explore**, with the Tempo data source.

## Kubernetes (Helm)

```sh
helm upgrade --install api deployments/helm/go-rest-template \
  --namespace api --create-namespace \
  --set secrets.values.AUTH_SECRET="$(openssl rand -base64 48)" \
  --set secrets.values.DATABASE_URL='postgres://user:password@postgres:5432/app?sslmode=require'
```

The chart deploys the image published at
`ghcr.io/antoniopaya22/go-rest-template`; set `image.repository` and
`image.tag` for your own build.

What it sets up:

- A Deployment with liveness, readiness and startup probes on `/health/live`
  and `/health/ready`, and a restricted security context: non-root, read-only
  root filesystem, all capabilities dropped, `RuntimeDefault` seccomp profile.
- A Service with two ports: `http` for the API and `admin` for metrics. The
  Ingress only ever routes to `http`.
- Configuration as environment variables: `config` becomes a ConfigMap and
  `secrets.values` a Secret. A change to either rolls the pods.
- Optional pieces, each behind its own `enabled` flag: Ingress,
  HorizontalPodAutoscaler, PodDisruptionBudget, NetworkPolicy and a
  ServiceMonitor for the Prometheus Operator.

### Secrets

Passing secrets with `--set` is fine for a first try. For anything else, create
the Secret yourself (or with External Secrets, Sealed Secrets and similar) with
environment variable names as keys, and reference it:

```yaml
secrets:
  create: false
  existingSecret: api-secrets
```

### Migrations

The image contains only the API server, so the chart has no migration job. The
server applies pending migrations at startup (`config.DATABASE_AUTO_MIGRATE`,
on by default). To run them as a separate step of your pipeline instead, set it
to `"false"` and run the `migrate up` command from a release binary or from
source against the same database before rolling out.

### Configuration

Any variable from the configuration reference can be added under `config`:

```yaml
config:
  HTTP_CORS_ALLOWED_ORIGINS: https://app.example.com
  HTTP_TRUSTED_PROXIES: "1"
  RATE_LIMIT_REQUESTS: "300"
```

Set `HTTP_TRUSTED_PROXIES` to the number of proxies in front of the pods
(usually `1` behind an ingress controller) so client addresses, and therefore
rate limits, are derived correctly.

See [`values.yaml`](helm/go-rest-template/values.yaml) for every option.

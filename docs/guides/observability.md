# Observability

The service produces structured logs, Prometheus metrics and, when enabled, OpenTelemetry
traces. Metrics and profiling are served on a separate port.

## Logs

Logging uses `log/slog`. Records go to standard output.

| Variable | Default | Effect |
|---|---|---|
| `LOG_LEVEL` | `INFO` | Minimum level: `DEBUG`, `INFO`, `WARN`, `ERROR` |
| `LOG_FORMAT` | `auto` | `text`, `json`, or `auto`: text in development, JSON in production |

Each request produces one record when it ends:

```json
{"time":"2026-11-16T14:36:34.44Z","level":"INFO","msg":"http request","request_id":"Z2TB2MN7NK7LHMDDPHB6TJCRRZ","method":"POST","path":"/api/v1/tasks","route":"/api/v1/tasks","status":201,"bytes":175,"duration":1290000,"client_ip":"203.0.113.7"}
```

| Field | Content |
|---|---|
| `request_id` | The identifier of the request, also returned in `X-Request-ID` and in problem responses |
| `route` | The route template, such as `/api/v1/tasks/{taskID}`; `unmatched` for a 404 |
| `status`, `bytes` | The response status and body size |
| `duration` | Time to serve the request, in nanoseconds in JSON |
| `client_ip` | The caller's address, resolved according to `HTTP_TRUSTED_PROXIES` |

The level follows the status: `INFO` below 400, `WARN` for 4xx, `ERROR` for 5xx.

### Logging from your code

The request logger is in the context and already carries `request_id`:

```go
logger := logging.FromContext(r.Context())
logger.InfoContext(r.Context(), "task archived", "task_id", task.ID)
```

Use it in handlers and services so that every record of one request can be found by its
identifier. Outside a request, `logging.FromContext` returns the default logger.

Errors do not need to be logged where they occur. Return them: `httpx.WriteError` logs
anything that becomes a 5xx, with the cause, and logs rejections at `DEBUG`.

### Following a request

A client that receives an error has a `request_id` in the body and in the `X-Request-ID`
header. Search the logs for it. A caller can also send its own `X-Request-ID`; it is kept
when it is at most 64 characters of letters, digits, `-`, `_` and `.`, which lets one
identifier follow a call across services.

## The admin port

A second HTTP server listens on `ADMIN_HOST:ADMIN_PORT`, `0.0.0.0:9090` by default.

| Path | Served when |
|---|---|
| `/health/live` | Always |
| `/metrics` | `METRICS_ENABLED=true` |
| `/debug/pprof/` | `ADMIN_PPROF_ENABLED=true` |

It has **no authentication**. It exists so that metrics and profiles never share a port
with the public API: publish 8080, keep 9090 inside your network. `ADMIN_PORT=0` disables
the server.

## Metrics

| Metric | Type | Labels |
|---|---|---|
| `http_requests_total` | Counter | `method`, `route`, `status` |
| `http_request_duration_seconds` | Histogram | `method`, `route`, `status` |
| `http_requests_in_flight` | Gauge | none |

plus the Go runtime and process collectors (`go_*`, `process_*`).

`route` is the route **template**, not the path, so `/api/v1/tasks/7` and
`/api/v1/tasks/8` share one series. Requests that match no route are labelled `unmatched`.
The histogram buckets run from 5 ms to 10 s.

Useful queries:

```promql
# request rate by route
sum by (route) (rate(http_requests_total[5m]))

# share of 5xx responses
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# 95th percentile latency by route
histogram_quantile(0.95, sum by (le, route) (rate(http_request_duration_seconds_bucket[5m])))
```

The metrics use their own registry, created in `internal/app/app.go`. To add a metric,
register it there and pass it to the component that updates it.

## Profiling

Set `ADMIN_PPROF_ENABLED=true` to expose `net/http/pprof` on the admin port:

```sh
go tool pprof http://localhost:9090/debug/pprof/profile?seconds=30
go tool pprof http://localhost:9090/debug/pprof/heap
```

It is off by default.

## Tracing

Tracing is off by default. Turn it on and point the exporter at an OTLP/HTTP endpoint:

```sh
OTEL_ENABLED=true
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=billing        # defaults to APP_NAME
```

The exporter reads the standard `OTEL_EXPORTER_OTLP_*` variables, so headers, timeouts and
TLS are configured the usual OpenTelemetry way.

What is traced:

- One server span per request, named `METHOD /path`, created by `otelhttp`.
- The incoming W3C `traceparent` and `baggage` headers are honoured, so the service joins
  an existing trace.

What is not traced: database queries. There is no SQL instrumentation; add spans in your
repositories, or wrap the driver, if you need them. To create a span in your own code:

```go
ctx, span := otel.Tracer("billing").Start(ctx, "archive task")
defer span.End()
```

Pending spans are flushed on shutdown.

## The local stack

The Compose file has an `observability` profile with Prometheus, Grafana and Tempo:

```sh
make obs-up                    # API, PostgreSQL, Prometheus, Grafana, Tempo
OTEL_ENABLED=true make obs-up  # the same, with the API sending traces to Tempo
make compose-down
```

| Service | Address | Notes |
|---|---|---|
| API | <http://localhost:8080> | |
| Admin | <http://localhost:9090/metrics> | |
| Grafana | <http://localhost:3000> | Anonymous viewer, opens on the API dashboard |
| Prometheus | <http://localhost:9091> | Scrapes `api:9090` every 15 seconds |
| Tempo | `localhost:4318` | OTLP/HTTP receiver |

Grafana is provisioned with both data sources and a dashboard, `go-rest-template API`, with the request
rate by route, the error rate, latency percentiles, in-flight requests, goroutines and
memory. The files are under `deployments/observability`.

## In Kubernetes

The Helm chart names the admin port `admin` on the pod and the Service. With the Prometheus
Operator, set `serviceMonitor.enabled=true` to create a `ServiceMonitor` for it. See
[Kubernetes](../deployment/kubernetes.md).

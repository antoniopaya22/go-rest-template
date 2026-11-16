# Production checklist

What to set and verify before the service takes real traffic. Most items are one
environment variable.

## Required

- [ ] **`APP_ENV=production`.** The container image sets it. It selects JSON logs and
      PostgreSQL by default, turns response validation off and enables the checks below.
- [ ] **`AUTH_SECRET` is random and at least 32 characters** (with HS256). The server
      refuses to start in production with the default value or a shorter one.
      Generate one with `openssl rand -base64 48`.
- [ ] **`DATABASE_URL` uses TLS** where the network is not trusted, for example
      `sslmode=require` or `verify-full`.
- [ ] **Secrets come from a secret store**, not from the image, the repository or a
      committed `.env` file: `AUTH_SECRET`, `DATABASE_URL`, `BOOTSTRAP_ADMIN_PASSWORD`, and
      the key file for EdDSA.

## Network

- [ ] **The admin port is not public.** Port 9090 serves metrics, and profiles when
      enabled, without authentication. Publish 8080 only. Bind the admin server to a
      private interface with `ADMIN_HOST`, restrict it with a network policy, or disable it
      with `ADMIN_PORT=0`.
- [ ] **`HTTP_CORS_ALLOWED_ORIGINS` lists your origins.** The default `*` lets any site
      call the API from a browser. Set explicit origins, comma-separated.
- [ ] **`HTTP_TRUSTED_PROXIES` matches your topology.** It is the number of reverse proxies
      in front of the service. With `0` the client address is the TCP peer, which behind a
      proxy is the proxy itself, so every client shares one rate limit bucket. Set it too
      high and clients can forge their address through `X-Forwarded-For`.
- [ ] **TLS terminates in front of the service.** The server speaks plain HTTP. It sends
      `Strict-Transport-Security` when the proxy sets `X-Forwarded-Proto: https`.
- [ ] **`/docs` is intended to be public, or off.** Set `OPENAPI_DOCS_ENABLED=false` to
      remove it. The page loads a script from a CDN.

## Accounts

- [ ] **Decide on open registration.** With `AUTH_REGISTRATION_ENABLED=true`, anyone can
      create an account. Set it to `false` when accounts are created by administrators.
- [ ] **The bootstrap administrator has a strong password**, and
      `BOOTSTRAP_ADMIN_PASSWORD` is removed from the environment once that account exists.
      The variables are ignored after the first administrator is created.
- [ ] **Token lifetimes fit your risk.** `AUTH_ACCESS_TOKEN_TTL` (15 minutes) bounds how
      long a stolen access token works. `AUTH_REFRESH_TOKEN_TTL` (30 days) bounds a session.

## Database

- [ ] **Choose how migrations run.** Either leave `DATABASE_AUTO_MIGRATE=true`, which is
      safe with several replicas on PostgreSQL because of the advisory lock, or set it to
      `false` and run `/app/migrate up` as a deployment step. The second option lets the
      server run with a database role that cannot change the schema.
- [ ] **Migrations are backward compatible** with the version still running during a
      rolling update: add columns before using them, remove them after.
- [ ] **The pool fits the database.** `DATABASE_MAX_OPEN_CONNS` times the number of
      replicas must stay below the connection limit of PostgreSQL.
- [ ] **Backups exist and have been restored once.**
- [ ] **PostgreSQL, not SQLite**, for anything with more than one instance.

## Limits

- [ ] **Rate limits are sized per replica.** The counters are in process memory:
      `RATE_LIMIT_REQUESTS` applies to each replica separately. For a strict global limit,
      enforce it at the gateway.
- [ ] **`HTTP_MAX_BODY_BYTES`** (1 MiB) fits your largest legitimate request.
- [ ] **Timeouts fit your slowest endpoint.** `HTTP_REQUEST_TIMEOUT` (30 s) must be shorter
      than the timeout of the proxy in front, and `HTTP_WRITE_TIMEOUT` (10 s) bounds how
      long a response may take to write.
- [ ] **`APP_SHUTDOWN_TIMEOUT`** (10 s) is shorter than the grace period your platform gives
      a container before killing it.

## Performance

- [ ] **`OPENAPI_VALIDATE_RESPONSES=false`.** It is the production default. When on, every
      response is buffered in memory and validated.
- [ ] **`AUTH_BCRYPT_COST`** (12) is measured on your hardware. Each login costs one hash;
      a higher cost slows logins and password guessing alike.
- [ ] **`ADMIN_PPROF_ENABLED=false`** unless you are profiling.

## Observability

- [ ] **Logs are collected** from standard output, as JSON.
- [ ] **Metrics are scraped** from `:9090/metrics`, with an alert on the 5xx rate and on
      latency.
- [ ] **Probes are configured**: `/health/live` for liveness and `/health/ready` for
      readiness.
- [ ] **Traces go somewhere**, if you enable them: `OTEL_ENABLED=true` and
      `OTEL_EXPORTER_OTLP_ENDPOINT`.

## Supply chain

- [ ] **Dependabot is enabled** for Go modules, actions and base images.
- [ ] **`make vuln` passes.** CI runs `govulncheck` on every change.
- [ ] **Images are pinned by digest** in your manifests, and their signature is verified
      where your platform supports it.

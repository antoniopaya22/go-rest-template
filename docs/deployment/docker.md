# Docker

## The image

The `Dockerfile` builds in two stages. The first compiles both commands; the second copies
them into a [distroless](https://github.com/GoogleContainerTools/distroless) static image.

| | |
|---|---|
| Base | `gcr.io/distroless/static-debian12:nonroot`, pinned by digest |
| User | `nonroot`, uid 65532 |
| Contents | `/app/server` and `/app/migrate`. No shell, no package manager. |
| Entrypoint | `/app/server` |
| Ports | 8080 (API), 9090 (admin) |
| Environment | `APP_ENV=production` |
| Health check | `/app/server -healthcheck` |

Both database drivers are pure Go, so the binaries are built with `CGO_ENABLED=0` and are
fully static. The OpenAPI document and the migrations are embedded; there are no other
files to copy.

### Health check

A distroless image has no `curl`. The server can probe itself: `/app/server -healthcheck`
requests `/health/live` on `127.0.0.1:$HTTP_PORT` and exits 0 when it answers 200. The
`HEALTHCHECK` instruction of the image and the Compose file both use it.

## Build

```sh
make docker-build
```

This tags `go-rest-template:dev` and passes the build metadata. The equivalent command:

```sh
docker build \
  --build-arg VERSION=$(git describe --tags --always --dirty) \
  --build-arg COMMIT=$(git rev-parse HEAD) \
  --build-arg DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t go-rest-template:dev .
```

| Build argument | Default | Shown by |
|---|---|---|
| `VERSION` | `dev` | `GET /version`, `/app/server -version`, the startup log |
| `COMMIT` | `unknown` | The same |
| `DATE` | `unknown` | The same |

Override the image name with `make docker-build IMAGE=registry.example.com/billing:1.0.0`.

## Run

With PostgreSQL:

```sh
docker run --rm -p 8080:8080 \
  -e AUTH_SECRET="$(openssl rand -base64 48)" \
  -e DATABASE_URL='postgres://user:password@db.example.com:5432/app?sslmode=require' \
  go-rest-template:dev
```

In production mode the server refuses to start without `DATABASE_URL` and with the default
`AUTH_SECRET`. Publish 9090 only where your metrics scraper can reach it.

To try the image without a database, use SQLite in a temporary directory:

```sh
docker run --rm -p 8080:8080 -p 9090:9090 \
  -e APP_ENV=development -e DATABASE_DRIVER=sqlite -e SQLITE_PATH=/tmp/app.db \
  go-rest-template:dev
```

The CI workflow starts the image this way and waits for `/health/ready`.

!!! note "SQLite in a container"

    The data lives where `SQLITE_PATH` points. Mount a volume there to keep it, and make
    sure uid 65532 can write to it. Run a single instance: SQLite has one writer.

## Migrations

By default the server applies pending migrations when it starts
(`DATABASE_AUTO_MIGRATE=true`). To run them as a separate step, disable that and use the
second binary:

```sh
docker run --rm --entrypoint /app/migrate \
  -e AUTH_SECRET=... -e DATABASE_URL=... \
  go-rest-template:dev up
```

`status` and `down` work the same way. See [Database](../guides/database.md#migrations).

## Read-only root filesystem

The server writes nothing to disk with PostgreSQL, so the container can run with
`--read-only`. The Helm chart does, with an `emptyDir` on `/tmp`.

## Verifying a published image

Images released by the workflow are signed with cosign (keyless) and carry a build
provenance attestation:

```sh
cosign verify ghcr.io/antoniopaya22/go-rest-template:<version> \
  --certificate-identity-regexp '^https://github.com/antoniopaya22/go-rest-template/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

gh attestation verify oci://ghcr.io/antoniopaya22/go-rest-template:<version> \
  --repo antoniopaya22/go-rest-template
```

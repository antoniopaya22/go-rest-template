# Quickstart

This page runs the API locally and walks through the main conventions with `curl`. The
responses shown were recorded from the template; identifiers, tokens and timestamps will
differ on your machine.

## Run the server

In development the default database is a SQLite file under `data/`, created on first
start. The first administrator is created from the `BOOTSTRAP_ADMIN_*` variables when no
administrator exists:

```sh
BOOTSTRAP_ADMIN_ENABLED=true BOOTSTRAP_ADMIN_USERNAME=admin \
BOOTSTRAP_ADMIN_FIRST_NAME=Ada BOOTSTRAP_ADMIN_LAST_NAME=Lovelace \
BOOTSTRAP_ADMIN_PASSWORD=change-me-now-123 make run
```

Instead of exporting variables you can copy `configs/env.example` to `.env` and edit it.

| Address | What is there |
|---|---|
| <http://localhost:8080/docs> | Interactive API reference |
| <http://localhost:8080/openapi.yaml> | The OpenAPI document |
| <http://localhost:8080/health/ready> | Readiness probe |
| <http://localhost:9090/metrics> | Prometheus metrics, on the admin port |

`make dev` does the same and restarts the server when a file changes.

## Log in

```sh
curl -s localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"change-me-now-123"}'
```

```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "...",
  "token_type": "Bearer",
  "expires_in": 900
}
```

The access token lasts 15 minutes by default. Keep both tokens in shell variables for the
rest of the page:

```sh
LOGIN=$(curl -s localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"change-me-now-123"}')
TOKEN=$(echo "$LOGIN" | sed -E 's/.*"access_token":"([^"]+)".*/\1/')
REFRESH=$(echo "$LOGIN" | sed -E 's/.*"refresh_token":"([^"]+)".*/\1/')
```

```sh
curl -s localhost:8080/api/v1/users/me -H "Authorization: Bearer $TOKEN"
```

```json
{"id":1,"username":"admin","first_name":"Ada","last_name":"Lovelace","role":"admin","created_at":"2026-11-16T14:36:34.000195Z","updated_at":"2026-11-16T14:36:34.000195Z"}
```

## Create and list tasks

```sh
curl -si localhost:8080/api/v1/tasks -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"Write docs","description":"All of them"}'
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
Etag: "1"
X-Request-Id: Z2TB2MN7NK7LHMDDPHB6TJCRRZ

{"id":1,"title":"Write docs","description":"All of them","status":"pending","user_id":1,"created_at":"2026-11-16T14:36:34.444868Z","updated_at":"2026-11-16T14:36:34.444868Z"}
```

The `ETag` is the version of the task. Create a second task, then list one per page:

```sh
curl -s "localhost:8080/api/v1/tasks?limit=1" -H "Authorization: Bearer $TOKEN"
```

```json
{"data":[{"id":1,"title":"Write docs","description":"All of them","status":"pending","user_id":1,"created_at":"2026-11-16T14:36:34.444868Z","updated_at":"2026-11-16T14:36:34.444868Z"}],"next_cursor":"MQ"}
```

Pass `next_cursor` back as `cursor` to get the next page. The last page has no
`next_cursor`.

## Update with If-Match

Send the `ETag` you read in `If-Match`. The update succeeds and the version moves on:

```sh
curl -si -X PATCH localhost:8080/api/v1/tasks/1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -H 'If-Match: "1"' -d '{"status":"done"}'
```

```http
HTTP/1.1 200 OK
Etag: "2"

{"id":1,"title":"Write docs","description":"All of them","status":"done","user_id":1,"created_at":"2026-11-16T14:36:34.444868Z","updated_at":"2026-11-16T14:36:34.481878Z"}
```

Repeat the request with the old `ETag` and it is refused:

```http
HTTP/1.1 412 Precondition Failed
Content-Type: application/problem+json

{"type":"about:blank","title":"Precondition Failed","status":412,"detail":"the resource has changed since it was last read","code":"precondition_failed","request_id":"W5PIKAB6ZRQY7JUDIJEO4LZYLS"}
```

## See a validation error

```sh
curl -s localhost:8080/api/v1/tasks -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"","description":"x"}'
```

```json
{"type":"about:blank","title":"Bad Request","status":400,"detail":"request validation failed","code":"invalid_argument","request_id":"GO5WXRAR52UC6J6HFXS6R6QPWZ","errors":[{"field":"title","message":"minimum string length is 1"}]}
```

Every error has this shape. `code` is stable and meant for programs; `request_id` also
appears in the server log line of the request. See [Errors](../architecture/errors.md).

## Refresh the tokens

```sh
curl -s localhost:8080/api/v1/auth/refresh -H 'Content-Type: application/json' \
  -d "{\"refresh_token\":\"$REFRESH\"}"
```

The response is a new token pair. The refresh token you sent is now used up. Sending it
again fails, and also revokes the pair you just received:

```json
{"type":"about:blank","title":"Unauthorized","status":401,"detail":"refresh token already used","code":"unauthorized","request_id":"5J32UGPAYNCABDKLH7DLMSHW4S"}
```

See [Authentication and authorization](../guides/authentication.md) for the reasoning.

## Look at the metrics

```sh
curl -s localhost:9090/metrics | grep '^http_requests_total'
```

```text
http_requests_total{method="GET",route="/api/v1/tasks",status="200"} 1
http_requests_total{method="POST",route="/api/v1/tasks",status="201"} 2
...
```

Metrics are served on the admin port, never on the public one.

## Run the tests

```sh
make test     # SQLite only
make check    # tidy, vet, lint and the tests with the race detector
```

To run the end-to-end tests against PostgreSQL as well, see [Testing](../guides/testing.md).

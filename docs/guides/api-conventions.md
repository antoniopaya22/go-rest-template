# API conventions

Every endpoint follows the same rules for lists, updates, validation and errors. The
helpers are in `internal/platform/httpx`.

## Lists

A list endpoint returns an envelope:

```json
{
  "data": [ { "id": 1, "title": "Write docs" } ],
  "next_cursor": "MQ"
}
```

| Parameter | Default | Rule |
|---|---|---|
| `limit` | 20 | Between 1 and 100. Anything else is a 400. |
| `cursor` | none | The `next_cursor` of the previous page. Anything else is a 400. |

- `data` is always an array, `[]` when empty.
- `next_cursor` is present only when another page exists.
- Results are ordered by ascending identifier. There is no sort parameter.
- The cursor is opaque. Today it encodes the last identifier of the page; clients must not
  build or parse it.

Pagination is by key, not by offset: the repository asks for rows with an identifier
greater than the cursor, so a page costs the same however deep it is and rows inserted
meanwhile do not shift the pages.

In a handler:

```go
page, err := httpx.ParsePage(r)
if err != nil {
	httpx.WriteError(w, r, err)
	return
}

tasks, err := h.service.List(r.Context(), filter, page)
// ...
httpx.WriteJSON(w, http.StatusOK, httpx.NewPage(tasks, page, func(t Task) int64 { return t.ID }))
```

The repository fetches `page.Limit+1` rows. `NewPage` uses the extra row only to know
whether to return a cursor.

Filters are plain query parameters defined per resource, such as `status` and `user_id`
for tasks and `username` and `role` for users.

## Concurrent updates

Every row has a `version` column that starts at 1 and grows by one on each update. It is
exposed as a strong `ETag`:

```http
HTTP/1.1 200 OK
Etag: "2"
```

`GET`, `POST` and `PATCH` responses of a single resource carry it. Two mechanisms protect
a write.

**`If-Match`, chosen by the client.** Send the `ETag` you read with a `PATCH` or `DELETE`.
When the resource has changed since, the request fails with **412** and the code
`precondition_failed`. Without the header the request is unconditional. `If-Match: *`
matches any version.

**The version check, always on.** The repository updates with
`WHERE id = ? AND version = ?`, using the version the handler loaded. When another request
changed the row between the read and the write, no row matches and the request fails with
**409** and the code `conflict`. This holds even when the client sends no `If-Match`.

| Situation | Status | Code |
|---|---|---|
| `If-Match` is older than the current version | 412 | `precondition_failed` |
| Two writes race; this one read before the other committed | 409 | `conflict` |

A client that gets either should read the resource again and retry.

## Validation

Input is validated twice, for different reasons.

**By the contract.** Under `/api/v1`, requests are validated against `api/openapi.yaml`
before the handler runs: required properties, types, enumerations, string lengths, and
`additionalProperties: false`. This is the declared shape of the API.

**By the service.** The service validates what a schema cannot express and what must hold
however the service is called: trimmed strings must not be blank, the password length is
checked in bytes, a status must be one the code knows.

Both report the same way, a 400 with the field:

```json
{
  "code": "invalid_argument",
  "detail": "request validation failed",
  "errors": [ { "field": "title", "message": "minimum string length is 1" } ]
}
```

Request validation can be turned off with `OPENAPI_VALIDATE_REQUESTS=false`. The service
checks remain, and so does `httpx.DecodeJSON`, which rejects unknown fields, malformed
JSON and trailing data on its own.

## Request bodies

- Bodies are JSON objects. Send `Content-Type: application/json`; the contract declares no
  other media type.
- The size is capped by `HTTP_MAX_BODY_BYTES`, 1 MiB by default.
- Unknown fields are rejected, not ignored.
- In a `PATCH`, a field that is absent is left unchanged. An empty object is rejected by
  the contract (`minProperties: 1`).

## Responses

| Operation | Status | Body |
|---|---|---|
| Read one | 200 | The resource |
| List | 200 | `{data, next_cursor}` |
| Create | 201 | The resource |
| Update | 200 | The resource |
| Delete | 204 | None |
| Any failure | 4xx or 5xx | A [problem response](../architecture/errors.md) |

Timestamps are RFC 3339 with microsecond precision. Identifiers are positive
64-bit integers.

## Headers

| Header | Direction | Meaning |
|---|---|---|
| `Authorization: Bearer <token>` | request | The access token. |
| `If-Match` | request | The `ETag` a write is based on. |
| `X-Request-ID` | both | Sent back on every response; reused when the client sends a valid one. |
| `ETag` | response | The version of the returned resource. |
| `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` | response | The rate limit state, when rate limiting is enabled. |

CORS allows the `Accept`, `Authorization`, `Content-Type`, `If-Match` and `X-Request-ID`
request headers and exposes `ETag` and `X-Request-ID` to browsers.

## The contract and the code

`api/openapi.yaml` is hand-written and the Go types are hand-written. Three checks keep
them aligned:

1. The document is parsed and validated when the server starts; an invalid document stops it.
2. `TestRoutesMatchContract` requires the router and the document to list the same operations.
3. The end-to-end tests validate every response against the document.

The document is served at `/openapi.yaml`, and an interactive reference at `/docs` unless
`OPENAPI_DOCS_ENABLED=false`. The reference page loads its script from a CDN at a fixed
version with Subresource Integrity; the version and the hash are constants in
`internal/platform/contract/contract.go`.

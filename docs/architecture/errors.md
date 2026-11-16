# Errors

Errors have two halves. Inside the application an error has a **code**. At the edge, one
function turns the code into an HTTP **problem response**.

## Application errors

`internal/platform/errs` defines the error type used by services and repositories:

```go
type Error struct {
	Code    Code
	Message string       // safe to show to clients
	Fields  []FieldError // per-field details of a validation failure
	Err     error        // the cause; logged, never sent
}
```

It knows nothing about HTTP. Constructors cover the common cases:

```go
errs.NotFound("task not found")
errs.Conflict("username already exists")
errs.Forbidden("insufficient permissions")
errs.InvalidField("title", "is required")
errs.Wrap(errs.CodeUnavailable, "database not ready", err)
```

`errs.CodeOf(err)` returns the code of the first application error in the chain, and
`internal` for any other error. `errs.Is(err, code)` tests for one code.

## Codes and statuses

`httpx.StatusOf` holds the only mapping from codes to statuses:

| Code | Status | Meaning |
|---|---|---|
| `invalid_argument` | 400 | The request is malformed or fails validation. |
| `unauthorized` | 401 | The credential is missing, invalid or expired. |
| `forbidden` | 403 | The caller is authenticated but may not do this. |
| `not_found` | 404 | The resource does not exist, or the caller may not know that it does. |
| `conflict` | 409 | The request conflicts with the current state: a duplicate, or a concurrent update. |
| `precondition_failed` | 412 | `If-Match` does not match the current version. |
| `rate_limited` | 429 | Too many requests from this client. |
| `unavailable` | 503 | A dependency is not ready, or the process is shutting down. |
| `internal` | 500 | Anything else. |

Two responses are produced by the router rather than by an application error: 405 for a
known path with the wrong method (code `invalid_argument`) and 504 when the request
deadline expires, which chi's timeout middleware writes without a body.

## The problem response

Every error body follows [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) and is sent as
`application/problem+json`:

```json
{
  "type": "about:blank",
  "title": "Bad Request",
  "status": 400,
  "detail": "request validation failed",
  "code": "invalid_argument",
  "request_id": "GO5WXRAR52UC6J6HFXS6R6QPWZ",
  "errors": [
    { "field": "title", "message": "minimum string length is 1" }
  ]
}
```

| Field | Content |
|---|---|
| `type` | Always `about:blank`. The category is in `code`. |
| `title` | The standard text of the status. |
| `status` | The HTTP status, repeated in the body. |
| `detail` | The message of the application error. Omitted for `internal`. |
| `code` | The stable code from the table above. Clients should branch on this. |
| `request_id` | The identifier of the request, also in the `X-Request-ID` header and in the logs. |
| `errors` | Field-level details, present for validation failures. |

## Internal details stay internal

`httpx.WriteError` is the only function that writes an error response:

- An error that is not an application error becomes `internal` with no `detail`. The
  client sees `Internal Server Error` and a `request_id`; the full error goes to the log.
- The cause wrapped in an application error (`Err`) is logged and never serialised. A
  duplicate username is reported to the client as `username already exists`, while the
  log keeps the driver error.
- Statuses of 500 and above are logged at `ERROR`. Rejections are logged at `DEBUG`, since
  the access record already shows the status.

The same applies to panics: `middleware.Recover` logs the value and the stack trace and
answers with a plain `internal` problem.

## Where errors come from

| Source | Example | Result |
|---|---|---|
| Contract validation | A missing required property | 400 `invalid_argument`, with the field |
| `httpx.DecodeJSON` | Malformed JSON, unknown field, body too large | 400 `invalid_argument` |
| Service validation | A title made of spaces | 400 `invalid_argument`, with the field |
| Authentication middleware | No bearer token | 401 `unauthorized` |
| Handler authorization | Another user's account | 403 `forbidden` |
| Repository, no row | An unknown identifier | 404 `not_found` |
| Repository, unique violation | A duplicate username | 409 `conflict` |
| Repository, foreign key violation | A task for a user that does not exist | 400 `invalid_argument` |
| Repository, version mismatch | Two updates at the same time | 409 `conflict` |
| `httpx.CheckIfMatch` | A stale `If-Match` | 412 `precondition_failed` |

Repositories translate driver errors with `db.Translate(err, "task")`, which produces
`task not found`, `task already exists` or
`task references a record that does not exist`. See [Database](../guides/database.md).

!!! note "404 instead of 403 for tasks"

    Reading another user's task returns 404, not 403, so task identifiers cannot be
    probed. User accounts return 403: the check happens before the lookup and does not
    reveal whether the account exists.

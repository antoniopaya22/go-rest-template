// Package httpx contains the HTTP conventions shared by every handler: JSON
// encoding, RFC 9457 problem responses, pagination and conditional requests.
package httpx

import (
	"encoding/json"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/logging"
)

// ProblemContentType is the media type of error responses (RFC 9457).
const ProblemContentType = "application/problem+json"

// Problem is the body of every error response.
type Problem struct {
	Type      string            `json:"type"`
	Title     string            `json:"title"`
	Status    int               `json:"status"`
	Detail    string            `json:"detail,omitempty"`
	Code      errs.Code         `json:"code"`
	RequestID string            `json:"request_id,omitempty"`
	Errors    []errs.FieldError `json:"errors,omitempty"`
}

// StatusOf maps an application error code to an HTTP status.
func StatusOf(code errs.Code) int {
	switch code {
	case errs.CodeInvalidArgument:
		return http.StatusBadRequest
	case errs.CodeUnauthorized:
		return http.StatusUnauthorized
	case errs.CodeForbidden:
		return http.StatusForbidden
	case errs.CodeNotFound:
		return http.StatusNotFound
	case errs.CodeConflict:
		return http.StatusConflict
	case errs.CodePreconditionFailed:
		return http.StatusPreconditionFailed
	case errs.CodeRateLimited:
		return http.StatusTooManyRequests
	case errs.CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// WriteError writes err as a problem response. Errors that are not application
// errors are logged and reported as a generic internal error, so internal
// details never reach the client.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	code := errs.CodeOf(err)
	status := StatusOf(code)

	problem := Problem{
		Type:      "about:blank",
		Title:     http.StatusText(status),
		Status:    status,
		Code:      code,
		RequestID: chimiddleware.GetReqID(r.Context()),
	}

	if appErr, ok := errs.As(err); ok && status != http.StatusInternalServerError {
		problem.Detail = appErr.Message
		problem.Errors = appErr.Fields
	}

	logger := logging.FromContext(r.Context())
	if status >= http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "request failed", "error", err)
	} else {
		logger.DebugContext(r.Context(), "request rejected", "code", code, "error", err)
	}

	WriteProblem(w, problem)
}

// WriteProblem writes a problem response.
func WriteProblem(w http.ResponseWriter, problem Problem) {
	w.Header().Set("Content-Type", ProblemContentType)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(problem.Status)
	_ = json.NewEncoder(w).Encode(problem)
}

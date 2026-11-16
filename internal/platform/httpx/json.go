package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
)

// WriteJSON writes payload as a JSON response.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(payload)
}

// DecodeJSON reads a single JSON object from the request body into dst.
// Unknown fields and trailing data are rejected.
func DecodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		var tooLarge *http.MaxBytesError
		switch {
		case errors.Is(err, io.EOF):
			return errs.Invalid("request body is required")
		case errors.As(err, &tooLarge):
			return errs.Invalid("request body is too large")
		default:
			return errs.Wrap(errs.CodeInvalidArgument, "request body is not valid JSON", err)
		}
	}

	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errs.Invalid("request body must contain a single JSON object")
	}
	return nil
}

// PathID reads a positive integer identifier from a URL path parameter.
func PathID(r *http.Request, name string) (int64, error) {
	return parseID(chi.URLParam(r, name), name)
}

// QueryID reads an optional positive integer identifier from the query string.
func QueryID(r *http.Request, name string) (*int64, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}

	id, err := parseID(raw, name)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseID(raw string, name string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errs.InvalidField(name, "must be a positive integer")
	}
	return id, nil
}

package middleware

import (
	"context"
	"crypto/rand"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const requestIDHeader = "X-Request-ID"

// RequestID gives every request an identifier, stored where chi's GetReqID
// finds it and echoed in the response. An identifier sent by the client or a
// proxy is kept when it looks sane, so one ID can follow a call across services.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if !validRequestID(id) {
			id = rand.Text()
		}

		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), chimiddleware.RequestIDKey, id)))
	})
}

// validRequestID accepts short identifiers made of URL-safe characters, which
// keeps client-supplied values from polluting logs.
func validRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
		default:
			return false
		}
	}
	return true
}

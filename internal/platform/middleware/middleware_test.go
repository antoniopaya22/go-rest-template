package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		incoming string
		wantKept bool
	}{
		{"generated when absent", "", false},
		{"kept when sane", "trace-01.abc_DEF", true},
		{"replaced when it has unsafe characters", "id with spaces\n", false},
		{"replaced when too long", strings.Repeat("a", 65), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seen string
			handler := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				seen = chimiddleware.GetReqID(r.Context())
			}))

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.incoming != "" {
				request.Header.Set("X-Request-ID", tt.incoming)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if seen == "" || recorder.Header().Get("X-Request-ID") != seen {
				t.Fatalf("context id %q and response header %q must match and be set", seen, recorder.Header().Get("X-Request-ID"))
			}
			if kept := seen == tt.incoming; kept != tt.wantKept {
				t.Errorf("id = %q, kept = %t, want kept = %t", seen, kept, tt.wantKept)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	t.Parallel()

	handler := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "boom") {
		t.Errorf("the panic value leaked to the client: %s", recorder.Body)
	}
}

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	handler := SecurityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	plain := httptest.NewRecorder()
	handler.ServeHTTP(plain, httptest.NewRequest(http.MethodGet, "/", nil))
	if plain.Header().Get("X-Content-Type-Options") != "nosniff" || plain.Header().Get("X-Frame-Options") != "DENY" {
		t.Errorf("missing security headers: %v", plain.Header())
	}
	if plain.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must not be sent over plain HTTP")
	}

	forwarded := httptest.NewRequest(http.MethodGet, "/", nil)
	forwarded.Header.Set("X-Forwarded-Proto", "https")
	secure := httptest.NewRecorder()
	handler.ServeHTTP(secure, forwarded)
	if secure.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS must be sent when the request arrived over HTTPS")
	}
}

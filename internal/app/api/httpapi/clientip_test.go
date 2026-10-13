package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestClientIP(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies int
		remoteAddr     string
		xff            string
		want           string
	}{
		{name: "no proxies ignores spoofed header", trustedProxies: 0, remoteAddr: "203.0.113.7:4321", xff: "1.2.3.4", want: "203.0.113.7"},
		{name: "one trusted proxy uses its entry", trustedProxies: 1, remoteAddr: "10.0.0.2:4321", xff: "1.2.3.4, 198.51.100.9", want: "198.51.100.9"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			handler := clientIP(tt.trustedProxies)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = chimiddleware.GetClientIP(r.Context())
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("X-Forwarded-For", tt.xff)
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if got != tt.want {
				t.Fatalf("client IP = %q, want %q", got, tt.want)
			}
		})
	}
}

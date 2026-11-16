package app_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/antoniopaya22/go-rest-template/internal/app/apptest"
	"github.com/antoniopaya22/go-rest-template/internal/auth"
)

func TestOperationalEndpoints(t *testing.T) {
	t.Parallel()

	apptest.Each(t, func(t *testing.T, env *apptest.Env) {
		for _, path := range []string{"/health/live", "/health/ready", "/version", "/docs"} {
			env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: path})
		}

		spec := env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/openapi.yaml"})
		if !strings.Contains(string(spec.Body), "openapi: 3.1.0") {
			t.Fatalf("unexpected OpenAPI document: %.80s", spec.Body)
		}

		response := env.Do(http.StatusNotFound, apptest.Request{Method: http.MethodGet, Path: "/nope"})
		if got := response.Header.Get("Content-Type"); got != "application/problem+json" {
			t.Errorf("content type = %q, want application/problem+json", got)
		}
		if response.Header.Get("X-Request-ID") == "" {
			t.Error("response has no X-Request-ID header")
		}
		if got := response.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
		}

		env.Do(http.StatusMethodNotAllowed, apptest.Request{Method: http.MethodPut, Path: "/health/live"})
	})
}

// TestRoutesMatchContract fails when a route is added without documenting it
// in api/openapi.yaml, or documented without being implemented.
func TestRoutesMatchContract(t *testing.T) {
	t.Parallel()

	env := apptest.New(t, apptest.Backends()[0])

	spec := env.App.ContractOperations()

	router, ok := env.App.Handler().(chi.Routes)
	if !ok {
		t.Fatal("the API handler is not a chi router")
	}

	var routes []string
	err := chi.Walk(router, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		switch route {
		case "/docs", "/openapi.yaml", "/.well-known/jwks.json":
			return nil // served by the platform, not part of the contract
		}
		routes = append(routes, method+" "+strings.TrimSuffix(route, "/"))
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	sort.Strings(routes)

	if got, want := strings.Join(routes, "\n"), strings.Join(spec, "\n"); got != want {
		t.Errorf("router and OpenAPI contract differ\n--- routes ---\n%s\n--- contract ---\n%s", got, want)
	}
}

func TestAuthenticationFlow(t *testing.T) {
	t.Parallel()

	apptest.Each(t, func(t *testing.T, env *apptest.Env) {
		env.Do(http.StatusUnauthorized, apptest.Request{
			Method: http.MethodPost, Path: "/api/v1/auth/login",
			Body: map[string]string{"username": apptest.AdminUsername, "password": "wrong-password"},
		})
		env.Do(http.StatusUnauthorized, apptest.Request{
			Method: http.MethodPost, Path: "/api/v1/auth/login",
			Body: map[string]string{"username": "nobody", "password": "wrong-password"},
		})

		pair := env.Login(apptest.AdminUsername, apptest.AdminPassword)
		if pair.TokenType != "Bearer" || pair.ExpiresIn <= 0 {
			t.Fatalf("unexpected token pair: %+v", pair)
		}

		env.Do(http.StatusUnauthorized, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users/me"})
		env.Do(http.StatusUnauthorized, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users/me", Token: "not-a-token"})
		env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users/me", Token: pair.AccessToken})

		refresh := func(token string, want int) auth.TokenPair {
			var next auth.TokenPair
			response := env.Do(want, apptest.Request{
				Method: http.MethodPost, Path: "/api/v1/auth/refresh",
				Body: map[string]string{"refresh_token": token},
			})
			if want == http.StatusOK {
				response.Decode(t, &next)
			}
			return next
		}

		rotated := refresh(pair.RefreshToken, http.StatusOK)
		if rotated.RefreshToken == pair.RefreshToken {
			t.Fatal("refresh did not rotate the refresh token")
		}

		// Reusing a rotated token revokes the whole family, including the
		// token that was issued from it.
		refresh(pair.RefreshToken, http.StatusUnauthorized)
		refresh(rotated.RefreshToken, http.StatusUnauthorized)
		refresh("unknown-token", http.StatusUnauthorized)

		// Logout revokes the family and is silent about unknown tokens.
		session := env.Login(apptest.AdminUsername, apptest.AdminPassword)
		for _, token := range []string{session.RefreshToken, "unknown-token"} {
			env.Do(http.StatusNoContent, apptest.Request{
				Method: http.MethodPost, Path: "/api/v1/auth/logout",
				Body: map[string]string{"refresh_token": token},
			})
		}
		refresh(session.RefreshToken, http.StatusUnauthorized)
	})
}

func TestDeletedUserLosesAccessImmediately(t *testing.T) {
	t.Parallel()

	apptest.Each(t, func(_ *testing.T, env *apptest.Env) {
		id, token := env.User("leaver")
		env.Do(http.StatusNoContent, apptest.Request{Method: http.MethodDelete, Path: pathf("/api/v1/users/%d", id), Token: env.AdminToken()})
		env.Do(http.StatusUnauthorized, apptest.Request{Method: http.MethodGet, Path: "/api/v1/users/me", Token: token})
	})
}

func TestRequestValidation(t *testing.T) {
	t.Parallel()

	env := apptest.New(t, apptest.Backends()[0])
	login := "/api/v1/auth/login"

	tests := []struct {
		name string
		body any
	}{
		{"empty body", ""},
		{"malformed JSON", "{"},
		{"missing field", map[string]string{"username": "admin"}},
		{"unknown field", map[string]string{"username": "a", "password": "b", "extra": "c"}},
		{"wrong type", map[string]any{"username": 1, "password": "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := env.Do(http.StatusBadRequest, apptest.Request{
				Method: http.MethodPost, Path: login, Body: tt.body,
				Header: map[string]string{"Content-Type": "application/json"},
			})
			if code := response.Code(t); code != "invalid_argument" {
				t.Errorf("code = %q, want invalid_argument", code)
			}
		})
	}
}

// init:ratelimit:begin
func TestLoginIsRateLimited(t *testing.T) {
	t.Parallel()

	env := apptest.New(t, apptest.Backends()[0], map[string]string{
		"RATE_LIMIT_ENABLED":        "true",
		"RATE_LIMIT_LOGIN_REQUESTS": "2",
	})

	attempt := func(want int) apptest.Response {
		return env.Do(want, apptest.Request{
			Method: http.MethodPost, Path: "/api/v1/auth/login",
			Body: map[string]string{"username": "nobody", "password": "wrong-password"},
		})
	}

	attempt(http.StatusUnauthorized)
	attempt(http.StatusUnauthorized)
	if code := attempt(http.StatusTooManyRequests).Code(t); code != "rate_limited" {
		t.Errorf("code = %q, want rate_limited", code)
	}
}

// init:ratelimit:end

func TestAdminServer(t *testing.T) {
	t.Parallel()

	env := apptest.New(t, apptest.Backends()[0], map[string]string{"ADMIN_PORT": "19191", "ADMIN_PPROF_ENABLED": "true"})
	env.Do(http.StatusOK, apptest.Request{Method: http.MethodGet, Path: "/health/live"})

	admin := httptest.NewServer(env.App.AdminHandler())
	t.Cleanup(admin.Close)

	get := func(path string) (int, string) {
		response, err := http.Get(admin.URL + path) //nolint:noctx // test helper.
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer func() { _ = response.Body.Close() }()
		body, _ := io.ReadAll(response.Body)
		return response.StatusCode, string(body)
	}

	if status, _ := get("/health/live"); status != http.StatusOK {
		t.Errorf("admin /health/live = %d", status)
	}
	if status, _ := get("/debug/pprof/cmdline"); status != http.StatusOK {
		t.Errorf("admin /debug/pprof/cmdline = %d", status)
	}
	// init:metrics:begin
	status, body := get("/metrics")
	if status != http.StatusOK || !strings.Contains(body, `http_requests_total{method="GET",route="/health/live",status="200"}`) {
		t.Errorf("admin /metrics = %d, missing the request counter", status)
	}
	// init:metrics:end
}

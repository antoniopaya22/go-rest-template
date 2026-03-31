// Package apicontract loads and validates the HTTP API contract.
package apicontract

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/antonioalfa22/go-rest-template/internal/pkg/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	legacyrouter "github.com/getkin/kin-openapi/routers/legacy"
)

// Contract contains the parsed OpenAPI specification and validation hooks.
type Contract struct {
	router            routers.Router
	spec              []byte
	validateRequests  bool
	validateResponses bool
}

// Load loads and validates an OpenAPI specification from disk.
func Load(cfg config.OpenAPIConfig) (*Contract, error) {
	spec, err := os.ReadFile(cfg.SpecPath)
	if err != nil {
		return nil, fmt.Errorf("read OpenAPI spec: %w", err)
	}

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(spec)
	if err != nil {
		return nil, fmt.Errorf("parse OpenAPI spec: %w", err)
	}

	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate OpenAPI spec: %w", err)
	}

	router, err := legacyrouter.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("build OpenAPI router: %w", err)
	}

	return &Contract{
		router:            router,
		spec:              spec,
		validateRequests:  cfg.ValidateRequests,
		validateResponses: cfg.ValidateResponses,
	}, nil
}

// Middleware validates requests and responses against the contract.
func (c *Contract) Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route, pathParams, err := c.router.FindRoute(r)
			if err != nil {
				logger.Error("openapi route lookup failed", "method", r.Method, "path", r.URL.Path, "error", err)
				writeValidationError(w, http.StatusInternalServerError, "internal", "route is not defined in the OpenAPI contract")
				return
			}

			requestInput := &openapi3filter.RequestValidationInput{
				Request:    r,
				PathParams: pathParams,
				Route:      route,
				Options: &openapi3filter.Options{
					AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
						return nil
					},
				},
			}

			if c.validateRequests {
				if err := openapi3filter.ValidateRequest(r.Context(), requestInput); err != nil {
					writeValidationError(w, http.StatusBadRequest, "invalid_argument", err.Error())
					return
				}
			}

			recorder := httptest.NewRecorder()
			next.ServeHTTP(recorder, r)

			response := recorder.Result()
			defer func() {
				_ = response.Body.Close()
			}()

			if c.validateResponses {
				validationInput := &openapi3filter.ResponseValidationInput{
					RequestValidationInput: requestInput,
					Status:                 response.StatusCode,
					Header:                 response.Header,
					Body:                   response.Body,
				}

				if err := openapi3filter.ValidateResponse(r.Context(), validationInput); err != nil {
					logger.Error("openapi response validation failed", "method", r.Method, "path", r.URL.Path, "status", response.StatusCode, "error", err)
					writeValidationError(w, http.StatusInternalServerError, "internal", "response does not match the OpenAPI contract")
					return
				}

				response.Body = io.NopCloser(recorder.Body)
			}

			copyHeaders(w.Header(), response.Header)
			w.WriteHeader(response.StatusCode)
			if _, err := io.Copy(w, response.Body); err != nil {
				logger.Error("write validated response", "method", r.Method, "path", r.URL.Path, "error", err)
			}
		})
	}
}

// SpecHandler serves the loaded OpenAPI spec.
func (c *Contract) SpecHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(c.spec)
	}
}

// DocsHandler serves a small ReDoc-based documentation page.
func (c *Contract) DocsHandler() http.HandlerFunc {
	const page = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>API Documentation</title>
  <style>
    body { margin: 0; font-family: Georgia, "Times New Roman", serif; background: #f5f0e8; }
    header { padding: 20px 24px; background: #122620; color: #f5f0e8; }
    h1 { margin: 0; font-size: 20px; letter-spacing: 0.04em; text-transform: uppercase; }
  </style>
</head>
<body>
  <header><h1>Go REST Template API</h1></header>
  <redoc spec-url="/openapi/openapi.yaml"></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}
}

func copyHeaders(dst http.Header, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func writeValidationError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"code":%q,"error":%q}`, code, message)
}

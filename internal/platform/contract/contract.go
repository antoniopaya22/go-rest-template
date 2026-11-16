// Package contract enforces the OpenAPI document at runtime: requests that do
// not match it are rejected before they reach a handler, and, when enabled,
// responses that drift from it fail loudly.
package contract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy"

	"github.com/antoniopaya22/go-rest-template/internal/platform/errs"
	"github.com/antoniopaya22/go-rest-template/internal/platform/httpx"
	"github.com/antoniopaya22/go-rest-template/internal/platform/logging"
)

// Options selects which directions of traffic are validated.
type Options struct {
	ValidateRequests  bool
	ValidateResponses bool
}

// Contract is a parsed OpenAPI document with its validation hooks.
type Contract struct {
	doc     *openapi3.T
	router  routers.Router
	spec    []byte
	options Options
}

// Load parses and validates an OpenAPI document.
func Load(spec []byte, options Options) (*Contract, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(spec)
	if err != nil {
		return nil, fmt.Errorf("parse OpenAPI document: %w", err)
	}

	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate OpenAPI document: %w", err)
	}

	router, err := legacy.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("build OpenAPI router: %w", err)
	}

	return &Contract{doc: doc, router: router, spec: spec, options: options}, nil
}

// Operations lists every operation of the document as "METHOD /path", sorted.
func (c *Contract) Operations() []string {
	operations := make([]string, 0)
	for path, item := range c.doc.Paths.Map() {
		for method := range item.Operations() {
			operations = append(operations, method+" "+path)
		}
	}
	sort.Strings(operations)
	return operations
}

// Middleware validates traffic against the contract.
func (c *Contract) Middleware(next http.Handler) http.Handler {
	if !c.options.ValidateRequests && !c.options.ValidateResponses {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route, pathParams, err := c.router.FindRoute(r)
		if err != nil {
			// The router owns 404 and 405 handling; the contract only
			// validates operations it knows about.
			next.ServeHTTP(w, r)
			return
		}

		input := &openapi3filter.RequestValidationInput{
			Request:    r,
			PathParams: pathParams,
			Route:      route,
			Options: &openapi3filter.Options{
				// Authentication is enforced by the auth middleware.
				AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error { return nil },
			},
		}

		if c.options.ValidateRequests {
			if err := openapi3filter.ValidateRequest(r.Context(), input); err != nil {
				httpx.WriteError(w, r, requestError(err))
				return
			}
		}

		if !c.options.ValidateResponses {
			next.ServeHTTP(w, r)
			return
		}

		buffer := &bufferedResponse{header: http.Header{}, status: http.StatusOK}
		next.ServeHTTP(buffer, r)

		err = openapi3filter.ValidateResponse(r.Context(), &openapi3filter.ResponseValidationInput{
			RequestValidationInput: input,
			Status:                 buffer.status,
			Header:                 buffer.header,
			Body:                   nopCloser{bytes.NewReader(buffer.body.Bytes())},
		})
		if err != nil {
			logging.FromContext(r.Context()).ErrorContext(r.Context(), "response does not match the OpenAPI contract",
				"method", r.Method, "path", r.URL.Path, "status", buffer.status, "error", err)
			httpx.WriteError(w, r, errs.Wrap(errs.CodeInternal, "response contract violation", err))
			return
		}

		for key, values := range buffer.header {
			w.Header()[key] = values
		}
		w.WriteHeader(buffer.status)
		_, _ = w.Write(buffer.body.Bytes())
	})
}

// requestError converts a validation failure into an invalid_argument error,
// naming the offending field when the validator reports one.
func requestError(err error) error {
	var requestErr *openapi3filter.RequestError
	if !errors.As(err, &requestErr) {
		return errs.Wrap(errs.CodeInvalidArgument, "request does not match the API contract", err)
	}

	location := "body"
	if requestErr.Parameter != nil {
		location = requestErr.Parameter.Name
	}

	var schemaErr *openapi3.SchemaError
	if errors.As(requestErr.Err, &schemaErr) {
		field := location
		if pointer := schemaErr.JSONPointer(); len(pointer) > 0 {
			field = strings.Join(pointer, ".")
		}
		return errs.InvalidField(field, schemaErr.Reason)
	}

	reason := requestErr.Reason
	if reason == "" && requestErr.Err != nil {
		reason = requestErr.Err.Error()
	}
	return errs.InvalidField(location, reason)
}

// SpecHandler serves the OpenAPI document.
func (c *Contract) SpecHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(c.spec)
	}
}

// The API reference is rendered by Scalar, loaded from a CDN at a fixed
// version and verified with Subresource Integrity. To upgrade, change the
// version and recompute the hash:
//
//	curl -sL <url> | openssl dgst -sha384 -binary | openssl base64 -A
const (
	scalarURL       = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.79.0/dist/browser/standalone.js"
	scalarIntegrity = "sha384-L1JnNDE1YhW1+DtFBec/sNUonpzUXyLvYkF21NCjmni3yu/untPXTfk5aDHn3a+h"
)

// DocsHandler serves an interactive API reference for the document at specURL.
func (c *Contract) DocsHandler(title string, specURL string) http.HandlerFunc {
	page := fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s</title>
</head>
<body>
  <script id="api-reference" data-url=%q></script>
  <script src=%q integrity=%q crossorigin="anonymous"></script>
</body>
</html>
`, title, specURL, scalarURL, scalarIntegrity)

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; style-src 'unsafe-inline'; "+
				"img-src 'self' data: https:; font-src https: data:; connect-src 'self'")
		_, _ = w.Write([]byte(page))
	}
}

type bufferedResponse struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (b *bufferedResponse) Header() http.Header         { return b.header }
func (b *bufferedResponse) WriteHeader(status int)      { b.status = status }
func (b *bufferedResponse) Write(p []byte) (int, error) { return b.body.Write(p) }

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

package contract

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const spec = `
openapi: 3.1.0
info: {title: test, version: "1"}
servers: [{url: /}]
paths:
  /items:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              additionalProperties: false
              properties:
                name: {type: string, minLength: 2}
      responses:
        '201':
          description: created
          content:
            application/json:
              schema:
                type: object
                required: [id]
                properties:
                  id: {type: integer}
`

func serve(t *testing.T, options Options, response string, body string) *httptest.ResponseRecorder {
	t.Helper()

	contract, err := Load([]byte(spec), options)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	handler := contract.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(response))
	}))

	request := httptest.NewRequest(http.MethodPost, "/items", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestRequestValidation(t *testing.T) {
	t.Parallel()

	both := Options{ValidateRequests: true, ValidateResponses: true}

	tests := []struct {
		name string
		body string
		want int
	}{
		{"valid", `{"name":"ok"}`, http.StatusCreated},
		{"missing property", `{}`, http.StatusBadRequest},
		{"too short", `{"name":"a"}`, http.StatusBadRequest},
		{"unknown property", `{"name":"ok","extra":1}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		if recorder := serve(t, both, `{"id":1}`, tt.body); recorder.Code != tt.want {
			t.Errorf("%s: status = %d, want %d; body: %s", tt.name, recorder.Code, tt.want, recorder.Body)
		}
	}

	if recorder := serve(t, both, `{"id":1}`, `{"name":"a"}`); !strings.Contains(recorder.Body.String(), `"field":"name"`) {
		t.Errorf("the offending field is not reported: %s", recorder.Body)
	}

	if recorder := serve(t, Options{}, `{"id":1}`, `{}`); recorder.Code != http.StatusCreated {
		t.Errorf("disabled validation must pass requests through, got %d", recorder.Code)
	}
}

func TestResponseValidation(t *testing.T) {
	t.Parallel()

	const drifted = `{"id":"not-an-integer"}`

	if recorder := serve(t, Options{ValidateResponses: true}, drifted, `{"name":"ok"}`); recorder.Code != http.StatusInternalServerError {
		t.Errorf("a drifted response must fail, got %d", recorder.Code)
	} else if strings.Contains(recorder.Body.String(), "not-an-integer") {
		t.Error("the invalid response body leaked to the client")
	}

	if recorder := serve(t, Options{ValidateRequests: true}, drifted, `{"name":"ok"}`); recorder.Code != http.StatusCreated {
		t.Errorf("response validation is off, got %d", recorder.Code)
	}
}

func TestLoadRejectsInvalidDocuments(t *testing.T) {
	t.Parallel()

	for name, document := range map[string]string{
		"not YAML":      "{{{",
		"missing paths": "openapi: 3.1.0\ninfo: {title: t}",
	} {
		if _, err := Load([]byte(document), Options{}); err == nil {
			t.Errorf("%s: Load must fail", name)
		}
	}
}

func TestOperationsAndHandlers(t *testing.T) {
	t.Parallel()

	contract, err := Load([]byte(spec), Options{})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got := contract.Operations(); len(got) != 1 || got[0] != "POST /items" {
		t.Errorf("Operations = %v", got)
	}

	recorder := httptest.NewRecorder()
	contract.SpecHandler()(recorder, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if recorder.Body.String() != spec {
		t.Error("the spec handler must serve the document unchanged")
	}

	recorder = httptest.NewRecorder()
	contract.DocsHandler("Docs", "/openapi.yaml")(recorder, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if body := recorder.Body.String(); !strings.Contains(body, `integrity="sha384-`) || !strings.Contains(body, `data-url="/openapi.yaml"`) {
		t.Errorf("unexpected docs page: %s", body)
	}
}

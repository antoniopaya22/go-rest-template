package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
)

func TestWriteError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   errs.Code
		wantDetail string
	}{
		{"validation", errs.InvalidField("title", "is required"), http.StatusBadRequest, errs.CodeInvalidArgument, "request validation failed"},
		{"not found", errs.NotFound("user not found"), http.StatusNotFound, errs.CodeNotFound, "user not found"},
		{"precondition", errs.New(errs.CodePreconditionFailed, "changed"), http.StatusPreconditionFailed, errs.CodePreconditionFailed, "changed"},
		{"rate limited", errs.New(errs.CodeRateLimited, "slow down"), http.StatusTooManyRequests, errs.CodeRateLimited, "slow down"},
		{"unavailable", errs.New(errs.CodeUnavailable, "database not ready"), http.StatusServiceUnavailable, errs.CodeUnavailable, "database not ready"},
		// Internal details must never reach the client.
		{"foreign error", errors.New("dial tcp 10.0.0.5:5432: refused"), http.StatusInternalServerError, errs.CodeInternal, ""},
		{"explicit internal", errs.Wrap(errs.CodeInternal, "signing key missing", errors.New("secret")), http.StatusInternalServerError, errs.CodeInternal, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()
			WriteError(recorder, httptest.NewRequest(http.MethodGet, "/", nil), tt.err)

			var problem Problem
			if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
				t.Fatalf("decode problem: %v", err)
			}

			if recorder.Code != tt.wantStatus || problem.Status != tt.wantStatus {
				t.Errorf("status = %d (body %d), want %d", recorder.Code, problem.Status, tt.wantStatus)
			}
			if problem.Code != tt.wantCode || problem.Detail != tt.wantDetail {
				t.Errorf("problem = %+v", problem)
			}
			if got := recorder.Header().Get("Content-Type"); got != ProblemContentType {
				t.Errorf("content type = %q", got)
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"valid", `{"name":"a"}`, false},
		{"empty", ``, true},
		{"malformed", `{"name":`, true},
		{"unknown field", `{"name":"a","other":1}`, true},
		{"wrong type", `{"name":1}`, true},
		{"trailing data", `{"name":"a"}{"name":"b"}`, true},
	}

	for _, tt := range tests {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
		var dst payload
		err := DecodeJSON(request, &dst)
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: error = %v, wantErr %t", tt.name, err, tt.wantErr)
		}
		if err != nil && !errs.Is(err, errs.CodeInvalidArgument) {
			t.Errorf("%s: code = %q, want invalid_argument", tt.name, errs.CodeOf(err))
		}
	}
}

func TestDecodeJSONRejectsLargeBodies(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"`+strings.Repeat("a", 100)+`"}`))
	request.Body = http.MaxBytesReader(httptest.NewRecorder(), request.Body, 16)

	var dst struct {
		Name string `json:"name"`
	}
	err := DecodeJSON(request, &dst)
	if e, ok := errs.As(err); !ok || e.Message != "request body is too large" {
		t.Errorf("error = %v, want the size limit error", err)
	}
}

func TestPagination(t *testing.T) {
	t.Parallel()

	parse := func(query string) (PageRequest, error) {
		return ParsePage(httptest.NewRequest(http.MethodGet, "/?"+query, nil))
	}

	page, err := parse("")
	if err != nil || page.Limit != DefaultPageSize || page.After != 0 {
		t.Fatalf("defaults = %+v, %v", page, err)
	}

	for _, query := range []string{"limit=0", "limit=-1", "limit=101", "limit=ten", "cursor=!!", "cursor=YWJj"} {
		if _, err := parse(query); !errs.Is(err, errs.CodeInvalidArgument) {
			t.Errorf("%s: error = %v, want invalid_argument", query, err)
		}
	}

	identity := func(i int64) int64 { return i }

	// Three items fetched with limit 2: one more page exists.
	full := NewPage([]int64{10, 11, 12}, PageRequest{Limit: 2}, identity)
	if len(full.Data) != 2 || full.NextCursor == "" {
		t.Fatalf("page = %+v, want 2 items and a cursor", full)
	}

	next, err := parse("limit=2&cursor=" + full.NextCursor)
	if err != nil || next.After != 11 {
		t.Fatalf("cursor round trip = %+v, %v; want After=11", next, err)
	}

	last := NewPage([]int64{12}, next, identity)
	if len(last.Data) != 1 || last.NextCursor != "" {
		t.Errorf("last page = %+v, want 1 item and no cursor", last)
	}

	// An empty page encodes as [], not null.
	empty, _ := json.Marshal(NewPage(nil, PageRequest{Limit: 2}, identity))
	if string(empty) != `{"data":[]}` {
		t.Errorf("empty page = %s", empty)
	}
}

func TestIfMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		header  string
		wantErr bool
	}{
		{"", false},
		{"*", false},
		{`"3"`, false},
		{`"1", "3"`, false},
		{`"2"`, true},
		{`3`, true},
	}

	for _, tt := range tests {
		request := httptest.NewRequest(http.MethodPatch, "/", nil)
		if tt.header != "" {
			request.Header.Set("If-Match", tt.header)
		}

		err := CheckIfMatch(request, 3)
		if (err != nil) != tt.wantErr {
			t.Errorf("If-Match %q: error = %v, wantErr %t", tt.header, err, tt.wantErr)
		}
		if err != nil && !errs.Is(err, errs.CodePreconditionFailed) {
			t.Errorf("If-Match %q: code = %q", tt.header, errs.CodeOf(err))
		}
	}

	recorder := httptest.NewRecorder()
	SetETag(recorder, 3)
	if got := recorder.Header().Get("ETag"); got != `"3"` {
		t.Errorf("ETag = %q", got)
	}
}

// FuzzParsePage checks that no query string can make pagination panic or
// produce a request outside its limits.
func FuzzParsePage(f *testing.F) {
	for _, seed := range []string{"", "limit=10", "cursor=MTE", "limit=0&cursor=%00", "limit=9999999999999999999", "cursor=LTE"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, query string) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.URL.RawQuery = query

		page, err := ParsePage(request)
		if err != nil {
			return
		}
		if page.Limit < 1 || page.Limit > MaxPageSize || page.After < 0 {
			t.Fatalf("query %q produced an out-of-range page: %+v", query, page)
		}
	})
}

// FuzzDecodeJSON checks that no body can make decoding panic and that every
// failure is reported as invalid input.
func FuzzDecodeJSON(f *testing.F) {
	for _, seed := range []string{`{"name":"a"}`, `{`, `[]`, `null`, `{"name":1}`, "\x00", `{"name":"a"} x`} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, body string) {
		var dst struct {
			Name string `json:"name"`
		}
		err := DecodeJSON(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)), &dst)
		if err != nil && !errs.Is(err, errs.CodeInvalidArgument) {
			t.Fatalf("body %q: error code = %q, want invalid_argument", body, errs.CodeOf(err))
		}
	})
}

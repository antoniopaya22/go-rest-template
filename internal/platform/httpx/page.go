package httpx

import (
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/antoniopaya22/go-rest-template/v3/internal/platform/errs"
)

// Pagination limits.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// PageRequest is the pagination input of a list endpoint. Results are ordered
// by ascending identifier and After is the last identifier already seen.
type PageRequest struct {
	Limit int
	After int64
}

// Page is the envelope of every list response.
type Page[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// ParsePage reads the "limit" and "cursor" query parameters.
func ParsePage(r *http.Request) (PageRequest, error) {
	page := PageRequest{Limit: DefaultPageSize}
	query := r.URL.Query()

	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 || limit > MaxPageSize {
			return PageRequest{}, errs.InvalidField("limit", "must be between 1 and "+strconv.Itoa(MaxPageSize))
		}
		page.Limit = limit
	}

	if raw := query.Get("cursor"); raw != "" {
		after, err := decodeCursor(raw)
		if err != nil {
			return PageRequest{}, errs.InvalidField("cursor", "is not a cursor returned by this API")
		}
		page.After = after
	}

	return page, nil
}

// NewPage builds a page from items fetched with a limit of request.Limit+1.
// The extra item, when present, only signals that another page exists.
func NewPage[T any](items []T, request PageRequest, id func(T) int64) Page[T] {
	page := Page[T]{Data: items}
	if len(items) > request.Limit {
		page.Data = items[:request.Limit]
		page.NextCursor = encodeCursor(id(page.Data[len(page.Data)-1]))
	}
	if page.Data == nil {
		page.Data = []T{}
	}
	return page
}

func encodeCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(id, 10)))
}

func decodeCursor(cursor string) (int64, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}

	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || id < 0 {
		return 0, strconv.ErrSyntax
	}
	return id, nil
}

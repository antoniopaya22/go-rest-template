package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/antoniopaya22/go-rest-template/internal/pkg/errs"
)

type errorResponse struct {
	Code  string `json:"code,omitempty"`
	Error string `json:"error"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer func() {
		_ = r.Body.Close()
	}()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if err == io.EOF {
			return errs.New(errs.CodeInvalidArgument, "request body is required", err)
		}

		return errs.New(errs.CodeInvalidArgument, fmt.Sprintf("invalid JSON body: %v", err), err)
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errs.New(errs.CodeInvalidArgument, "request body must contain a single JSON object", nil)
	}

	return nil
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, errs.New(errs.CodeInvalidArgument, "resource id must be a positive integer", err)
	}

	return id, nil
}

func parseOptionalID(raw string) (*int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	value, err := parseID(trimmed)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	status := errs.StatusFromCode(errs.CodeOf(err))
	writeJSON(w, status, errorResponse{
		Code:  string(errs.CodeOf(err)),
		Error: errs.MessageOf(err),
	})
}

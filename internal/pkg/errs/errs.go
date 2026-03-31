// Package errs defines transport-agnostic application errors.
package errs

import (
	"errors"
	"net/http"
)

// Code identifies a category of application error.
type Code string

const (
	// CodeInvalidArgument identifies malformed or semantically invalid input.
	CodeInvalidArgument Code = "invalid_argument"
	// CodeUnauthorized identifies authentication failures.
	CodeUnauthorized Code = "unauthorized"
	// CodeForbidden identifies authorization failures.
	CodeForbidden Code = "forbidden"
	// CodeNotFound identifies missing resources.
	CodeNotFound Code = "not_found"
	// CodeConflict identifies state conflicts such as duplicate keys.
	CodeConflict Code = "conflict"
	// CodeInternal identifies unexpected internal failures.
	CodeInternal Code = "internal"
)

// Error wraps an underlying error with a stable application code and message.
type Error struct {
	Code    Code
	Message string
	Err     error
}

// Error returns the printable error message.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	if e.Message != "" {
		return e.Message
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return string(e.Code)
}

// Unwrap returns the underlying wrapped error.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// New creates a new application error.
func New(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// CodeOf returns the application code for an error.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}

	var target *Error
	if errors.As(err, &target) {
		return target.Code
	}

	return CodeInternal
}

// MessageOf returns the user-facing message for an error.
func MessageOf(err error) string {
	if err == nil {
		return ""
	}

	var target *Error
	if errors.As(err, &target) && target.Message != "" {
		return target.Message
	}

	return http.StatusText(StatusFromCode(CodeOf(err)))
}

// StatusFromCode maps an error code to an HTTP status.
func StatusFromCode(code Code) int {
	switch code {
	case CodeInvalidArgument:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

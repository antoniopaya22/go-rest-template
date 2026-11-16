// Package errs defines transport-agnostic application errors.
//
// Services and repositories return *Error values with a stable Code. The HTTP
// layer maps codes to status codes; nothing in this package knows about HTTP.
package errs

import (
	"errors"
	"fmt"
)

// Code identifies a category of application error.
type Code string

// Application error codes.
const (
	CodeInvalidArgument    Code = "invalid_argument"
	CodeUnauthorized       Code = "unauthorized"
	CodeForbidden          Code = "forbidden"
	CodeNotFound           Code = "not_found"
	CodeConflict           Code = "conflict"
	CodePreconditionFailed Code = "precondition_failed"
	CodeRateLimited        Code = "rate_limited"
	CodeUnavailable        Code = "unavailable"
	CodeInternal           Code = "internal"
)

// FieldError describes a problem with a single input field.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error is an application error with a stable code and a client-safe message.
type Error struct {
	Code    Code
	Message string
	Fields  []FieldError
	Err     error
}

// Error returns the client-safe message, followed by the cause when present.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *Error) Unwrap() error { return e.Err }

// New creates an application error.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap creates an application error that keeps the cause for logging.
// The cause is never sent to clients.
func Wrap(code Code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// Invalid reports malformed or semantically invalid input.
func Invalid(message string) *Error { return New(CodeInvalidArgument, message) }

// InvalidField reports invalid input attributed to one field.
func InvalidField(field string, message string) *Error {
	return &Error{
		Code:    CodeInvalidArgument,
		Message: "request validation failed",
		Fields:  []FieldError{{Field: field, Message: message}},
	}
}

// Unauthorized reports a missing or invalid credential.
func Unauthorized(message string) *Error { return New(CodeUnauthorized, message) }

// Forbidden reports an authenticated caller without permission.
func Forbidden(message string) *Error { return New(CodeForbidden, message) }

// NotFound reports a missing resource.
func NotFound(message string) *Error { return New(CodeNotFound, message) }

// Conflict reports a state conflict such as a duplicate key.
func Conflict(message string) *Error { return New(CodeConflict, message) }

// As returns the application error in err's chain, if any.
func As(err error) (*Error, bool) {
	var target *Error
	if errors.As(err, &target) {
		return target, true
	}
	return nil, false
}

// CodeOf returns the code of err, or CodeInternal for foreign errors.
func CodeOf(err error) Code {
	if err == nil {
		return ""
	}
	if target, ok := As(err); ok {
		return target.Code
	}
	return CodeInternal
}

// Is reports whether err carries the given code.
func Is(err error, code Code) bool { return err != nil && CodeOf(err) == code }

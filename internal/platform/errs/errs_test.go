package errs

import (
	"errors"
	"fmt"
	"testing"
)

func TestCodeOf(t *testing.T) {
	t.Parallel()

	cause := errors.New("driver: connection reset")
	tests := []struct {
		name string
		err  error
		want Code
	}{
		{"nil", nil, ""},
		{"application error", NotFound("user not found"), CodeNotFound},
		{"wrapped application error", fmt.Errorf("load user: %w", Conflict("duplicate")), CodeConflict},
		{"foreign error", cause, CodeInternal},
		{"application error with cause", Wrap(CodeUnavailable, "database not ready", cause), CodeUnavailable},
	}

	for _, tt := range tests {
		if got := CodeOf(tt.err); got != tt.want {
			t.Errorf("%s: CodeOf = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestErrorKeepsCauseForLogsOnly(t *testing.T) {
	t.Parallel()

	cause := errors.New("pq: duplicate key")
	err := Wrap(CodeConflict, "username already exists", cause)

	if !errors.Is(err, cause) {
		t.Error("the cause must stay reachable with errors.Is")
	}
	if err.Message != "username already exists" {
		t.Errorf("client message = %q", err.Message)
	}
	if got := err.Error(); got != "username already exists: pq: duplicate key" {
		t.Errorf("Error() = %q", got)
	}
}

func TestInvalidField(t *testing.T) {
	t.Parallel()

	err := InvalidField("title", "is required")
	if !Is(err, CodeInvalidArgument) || len(err.Fields) != 1 || err.Fields[0].Field != "title" {
		t.Errorf("unexpected error: %+v", err)
	}
	if Is(nil, CodeInvalidArgument) {
		t.Error("Is(nil) must be false")
	}
}

package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
)

func wrapQueryError(err error, notFoundMessage string, conflictMessage string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return errs.New(errs.CodeNotFound, notFoundMessage, err)
	}

	lower := strings.ToLower(err.Error())
	if conflictMessage != "" && strings.Contains(lower, "unique constraint failed") {
		return errs.New(errs.CodeConflict, conflictMessage, err)
	}

	if strings.Contains(lower, "foreign key constraint failed") {
		return errs.New(errs.CodeInvalidArgument, "invalid foreign key reference", err)
	}

	return err
}

func rowsAffectedOrNotFound(affected int64, message string) error {
	if affected == 0 {
		return errs.New(errs.CodeNotFound, message, nil)
	}

	return nil
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTimestamp(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", value, err)
	}

	return parsed, nil
}

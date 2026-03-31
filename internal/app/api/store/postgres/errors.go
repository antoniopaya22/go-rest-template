package postgres

import (
	"database/sql"
	"errors"

	"github.com/antonioalfa22/go-rest-template/internal/pkg/errs"
	"github.com/jackc/pgx/v5/pgconn"
)

func wrapQueryError(err error, notFoundMessage string, conflictMessage string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return errs.New(errs.CodeNotFound, notFoundMessage, err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if conflictMessage != "" {
				return errs.New(errs.CodeConflict, conflictMessage, err)
			}
		case "23503":
			return errs.New(errs.CodeInvalidArgument, "invalid foreign key reference", err)
		}
	}

	return err
}

func rowsAffectedOrNotFound(affected int64, message string) error {
	if affected == 0 {
		return errs.New(errs.CodeNotFound, message, nil)
	}

	return nil
}

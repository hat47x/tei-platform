package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	sqlgen "tei/internal/database/sqlgen"
)

// pgCodedError pairs a driver error with the SQLSTATE it carries, so sqlgen can
// classify by code instead of by message text.
//
// pgconn.PgError exposes the SQLSTATE directly; the message text around it is
// dialect- and version-dependent and, on a localized server, may not even be
// English. The original error is preserved through Unwrap so the diagnostic a
// developer needs is not lost.
type pgCodedError struct {
	state string
	err   error
}

func (e *pgCodedError) Error() string { return e.err.Error() }

func (e *pgCodedError) Unwrap() error { return e.err }

func (e *pgCodedError) DatabaseProvider() string { return "postgres" }

func (e *pgCodedError) DatabaseCode() string { return e.state }

// Ensure pgCodedError satisfies the carrier sqlgen looks for.
var _ sqlgen.CodeCarrier = (*pgCodedError)(nil)

// FailureOf converts a driver error into a TEI failure, classified by the
// SQLSTATE when the error carries one. An error without a SQLSTATE is returned
// unchanged, so the caller decides whether to fall back to
// sqlgen.ToDatabaseFailure's message matching rather than having that decision
// made silently here.
func FailureOf(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	return sqlgen.ToDatabaseFailure("postgres", &pgCodedError{state: pgErr.Code, err: err})
}

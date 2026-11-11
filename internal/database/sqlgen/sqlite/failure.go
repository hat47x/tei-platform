package sqlite

import (
	"errors"
	"strconv"

	"modernc.org/sqlite"

	sqlgen "tei/internal/database/sqlgen"
)

// sqliteCodedError pairs a driver error with the SQLite result code it carries,
// so sqlgen can classify by code instead of by message text.
//
// The driver reports the extended result code, which puts the constraint
// subtype (unique, not-null, foreign-key) in the high byte. Only the primary
// code is exposed, because that is what every SQLite version agrees on and what
// sqlgen classifies; the subtype stays in the driver's own message, which is
// preserved through Unwrap.
type sqliteCodedError struct {
	code int
	err  error
}

func (e *sqliteCodedError) Error() string { return e.err.Error() }

func (e *sqliteCodedError) Unwrap() error { return e.err }

func (e *sqliteCodedError) DatabaseProvider() string { return "sqlite" }

func (e *sqliteCodedError) DatabaseCode() string {
	// Mask the extended bits: code & 0xff is the primary result code.
	return strconv.Itoa(e.code & 0xff)
}

// Ensure sqliteCodedError satisfies the carrier sqlgen looks for.
var _ sqlgen.CodeCarrier = (*sqliteCodedError)(nil)

// FailureOf converts a driver error into a TEI failure, classified by the
// SQLite result code when the error carries one. An error without a code is
// returned unchanged, so the caller decides whether to fall back to
// sqlgen.ToDatabaseFailure's message matching rather than having that decision
// made silently here.
func FailureOf(err error) error {
	if err == nil {
		return nil
	}
	var driverErr *sqlite.Error
	if !errors.As(err, &driverErr) {
		return err
	}
	return sqlgen.ToDatabaseFailure("sqlite", &sqliteCodedError{code: driverErr.Code(), err: err})
}

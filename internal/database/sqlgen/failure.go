package sqlgen

import (
	"errors"
	"fmt"
	"strings"

	"tei/internal/failure"
)

// DatabaseFailure wraps a database error into TEI's common failure model.
type DatabaseFailure struct {
	code    string
	message string
	cause   error
	kind    failure.Kind
	// retryable is the base retry decision for this failure. It is deliberately
	// separate from the policy Retryable() returns: an effect-unknown failure is
	// never blindly retryable, whatever the base decision says.
	retryable bool
	effect    bool
}

func (f DatabaseFailure) Error() string {
	if f.cause != nil {
		return fmt.Sprintf("%s: %v", f.message, f.cause)
	}
	return f.message
}

func (f DatabaseFailure) FailureCode() string {
	return f.code
}

func (f DatabaseFailure) FailureMessage() string {
	return f.message
}

func (f DatabaseFailure) FailureCause() error {
	return f.cause
}

// Unwrap exposes the cause to the standard library, so errors.Is and errors.As
// reach the driver error underneath the failure. Without it a caller that
// already matches on a driver-specific type would silently stop matching once
// the failure passed through this conversion.
func (f DatabaseFailure) Unwrap() error {
	return f.cause
}

func (f DatabaseFailure) Kind() failure.Kind {
	return f.kind
}

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure.
//
// The policy is derived, not stored. A failure whose effect may already be
// applied is never blindly retryable: resending with the same identity
// duplicates the write, and resending with a new one duplicates the record, so
// the caller must look the result up under the same identity first. Deriving it
// here keeps that invariant true for every code, including ones added later,
// instead of relying on each case to remember it.
func (f DatabaseFailure) Retryable() failure.RetryPolicy {
	if f.effect {
		return failure.LookupFirst
	}
	if f.retryable {
		return failure.Retryable
	}
	return failure.NotRetryable
}

func (f DatabaseFailure) HasEffectUnknown() bool {
	return f.effect
}

// Ensure DatabaseFailure implements failure.Failure.
var _ failure.Failure = DatabaseFailure{}

// CodeCarrier is implemented by driver errors that expose a machine-readable
// code. PostgreSQL's pgconn.PgError carries a SQLSTATE and modernc sqlite's
// Error carries a result code. Each provider package adapts its own driver
// error to this interface, so this package does not import a database driver.
//
// A code beats a message: the message text is dialect- and version-dependent
// ("duplicate key", "UNIQUE constraint failed", a localized string), while the
// code is the contract the driver itself documents.
type CodeCarrier interface {
	error
	// DatabaseProvider names the provider the code belongs to, e.g. "postgres".
	DatabaseProvider() string
	// DatabaseCode returns the provider's own code, e.g. "23505" or "19".
	DatabaseCode() string
}

// databaseClassification is the decision one code maps to.
type databaseClassification struct {
	code      string
	kind      failure.Kind
	retryable bool
	effect    bool
}

// postgresSQLState maps the SQLSTATE codes a write path can actually meet. The
// class 23 codes are constraint violations: the statement was refused before it
// changed anything, so they are business rejections with no effect. Class 40
// and 55 are concurrency and cancellation: the transaction was aborted, but
// whether an earlier statement in the same transaction had already applied is
// not decidable from the error alone, so the effect is unknown.
var postgresSQLState = map[string]databaseClassification{
	"23505": {"UNIQUE_VIOLATION", failure.BusinessRejection, false, false},
	"23503": {"FOREIGN_KEY_VIOLATION", failure.BusinessRejection, false, false},
	"23502": {"NOT_NULL_VIOLATION", failure.BusinessRejection, false, false},
	"23514": {"CHECK_VIOLATION", failure.BusinessRejection, false, false},
	"23P01": {"EXCLUSION_VIOLATION", failure.BusinessRejection, false, false},
	"42501": {"INSUFFICIENT_PRIVILEGE", failure.BusinessRejection, false, false},
	"40001": {"SERIALIZATION_FAILURE", failure.TechnicalFailure, true, true},
	"40P01": {"DEADLOCK_DETECTED", failure.TechnicalFailure, true, true},
	"55P03": {"LOCK_NOT_AVAILABLE", failure.TechnicalFailure, true, true},
	"57014": {"QUERY_CANCELED", failure.TechnicalFailure, true, true},
	"08000": {"DATABASE_UNAVAILABLE", failure.TechnicalFailure, true, true},
	"08003": {"DATABASE_UNAVAILABLE", failure.TechnicalFailure, true, true},
	"08006": {"DATABASE_UNAVAILABLE", failure.TechnicalFailure, true, true},
	"53300": {"TOO_MANY_CONNECTIONS", failure.TechnicalFailure, true, true},
}

// sqliteResultCode maps the SQLite primary result codes. The driver reports the
// extended code, which puts the constraint subtype in the high byte; only the
// primary code is mapped here, because that is what every SQLite version agrees
// on and what the provider adapter masks to.
const (
	sqliteBusy       = 5  // SQLITE_BUSY
	sqliteLocked     = 6  // SQLITE_LOCKED
	sqliteCantOpen   = 14 // SQLITE_CANTOPEN
	sqliteConstraint = 19 // SQLITE_CONSTRAINT
)

var sqliteResultCodes = map[string]databaseClassification{
	"19": {"CONSTRAINT_VIOLATION", failure.BusinessRejection, false, false},
	"5":  {"DATABASE_BUSY", failure.TechnicalFailure, true, true},
	"6":  {"DATABASE_LOCKED", failure.TechnicalFailure, true, true},
	"14": {"DATABASE_UNAVAILABLE", failure.TechnicalFailure, true, true},
}

// ToDatabaseFailure converts a raw database error into a TEI Failure.
//
// A typed code is preferred: when the error carries one (see CodeCarrier), the
// classification comes from the provider's documented code table. Otherwise the
// message text is matched, which is a fallback for drivers that expose no code
// and for errors that were already flattened to a string. An unrecognised error
// is technical and requires a lookup first, so it is never reported as a
// permanent business rejection.
func ToDatabaseFailure(provider string, err error) failure.Failure {
	if err == nil {
		return nil
	}

	var carrier CodeCarrier
	if errors.As(err, &carrier) {
		if classified, ok := classifyCode(carrier.DatabaseProvider(), carrier.DatabaseCode()); ok {
			return newDatabaseFailure(carrier.DatabaseProvider(), classified, err)
		}
	}

	classified := classifyMessage(strings.ToLower(err.Error()))
	return newDatabaseFailure(provider, classified, err)
}

// classifyCode resolves a provider code. An unknown code falls back to the
// generic technical failure rather than being guessed from its digits.
func classifyCode(provider, code string) (databaseClassification, bool) {
	switch strings.ToLower(provider) {
	case "postgres", "postgresql", "pgx":
		if c, ok := postgresSQLState[code]; ok {
			return c, true
		}
	case "sqlite", "sqlite3":
		if c, ok := sqliteResultCodes[code]; ok {
			return c, true
		}
	}
	return databaseClassification{}, false
}

// classifyMessage matches the message text. It is the fallback for drivers that
// expose no code, and it keeps the same decisions the code tables make so a
// caller cannot tell which path produced the failure.
func classifyMessage(msg string) databaseClassification {
	switch {
	case strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique"):
		return databaseClassification{"UNIQUE_VIOLATION", failure.BusinessRejection, false, false}
	case strings.Contains(msg, "foreign key"):
		return databaseClassification{"FOREIGN_KEY_VIOLATION", failure.BusinessRejection, false, false}
	case strings.Contains(msg, "not null") || strings.Contains(msg, "null value"):
		return databaseClassification{"NOT_NULL_VIOLATION", failure.BusinessRejection, false, false}
	case strings.Contains(msg, "check constraint") || strings.Contains(msg, "violates check"):
		return databaseClassification{"CHECK_VIOLATION", failure.BusinessRejection, false, false}
	case strings.Contains(msg, "exclusion constraint"):
		return databaseClassification{"EXCLUSION_VIOLATION", failure.BusinessRejection, false, false}
	case strings.Contains(msg, "insufficient privilege") || strings.Contains(msg, "permission denied"):
		return databaseClassification{"INSUFFICIENT_PRIVILEGE", failure.BusinessRejection, false, false}
	case strings.Contains(msg, "deadlock") || strings.Contains(msg, "serialization failure"):
		// The transaction was aborted, but an earlier statement in it may have
		// applied, so the effect is unknown and the retry needs a lookup first.
		return databaseClassification{"CONCURRENCY_FAILURE", failure.TechnicalFailure, true, true}
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "canceled") || strings.Contains(msg, "cancelled"):
		return databaseClassification{"DATABASE_TIMEOUT", failure.TechnicalFailure, true, true}
	case strings.Contains(msg, "connection") || strings.Contains(msg, "unavailable") || strings.Contains(msg, "unable to open") || strings.Contains(msg, "too many connections"):
		return databaseClassification{"DATABASE_UNAVAILABLE", failure.TechnicalFailure, true, true}
	default:
		return databaseClassification{"DATABASE_ERROR", failure.TechnicalFailure, false, true}
	}
}

func newDatabaseFailure(provider string, c databaseClassification, cause error) DatabaseFailure {
	return DatabaseFailure{
		code:      fmt.Sprintf("%s_%s", strings.ToUpper(provider), c.code),
		message:   fmt.Sprintf("Database error from %s: %s", provider, cause.Error()),
		cause:     cause,
		kind:      c.kind,
		retryable: c.retryable,
		effect:    c.effect,
	}
}

// AsDatabaseFailure attempts to cast err to a DatabaseFailure.
func AsDatabaseFailure(err error) (DatabaseFailure, bool) {
	if err == nil {
		return DatabaseFailure{}, false
	}
	var f DatabaseFailure
	if errors.As(err, &f) {
		return f, true
	}
	return DatabaseFailure{}, false
}

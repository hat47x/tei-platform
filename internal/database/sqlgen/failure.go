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
	retry   failure.RetryPolicy
	effect  bool
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

func (f DatabaseFailure) Kind() failure.Kind {
	return f.kind
}

func (f DatabaseFailure) Retryable() failure.RetryPolicy {
	return f.retry
}

func (f DatabaseFailure) HasEffectUnknown() bool {
	return f.effect
}

// Ensure DatabaseFailure implements failure.Failure.
var _ failure.Failure = DatabaseFailure{}

// ToDatabaseFailure converts a raw database error into a TEI Failure.
// The provider parameter indicates which database dialect produced the error.
func ToDatabaseFailure(provider string, err error) failure.Failure {
	if err == nil {
		return nil
	}

	msg := strings.ToLower(err.Error())
	code := "DATABASE_ERROR"
	kind := failure.TechnicalFailure
	retry := failure.NotRetryable
	effect := true

	switch {
	case strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique"):
		code = "UNIQUE_VIOLATION"
		kind = failure.BusinessRejection
		retry = failure.NotRetryable
		effect = false
	case strings.Contains(msg, "foreign key") || strings.Contains(msg, "violates foreign key"):
		code = "FOREIGN_KEY_VIOLATION"
		kind = failure.BusinessRejection
		retry = failure.NotRetryable
		effect = false
	case strings.Contains(msg, "not null") || strings.Contains(msg, "null value"):
		code = "NOT_NULL_VIOLATION"
		kind = failure.BusinessRejection
		retry = failure.NotRetryable
		effect = false
	case strings.Contains(msg, "check constraint") || strings.Contains(msg, "violates check"):
		code = "CHECK_VIOLATION"
		kind = failure.BusinessRejection
		retry = failure.NotRetryable
		effect = false
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadlock"):
		code = "DATABASE_TIMEOUT"
		kind = failure.TechnicalFailure
		retry = failure.Retryable
		effect = true
	case strings.Contains(msg, "connection") || strings.Contains(msg, "unavailable") || strings.Contains(msg, "unable to open"):
		code = "DATABASE_UNAVAILABLE"
		kind = failure.TechnicalFailure
		retry = failure.Retryable
		effect = true
	default:
		code = "DATABASE_ERROR"
		kind = failure.TechnicalFailure
		retry = failure.LookupFirst
		effect = true
	}

	return DatabaseFailure{
		code:    fmt.Sprintf("%s_%s", strings.ToUpper(provider), code),
		message: fmt.Sprintf("Database error from %s: %s", provider, err.Error()),
		cause:   err,
		kind:    kind,
		retry:   retry,
		effect:  effect,
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

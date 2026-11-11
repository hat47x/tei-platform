// Package failure defines the unified API failure contract for TEI.
//
// It classifies failures into three business-meaningful kinds:
//   - BusinessRejection: the request is valid but refused by policy, authorization,
//     or business rules. No partial effect is expected.
//   - TechnicalFailure: the operation failed due to infrastructure, connectivity,
//     or system errors. Effects may or may not have been applied.
//   - PartialSuccess: the operation completed only partially. Some effects may
//     have been applied.
//
// Every failure also carries a retry policy and an effect-unknown flag so that
// callers can decide whether to retry, lookup, or compensate.
package failure

import (
	"context"
	"errors"
)

// Kind categorizes the business meaning of a failure.
type Kind string

const (
	// BusinessRejection means the request was refused by business rules,
	// authorization, or policy. The caller supplied a valid request that is
	// not permitted. No partial effect is expected.
	BusinessRejection Kind = "business_rejection"

	// TechnicalFailure means the operation failed due to infrastructure,
	// connectivity, or system errors. Effects may or may not have been
	// applied.
	TechnicalFailure Kind = "technical_failure"

	// PartialSuccess means the operation completed only partially. Some
	// effects may have been applied.
	PartialSuccess Kind = "partial_success"
)

// RetryPolicy describes whether and how a caller may retry a failed operation.
type RetryPolicy string

const (
	// Retryable means the same operation identity may be sent again without
	// a prior lookup.
	Retryable RetryPolicy = "retryable"

	// NotRetryable means the caller must not blindly retry the same
	// operation. Either the failure is permanent or the operation must be
	// changed before resending.
	NotRetryable RetryPolicy = "not_retryable"

	// LookupFirst means the caller should first query the result of the
	// original operation identity before deciding whether to retry with a
	// new identity.
	LookupFirst RetryPolicy = "lookup_first"
)

// Failure is the unified API failure contract.
//
// Implementations must also satisfy the error interface. The FailureCode and
// FailureMessage methods return the stable, machine-readable reason and the
// human-readable description. FailureCause is the underlying error, if any.
// Kind, Retryable, and HasEffectUnknown are the three axes callers branch on.
type Failure interface {
	error
	FailureCode() string
	FailureMessage() string
	FailureCause() error
	Kind() Kind
	Retryable() RetryPolicy
	HasEffectUnknown() bool
}

// AsFailure attempts to cast err to the unified Failure contract.
func AsFailure(err error) (Failure, bool) {
	if err == nil {
		return nil, false
	}
	var f Failure
	_ = errors.As(err, &f)
	return f, f != nil
}

// PartialError reports that an operation completed only partially: some of its
// effects were applied and some were not, and the caller cannot tell which from
// the error alone.
//
// It exists because "some applied, some not" is not either of the other two
// kinds. Reporting it as a business rejection would make a caller give up on
// work that did land; reporting it as a plain technical failure would make a
// caller either resend everything - duplicating the part that succeeded - or
// discard everything - losing the part that landed. Neither is acceptable, so
// the state gets its own kind.
//
// The retry policy is always LookupFirst and the effect is always unknown: the
// caller must find out what was applied before deciding anything, and that is
// not derivable from this error.
type PartialError struct {
	Code    string
	Message string
	Cause   error
}

// FailureCode returns the machine-readable failure code.
func (e *PartialError) FailureCode() string { return e.Code }

// FailureMessage returns the human-readable failure message.
func (e *PartialError) FailureMessage() string { return e.Message }

// FailureCause returns the underlying error, if any.
func (e *PartialError) FailureCause() error { return e.Cause }

// Error satisfies the error interface. The message names what was and was not
// applied, because a reader of a bare "partial" has no way to act on it.
func (e *PartialError) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Message + " (" + e.Cause.Error() + ")"
	}
	return e.Code + ": " + e.Message
}

// Kind classifies the business meaning of the failure.
func (e *PartialError) Kind() Kind { return PartialSuccess }

// Retryable reports whether the caller may safely retry the same operation
// identity after this failure.
func (e *PartialError) Retryable() RetryPolicy { return LookupFirst }

// HasEffectUnknown reports whether the failure may have left partial effects
// applied.
func (e *PartialError) HasEffectUnknown() bool { return true }

// Unwrap exposes the cause to the standard library, so errors.Is and errors.As
// still reach whatever the boundary was working on.
func (e *PartialError) Unwrap() error { return e.Cause }

// Ensure PartialError satisfies the unified failure contract.
var _ Failure = (*PartialError)(nil)

// IsBusinessRejection reports whether err is a known Failure whose Kind is
// BusinessRejection.
func IsBusinessRejection(err error) bool {
	f, ok := AsFailure(err)
	return ok && f.Kind() == BusinessRejection
}

// IsTechnicalFailure reports whether err is a known Failure whose Kind is
// TechnicalFailure.
func IsTechnicalFailure(err error) bool {
	f, ok := AsFailure(err)
	return ok && f.Kind() == TechnicalFailure
}

// IsPartialSuccess reports whether err is a known Failure whose Kind is
// PartialSuccess.
func IsPartialSuccess(err error) bool {
	f, ok := AsFailure(err)
	return ok && f.Kind() == PartialSuccess
}

// IsRetryable reports whether err is a known Failure whose Retryable is not
// NotRetryable.
func IsRetryable(err error) bool {
	f, ok := AsFailure(err)
	return ok && f.Retryable() != NotRetryable
}

// EffectMayBeUnknown reports whether the failed operation may have left partial
// effects applied.
func EffectMayBeUnknown(err error) bool {
	f, ok := AsFailure(err)
	return ok && f.HasEffectUnknown()
}

// IsTimeout reports whether err is a known Failure caused by a timeout or
// deadline exceeded.
func IsTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	f, ok := AsFailure(err)
	return ok && f.Kind() == TechnicalFailure && f.FailureCode() == "TIMEOUT"
}

// IsExternalServiceFailure reports whether err is a known Failure caused by an
// external service or provider being unavailable or returning an error.
func IsExternalServiceFailure(err error) bool {
	f, ok := AsFailure(err)
	if !ok {
		return false
	}
	switch f.FailureCode() {
	case "EXTERNAL_SERVICE_UNAVAILABLE",
		"EXTERNAL_SERVICE_ERROR",
		"PROVIDER_UNAVAILABLE",
		"PROVIDER_ERROR":
		return true
	}
	return false
}

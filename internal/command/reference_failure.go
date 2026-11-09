package command

import (
	"errors"
	"fmt"
)

// ReferenceFailureCode names why a reference execution refused a request. The
// codes exist so a caller can report the business outcome without matching on
// message text.
type ReferenceFailureCode string

const (
	// ReferenceRecordMissing is a request that names a record the store does not
	// hold. It is a normal business outcome, not an execution fault.
	ReferenceRecordMissing ReferenceFailureCode = "record-missing"
	// ReferenceRecordExists is a create of a record that already exists.
	ReferenceRecordExists ReferenceFailureCode = "record-exists"
	// ReferenceValueRejected is a value outside the declared domain, a required
	// value that is absent, an immutable value that changed, or a duplicate
	// candidate key.
	ReferenceValueRejected ReferenceFailureCode = "value-rejected"
	// ReferenceRuleRejected is a declared policy that refused the change, such
	// as a delete restricted by an inbound relation.
	ReferenceRuleRejected ReferenceFailureCode = "rule-rejected"
	// ReferenceUndeclared is a request the declaration cannot describe.
	ReferenceUndeclared ReferenceFailureCode = "undeclared"
)

// ReferenceFailure is a refusal the reference store decided. It is not an
// execution fault: the store understood the request and answered no.
type ReferenceFailure struct {
	Code      ReferenceFailureCode
	Operation string
	Message   string
}

func (f *ReferenceFailure) Error() string {
	if f.Operation == "" {
		return string(f.Code) + ": " + f.Message
	}
	return fmt.Sprintf("%s: %s (%s)", f.Code, f.Message, f.Operation)
}

func referenceFailure(code ReferenceFailureCode, operation, format string, args ...any) *ReferenceFailure {
	return &ReferenceFailure{Code: code, Operation: operation, Message: fmt.Sprintf(format, args...)}
}

// FailureOutcome names the business outcome a refusal corresponds to. It
// returns false when the error is not a reference refusal, so an execution
// fault is not reported as a business outcome.
func FailureOutcome(err error) (string, bool) {
	var failure *ReferenceFailure
	if !errors.As(err, &failure) {
		return "", false
	}
	switch failure.Code {
	case ReferenceRecordMissing:
		return "not-found", true
	case ReferenceRecordExists:
		return "already-exists", true
	case ReferenceValueRejected:
		return "value-rejected", true
	case ReferenceRuleRejected:
		return "rule-rejected", true
	case ReferenceUndeclared:
		return "undeclared", true
	default:
		return string(failure.Code), true
	}
}

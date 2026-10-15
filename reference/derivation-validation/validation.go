// Package derivationvalidation is a D15 read-only reference boundary.
// It feeds one accepted pure derived value into an existing business
// validation and keeps the derivation phase and the validation phase
// distinct, so that "cannot derive" is never reduced to "not satisfied".
//
// It does not define the D1 rule contract, the D4 target binding, the D5
// authorization decision, the D11 snapshot, the D13 batch result or the
// commit/rollback owned by #1129/#1174.
package derivationvalidation

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"tei/internal/semantic/derivationref"
)

var (
	ErrInvalidRef       = errors.New("derivation validation reference is invalid")
	ErrLimitUnavailable = errors.New("validation limit is unavailable")
	ErrStalePreview     = errors.New("validation preview is stale")
	ErrPhase            = errors.New("validation phase is invalid")
)

// DerivedRef is how an existing validation refers to a derived value. Only the
// explicit Rule ID and the typed output ID identify it; Name is a label and is
// never used for resolution, so a same-named attribute cannot substitute.
type DerivedRef struct {
	RuleID   string
	OutputID string
	Name     string
}

// ValidateRef rejects an operand that does not carry an explicit typed
// identity. It does not resolve the rule; resolution belongs to D4.
func ValidateRef(ref DerivedRef) error {
	if ref.RuleID == "" || ref.OutputID == "" ||
		ref.RuleID != strings.TrimSpace(ref.RuleID) || ref.OutputID != strings.TrimSpace(ref.OutputID) {
		return fmt.Errorf("%w: rule and typed output are required", ErrInvalidRef)
	}
	return nil
}

// DerivationState is the phase-1 result of obtaining the derived value.
type DerivationState string

const (
	DerivationValue       DerivationState = "value"
	DerivationNull        DerivationState = "null"
	DerivationUnavailable DerivationState = "input-unavailable"
	DerivationFailure     DerivationState = "evaluation-failure"
	DerivationDenied      DerivationState = "authorization-denied"
)

// ValidationState is the phase-2 business-constraint result. It is evaluated
// only when the derivation produced a comparable value.
type ValidationState string

const (
	ValidationSatisfied    ValidationState = "satisfied"
	ValidationNotSatisfied ValidationState = "not-satisfied"
	ValidationNotEvaluated ValidationState = "not-evaluated"
)

// Result keeps the two phases separate. A caller must read Validation only
// when Derivation is DerivationValue.
type Result struct {
	Derivation DerivationState `json:"derivation"`
	Validation ValidationState `json:"validation"`
	Value      string          `json:"value,omitempty"`
	Limit      string          `json:"limit,omitempty"`
}

// Check supplies an already evaluated derived value to the existing "derived
// <= limit" constraint. A missing input, a failed evaluation, a declared NULL
// or an authorization denial are not a business "not-satisfied": they leave
// the validation not evaluated. denied comes from D5; this reference never
// decides authorization itself.
func Check(derived derivationref.Input, derivedErr error, denied bool, limit derivationref.Input) (Result, error) {
	if denied {
		return Result{Derivation: DerivationDenied, Validation: ValidationNotEvaluated}, nil
	}
	if derivedErr != nil {
		state := DerivationFailure
		if errors.Is(derivedErr, derivationref.ErrInput) {
			state = DerivationUnavailable
		}
		return Result{Derivation: state, Validation: ValidationNotEvaluated}, nil
	}
	switch derived.State {
	case derivationref.Null:
		return Result{Derivation: DerivationNull, Validation: ValidationNotEvaluated}, nil
	case derivationref.Present:
		// continue below
	case derivationref.Missing, derivationref.AbsentRelation, derivationref.MultipleRelations:
		return Result{Derivation: DerivationUnavailable, Validation: ValidationNotEvaluated}, nil
	default:
		return Result{Derivation: DerivationFailure, Validation: ValidationNotEvaluated}, nil
	}

	value, ok := new(big.Rat).SetString(derived.Decimal)
	if !ok {
		return Result{Derivation: DerivationFailure, Validation: ValidationNotEvaluated}, nil
	}
	limitValue, ok := new(big.Rat).SetString(limit.Decimal)
	if limit.State != derivationref.Present || !ok {
		return Result{Derivation: DerivationValue, Validation: ValidationNotEvaluated}, ErrLimitUnavailable
	}
	result := Result{
		Derivation: DerivationValue,
		Value:      derived.Decimal,
		Limit:      limit.Decimal,
	}
	if value.Cmp(limitValue) <= 0 {
		result.Validation = ValidationSatisfied
	} else {
		result.Validation = ValidationNotSatisfied
	}
	return result, nil
}

// Phase distinguishes an unsaved preview from a commit-time decision.
type Phase string

const (
	PreviewPhase Phase = "preview"
	CommitPhase  Phase = "commit"
)

// Decision records a validation result together with the input version it was
// computed against and the phase. A preview decision is not evidence for a
// saved business constraint.
type Decision struct {
	Result       Result `json:"result"`
	InputVersion string `json:"inputVersion"`
	Phase        Phase  `json:"phase"`
}

// RecheckAtCommit re-evaluates at commit time and refuses to reuse a preview
// whose input version changed. It performs no commit, rollback or notification
// (#1129/#1174). recheck is the fresh result computed against currentVersion.
func RecheckAtCommit(preview Decision, currentVersion string, recheck Result) (Decision, error) {
	if preview.Phase != PreviewPhase {
		return Decision{}, fmt.Errorf("%w: not a preview decision", ErrPhase)
	}
	if preview.InputVersion == "" || currentVersion == "" {
		return Decision{}, fmt.Errorf("%w: input version", ErrInvalidRef)
	}
	if preview.InputVersion != currentVersion {
		return Decision{}, fmt.Errorf("%w: %s -> %s", ErrStalePreview, preview.InputVersion, currentVersion)
	}
	return Decision{Result: recheck, InputVersion: currentVersion, Phase: CommitPhase}, nil
}

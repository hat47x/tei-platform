// Package derivedvalidation is a read-only, synthetic reference for Data
// Derivation D15 (#1729). It checks the order and the state distinctions for
// feeding a pure derived value into an existing business validation.
//
// It is not a validation engine, an expression language, a lifecycle runner, a
// transaction manager, or an accepted API. It does not write a Source and does
// not execute a business action. A derivation that cannot be evaluated is never
// collapsed into "validation not satisfied".
package derivedvalidation

import (
	"fmt"
	"math/big"
	"strings"
)

// Result codes. The distinct states are the point of this reference.
const (
	CodeValidationSatisfied        = "VALIDATION_SATISFIED"
	CodeValidationNotSatisfied     = "VALIDATION_NOT_SATISFIED"
	CodeDerivationInputUnavailable = "DERIVATION_INPUT_UNAVAILABLE"
	CodeDerivationExecutionFailure = "DERIVATION_EXECUTION_FAILURE"
	CodeDerivationNullUnspecified  = "DERIVATION_NULL_UNSPECIFIED"
	CodeLimitUnavailable           = "LIMIT_UNAVAILABLE"
	CodeAuthorizationDenied        = "AUTHORIZATION_DENIED"
	CodeDependencyCycle            = "DEPENDENCY_CYCLE"
	CodeAmbiguousReference         = "AMBIGUOUS_REFERENCE"
	CodeInvalidRequest             = "INVALID_REQUEST"
)

// Derivation states.
const (
	DerivationPresent          = "present"
	DerivationNull             = "null"
	DerivationInputUnavailable = "input-unavailable"
	DerivationExecutionFailure = "execution-failure"
)

// Phases.
const (
	PhasePreview = "preview"
	PhaseSave    = "save"
)

// Derivation is the already-evaluated pure derived value.
type Derivation struct {
	State  string `json:"state"`
	Amount string `json:"amount,omitempty"`
}

// Operand is the other typed input of the validation, such as a credit limit.
type Operand struct {
	State string `json:"state"`
	Value string `json:"value,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	// ReferenceByID is true only when the validation references the derived value
	// by an explicit identity, never by a shared attribute name.
	ReferenceByID bool `json:"referenceByID"`
	// Cycle is true when the validation makes the same pure rule depend on itself.
	Cycle         bool       `json:"cycle,omitempty"`
	Authorization string     `json:"authorization"`
	Derivation    Derivation `json:"derivation"`
	Limit         Operand    `json:"limit"`
	Phase         string     `json:"phase"`
}

// Report is the read-only result. This reference never performs a save or
// observes a commit; SaveGuaranteed remains for JSON compatibility but is false.
type Report struct {
	Code            string `json:"code"`
	DerivationState string `json:"derivationState,omitempty"`
	Validation      string `json:"validation"`
	SaveGuaranteed  bool   `json:"saveGuaranteed"`
	Message         string `json:"message,omitempty"`
}

// Evaluate orders the checks and returns one distinct state. It never writes
// anything and never lets a preview result stand in for a save-time decision.
func Evaluate(in Input) Report {
	report := Report{DerivationState: in.Derivation.State, Validation: "not-evaluated"}

	if strings.TrimSpace(in.Authorization) == "denied" {
		report.Code = CodeAuthorizationDenied
		report.Message = "認可されていないため業務制約を評価しない。"
		return report
	}
	if in.Cycle {
		report.Code = CodeDependencyCycle
		report.Message = "純粋規則への循環依存を実行前に拒否する。"
		return report
	}
	if !in.ReferenceByID {
		report.Code = CodeAmbiguousReference
		report.Message = "明示IDに基づく参照以外は拒否する。同名属性の参照は対象を誤る。"
		return report
	}

	switch strings.TrimSpace(in.Derivation.State) {
	case DerivationInputUnavailable:
		report.Code = CodeDerivationInputUnavailable
		report.Message = "導出の入力未取得。Validation not satisfied と区別する。"
		return report
	case DerivationExecutionFailure:
		report.Code = CodeDerivationExecutionFailure
		report.Message = "導出の実行失敗。Validation not satisfied と区別する。"
		return report
	case DerivationNull:
		report.Code = CodeDerivationNullUnspecified
		report.Message = "NULL方針が未定義。金額0やValidation not satisfiedへ黙って変換しない。"
		return report
	case DerivationPresent:
	default:
		report.Code = CodeInvalidRequest
		report.Message = fmt.Sprintf("invalid derivation state %q", in.Derivation.State)
		return report
	}

	if strings.TrimSpace(in.Limit.State) != "present" {
		report.Code = CodeLimitUnavailable
		report.Message = "与信上限などの入力が未取得。"
		return report
	}

	amount, ok := new(big.Rat).SetString(strings.TrimSpace(in.Derivation.Amount))
	if !ok {
		report.Code = CodeInvalidRequest
		report.Message = fmt.Sprintf("invalid amount %q", in.Derivation.Amount)
		return report
	}
	limit, ok := new(big.Rat).SetString(strings.TrimSpace(in.Limit.Value))
	if !ok {
		report.Code = CodeInvalidRequest
		report.Message = fmt.Sprintf("invalid limit %q", in.Limit.Value)
		return report
	}

	if amount.Cmp(limit) > 0 {
		report.Code = CodeValidationNotSatisfied
		report.Validation = "not-satisfied"
		report.Message = "導出は成功したが業務制約を満たさない。"
		return report
	}
	report.Code = CodeValidationSatisfied
	report.Validation = "satisfied"
	if strings.TrimSpace(in.Phase) == PhaseSave {
		report.Message = "保存時相当の合成Validation成立。実Action・Commit・保存結果を観測していないため保存保証ではない。"
	} else {
		report.Message = "Preview上の成立であり、保存時点の確定判定ではない。保存Actionで現行版を再評価する。"
	}
	return report
}

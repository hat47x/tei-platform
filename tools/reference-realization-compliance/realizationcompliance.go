// Package realizationcompliance is a read-only, synthetic reference for Data
// Derivation D6 (#1694). It checks whether one realization of a rule keeps the
// rule's business meaning: output type, NULL policy, rounding, purity, the
// locked identity, and the shared fixture oracle results.
//
// It is not an Operator registry, a Plugin Lock authority, a Build API, a
// general expression engine, or an accepted API. It does not write a Source. It
// never falls back silently to another implementation on a mismatch, never
// accepts a rounding-mode or NULL-policy change as an "equivalent
// replacement", and never treats a synthetic fixture pass as a general
// compliance proof.
package realizationcompliance

import (
	"fmt"
	"math/big"
	"strings"
)

// Result codes.
const (
	CodeCompliant                  = "COMPLIANT"
	CodeEquivalenceUnproven        = "EQUIVALENCE_UNPROVEN"
	CodeFixtureMismatch            = "FIXTURE_MISMATCH"
	CodeTypeMismatch               = "TYPE_MISMATCH"
	CodeNullPolicyMismatch         = "NULL_POLICY_MISMATCH"
	CodeMeaningMismatch            = "MEANING_MISMATCH"
	CodeImpureRealization          = "IMPURE_REALIZATION"
	CodeRealizationNotRegistered   = "REALIZATION_NOT_REGISTERED"
	CodeRealizationLockMissing     = "REALIZATION_LOCK_MISSING"
	CodeRealizationVersionMismatch = "REALIZATION_VERSION_MISMATCH"
	CodeInvalidRequest             = "INVALID_REQUEST"
)

// Element states.
const (
	StatePresent = "present"
	StateNull    = "null"
	StateMissing = "missing"
)

// NULL policies.
const (
	NullPropagate = "propagate"
	NullZero      = "zero"
)

// Rounding modes.
const (
	RoundHalfEven = "half-even"
	RoundHalfUp   = "half-up"
)

// Realization is one candidate implementation of the rule.
type Realization struct {
	Plugin     string `json:"plugin"`
	Version    string `json:"version"`
	Language   string `json:"language"`
	OutputType string `json:"outputType"`
	NullPolicy string `json:"nullPolicy"`
	Rounding   string `json:"rounding"`
	Pure       bool   `json:"pure"`
	// Realization identity state.
	Registered    bool   `json:"registered"`
	Locked        bool   `json:"locked"`
	LockedVersion string `json:"lockedVersion,omitempty"`
	// Cases are the shared fixture oracle results this realization reports.
	Cases []Case `json:"cases"`
}

// Case is one fixture point with the realization's reported result.
type Case struct {
	QuantityState string `json:"quantityState"`
	Quantity      string `json:"quantity,omitempty"`
	UnitPrice     string `json:"unitPrice,omitempty"`
	// ResultState is what the realization reports for this input.
	ResultState string `json:"resultState"`
	// Amount is the realization's reported value, empty for a null result.
	Amount string `json:"amount,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	RuleID         string        `json:"ruleId"`
	RuleOutputType string        `json:"ruleOutputType"`
	RuleNullPolicy string        `json:"ruleNullPolicy"`
	RuleRounding   string        `json:"ruleRounding"`
	RulePure       bool          `json:"rulePure"`
	Realizations   []Realization `json:"realizations"`
}

// Report is the read-only result.
type Report struct {
	Code                string   `json:"code"`
	Accepted            bool     `json:"accepted"`
	RuleID              string   `json:"ruleId,omitempty"`
	Implementations     []string `json:"implementations,omitempty"`
	EquivalenceUnproven bool     `json:"equivalenceUnproven,omitempty"`
	Message             string   `json:"message,omitempty"`
}

func reject(code, message string) Report {
	return Report{Code: code, Message: message}
}

func product(quantity, unitPrice string) (*big.Rat, error) {
	left, ok := new(big.Rat).SetString(strings.TrimSpace(quantity))
	if !ok {
		return nil, fmt.Errorf("invalid quantity %q", quantity)
	}
	right, ok := new(big.Rat).SetString(strings.TrimSpace(unitPrice))
	if !ok {
		return nil, fmt.Errorf("invalid unit price %q", unitPrice)
	}
	return left.Mul(left, right), nil
}

// expected returns the oracle state for one case under the rule's declared
// meaning.
func expected(ruleNullPolicy string, c Case) string {
	switch strings.TrimSpace(c.QuantityState) {
	case StateMissing:
		return StateMissing
	case StateNull:
		if ruleNullPolicy == NullZero {
			return StatePresent
		}
		return StateNull
	default:
		return StatePresent
	}
}

// checkOne verifies one realization against the rule's declared meaning.
func checkOne(r Realization, in Input) string {
	if !r.Registered {
		return CodeRealizationNotRegistered
	}
	if !r.Locked {
		return CodeRealizationLockMissing
	}
	if r.LockedVersion != "" && r.Version != r.LockedVersion {
		return CodeRealizationVersionMismatch
	}
	if in.RulePure && !r.Pure {
		return CodeImpureRealization
	}
	if r.OutputType != strings.TrimSpace(in.RuleOutputType) {
		return CodeTypeMismatch
	}
	if r.NullPolicy != strings.TrimSpace(in.RuleNullPolicy) {
		return CodeNullPolicyMismatch
	}
	if r.Rounding != strings.TrimSpace(in.RuleRounding) {
		return CodeMeaningMismatch
	}
	ruleNullPolicy := strings.TrimSpace(in.RuleNullPolicy)
	for _, c := range r.Cases {
		wantState := expected(ruleNullPolicy, c)
		if c.ResultState != wantState {
			return CodeFixtureMismatch
		}
		if wantState != StatePresent {
			continue
		}
		oracle, err := product(c.Quantity, c.UnitPrice)
		if err != nil {
			return CodeFixtureMismatch
		}
		reported, ok := new(big.Rat).SetString(strings.TrimSpace(c.Amount))
		if !ok {
			return CodeFixtureMismatch
		}
		if oracle.Cmp(reported) != 0 {
			return CodeFixtureMismatch
		}
	}
	return ""
}

// equivalentCases compares two realizations' reported fixture results.
func equivalentCases(one []Case, other []Case) bool {
	if len(one) != len(other) {
		return false
	}
	for i := range one {
		if one[i].ResultState != other[i].ResultState {
			return false
		}
		if one[i].ResultState != StatePresent {
			continue
		}
		left, ok := new(big.Rat).SetString(strings.TrimSpace(one[i].Amount))
		if !ok {
			return false
		}
		right, ok := new(big.Rat).SetString(strings.TrimSpace(other[i].Amount))
		if !ok {
			return false
		}
		if left.Cmp(right) != 0 {
			return false
		}
	}
	return true
}

func contractMessage(code string, r Realization) string {
	switch code {
	case CodeRealizationNotRegistered:
		return fmt.Sprintf("未登録Plugin %q へ落とさない。", r.Plugin)
	case CodeRealizationLockMissing:
		return fmt.Sprintf("Plugin %q の固定版が欠落。", r.Plugin)
	case CodeRealizationVersionMismatch:
		return fmt.Sprintf("Plugin %q の版 %q が固定版と不一致。黙って落とさない。", r.Plugin, r.Version)
	case CodeImpureRealization:
		return "pure な規則に外部書込みを要求する実現は選ばない。"
	case CodeTypeMismatch:
		return fmt.Sprintf("出力型 %q が規則の契約と不一致。", r.OutputType)
	case CodeNullPolicyMismatch:
		return "NULL方針の差違は実装差替えではなく契約変更として再受理する。"
	case CodeMeaningMismatch:
		return fmt.Sprintf("丸めモード %q は規則の宣言と異なる。同等実装の差替えとして認めない。", r.Rounding)
	default:
		return "実現のfixture結果が契約のOracleと食い違う。"
	}
}

// Check verifies the rule-to-realization selection. It never writes anything.
func Check(in Input) Report {
	if strings.TrimSpace(in.RuleID) == "" {
		return reject(CodeInvalidRequest, "ruleId が未指定。")
	}
	if len(in.Realizations) == 0 {
		return reject(CodeInvalidRequest, "検査する実現が未指定。")
	}

	primary := in.Realizations[0]
	if code := checkOne(primary, in); code != "" {
		return reject(code, contractMessage(code, primary))
	}

	report := Report{
		Code:            CodeCompliant,
		Accepted:        true,
		RuleID:          in.RuleID,
		Implementations: []string{primary.Language},
	}
	if len(in.Realizations) == 1 {
		report.EquivalenceUnproven = true
		report.Message = "一つの実現はfixtureに合格した。第二実装の同等性は未証明。"
		return report
	}
	for _, other := range in.Realizations[1:] {
		if code := checkOne(other, in); code != "" {
			return reject(code, contractMessage(code, other))
		}
		report.Implementations = append(report.Implementations, other.Language)
		if !equivalentCases(primary.Cases, other.Cases) {
			return reject(CodeEquivalenceUnproven,
				fmt.Sprintf("実現 %q はNULL・型・精度のいずれかで実現 %q と食い違う。同等性を証明するまで実現を差替えない。", other.Plugin, primary.Plugin))
		}
	}
	report.Message = "実現identityは規則identityと独立。意味契約を固定した差替えが可能。"
	return report
}

package quantityarith

import (
	"fmt"
	"sort"
)

// Delegation schema and statuses. A target that cannot keep the declared
// precision, scale, rounding and exact-decimal requirement is not delegated to;
// the reference never emits a CAST or rewrites a SQL expression.
const (
	DelegationSchema   = "tei.reference.quantity-delegation/v0"
	DelegationReady    = "DELEGATION_READY"
	DelegationUnproven = "DELEGATION_UNPROVEN"
)

// TargetCapability is what a realization target declares it can do. A zero
// value declares nothing, and nothing is inferred from a target name.
type TargetCapability struct {
	Name          string   `json:"name"`
	MaxPrecision  int      `json:"maxPrecision"`
	MaxScale      int      `json:"maxScale"`
	RoundingModes []string `json:"roundingModes,omitempty"`
	ExactDecimal  bool     `json:"exactDecimal"`
}

// DelegationInput asks whether the declared result type can be delegated to the
// target without changing the business result.
type DelegationInput struct {
	Target     TargetCapability          `json:"target"`
	ResultType string                    `json:"resultType"`
	Types      map[string]TypeAnnotation `json:"types"`
	Rounding   string                    `json:"rounding,omitempty"`
}

// DelegationReport records why a delegation is unproven. The reasons are
// additive so the caller sees every unmet condition, not only the first.
type DelegationReport struct {
	Schema  string   `json:"schema"`
	Target  string   `json:"target"`
	Status  string   `json:"status"`
	Reasons []string `json:"reasons,omitempty"`
	Note    string   `json:"note"`
}

// CheckDelegation checks whether the declared result type and rounding can be
// delegated to the target. A missing declaration stays unproven instead of being
// assumed equal to Go.
func CheckDelegation(in DelegationInput) DelegationReport {
	report := DelegationReport{
		Schema: DelegationSchema,
		Target: in.Target.Name,
		Note:   "宣言された型・桁・丸めとTargetの能力を比べるだけです。CASTの生成やSQLの書換えはしません。",
	}
	var reasons []string

	if in.Target.Name == "" {
		reasons = append(reasons, "target-not-declared")
	}
	if !in.Target.ExactDecimal {
		reasons = append(reasons, "target-not-exact-decimal")
	}
	if in.ResultType == "" {
		reasons = append(reasons, "result-type-not-declared")
	} else if annotation, ok := in.Types[in.ResultType]; !ok {
		reasons = append(reasons, fmt.Sprintf("result-type-unknown:%s", in.ResultType))
	} else {
		switch {
		case annotation.Precision == nil:
			// A store or delegation without a declared precision would let the
			// target pick one silently.
			reasons = append(reasons, "precision-not-declared")
		case *annotation.Precision > in.Target.MaxPrecision:
			reasons = append(reasons, fmt.Sprintf("precision-exceeds-target:%d>%d", *annotation.Precision, in.Target.MaxPrecision))
		}
		switch {
		case annotation.Scale == nil:
			reasons = append(reasons, "scale-not-declared")
		case *annotation.Scale > in.Target.MaxScale:
			reasons = append(reasons, fmt.Sprintf("scale-exceeds-target:%d>%d", *annotation.Scale, in.Target.MaxScale))
		}
	}

	if rounding := normalizeRounding(in.Rounding); rounding != RoundHalfUp && rounding != RoundHalfEven {
		reasons = append(reasons, fmt.Sprintf("rounding-mode-unknown:%s", in.Rounding))
	} else if !containsString(in.Target.RoundingModes, rounding) {
		reasons = append(reasons, fmt.Sprintf("rounding-mode-unsupported:%s", rounding))
	}

	sort.Strings(reasons)
	report.Reasons = reasons
	if len(reasons) == 0 {
		report.Status = DelegationReady
	} else {
		report.Status = DelegationUnproven
	}
	return report
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

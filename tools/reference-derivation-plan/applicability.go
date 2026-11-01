package derivationplan

import (
	"fmt"
	"sort"
)

// Applicability schema and codes. A plan is delegated to SQL only when the
// provider's meaning matches; otherwise it is evaluated after a complete fetch,
// or declared unsupported. The reference never falls back silently.
const (
	ApplicabilitySchema = "tei.reference.derivation-plan-applicability/v0"

	CodeDelegateToSQL         = "DELEGATE_TO_SQL"
	CodeEvaluateAfterFetch    = "EVALUATE_AFTER_FETCH"
	CodeDelegationUnsupported = "DELEGATION_UNSUPPORTED"
)

// ApplicabilityInput declares what the provider and the fetch can guarantee.
// Zero values mean "unproven", never "supported".
type ApplicabilityInput struct {
	// ProviderNullSemantics is true when the provider's NULL comparison matches
	// the rule's declared NULL policy.
	ProviderNullSemantics bool `json:"providerNullSemantics"`
	// ProviderDecimalPrecision is true when the provider keeps the declared
	// decimal precision and rounding.
	ProviderDecimalPrecision bool `json:"providerDecimalPrecision"`
	// ProviderRelationMultiplicity is true when the provider's join does not
	// multiply the business rows.
	ProviderRelationMultiplicity bool `json:"providerRelationMultiplicity"`
	// ProviderAuthority is true when the provider runs under the same
	// item-level authority as the requester.
	ProviderAuthority bool `json:"providerAuthority"`
	// LimitBeforeDerive is true when the plan limits rows before deriving the
	// value the later steps use.
	LimitBeforeDerive bool `json:"limitBeforeDerive"`
	// UnsupportedExpression is true when the rule cannot be expressed in SQL.
	UnsupportedExpression bool `json:"unsupportedExpression"`
	// FetchComplete is true when every row the later steps need is fetched, and
	// the fetch is bounded.
	FetchComplete bool `json:"fetchComplete"`
	// FetchBounded is true when the complete fetch has a declared bound.
	FetchBounded bool `json:"fetchBounded"`
}

// ApplicabilityReport records the decision and every unmet condition.
type ApplicabilityReport struct {
	Schema  string   `json:"schema"`
	Code    string   `json:"code"`
	Reasons []string `json:"reasons,omitempty"`
	Message string   `json:"message,omitempty"`
}

// CheckApplicability decides whether the plan can be delegated to SQL or must be
// evaluated after a complete, bounded fetch. It returns every unmet condition so
// a caller can fix the contract instead of relying on a silent fallback.
func CheckApplicability(in ApplicabilityInput) ApplicabilityReport {
	report := ApplicabilityReport{
		Schema:  ApplicabilitySchema,
		Message: "宣言された保証だけを比べる。保証できない方式へ黙って落とさない。",
	}
	var reasons []string

	if in.UnsupportedExpression {
		reasons = append(reasons, "expression-not-expressible-in-sql")
	}
	if in.LimitBeforeDerive {
		reasons = append(reasons, "limit-before-derive")
	}
	if !in.ProviderNullSemantics {
		reasons = append(reasons, "provider-null-semantics-unproven")
	}
	if !in.ProviderDecimalPrecision {
		reasons = append(reasons, "provider-decimal-precision-unproven")
	}
	if !in.ProviderRelationMultiplicity {
		reasons = append(reasons, "provider-relation-multiplicity-unproven")
	}
	if !in.ProviderAuthority {
		reasons = append(reasons, "provider-authority-unproven")
	}

	sort.Strings(reasons)
	if len(reasons) == 0 && !in.UnsupportedExpression {
		report.Code = CodeDelegateToSQL
		return report
	}

	// Post-fetch evaluation is only allowed when the whole required row set is
	// fetched and the fetch has a declared bound. It still cannot repair a
	// limit-before-derive plan, because rows were already cut.
	var fetchReasons []string
	if !in.FetchComplete {
		fetchReasons = append(fetchReasons, "fetch-not-complete")
	}
	if !in.FetchBounded {
		fetchReasons = append(fetchReasons, "fetch-not-bounded")
	}
	if in.LimitBeforeDerive {
		fetchReasons = append(fetchReasons, "limit-before-derive")
	}
	if in.UnsupportedExpression {
		fetchReasons = append(fetchReasons, "expression-not-evaluable")
	}
	if len(fetchReasons) == 0 {
		report.Code = CodeEvaluateAfterFetch
		report.Reasons = reasons
		return report
	}

	sort.Strings(fetchReasons)
	report.Code = CodeDelegationUnsupported
	report.Reasons = uniqueStrings(append(reasons, fetchReasons...))
	report.Message = fmt.Sprintf("SQL移譲も取得後評価も保証できない: %v", report.Reasons)
	return report
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

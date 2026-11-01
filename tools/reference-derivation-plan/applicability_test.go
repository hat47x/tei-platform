package derivationplan

import (
	"reflect"
	"testing"
)

func supportedProvider() ApplicabilityInput {
	return ApplicabilityInput{
		ProviderNullSemantics:        true,
		ProviderDecimalPrecision:     true,
		ProviderRelationMultiplicity: true,
		ProviderAuthority:            true,
		FetchComplete:                true,
		FetchBounded:                 true,
	}
}

func hasReason(report ApplicabilityReport, want string) bool {
	for _, reason := range report.Reasons {
		if reason == want {
			return true
		}
	}
	return false
}

func TestFullySupportedPlanIsDelegatedToSQL(t *testing.T) {
	report := CheckApplicability(supportedProvider())
	if report.Code != CodeDelegateToSQL {
		t.Fatalf("report=%+v want %s", report, CodeDelegateToSQL)
	}
	if len(report.Reasons) != 0 {
		t.Fatalf("reasons=%v", report.Reasons)
	}
}

func TestUnprovenProviderFallsBackToCompleteBoundedFetch(t *testing.T) {
	in := supportedProvider()
	in.ProviderNullSemantics = false
	in.ProviderDecimalPrecision = false
	in.ProviderRelationMultiplicity = false
	in.ProviderAuthority = false
	report := CheckApplicability(in)
	if report.Code != CodeEvaluateAfterFetch {
		t.Fatalf("report=%+v want %s", report, CodeEvaluateAfterFetch)
	}
	if !hasReason(report, "provider-null-semantics-unproven") {
		t.Fatalf("reasons=%v", report.Reasons)
	}
}

func TestNeitherPathIsProvenIsUnsupported(t *testing.T) {
	in := supportedProvider()
	in.ProviderNullSemantics = false
	in.ProviderDecimalPrecision = false
	in.ProviderRelationMultiplicity = false
	in.ProviderAuthority = false
	in.FetchComplete = false
	report := CheckApplicability(in)
	if report.Code != CodeDelegationUnsupported {
		t.Fatalf("report=%+v want %s", report, CodeDelegationUnsupported)
	}
	for _, want := range []string{"provider-null-semantics-unproven", "fetch-not-complete"} {
		if !hasReason(report, want) {
			t.Errorf("missing reason %q in %v", want, report.Reasons)
		}
	}
}

func TestLimitBeforeDeriveCannotBeRepairedByPostFetch(t *testing.T) {
	in := supportedProvider()
	in.LimitBeforeDerive = true
	report := CheckApplicability(in)
	if report.Code != CodeDelegationUnsupported {
		t.Fatalf("report=%+v want %s (rows were already cut)", report, CodeDelegationUnsupported)
	}
	if !hasReason(report, "limit-before-derive") {
		t.Fatalf("reasons=%v", report.Reasons)
	}
}

func TestUnsupportedExpressionNeverSilentlyFallsBack(t *testing.T) {
	in := supportedProvider()
	in.UnsupportedExpression = true
	report := CheckApplicability(in)
	if report.Code != CodeDelegationUnsupported {
		t.Fatalf("report=%+v want %s", report, CodeDelegationUnsupported)
	}
	if !hasReason(report, "expression-not-expressible-in-sql") {
		t.Fatalf("reasons=%v", report.Reasons)
	}
}

func TestUnboundedFetchIsNotAcceptedAsPostFetch(t *testing.T) {
	in := supportedProvider()
	in.ProviderNullSemantics = false
	in.FetchBounded = false
	report := CheckApplicability(in)
	if report.Code == CodeEvaluateAfterFetch {
		t.Fatalf("an unbounded fetch was accepted: %+v", report)
	}
	if !hasReason(report, "fetch-not-bounded") {
		t.Fatalf("reasons=%v", report.Reasons)
	}
}

func TestApplicabilityIsDeterministic(t *testing.T) {
	in := supportedProvider()
	in.ProviderAuthority = false
	in.ProviderDecimalPrecision = false
	first := CheckApplicability(in)
	second := CheckApplicability(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not deterministic: %+v != %+v", first, second)
	}
	if len(first.Reasons) != 2 {
		t.Fatalf("reasons=%v want every unmet condition", first.Reasons)
	}
}

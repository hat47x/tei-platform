package derivationcache

import "testing"

// The fixtures follow Data Derivation D3 (#1641): quantity 2 × unit price 300
// gives 600; stock 120 - reserved 100 gives 20.

func TestSimplePureComputationNeedsNoCacheContract(t *testing.T) {
	report := Reuse(Input{CacheRequested: false, NewComputation: "600"})
	if !report.Accepted || report.Code != CodeDirectEvaluation || report.Value != "600" {
		t.Fatalf("report=%+v want DIRECT_EVALUATION 600", report)
	}
}

func TestStaleInputIsNotReturnedAsCurrent(t *testing.T) {
	report := Reuse(Input{
		CacheRequested: true,
		Entry:          &Entry{RuleVersion: "1", InputVersion: "R0", ContextClass: "tenantA/admin", SourceTimePoint: "t0", Value: "20"},
		Current:        &Context{RuleVersion: "1", InputVersion: "R1", ContextClass: "tenantA/admin", SourceTimePoint: "t0"},
	})
	if report.Accepted || report.Code != CodeStaleInput || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_INPUT with unknown value", report)
	}
}

func TestConfirmedSavedValueSurvivesMasterChange(t *testing.T) {
	report := Reuse(Input{
		CacheRequested: true,
		Entry:          &Entry{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0", Value: "600"},
		Current:        &Context{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0"},
		ConfirmedSaved: true,
		ConfirmedValue: "600",
		NewComputation: "700",
	})
	if !report.Accepted || report.Code != CodeConfirmedSavedImmutable || report.Value != "600" {
		t.Fatalf("report=%+v want CONFIRMED_SAVED_IMMUTABLE 600", report)
	}
}

func TestStaleRuleIsNotReusedWithoutEvidence(t *testing.T) {
	report := Reuse(Input{
		CacheRequested: true,
		Entry:          &Entry{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0", Value: "600"},
		Current:        &Context{RuleVersion: "2", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0"},
	})
	if report.Accepted || report.Code != CodeStaleRule || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_RULE with unknown value", report)
	}
}

func TestDifferentContextIsNotOneCache(t *testing.T) {
	report := Reuse(Input{
		CacheRequested: true,
		Entry:          &Entry{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0", Value: "600"},
		Current:        &Context{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantB/general", SourceTimePoint: "t0"},
	})
	if report.Accepted || report.Code != CodeContextMismatch || !report.ValueNull {
		t.Fatalf("report=%+v want CONTEXT_MISMATCH with unknown value", report)
	}
}

func TestUnalignedSourcesDoNotPretendStrongConsistency(t *testing.T) {
	report := Reuse(Input{
		CacheRequested:   true,
		Entry:            &Entry{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0", Value: "20"},
		Current:          &Context{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0"},
		MultiSource:      true,
		SnapshotsAligned: false,
	})
	if report.Accepted || report.Code != CodeSnapshotIntegrityUnproven {
		t.Fatalf("report=%+v want SNAPSHOT_INTEGRITY_UNPROVEN", report)
	}
}

func TestFreshCacheIsReused(t *testing.T) {
	report := Reuse(Input{
		CacheRequested: true,
		Entry:          &Entry{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0", Value: "600"},
		Current:        &Context{RuleVersion: "1", InputVersion: "1", ContextClass: "tenantA/admin", SourceTimePoint: "t0"},
	})
	if !report.Accepted || report.Code != CodeCacheReused || report.Value != "600" {
		t.Fatalf("report=%+v want CACHE_REUSED 600", report)
	}
}

func TestMissingEntryOrContextIsRejected(t *testing.T) {
	if report := Reuse(Input{CacheRequested: true}); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}

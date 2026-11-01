package derivationcache

import (
	"reflect"
	"testing"
)

// The D3 fixture: the same rule and input version, but the information source
// moved from t0 to t1. The evaluation time is separate and must not be used as
// a substitute for the source time point.
func baseReuse() Input {
	return Input{
		CacheRequested: true,
		Entry: &Entry{
			RuleVersion: "1", InputVersion: "R0", ContextClass: "tenantA/admin",
			SourceTimePoint: "t0", EvaluatedAt: "2026-10-01T09:00:00Z", Value: "20",
		},
		Current: &Context{
			RuleVersion: "1", InputVersion: "R0", ContextClass: "tenantA/admin",
			SourceTimePoint: "t0", EvaluatedAt: "2026-10-01T10:00:00Z",
		},
	}
}

func TestFreshCacheWithDifferentEvaluationTimeIsReused(t *testing.T) {
	report := Reuse(baseReuse())
	if !report.Accepted || report.Code != CodeCacheReused || report.Value != "20" {
		t.Fatalf("report=%+v want CACHE_REUSED 20", report)
	}
}

func TestSourceTimePointChangeIsStaleEvenWithSameVersions(t *testing.T) {
	in := baseReuse()
	in.Current.SourceTimePoint = "t1"
	in.Current.EvaluatedAt = in.Entry.EvaluatedAt
	report := Reuse(in)
	if report.Accepted || report.Code != CodeStaleSourceTime || !report.ValueNull {
		t.Fatalf("report=%+v want STALE_SOURCE_TIME with unknown value", report)
	}
}

func TestMissingCurrentSourceTimeIsUnprovenNotReused(t *testing.T) {
	in := baseReuse()
	in.Current.SourceTimePoint = ""
	report := Reuse(in)
	if report.Accepted || report.Code != CodeSourceTimeUnspecified || !report.ValueNull {
		t.Fatalf("report=%+v want SOURCE_TIME_UNSPECIFIED with unknown value", report)
	}
}

func TestConfirmedSavedValueIsNotAffectedBySourceTime(t *testing.T) {
	in := baseReuse()
	in.Current.SourceTimePoint = "t1"
	in.ConfirmedSaved = true
	in.ConfirmedValue = "600"
	in.NewComputation = "700"
	report := Reuse(in)
	if !report.Accepted || report.Code != CodeConfirmedSavedImmutable || report.Value != "600" {
		t.Fatalf("report=%+v want CONFIRMED_SAVED_IMMUTABLE 600", report)
	}
}

func TestSourceTimeChecksAreDeterministic(t *testing.T) {
	in := baseReuse()
	in.Current.SourceTimePoint = "t1"
	first := Reuse(in)
	second := Reuse(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not deterministic: %+v != %+v", first, second)
	}
}

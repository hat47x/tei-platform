package facetmappingcheck

import (
	"path/filepath"
	"runtime"
	"testing"
)

// fixtureDir resolves plan/fixtures/facet-mapping from the package directory so
// the test does not depend on the working directory of the caller.
func fixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "plan", "fixtures", "facet-mapping")
}

func loadFixtures(t *testing.T) (CaseFile, ExpectedFile) {
	t.Helper()
	dir := fixtureDir(t)
	cases, err := LoadCases(filepath.Join(dir, "cases.json"))
	if err != nil {
		t.Fatalf("load cases: %v", err)
	}
	expected, err := LoadExpected(filepath.Join(dir, "expected.json"))
	if err != nil {
		t.Fatalf("load expected: %v", err)
	}
	return cases, expected
}

// TestHandWrittenExpectationsMatch is the S1 acceptance: every case in the
// fixture is judged the way the issue's rule table says it should be judged.
func TestHandWrittenExpectationsMatch(t *testing.T) {
	cases, expected := loadFixtures(t)
	report := Check(cases, Options{})
	cmp := Compare(report, expected)
	if cmp.Status != "ok" {
		t.Fatalf("expectations do not match: %+v", cmp.Mismatches)
	}
	if cmp.Matched != cmp.Total || cmp.Total != len(cases.Cases) {
		t.Fatalf("matched %d of %d, allowed %d cases", cmp.Matched, cmp.Total, len(cases.Cases))
	}
}

// TestEveryVerdictAppears keeps the fixture honest: a fixture that only ever
// produces one verdict cannot discriminate between verdicts.
func TestEveryVerdictAppears(t *testing.T) {
	cases, _ := loadFixtures(t)
	report := Check(cases, Options{})
	seen := map[string]int{}
	for _, r := range report.Results {
		seen[r.Verdict]++
	}
	for _, want := range []string{VerdictPreserved, VerdictLossy, VerdictRejected, VerdictUnqualified} {
		if seen[want] == 0 {
			t.Errorf("fixture produces no %q case, so the verdict is not discriminated", want)
		}
	}
}

// mutants are the four known mutants the issue names. Each mutant must break the
// hand-written expectation, otherwise the expectations do not discriminate the
// behaviour they claim to check. The exact cases each mutant breaks are asserted
// in mutant_test.go.
func TestKnownMutantsAreDetected(t *testing.T) {
	cases, expected := loadFixtures(t)
	mutants := map[string]Options{
		"order column ignored":        {IgnoreOrder: true},
		"presence binarized":          {BinarizePresence: true},
		"aggregate writeback allowed": {AllowAggregateInverse: true},
		"quantity scale ignored":      {IgnoreQuantityScale: true},
	}
	for name, opts := range mutants {
		opts := opts
		t.Run(name, func(t *testing.T) {
			report := Check(cases, opts)
			cmp := Compare(report, expected)
			if cmp.Status == "ok" {
				t.Fatalf("mutant %q is not detected: the expectations still match", name)
			}
		})
	}
}

// TestPreservedNeedsEvidence pins the rule that a claim without evidence falls
// to unknown instead of preserved.
func TestPreservedNeedsEvidence(t *testing.T) {
	c := Case{
		ID: "evidence-gap",
		Structure: Structure{
			ID: "s", Family: "collection",
			Facets: Facets{Order: "ordered", OrderOrigin: "source", Multiplicity: "multi", Grain: "line"},
		},
		Representation: Representation{
			ID: "r", Kind: "table", ContextID: "ctx",
			Facets: Facets{Order: "ordered", OrderOrigin: "source", Multiplicity: "multi", Grain: "line"},
		},
		Mapping: Mapping{
			Identity:  IdentityColumn{Kind: "1:1", ClaimedPreserved: true},
			Order:     Claim{ClaimedPreserved: true, Evidence: []string{"rendered in source order"}},
			Multiset:  MultisetColumn{Fanout: 1, ClaimedPreserved: true},
			Grain:     GrainColumn{SourceGrain: "line", TargetGrain: "line", ClaimedPreserved: true, Evidence: []string{"same grain"}},
			Coverage:  CoverageColumn{Claimed: "complete", Observed: "complete"},
			Operation: OperationColumn{Mode: "read-only", InverseEligible: true},
		},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	if len(report.Results) != 1 {
		t.Fatalf("want one result, got %d", len(report.Results))
	}
	got := report.Results[0]
	if got.Verdict != VerdictUnknown {
		t.Fatalf("a claim without evidence must be unknown, got %q", got.Verdict)
	}
	if got.Inverse != InverseIneligible {
		t.Fatalf("an unknown mapping has no usable inverse, got %q", got.Inverse)
	}
	assertHasDiagnostic(t, got, DiagEvidenceMissing)
}

// TestReadOnlySuccessIsNotInverseSuccess pins the separation the issue asks for:
// a row-to-row read-only projection can be preserved and still not offer an
// inverse, and a declared inverse must not be reported eligible when a
// non-invertible loss is present.
func TestReadOnlySuccessIsNotInverseSuccess(t *testing.T) {
	opts := Options{}

	// A: preserved, but the contract never claims an inverse.
	readOnly := Case{
		ID: "A",
		Structure: Structure{ID: "s", Family: "collection",
			Facets: Facets{Order: "ordered", OrderOrigin: "source", Multiplicity: "multi", Grain: "line"}},
		Representation: Representation{ID: "r", Kind: "table", ContextID: "ctx",
			Facets: Facets{Order: "ordered", OrderOrigin: "source", Multiplicity: "multi", Grain: "line"}},
		Mapping: Mapping{
			Identity:  IdentityColumn{Kind: "1:1", ClaimedPreserved: true, Evidence: []string{"id carried"}},
			Order:     Claim{ClaimedPreserved: true, Evidence: []string{"source order rendered"}},
			Multiset:  MultisetColumn{Fanout: 1, ClaimedPreserved: true, Evidence: []string{"no join"}},
			Grain:     GrainColumn{SourceGrain: "line", TargetGrain: "line", ClaimedPreserved: true, Evidence: []string{"same grain"}},
			Coverage:  CoverageColumn{Claimed: "complete", Observed: "complete"},
			Operation: OperationColumn{Mode: "read-only", InverseEligible: false},
		},
	}

	// B: the same projection, but the author declares an inverse while the
	// render drops the ordering the inverse would need.
	lossIDeclaresInverse := readOnly
	lossIDeclaresInverse.ID = "B"
	lossIDeclaresInverse.Representation.Facets.Order = "unordered"
	lossIDeclaresInverse.Representation.Facets.OrderOrigin = "none"
	lossIDeclaresInverse.Mapping.Operation = OperationColumn{Mode: "edit-inverse", InverseEligible: true}

	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{readOnly, lossIDeclaresInverse}}, opts)
	byID := map[string]Result{}
	for _, r := range report.Results {
		byID[r.ID] = r
	}
	if byID["A"].Verdict != VerdictPreserved {
		t.Fatalf("A should be preserved, got %q (%v)", byID["A"].Verdict, byID["A"].Losses)
	}
	if byID["A"].Inverse != InverseNotClaimed {
		t.Fatalf("A must not claim an inverse, got %q", byID["A"].Inverse)
	}
	if byID["B"].Verdict != VerdictLossy {
		t.Fatalf("B should be lossy, got %q", byID["B"].Verdict)
	}
	if byID["B"].Inverse != InverseIneligible {
		t.Fatalf("B declares an inverse while order is lost; got %q", byID["B"].Inverse)
	}
}

func TestParseRejectsUnknownSchema(t *testing.T) {
	if _, err := ParseCases([]byte(`{"schemaVersion":"tei.reference.facet-mapping-input/v9","cases":[]}`)); err == nil {
		t.Fatal("unknown schema version must be rejected")
	}
	if _, err := ParseCases([]byte(`{"schemaVersion":"tei.reference.facet-mapping-input/v0","cases":[]}`)); err == nil {
		t.Fatal("an empty case list must be rejected")
	}
	if _, err := ParseCases([]byte(`{"schemaVersion":"tei.reference.facet-mapping-input/v0","cases":[{"id":"a"},{"id":"a"}]}`)); err == nil {
		t.Fatal("duplicate case ids must be rejected")
	}
	if _, err := ParseExpected([]byte(`{"schemaVersion":"tei.reference.facet-mapping-expected/v0","expectations":[{"id":"a","verdict":"ok"}]}`)); err == nil {
		t.Fatal("a verdict outside the fixed vocabulary must be rejected")
	}
}

func TestCompareReportsMissingCase(t *testing.T) {
	report := Output{SchemaVersion: ReportSchema, Results: []Result{{ID: "a", Verdict: VerdictPreserved}}}
	expected := ExpectedFile{SchemaVersion: ExpectedSchema, Expectations: []Expectation{
		{ID: "a", Verdict: VerdictPreserved},
		{ID: "b", Verdict: VerdictLossy},
	}}
	cmp := Compare(report, expected)
	if cmp.Status == "ok" {
		t.Fatal("a case that was never checked must not compare as ok")
	}
	if len(cmp.Mismatches) != 1 || cmp.Mismatches[0].Field != "missing" {
		t.Fatalf("want one missing mismatch, got %+v", cmp.Mismatches)
	}
}

func assertHasDiagnostic(t *testing.T, r Result, code string) {
	t.Helper()
	for _, d := range r.Diagnostics {
		if d.Code == code {
			return
		}
	}
	t.Fatalf("result %s has no %s diagnostic: %+v", r.ID, code, r.Diagnostics)
}

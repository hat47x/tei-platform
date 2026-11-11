package facetstructurecheck

import (
	"path/filepath"
	"runtime"
	"testing"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "plan", "fixtures", "facet-structure")
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

// TestHandWrittenExpectationsMatch is the S2 acceptance: every family is judged
// the way the issue's DSF table says it should be judged.
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

// TestEveryDestinationAppears keeps the fixture honest: a fixture that only
// produces one destination cannot discriminate the destinations.
func TestEveryDestinationAppears(t *testing.T) {
	cases, _ := loadFixtures(t)
	report := Check(cases, Options{})
	seen := map[string]int{}
	for _, r := range report.Results {
		seen[r.Destination]++
	}
	for _, want := range []string{DestBasic, DestComposite, DestFacet, DestPlugin} {
		if seen[want] == 0 {
			t.Errorf("fixture produces no %q case, so the destination is not discriminated", want)
		}
	}
}

func TestEveryVerdictAppears(t *testing.T) {
	cases, _ := loadFixtures(t)
	report := Check(cases, Options{})
	seen := map[string]int{}
	for _, r := range report.Results {
		seen[r.Verdict]++
	}
	for _, want := range []string{VerdictFits, VerdictGap, VerdictRejected} {
		if seen[want] == 0 {
			t.Errorf("fixture produces no %q case, so the verdict is not discriminated", want)
		}
	}
}

// TestKnownMutantsAreDetected pins the three mutants the issue names.
func TestKnownMutantsAreDetected(t *testing.T) {
	cases, expected := loadFixtures(t)
	mutants := map[string]Options{
		"multiplicity inferred from composition": {InferMultiplicityFromComposition: true},
		"core without counterexample":            {AllowCoreWithoutCounterexample: true},
		"unit read from the base type":           {TreatSameBaseAsSameUnit: true},
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

func TestCoreDestinationIsNotAvailable(t *testing.T) {
	c := Case{
		ID: "core",
		Structure: Structure{ID: "s", Family: "collection",
			Facets: Facets{Identity: "occurrence", Composition: "sequence",
				Order: "ordered", Multiplicity: "multi"}},
		Judgment: Judgment{Destination: "core-type", Evidence: []string{"the author asked"}},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictRejected {
		t.Fatalf("core-type must be rejected, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagUnknownDestination)
}

func TestMultiplicityIsNeverInferred(t *testing.T) {
	c := Case{
		ID: "inferred",
		Structure: Structure{ID: "s", Family: "collection",
			Facets: Facets{Identity: "occurrence", Composition: "sequence"}},
		Judgment: Judgment{Destination: DestFacet, Relations: []string{RelHasFacet},
			Evidence: []string{"a sequence is assumed ordered, unique and complete"}},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictRejected {
		t.Fatalf("an inferred multiplicity must be rejected, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagMultiplicityInference)
}

func TestBaseTypeNeverEstablishesAUnit(t *testing.T) {
	c := Case{
		ID:        "unit",
		Structure: Structure{ID: "s", Family: "quantity", BaseType: "std.decimal"},
		Judgment:  Judgment{Destination: DestPlugin, Evidence: []string{"both are std.decimal"}},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictGap {
		t.Fatalf("a missing unit must be a gap, got %q", got.Verdict)
	}
	assertGap(t, got, GapUnitFacet)
	assertHasDiagnostic(t, got, GapUnitFacet)
}

func TestCorpusRefMustBeUnique(t *testing.T) {
	raw := []byte(`{"schemaVersion":"tei.reference.facet-structure-input/v0","cases":[
		{"id":"a","structure":{"id":"a","family":"collection","corpusRef":"PC-01"}},
		{"id":"b","structure":{"id":"b","family":"map","corpusRef":"PC-01"}}]}`)
	if _, err := ParseCases(raw); err == nil {
		t.Fatal("a shared corpusRef must be rejected so no second corpus is created")
	}
}

func TestParseRejectsUnknownVocabulary(t *testing.T) {
	if _, err := ParseCases([]byte(`{"schemaVersion":"tei.reference.facet-structure-input/v9","cases":[]}`)); err == nil {
		t.Fatal("unknown schema version must be rejected")
	}
	if _, err := ParseCases([]byte(`{"schemaVersion":"tei.reference.facet-structure-input/v0","cases":[]}`)); err == nil {
		t.Fatal("an empty case list must be rejected")
	}
	if _, err := ParseExpected([]byte(`{"schemaVersion":"tei.reference.facet-structure-expected/v0","expectations":[{"id":"a","verdict":"ok"}]}`)); err == nil {
		t.Fatal("a verdict outside the fixed vocabulary must be rejected")
	}
}

func TestContractIsMachineReadable(t *testing.T) {
	desc := ContractDescription()
	if len(desc.Destinations) != 5 {
		t.Errorf("want 5 destinations, got %d", len(desc.Destinations))
	}
	if len(desc.RelationKinds) != 7 {
		t.Errorf("want the 7 relation roles from #2495, got %d", len(desc.RelationKinds))
	}
	if len(desc.Gaps) == 0 || len(desc.Rules) == 0 || len(desc.NotClaims) == 0 {
		t.Fatal("the contract must publish gaps, rules and non-claims")
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{{ID: "empty"}}}, Options{})
	if report.Contract == nil {
		t.Fatal("the report does not embed the contract")
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

func assertGap(t *testing.T, r Result, code string) {
	t.Helper()
	for _, g := range r.Gaps {
		if g == code {
			return
		}
	}
	t.Fatalf("result %s has no %s gap: %v", r.ID, code, r.Gaps)
}

package representationbindingcheck

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
	return filepath.Join(filepath.Dir(file), "..", "..", "plan", "fixtures", "representation-binding")
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

// TestHandWrittenExpectationsMatch is the S3 acceptance.
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

// TestEveryVerdictAppears keeps the fixture discriminating.
func TestEveryVerdictAppears(t *testing.T) {
	cases, _ := loadFixtures(t)
	report := Check(cases, Options{})
	seen := map[string]int{}
	for _, r := range report.Results {
		seen[r.Verdict]++
	}
	for _, want := range []string{VerdictBound, VerdictUnqualified, VerdictRejected} {
		if seen[want] == 0 {
			t.Errorf("fixture produces no %q case, so the verdict is not discriminated", want)
		}
	}
}

// TestKnownMutantsAreDetected pins the four mutants the issue names.
func TestKnownMutantsAreDetected(t *testing.T) {
	cases, expected := loadFixtures(t)
	mutants := map[string]Options{
		"contextless instance accepted":    {AcceptContextless: true},
		"topology adopted as meaning":      {TreatTopologyAsMeaning: true},
		"selection read as identity":       {TreatSelectionAsIdentity: true},
		"unknown clock accepted as synced": {AcceptUnknownClock: true},
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

func TestMeaningBearingInstanceNeedsAContext(t *testing.T) {
	c := Case{
		ID:             "no-context",
		Representation: Representation{ID: "r", Kind: RepTable, Topology: []string{"row", "column"}},
		Binding:        Binding{Scope: "s", Grain: "line", Operation: OpReadOnly, Evidence: []string{"rendered"}},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictRejected {
		t.Fatalf("a contextless meaning-bearing instance must be rejected, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagContextMissing)
}

func TestDomainTimeIsNeverPlaybackTime(t *testing.T) {
	c := Case{
		ID:             "axes",
		Representation: Representation{ID: "r", Kind: RepVideo, Topology: []string{"frame", "timestamp"}},
		Binding: Binding{
			ContextID: "review.media", Scope: "window", Grain: "frame",
			Axes:       []Axis{{Name: "playback", Role: AxisPresentation}},
			SourceAxes: []string{AxisDomain},
			Clock:      "device", SyncClaimed: true, SyncError: "0ms",
			Operation: OpReadOnly, Evidence: []string{"rendered"},
		},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictRejected {
		t.Fatalf("domain time reported as playback time must be rejected, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagAxisConflated)
}

func TestGeneratedAxisIsNotASourceAxis(t *testing.T) {
	c := Case{
		ID:             "generated",
		Representation: Representation{ID: "r", Kind: RepMarkdown, Topology: []string{"heading", "paragraph", "page"}},
		Binding: Binding{
			ContextID: "doc.render", Scope: "section", Grain: "paragraph",
			Axes:       []Axis{{Name: "page", Role: AxisStructural, Generated: true}},
			SourceAxes: []string{AxisStructural},
			Operation:  OpReadOnly, Evidence: []string{"page follows paragraph"},
		},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictBound {
		t.Fatalf("a declared generated axis must stay bound, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagGeneratedAxis)
}

func TestAggregateCannotQualifyAnInverse(t *testing.T) {
	c := Case{
		ID:             "aggregate",
		Representation: Representation{ID: "r", Kind: RepChart, Topology: []string{"axis", "point"}},
		Binding: Binding{
			ContextID: "finance.dashboard", Scope: "tenant", SourceGrain: "invoice", Grain: "tenant-month",
			Operation: OpInverseClaim, DeclaredLosses: []string{"grain", "fact-identity"},
			Evidence: []string{"one bar per month"},
		},
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Cases: []Case{c}}, Options{})
	got := report.Results[0]
	if got.Verdict != VerdictRejected {
		t.Fatalf("an aggregate over source rows must be rejected, got %q", got.Verdict)
	}
	assertHasDiagnostic(t, got, DiagAggregateNotEntity)
}

func TestParseRejectsUnknownVocabulary(t *testing.T) {
	if _, err := ParseCases([]byte(`{"schemaVersion":"tei.reference.representation-binding-input/v9","cases":[]}`)); err == nil {
		t.Fatal("unknown schema version must be rejected")
	}
	if _, err := ParseCases([]byte(`{"schemaVersion":"tei.reference.representation-binding-input/v0","cases":[]}`)); err == nil {
		t.Fatal("an empty case list must be rejected")
	}
	if _, err := ParseExpected([]byte(`{"schemaVersion":"tei.reference.representation-binding-expected/v0","expectations":[{"id":"a","verdict":"ok"}]}`)); err == nil {
		t.Fatal("a verdict outside the fixed vocabulary must be rejected")
	}
}

func TestContractIsMachineReadable(t *testing.T) {
	desc := ContractDescription()
	if len(desc.RepresentationTypes) != 15 {
		t.Errorf("want the 15 representation types, got %d", len(desc.RepresentationTypes))
	}
	if len(desc.TimeAxes) != 4 {
		t.Errorf("want the 4 time axes, got %d", len(desc.TimeAxes))
	}
	if len(desc.RequiredBindingColumns) != 6 {
		t.Errorf("want 6 required binding columns, got %d", len(desc.RequiredBindingColumns))
	}
	if len(desc.Rules) == 0 || len(desc.NotClaims) == 0 {
		t.Fatal("the contract must publish rules and non-claims")
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

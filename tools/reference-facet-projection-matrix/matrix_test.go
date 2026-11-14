package facetprojectionmatrix

import (
	"path/filepath"
	"runtime"
	"testing"

	facetmappingcheck "tei/tools/reference-facet-mapping-check"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "plan", "fixtures", "facet-projection-matrix")
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

// TestHandWrittenExpectationsMatch is the matrix acceptance.
func TestHandWrittenExpectationsMatch(t *testing.T) {
	cases, expected := loadFixtures(t)
	report := Check(cases)
	cmp := Compare(report, expected)
	if cmp.Status != "ok" {
		t.Fatalf("expectations do not match: %+v", cmp.Mismatches)
	}
	if cmp.Matched != cmp.Total || cmp.Total != len(cases.Cells) {
		t.Fatalf("matched %d of %d for %d cells", cmp.Matched, cmp.Total, len(cases.Cells))
	}
}

// TestMatrixNeverCollapsesVerdicts pins the rule the planner asked for: the
// matrix must keep the unsafe answers visible instead of reporting "possible".
func TestMatrixNeverCollapsesVerdicts(t *testing.T) {
	cases, _ := loadFixtures(t)
	report := Check(cases)
	seen := map[string]int{}
	for _, row := range report.Matrix.Structures {
		for _, cell := range row.Cells {
			seen[cell.Verdict]++
		}
	}
	for _, want := range []string{CellPreserved, CellLossy, CellUnqualified, CellRejected} {
		if seen[want] == 0 {
			t.Errorf("matrix produces no %q cell, so it cannot discriminate", want)
		}
	}
	for _, row := range report.Matrix.Structures {
		if row.Best == "" {
			t.Fatalf("structure %q has no best verdict", row.Structure)
		}
		// A row whose best verdict is rejected must not list any safe pair.
		if row.Best == CellRejected && len(row.Safe) > 0 {
			t.Errorf("structure %q is rejected but lists safe representations %v", row.Structure, row.Safe)
		}
	}
}

func TestUncoveredPairsAreReportedNotAssumed(t *testing.T) {
	c := CaseFile{
		SchemaVersion: InputSchema,
		Structures: []Structure{
			{ID: "s1", Label: "S1", Family: "collection"},
			{ID: "s2", Label: "S2", Family: "collection"},
		},
		Representations: []Representation{
			{ID: "r1", Label: "R1", Kind: "table", Topology: []string{"row"}},
		},
		Cells: []Cell{
			{StructureIndex: 0, RepresentationIndex: 0, ContextID: "ctx",
				Mapping: emptyMapping()},
		},
	}
	report := Check(c)
	if len(report.Matrix.Uncovered) != 1 {
		t.Fatalf("want one uncovered pair, got %v", report.Matrix.Uncovered)
	}
	if report.Matrix.Uncovered[0] != "S2 -> R1" {
		t.Fatalf("uncovered pair is %q", report.Matrix.Uncovered[0])
	}
	assertHasDiagnostic(t, report, DiagPairUncovered)
}

func TestCellWithoutContextIsRejected(t *testing.T) {
	c := CaseFile{
		SchemaVersion: InputSchema,
		Structures:    []Structure{{ID: "s", Label: "S", Family: "collection"}},
		Representations: []Representation{
			{ID: "r", Label: "R", Kind: "table", Topology: []string{"row"}},
		},
		Cells: []Cell{{StructureIndex: 0, RepresentationIndex: 0, Mapping: emptyMapping()}},
	}
	report := Check(c)
	assertHasDiagnostic(t, report, DiagContextMissing)
}

func TestDuplicateCellsAreRejected(t *testing.T) {
	raw := []byte(`{"schemaVersion":"tei.reference.facet-projection-matrix-input/v0",
		"structures":[{"id":"s","label":"S","family":"collection"}],
		"representations":[{"id":"r","label":"R","kind":"table"}],
		"cells":[{"structure":0,"representation":0,"contextId":"c"},
		         {"structure":0,"representation":0,"contextId":"c"}]}`)
	if _, err := ParseCases(raw); err == nil {
		t.Fatal("a duplicated cell must be rejected")
	}
}

func TestParseRejectsOutOfRangeIndex(t *testing.T) {
	raw := []byte(`{"schemaVersion":"tei.reference.facet-projection-matrix-input/v0",
		"structures":[{"id":"s","label":"S","family":"collection"}],
		"representations":[{"id":"r","label":"R","kind":"table"}],
		"cells":[{"structure":3,"representation":0,"contextId":"c"}]}`)
	if _, err := ParseCases(raw); err == nil {
		t.Fatal("an out-of-range index must be rejected")
	}
}

func TestMutantsChangeTheMatrix(t *testing.T) {
	cases, expected := loadFixtures(t)
	mutants := map[string]MatrixOptions{
		"order ignored":             {IgnoreOrder: true},
		"presence binarized":        {BinarizePresence: true},
		"aggregate inverse allowed": {AllowAggregateInverse: true},
		"quantity scale ignored":    {IgnoreQuantityScale: true},
	}
	for name, opts := range mutants {
		opts := opts
		t.Run(name, func(t *testing.T) {
			withMutant := cases
			withMutant.Options = opts
			report := Check(withMutant)
			cmp := Compare(report, expected)
			if cmp.Status == "ok" {
				t.Fatalf("mutant %q is not detected by the matrix", name)
			}
		})
	}
}

func TestContractSharesTheMappingContract(t *testing.T) {
	desc := ContractDescription()
	if len(desc.Columns) != 12 {
		t.Errorf("the matrix must reuse the 12 mapping columns, got %d", len(desc.Columns))
	}
	if len(desc.Verdicts) != 5 {
		t.Errorf("want the 5 #4351 verdicts, got %d", len(desc.Verdicts))
	}
	if desc.SharedContract == "" || len(desc.Rules) == 0 || len(desc.NotClaims) == 0 {
		t.Fatal("the contract must state the shared contract, rules and non-claims")
	}
	report := Check(CaseFile{SchemaVersion: InputSchema, Structures: []Structure{{ID: "s", Label: "S"}},
		Representations: []Representation{{ID: "r", Label: "R", Kind: "table"}},
		Cells:           []Cell{{StructureIndex: 0, RepresentationIndex: 0, ContextID: "c", Mapping: emptyMapping()}}})
	if report.Contract == nil {
		t.Fatal("the report does not embed the contract")
	}
}

// emptyMapping returns a mapping with an identity claim and no loss, so a cell
// can be built without repeating the twelve columns in every test.
func emptyMapping() facetmappingcheck.Mapping {
	return facetmappingcheck.Mapping{
		Identity: facetmappingcheck.IdentityColumn{Kind: "1:1", ClaimedPreserved: true, Evidence: []string{"id carried"}},
		Coverage: facetmappingcheck.CoverageColumn{Claimed: "complete", Observed: "complete"},
	}
}

func assertHasDiagnostic(t *testing.T, out Output, code string) {
	t.Helper()
	for _, d := range out.Diagnostics {
		if d.Code == code {
			return
		}
	}
	t.Fatalf("report has no %s diagnostic: %+v", code, out.Diagnostics)
}

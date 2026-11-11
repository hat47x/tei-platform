// Package facetprojectionmatrix is the read-only reference for the matrix stage
// of #4350. It composes the structure side (#4352) and the representation side
// (#4353) by asking, for every (structure, representation) pair, what the
// projection preserves and what it loses.
//
// The package does not invent a projection engine: it calls the #4351 checker on
// each cell and aggregates the verdicts into a matrix. One Contract is published
// so a generating model can read the whole projection space without prose.
package facetprojectionmatrix

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	facetmappingcheck "tei/tools/reference-facet-mapping-check"
)

const (
	InputSchema    = "tei.reference.facet-projection-matrix-input/v0"
	ExpectedSchema = "tei.reference.facet-projection-matrix-expected/v0"
	ReportSchema   = "tei.reference.facet-projection-matrix/v0"
)

// CellVerdicts re-use the #4351 vocabulary. A cell never invents a verdict of
// its own; the matrix only orders the answers.
const (
	CellPreserved   = facetmappingcheck.VerdictPreserved
	CellLossy       = facetmappingcheck.VerdictLossy
	CellRejected    = facetmappingcheck.VerdictRejected
	CellUnqualified = facetmappingcheck.VerdictUnqualified
	CellUnknown     = facetmappingcheck.VerdictUnknown
)

// Rank orders the verdicts from safest to least usable. A lower rank is a
// stronger answer; a matrix that only shows "possible" collapses these into one
// and is exactly what this reference refuses to do.
var verdictRank = map[string]int{
	CellPreserved:   0,
	CellLossy:       1,
	CellUnqualified: 2,
	CellUnknown:     3,
	CellRejected:    4,
}

// Structure is one data structure the matrix projects from. The facets are the
// same orthogonal description #4351 compares on the data side.
type Structure struct {
	ID     string                   `json:"id"`
	Label  string                   `json:"label"`
	Family string                   `json:"family"`
	Facets facetmappingcheck.Facets `json:"facets"`
}

// Representation is one representation type the matrix projects to. Facets are
// the observations the representation makes about the meaning it can carry; they
// are the same shape #4351 compares on the representation side.
type Representation struct {
	ID        string                   `json:"id"`
	Label     string                   `json:"label"`
	Kind      string                   `json:"kind"`
	Topology  []string                 `json:"topology,omitempty"`
	Axes      []string                 `json:"axes,omitempty"`
	Selection string                   `json:"selection,omitempty"`
	Facets    facetmappingcheck.Facets `json:"facets"`
}

// Cell is one projection under test. Mapping is the #4351 contract verbatim, so
// the matrix neither restates nor relaxes the twelve columns.
type Cell struct {
	StructureIndex      int                          `json:"structure"`
	RepresentationIndex int                          `json:"representation"`
	Mapping             facetmappingcheck.Mapping    `json:"mapping"`
	Composite           *facetmappingcheck.Composite `json:"composite,omitempty"`
	// ContextID is mandatory: a cell without a Function Data Context is not a
	// meaning-bearing projection.
	ContextID string `json:"contextId"`
	Epoch     string `json:"epoch,omitempty"`
}

type CaseFile struct {
	SchemaVersion   string           `json:"schemaVersion"`
	Structures      []Structure      `json:"structures"`
	Representations []Representation `json:"representations"`
	Cells           []Cell           `json:"cells"`
	Options         MatrixOptions    `json:"options,omitempty"`
}

// MatrixOptions carries the known mutants so one input can demonstrate that the
// matrix discriminates them.
type MatrixOptions struct {
	IgnoreOrder           bool `json:"ignoreOrder,omitempty"`
	BinarizePresence      bool `json:"binarizePresence,omitempty"`
	AllowAggregateInverse bool `json:"allowAggregateInverse,omitempty"`
	IgnoreQuantityScale   bool `json:"ignoreQuantityScale,omitempty"`
	AcceptContextless     bool `json:"acceptContextless,omitempty"`
}

func (o MatrixOptions) toChecker() facetmappingcheck.Options {
	return facetmappingcheck.Options{
		IgnoreOrder:           o.IgnoreOrder,
		BinarizePresence:      o.BinarizePresence,
		AllowAggregateInverse: o.AllowAggregateInverse,
		IgnoreQuantityScale:   o.IgnoreQuantityScale,
	}
}

// CellResult is one answered cell.
type CellResult struct {
	Structure      string   `json:"structure"`
	Representation string   `json:"representation"`
	Verdict        string   `json:"verdict"`
	Losses         []string `json:"losses"`
	Inverse        string   `json:"inverse"`
	// Entry is the synthetic case id the #4351 checker used, so one result can be
	// traced back to that report.
	Entry string `json:"entry"`
}

// StructureRow summarizes one structure across every representation.
type StructureRow struct {
	Structure string       `json:"structure"`
	Cells     []CellResult `json:"cells"`
	// Best is the strongest verdict any representation reached.
	Best string `json:"best"`
	// Safe lists the representations that preserve the projection.
	Safe []string `json:"safe"`
	// Lossy lists the representations that carry it with a declared loss.
	Lossy []string `json:"lossy"`
	// Refused lists the representations that cannot carry it at all.
	Refused []string `json:"refused"`
}

// RepresentationColumn summarizes one representation across every structure.
type RepresentationColumn struct {
	Representation string   `json:"representation"`
	Carries        []string `json:"carries"`
	Loses          []string `json:"loses"`
	Refuses        []string `json:"refuses"`
}

type Matrix struct {
	Structures      []StructureRow         `json:"structures"`
	Representations []RepresentationColumn `json:"representations"`
	// Uncovered lists the pairs the input leaves out. A gap in the matrix is
	// reported, never filled by an assumption.
	Uncovered []string `json:"uncovered"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Detail   string `json:"detail,omitempty"`
}

const (
	DiagContextMissing  = "CELL_CONTEXT_MISSING"
	DiagCellDuplicate   = "CELL_DUPLICATE"
	DiagIndexOutOfRange = "CELL_INDEX_OUT_OF_RANGE"
	DiagPairUncovered   = "MATRIX_PAIR_UNCOVERED"
	DiagUnsafeAsSafe    = "UNSAFE_REPORTED_AS_SAFE"
	DiagLossNotCarried  = "LOSS_NOT_REPORTED"
)

const (
	SeverityReject = "reject"
	SeverityGap    = "gap"
	SeverityInfo   = "info"
)

// Contract is the machine-readable self-description.
type Contract struct {
	SchemaVersion  string   `json:"schemaVersion"`
	Verdicts       []string `json:"verdicts"`
	Columns        []string `json:"columns"`
	Losses         []string `json:"losses"`
	Inverses       []string `json:"inverses"`
	Uncovered      string   `json:"uncoveredHandling"`
	SharedContract string   `json:"sharedContract"`
	Rules          []string `json:"rules"`
	NotClaims      []string `json:"notClaims"`
}

type Output struct {
	SchemaVersion string       `json:"schemaVersion"`
	Checked       int          `json:"checked"`
	Contract      *Contract    `json:"contract,omitempty"`
	Matrix        Matrix       `json:"matrix"`
	Diagnostics   []Diagnostic `json:"diagnostics"`
	Comparison    *Comparison  `json:"comparison,omitempty"`
}

type Expectation struct {
	Structure      string   `json:"structure"`
	Representation string   `json:"representation"`
	Verdict        string   `json:"verdict"`
	Losses         []string `json:"losses"`
	Note           string   `json:"note,omitempty"`
}

type ExpectedFile struct {
	SchemaVersion string        `json:"schemaVersion"`
	Expectations  []Expectation `json:"expectations"`
}

type Mismatch struct {
	Pair     string `json:"pair"`
	Field    string `json:"field"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
}

type Comparison struct {
	SchemaVersion string     `json:"schemaVersion"`
	Status        string     `json:"status"`
	Matched       int        `json:"matched"`
	Total         int        `json:"total"`
	Mismatches    []Mismatch `json:"mismatches,omitempty"`
}

func ContractDescription() Contract {
	return Contract{
		SchemaVersion:  ReportSchema,
		Verdicts:       []string{CellPreserved, CellLossy, CellUnqualified, CellUnknown, CellRejected},
		Columns:        facetmappingcheck.ContractDescription().Columns,
		Losses:         facetmappingcheck.ContractDescription().Losses,
		Inverses:       facetmappingcheck.ContractDescription().Inverses,
		Uncovered:      "a pair the input leaves out is listed as uncovered; it is never assumed to be possible",
		SharedContract: "each cell is evaluated by the #4351 twelve-column mapping contract; this reference adds no column",
		Rules: []string{
			"the matrix orders verdicts; it never merges them into one 'possible' answer",
			"a cell is preserved only when the #4351 checker says preserved for that cell",
			"a representation that refuses one structure is not removed from the others",
			"an uncovered pair is reported, never filled",
		},
		NotClaims: []string{
			"no projection engine, registry, or renderer is created; the #4351 checker does the evaluation",
			"no Core type or public mapping schema is adopted",
			"no business meaning is accepted; the matrix is a comparison result on synthetic descriptors",
		},
	}
}

func ParseCases(raw []byte) (CaseFile, error) {
	var f CaseFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return CaseFile{}, fmt.Errorf("parse cases: %w", err)
	}
	if f.SchemaVersion != InputSchema {
		return CaseFile{}, fmt.Errorf("cases schemaVersion is %q, want %q", f.SchemaVersion, InputSchema)
	}
	if len(f.Structures) == 0 || len(f.Representations) == 0 || len(f.Cells) == 0 {
		return CaseFile{}, fmt.Errorf("structures, representations and cells are all required")
	}
	seen := map[string]bool{}
	for _, c := range f.Cells {
		key := fmt.Sprintf("%d/%d", c.StructureIndex, c.RepresentationIndex)
		if seen[key] {
			return CaseFile{}, fmt.Errorf("cell %s is declared twice", key)
		}
		seen[key] = true
		if c.StructureIndex < 0 || c.StructureIndex >= len(f.Structures) {
			return CaseFile{}, fmt.Errorf("cell %s: structure index out of range", key)
		}
		if c.RepresentationIndex < 0 || c.RepresentationIndex >= len(f.Representations) {
			return CaseFile{}, fmt.Errorf("cell %s: representation index out of range", key)
		}
	}
	return f, nil
}

func ParseExpected(raw []byte) (ExpectedFile, error) {
	var f ExpectedFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return ExpectedFile{}, fmt.Errorf("parse expected: %w", err)
	}
	if f.SchemaVersion != ExpectedSchema {
		return ExpectedFile{}, fmt.Errorf("expected schemaVersion is %q, want %q", f.SchemaVersion, ExpectedSchema)
	}
	seen := map[string]bool{}
	for _, e := range f.Expectations {
		pair := e.Structure + "/" + e.Representation
		if e.Structure == "" || e.Representation == "" {
			return ExpectedFile{}, fmt.Errorf("expectation without both names")
		}
		if seen[pair] {
			return ExpectedFile{}, fmt.Errorf("duplicate expectation %q", pair)
		}
		seen[pair] = true
		if _, ok := verdictRank[e.Verdict]; !ok {
			return ExpectedFile{}, fmt.Errorf("expectation %q has unsupported verdict %q", pair, e.Verdict)
		}
	}
	return f, nil
}

func LoadCases(path string) (CaseFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return CaseFile{}, err
	}
	return ParseCases(raw)
}

func LoadExpected(path string) (ExpectedFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ExpectedFile{}, err
	}
	return ParseExpected(raw)
}

// Check evaluates every cell through the #4351 checker and builds the matrix.
func Check(file CaseFile) Output {
	desc := ContractDescription()
	out := Output{SchemaVersion: ReportSchema, Contract: &desc}

	checkerOptions := file.Options.toChecker()
	checkerCases := make([]facetmappingcheck.Case, 0, len(file.Cells))
	cellIndex := make([]Cell, 0, len(file.Cells))
	for i, c := range file.Cells {
		entry := fmt.Sprintf("cell-%02d", i+1)
		checkerCases = append(checkerCases, facetmappingcheck.Case{
			ID:             entry,
			Title:          file.Structures[c.StructureIndex].Label + " -> " + file.Representations[c.RepresentationIndex].Label,
			Structure:      structToChecker(file.Structures[c.StructureIndex]),
			Representation: structRepresentation(file.Representations[c.RepresentationIndex], c),
			Mapping:        c.Mapping,
			Composite:      c.Composite,
		})
		cellIndex = append(cellIndex, c)
	}

	report := facetmappingcheck.Check(
		facetmappingcheck.CaseFile{SchemaVersion: facetmappingcheck.InputSchema, Cases: checkerCases},
		checkerOptions,
	)

	byEntry := map[string]facetmappingcheck.Result{}
	for _, r := range report.Results {
		byEntry[r.ID] = r
	}

	structureRows := make([]StructureRow, len(file.Structures))
	for i, s := range file.Structures {
		structureRows[i] = StructureRow{Structure: s.Label, Cells: []CellResult{}}
	}
	repColumns := make([]RepresentationColumn, len(file.Representations))
	for i, r := range file.Representations {
		repColumns[i] = RepresentationColumn{Representation: r.Label}
	}

	for i, c := range cellIndex {
		entry := fmt.Sprintf("cell-%02d", i+1)
		res := byEntry[entry]
		cell := CellResult{
			Structure:      file.Structures[c.StructureIndex].Label,
			Representation: file.Representations[c.RepresentationIndex].Label,
			Verdict:        res.Verdict,
			Losses:         res.Losses,
			Inverse:        res.Inverse,
			Entry:          entry,
		}
		row := &structureRows[c.StructureIndex]
		row.Cells = append(row.Cells, cell)
		col := &repColumns[c.RepresentationIndex]
		switch res.Verdict {
		case CellPreserved:
			col.Carries = append(col.Carries, file.Structures[c.StructureIndex].Label)
		case CellLossy, CellUnqualified, CellUnknown:
			col.Loses = append(col.Loses, file.Structures[c.StructureIndex].Label)
		case CellRejected:
			col.Refuses = append(col.Refuses, file.Structures[c.StructureIndex].Label)
		}
	}

	for i := range structureRows {
		structureRows[i].Best = bestVerdict(structureRows[i].Cells)
		for _, cell := range structureRows[i].Cells {
			switch cell.Verdict {
			case CellPreserved:
				structureRows[i].Safe = append(structureRows[i].Safe, cell.Representation)
			case CellRejected:
				structureRows[i].Refused = append(structureRows[i].Refused, cell.Representation)
			default:
				structureRows[i].Lossy = append(structureRows[i].Lossy, cell.Representation)
			}
		}
		structureRows[i].Safe = sortedCopy(structureRows[i].Safe)
		structureRows[i].Lossy = sortedCopy(structureRows[i].Lossy)
		structureRows[i].Refused = sortedCopy(structureRows[i].Refused)
	}
	for i := range repColumns {
		repColumns[i].Carries = sortedCopy(repColumns[i].Carries)
		repColumns[i].Loses = sortedCopy(repColumns[i].Loses)
		repColumns[i].Refuses = sortedCopy(repColumns[i].Refuses)
	}

	// Uncovered pairs are reported, never assumed.
	var uncovered []string
	for i := range file.Structures {
		for j := range file.Representations {
			key := fmt.Sprintf("%d/%d", i, j)
			covered := false
			for _, c := range file.Cells {
				if fmt.Sprintf("%d/%d", c.StructureIndex, c.RepresentationIndex) == key {
					covered = true
					break
				}
			}
			if !covered {
				uncovered = append(uncovered, file.Structures[i].Label+" -> "+file.Representations[j].Label)
			}
		}
	}

	out.Checked = len(file.Cells)
	out.Matrix = Matrix{
		Structures:      structureRows,
		Representations: repColumns,
		Uncovered:       sortedCopy(uncovered),
	}

	var diagnostics []Diagnostic
	for _, c := range file.Cells {
		if strings.TrimSpace(c.ContextID) == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Code: DiagContextMissing, Severity: SeverityReject,
				Detail: "a cell without a Function Data Context is not a meaning-bearing projection",
			})
		}
	}
	if len(uncovered) > 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Code: DiagPairUncovered, Severity: SeverityGap,
			Detail: fmt.Sprintf("%d pair(s) are not covered by any cell", len(uncovered)),
		})
	}
	for _, row := range structureRows {
		if len(row.Safe) > 0 && len(row.Cells) > 0 {
			// A structure with no cell at all must not read as safe.
			for _, cell := range row.Cells {
				if cell.Verdict == CellRejected && contains(row.Safe, cell.Representation) {
					diagnostics = append(diagnostics, Diagnostic{
						Code: DiagUnsafeAsSafe, Severity: SeverityReject,
						Detail: row.Structure + " lists a refused representation as safe",
					})
				}
			}
		}
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Detail < diagnostics[j].Detail
	})
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	out.Diagnostics = diagnostics
	return out
}

func structToChecker(s Structure) facetmappingcheck.Structure {
	return facetmappingcheck.Structure{
		ID:     s.ID,
		Family: s.Family,
		Facets: s.Facets,
	}
}

func structRepresentation(r Representation, c Cell) facetmappingcheck.Representation {
	return facetmappingcheck.Representation{
		ID:        r.ID,
		Kind:      r.Kind,
		Topology:  r.Topology,
		Axes:      r.Axes,
		Selection: r.Selection,
		ContextID: c.ContextID,
		Epoch:     c.Epoch,
		Facets:    r.Facets,
	}
}

func bestVerdict(cells []CellResult) string {
	if len(cells) == 0 {
		return CellUnknown
	}
	best := CellRejected
	for _, c := range cells {
		if verdictRank[c.Verdict] < verdictRank[best] {
			best = c.Verdict
		}
	}
	return best
}

// Compare matches each matrix cell against the hand-written expectation.
func Compare(out Output, expected ExpectedFile) Comparison {
	cmp := Comparison{SchemaVersion: ReportSchema, Total: len(expected.Expectations)}
	cells := map[string]CellResult{}
	for _, row := range out.Matrix.Structures {
		for _, c := range row.Cells {
			cells[row.Structure+"/"+c.Representation] = c
		}
	}
	for _, want := range expected.Expectations {
		key := want.Structure + "/" + want.Representation
		got, ok := cells[key]
		if !ok {
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{Pair: key, Field: "missing", Expected: "covered", Observed: "uncovered"})
			continue
		}
		matched := true
		if got.Verdict != want.Verdict {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{Pair: key, Field: "verdict", Expected: want.Verdict, Observed: got.Verdict})
		}
		if !equalSet(got.Losses, sortedCopy(want.Losses)) {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{Pair: key, Field: "losses", Expected: joinList(want.Losses), Observed: joinList(got.Losses)})
		}
		if matched {
			cmp.Matched++
		}
	}
	if len(cmp.Mismatches) == 0 {
		cmp.Status = "ok"
	} else {
		cmp.Status = "mismatch"
	}
	return cmp
}

func contains(in []string, want string) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}

func sortedCopy(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if strings.TrimSpace(s) == "" {
			continue
		}
		out = append(out, s)
	}
	sort.Strings(out)
	unique := out[:0]
	for i, s := range out {
		if i > 0 && s == out[i-1] {
			continue
		}
		unique = append(unique, s)
	}
	return unique
}

func equalSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinList(in []string) string {
	if len(in) == 0 {
		return "(none)"
	}
	return strings.Join(in, ",")
}

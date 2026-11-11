// Package facetstructurecheck is the read-only reference for #4352, the S2 unit
// of #4350. It judges, for each standard data structure, whether the orthogonal
// Facet description can carry it, whether the current Source form cannot, and
// which destination should own the extension.
//
// The package deliberately does not adopt a Core type, does not extend the
// product syntax, and does not create a second Pressure Corpus. It reuses the
// facet vocabulary of #4351 and reports the gap.
package facetstructurecheck

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	InputSchema    = "tei.reference.facet-structure-input/v0"
	ExpectedSchema = "tei.reference.facet-structure-expected/v0"
	ReportSchema   = "tei.reference.facet-structure-check/v0"
)

// Destinations. A structure is carried by exactly one of these. The list is
// closed: an unlisted destination is a malformed descriptor, not a new option.
const (
	DestBasic          = "basic-type"
	DestComposite      = "composite-type"
	DestFacet          = "semantic-facet"
	DestPlugin         = "standard-plugin"
	DestRepresentation = "representation-type"
	DestUnknown        = "unknown"
)

// Verdicts. `fits` means the Facet description carries the structure without a
// new Core type; `gap` means it does not and the gap is recorded; `rejected`
// means the declaration itself is malformed; `unknown` means the comparison
// could not be made.
const (
	VerdictFits     = "fits"
	VerdictGap      = "gap"
	VerdictRejected = "rejected"
	VerdictUnknown  = "unknown"
)

// Relation kinds. The list mirrors #2495's seven roles; they are not synonyms.
const (
	RelIsA         = "is-a"
	RelRefines     = "refines"
	RelHasFacet    = "has-facet"
	RelComposedOf  = "composed-of"
	RelRealizedBy  = "realized-by"
	RelProjectedAs = "projected-as"
	RelConvertedTo = "converted-to"
)

// Gap codes name a meaning that the current Source form cannot express.
const (
	GapUnitFacet         = "unit-facet-absent"
	GapScaleFacet        = "scale-facet-absent"
	GapNamespaceFacet    = "namespace-facet-absent"
	GapCRSFacet          = "crs-facet-absent"
	GapDimensionFacet    = "dimension-facet-absent"
	GapTimeAxisFacet     = "time-axis-facet-absent"
	GapKeyEqualityFacet  = "key-equality-facet-absent"
	GapOccurrenceFacet   = "occurrence-facet-absent"
	GapCompletenessFacet = "completeness-facet-absent"
	GapEdgeIdentityFacet = "edge-identity-facet-absent"
	GapRecursionFacet    = "recursion-facet-absent"
	GapEffectFacet       = "effect-facet-absent"
	GapStateMachineFacet = "state-machine-facet-absent"
	GapArityFacet        = "arity-facet-absent"
	GapUnnamedPosition   = "positional-composition-absent"
)

// Diagnostic codes explain a judgment. They never grant a destination.
const (
	DiagThreefoldClaim        = "THREE_WAY_CLAIM" // facet + plugin + core claimed at once
	DiagCoreWithoutCounter    = "CORE_WITHOUT_COUNTEREXAMPLE"
	DiagMultiplicityInference = "MULTIPLICITY_INFERRED"
	DiagUnitInference         = "UNIT_INFERRED_FROM_BASE"
	DiagRelationKindUnknown   = "RELATION_KIND_UNSUPPORTED"
	DiagRelationMisuse        = "RELATION_KIND_MISUSED"
	DiagCorpusRefDuplicate    = "CORPUS_REF_DUPLICATED"
	DiagDestinationMissing    = "DESTINATION_MISSING"
	DiagGapUndeclared         = "GAP_UNDECLARED"
	DiagEvidenceMissing       = "EVIDENCE_MISSING"
	DiagUnknownDestination    = "DESTINATION_UNKNOWN"
)

// Facets is the same orthogonal description #4351 compares on the data side.
// Declaring a facet and inferring one from a base type are different acts.
type Facets struct {
	Identity     string   `json:"identity,omitempty"`
	Composition  string   `json:"composition,omitempty"`
	Order        string   `json:"order,omitempty"`
	OrderOrigin  string   `json:"orderOrigin,omitempty"`
	Multiplicity string   `json:"multiplicity,omitempty"`
	Key          string   `json:"key,omitempty"`
	Grain        string   `json:"grain,omitempty"`
	Presence     []string `json:"presence,omitempty"`
	Precision    string   `json:"precision,omitempty"`
	Unit         string   `json:"unit,omitempty"`
	Scale        string   `json:"scale,omitempty"`
	Dimension    string   `json:"dimension,omitempty"`
	CRS          string   `json:"crs,omitempty"`
	TimeAxes     []string `json:"timeAxes,omitempty"`
	Timebase     string   `json:"timebase,omitempty"`
	Clock        string   `json:"clock,omitempty"`
	Arity        string   `json:"arity,omitempty"`
	StateSpace   string   `json:"stateSpace,omitempty"`
	Purity       string   `json:"purity,omitempty"`
}

// Structure is one family under judgment.
type Structure struct {
	ID        string   `json:"id"`
	Family    string   `json:"family"`
	BaseType  string   `json:"baseType,omitempty"`
	Relations []string `json:"relations,omitempty"`
	CorpusRef string   `json:"corpusRef,omitempty"`
	Facets    Facets   `json:"facets"`
	// Unsupported lists the meaning this family needs but the current Source
	// form has no place for. It is an observation, not a request.
	Unsupported []string `json:"unsupported,omitempty"`
}

// Judgment is the contract under test: which destination should carry the
// structure, on what relation kinds, and with what evidence.
type Judgment struct {
	Destination string   `json:"destination"`
	Relations   []string `json:"relations,omitempty"`
	// CoreCounterexample names the concrete case that the Facet description
	// cannot carry. Without it, a core destination is not allowed.
	CoreCounterexample string   `json:"coreCounterexample,omitempty"`
	Evidence           []string `json:"evidence,omitempty"`
}

type Case struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Families  []string  `json:"families"`
	Structure Structure `json:"structure"`
	Judgment  Judgment  `json:"judgment"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Detail   string `json:"detail,omitempty"`
}

const (
	SeverityReject  = "reject"
	SeverityGap     = "gap"
	SeverityUnknown = "unknown"
	SeverityInfo    = "info"
)

type Result struct {
	ID          string       `json:"id"`
	Title       string       `json:"title,omitempty"`
	Verdict     string       `json:"verdict"`
	Destination string       `json:"destination"`
	Gaps        []string     `json:"gaps"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Evidence    []string     `json:"evidence"`
}

type Expectation struct {
	ID          string   `json:"id"`
	Verdict     string   `json:"verdict"`
	Destination string   `json:"destination"`
	Gaps        []string `json:"gaps"`
	Note        string   `json:"note,omitempty"`
}

type CaseFile struct {
	SchemaVersion string `json:"schemaVersion"`
	Cases         []Case `json:"cases"`
}

type ExpectedFile struct {
	SchemaVersion string        `json:"schemaVersion"`
	Expectations  []Expectation `json:"expectations"`
}

type Mismatch struct {
	ID       string `json:"id"`
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

// Contract is the machine-readable self-description, matching the style of
// #4351 so a consumer reads one shape from both references.
type Contract struct {
	SchemaVersion string   `json:"schemaVersion"`
	Destinations  []string `json:"destinations"`
	Verdicts      []string `json:"verdicts"`
	RelationKinds []string `json:"relationKinds"`
	Gaps          []string `json:"gaps"`
	Rules         []string `json:"rules"`
	NotClaims     []string `json:"notClaims"`
}

type Output struct {
	SchemaVersion string      `json:"schemaVersion"`
	Checked       int         `json:"checked"`
	Contract      *Contract   `json:"contract,omitempty"`
	Results       []Result    `json:"results"`
	Comparison    *Comparison `json:"comparison,omitempty"`
}

type Options struct {
	// InferMultiplicityFromComposition mimics the mutant that reads `many`
	// from a sequence and calls ordered/unique/complete settled.
	InferMultiplicityFromComposition bool
	// AllowCoreWithoutCounterexample mimics the mutant that adds a Core type
	// with no concrete case the Facet description cannot carry.
	AllowCoreWithoutCounterexample bool
	// TreatSameBaseAsSameUnit mimics the mutant that reads a unit from the
	// base type.
	TreatSameBaseAsSameUnit bool
}

func ParseCases(raw []byte) (CaseFile, error) {
	var f CaseFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return CaseFile{}, fmt.Errorf("parse cases: %w", err)
	}
	if f.SchemaVersion != InputSchema {
		return CaseFile{}, fmt.Errorf("cases schemaVersion is %q, want %q", f.SchemaVersion, InputSchema)
	}
	if len(f.Cases) == 0 {
		return CaseFile{}, fmt.Errorf("cases file has no cases")
	}
	seen := map[string]bool{}
	corpus := map[string]string{}
	for _, c := range f.Cases {
		if c.ID == "" {
			return CaseFile{}, fmt.Errorf("case without id")
		}
		if seen[c.ID] {
			return CaseFile{}, fmt.Errorf("duplicate case id %q", c.ID)
		}
		seen[c.ID] = true
		if c.Structure.CorpusRef != "" {
			if owner, dup := corpus[c.Structure.CorpusRef]; dup {
				return CaseFile{}, fmt.Errorf("corpusRef %q is used by %q and %q", c.Structure.CorpusRef, owner, c.ID)
			}
			corpus[c.Structure.CorpusRef] = c.ID
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
		if e.ID == "" {
			return ExpectedFile{}, fmt.Errorf("expectation without id")
		}
		if seen[e.ID] {
			return ExpectedFile{}, fmt.Errorf("duplicate expectation id %q", e.ID)
		}
		seen[e.ID] = true
		if !validVerdict(e.Verdict) {
			return ExpectedFile{}, fmt.Errorf("expectation %q has unsupported verdict %q", e.ID, e.Verdict)
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

func validVerdict(v string) bool {
	switch v {
	case VerdictFits, VerdictGap, VerdictRejected, VerdictUnknown:
		return true
	}
	return false
}

func ContractDescription() Contract {
	return Contract{
		SchemaVersion: ReportSchema,
		Destinations:  []string{DestBasic, DestComposite, DestFacet, DestPlugin, DestRepresentation},
		Verdicts:      []string{VerdictFits, VerdictGap, VerdictRejected, VerdictUnknown},
		RelationKinds: []string{RelIsA, RelRefines, RelHasFacet, RelComposedOf, RelRealizedBy, RelProjectedAs, RelConvertedTo},
		Gaps: []string{GapUnitFacet, GapScaleFacet, GapNamespaceFacet, GapCRSFacet, GapDimensionFacet,
			GapTimeAxisFacet, GapKeyEqualityFacet, GapOccurrenceFacet, GapCompletenessFacet,
			GapEdgeIdentityFacet, GapRecursionFacet, GapEffectFacet, GapStateMachineFacet,
			GapArityFacet, GapUnnamedPosition},
		Rules: []string{
			"fits: the Facet description carries the family and no Core type is required",
			"gap: the family needs a meaning the current Source cannot express; the gap is listed",
			"unknown: the destination or the evidence is missing, so no judgment is made",
			"a core destination requires a concrete counterexample the Facet description cannot carry",
			"multiplicity, order, uniqueness and completeness are declared, never inferred from composition",
			"a unit is declared; sharing a base type never establishes a shared unit",
			"one corpusRef belongs to exactly one case; a second Pressure Corpus is not created",
		},
		NotClaims: []string{
			"no Core type, type syntax, or semantic schema is adopted",
			"no second Pressure Corpus, scenario, or expected source is created",
			"no provider, database, renderer, or host is exercised",
			"fits is a comparison result on synthetic descriptors, not business acceptance",
		},
	}
}

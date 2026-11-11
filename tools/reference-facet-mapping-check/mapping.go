// Package facetmappingcheck is the read-only reference for #4351, the S1 unit
// of #4350. It checks a typed mapping contract between a standard data
// structure and a standard representation type, and reports what the
// transformation preserves, what it loses, and what stays unknown.
//
// The package deliberately does not adopt a Core type, does not render, does
// not read a Provider, and never turns a read-only projection into an edit
// inverse. It only compares observations (the Facets declared on the structure
// and on the representation) with the claims and declarations in the mapping
// contract, and it reports the disagreement.
package facetmappingcheck

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	InputSchema    = "tei.reference.facet-mapping-input/v0"
	ExpectedSchema = "tei.reference.facet-mapping-expected/v0"
	ReportSchema   = "tei.reference.facet-mapping-check/v0"
)

// Verdicts. The vocabulary is fixed: a mapping is either preserved, lossy,
// rejected, unqualified, or unknown. Nothing else is reported as a verdict.
const (
	VerdictPreserved   = "preserved"
	VerdictLossy       = "lossy"
	VerdictRejected    = "rejected"
	VerdictUnqualified = "unqualified"
	VerdictUnknown     = "unknown"
)

// Inverse qualification. A successful read-only projection never implies that
// the edit inverse is available, so this is reported separately from the
// verdict.
const (
	InverseEligible   = "eligible"
	InverseIneligible = "ineligible"
	InverseNotClaimed = "not-claimed"
)

// Loss codes name the meaning that the transformation drops.
const (
	LossOrder              = "order"
	LossOccurrenceIdentity = "occurrence-identity"
	LossEdgeMultiplicity   = "edge-multiplicity"
	LossParentArity        = "parent-arity"
	LossHiddenEdge         = "hidden-edge"
	LossGrain              = "grain"
	LossFactIdentity       = "fact-identity"
	LossAggregationInverse = "aggregation-inverse"
	LossQuantityScale      = "quantity-scale"
	LossTimeAxisRole       = "time-axis-role"
	LossSelectionIdentity  = "selection-identity"
	LossPresenceStates     = "presence-states"
	LossCoverage           = "coverage-completeness"
	LossSourceRevision     = "source-revision"
	LossUnitConversion     = "unit-conversion"
	LossUnitOriginal       = "unit-original"
	LossCRS                = "crs"
	LossPrecision          = "precision"
	LossFanout             = "fanout"
	LossRelationArity      = "relation-arity"
	LossInjectivity        = "injectivity"
	LossContextIdentity    = "context-identity"
)

// Diagnostic codes name why a verdict was reached. Severity decides which
// verdict wins; a diagnostic never replaces the loss list.
const (
	DiagClaimContradicted    = "CLAIM_CONTRADICTED"
	DiagLossUndeclared       = "LOSS_UNDECLARED"
	DiagEvidenceMissing      = "EVIDENCE_MISSING"
	DiagNoClaim              = "NO_PRESERVATION_CLAIM"
	DiagUnitUndeclared       = "UNIT_CONVERSION_UNDECLARED"
	DiagCRSUndeclared        = "CRS_UNDECLARED"
	DiagScaleConflated       = "QUANTITY_SCALE_CONFLATED"
	DiagAxisConflated        = "TIME_AXIS_CONFLATED"
	DiagSyncUnknown          = "SYNC_UNKNOWN"
	DiagSelectionIdentity    = "SELECTION_NOT_IDENTITY"
	DiagGrainWriteback       = "GRAIN_WRITEBACK"
	DiagCoverageOverclaim    = "COVERAGE_OVERCLAIMED"
	DiagRevisionStale        = "SOURCE_REVISION_STALE"
	DiagInverseUnqualified   = "INVERSE_UNQUALIFIED"
	DiagInjectivityUnproven  = "INJECTIVITY_UNPROVEN"
	DiagFanoutUndeclared     = "FANOUT_UNDECLARED"
	DiagApproximationClaimed = "APPROXIMATION_CLAIMED_PRESERVED"
	DiagContextMismatch      = "CONTEXT_BINDING_MISMATCH"
)

// Diagnostic severities. The verdict is chosen by the severest diagnostic.
const (
	SeverityReject      = "reject"
	SeverityUnqualified = "unqualified"
	SeverityUnknown     = "unknown"
	SeverityInfo        = "info"
)

// Facets are the observations on one side of the mapping. A facet that is not
// declared is empty and is never filled in by a guess.
type Facets struct {
	Identity     string   `json:"identity,omitempty"`
	Composition  string   `json:"composition,omitempty"`
	Order        string   `json:"order,omitempty"`
	OrderOrigin  string   `json:"orderOrigin,omitempty"`
	Multiplicity string   `json:"multiplicity,omitempty"`
	Grain        string   `json:"grain,omitempty"`
	Presence     []string `json:"presence,omitempty"`
	Precision    string   `json:"precision,omitempty"`
	TimeAxes     []string `json:"timeAxes,omitempty"`
	Timebase     string   `json:"timebase,omitempty"`
	Clock        string   `json:"clock,omitempty"`
}

// Structure is the data side. Relations lists the relation paths that the
// mapping actually consumes; an unconsumed relation is not an input.
type Structure struct {
	ID        string   `json:"id"`
	Family    string   `json:"family"`
	Relations []string `json:"relations,omitempty"`
	Facets    Facets   `json:"facets"`
}

// Representation is the representation side. A representation that carries
// meaning is always owned by a Function Data Context.
type Representation struct {
	ID        string   `json:"id"`
	Kind      string   `json:"kind"`
	Topology  []string `json:"topology,omitempty"`
	Axes      []string `json:"axes,omitempty"`
	Selection string   `json:"selection,omitempty"`
	ContextID string   `json:"contextId,omitempty"`
	Epoch     string   `json:"epoch,omitempty"`
	Facets    Facets   `json:"facets"`
}

// Claim is a preservation claim plus the evidence that is supposed to support
// it. A claim without evidence is reported as unknown, not as preserved.
type Claim struct {
	ClaimedPreserved bool     `json:"claimedPreserved"`
	Evidence         []string `json:"evidence,omitempty"`
}

type SourceColumn struct {
	Subject    string `json:"subject,omitempty"`
	Occurrence string `json:"occurrence,omitempty"`
	Grain      string `json:"grain,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Snapshot   string `json:"snapshot,omitempty"`
}

type TargetColumn struct {
	Representation string   `json:"representation,omitempty"`
	Slot           string   `json:"slot,omitempty"`
	Axes           []string `json:"axes,omitempty"`
	Selection      string   `json:"selection,omitempty"`
	ContextID      string   `json:"contextId,omitempty"`
	Epoch          string   `json:"epoch,omitempty"`
	Revision       string   `json:"revision,omitempty"`
}

// IdentityColumn separates the shape of the correspondence from the claim that
// individual identity survives it.
type IdentityColumn struct {
	Kind             string   `json:"kind"`
	ClaimedPreserved bool     `json:"claimedPreserved"`
	Evidence         []string `json:"evidence,omitempty"`
}

type MultisetColumn struct {
	Fanout           int      `json:"fanout"`
	ClaimedPreserved bool     `json:"claimedPreserved"`
	Evidence         []string `json:"evidence,omitempty"`
}

type GrainColumn struct {
	SourceGrain      string   `json:"sourceGrain,omitempty"`
	TargetGrain      string   `json:"targetGrain,omitempty"`
	Steps            int      `json:"steps"`
	ClaimedPreserved bool     `json:"claimedPreserved"`
	Evidence         []string `json:"evidence,omitempty"`
}

// QuantityColumn holds the declared quantity context. A conversion needs a
// policy, an instant (or a rate revision), and the source/target units; without
// them the conversion is unqualified rather than silently defaulted.
type QuantityColumn struct {
	SourceUnit        string   `json:"sourceUnit,omitempty"`
	TargetUnit        string   `json:"targetUnit,omitempty"`
	SourceScale       string   `json:"sourceScale,omitempty"`
	TargetScale       string   `json:"targetScale,omitempty"`
	Precision         string   `json:"precision,omitempty"`
	TargetPrecision   string   `json:"targetPrecision,omitempty"`
	ConversionPolicy  string   `json:"conversionPolicy,omitempty"`
	ConversionInstant string   `json:"conversionInstant,omitempty"`
	RateRevision      string   `json:"rateRevision,omitempty"`
	SourceCRS         string   `json:"sourceCrs,omitempty"`
	TargetCRS         string   `json:"targetCrs,omitempty"`
	CRSTransform      string   `json:"crsTransform,omitempty"`
	Approximation     string   `json:"approximation,omitempty"`
	ClaimedPreserved  bool     `json:"claimedPreserved"`
	Evidence          []string `json:"evidence,omitempty"`
}

type TimeColumn struct {
	SourceAxes       []string `json:"sourceAxes,omitempty"`
	TargetAxes       []string `json:"targetAxes,omitempty"`
	SourceClock      string   `json:"sourceClock,omitempty"`
	TargetClock      string   `json:"targetClock,omitempty"`
	SyncError        string   `json:"syncError,omitempty"`
	ClaimedPreserved bool     `json:"claimedPreserved"`
	Evidence         []string `json:"evidence,omitempty"`
}

type CoverageColumn struct {
	Claimed            string   `json:"claimed"`
	Observed           string   `json:"observed"`
	NegativeDependency []string `json:"negativeDependency,omitempty"`
	Evidence           []string `json:"evidence,omitempty"`
}

type LossColumn struct {
	Declared []string `json:"declared,omitempty"`
}

type OperationColumn struct {
	Mode            string `json:"mode,omitempty"`
	InverseEligible bool   `json:"inverseEligible"`
	ExpectedState   string `json:"expectedState,omitempty"`
	Authority       string `json:"authority,omitempty"`
}

// Composite describes a composed mapping Q∘P. Equal representation bytes from
// different sources never prove equal sources.
type Composite struct {
	Stages                    []string `json:"stages,omitempty"`
	RepresentationEqual       bool     `json:"representationEqual"`
	SourceEqual               bool     `json:"sourceEqual"`
	InverseFromRepresentation bool     `json:"inverseFromRepresentation"`
}

// Mapping is the typed contract. Twelve logical columns, no new Core schema.
type Mapping struct {
	Source    SourceColumn    `json:"source"`
	Target    TargetColumn    `json:"target"`
	Identity  IdentityColumn  `json:"identity"`
	Order     Claim           `json:"order"`
	Multiset  MultisetColumn  `json:"multiset"`
	Grain     GrainColumn     `json:"grain"`
	Presence  Claim           `json:"presence"`
	Quantity  QuantityColumn  `json:"quantity"`
	Time      TimeColumn      `json:"time"`
	Coverage  CoverageColumn  `json:"coverage"`
	Loss      LossColumn      `json:"loss"`
	Operation OperationColumn `json:"operation"`
}

// Case is one mapping under test.
type Case struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Structure      Structure      `json:"structure"`
	Representation Representation `json:"representation"`
	Mapping        Mapping        `json:"mapping"`
	Composite      *Composite     `json:"composite,omitempty"`
}

// Diagnostic explains one observation; it never grants a permission.
type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Detail   string `json:"detail,omitempty"`
}

// Result is the verdict for one case.
type Result struct {
	ID          string       `json:"id"`
	Title       string       `json:"title,omitempty"`
	Verdict     string       `json:"verdict"`
	Losses      []string     `json:"losses"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Inverse     string       `json:"inverse"`
	Evidence    []string     `json:"evidence"`
	// Unknowns lists the facets that were claimed but not evidenced, or that
	// could not be compared at all.
	Unknowns []string `json:"unknowns,omitempty"`
}

// Expectation is the hand-written independent expected for one case.
type Expectation struct {
	ID      string   `json:"id"`
	Verdict string   `json:"verdict"`
	Losses  []string `json:"losses"`
	Inverse string   `json:"inverse"`
	Note    string   `json:"note,omitempty"`
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

type Output struct {
	SchemaVersion string      `json:"schemaVersion"`
	Checked       int         `json:"checked"`
	Results       []Result    `json:"results"`
	Comparison    *Comparison `json:"comparison,omitempty"`
}

// Options exist so a known mutant can be injected into the checker itself. A
// mutant that cannot be told apart from the correct checker is not detected.
type Options struct {
	IgnoreOrder           bool
	BinarizePresence      bool
	AllowAggregateInverse bool
	IgnoreQuantityScale   bool
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
	for _, c := range f.Cases {
		if c.ID == "" {
			return CaseFile{}, fmt.Errorf("case without id")
		}
		if seen[c.ID] {
			return CaseFile{}, fmt.Errorf("duplicate case id %q", c.ID)
		}
		seen[c.ID] = true
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
	case VerdictPreserved, VerdictLossy, VerdictRejected, VerdictUnqualified, VerdictUnknown:
		return true
	}
	return false
}

// Compare matches the checked results against the hand-written expectations.
// It compares the verdict, the loss set, and the inverse qualification, and
// nothing else: diagnostics are explanatory and are not part of the contract.
func Compare(report Output, expected ExpectedFile) Comparison {
	byID := make(map[string]Result, len(report.Results))
	for _, r := range report.Results {
		byID[r.ID] = r
	}
	cmp := Comparison{SchemaVersion: ReportSchema, Total: len(expected.Expectations)}
	for _, want := range expected.Expectations {
		got, ok := byID[want.ID]
		if !ok {
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{
				ID: want.ID, Field: "missing", Expected: "checked", Observed: "absent",
			})
			continue
		}
		matched := true
		if got.Verdict != want.Verdict {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{
				ID: want.ID, Field: "verdict", Expected: want.Verdict, Observed: got.Verdict,
			})
		}
		if !equalSet(sortedCopy(got.Losses), sortedCopy(want.Losses)) {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{
				ID: want.ID, Field: "losses",
				Expected: joinList(sortedCopy(want.Losses)),
				Observed: joinList(sortedCopy(got.Losses)),
			})
		}
		if got.Inverse != want.Inverse {
			matched = false
			cmp.Mismatches = append(cmp.Mismatches, Mismatch{
				ID: want.ID, Field: "inverse", Expected: want.Inverse, Observed: got.Inverse,
			})
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

func joinList(in []string) string {
	if len(in) == 0 {
		return "(none)"
	}
	out := ""
	for i, s := range in {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

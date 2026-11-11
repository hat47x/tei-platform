package representationbindingcheck

import (
	"encoding/json"
	"fmt"
	"os"
)

const (
	InputSchema    = "tei.reference.representation-binding-input/v0"
	ExpectedSchema = "tei.reference.representation-binding-expected/v0"
	ReportSchema   = "tei.reference.representation-binding-check/v0"
)

// The fifteen representation types the issue names. The vocabulary is closed:
// a type outside it is a malformed descriptor, not a new representation type.
const (
	RepTable     = "table"
	RepForm      = "form"
	RepTree      = "tree"
	RepGraph     = "graph"
	RepChart     = "chart"
	RepGantt     = "gantt"
	RepMap       = "map"
	RepMatrix    = "matrix"
	RepMarkdown  = "markdown"
	RepFormula   = "formula"
	RepDrawing   = "drawing"
	RepImage     = "image"
	RepAudio     = "audio"
	RepVideo     = "video"
	RepComposite = "composite"
)

// Verdicts.
const (
	VerdictBound       = "bound"
	VerdictUnqualified = "unqualified"
	VerdictRejected    = "rejected"
	VerdictUnknown     = "unknown"
)

// Operation qualifications. A read-only projection never implies an inverse.
const (
	OpReadOnly       = "read-only"
	OpInverseClaim   = "inverse-candidate"
	OpExplicitAction = "explicit-action"
)

// Time axis roles. Four axes, never one.
const (
	AxisDomain       = "domain"
	AxisObservation  = "observation"
	AxisPresentation = "presentation"
	AxisStructural   = "structural"
)

// Diagnostic codes.
const (
	DiagContextMissing        = "CONTEXT_MISSING"
	DiagContextDuplicate      = "CONTEXT_DUPLICATE"
	DiagScopeReused           = "SCOPE_REUSED_ACROSS_CONTEXTS"
	DiagAxisUndeclared        = "AXIS_UNDECLARED"
	DiagAxisConflated         = "AXIS_ROLE_CONFLATED"
	DiagGrainUndeclared       = "GRAIN_UNDECLARED"
	DiagSelectionNotIdentity  = "SELECTION_IS_NOT_IDENTITY"
	DiagTopologyPromoted      = "TOPOLOGY_PROMOTED_TO_MEANING"
	DiagOperationUnqualified  = "OPERATION_UNQUALIFIED"
	DiagInverseUnqualified    = "INVERSE_UNQUALIFIED"
	DiagDisplayNotMeaning     = "DISPLAY_EQUALITY_IS_NOT_MEANING"
	DiagAggregateNotEntity    = "AGGREGATE_IS_NOT_ENTITY"
	DiagNodeNotIdentity       = "TREE_NODE_IS_NOT_IDENTITY"
	DiagCRSUndeclared         = "CRS_UNDECLARED"
	DiagGeneratedAxis         = "GENERATED_AXIS_NOT_SOURCE"
	DiagClockUnknown          = "CLOCK_UNKNOWN"
	DiagSyncClaimUnbound      = "SYNC_CLAIM_WITHOUT_BOUND"
	DiagEvidenceMissing       = "EVIDENCE_MISSING"
	DiagUnknownRepresentation = "REPRESENTATION_UNKNOWN"
)

const (
	SeverityReject      = "reject"
	SeverityUnqualified = "unqualified"
	SeverityUnknown     = "unknown"
	SeverityInfo        = "info"
)

// Representation is the declaration side. The internal topology is listed and
// never adopted as data meaning.
type Representation struct {
	ID                    string   `json:"id"`
	Kind                  string   `json:"kind"`
	Topology              []string `json:"topology,omitempty"`
	TopologyIsDataMeaning bool     `json:"topologyIsDataMeaning"`
}

// Axis is one axis the representation offers and the role it plays.
type Axis struct {
	Name      string `json:"name"`
	Role      string `json:"role"`
	Generated bool   `json:"generated"`
}

// Binding is the Function Data Context side. It also carries what the binding
// claims about the source, so an overclaim can be refused.
type Binding struct {
	ContextID string `json:"contextId"`
	Epoch     string `json:"epoch,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Grain     string `json:"grain,omitempty"`
	// SourceGrain names the grain the source actually has. A rendered grain
	// that differs from it means an aggregation happened.
	SourceGrain string `json:"sourceGrain,omitempty"`
	Axes        []Axis `json:"axes,omitempty"`
	Selection   string `json:"selection,omitempty"`
	// SelectionIsIdentity records the claim that the presentation selection is
	// the data identity. It is refused.
	SelectionIsIdentity bool `json:"selectionIsIdentity"`
	// NodeIsUniqueTarget records the claim that a tree node is the unique edit
	// target. It is refused when the binding also declares hidden edges.
	NodeIsUniqueTarget bool `json:"nodeIsUniqueTarget"`
	// DisplayEqualityImpliesMeaning records the claim that equal rendered bytes
	// establish equal computation. It is refused.
	DisplayEqualityImpliesMeaning bool     `json:"displayEqualityImpliesMeaning"`
	Operation                     string   `json:"operation,omitempty"`
	SourceAxes                    []string `json:"sourceAxes,omitempty"`
	Clock                         string   `json:"clock,omitempty"`
	SyncError                     string   `json:"syncError,omitempty"`
	SyncClaimed                   bool     `json:"syncClaimed"`
	// CRS names the coordinate reference system. A map without one cannot claim
	// to place a feature geographically.
	CRS string `json:"crs,omitempty"`
	// DeclaredLosses lists the meaning the binding already knows it drops.
	DeclaredLosses []string `json:"declaredLosses,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
}

// Case is one representation instance under test.
type Case struct {
	ID             string         `json:"id"`
	Title          string         `json:"title"`
	Representation Representation `json:"representation"`
	Binding        Binding        `json:"binding"`
	OtherBindings  []Binding      `json:"otherBindings,omitempty"`
}

type Diagnostic struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Detail   string `json:"detail,omitempty"`
}

type Result struct {
	ID          string       `json:"id"`
	Title       string       `json:"title,omitempty"`
	Verdict     string       `json:"verdict"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Evidence    []string     `json:"evidence"`
	Unqualified []string     `json:"unqualified,omitempty"`
}

type Expectation struct {
	ID          string   `json:"id"`
	Verdict     string   `json:"verdict"`
	Unqualified []string `json:"unqualified,omitempty"`
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

// Contract is the machine-readable self-description, matching the shape used by
// #4351 and #4352 so one consumer reads all three the same way.
type Contract struct {
	SchemaVersion          string   `json:"schemaVersion"`
	RepresentationTypes    []string `json:"representationTypes"`
	Verdicts               []string `json:"verdicts"`
	Operations             []string `json:"operations"`
	TimeAxes               []string `json:"timeAxes"`
	RequiredBindingColumns []string `json:"requiredBindingColumns"`
	Rules                  []string `json:"rules"`
	NotClaims              []string `json:"notClaims"`
}

type Output struct {
	SchemaVersion string      `json:"schemaVersion"`
	Checked       int         `json:"checked"`
	Contract      *Contract   `json:"contract,omitempty"`
	Results       []Result    `json:"results"`
	Comparison    *Comparison `json:"comparison,omitempty"`
}

// Options injects the known mutants into the checker itself.
type Options struct {
	// AcceptContextless accepts an instance that is placed outside any context.
	AcceptContextless bool
	// TreatTopologyAsMeaning adopts the layout as data meaning.
	TreatTopologyAsMeaning bool
	// TreatSelectionAsIdentity reads the data identity from the selection.
	TreatSelectionAsIdentity bool
	// AcceptUnknownClock claims synchronization without a clock.
	AcceptUnknownClock bool
}

func ContractDescription() Contract {
	return Contract{
		SchemaVersion: ReportSchema,
		RepresentationTypes: []string{
			RepTable, RepForm, RepTree, RepGraph, RepChart, RepGantt, RepMap,
			RepMatrix, RepMarkdown, RepFormula, RepDrawing, RepImage, RepAudio,
			RepVideo, RepComposite,
		},
		Verdicts:               []string{VerdictBound, VerdictUnqualified, VerdictRejected, VerdictUnknown},
		Operations:             []string{OpReadOnly, OpInverseClaim, OpExplicitAction},
		TimeAxes:               []string{AxisDomain, AxisObservation, AxisPresentation, AxisStructural},
		RequiredBindingColumns: []string{"contextId", "scope", "grain", "axes", "selection", "operation"},
		Rules: []string{
			"bound: the instance is owned by one context and every required binding column is declared",
			"unqualified: the instance is owned but a required binding column is missing",
			"rejected: the declaration contradicts an observed boundary (layout as meaning, selection as identity, two contexts, scope reuse)",
			"unknown: the representation type is outside the closed fifteen, or no evidence is recorded",
			"an instance that carries meaning is always placed inside a Function Data Context",
			"a generated axis is not a source axis; the four time roles are never conflated",
			"a read-only operation never qualifies an inverse; an inverse claim needs its own qualification",
			"a tree node is not the identity of the relation it renders; hidden edges forbid a unique target",
			"an aggregate point is not the entity it aggregates",
			"equal rendered bytes do not establish equal computation",
		},
		NotClaims: []string{
			"no renderer, widget, UI, media playback, or synchronization is exercised",
			"no Context type, representation registry, or renderer plugin contract is adopted",
			"no business meaning is accepted; bound is a comparison result on synthetic descriptors",
			"a representation's internal topology never becomes data meaning",
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
	case VerdictBound, VerdictUnqualified, VerdictRejected, VerdictUnknown:
		return true
	}
	return false
}

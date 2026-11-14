package operations

import (
	"errors"
	"fmt"
	"regexp"

	"tei/internal/workspace"
)

// The three-view navigation Operation is a read-only contract. It says how a
// Data / Function / Business reading of one Workspace source is carried to a
// consumer (CLI, Studio), and nothing about where the reading comes from: a
// Provider selected by the Host implements it. This package holds the contract
// and its check only, so no source format and no Provider is named here.
//
// Three layers are kept apart on every item and never merged:
//
//   - a source fact is what the source says explicitly;
//   - a structural candidate is a reading a Provider derived from source facts
//     (the view an element belongs to, a path joined from several facts);
//   - an accepted meaning is a Canonical subject an Owner accepted.
//
// Version 0 carries no accepted-meaning binding at all. A Provider cannot
// report one, and ValidateThreeViewNavigation rejects an attempt.
const (
	ThreeViewNavigationSchemaVersion = "tei.operations.three-view-navigation/v0"

	ThreeViewData     = "data"
	ThreeViewFunction = "function"
	ThreeViewBusiness = "business"

	ThreeViewNatureSourceFact          = "source-fact"
	ThreeViewNatureStructuralCandidate = "structural-candidate"

	// ThreeViewMeaningNotBound is the only meaning status of version 0.
	ThreeViewMeaningNotBound = "not-bound"

	ThreeViewSourceIDUnique     = "unique"
	ThreeViewSourceIDMissing    = "missing"
	ThreeViewSourceIDDuplicated = "duplicated"

	ThreeViewCoverageComplete = "complete"
	ThreeViewCoveragePartial  = "partial"

	// ThreeViewScopeSingleSource says the projection covers one source file.
	ThreeViewScopeSingleSource = "single-source-artifact"

	ThreeViewPinHeld               = "held"
	ThreeViewPinStaleSourceChanged = "stale-source-changed"
	ThreeViewPinOccurrenceNotFound = "occurrence-not-found"

	ThreeViewRelationFunctionDataOperation = "function-data-operation"
	ThreeViewCRUDScopeFunctionDeclared     = "function-declared"
)

// Typed failure codes of the Operation. A consumer shows the code; Provider
// text is not forwarded, because it may carry a private path or source text.
const (
	ThreeViewSourceReadFailed  = "THREE_VIEW_SOURCE_READ_FAILED"
	ThreeViewSourceTooLarge    = "THREE_VIEW_SOURCE_TOO_LARGE"
	ThreeViewUnsupportedSource = "THREE_VIEW_UNSUPPORTED_SOURCE"
	ThreeViewSourceRejected    = "THREE_VIEW_SOURCE_REJECTED"
	ThreeViewProjectionFailed  = "THREE_VIEW_PROJECTION_FAILED"
)

// ThreeViewError is the typed failure a Provider returns.
type ThreeViewError struct{ Code string }

func (e *ThreeViewError) Error() string {
	if e == nil {
		return ThreeViewProjectionFailed
	}
	return e.Code
}

// ThreeViewRequest names a Workspace-relative source and, optionally, a pin.
type ThreeViewRequest struct {
	SourcePath string        `json:"sourcePath"`
	Pin        *ThreeViewPin `json:"pin,omitempty"`
}

// ThreeViewPin is a target the caller fixed on purpose. It is the caller's
// value: the Operation reports whether it still holds and returns it
// unchanged. It is bound to the exact source bytes it was taken from, so a
// changed source makes it stale; it is never re-pointed at an element that has
// the same name or the same source ID.
type ThreeViewPin struct {
	Occurrence           string `json:"occurrence"`
	SourceArtifactDigest string `json:"sourceArtifactDigest"`
}

// ThreeViewPinState is the pin the request carried and what became of it.
type ThreeViewPinState struct {
	ThreeViewPin
	Status string `json:"status"`
}

// ThreeViewNavigation is the result of the Operation.
type ThreeViewNavigation struct {
	SchemaVersion string                       `json:"schemaVersion"`
	Effect        string                       `json:"effect"`
	Workspace     *workspace.WorkspaceIdentity `json:"workspace"`
	Source        ThreeViewSource              `json:"source"`
	Subjects      []ThreeViewSubject           `json:"subjects"`
	Fields        []ThreeViewField             `json:"fields,omitempty"`
	Relations     []ThreeViewRelation          `json:"relations"`
	CRUD          []ThreeViewCRUDRow           `json:"crud"`
	// FieldReferences are item-level references: which occurrence refers to
	// which field of which data subject.
	FieldReferences []ThreeViewFieldReference `json:"fieldReferences,omitempty"`
	Unresolved      []ThreeViewUnresolved     `json:"unresolved,omitempty"`
	Pin             *ThreeViewPinState        `json:"pin,omitempty"`
	AuthorityLimits []string                  `json:"authorityLimits"`
}

// ThreeViewSource identifies the exact bytes the projection was read from.
type ThreeViewSource struct {
	Path           string `json:"path"`
	Format         string `json:"format"`
	ArtifactDigest string `json:"artifactDigest"`
	// MappingFingerprint and DerivationRevision identify the reading rules. A
	// change of either makes an earlier projection of the same bytes stale.
	MappingFingerprint    string            `json:"mappingFingerprint"`
	DerivationRevision    string            `json:"derivationRevision"`
	ProjectionFingerprint string            `json:"projectionFingerprint"`
	Coverage              ThreeViewCoverage `json:"coverage"`
}

// ThreeViewCoverage says how much of the source the projection accounts for.
// Something absent from a projection is not thereby absent from the system:
// Scope names what was read, and the counts name what was read but not placed.
type ThreeViewCoverage struct {
	Scope  string `json:"scope"`
	Status string `json:"status"`
	// UnplacedConstructs counts elements the reading rules do not place in any
	// view; UnknownConstructs counts elements the reader does not recognise.
	UnplacedConstructs  int `json:"unplacedConstructs"`
	UnknownConstructs   int `json:"unknownConstructs"`
	UnresolvedRelations int `json:"unresolvedRelations"`
	// OmittedRelations counts, by kind, source relations the reader found whose
	// ends are not subjects of this contract. They are counted, not dropped.
	OmittedRelations map[string]int `json:"omittedRelations,omitempty"`
}

// ThreeViewSubject is one occurrence shown in one of the three views.
type ThreeViewSubject struct {
	// Occurrence addresses this element in these exact source bytes. It is the
	// key every relation, CRUD row and field reference uses, so the same
	// occurrence is reachable from each of them. It is not an identity.
	Occurrence string `json:"occurrence"`
	View       string `json:"view"`
	// ParentOccurrence is set for a subject nested in another subject.
	ParentOccurrence string              `json:"parentOccurrence,omitempty"`
	SourceFact       ThreeViewSourceFact `json:"sourceFact"`
	Candidate        ThreeViewCandidate  `json:"candidate"`
	Meaning          ThreeViewMeaning    `json:"meaning"`
	// SameNameOccurrences lists other subjects of this view with the same name.
	// They are different occurrences; nothing here says they are the same thing.
	SameNameOccurrences []string `json:"sameNameOccurrences,omitempty"`
}

// ThreeViewSourceFact is what the source states about one element.
type ThreeViewSourceFact struct {
	Kind    string `json:"kind"`
	Locator string `json:"locator"`
	Name    string `json:"name,omitempty"`
	// SourceID is the element's own ID in the source. SourceIDStatus says
	// whether that ID singles the element out within its kind in this source.
	SourceID       string `json:"sourceId,omitempty"`
	SourceIDStatus string `json:"sourceIdStatus"`
}

// ThreeViewCandidate is the Provider's reading of one element.
type ThreeViewCandidate struct {
	Nature          string   `json:"nature"`
	Classifications []string `json:"classifications,omitempty"`
	AuthorityLimits []string `json:"authorityLimits"`
}

// ThreeViewMeaning is the accepted-meaning layer of one subject.
type ThreeViewMeaning struct {
	Status string `json:"status"`
}

// ThreeViewField is one field of a data subject.
type ThreeViewField struct {
	Occurrence      string              `json:"occurrence"`
	OwnerOccurrence string              `json:"ownerOccurrence"`
	SourceFact      ThreeViewSourceFact `json:"sourceFact"`
}

// ThreeViewRelation is one relation occurrence between two subjects.
type ThreeViewRelation struct {
	Occurrence string `json:"occurrence"`
	Kind       string `json:"kind"`
	Nature     string `json:"nature"`
	From       string `json:"from"`
	To         string `json:"to"`
	Operations string `json:"operations,omitempty"`
	// SourceOccurrences are the source elements the relation rests on. A
	// relation derived from several facts keeps all of them.
	SourceOccurrences         []string `json:"sourceOccurrences"`
	DerivationRuleFingerprint string   `json:"derivationRuleFingerprint,omitempty"`
	AuthorityLimits           []string `json:"authorityLimits"`
}

// ThreeViewCRUDRow is one cell group of the CRUD matrix. It is always the
// function's own declaration; it is never attributed to a business action.
type ThreeViewCRUDRow struct {
	Function     string `json:"function"`
	FunctionName string `json:"functionName,omitempty"`
	Data         string `json:"data"`
	DataName     string `json:"dataName,omitempty"`
	Create       bool   `json:"create"`
	Read         bool   `json:"read"`
	Update       bool   `json:"update"`
	Delete       bool   `json:"delete"`
	Scope        string `json:"scope"`
	// Relations are the source-fact relation occurrences behind this row.
	Relations []string `json:"relations"`
	// Conflicting is set when the declarations behind this row disagree. The
	// row then shows one of them; it is a review flag, not a resolution.
	Conflicting bool `json:"conflicting,omitempty"`
}

// ThreeViewFieldReference is one item-level reference.
type ThreeViewFieldReference struct {
	Occurrence string `json:"occurrence"`
	Kind       string `json:"kind"`
	Nature     string `json:"nature"`
	// Owner is the data subject the reference is declared under.
	Owner string `json:"owner"`
	// Field is set only when exactly one field of Owner has the referenced
	// source ID. Otherwise Resolution says why it is not.
	Field           string   `json:"field,omitempty"`
	ReferencedID    string   `json:"referencedId,omitempty"`
	Resolution      string   `json:"resolution"`
	AuthorityLimits []string `json:"authorityLimits"`
}

// ThreeViewUnresolved is a relation the source implies that could not be tied
// to exactly one element at each end.
type ThreeViewUnresolved struct {
	Code              string   `json:"code"`
	RelationKind      string   `json:"relationKind"`
	SourceOccurrences []string `json:"sourceOccurrences"`
}

var threeViewDigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// ValidateThreeViewNavigation is the Host-side check of a Provider's answer.
// It checks shape and internal consistency: that the answer is about the
// requested source, that every reference lands on a listed occurrence, that
// the layers use the closed vocabulary, and that the pin came back unchanged.
// It does not check that the reading is right, and passing it grants nothing.
func ValidateThreeViewNavigation(request ThreeViewRequest, value ThreeViewNavigation) error {
	if value.SchemaVersion != ThreeViewNavigationSchemaVersion || value.Effect != "read-only" {
		return errors.New("three-view navigation has an unexpected schema or effect")
	}
	if value.Workspace == nil || value.Workspace.Manifest.SHA256 == "" {
		return errors.New("three-view navigation does not name its Workspace")
	}
	if value.Source.Path == "" || value.Source.Path != request.SourcePath {
		return errors.New("three-view navigation answers for another source")
	}
	if !threeViewDigestPattern.MatchString(value.Source.ArtifactDigest) ||
		value.Source.MappingFingerprint == "" || value.Source.DerivationRevision == "" ||
		value.Source.ProjectionFingerprint == "" || value.Source.Format == "" {
		return errors.New("three-view navigation source binding is incomplete")
	}
	coverage := value.Source.Coverage
	if coverage.Scope != ThreeViewScopeSingleSource ||
		(coverage.Status != ThreeViewCoverageComplete && coverage.Status != ThreeViewCoveragePartial) {
		return errors.New("three-view navigation coverage is not stated")
	}
	omitted := 0
	for _, count := range coverage.OmittedRelations {
		omitted += count
	}
	incomplete := coverage.UnplacedConstructs != 0 || coverage.UnknownConstructs != 0 ||
		coverage.UnresolvedRelations != 0 || omitted != 0
	if incomplete == (coverage.Status == ThreeViewCoverageComplete) ||
		coverage.UnresolvedRelations != len(value.Unresolved) {
		return errors.New("three-view navigation coverage disagrees with its counts")
	}
	if len(value.AuthorityLimits) == 0 {
		return errors.New("three-view navigation states no authority limits")
	}

	subjects := map[string]ThreeViewSubject{}
	occurrences := map[string]bool{}
	claim := func(occurrence string) error {
		if occurrence == "" || occurrences[occurrence] {
			return fmt.Errorf("three-view navigation repeats or omits an occurrence")
		}
		occurrences[occurrence] = true
		return nil
	}
	for _, subject := range value.Subjects {
		if err := claim(subject.Occurrence); err != nil {
			return err
		}
		switch subject.View {
		case ThreeViewData, ThreeViewFunction, ThreeViewBusiness:
		default:
			return errors.New("three-view navigation has a subject outside the three views")
		}
		if err := validateThreeViewSourceFact(subject.SourceFact); err != nil {
			return err
		}
		if subject.Candidate.Nature != ThreeViewNatureStructuralCandidate || len(subject.Candidate.AuthorityLimits) == 0 {
			return errors.New("three-view subject placement must be a limited structural candidate")
		}
		// No accepted meaning can be reported in this version, so a Provider
		// cannot make a candidate look accepted by labelling it.
		if subject.Meaning.Status != ThreeViewMeaningNotBound {
			return errors.New("three-view navigation v0 cannot carry an accepted meaning")
		}
		subjects[subject.Occurrence] = subject
	}
	for _, subject := range value.Subjects {
		if subject.ParentOccurrence != "" {
			if _, ok := subjects[subject.ParentOccurrence]; !ok {
				return errors.New("three-view subject names a parent that is not a subject")
			}
		}
		for _, other := range subject.SameNameOccurrences {
			if _, ok := subjects[other]; !ok || other == subject.Occurrence {
				return errors.New("three-view same-name occurrence is not another subject")
			}
		}
	}
	fields := map[string]ThreeViewField{}
	for _, field := range value.Fields {
		if err := claim(field.Occurrence); err != nil {
			return err
		}
		if owner, ok := subjects[field.OwnerOccurrence]; !ok || owner.View != ThreeViewData {
			return errors.New("three-view field is not owned by a data subject")
		}
		if err := validateThreeViewSourceFact(field.SourceFact); err != nil {
			return err
		}
		fields[field.Occurrence] = field
	}

	relations := map[string]ThreeViewRelation{}
	for _, relation := range value.Relations {
		if err := claim(relation.Occurrence); err != nil {
			return err
		}
		if relation.Nature != ThreeViewNatureSourceFact && relation.Nature != ThreeViewNatureStructuralCandidate {
			return errors.New("three-view relation nature is outside the closed vocabulary")
		}
		if relation.Kind == "" || len(relation.SourceOccurrences) == 0 || len(relation.AuthorityLimits) == 0 {
			return errors.New("three-view relation lacks its kind, evidence or limits")
		}
		if relation.Nature == ThreeViewNatureStructuralCandidate && relation.DerivationRuleFingerprint == "" {
			return errors.New("three-view derived relation does not name its derivation rule")
		}
		if _, ok := subjects[relation.From]; !ok {
			return errors.New("three-view relation starts at an occurrence that is not a subject")
		}
		if _, ok := subjects[relation.To]; !ok {
			return errors.New("three-view relation ends at an occurrence that is not a subject")
		}
		relations[relation.Occurrence] = relation
	}
	for _, row := range value.CRUD {
		function, okFunction := subjects[row.Function]
		data, okData := subjects[row.Data]
		if !okFunction || !okData || function.View != ThreeViewFunction || data.View != ThreeViewData {
			return errors.New("three-view CRUD row is not a function over a data subject")
		}
		if row.Scope != ThreeViewCRUDScopeFunctionDeclared || len(row.Relations) == 0 {
			return errors.New("three-view CRUD row must be a function-level declaration with its relations")
		}
		for _, occurrence := range row.Relations {
			relation, ok := relations[occurrence]
			if !ok || relation.Nature != ThreeViewNatureSourceFact ||
				relation.Kind != ThreeViewRelationFunctionDataOperation ||
				relation.From != row.Function || relation.To != row.Data {
				return errors.New("three-view CRUD row does not rest on its own source-fact relation")
			}
		}
	}
	for _, reference := range value.FieldReferences {
		if err := claim(reference.Occurrence); err != nil {
			return err
		}
		if reference.Nature != ThreeViewNatureSourceFact || reference.Kind == "" ||
			reference.Resolution == "" || len(reference.AuthorityLimits) == 0 {
			return errors.New("three-view field reference is incomplete")
		}
		if owner, ok := subjects[reference.Owner]; !ok || owner.View != ThreeViewData {
			return errors.New("three-view field reference is not declared under a data subject")
		}
		if reference.Field != "" {
			if field, ok := fields[reference.Field]; !ok || field.OwnerOccurrence != reference.Owner {
				return errors.New("three-view field reference points outside its data subject")
			}
		}
	}

	switch {
	case request.Pin == nil && value.Pin != nil:
		return errors.New("three-view navigation invented a pin")
	case request.Pin != nil && value.Pin == nil:
		return errors.New("three-view navigation dropped the caller's pin")
	case request.Pin != nil:
		if value.Pin.ThreeViewPin != *request.Pin {
			return errors.New("three-view navigation changed the caller's pin")
		}
		_, isSubject := subjects[request.Pin.Occurrence]
		sameSource := request.Pin.SourceArtifactDigest == value.Source.ArtifactDigest
		var want string
		switch {
		case !sameSource:
			want = ThreeViewPinStaleSourceChanged
		case isSubject:
			want = ThreeViewPinHeld
		default:
			want = ThreeViewPinOccurrenceNotFound
		}
		if value.Pin.Status != want {
			return errors.New("three-view navigation misreports the caller's pin")
		}
	}
	return nil
}

func validateThreeViewSourceFact(fact ThreeViewSourceFact) error {
	if fact.Kind == "" || fact.Locator == "" {
		return errors.New("three-view source fact lacks its kind or locator")
	}
	switch fact.SourceIDStatus {
	case ThreeViewSourceIDUnique, ThreeViewSourceIDDuplicated:
		if fact.SourceID == "" {
			return errors.New("three-view source fact reports a status for an ID it does not carry")
		}
	case ThreeViewSourceIDMissing:
		if fact.SourceID != "" {
			return errors.New("three-view source fact reports a present ID as missing")
		}
	default:
		return errors.New("three-view source ID status is outside the closed vocabulary")
	}
	return nil
}

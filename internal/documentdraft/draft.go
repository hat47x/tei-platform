// Package documentdraft projects ONE accepted element and ONE declared relation
// of a TEI Workspace into a document draft that an external editor (SHO) can
// open, edit, save and resume.
//
// The package is a read-only projection layer. It never writes a Source, never
// accepts meaning and never grants authority. It owns three things and nothing
// else:
//
//   - the model reference: which Workspace, which World/Source Index
//     fingerprints and which managed Source revision the draft was read at;
//   - the document projection: the generated explanation, relation, table and
//     citation occurrences, each carrying the subject identity, the revision it
//     was read at and the generation conditions that produced it;
//   - the reconciliation: a read-only comparison between a stored draft and the
//     current Workspace that names which occurrences need reconfirmation.
//
// The draft is a derived artifact, not a second canonical model. A consumer that
// saved a document MUST NOT be read as having ingested the model or updated
// Accepted Meaning.
package documentdraft

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tei/internal/operations"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// SchemaVersion is the draft contract. A consumer that does not know this exact
// value must refuse the document instead of guessing a shape.
const SchemaVersion = "tei.document-draft/v0"

// EffectReadOnlyProjection marks every answer this package returns.
const EffectReadOnlyProjection = "read-only-projection"

// GeneratorName and GeneratorVersion name the projection that produced the
// occurrences. They are recorded so a later reader can tell whether the same
// rule produced the text it is looking at.
const (
	GeneratorName    = "tei.internal.documentdraft"
	GeneratorVersion = "v0"
)

// DocumentKindElementExplanation is the only kind this first integration
// supports: one element plus one relation explained as prose, a table and
// citations. Later stages (CRUD table, three-element diagram, specification,
// frozen delivery) are separate qualifications.
const DocumentKindElementExplanation = "element-explanation"

// Occurrence kinds. Every generated thing in the document is one occurrence so
// that the model identity, revision and generation conditions it came from can
// be traced back.
const (
	OccurrenceExplanation = "explanation"
	OccurrenceRelation    = "relation"
	OccurrenceTable       = "table"
	OccurrenceCitation    = "citation"
	OccurrenceDiagram     = "diagram"
)

// Relation resolutions. A relation whose target is not declared is reported as
// unresolved rather than silently dropped, and an ambiguous target stays
// ambiguous.
const (
	ResolutionResolved   = "resolved"
	ResolutionUnresolved = "unresolved"
	ResolutionAmbiguous  = "ambiguous"
)

// DraftAuthorityLimits is the fixed set of statements every draft makes about
// its own authority. It is a constant list, not a per-caller decision.
func DraftAuthorityLimits() []string {
	return []string{
		"draft-is-a-derived-read-only-projection",
		"source-location-is-not-semantic-acceptance",
		"source-location-is-not-write-authority",
		"model-reference-is-not-a-lock-or-acceptance-record",
		"consumer-save-is-not-tei-model-ingestion",
		"reprojection-never-overwrites-author-text",
		"occurrence-id-is-snapshot-local-not-a-persistent-document-id",
		"tei-md-authoring-parser-is-unbound-on-current-main",
	}
}

// ModelRef identifies the exact read a draft was produced from.
type ModelRef struct {
	Workspace              *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
	WorldFingerprint       string                       `json:"worldFingerprint"`
	SourceIndexFingerprint string                       `json:"sourceIndexFingerprint"`
	ModelRevision          string                       `json:"modelRevision"`
}

// SourceBinding is one Source artifact a subject was read from. The fingerprint
// is the byte identity of that artifact; the ranges are the declaration
// positions. None of them is an edit or disclosure permission.
type SourceBinding struct {
	ArtifactPath string              `json:"artifactPath"`
	Format       string              `json:"format,omitempty"`
	Fingerprint  string              `json:"fingerprint"`
	Role         string              `json:"role,omitempty"`
	Ranges       []sourceindex.Range `json:"ranges,omitempty"`
}

// SubjectRef is one referenced subject with the Sources it was read from.
type SubjectRef struct {
	Subject sourceindex.Subject `json:"subject"`
	Symbol  string              `json:"symbol,omitempty"`
	Sources []SourceBinding     `json:"sources"`
}

// RelationKeyBinding is one declared source/target attribute correspondence.
type RelationKeyBinding struct {
	SourceAttribute string `json:"sourceAttribute"`
	TargetAttribute string `json:"targetAttribute"`
}

// RelationRef is the ONE relation the draft read. Resolution reports whether the
// declared target was found in the same read.
type RelationRef struct {
	Relation    sourceindex.Subject  `json:"relation"`
	From        sourceindex.Subject  `json:"from"`
	To          sourceindex.Subject  `json:"to"`
	Resolution  string               `json:"resolution"`
	Cardinality string               `json:"cardinality,omitempty"`
	Keys        []RelationKeyBinding `json:"keys,omitempty"`
	OnDelete    string               `json:"onDelete,omitempty"`
}

// GenerationConditions records what produced an occurrence. InputDigests holds
// the model read the content was derived from, so a later comparison can tell a
// content change from a model-reference change.
type GenerationConditions struct {
	Generator        string            `json:"generator"`
	GeneratorVersion string            `json:"generatorVersion"`
	Rule             string            `json:"rule"`
	InputDigests     map[string]string `json:"inputDigests"`
}

// TableContent is a generated table: a header row and body rows of strings.
type TableContent struct {
	Caption string     `json:"caption,omitempty"`
	Header  []string   `json:"header"`
	Rows    [][]string `json:"rows"`
}

// CitationContent is one citation occurrence. It records the Source identity and
// revision; it does not acquire, resolve or verify them.
type CitationContent struct {
	ArtifactPath string              `json:"artifactPath"`
	Fingerprint  string              `json:"fingerprint"`
	Role         string              `json:"role,omitempty"`
	Ranges       []sourceindex.Range `json:"ranges,omitempty"`
	Resolution   string              `json:"resolution"`
}

// OccurrenceContent is the generated payload of one occurrence. Exactly one of
// Text, Relation, Table or Citation is set for a given Kind.
type OccurrenceContent struct {
	Text     string           `json:"text,omitempty"`
	Relation *RelationRef     `json:"relation,omitempty"`
	Table    *TableContent    `json:"table,omitempty"`
	Citation *CitationContent `json:"citation,omitempty"`
}

// Occurrence is the unit a document consumer tracks. The same occurrence ID in a
// later read does not promise the same bytes: compare ContentDigest.
type Occurrence struct {
	OccurrenceID         string                `json:"occurrenceId"`
	Kind                 string                `json:"kind"`
	SubjectRefs          []sourceindex.Subject `json:"subjectRefs"`
	GenerationConditions GenerationConditions  `json:"generationConditions"`
	Content              OccurrenceContent     `json:"content"`
	ContentDigest        string                `json:"contentDigest"`
}

// Draft is the TEI to SHO document draft contract.
type Draft struct {
	SchemaVersion   string        `json:"schemaVersion"`
	Effect          string        `json:"effect"`
	DraftID         string        `json:"draftId"`
	DocumentKind    string        `json:"documentKind"`
	Title           string        `json:"title"`
	Model           ModelRef      `json:"model"`
	Subjects        []SubjectRef  `json:"subjects"`
	Relations       []RelationRef `json:"relations,omitempty"`
	Occurrences     []Occurrence  `json:"occurrences"`
	AuthorityLimits []string      `json:"authorityLimits"`
}

// Request is the input of Build.
type Request struct {
	Root         string
	Subject      sourceindex.Subject
	RelationID   string
	DocumentKind string
	Title        string
}

// DraftError is a typed draft failure. Code is stable so a CLI and a library
// caller report the same string.
type DraftError struct {
	Code    string
	Message string
}

func (e *DraftError) Error() string {
	if e == nil {
		return "document draft failure"
	}
	return e.Message
}

func draftError(code, format string, args ...any) *DraftError {
	return &DraftError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Error codes.
const (
	CodeInvalidArgument      = "INVALID_ARGUMENT"
	CodeWorkspaceLoadFailed  = "WORKSPACE_LOAD_FAILED"
	CodeSourceIndexFailed    = "SOURCE_INDEX_FAILED"
	CodeSubjectNotFound      = "SUBJECT_NOT_FOUND"
	CodeSubjectAmbiguous     = "SUBJECT_AMBIGUOUS"
	CodeUnsupportedKind      = "UNSUPPORTED_SUBJECT_KIND"
	CodeUnsupportedDocKind   = "UNSUPPORTED_DOCUMENT_KIND"
	CodeRelationNotFound     = "RELATION_NOT_FOUND"
	CodeRelationNotOfSubject = "RELATION_NOT_OF_SUBJECT"
	CodeInvalidDraft         = "INVALID_DRAFT"
)

// SupportedElementKinds lists the element kinds the first integration projects.
// It is exported so a caller learns the closed set instead of guessing.
func SupportedElementKinds() []string {
	return []string{"semantic-data", "semantic-type", "semantic-capability"}
}

// Build reads one Workspace and projects one element and, when requested, one
// declared relation of that element into a draft. It performs no write.
func Build(req Request) (Draft, error) {
	kind := strings.TrimSpace(req.DocumentKind)
	if kind == "" {
		kind = DocumentKindElementExplanation
	}
	if kind != DocumentKindElementExplanation {
		return Draft{}, draftError(CodeUnsupportedDocKind,
			"document kind %q is not supported by this integration; supported: %s",
			kind, DocumentKindElementExplanation)
	}
	if strings.TrimSpace(req.Root) == "" || req.Root != strings.TrimSpace(req.Root) {
		return Draft{}, draftError(CodeInvalidArgument, "workspace path must be nonempty and exact")
	}
	if err := validateSubject(req.Subject); err != nil {
		return Draft{}, err
	}
	relationID := strings.TrimSpace(req.RelationID)
	if relationID != "" && relationID != strings.TrimSpace(relationID) {
		return Draft{}, draftError(CodeInvalidArgument, "relation id must be an exact identity")
	}

	read, err := readWorkspace(req.Root, req.Subject)
	if err != nil {
		return Draft{}, err
	}
	inspection, err := read.inspect(req.Subject)
	if err != nil {
		return Draft{}, err
	}
	draft := Draft{
		SchemaVersion: SchemaVersion,
		Effect:        EffectReadOnlyProjection,
		DocumentKind:  kind,
		Model: ModelRef{
			Workspace:              inspection.Workspace,
			WorldFingerprint:       inspection.WorldFingerprint,
			SourceIndexFingerprint: inspection.SourceIndexFingerprint,
			ModelRevision:          read.modelRevision,
		},
		AuthorityLimits: DraftAuthorityLimits(),
	}
	if err := draft.project(read, req.Subject, relationID); err != nil {
		return Draft{}, err
	}
	draft.Title = buildTitle(req.Title, req.Subject)
	draft.DraftID = draftID(draft)
	return draft, nil
}

func validateSubject(subject sourceindex.Subject) error {
	if subject.Kind == "" || subject.ID == "" ||
		subject.Kind != strings.TrimSpace(subject.Kind) ||
		subject.ID != strings.TrimSpace(subject.ID) {
		return draftError(CodeInvalidArgument, "subject kind and id must be nonempty exact identities")
	}
	for _, kind := range SupportedElementKinds() {
		if subject.Kind == kind {
			return nil
		}
	}
	return draftError(CodeUnsupportedKind,
		"document draft does not project subject kind %q; supported: %s",
		subject.Kind, strings.Join(SupportedElementKinds(), ", "))
}

func buildTitle(requested string, subject sourceindex.Subject) string {
	if strings.TrimSpace(requested) != "" {
		return requested
	}
	return string(subject.ID) + " — 要素説明"
}

// workspaceRead is one Workspace load with everything a projection derives from
// it. A draft and its reconciliation read the Workspace once each, so the model
// reference and the occurrences can never come from two different snapshots.
type workspaceRead struct {
	loaded        workspace.Result
	index         sourceindex.Index
	modelRevision string
}

// inspect projects one subject from the SAME load, so an element and its
// relation target are answered from one snapshot instead of two reads.
func (r workspaceRead) inspect(subject sourceindex.Subject) (operations.SubjectInspection, error) {
	inspection, err := operations.InspectionFromLoad(r.loaded, r.index, subject)
	if err != nil {
		return operations.SubjectInspection{}, mapInspectionError(err)
	}
	return inspection, nil
}

func readWorkspace(root string, subject sourceindex.Subject) (workspaceRead, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return workspaceRead{}, &operations.ValidationFailure{Stage: operations.StageLoad, Err: err}
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return workspaceRead{}, &operations.ValidationFailure{Stage: operations.StageSourceIndex, Err: err}
	}
	read := workspaceRead{loaded: loaded, index: idx, modelRevision: sourceindex.ModelRevision(idx)}
	// Resolve the requested subject before anything else, so a missing element
	// is reported as SUBJECT_NOT_FOUND rather than a projection failure.
	if _, err := read.inspect(subject); err != nil {
		return workspaceRead{}, err
	}
	return read, nil
}

// mapInspectionError keeps the inspection's own codes, so a caller that asked
// for a subject that does not exist reads SUBJECT_NOT_FOUND rather than a
// generic failure.
func mapInspectionError(err error) error {
	var inspectionErr *operations.InspectionError
	if errors.As(err, &inspectionErr) {
		code := CodeInvalidArgument
		switch inspectionErr.Code {
		case operations.InspectionNotFound:
			code = CodeSubjectNotFound
		case operations.InspectionAmbiguous:
			code = CodeSubjectAmbiguous
		case operations.InspectionUnsupportedKind:
			code = CodeUnsupportedKind
		}
		return &DraftError{Code: code, Message: inspectionErr.Message}
	}
	return err
}

func inputDigests(inspection operations.SubjectInspection, read workspaceRead) map[string]string {
	return map[string]string{
		"worldFingerprint":       inspection.WorldFingerprint,
		"sourceIndexFingerprint": inspection.SourceIndexFingerprint,
		"modelRevision":          read.modelRevision,
	}
}

// project fills the subjects, relations and occurrences of one draft. The order
// is deterministic: explanation, relation, table, then one citation per source
// artifact.
func (d *Draft) project(read workspaceRead, subject sourceindex.Subject, relationID string) error {
	inspection, err := read.inspect(subject)
	if err != nil {
		return err
	}
	digests := inputDigests(inspection, read)
	numbering := &occurrenceNumbering{}
	d.Subjects = []SubjectRef{subjectRefOf(inspection, read, subject)}

	switch subject.Kind {
	case "semantic-data":
		var data semantic.Data
		if err := json.Unmarshal(inspection.Canonical, &data); err != nil {
			return draftError(CodeInvalidArgument,
				"canonical subject %s:%s is not a readable data declaration: %v",
				subject.Kind, subject.ID, err)
		}
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceExplanation,
			[]sourceindex.Subject{subject},
			conditions(digests, "element-summary-v0"),
			OccurrenceContent{Text: dataExplanation(subject, data)}))
		if relationID != "" {
			relation, target, err := resolveRelation(read, data, subject, relationID)
			if err != nil {
				return err
			}
			d.Relations = append(d.Relations, relation)
			d.Subjects = append(d.Subjects, target)
			d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceRelation,
				[]sourceindex.Subject{subject, relation.To},
				conditions(digests, "relation-summary-v0"),
				OccurrenceContent{Relation: &relation}))
		}
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceTable,
			[]sourceindex.Subject{subject},
			conditions(digests, "data-attributes-v0"),
			OccurrenceContent{Table: attributeTable(data)}))
	case "semantic-type":
		var typ semantic.Type
		if err := json.Unmarshal(inspection.Canonical, &typ); err != nil {
			return draftError(CodeInvalidArgument,
				"canonical subject %s:%s is not a readable type declaration: %v",
				subject.Kind, subject.ID, err)
		}
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceExplanation,
			[]sourceindex.Subject{subject},
			conditions(digests, "element-summary-v0"),
			OccurrenceContent{Text: typeExplanation(subject, typ)}))
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceTable,
			[]sourceindex.Subject{subject},
			conditions(digests, "type-shape-v0"),
			OccurrenceContent{Table: typeTable(typ)}))
	case "semantic-capability":
		var capability semantic.Capability
		if err := json.Unmarshal(inspection.Canonical, &capability); err != nil {
			return draftError(CodeInvalidArgument,
				"canonical subject %s:%s is not a readable capability declaration: %v",
				subject.Kind, subject.ID, err)
		}
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceExplanation,
			[]sourceindex.Subject{subject},
			conditions(digests, "element-summary-v0"),
			OccurrenceContent{Text: capabilityExplanation(subject, capability)}))
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceTable,
			[]sourceindex.Subject{subject},
			conditions(digests, "capability-shape-v0"),
			OccurrenceContent{Table: capabilityTable(capability)}))
	default:
		return draftError(CodeUnsupportedKind,
			"document draft does not project subject kind %q", subject.Kind)
	}

	for _, artifact := range inspection.Sources.Artifacts {
		d.Occurrences = append(d.Occurrences, numbering.add(OccurrenceCitation,
			[]sourceindex.Subject{subject},
			conditions(digests, "source-citation-v0"),
			OccurrenceContent{Citation: citationOf(inspection, artifact)}))
	}
	return nil
}

// resolveRelation reads ONE declared relation of the requested element. The
// relation must be declared by that element: a relation id that belongs to
// another data is refused instead of silently projected.
func resolveRelation(
	read workspaceRead, data semantic.Data, from sourceindex.Subject, relationID string,
) (RelationRef, SubjectRef, error) {
	for _, declared := range data.Relations {
		if string(declared.ID) != relationID {
			continue
		}
		target := sourceindex.Subject{Kind: "semantic-data", ID: string(declared.Target)}
		relation := RelationRef{
			Relation:    sourceindex.Subject{Kind: "semantic-relation", ID: relationID},
			From:        from,
			To:          target,
			Resolution:  resolutionOf(read, target),
			Cardinality: string(declared.Cardinality),
			OnDelete:    declared.OnDelete,
		}
		for _, key := range declared.Keys {
			relation.Keys = append(relation.Keys, RelationKeyBinding{
				SourceAttribute: string(key.SourceAttribute),
				TargetAttribute: string(key.TargetAttribute),
			})
		}
		targetInspection, err := read.inspect(target)
		if err != nil {
			// An unresolved or ambiguous target keeps its own status instead of
			// failing the whole draft: the relation itself is still declared.
			return relation, SubjectRef{Subject: target, Sources: []SourceBinding{}}, nil
		}
		return relation, subjectRefOf(targetInspection, read, target), nil
	}
	return RelationRef{}, SubjectRef{}, draftError(CodeRelationNotFound,
		"relation %q is not declared by %s:%s", relationID, from.Kind, from.ID)
}

// resolutionOf reports whether the relation target is declared exactly once in
// the same read. A name is never used to decide identity.
func resolutionOf(read workspaceRead, target sourceindex.Subject) string {
	count := 0
	for _, namespace := range read.loaded.World.Semantics {
		for _, data := range namespace.Data {
			if string(data.ID) == target.ID {
				count++
			}
		}
	}
	switch count {
	case 1:
		return ResolutionResolved
	case 0:
		return ResolutionUnresolved
	default:
		return ResolutionAmbiguous
	}
}

func subjectRefOf(inspection operations.SubjectInspection, read workspaceRead, subject sourceindex.Subject) SubjectRef {
	ref := SubjectRef{Subject: subject, Sources: []SourceBinding{}}
	for _, artifact := range inspection.Sources.Artifacts {
		ref.Sources = append(ref.Sources, SourceBinding{
			ArtifactPath: artifact.Path,
			Format:       artifact.Format,
			Fingerprint:  artifact.Fingerprint,
			Ranges:       rangesFor(inspection, subject, artifact.Path),
		})
	}
	ref.Symbol = symbolOf(inspection.Canonical)
	sort.SliceStable(ref.Sources, func(i, j int) bool {
		if ref.Sources[i].ArtifactPath != ref.Sources[j].ArtifactPath {
			return ref.Sources[i].ArtifactPath < ref.Sources[j].ArtifactPath
		}
		return ref.Sources[i].Fingerprint < ref.Sources[j].Fingerprint
	})
	return ref
}

func rangesFor(inspection operations.SubjectInspection, subject sourceindex.Subject, artifactPath string) []sourceindex.Range {
	var ranges []sourceindex.Range
	for _, reference := range inspection.Sources.References {
		if reference.Subject == subject && reference.ArtifactPath == artifactPath {
			ranges = append(ranges, reference.Range)
		}
	}
	return ranges
}

func symbolOf(canonical json.RawMessage) string {
	var shape struct {
		Symbol string `json:"symbol"`
	}
	if err := json.Unmarshal(canonical, &shape); err != nil {
		return ""
	}
	return shape.Symbol
}

// occurrenceNumbering hands out deterministic occurrence IDs. The ID is
// snapshot-local: it identifies a position in THIS draft, not a persistent
// document identity.
type occurrenceNumbering struct {
	counts map[string]int
}

func (n *occurrenceNumbering) add(
	kind string, subjects []sourceindex.Subject,
	conditions GenerationConditions, content OccurrenceContent,
) Occurrence {
	if n.counts == nil {
		n.counts = map[string]int{}
	}
	n.counts[kind]++
	return Occurrence{
		OccurrenceID:         fmt.Sprintf("occ-%s-%d", kind, n.counts[kind]),
		Kind:                 kind,
		SubjectRefs:          subjects,
		GenerationConditions: conditions,
		Content:              content,
		ContentDigest:        contentDigest(content),
	}
}

func conditions(digests map[string]string, rule string) GenerationConditions {
	copied := make(map[string]string, len(digests))
	for key, value := range digests {
		copied[key] = value
	}
	return GenerationConditions{
		Generator:        GeneratorName,
		GeneratorVersion: GeneratorVersion,
		Rule:             rule,
		InputDigests:     copied,
	}
}

// contentDigest is a stable digest of one occurrence's payload. Two reads that
// produce the same digest produced the same bytes for that occurrence.
func contentDigest(content OccurrenceContent) string {
	encoded, err := json.Marshal(content)
	if err != nil {
		return "sha256:unencodable"
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// draftID identifies this draft read. It binds the schema, the subject, the
// relation and the model reference; it is not a document identity and not an
// acceptance record.
func draftID(draft Draft) string {
	payload := struct {
		SchemaVersion    string `json:"schemaVersion"`
		DocumentKind     string `json:"documentKind"`
		SubjectKind      string `json:"subjectKind"`
		SubjectID        string `json:"subjectId"`
		RelationID       string `json:"relationId,omitempty"`
		WorldFingerprint string `json:"worldFingerprint"`
		ModelRevision    string `json:"modelRevision"`
	}{
		SchemaVersion:    draft.SchemaVersion,
		DocumentKind:     draft.DocumentKind,
		SubjectKind:      draft.Subjects[0].Subject.Kind,
		SubjectID:        draft.Subjects[0].Subject.ID,
		WorldFingerprint: draft.Model.WorldFingerprint,
		ModelRevision:    draft.Model.ModelRevision,
	}
	if len(draft.Relations) > 0 {
		payload.RelationID = draft.Relations[0].Relation.ID
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "sha256:unencodable"
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

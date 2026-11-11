package reacandidate

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// ArtifactRecord is one artifact of a read workspace: its normalized path and
// its CURRENT byte fingerprint. It identifies current bytes, not history.
type ArtifactRecord struct {
	Path        string `json:"path"`
	Fingerprint string `json:"fingerprint"`
}

// DeclarationRecord is one Canonical declaration found in a source range. A
// coincidence between an external occurrence and a Canonical declaration is a
// structural observation, never an acceptance.
type DeclarationRecord struct {
	Subject sourceindex.Subject `json:"subject"`
	Role    string              `json:"role,omitempty"`
	Range   sourceindex.Range   `json:"range"`
}

// SubjectOutcome is the result of resolving one Canonical subject through the
// existing read-only operation.
type SubjectOutcome struct {
	Status                 string `json:"status"`
	SourceStatus           string `json:"sourceStatus,omitempty"`
	WorldFingerprint       string `json:"worldFingerprint,omitempty"`
	SourceIndexFingerprint string `json:"sourceIndexFingerprint,omitempty"`
	Message                string `json:"message,omitempty"`
}

// ModelMapLink mirrors the existing typed navigation link. It is copied, not
// aliased, so this package never holds a live projection.
type ModelMapLink struct {
	FromKind   string `json:"fromKind"`
	FromID     string `json:"fromId"`
	ToKind     string `json:"toKind"`
	ToID       string `json:"toId"`
	Relation   string `json:"relation"`
	Resolution string `json:"resolution"`
}

// SourceIndexView is the read-only Source Index projection a binding needs.
type SourceIndexView struct {
	Artifacts    []ArtifactRecord               `json:"artifacts"`
	Declarations map[string][]DeclarationRecord `json:"declarations"`
}

// ReadOnlyModelPort is the narrow, read-only view of the existing model
// operations this package is allowed to use.
//
// The port exists so a binding can be exercised without a workspace, and so
// the read-only boundary is explicit: it exposes no write, no acceptance, no
// authorization, and no Canonical edit. The production implementation
// delegates to the existing operations and workspace loader only.
type ReadOnlyModelPort interface {
	Inspect(subject sourceindex.Subject) (SubjectOutcome, error)
	ModelLinks() ([]ModelMapLink, error)
	SourceIndex() (SourceIndexView, error)
	WorkspaceIdentity() (string, error)
}

// CandidateBinding is the read-only comparison of one candidate against an
// existing workspace.
type CandidateBinding struct {
	CandidateID             string               `json:"candidateId"`
	SourceBinding           string               `json:"sourceBinding"`
	SourceBindingNotes      []string             `json:"sourceBindingNotes,omitempty"`
	CoincidentDeclarations  []DeclarationRecord  `json:"coincidentDeclarations,omitempty"`
	MappedSubject           *sourceindex.Subject `json:"mappedSubject,omitempty"`
	InspectStatus           string               `json:"inspectStatus"`
	ExistingLinksForSubject []ModelMapLink       `json:"existingLinksForSubject,omitempty"`
	Resolution              string               `json:"resolution"`
	ResolutionNotes         []string             `json:"resolutionNotes,omitempty"`
}

// BindingResult is the complete read-only binding of a candidate set.
type BindingResult struct {
	SchemaVersion   string             `json:"schemaVersion"`
	Effect          string             `json:"effect"`
	Workspace       string             `json:"workspace,omitempty"`
	Bindings        []CandidateBinding `json:"bindings"`
	AuthorityLimits []string           `json:"authorityLimits"`
}

// BindingAuthorityLimits states what a binding does not establish.
func BindingAuthorityLimits() []string {
	return []string{
		"binding-is-a-read-only-comparison",
		"source-digest-match-identifies-current-bytes-not-history",
		"source-digest-conflict-is-not-a-candidate-failure-and-not-an-acceptance",
		"coincident-declaration-is-structural-not-meaning-acceptance",
		"resolved-requires-an-explicit-owner-mapping-and-one-canonical-subject",
		"binding-does-not-write-accept-or-authorize",
	}
}

// BindCandidates compares a candidate set against one read workspace.
//
// mappings is an OPTIONAL, caller-supplied map from candidate id to Canonical
// subject. It is the only way a candidate becomes resolved, and it is never
// inferred here: without an explicit mapping every candidate stays
// unresolved. A candidate whose source digest conflicts with the current
// workspace bytes is reported, never accepted or silently dropped.
func BindCandidates(port ReadOnlyModelPort, set CandidateSet, mappings map[string]sourceindex.Subject) (BindingResult, error) {
	if port == nil {
		return BindingResult{}, refuse("READ_ONLY_PORT_MISSING", "binding requires a read-only model port")
	}
	index, err := port.SourceIndex()
	if err != nil {
		return BindingResult{}, refuse("SOURCE_INDEX_READ_FAILED", "read the workspace source index: %v", err)
	}
	artifacts := map[string]string{}
	for _, artifact := range index.Artifacts {
		artifacts[artifact.Path] = artifact.Fingerprint
	}
	links, err := port.ModelLinks()
	if err != nil {
		return BindingResult{}, refuse("MODEL_LINKS_READ_FAILED", "read the existing model links: %v", err)
	}
	identity, err := port.WorkspaceIdentity()
	if err != nil {
		return BindingResult{}, refuse("WORKSPACE_IDENTITY_READ_FAILED", "read the workspace identity: %v", err)
	}

	result := BindingResult{
		SchemaVersion:   "tei.rea.candidate-binding/v0",
		Effect:          EffectReadOnly,
		Workspace:       identity,
		AuthorityLimits: BindingAuthorityLimits(),
	}
	for _, candidate := range set.Candidates {
		binding := CandidateBinding{
			CandidateID:        candidate.CandidateID,
			SourceBinding:      candidate.SourceRevision.Binding,
			SourceBindingNotes: append([]string(nil), candidate.SourceRevision.BindingNotes...),
			InspectStatus:      "not-attempted",
			Resolution:         ResolutionUnresolved,
		}
		bindSource(&binding, candidate, artifacts, index.Declarations)
		if mapped, ok := mappings[candidate.CandidateID]; ok {
			binding.MappedSubject = &mapped
			outcome, inspectErr := port.Inspect(mapped)
			switch {
			case inspectErr != nil:
				binding.InspectStatus = "error"
				binding.ResolutionNotes = append(binding.ResolutionNotes, "inspect-failed:"+inspectErr.Error())
			case outcome.Status == "found":
				binding.InspectStatus = outcome.Status
				binding.Resolution = ResolutionResolved
				binding.ExistingLinksForSubject = linksFor(links, mapped)
			case outcome.Status == "ambiguous":
				binding.InspectStatus = outcome.Status
				binding.Resolution = ResolutionAmbiguous
				binding.ResolutionNotes = append(binding.ResolutionNotes, "mapped-subject-is-ambiguous")
			default:
				binding.InspectStatus = outcome.Status
				binding.ResolutionNotes = append(binding.ResolutionNotes, "mapped-subject-not-found")
			}
		} else {
			binding.ResolutionNotes = append(binding.ResolutionNotes, "no-owner-mapping")
		}
		result.Bindings = append(result.Bindings, binding)
	}
	sort.Slice(result.Bindings, func(i, j int) bool {
		return result.Bindings[i].CandidateID < result.Bindings[j].CandidateID
	})
	return result, nil
}

// bindSource compares the candidate's provider-reported source against the
// workspace's current bytes and records any Canonical declaration that shares
// the range.
func bindSource(binding *CandidateBinding, candidate StructureCandidate, artifacts map[string]string, declarations map[string][]DeclarationRecord) {
	revision := candidate.SourceRevision
	if revision.ArtifactPath == "" {
		binding.SourceBinding = SourceBindingUnavailable
		binding.SourceBindingNotes = append(binding.SourceBindingNotes, "candidate-carries-no-artifact-path")
		return
	}
	current, present := artifacts[revision.ArtifactPath]
	switch {
	case !present:
		binding.SourceBinding = SourceBindingAbsent
		binding.SourceBindingNotes = append(binding.SourceBindingNotes, "artifact-path-is-not-in-the-read-workspace")
	case revision.Fingerprint == "":
		binding.SourceBinding = SourceBindingProviderReported
		binding.SourceBindingNotes = append(binding.SourceBindingNotes, "candidate-carries-no-artifact-digest")
	case current == revision.Fingerprint:
		binding.SourceBinding = SourceBindingIndependent
	default:
		binding.SourceBinding = SourceBindingConflict
		binding.SourceBindingNotes = append(binding.SourceBindingNotes,
			"current-workspace-bytes-differ-from-the-analyzed-bytes")
	}
	if span := revision.Span; span != nil {
		for _, declaration := range declarations[revision.ArtifactPath] {
			if rangeContains(declaration.Range, span) {
				binding.CoincidentDeclarations = append(binding.CoincidentDeclarations, declaration)
			}
		}
		sort.Slice(binding.CoincidentDeclarations, func(i, j int) bool {
			a, b := binding.CoincidentDeclarations[i], binding.CoincidentDeclarations[j]
			if a.Subject.Kind != b.Subject.Kind {
				return a.Subject.Kind < b.Subject.Kind
			}
			return a.Subject.ID < b.Subject.ID
		})
		if len(binding.CoincidentDeclarations) > 0 {
			binding.SourceBindingNotes = append(binding.SourceBindingNotes,
				"coincident-declaration-is-structural-not-acceptance")
		}
	}
}

// rangeContains reports whether a declared Canonical range covers the
// candidate span. When offsets are unknown on either side the comparison
// falls back to line/column, which can only under-report, never over-report.
func rangeContains(declared sourceindex.Range, span *Span) bool {
	start := span.Start
	end := span.Start
	if span.End != nil {
		end = *span.End
	}
	if declared.Start.Offset > 0 && start.Offset > 0 && declared.End.Offset > 0 && end.Offset > 0 {
		return declared.Start.Offset <= start.Offset && end.Offset <= declared.End.Offset
	}
	return pointAtOrAfter(start.Line, start.Column, declared.Start.Line, declared.Start.Column) &&
		pointAtOrAfter(declared.End.Line, declared.End.Column, end.Line, end.Column)
}

// pointAtOrAfter reports whether (leftLine,leftColumn) is at or after
// (rightLine,rightColumn) in document order.
func pointAtOrAfter(leftLine, leftColumn, rightLine, rightColumn int) bool {
	if leftLine != rightLine {
		return leftLine >= rightLine
	}
	return leftColumn >= rightColumn
}

func linksFor(links []ModelMapLink, subject sourceindex.Subject) []ModelMapLink {
	var out []ModelMapLink
	for _, link := range links {
		if (link.FromKind == subject.Kind && link.FromID == subject.ID) ||
			(link.ToKind == subject.Kind && link.ToID == subject.ID) {
			out = append(out, link)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		for _, pair := range [][2]string{
			{a.FromKind, b.FromKind}, {a.FromID, b.FromID}, {a.Relation, b.Relation},
			{a.ToKind, b.ToKind}, {a.ToID, b.ToID}, {a.Resolution, b.Resolution},
		} {
			if pair[0] != pair[1] {
				return pair[0] < pair[1]
			}
		}
		return false
	})
	return out
}

// WorkspaceReadOnlyModel is the production ReadOnlyModelPort. It reads one
// workspace through the existing operations and workspace loader and never
// writes to it.
type WorkspaceReadOnlyModel struct {
	root string

	once     sync.Once
	index    SourceIndexView
	indexErr error
}

// NewWorkspaceReadOnlyModel validates the workspace path exactly, in the same
// shape the existing read-only operations require.
func NewWorkspaceReadOnlyModel(root string) (*WorkspaceReadOnlyModel, error) {
	if strings.TrimSpace(root) == "" || root != strings.TrimSpace(root) {
		return nil, refuse("INVALID_WORKSPACE", "workspace path must be nonempty and exact")
	}
	return &WorkspaceReadOnlyModel{root: root}, nil
}

func (m *WorkspaceReadOnlyModel) Inspect(subject sourceindex.Subject) (SubjectOutcome, error) {
	inspection, err := operations.InspectSubject(m.root, subject)
	if err != nil {
		var inspectionErr *operations.InspectionError
		if errors.As(err, &inspectionErr) {
			return SubjectOutcome{Status: inspectionErr.Code, Message: inspectionErr.Message}, nil
		}
		return SubjectOutcome{}, err
	}
	return SubjectOutcome{
		Status:                 "found",
		SourceStatus:           inspection.SourceStatus,
		WorldFingerprint:       inspection.WorldFingerprint,
		SourceIndexFingerprint: inspection.SourceIndexFingerprint,
	}, nil
}

func (m *WorkspaceReadOnlyModel) ModelLinks() ([]ModelMapLink, error) {
	mapped, err := operations.MapWorkspace(m.root)
	if err != nil {
		return nil, err
	}
	out := make([]ModelMapLink, 0, len(mapped.Links))
	for _, link := range mapped.Links {
		out = append(out, ModelMapLink{
			FromKind:   link.From.Kind,
			FromID:     link.From.ID,
			ToKind:     link.To.Kind,
			ToID:       link.To.ID,
			Relation:   link.Relation,
			Resolution: link.Resolution,
		})
	}
	return out, nil
}

func (m *WorkspaceReadOnlyModel) WorkspaceIdentity() (string, error) {
	loaded, err := workspace.Load(m.root)
	if err != nil {
		return "", err
	}
	identity := loaded.Identity()
	name := identity.Name
	if name == "" {
		name = "(unnamed-workspace)"
	}
	return name + " formatVersion=" + strconv.Itoa(identity.FormatVersion) +
		" manifest=" + identity.Manifest.Path + "@" + identity.Manifest.SHA256, nil
}

// SourceIndex loads the workspace once and projects its artifacts and
// declarations. Loading is read-only; nothing here writes the workspace.
func (m *WorkspaceReadOnlyModel) SourceIndex() (SourceIndexView, error) {
	m.once.Do(func() {
		loaded, err := workspace.Load(m.root)
		if err != nil {
			m.indexErr = err
			return
		}
		index, err := workspace.BuildSourceIndex(m.root, loaded)
		if err != nil {
			m.indexErr = err
			return
		}
		view := SourceIndexView{
			Artifacts:    make([]ArtifactRecord, 0, len(index.Artifacts)),
			Declarations: map[string][]DeclarationRecord{},
		}
		for _, artifact := range index.Artifacts {
			view.Artifacts = append(view.Artifacts, ArtifactRecord{
				Path:        artifact.Path,
				Fingerprint: artifact.Fingerprint,
			})
		}
		for _, reference := range index.References {
			path := reference.ArtifactPath
			view.Declarations[path] = append(view.Declarations[path], DeclarationRecord{
				Subject: reference.Subject,
				Role:    string(reference.Role),
				Range:   reference.Range,
			})
		}
		sort.Slice(view.Artifacts, func(i, j int) bool {
			return view.Artifacts[i].Path < view.Artifacts[j].Path
		})
		for path := range view.Declarations {
			records := view.Declarations[path]
			sort.Slice(records, func(i, j int) bool {
				a, b := records[i], records[j]
				if a.Subject.Kind != b.Subject.Kind {
					return a.Subject.Kind < b.Subject.Kind
				}
				if a.Subject.ID != b.Subject.ID {
					return a.Subject.ID < b.Subject.ID
				}
				return a.Range.Start.Offset < b.Range.Start.Offset
			})
			view.Declarations[path] = records
		}
		m.index = view
	})
	if m.indexErr != nil {
		return SourceIndexView{}, fmt.Errorf("load workspace source index: %w", m.indexErr)
	}
	return m.index, nil
}

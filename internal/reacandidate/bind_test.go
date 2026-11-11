package reacandidate

import (
	"testing"

	"tei/internal/sourceindex"
)

// fakeReadOnlyModel is a test double for the read-only port. It has no write
// method at all, which is the point: a binding cannot write through it.
type fakeReadOnlyModel struct {
	inspect func(sourceindex.Subject) (SubjectOutcome, error)
	links   []ModelMapLink
	index   SourceIndexView
	failOn  string
}

func (f fakeReadOnlyModel) Inspect(subject sourceindex.Subject) (SubjectOutcome, error) {
	if f.inspect == nil {
		return SubjectOutcome{Status: "not-found"}, nil
	}
	return f.inspect(subject)
}

func (f fakeReadOnlyModel) ModelLinks() ([]ModelMapLink, error) {
	if f.failOn == "links" {
		return nil, refuse("MODEL_LINKS_READ_FAILED", "synthetic link failure")
	}
	return f.links, nil
}

func (f fakeReadOnlyModel) SourceIndex() (SourceIndexView, error) {
	if f.failOn == "index" {
		return SourceIndexView{}, refuse("SOURCE_INDEX_READ_FAILED", "synthetic index failure")
	}
	return f.index, nil
}

func (f fakeReadOnlyModel) WorkspaceIdentity() (string, error) {
	if f.failOn == "identity" {
		return "", refuse("WORKSPACE_IDENTITY_READ_FAILED", "synthetic identity failure")
	}
	return "probe formatVersion=1 manifest=manifest.json@sha256:" + digestA, nil
}

func boundFakeIndex() SourceIndexView {
	return SourceIndexView{
		Artifacts: []ArtifactRecord{
			{Path: "src/order.js", Fingerprint: "sha256:" + digestA},
			{Path: "src/line.js", Fingerprint: "sha256:" + digestB},
		},
		Declarations: map[string][]DeclarationRecord{
			"src/order.js": {{
				Subject: sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"},
				Role:    "definition",
				Range: sourceindex.Range{
					Start: sourceindex.Position{Offset: 0, Line: 1, Column: 1},
					End:   sourceindex.Position{Offset: 900, Line: 20, Column: 1},
				},
			}},
		},
	}
}

// Without an explicit Owner mapping nothing resolves, and the binding says so
// instead of guessing a Canonical subject.
func TestBindCandidatesWithoutMappingResolvesNothing(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	result, err := BindCandidates(fakeReadOnlyModel{index: boundFakeIndex()}, set, nil)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if len(result.Bindings) != len(set.Candidates) {
		t.Fatalf("bindings = %d, candidates = %d", len(result.Bindings), len(set.Candidates))
	}
	for _, binding := range result.Bindings {
		if binding.Resolution != ResolutionUnresolved {
			t.Fatalf("binding %s resolved without a mapping: %q", binding.CandidateID, binding.Resolution)
		}
		if !hasString(binding.ResolutionNotes, "no-owner-mapping") {
			t.Fatalf("binding %s does not record the missing mapping: %v", binding.CandidateID, binding.ResolutionNotes)
		}
		if binding.InspectStatus != "not-attempted" {
			t.Fatalf("binding %s attempted an inspect: %q", binding.CandidateID, binding.InspectStatus)
		}
	}
}

// Matching current bytes bind independently; differing bytes are reported as
// a conflict, which is never an acceptance and never a silent drop.
func TestBindCandidatesSourceDigestComparison(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	result, err := BindCandidates(fakeReadOnlyModel{index: boundFakeIndex()}, set, nil)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	byCandidate := map[string]CandidateBinding{}
	for _, binding := range result.Bindings {
		byCandidate[binding.CandidateID] = binding
	}
	for _, candidate := range set.Candidates {
		binding := byCandidate[candidate.CandidateID]
		want := SourceBindingAbsent
		switch candidate.SourceRevision.ArtifactPath {
		case "src/order.js":
			want = SourceBindingIndependent
		case "src/line.js":
			want = SourceBindingConflict
		}
		if binding.SourceBinding != want {
			t.Fatalf("candidate %s binding = %q, want %q", candidate.CandidateID, binding.SourceBinding, want)
		}
	}
	// The order.js data candidate's span (lines 1-5) sits inside the declared
	// semantic-data range, so the coincidence is reported as an observation.
	coincidences := 0
	for _, candidate := range set.Candidates {
		if candidate.SourceRevision.ArtifactPath != "src/order.js" {
			continue
		}
		binding := byCandidate[candidate.CandidateID]
		if candidate.SourceRevision.Span == nil {
			continue
		}
		coincidences++
		if len(binding.CoincidentDeclarations) != 1 {
			t.Fatalf("coincident declarations = %+v", binding.CoincidentDeclarations)
		}
		if binding.CoincidentDeclarations[0].Subject.ID != "sales.order" {
			t.Fatalf("coincident subject = %+v", binding.CoincidentDeclarations[0].Subject)
		}
		if !hasString(binding.SourceBindingNotes, "coincident-declaration-is-structural-not-acceptance") {
			t.Fatalf("coincidence is not marked structural: %v", binding.SourceBindingNotes)
		}
		if binding.Resolution != ResolutionUnresolved {
			t.Fatalf("coincidence was promoted to a resolution: %q", binding.Resolution)
		}
	}
	if coincidences == 0 {
		t.Fatal("no candidate carried a span, so containment was not exercised")
	}
}

// An explicit Owner mapping is the only route to a resolved candidate, and it
// still goes through the existing read-only operation.
func TestBindCandidatesOwnerMappingResolvesThroughReadOnlyOperation(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	var dataCandidate string
	for _, candidate := range set.Candidates {
		if candidate.Kind == CandidateKindDataElement && candidate.Facts.ExternalIdentity == "node-order" {
			dataCandidate = candidate.CandidateID
		}
	}
	if dataCandidate == "" {
		t.Fatal("no node-order data candidate")
	}
	model := fakeReadOnlyModel{
		index: boundFakeIndex(),
		inspect: func(sourceindex.Subject) (SubjectOutcome, error) {
			return SubjectOutcome{Status: "found", SourceStatus: "indexed"}, nil
		},
		links: []ModelMapLink{{
			FromKind: "semantic-relation", FromID: "sales.order.lines",
			ToKind: "semantic-data", ToID: "sales.order", Relation: "refers-to", Resolution: "resolved",
		}},
	}
	mapping := map[string]sourceindex.Subject{
		dataCandidate: {Kind: "semantic-data", ID: "sales.order"},
	}
	result, err := BindCandidates(model, set, mapping)
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	var bound CandidateBinding
	for _, binding := range result.Bindings {
		if binding.CandidateID == dataCandidate {
			bound = binding
		}
	}
	if bound.Resolution != ResolutionResolved {
		t.Fatalf("mapped candidate resolution = %q", bound.Resolution)
	}
	if bound.MappedSubject == nil || bound.MappedSubject.ID != "sales.order" {
		t.Fatalf("mapped subject = %+v", bound.MappedSubject)
	}
	if len(bound.ExistingLinksForSubject) != 1 {
		t.Fatalf("existing links = %+v", bound.ExistingLinksForSubject)
	}
}

// A mapping to an absent or ambiguous Canonical subject stays unresolved or
// ambiguous; it is never treated as a resolution.
func TestBindCandidatesMappingToMissingOrAmbiguousSubject(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	mappedID := set.Candidates[0].CandidateID
	bindingFor := func(result BindingResult) CandidateBinding {
		for _, binding := range result.Bindings {
			if binding.CandidateID == mappedID {
				return binding
			}
		}
		t.Fatalf("mapped candidate %s is missing from the bindings", mappedID)
		return CandidateBinding{}
	}
	missing := fakeReadOnlyModel{
		index: boundFakeIndex(),
		inspect: func(sourceindex.Subject) (SubjectOutcome, error) {
			return SubjectOutcome{Status: "not-found"}, nil
		},
	}
	result, err := BindCandidates(missing, set, map[string]sourceindex.Subject{
		mappedID: {Kind: "semantic-data", ID: "sales.absent"},
	})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if bindingFor(result).Resolution != ResolutionUnresolved {
		t.Fatalf("missing subject resolution = %q", bindingFor(result).Resolution)
	}
	ambiguous := fakeReadOnlyModel{
		index: boundFakeIndex(),
		inspect: func(sourceindex.Subject) (SubjectOutcome, error) {
			return SubjectOutcome{Status: "ambiguous"}, nil
		},
	}
	result, err = BindCandidates(ambiguous, set, map[string]sourceindex.Subject{
		mappedID: {Kind: "semantic-data", ID: "sales.duplicated"},
	})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if bindingFor(result).Resolution != ResolutionAmbiguous {
		t.Fatalf("ambiguous subject resolution = %q", bindingFor(result).Resolution)
	}
	if !hasString(bindingFor(result).ResolutionNotes, "mapped-subject-is-ambiguous") {
		t.Fatalf("ambiguity note = %v", bindingFor(result).ResolutionNotes)
	}
}

// A read failure is a failure, not an empty binding.
func TestBindCandidatesReadFailuresAreFailures(t *testing.T) {
	set := mustGenerate(t, defaultEvidence())
	for _, failOn := range []string{"index", "links", "identity"} {
		_, err := BindCandidates(fakeReadOnlyModel{failOn: failOn}, set, nil)
		if err == nil {
			t.Fatalf("%s read failure was swallowed", failOn)
		}
	}
	if _, err := BindCandidates(nil, set, nil); err == nil {
		t.Fatal("a missing port was accepted")
	}
}

func TestBindCandidatesRequiresNonEmptyExactWorkspacePath(t *testing.T) {
	for _, root := range []string{"", " reference/workspace", "reference/workspace "} {
		if _, err := NewWorkspaceReadOnlyModel(root); err == nil {
			t.Fatalf("workspace path %q accepted", root)
		}
	}
}

package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type actorContextR3RSemanticSubjectDependencyProbe struct {
	Subject            sourceindex.Subject
	ContentFingerprint string
	Locator            string
	Role               sourceindex.Role
}

type actorContextR3RReferenceArtifactDependencyProbe struct {
	InputClass          string
	ContentFingerprint string
	Locator             string
}

type actorContextR3RRuntimeObservationDependencyProbe struct {
	BusinessAddress string
	SubjectID       string
	Action          string
	Resource        string
	Outcome         string
}

type actorContextR3RDependencyEnvelopeProbe struct {
	ProfileFingerprint string
	SemanticSubjects   []actorContextR3RSemanticSubjectDependencyProbe
	ReferenceArtifacts []actorContextR3RReferenceArtifactDependencyProbe
	RuntimeObservations []actorContextR3RRuntimeObservationDependencyProbe
}

type actorContextR3RFreshnessDeltaProbe struct {
	ProfileChanged          bool
	SemanticMeaningStale    bool
	ReferenceContentStale   bool
	RuntimeObservationStale bool
	LocatorRefreshRequired  bool
}

func TestActorContextR3RBuildsActualSourceIndexSubjectDependencyForSubmitPattern(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}

	dependency, err := actorContextR3RSemanticDependencyFromSourceIndex(
		idx,
		sourceindex.Subject{
			Kind: "application-pattern",
			ID:   "order_entry.submit_order",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if dependency.Subject.Kind != "application-pattern" ||
		dependency.Subject.ID != "order_entry.submit_order" {
		t.Fatalf("R3R lost semantic subject identity: %+v", dependency)
	}
	if dependency.ContentFingerprint == "" ||
		dependency.Locator == "" ||
		dependency.Role != sourceindex.RoleDefinition {
		t.Fatalf("R3R Source Index dependency is incomplete: %+v", dependency)
	}
	if dependency.Locator != "applications/order-entry/patterns/submit-order.tei" {
		t.Fatalf("test setup changed: unexpected submit_order locator %q", dependency.Locator)
	}
}

func TestActorContextR3RLocatorOnlyMoveRequiresLocatorRefreshNotSemanticStale(t *testing.T) {
	before := actorContextR3RSemanticSubjectDependencyProbe{
		Subject: sourceindex.Subject{
			Kind: "application-pattern",
			ID:   "order_entry.submit_order",
		},
		ContentFingerprint: "sha256:same-content",
		Locator:            "applications/order-entry/patterns/submit-order.tei",
		Role:               sourceindex.RoleDefinition,
	}
	after := before
	after.Locator = "applications/order-entry/interactions/submit-order.tei"

	delta := actorContextR3RCompareEnvelopes(
		actorContextR3RDependencyEnvelopeProbe{SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{before}},
		actorContextR3RDependencyEnvelopeProbe{SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{after}},
	)

	if delta.SemanticMeaningStale {
		t.Fatalf("locator-only move was treated as semantic stale: %+v", delta)
	}
	if !delta.LocatorRefreshRequired {
		t.Fatalf("locator-only move did not require drill-down locator refresh: %+v", delta)
	}
}

func TestActorContextR3RSameSubjectChangedContentIsSemanticStaleEvenAtSameLocator(t *testing.T) {
	before := actorContextR3RSemanticSubjectDependencyProbe{
		Subject: sourceindex.Subject{
			Kind: "application-pattern",
			ID:   "order_entry.submit_order",
		},
		ContentFingerprint: "sha256:before",
		Locator:            "applications/order-entry/patterns/submit-order.tei",
		Role:               sourceindex.RoleDefinition,
	}
	after := before
	after.ContentFingerprint = "sha256:after"

	delta := actorContextR3RCompareEnvelopes(
		actorContextR3RDependencyEnvelopeProbe{SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{before}},
		actorContextR3RDependencyEnvelopeProbe{SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{after}},
	)

	if !delta.SemanticMeaningStale {
		t.Fatalf("changed subject content did not stale semantic dependency: %+v", delta)
	}
	if delta.LocatorRefreshRequired {
		t.Fatalf("same locator was incorrectly treated as a locator move: %+v", delta)
	}
}

func TestActorContextR3RReferenceArtifactsStaySeparateFromSourceIndexSubjects(t *testing.T) {
	root := repoPath()

	businessPath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	overlayPath := filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json")

	businessBytes, err := osReadFileForR3R(businessPath)
	if err != nil {
		t.Fatal(err)
	}
	overlayBytes, err := osReadFileForR3R(overlayPath)
	if err != nil {
		t.Fatal(err)
	}

	deps := []actorContextR3RReferenceArtifactDependencyProbe{
		{
			InputClass:          "business-procedure-reference",
			ContentFingerprint: sourceindex.Fingerprint(businessBytes),
			Locator:             "reference/business-system-design/order-entry-business-procedure.json",
		},
		{
			InputClass:          "protected-target-overlay",
			ContentFingerprint: sourceindex.Fingerprint(overlayBytes),
			Locator:             "reference/business-system-design/order-entry-protected-target-overlay.json",
		},
	}

	for _, dep := range deps {
		if dep.InputClass == "" || dep.ContentFingerprint == "" || dep.Locator == "" {
			t.Fatalf("R3R Reference artifact dependency is incomplete: %+v", dep)
		}
	}

	typ := reflect.TypeOf(actorContextR3RReferenceArtifactDependencyProbe{})
	for _, forbidden := range []string{
		"Subject",
		"SubjectKind",
		"SubjectID",
		"CanonicalID",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3R invented Source Index / Canonical identity for Reference artifact via %q", forbidden)
		}
	}
}

func TestActorContextR3RRuntimeObservationChangesRuntimePlaneOnly(t *testing.T) {
	static := actorContextR3RSemanticSubjectDependencyProbe{
		Subject: sourceindex.Subject{
			Kind: "application-pattern",
			ID:   "order_entry.submit_order",
		},
		ContentFingerprint: "sha256:static",
		Locator:            "applications/order-entry/patterns/submit-order.tei",
		Role:               sourceindex.RoleDefinition,
	}
	allowed := actorContextR3RRuntimeObservationDependencyProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		SubjectID:       "subject-order-viewer",
		Action:          "execute",
		Resource:        "order_entry.submit_order",
		Outcome:         "allowed",
	}
	denied := allowed
	denied.Outcome = "denied"

	before := actorContextR3RDependencyEnvelopeProbe{
		ProfileFingerprint: "profile:agent-resolution",
		SemanticSubjects:   []actorContextR3RSemanticSubjectDependencyProbe{static},
		RuntimeObservations: []actorContextR3RRuntimeObservationDependencyProbe{allowed},
	}
	after := before
	after.RuntimeObservations = []actorContextR3RRuntimeObservationDependencyProbe{denied}

	delta := actorContextR3RCompareEnvelopes(before, after)
	if !delta.RuntimeObservationStale {
		t.Fatalf("runtime Observation change did not stale runtime plane: %+v", delta)
	}
	if delta.SemanticMeaningStale ||
		delta.ReferenceContentStale ||
		delta.ProfileChanged ||
		delta.LocatorRefreshRequired {
		t.Fatalf("runtime-only change leaked into static/profile freshness planes: %+v", delta)
	}
}

func TestActorContextR3RStaticSourceEqualityDoesNotProveRuntimeCurrent(t *testing.T) {
	static := actorContextR3RSemanticSubjectDependencyProbe{
		Subject: sourceindex.Subject{
			Kind: "application-pattern",
			ID:   "order_entry.submit_order",
		},
		ContentFingerprint: "sha256:static",
		Locator:            "applications/order-entry/patterns/submit-order.tei",
		Role:               sourceindex.RoleDefinition,
	}

	before := actorContextR3RDependencyEnvelopeProbe{
		SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{static},
	}
	after := actorContextR3RDependencyEnvelopeProbe{
		SemanticSubjects: []actorContextR3RSemanticSubjectDependencyProbe{static},
		RuntimeObservations: []actorContextR3RRuntimeObservationDependencyProbe{
			{
				BusinessAddress: "order-entry.order-handling/BP-06",
				SubjectID:       "subject-order-viewer",
				Action:          "execute",
				Resource:        "order_entry.submit_order",
				Outcome:         "denied",
			},
		},
	}

	delta := actorContextR3RCompareEnvelopes(before, after)
	if delta.SemanticMeaningStale {
		t.Fatalf("unchanged static Source Index dependency became semantic stale: %+v", delta)
	}
	if !delta.RuntimeObservationStale {
		t.Fatalf("same static source incorrectly proved runtime Observation freshness: %+v", delta)
	}
}

func TestActorContextR3RProfileIdentityRemainsSeparateFreshnessPlane(t *testing.T) {
	before := actorContextR3RDependencyEnvelopeProbe{
		ProfileFingerprint: "profile:human-orientation-v1",
	}
	after := before
	after.ProfileFingerprint = "profile:human-orientation-v2"

	delta := actorContextR3RCompareEnvelopes(before, after)
	if !delta.ProfileChanged {
		t.Fatalf("projection profile change was not preserved as its own freshness plane: %+v", delta)
	}
	if delta.SemanticMeaningStale ||
		delta.ReferenceContentStale ||
		delta.RuntimeObservationStale ||
		delta.LocatorRefreshRequired {
		t.Fatalf("profile-only change leaked into other freshness planes: %+v", delta)
	}
}

func TestActorContextR3RDependencyEnvelopeHasNoGlobalFreshnessShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3RDependencyEnvelopeProbe{})
	for _, forbidden := range []string{
		"CombinedFingerprint",
		"GlobalFingerprint",
		"GlobalRevision",
		"RepositoryRevision",
		"Current",
		"Fresh",
		"Valid",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3R dependency envelope introduced forbidden global freshness shortcut %q", forbidden)
		}
	}
}

func actorContextR3RSemanticDependencyFromSourceIndex(
	idx sourceindex.Index,
	subject sourceindex.Subject,
) (actorContextR3RSemanticSubjectDependencyProbe, error) {
	refs := idx.ReferencesFor(subject)
	var selected *sourceindex.Reference
	for i := range refs {
		if refs[i].Role != sourceindex.RoleDefinition {
			continue
		}
		if selected != nil {
			return actorContextR3RSemanticSubjectDependencyProbe{}, fmt.Errorf("R3R subject %s:%s has multiple definition refs", subject.Kind, subject.ID)
		}
		selected = &refs[i]
	}
	if selected == nil {
		return actorContextR3RSemanticSubjectDependencyProbe{}, fmt.Errorf("R3R subject %s:%s has no definition ref", subject.Kind, subject.ID)
	}

	var artifact *sourceindex.Artifact
	for i := range idx.Artifacts {
		if idx.Artifacts[i].Path != selected.ArtifactPath {
			continue
		}
		artifact = &idx.Artifacts[i]
		break
	}
	if artifact == nil {
		return actorContextR3RSemanticSubjectDependencyProbe{}, fmt.Errorf("R3R source artifact %q is not indexed", selected.ArtifactPath)
	}

	return actorContextR3RSemanticSubjectDependencyProbe{
		Subject:            selected.Subject,
		ContentFingerprint: artifact.Fingerprint,
		Locator:            selected.ArtifactPath,
		Role:               selected.Role,
	}, nil
}

func actorContextR3RCompareEnvelopes(
	before actorContextR3RDependencyEnvelopeProbe,
	after actorContextR3RDependencyEnvelopeProbe,
) actorContextR3RFreshnessDeltaProbe {
	return actorContextR3RFreshnessDeltaProbe{
		ProfileChanged:          before.ProfileFingerprint != after.ProfileFingerprint,
		SemanticMeaningStale:    actorContextR3RSemanticDependenciesChanged(before.SemanticSubjects, after.SemanticSubjects),
		ReferenceContentStale:   actorContextR3RReferenceContentChanged(before.ReferenceArtifacts, after.ReferenceArtifacts),
		RuntimeObservationStale: !reflect.DeepEqual(before.RuntimeObservations, after.RuntimeObservations),
		LocatorRefreshRequired:  actorContextR3RLocatorsChanged(before.SemanticSubjects, after.SemanticSubjects) ||
			actorContextR3RReferenceLocatorsChanged(before.ReferenceArtifacts, after.ReferenceArtifacts),
	}
}

func actorContextR3RSemanticDependenciesChanged(
	before []actorContextR3RSemanticSubjectDependencyProbe,
	after []actorContextR3RSemanticSubjectDependencyProbe,
) bool {
	if len(before) != len(after) {
		return true
	}
	bySubject := map[sourceindex.Subject]actorContextR3RSemanticSubjectDependencyProbe{}
	for _, dep := range before {
		bySubject[dep.Subject] = dep
	}
	for _, dep := range after {
		prior, ok := bySubject[dep.Subject]
		if !ok ||
			prior.ContentFingerprint != dep.ContentFingerprint ||
			prior.Role != dep.Role {
			return true
		}
	}
	return false
}

func actorContextR3RLocatorsChanged(
	before []actorContextR3RSemanticSubjectDependencyProbe,
	after []actorContextR3RSemanticSubjectDependencyProbe,
) bool {
	bySubject := map[sourceindex.Subject]string{}
	for _, dep := range before {
		bySubject[dep.Subject] = dep.Locator
	}
	for _, dep := range after {
		if prior, ok := bySubject[dep.Subject]; ok && prior != dep.Locator {
			return true
		}
	}
	return false
}

func actorContextR3RReferenceContentChanged(
	before []actorContextR3RReferenceArtifactDependencyProbe,
	after []actorContextR3RReferenceArtifactDependencyProbe,
) bool {
	if len(before) != len(after) {
		return true
	}
	byClass := map[string]actorContextR3RReferenceArtifactDependencyProbe{}
	for _, dep := range before {
		byClass[dep.InputClass] = dep
	}
	for _, dep := range after {
		prior, ok := byClass[dep.InputClass]
		if !ok || prior.ContentFingerprint != dep.ContentFingerprint {
			return true
		}
	}
	return false
}

func actorContextR3RReferenceLocatorsChanged(
	before []actorContextR3RReferenceArtifactDependencyProbe,
	after []actorContextR3RReferenceArtifactDependencyProbe,
) bool {
	byClass := map[string]string{}
	for _, dep := range before {
		byClass[dep.InputClass] = dep.Locator
	}
	for _, dep := range after {
		if prior, ok := byClass[dep.InputClass]; ok && prior != dep.Locator {
			return true
		}
	}
	return false
}

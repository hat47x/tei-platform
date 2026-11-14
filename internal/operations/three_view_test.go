package operations

import (
	"testing"

	"tei/internal/workspace"
)

const threeViewTestDigest = "sha256:1111111111111111111111111111111111111111111111111111111111111111"

// threeViewTestValue is a smallest complete answer, written by hand: one data
// subject, one function subject and the function's declaration over the data.
func threeViewTestValue() (ThreeViewRequest, ThreeViewNavigation) {
	limits := []string{"limit"}
	subject := func(occurrence, view, kind, id string) ThreeViewSubject {
		return ThreeViewSubject{
			Occurrence: occurrence, View: view,
			SourceFact: ThreeViewSourceFact{Kind: kind, Locator: "/" + kind, SourceID: id, SourceIDStatus: ThreeViewSourceIDUnique},
			Candidate:  ThreeViewCandidate{Nature: ThreeViewNatureStructuralCandidate, AuthorityLimits: limits},
			Meaning:    ThreeViewMeaning{Status: ThreeViewMeaningNotBound},
		}
	}
	return ThreeViewRequest{SourcePath: "design/a.src"}, ThreeViewNavigation{
		SchemaVersion: "tei.operations.three-view-navigation/v0",
		Effect:        "read-only",
		Workspace:     &workspace.WorkspaceIdentity{Manifest: workspace.ManifestIdentity{Path: "tei.workspace.yaml", SHA256: "m"}},
		Source: ThreeViewSource{
			Path: "design/a.src", Format: "src", ArtifactDigest: threeViewTestDigest,
			MappingFingerprint: "map", DerivationRevision: "rev", ProjectionFingerprint: "fp",
			Coverage: ThreeViewCoverage{Scope: "single-source-artifact", Status: ThreeViewCoverageComplete},
		},
		Subjects: []ThreeViewSubject{
			subject("d", ThreeViewData, "Table", "T1"),
			subject("f", ThreeViewFunction, "Function", "F1"),
		},
		Fields: []ThreeViewField{{
			Occurrence: "x", OwnerOccurrence: "d",
			SourceFact: ThreeViewSourceFact{Kind: "Field", Locator: "/Field", SourceIDStatus: ThreeViewSourceIDMissing},
		}},
		Relations: []ThreeViewRelation{{
			Occurrence: "r", Kind: "function-data-operation", Nature: ThreeViewNatureSourceFact,
			From: "f", To: "d", Operations: "R", SourceOccurrences: []string{"f", "d"}, AuthorityLimits: limits,
		}},
		CRUD: []ThreeViewCRUDRow{{
			Function: "f", Data: "d", Read: true, Scope: ThreeViewCRUDScopeFunctionDeclared, Relations: []string{"r"},
		}},
		FieldReferences: []ThreeViewFieldReference{{
			Occurrence: "k", Kind: "key-member", Nature: ThreeViewNatureSourceFact,
			Owner: "d", Field: "x", Resolution: "resolved", AuthorityLimits: limits,
		}},
		AuthorityLimits: limits,
	}
}

func TestValidateThreeViewNavigationAcceptsACompleteAnswer(t *testing.T) {
	request, value := threeViewTestValue()
	if err := ValidateThreeViewNavigation(request, value); err != nil {
		t.Fatal(err)
	}
	if ThreeViewScopeSingleSource != "single-source-artifact" ||
		ThreeViewRelationFunctionDataOperation != "function-data-operation" ||
		ThreeViewSourceIDUnique != "unique" {
		t.Fatal("a contract word changed without a version change")
	}
}

func TestValidateThreeViewNavigationRejectsInconsistentAnswers(t *testing.T) {
	cases := map[string]func(*ThreeViewRequest, *ThreeViewNavigation){
		"another source":      func(r *ThreeViewRequest, _ *ThreeViewNavigation) { r.SourcePath = "design/b.src" },
		"written effect":      func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Effect = "write" },
		"no workspace":        func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Workspace = nil },
		"no source digest":    func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Source.ArtifactDigest = "T1" },
		"no authority limits": func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.AuthorityLimits = nil },
		"accepted meaning":    func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Subjects[0].Meaning.Status = "accepted" },
		"candidate as fact": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Subjects[0].Candidate.Nature = ThreeViewNatureSourceFact
		},
		"fourth view":         func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Subjects[0].View = "screen" },
		"repeated occurrence": func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Subjects[1].Occurrence = "d" },
		"missing ID with a ID": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Subjects[0].SourceFact.SourceIDStatus = ThreeViewSourceIDMissing
		},
		"duplicated without ID": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Fields[0].SourceFact.SourceIDStatus = ThreeViewSourceIDDuplicated
		},
		"relation to a non-subject": func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Relations[0].To = "x" },
		"relation without evidence": func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Relations[0].SourceOccurrences = nil },
		"derived without its rule": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Relations[0].Nature = ThreeViewNatureStructuralCandidate
		},
		"CRUD on a derived relation": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Relations[0].Nature = ThreeViewNatureStructuralCandidate
			v.Relations[0].DerivationRuleFingerprint = "rule"
		},
		"CRUD of another pair":     func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.CRUD[0].Function, v.CRUD[0].Data = "d", "f" },
		"CRUD outside its scope":   func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.CRUD[0].Scope = "business-action" },
		"CRUD without a relation":  func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.CRUD[0].Relations = nil },
		"field of a function":      func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Fields[0].OwnerOccurrence = "f" },
		"reference to a non-field": func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.FieldReferences[0].Field = "f" },
		"complete with unplaced":   func(_ *ThreeViewRequest, v *ThreeViewNavigation) { v.Source.Coverage.UnplacedConstructs = 1 },
		"partial with nothing left": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Source.Coverage.Status = ThreeViewCoveragePartial
		},
		"unresolved not counted": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Unresolved = []ThreeViewUnresolved{{Code: "c", RelationKind: "k", SourceOccurrences: []string{"f"}}}
		},
		"invented pin": func(_ *ThreeViewRequest, v *ThreeViewNavigation) {
			v.Pin = &ThreeViewPinState{ThreeViewPin: ThreeViewPin{Occurrence: "f", SourceArtifactDigest: threeViewTestDigest}, Status: ThreeViewPinHeld}
		},
		"dropped pin": func(r *ThreeViewRequest, _ *ThreeViewNavigation) {
			r.Pin = &ThreeViewPin{Occurrence: "f", SourceArtifactDigest: threeViewTestDigest}
		},
	}
	for name, mutate := range cases {
		request, value := threeViewTestValue()
		mutate(&request, &value)
		if err := ValidateThreeViewNavigation(request, value); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

// The pin must come back as the caller sent it, with the status the answer's
// own source binding and subjects imply. It cannot be moved to another subject.
func TestValidateThreeViewNavigationHoldsThePinToTheCallersValue(t *testing.T) {
	otherDigest := "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	cases := []struct {
		name     string
		pin      ThreeViewPin
		returned ThreeViewPinState
		ok       bool
	}{
		{"held", ThreeViewPin{"f", threeViewTestDigest},
			ThreeViewPinState{ThreeViewPin{"f", threeViewTestDigest}, ThreeViewPinHeld}, true},
		{"stale after a source change", ThreeViewPin{"f", otherDigest},
			ThreeViewPinState{ThreeViewPin{"f", otherDigest}, ThreeViewPinStaleSourceChanged}, true},
		{"not found", ThreeViewPin{"gone", threeViewTestDigest},
			ThreeViewPinState{ThreeViewPin{"gone", threeViewTestDigest}, ThreeViewPinOccurrenceNotFound}, true},
		{"moved to another subject", ThreeViewPin{"gone", threeViewTestDigest},
			ThreeViewPinState{ThreeViewPin{"f", threeViewTestDigest}, ThreeViewPinHeld}, false},
		{"stale reported as held", ThreeViewPin{"f", otherDigest},
			ThreeViewPinState{ThreeViewPin{"f", otherDigest}, ThreeViewPinHeld}, false},
		{"rebased onto the new source", ThreeViewPin{"f", otherDigest},
			ThreeViewPinState{ThreeViewPin{"f", threeViewTestDigest}, ThreeViewPinHeld}, false},
		{"a field is not a pinned subject", ThreeViewPin{"x", threeViewTestDigest},
			ThreeViewPinState{ThreeViewPin{"x", threeViewTestDigest}, ThreeViewPinHeld}, false},
	}
	for _, c := range cases {
		request, value := threeViewTestValue()
		pin := c.pin
		request.Pin = &pin
		returned := c.returned
		value.Pin = &returned
		if err := ValidateThreeViewNavigation(request, value); (err == nil) != c.ok {
			t.Errorf("%s: err = %v, want ok = %v", c.name, err, c.ok)
		}
	}
}

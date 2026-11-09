package webcomposition

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"tei/internal/application"
	"tei/internal/operations"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// This is a bounded admission witness against the *actual* reference
// Workspace. It does not prove E1->Go runtime selection, complete Graph
// closure, Provider invocation, a current Grant, or a business effect.
func TestE1GoPairRealWorkspaceSourceAdmission(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate checked-out reference Workspace")
	}
	root := filepath.Join(filepath.Dir(here), "..", "..", "workspace")
	fixtureBytes, err := os.ReadFile(filepath.Join(filepath.Dir(here), "..", "e1-go-route-pair.fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	var pair routePairFixture
	if err := json.Unmarshal(fixtureBytes, &pair); err != nil {
		t.Fatal(err)
	}
	if !routePairMatches(pair) {
		t.Fatal("the two independently authored Route/Action declarations do not match")
	}

	patternSubject := sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.submit_order"}
	capabilitySubject := sourceindex.Subject{Kind: "semantic-capability", ID: "sales.order.submit"}
	patternResult, err := operations.InspectSubject(root, patternSubject)
	if err != nil {
		t.Fatalf("real canonical Pattern not inspectable: %v", err)
	}
	capabilityResult, err := operations.InspectSubject(root, capabilitySubject)
	if err != nil {
		t.Fatalf("real canonical Capability not inspectable: %v", err)
	}
	for _, r := range []struct {
		name    string
		subject sourceindex.Subject
		read    operations.SubjectInspection
	}{
		{"pattern", patternSubject, patternResult},
		{"capability", capabilitySubject, capabilityResult},
	} {
		if r.read.Subject != r.subject || r.read.Effect != "read-only" ||
			r.read.SourceStatus != "indexed" ||
			r.read.WorldFingerprint == "" || r.read.SourceIndexFingerprint == "" ||
			len(r.read.Sources.References) == 0 || len(r.read.Sources.Artifacts) == 0 {
			t.Fatalf("%s source admission is incomplete: %+v", r.name, r.read)
		}
		for _, source := range r.read.Sources.References {
			if source.Subject != r.subject {
				t.Fatalf("%s returned a wrong-subject Source reference: %+v", r.name, source)
			}
		}
	}
	if patternResult.WorldFingerprint != capabilityResult.WorldFingerprint ||
		patternResult.SourceIndexFingerprint != capabilityResult.SourceIndexFingerprint {
		t.Fatal("Pattern and Capability were not observed in the same reference Workspace view")
	}
	var pattern application.Pattern
	var capability semantic.Capability
	if err := json.Unmarshal(patternResult.Canonical, &pattern); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(capabilityResult.Canonical, &capability); err != nil {
		t.Fatal(err)
	}
	// Fixed expected identities come from independently authored reference
	// declarations, not from the current E1 or Go Host's returned values.
	if pattern.ID != "order_entry.submit_order" || pattern.Kind != application.PatternCommand ||
		capability.ID != "sales.order.submit" || pattern.Capability != capability.ID ||
		string(pattern.ID) != pair.E1.Route.ActionID ||
		string(pattern.ID) != pair.GoHost.ActionID {
		t.Fatalf("real Source -> Pattern -> Capability does not match declared Action: pattern=%+v capability=%+v", pattern, capability)
	}

	t.Run("HDG-S02-both-routes-match-but-canonical-command-is-absent", func(t *testing.T) {
		mutated := pair
		mutated.E1.Route.ActionID = "order_entry.unknown_action"
		mutated.GoHost.ActionID = "order_entry.unknown_action"
		if !routePairMatches(mutated) {
			t.Fatal("negative control should keep route pairing identical")
		}
		_, err := operations.InspectSubject(root, sourceindex.Subject{
			Kind: "application-pattern", ID: mutated.E1.Route.ActionID,
		})
		var inspection *operations.InspectionError
		if !errors.As(err, &inspection) || inspection.Code != operations.InspectionNotFound {
			t.Fatalf("matched declarations must not invent canonical identity, got %v", err)
		}
	})

	t.Run("HDG-S03-matching-pattern-does-not-imply-Provider-invocation", func(t *testing.T) {
		if string(pattern.ID) != pair.GoHost.ActionID || pattern.Capability != capability.ID {
			t.Fatal("the bounded canonical subject admission regressed")
		}
		// No Provider selection, binding, runtime effect or response Client
		// receipt is called here. Those qualifications remain unknown.
	})
}

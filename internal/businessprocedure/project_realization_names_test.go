package businessprocedure

import (
	"strings"
	"testing"
)

// Three realization words are a closed set. "unknown realization" alone leaves
// an author who guessed `application-use` to guess again, so the refusal quotes
// the same list Valid() checks.
func TestUnknownRealizationNamesTheAcceptedWords(t *testing.T) {
	projection := Project(Fixture{
		Procedure: Procedure{ID: "p", Trigger: Trigger{Status: StatusCandidate}, Actor: Actor{Status: StatusCandidate}},
		Steps: []Step{{
			ID:          "L-01",
			Action:      "do it",
			Realization: "application-use",
		}},
	}, "sha256:fixed", NewResolver(nil))
	var found string
	for _, d := range projection.Diagnostics {
		if d.Code == CodeRealizationUnknown {
			found = d.Message
		}
	}
	if found == "" {
		t.Fatalf("an unknown realization was accepted: %+v", projection.Diagnostics)
	}
	for _, want := range []string{"manual", "application", "system"} {
		if !strings.Contains(found, want) {
			t.Errorf("the refusal omits %q: %s", want, found)
		}
	}
	if !strings.Contains(found, "valid values are") {
		t.Errorf("the refusal does not say it is a closed set: %s", found)
	}
	if !strings.Contains(found, "application-use") {
		t.Errorf("the refusal does not name what was written: %s", found)
	}
}

// The list quoted in the message is the list Valid() accepts, so they cannot
// drift apart.
func TestRealizationNamesMatchTheAcceptedSet(t *testing.T) {
	names := strings.Split(realizationNames(), ", ")
	if len(names) != len(realizations) {
		t.Fatalf("realizationNames lists %d of %d", len(names), len(realizations))
	}
	for _, name := range names {
		if !Realization(name).Valid() {
			t.Errorf("realizationNames quotes %q but Valid() rejects it", name)
		}
	}
	if Realization("application-use").Valid() {
		t.Error("application-use is accepted")
	}
}

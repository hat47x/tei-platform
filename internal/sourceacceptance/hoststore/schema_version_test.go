package hoststore

import "testing"

// The store stamps its file with a schema version. The value is a wire contract:
// a different version in the file is refused (store.go compares it on every
// read), so the literal is asserted here as well as used there (#4032).
func TestStoreSchemaVersionIsFixed(t *testing.T) {
	if schemaVersion != "tei.host-acceptance-store/v0" {
		t.Fatalf("schemaVersion=%q", schemaVersion)
	}
}

// A stored Decision is in one of three states, and a reader asks "which one is
// current" by comparing against this value. A state that no test names is a
// transition nothing proves can be produced (#4032).
func TestDecisionStatesAreCovered(t *testing.T) {
	states := []string{StateCurrent, StateRevoked, StateSuperseded}
	want := []string{"current", "revoked", "superseded"}
	if len(states) != len(want) {
		t.Fatalf("a decision state was dropped: %d values", len(states))
	}
	seen := map[string]bool{}
	for i, state := range states {
		if state != want[i] {
			t.Fatalf("states[%d]=%q want %q", i, state, want[i])
		}
		if seen[state] {
			t.Fatalf("decision state %q is declared twice", state)
		}
		seen[state] = true
	}
}

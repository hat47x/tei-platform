package provider

import "testing"

// Capability is what a database provider claims it can do. A planner selects a
// provider by this set, so a value that no test names is a capability nothing
// shows to be real (#4032).
func TestCapabilitiesAreCovered(t *testing.T) {
	capabilities := []Capability{
		CapabilityMetadataRead, CapabilityQuery, CapabilityDataMutation, CapabilitySchemaChange,
		CapabilityExplain, CapabilityExplainAnalyze, CapabilityTransaction, CapabilityCancel,
	}
	want := []string{
		"metadata-read", "query", "data-mutation", "schema-change",
		"explain", "explain-analyze", "transaction", "cancel",
	}
	if len(capabilities) != len(want) {
		t.Fatalf("a capability was dropped: %d values", len(capabilities))
	}
	seen := map[string]bool{}
	for i, capability := range capabilities {
		if string(capability) != want[i] {
			t.Fatalf("capabilities[%d]=%q want %q", i, capability, want[i])
		}
		if seen[string(capability)] {
			t.Fatalf("capability %q is declared twice", capability)
		}
		seen[string(capability)] = true
	}
}

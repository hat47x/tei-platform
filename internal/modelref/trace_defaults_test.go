package modelref_test

import (
	"testing"

	"tei/internal/modelref"
)

// TestTraceDefaultsAreTheAdvertisedValues pins the single source for the values
// an adapter applies when a flag is omitted. Before this the CLI flag default,
// `tei commands` argumentDefaults and the MCP adapter each held a copy, and only
// the internal normalization disagreed ("full" rather than "summary") (#3605).
func TestTraceDefaultsAreTheAdvertisedValues(t *testing.T) {
	if modelref.TraceDefaultDirection != "both" {
		t.Fatalf("default direction = %q", modelref.TraceDefaultDirection)
	}
	if modelref.TraceDefaultMode != "summary" {
		t.Fatalf("default mode = %q", modelref.TraceDefaultMode)
	}
	// Both values must be members of the closed sets the operation validates,
	// so an advertised default can never be rejected as an invalid value.
	found := map[string]bool{}
	for _, value := range modelref.TraceDirections {
		found[value] = true
	}
	if !found[modelref.TraceDefaultDirection] {
		t.Fatalf("default direction %q is not in %v", modelref.TraceDefaultDirection, modelref.TraceDirections)
	}
	found = map[string]bool{}
	for _, value := range modelref.TraceModes {
		found[value] = true
	}
	if !found[modelref.TraceDefaultMode] {
		t.Fatalf("default mode %q is not in %v", modelref.TraceDefaultMode, modelref.TraceModes)
	}
}

// The omitted-flag path and the explicit-value path must produce the same trace
// for the default values, so the advertised default is not a different request.
func TestTraceDefaultModeEqualsAnExplicitSummary(t *testing.T) {
	g, rules, fn := traceGraph()
	implicit, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{
		Direction: modelref.TraceDefaultDirection,
		Mode:      modelref.TraceDefaultMode,
	})
	if err != nil {
		t.Fatal(err)
	}
	explicit, err := modelref.TraceDirect(g, rules, fn, modelref.TraceOptions{Direction: "both", Mode: "summary"})
	if err != nil {
		t.Fatal(err)
	}
	if implicit.Mode != explicit.Mode || implicit.Total != explicit.Total ||
		len(implicit.Rows) != len(explicit.Rows) || implicit.SnapshotFingerprint != explicit.SnapshotFingerprint {
		t.Fatalf("default and explicit summary differ:\n%+v\n%+v", implicit, explicit)
	}
}

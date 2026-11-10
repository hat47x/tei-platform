package plugincoverage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/workspace"
	plugincoverage "tei/tools/reference-plugin-capability-coverage"
)

// This is intentionally a historical record of an older selection policy.
// It is not the current Resolver or a Verified Composition BOM witness.
func TestHistoricalCapabilityCoverageMustNotBeReusedAsCurrent(t *testing.T) {
	path := filepath.Join("..", "..", "plan", "evidence", "PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08", "raw", "reference-coverage.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var old plugincoverage.Report
	if err := json.Unmarshal(data, &old); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	current := plugincoverage.Derive(loaded.World)
	if old.Schema != current.Schema || old.RuleVersion != current.RuleVersion {
		t.Fatalf("the historical fixture changed schema/version: old=%s/%s current=%s/%s", old.Schema, old.RuleVersion, current.Schema, current.RuleVersion)
	}
	if old.RuleFingerprint == "" || current.RuleFingerprint == "" || old.RuleFingerprint == current.RuleFingerprint {
		t.Fatalf("historical selection policy was incorrectly treated as current: old=%s current=%s", old.RuleFingerprint, current.RuleFingerprint)
	}
	if old.Authority != "derived-read-only" || current.Authority != "derived-read-only" {
		t.Fatalf("capability selection is not a trusted runtime witness: old=%s current=%s", old.Authority, current.Authority)
	}
	// Compare the same logical Kind + ID + Plugin + Version. The previously
	// unselected database capability is now resolved by instance technology.
	previous := map[string]plugincoverage.Capability{}
	for _, c := range old.Capabilities {
		previous[c.Plugin+"|"+c.Version+"|"+c.Kind+"|"+c.ID] = c
	}
	changedDatabaseCount := 0
	for _, c := range current.Capabilities {
		k := c.Plugin + "|" + c.Version + "|" + c.Kind + "|" + c.ID
		p, ok := previous[k]
		if !ok || c.Kind != "database-provider" {
			continue
		}
		if p.Status == "unselected" && c.Status == "selected" {
			changedDatabaseCount++
		}
	}
	if changedDatabaseCount < 1 {
		t.Fatalf("expected actual policy drift in database-provider selection: old-unselected=%d current-unselected=%d", old.Unselected, current.Unselected)
	}
	// This test MUST NOT set qualified=true, because both reports only read
	// declarations and environment binding; neither invokes a Provider.
	t.Logf("historical=%s, current=%s, database providers newly selected=%d (no runtime qualification)", old.RuleFingerprint, current.RuleFingerprint, changedDatabaseCount)
}

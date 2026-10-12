package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldRequirementEntryDiagnosticsAreDistinctFromDuplicate(t *testing.T) {
	tests := []struct {
		name    string
		req     plugin.Requirement
		bundle  bool
		code    string
	}{
		{name: "plugin invalid id", req: plugin.Requirement{Plugin: core.CanonicalID("bad id"), CompatibleMajor: 1}, code: "INVALID_PLUGIN_REQUIREMENT"},
		{name: "plugin zero major", req: plugin.Requirement{Plugin: cid("tei.dep"), CompatibleMajor: 0}, code: "INVALID_PLUGIN_REQUIREMENT"},
		{name: "bundle invalid id", req: plugin.Requirement{Plugin: core.CanonicalID("bad id"), CompatibleMajor: 1}, bundle: true, code: "INVALID_BUNDLE_PLUGIN_REQUIREMENT"},
		{name: "bundle negative major", req: plugin.Requirement{Plugin: cid("tei.dep"), CompatibleMajor: -1}, bundle: true, code: "INVALID_BUNDLE_PLUGIN_REQUIREMENT"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := referenceWorld()
			if tc.bundle {
				w.Bundles = []plugin.Bundle{{ID: cid("tei.bundle"), Version: "1.0.0", Plugins: []plugin.Requirement{tc.req}}}
			} else {
				w.Plugins[0].Requires = []plugin.Requirement{tc.req}
			}
			found := false
			for _, d := range validation.Validate(w) {
				if d.Code == tc.code && d.Severity == validation.Error {
					found = true
				}
				if d.Code == "DUPLICATE_PLUGIN_REQUIREMENT" || d.Code == "DUPLICATE_BUNDLE_PLUGIN_REQUIREMENT" {
					t.Fatalf("invalid entry must not be mislabeled as duplicate: %s", d.Code)
				}
			}
			if !found {
				t.Fatalf("world lacks expected %s diagnostic", tc.code)
			}
		})
	}
}

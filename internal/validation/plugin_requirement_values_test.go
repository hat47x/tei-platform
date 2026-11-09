package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldReportsInvalidRequirementValuesSeparatelyFromDuplicates(t *testing.T) {
	cases := []plugin.Requirement{
		{Plugin: core.CanonicalID("bad id"), CompatibleMajor: 1},
		{Plugin: cid("tei.base"), CompatibleMajor: 0},
		{Plugin: cid("tei.base"), CompatibleMajor: -1},
	}
	for _, req := range cases {
		world := ir.World{
			Plugins: []plugin.Definition{{ID: cid("tei.root"), Version: "1.0.0", Requires: []plugin.Requirement{req}}},
			Bundles: []plugin.Bundle{{ID: cid("tei.bundle"), Version: "1.0.0", Plugins: []plugin.Requirement{req}}},
		}
		diagnostics := validation.Validate(world)
		seen := make(map[string]bool)
		for _, diagnostic := range diagnostics {
			seen[diagnostic.Code] = true
		}
		if !seen["INVALID_PLUGIN_REQUIREMENT"] || !seen["INVALID_BUNDLE_PLUGIN_REQUIREMENT"] {
			t.Fatalf("missing invalid-value diagnostics for %+v: %+v", req, diagnostics)
		}
		if seen["DUPLICATE_PLUGIN_REQUIREMENT"] || seen["DUPLICATE_BUNDLE_PLUGIN_REQUIREMENT"] {
			t.Fatalf("incorrectly reported duplicate for %+v: %+v", req, diagnostics)
		}
	}
}

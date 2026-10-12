package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/validation"
)

func TestEnvironmentSelectionWorldReportsIDAndVersionIndependently(t *testing.T) {
	for _, tc := range []struct {
		name     string
		kind     string
		req      environment.ExtensionRequirement
		wantCode string
	}{
		{"plugin malformed ID", "plugin", environment.ExtensionRequirement{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}, "INVALID_ENVIRONMENT_PLUGIN_ID"},
		{"plugin padded ID", "plugin", environment.ExtensionRequirement{ID: core.CanonicalID(" tei.base "), CompatibleMajor: 1}, "INVALID_ENVIRONMENT_PLUGIN_ID"},
		{"bundle empty ID", "bundle", environment.ExtensionRequirement{ID: "", CompatibleMajor: 1}, "INVALID_ENVIRONMENT_BUNDLE_ID"},
		{"bundle malformed ID", "bundle", environment.ExtensionRequirement{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}, "INVALID_ENVIRONMENT_BUNDLE_ID"},
		{"plugin zero major", "plugin", environment.ExtensionRequirement{ID: cid("tei.base"), CompatibleMajor: 0}, "INVALID_ENVIRONMENT_PLUGIN_VERSION"},
		{"bundle negative major", "bundle", environment.ExtensionRequirement{ID: cid("tei.bundle"), CompatibleMajor: -1}, "INVALID_ENVIRONMENT_BUNDLE_VERSION"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := environment.Environment{ID: cid("prod")}
			if tc.kind == "bundle" {
				env.Bundles = []environment.ExtensionRequirement{tc.req}
			} else {
				env.Plugins = []environment.ExtensionRequirement{tc.req}
			}
			diagnostics := validation.Validate(ir.World{Environments: []environment.Environment{env}})
			hasCode := false
			for _, d := range diagnostics {
				if d.Code == tc.wantCode {
					hasCode = true
				}
				if d.Code == "DUPLICATE_ENVIRONMENT_PLUGIN_REQUIREMENT" || d.Code == "DUPLICATE_ENVIRONMENT_BUNDLE_REQUIREMENT" {
					t.Fatalf("invalid selector must not be a duplicate: %s", d.Code)
				}
			}
			if !hasCode {
				t.Fatalf("no expected diagnostic %q for %v: %+v", tc.wantCode, tc.req, diagnostics)
			}
		})
	}
}

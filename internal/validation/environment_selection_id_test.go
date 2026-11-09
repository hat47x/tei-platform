package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/validation"
)

func TestWorldEnvironmentSelectorIDDiagnostics(t *testing.T) {
	cases := []struct {
		name string
		env  environment.Environment
		code string
	}{
		{"invalid plugin id", environment.Environment{Plugins: []environment.ExtensionRequirement{{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}}}, "INVALID_ENVIRONMENT_PLUGIN_ID"},
		{"empty plugin id", environment.Environment{Plugins: []environment.ExtensionRequirement{{ID: "", CompatibleMajor: 1}}}, "INVALID_ENVIRONMENT_PLUGIN_ID"},
		{"padded bundle id", environment.Environment{Bundles: []environment.ExtensionRequirement{{ID: core.CanonicalID(" tei.standard "), CompatibleMajor: 1}}}, "INVALID_ENVIRONMENT_BUNDLE_ID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.env.ID = cid("prod")
			diagnostics := validation.Validate(ir.World{Environments: []environment.Environment{tc.env}})
			found := false
			for _, d := range diagnostics {
				if d.Code == tc.code && d.Severity == validation.Error {
					found = true
				}
			}
			if !found {
				t.Fatalf("World did not report %s for %+v: %+v", tc.code, tc.env, diagnostics)
			}
		})
	}
}

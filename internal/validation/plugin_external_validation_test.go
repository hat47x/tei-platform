package validation_test

import (
	"testing"

	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldValidationSharesExternalRequirementRules(t *testing.T) {
	valid := plugin.ExternalRequirement{Scope: "build", Kind: "toolchain", Name: "go"}
	for _, tc := range []struct {
		name  string
		items []plugin.ExternalRequirement
		code  string
	}{
		{name: "valid", items: []plugin.ExternalRequirement{valid}},
		{name: "duplicate", items: []plugin.ExternalRequirement{valid, valid}, code: "DUPLICATE_PLUGIN_EXTERNAL_REQUIREMENT"},
		{name: "invalid scope", items: []plugin.ExternalRequirement{{Scope: "other", Kind: "toolchain", Name: "go"}}, code: "INVALID_PLUGIN_EXTERNAL_SCOPE"},
		{name: "invalid kind", items: []plugin.ExternalRequirement{{Scope: "build", Kind: "other", Name: "go"}}, code: "INVALID_PLUGIN_EXTERNAL_KIND"},
		{name: "blank name", items: []plugin.ExternalRequirement{{Scope: "build", Kind: "toolchain", Name: " \t "}}, code: "PLUGIN_EXTERNAL_NAME_REQUIRED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := referenceWorld()
			world.Plugins[0].External = tc.items
			found := false
			for _, d := range validation.Validate(world) {
				if d.Code == tc.code && d.Severity == validation.Error {
					found = true
				}
				if tc.code == "" && (d.Code == "INVALID_PLUGIN_EXTERNAL_SCOPE" ||
					d.Code == "INVALID_PLUGIN_EXTERNAL_KIND" ||
					d.Code == "PLUGIN_EXTERNAL_NAME_REQUIRED" ||
					d.Code == "DUPLICATE_PLUGIN_EXTERNAL_REQUIREMENT") {
					t.Fatalf("valid external requirement rejected: %s", d.Code)
				}
			}
			if tc.code != "" && !found {
				t.Fatalf("missing expected diagnostic %s", tc.code)
			}
		})
	}
}

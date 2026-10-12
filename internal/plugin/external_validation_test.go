package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/plugin"
)

func TestValidateExternalRequirements(t *testing.T) {
	valid := plugin.ExternalRequirement{Scope: "build", Kind: "toolchain", Name: "go"}
	tests := []struct {
		name string
		items []plugin.ExternalRequirement
		reject string
	}{
		{name: "no requirements"},
		{name: "one valid requirement", items: []plugin.ExternalRequirement{valid}},
		{name: "same name with different phases", items: []plugin.ExternalRequirement{valid, {Scope: "runtime", Kind: "toolchain", Name: "go"}}},
		{name: "same name with different kinds", items: []plugin.ExternalRequirement{valid, {Scope: "build", Kind: "tool", Name: "go"}}},
		{name: "duplicate tuple", items: []plugin.ExternalRequirement{valid, valid}, reject: "more than once"},
		{name: "unsupported scope", items: []plugin.ExternalRequirement{{Scope: "unknown", Kind: "toolchain", Name: "go"}}, reject: "invalid scope"},
		{name: "unsupported kind", items: []plugin.ExternalRequirement{{Scope: "build", Kind: "unknown", Name: "go"}}, reject: "invalid kind"},
		{name: "missing name", items: []plugin.ExternalRequirement{{Scope: "build", Kind: "toolchain", Name: ""}}, reject: "name is required"},
		{name: "whitespace name", items: []plugin.ExternalRequirement{{Scope: "build", Kind: "toolchain", Name: "  \t"}}, reject: "name is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := plugin.ValidateExternalRequirements(tc.items)
			if tc.reject == "" && err != nil {
				t.Fatalf("valid requirements rejected: %v", err)
			}
			if tc.reject != "" && (err == nil || !strings.Contains(err.Error(), tc.reject)) {
				t.Fatalf("expected error %q, got %v", tc.reject, err)
			}
			source := artifactFreePackageSource()
			source.External = tc.items
			_, err = plugin.MaterializePluginPackage(source, pluginPackagePlan(source))
			if tc.reject == "" && err != nil {
				t.Fatalf("standalone package rejected valid external requirements: %v", err)
			}
			if tc.reject != "" && (err == nil || !strings.Contains(err.Error(), tc.reject)) {
				t.Fatalf("standalone package expected error %q, got %v", tc.reject, err)
			}
		})
	}
}

func TestExternalDuplicateDiagnosticIsPermutationInvariant(t *testing.T) {
	a := plugin.ExternalRequirement{Scope: "build", Kind: "tool", Name: "a"}
	z := plugin.ExternalRequirement{Scope: "runtime", Kind: "tool", Name: "z"}
	first := []plugin.ExternalRequirement{z, a, z, a}
	second := []plugin.ExternalRequirement{a, z, a, z}
	e1 := plugin.ValidateUniqueExternalRequirements(first)
	e2 := plugin.ValidateUniqueExternalRequirements(second)
	if e1 == nil || e2 == nil || e1.Error() != e2.Error() || !strings.Contains(e1.Error(), "/\"a\"") {
		t.Fatalf("unexpected permutation-dependent error: %v / %v", e1, e2)
	}
}

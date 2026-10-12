package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestRequirementEntryValidationInStandalonePackageAndResolver(t *testing.T) {
	tests := []struct {
		name        string
		requirement plugin.Requirement
		reject      string
	}{
		{name: "zero compatible major", requirement: plugin.Requirement{Plugin: cid("tei.base"), CompatibleMajor: 0}, reject: "compatible major"},
		{name: "negative compatible major", requirement: plugin.Requirement{Plugin: cid("tei.base"), CompatibleMajor: -1}, reject: "compatible major"},
		{name: "empty plugin id", requirement: plugin.Requirement{Plugin: "", CompatibleMajor: 1}, reject: "canonical id"},
		{name: "invalid plugin id", requirement: plugin.Requirement{Plugin: core.CanonicalID("not a valid id"), CompatibleMajor: 1}, reject: "canonical id"},
		{name: "surrounding spaces", requirement: plugin.Requirement{Plugin: core.CanonicalID(" tei.base "), CompatibleMajor: 1}, reject: "canonical id"},
		{name: "valid", requirement: plugin.Requirement{Plugin: cid("tei.base"), CompatibleMajor: 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := artifactFreePackageSource()
			source.Requires = []plugin.Requirement{tc.requirement}
			_, err := plugin.MaterializePluginPackage(source, pluginPackagePlan(source))
			if tc.reject == "" && err != nil {
				t.Fatalf("standalone package rejected valid requirement: %v", err)
			}
			if tc.reject != "" && (err == nil || !strings.Contains(err.Error(), tc.reject)) {
				t.Fatalf("standalone package expected %q: %v", tc.reject, err)
			}

			catalog := plugin.Catalog{Plugins: []plugin.Definition{
				{ID: cid("tei.root"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{tc.requirement}},
				{ID: cid("tei.base"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
			}}
			env := environment.Environment{ID: cid("prod"), Plugins: []environment.ExtensionRequirement{{ID: cid("tei.root"), CompatibleMajor: 1}}}
			_, err = catalog.ResolveEnvironment(env)
			if tc.reject == "" && err != nil {
				t.Fatalf("resolver rejected valid requirement: %v", err)
			}
			if tc.reject != "" && (err == nil || !strings.Contains(err.Error(), tc.reject)) {
				t.Fatalf("resolver expected %q: %v", tc.reject, err)
			}

			catalog = plugin.Catalog{Plugins: catalog.Plugins[1:], Bundles: []plugin.Bundle{{
				ID: cid("tei.bundle"), Version: "1.0.0", Plugins: []plugin.Requirement{tc.requirement},
			}}}
			env = environment.Environment{ID: cid("prod"), Bundles: []environment.ExtensionRequirement{{ID: cid("tei.bundle"), CompatibleMajor: 1}}}
			_, err = catalog.ResolveEnvironment(env)
			if tc.reject == "" && err != nil {
				t.Fatalf("bundle resolver rejected valid requirement: %v", err)
			}
			if tc.reject != "" && (err == nil || !strings.Contains(err.Error(), tc.reject)) {
				t.Fatalf("bundle resolver expected %q: %v", tc.reject, err)
			}
		})
	}
}

func TestRequirementEntryErrorIsOrderIndependent(t *testing.T) {
	a := plugin.Requirement{Plugin: core.CanonicalID("bad ID"), CompatibleMajor: 1}
	z := plugin.Requirement{Plugin: cid("tei.z"), CompatibleMajor: 0}
	first := []plugin.Requirement{z, a}
	second := []plugin.Requirement{a, z}
	err1 := plugin.ValidateRequirementEntries(first)
	err2 := plugin.ValidateRequirementEntries(second)
	if err1 == nil || err2 == nil || err1.Error() != err2.Error() {
		t.Fatalf("declaration order changed diagnostic: first=%v second=%v", err1, err2)
	}
}

package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestEnvironmentSelectionRejectsInvalidValuesBeforeBundleLookup(t *testing.T) {
	catalog := plugin.Catalog{
		Plugins: []plugin.Definition{
			{ID: cid("tei.base"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		},
		Bundles: []plugin.Bundle{
			{ID: cid("tei.bundle"), Version: "0.0.0", Plugins: nil},
			{ID: cid("tei.bundle"), Version: "1.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 1}}},
		},
	}
	tests := []struct {
		name string
		kind string
		req  environment.ExtensionRequirement
		want string
	}{
		{"bundle zero version match", "bundle", environment.ExtensionRequirement{ID: cid("tei.bundle"), CompatibleMajor: 0}, "compatible major"},
		{"bundle negative version", "bundle", environment.ExtensionRequirement{ID: cid("tei.bundle"), CompatibleMajor: -1}, "compatible major"},
		{"plugin zero version", "plugin", environment.ExtensionRequirement{ID: cid("tei.base"), CompatibleMajor: 0}, "compatible major"},
		{"plugin negative version", "plugin", environment.ExtensionRequirement{ID: cid("tei.base"), CompatibleMajor: -1}, "compatible major"},
		{"bundle empty ID", "bundle", environment.ExtensionRequirement{ID: "", CompatibleMajor: 1}, "invalid canonical id"},
		{"plugin empty ID", "plugin", environment.ExtensionRequirement{ID: "", CompatibleMajor: 1}, "invalid canonical id"},
		{"bundle padded ID", "bundle", environment.ExtensionRequirement{ID: core.CanonicalID(" tei.bundle "), CompatibleMajor: 1}, "invalid canonical id"},
		{"plugin padded ID", "plugin", environment.ExtensionRequirement{ID: core.CanonicalID(" tei.base "), CompatibleMajor: 1}, "invalid canonical id"},
		{"bundle invalid ID", "bundle", environment.ExtensionRequirement{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}, "invalid canonical id"},
		{"plugin invalid ID", "plugin", environment.ExtensionRequirement{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}, "invalid canonical id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := environment.Environment{ID: cid("prod")}
			if tc.kind == "bundle" {
				env.Bundles = []environment.ExtensionRequirement{tc.req}
			} else {
				env.Plugins = []environment.ExtensionRequirement{tc.req}
			}
			_, err := catalog.ResolveEnvironment(env)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("selection must fail before catalog lookup: expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestEnvironmentSelectionKeepsValidHighestCompatibleAndEmpty(t *testing.T) {
	catalog := plugin.Catalog{
		Plugins: []plugin.Definition{
			{ID: cid("tei.base"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
			{ID: cid("tei.base"), Version: "1.3.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		},
		Bundles: []plugin.Bundle{
			{ID: cid("tei.bundle"), Version: "1.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 1}}},
		},
	}
	empty := environment.Environment{ID: cid("prod")}
	resolved, err := catalog.ResolveEnvironment(empty)
	if err != nil || len(resolved.Plugins) != 0 {
		t.Fatalf("empty environment changed: %v %#v", err, resolved)
	}
	valid := environment.Environment{
		ID: cid("prod"),
		Plugins: []environment.ExtensionRequirement{{ID: cid("tei.base"), CompatibleMajor: 1}},
		Bundles: []environment.ExtensionRequirement{{ID: cid("tei.bundle"), CompatibleMajor: 1}},
	}
	resolved, err = catalog.ResolveEnvironment(valid)
	if err != nil || len(resolved.Plugins) != 1 || resolved.Plugins[0].Version != "1.3.0" {
		t.Fatalf("valid shared dependency no longer resolves highest compatible: %v %#v", err, resolved)
	}
}

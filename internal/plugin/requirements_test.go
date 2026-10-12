package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestPluginRequirementsRejectDuplicateIDs(t *testing.T) {
	id := cid("tei.dependency")
	other := cid("tei.other")
	for _, tc := range []struct {
		name string
		requires []plugin.Requirement
		reject bool
	}{
		{name: "same major", requires: []plugin.Requirement{{Plugin: id, CompatibleMajor: 1}, {Plugin: id, CompatibleMajor: 1}}, reject: true},
		{name: "different major", requires: []plugin.Requirement{{Plugin: id, CompatibleMajor: 1}, {Plugin: id, CompatibleMajor: 2}}, reject: true},
		{name: "distinct plugins", requires: []plugin.Requirement{{Plugin: id, CompatibleMajor: 1}, {Plugin: other, CompatibleMajor: 1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := plugin.ValidateRequirements(tc.requires)
			if (err != nil) != tc.reject {
				t.Fatalf("ValidateRequirements(%v) = %v; reject = %v", tc.requires, err, tc.reject)
			}
			if tc.reject && !strings.Contains(err.Error(), string(id)) {
				t.Fatalf("duplicate ID is missing from error: %v", err)
			}
		})
	}
}

func TestPluginRequirementsNormalizeOrderDeterministically(t *testing.T) {
	first := plugin.Definition{
		ID: cid("tei.root"), Version: "1.0.0",
		Requires: []plugin.Requirement{
			{Plugin: cid("tei.z"), CompatibleMajor: 2},
			{Plugin: cid("tei.a"), CompatibleMajor: 1},
		},
	}
	second := first
	second.Requires = []plugin.Requirement{first.Requires[1], first.Requires[0]}
	if plugin.BuildDefinitionFingerprint(first) != plugin.BuildDefinitionFingerprint(second) {
		t.Fatal("valid requirement declaration order changed the definition fingerprint")
	}
	// Even an invalid duplicate declaration has an unambiguous normalization.
	first.Requires = []plugin.Requirement{
		{Plugin: cid("tei.a"), CompatibleMajor: 2},
		{Plugin: cid("tei.a"), CompatibleMajor: 1},
	}
	second.Requires = []plugin.Requirement{first.Requires[1], first.Requires[0]}
	if plugin.BuildDefinitionFingerprint(first) != plugin.BuildDefinitionFingerprint(second) {
		t.Fatal("duplicate requirement order changed the definition fingerprint")
	}
}

func TestPluginPackageRejectsDuplicateDependenciesWithoutWorldValidation(t *testing.T) {
	source := artifactFreePackageSource()
	source.Requires = []plugin.Requirement{
		{Plugin: cid("tei.base"), CompatibleMajor: 1},
		{Plugin: cid("tei.base"), CompatibleMajor: 2},
	}
	if _, err := plugin.MaterializePluginPackage(source, pluginPackagePlan(source)); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("standalone package materializer must reject duplicate dependencies: %v", err)
	}
}

func TestPluginResolverRejectsDuplicateDependencies(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		{
			ID: cid("tei.root"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			Requires: []plugin.Requirement{
				{Plugin: cid("tei.base"), CompatibleMajor: 1},
				{Plugin: cid("tei.base"), CompatibleMajor: 1},
			},
		},
		{ID: cid("tei.base"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
	}}
	env := environment.Environment{ID: cid("prod"), Plugins: []environment.ExtensionRequirement{{ID: cid("tei.root"), CompatibleMajor: 1}}}
	if _, err := c.ResolveEnvironment(env); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("resolver must reject duplicate dependencies: %v", err)
	}
}

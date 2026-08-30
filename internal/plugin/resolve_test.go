package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func cid(s string) core.CanonicalID { return core.MustCanonicalID(s) }

func TestBundleResolvesHighestCompatiblePluginVersion(t *testing.T) {
	c := plugin.Catalog{
		Plugins: []plugin.Definition{
			{ID: cid("tei.runtime"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
			{ID: cid("tei.runtime"), Version: "1.3.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Provides: []plugin.Capability{{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.main"), Actions: []deployment.ActionKind{deployment.ActionApplicationStart}}}},
			{ID: cid("tei.runtime"), Version: "2.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		},
		Bundles: []plugin.Bundle{{ID: cid("tei.standard"), Version: "1.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.runtime"), CompatibleMajor: 1}}}},
	}
	env := environment.Environment{ID: cid("prod"), Bundles: []environment.ExtensionRequirement{{ID: cid("tei.standard"), CompatibleMajor: 1}}}
	r, err := c.ResolveEnvironment(env)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Plugins) != 1 || r.Plugins[0].Version != "1.3.0" {
		t.Fatalf("resolved: %#v", r.Plugins)
	}
	cap, ok := r.Capability(plugin.CapabilityDeploymentExecutor, cid("runtime.main"))
	if !ok || !cap.Capability.SupportsAction(deployment.ActionApplicationStart) {
		t.Fatalf("capability: %#v %v", cap, ok)
	}
}

func TestPluginDependencyCycleFails(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: cid("tei.a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{{Plugin: cid("tei.b"), CompatibleMajor: 1}}},
		{ID: cid("tei.b"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{{Plugin: cid("tei.a"), CompatibleMajor: 1}}},
	}}
	env := environment.Environment{ID: cid("prod"), Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}}}
	if _, err := c.ResolveEnvironment(env); err == nil {
		t.Fatal("expected dependency cycle")
	}
}

func TestIncompatibleTransitiveRequirementFails(t *testing.T) {
	c := plugin.Catalog{Plugins: []plugin.Definition{
		{ID: cid("tei.base"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		{ID: cid("tei.base"), Version: "2.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		{ID: cid("tei.a"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}, Requires: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 1}}},
	}}
	env := environment.Environment{ID: cid("prod"), Plugins: []environment.ExtensionRequirement{{ID: cid("tei.a"), CompatibleMajor: 1}, {ID: cid("tei.base"), CompatibleMajor: 2}}}
	if _, err := c.ResolveEnvironment(env); err == nil {
		t.Fatal("expected incompatible plugin resolution")
	}
}

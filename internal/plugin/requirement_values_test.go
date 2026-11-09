package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestRequirementValuesRejectMalformedIdentityAndNonpositiveMajor(t *testing.T) {
	cases := []struct {
		name string
		req  plugin.Requirement
		want string
	}{
		{"empty ID", plugin.Requirement{Plugin: "", CompatibleMajor: 1}, "invalid canonical id"},
		{"invalid ID", plugin.Requirement{Plugin: core.CanonicalID("not a valid id"), CompatibleMajor: 1}, "invalid canonical id"},
		{"whitespace-padded ID", plugin.Requirement{Plugin: core.CanonicalID(" tei.base "), CompatibleMajor: 1}, "invalid canonical id"},
		{"zero major", plugin.Requirement{Plugin: cid("tei.base"), CompatibleMajor: 0}, "invalid compatible major"},
		{"negative major", plugin.Requirement{Plugin: cid("tei.base"), CompatibleMajor: -1}, "invalid compatible major"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := plugin.ValidateRequirements([]plugin.Requirement{tc.req}); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("requirement validation: expected %q, got %v", tc.want, err)
			}
			source := artifactFreePackageSource()
			source.Requires = []plugin.Requirement{tc.req}
			if _, err := plugin.MaterializePluginPackage(source, pluginPackagePlan(source)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Go SB-1 materializer: expected %q, got %v", tc.want, err)
			}

			c := plugin.Catalog{Plugins: []plugin.Definition{{
				ID: cid("tei.root"), Version: "1.0.0",
				Requires: []plugin.Requirement{tc.req},
			}}}
			env := environment.Environment{ID: cid("env"), Plugins: []environment.ExtensionRequirement{{ID: cid("tei.root"), CompatibleMajor: 1}}}
			if _, err := c.ResolveEnvironment(env); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("standalone plugin resolver: expected %q, got %v", tc.want, err)
			}

			b := plugin.Catalog{Bundles: []plugin.Bundle{{
				ID: cid("tei.bundle"), Version: "1.0.0",
				Plugins: []plugin.Requirement{tc.req},
			}}}
			bundleEnv := environment.Environment{ID: cid("env"), Bundles: []environment.ExtensionRequirement{{ID: cid("tei.bundle"), CompatibleMajor: 1}}}
			if _, err := b.ResolveEnvironment(bundleEnv); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("standalone bundle resolver: expected %q, got %v", tc.want, err)
			}
		})
	}
}

func TestRequirementValueDiagnosticDoesNotDependOnOrder(t *testing.T) {
	a := []plugin.Requirement{
		{Plugin: cid("tei.z"), CompatibleMajor: 0},
		{Plugin: cid("tei.a"), CompatibleMajor: -1},
	}
	b := []plugin.Requirement{a[1], a[0]}
	errA := plugin.ValidateRequirements(a)
	errB := plugin.ValidateRequirements(b)
	if errA == nil || errB == nil || errA.Error() != errB.Error() {
		t.Fatalf("inconsistent diagnostic for permuted declarations: %v vs %v", errA, errB)
	}
	if !strings.Contains(errA.Error(), "tei.a") {
		t.Fatalf("expected lexically smallest invalid declaration: %v", errA)
	}
}

func TestRequirementValuesAllowEmptyAndValidRequirements(t *testing.T) {
	for _, requires := range [][]plugin.Requirement{
		nil,
		{},
		{{Plugin: cid("tei.a"), CompatibleMajor: 1}},
		{{Plugin: cid("tei.a"), CompatibleMajor: 1}, {Plugin: cid("tei.b"), CompatibleMajor: 2}},
	} {
		if err := plugin.ValidateRequirements(requires); err != nil {
			t.Fatalf("valid requires %v rejected: %v", requires, err)
		}
		source := artifactFreePackageSource()
		source.Requires = requires
		if _, err := plugin.MaterializePluginPackage(source, pluginPackagePlan(source)); err != nil {
			t.Fatalf("valid materialization %v rejected: %v", requires, err)
		}
	}
}

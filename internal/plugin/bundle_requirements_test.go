package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestBundleStandaloneResolverRejectsRepeatedPluginID(t *testing.T) {
	env := environment.Environment{
		ID:      cid("prod"),
		Bundles: []environment.ExtensionRequirement{
			{ID: cid("tei.bundle"), CompatibleMajor: 1},
		},
	}
	for _, majors := range [][2]int{{1, 1}, {1, 2}, {2, 1}} {
		c := plugin.Catalog{Bundles: []plugin.Bundle{{
			ID: cid("tei.bundle"), Version: "1.0.0",
			Plugins: []plugin.Requirement{
				{Plugin: cid("tei.base"), CompatibleMajor: majors[0]},
				{Plugin: cid("tei.base"), CompatibleMajor: majors[1]},
			},
		}}}
		_, err := c.ResolveEnvironment(env)
		if err == nil || !strings.Contains(err.Error(), "plugin requirement tei.base appears more than once") {
			t.Fatalf("majors %v: expected declaration-level duplicate error, got %v", majors, err)
		}
		if !strings.Contains(err.Error(), "bundle tei.bundle version 1.0.0") {
			t.Fatalf("majors %v: duplicate error must identify the bundle: %v", majors, err)
		}
	}
}

func TestBundleStandaloneResolverDuplicateDiagnosticIsOrderIndependent(t *testing.T) {
	a := plugin.Requirement{Plugin: cid("tei.a"), CompatibleMajor: 1}
	a2 := plugin.Requirement{Plugin: cid("tei.a"), CompatibleMajor: 2}
	z := plugin.Requirement{Plugin: cid("tei.z"), CompatibleMajor: 1}
	z2 := plugin.Requirement{Plugin: cid("tei.z"), CompatibleMajor: 2}
	inputs := [][]plugin.Requirement{
		{z, a, z2, a2},
		{a2, z2, a, z},
	}
	env := environment.Environment{
		ID:      cid("prod"),
		Bundles: []environment.ExtensionRequirement{
			{ID: cid("tei.bundle"), CompatibleMajor: 1},
		},
	}
	var firstError string
	for _, requirements := range inputs {
		c := plugin.Catalog{Bundles: []plugin.Bundle{{
			ID: cid("tei.bundle"), Version: "1.0.0", Plugins: requirements,
		}}}
		_, err := c.ResolveEnvironment(env)
		if err == nil || !strings.Contains(err.Error(), "plugin requirement tei.a appears more than once") {
			t.Fatalf("expected stable lexically smallest duplicate, got %v", err)
		}
		if firstError == "" {
			firstError = err.Error()
		} else if err.Error() != firstError {
			t.Fatalf("input order changed the error: first=%q, second=%q", firstError, err)
		}
	}
}

func TestBundleStandaloneResolverAllowsSharedRequirements(t *testing.T) {
	requirement := plugin.Requirement{Plugin: cid("tei.base"), CompatibleMajor: 1}
	c := plugin.Catalog{
		Plugins: []plugin.Definition{{
			ID: cid("tei.base"), Version: "1.0.0",
			Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		}},
		Bundles: []plugin.Bundle{
			{ID: cid("tei.first"), Version: "1.0.0", Plugins: []plugin.Requirement{requirement}},
			{ID: cid("tei.second"), Version: "1.0.0", Plugins: []plugin.Requirement{requirement}},
		},
	}
	env := environment.Environment{
		ID:      cid("prod"),
		Plugins: []environment.ExtensionRequirement{
			{ID: cid("tei.base"), CompatibleMajor: 1},
		},
		Bundles: []environment.ExtensionRequirement{
			{ID: cid("tei.first"), CompatibleMajor: 1},
			{ID: cid("tei.second"), CompatibleMajor: 1},
		},
	}
	resolved, err := c.ResolveEnvironment(env)
	if err != nil {
		t.Fatalf("shared requirement must be valid across declarations: %v", err)
	}
	if len(resolved.Plugins) != 1 || resolved.Plugins[0].ID != cid("tei.base") {
		t.Fatalf("expected one resolved shared plugin, got %#v", resolved.Plugins)
	}
}

func TestBundleStandaloneResolverValidatesOnlySelectedVersion(t *testing.T) {
	requirement := plugin.Requirement{Plugin: cid("tei.base"), CompatibleMajor: 1}
	bad := plugin.Bundle{
		ID: cid("tei.bundle"), Version: "1.0.0",
		Plugins: []plugin.Requirement{requirement, requirement},
	}
	good := plugin.Bundle{
		ID: cid("tei.bundle"), Version: "1.0.0",
		Plugins: []plugin.Requirement{requirement},
	}
	env := environment.Environment{
		ID:      cid("prod"),
		Bundles: []environment.ExtensionRequirement{
			{ID: cid("tei.bundle"), CompatibleMajor: 1},
		},
	}
	for _, tc := range []struct {
		name      string
		old       plugin.Bundle
		latest    plugin.Bundle
		wantError bool
	}{
		{name: "newest version invalid", old: good, latest: bad, wantError: true},
		{name: "older version invalid", old: bad, latest: good},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.latest.Version = "1.2.0"
			c := plugin.Catalog{
				Plugins: []plugin.Definition{{
					ID: cid("tei.base"), Version: "1.0.0",
					Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
				}},
				Bundles: []plugin.Bundle{tc.old, tc.latest},
			}
			resolved, err := c.ResolveEnvironment(env)
			if tc.wantError {
				if err == nil || !strings.Contains(err.Error(), "bundle tei.bundle version 1.2.0: plugin requirement tei.base appears more than once") {
					t.Fatalf("selected invalid bundle must fail before resolution: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("older unselected declaration must not block standalone resolution: %v", err)
			}
			if len(resolved.Plugins) != 1 || resolved.Plugins[0].ID != cid("tei.base") {
				t.Fatalf("unexpected resolved plugin set: %#v", resolved.Plugins)
			}
		})
	}
}

func TestBundleStandaloneResolverKeepsCrossBundleMajorConflict(t *testing.T) {
	c := plugin.Catalog{
		Plugins: []plugin.Definition{
			{ID: cid("tei.base"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
			{ID: cid("tei.base"), Version: "2.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		},
		Bundles: []plugin.Bundle{
			{ID: cid("tei.first"), Version: "1.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 1}}},
			{ID: cid("tei.second"), Version: "1.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 2}}},
		},
	}
	env := environment.Environment{
		ID:      cid("prod"),
		Bundles: []environment.ExtensionRequirement{
			{ID: cid("tei.first"), CompatibleMajor: 1},
			{ID: cid("tei.second"), CompatibleMajor: 1},
		},
	}
	_, err := c.ResolveEnvironment(env)
	if err == nil || !strings.Contains(err.Error(), "incompatible with required major 2") {
		t.Fatalf("cross-bundle version conflict must remain an incompatibility: %v", err)
	}
	if strings.Contains(err.Error(), "appears more than once") {
		t.Fatalf("different bundle declarations must not be mistaken for an intra-bundle duplicate: %v", err)
	}
}

package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func environmentSelectionCatalog() plugin.Catalog {
	return plugin.Catalog{
		Plugins: []plugin.Definition{
			{ID: cid("tei.base"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
			{ID: cid("tei.base"), Version: "2.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
			{ID: cid("tei.other"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}},
		},
		Bundles: []plugin.Bundle{
			{ID: cid("tei.standard"), Version: "1.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 1}}},
			{ID: cid("tei.standard"), Version: "2.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 2}}},
			{ID: cid("tei.other_bundle"), Version: "1.0.0", Plugins: []plugin.Requirement{{Plugin: cid("tei.base"), CompatibleMajor: 1}}},
		},
	}
}

func TestEnvironmentSelectorDuplicatesRejectedBeforeResolution(t *testing.T) {
	catalog := environmentSelectionCatalog()
	cases := []struct {
		name    string
		env     environment.Environment
		context string
	}{
		{"plugin same major", environment.Environment{Plugins: []environment.ExtensionRequirement{{ID: cid("tei.base"), CompatibleMajor: 1}, {ID: cid("tei.base"), CompatibleMajor: 1}}}, "plugin selections"},
		{"plugin different major", environment.Environment{Plugins: []environment.ExtensionRequirement{{ID: cid("tei.base"), CompatibleMajor: 1}, {ID: cid("tei.base"), CompatibleMajor: 2}}}, "plugin selections"},
		{"bundle same major", environment.Environment{Bundles: []environment.ExtensionRequirement{{ID: cid("tei.standard"), CompatibleMajor: 1}, {ID: cid("tei.standard"), CompatibleMajor: 1}}}, "bundle selections"},
		{"bundle different major", environment.Environment{Bundles: []environment.ExtensionRequirement{{ID: cid("tei.standard"), CompatibleMajor: 1}, {ID: cid("tei.standard"), CompatibleMajor: 2}}}, "bundle selections"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.env.ID = cid("prod")
			_, err := catalog.ResolveEnvironment(tc.env)
			if err == nil || !strings.Contains(err.Error(), "more than once") || !strings.Contains(err.Error(), tc.context) {
				t.Fatalf("duplicate environment selection accepted or misdiagnosed: %v", err)
			}
		})
	}
}

func TestEnvironmentSelectorDuplicateDiagnosticIndependentOfOrder(t *testing.T) {
	catalog := environmentSelectionCatalog()
	original := []environment.ExtensionRequirement{
		{ID: cid("tei.other"), CompatibleMajor: 1},
		{ID: cid("tei.base"), CompatibleMajor: 1},
		{ID: cid("tei.other"), CompatibleMajor: 1},
		{ID: cid("tei.base"), CompatibleMajor: 2},
	}
	reversed := []environment.ExtensionRequirement{original[3], original[2], original[1], original[0]}
	first := environment.Environment{ID: cid("prod"), Plugins: original}
	second := environment.Environment{ID: cid("prod"), Plugins: reversed}
	_, errA := catalog.ResolveEnvironment(first)
	_, errB := catalog.ResolveEnvironment(second)
	if errA == nil || errB == nil || errA.Error() != errB.Error() || !strings.Contains(errA.Error(), "tei.base") {
		t.Fatalf("duplicate diagnostic depends on declaration order: first=%v second=%v", errA, errB)
	}
}

func TestEnvironmentSharedDependenciesRemainValid(t *testing.T) {
	catalog := environmentSelectionCatalog()
	cases := []struct {
		name string
		env  environment.Environment
	}{
		{"plugin and bundle share plugin", environment.Environment{
			Plugins: []environment.ExtensionRequirement{{ID: cid("tei.base"), CompatibleMajor: 1}},
			Bundles: []environment.ExtensionRequirement{{ID: cid("tei.standard"), CompatibleMajor: 1}},
		}},
		{"two bundles share plugin", environment.Environment{
			Bundles: []environment.ExtensionRequirement{{ID: cid("tei.standard"), CompatibleMajor: 1}, {ID: cid("tei.other_bundle"), CompatibleMajor: 1}},
		}},
		{"different direct plugins", environment.Environment{
			Plugins: []environment.ExtensionRequirement{{ID: cid("tei.base"), CompatibleMajor: 1}, {ID: cid("tei.other"), CompatibleMajor: 1}},
		}},
		{"empty selection", environment.Environment{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.env.ID = cid("prod")
			got, err := catalog.ResolveEnvironment(tc.env)
			if err != nil {
				t.Fatalf("valid shared dependency rejected: %v", err)
			}
			if tc.name != "empty selection" && len(got.Plugins) == 0 {
				t.Fatal("valid selector produced no Plugin")
			}
			if len(got.Plugins) > 2 {
				t.Fatalf("shared dependencies should resolve once: %+v", got.Plugins)
			}
		})
	}
}

func TestEnvironmentSelectorValuesRejectMalformedSourceBeforeLookup(t *testing.T) {
	catalog := environmentSelectionCatalog()
	cases := []struct {
		name string
		env  environment.Environment
		want string
	}{
		{"plugin invalid id", environment.Environment{Plugins: []environment.ExtensionRequirement{{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}}}, "invalid canonical id"},
		{"plugin padded id", environment.Environment{Plugins: []environment.ExtensionRequirement{{ID: core.CanonicalID(" tei.base "), CompatibleMajor: 1}}}, "invalid canonical id"},
		{"plugin zero major", environment.Environment{Plugins: []environment.ExtensionRequirement{{ID: cid("tei.base"), CompatibleMajor: 0}}}, "invalid compatible major"},
		{"bundle invalid id", environment.Environment{Bundles: []environment.ExtensionRequirement{{ID: core.CanonicalID("bad id"), CompatibleMajor: 1}}}, "invalid canonical id"},
		{"bundle negative major", environment.Environment{Bundles: []environment.ExtensionRequirement{{ID: cid("tei.standard"), CompatibleMajor: -1}}}, "invalid compatible major"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.env.ID = cid("prod")
			if _, err := catalog.ResolveEnvironment(tc.env); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid environment selection accepted/misdiagnosed, need %q: %v", tc.want, err)
			}
		})
	}
}

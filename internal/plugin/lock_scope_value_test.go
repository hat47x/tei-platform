package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

// An environment-scoped API comparison, not a Workspace validation bypass.
func TestBuildLockSelectedEnvironmentAndUnrelatedFailure(t *testing.T) {
	id := core.MustCanonicalID("tei.example.runtime")
	def := plugin.Definition{ID: id, Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin}}
	development := environment.Environment{ID: core.MustCanonicalID("development"), Plugins: []environment.ExtensionRequirement{{ID: id, CompatibleMajor: 1}}}
	production := environment.Environment{ID: core.MustCanonicalID("production"), Plugins: []environment.ExtensionRequirement{{ID: core.MustCanonicalID("tei.missing"), CompatibleMajor: 1}}}
	catalog := plugin.Catalog{Plugins: []plugin.Definition{def}}
	if _, err := plugin.BuildLock(catalog, []environment.Environment{development, production}); err == nil {
		t.Fatal("all-environment lock did not reject missing production plugin")
	}
	selected, err := plugin.BuildLock(catalog, []environment.Environment{development})
	if err != nil || len(selected.Environments) != 1 || selected.Environments[0].Environment != development.ID {
		t.Fatalf("selected environment lock: %+v, %v", selected, err)
	}
	resolved, err := catalog.ResolveEnvironmentWithLock(development, selected)
	if err != nil || len(resolved.Plugins) != 1 || resolved.Plugins[0].ID != id {
		t.Fatalf("selected resolution: %+v, %v", resolved, err)
	}
	if _, err := catalog.ResolveEnvironmentWithLock(production, selected); err == nil || !strings.Contains(err.Error(), "no environment") {
		t.Fatalf("partial lock was accepted for an unpinned environment: %v", err)
	}
	changed := def
	changed.Runtime.Command = "modified-descriptor"
	if _, err := (plugin.Catalog{Plugins: []plugin.Definition{changed}}).ResolveEnvironmentWithLock(development, selected); err == nil || !strings.Contains(err.Error(), "fingerprint diverged") {
		t.Fatalf("changed selected descriptor was accepted: %v", err)
	}
	t.Log("existing API: all environments fail on missing production plugin; development-only lock resolves development and rejects production and modified descriptor")
}

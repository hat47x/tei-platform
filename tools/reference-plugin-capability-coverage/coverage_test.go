package plugincoverage

import (
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
)

func capability(kind plugin.CapabilityKind, id core.CanonicalID) plugin.Capability {
	return plugin.Capability{Kind: kind, ID: id}
}

func world() ir.World {
	return ir.World{
		Plugins: []plugin.Definition{
			{ID: "tei.db.postgres", Version: "1.0.0", Provides: []plugin.Capability{
				capability(plugin.CapabilityDeploymentExecutor, "database.postgres"),
			}},
			{ID: "tei.editor.sql", Version: "1.0.0", Provides: []plugin.Capability{
				capability(plugin.CapabilityEditorIntelligence, "editor.sqlglot"),
			}},
		},
		Environments: []environment.Environment{
			{ID: "development", DeploymentExecutors: []environment.DeploymentExecutorBinding{
				{Action: deployment.ActionDatabaseRelease, Executor: "database.postgres"},
			}},
		},
	}
}

func find(r Report, pluginID, capabilityID string) *Capability {
	for i := range r.Capabilities {
		if r.Capabilities[i].Plugin == pluginID && r.Capabilities[i].ID == capabilityID {
			return &r.Capabilities[i]
		}
	}
	return nil
}

func TestASelectedCapabilityNamesItsEnvironment(t *testing.T) {
	r := Derive(world())
	got := find(r, "tei.db.postgres", "database.postgres")
	if got == nil || got.Status != Selected {
		t.Fatalf("postgres = %+v", got)
	}
	if len(got.Environments) != 1 || got.Environments[0] != "development" {
		t.Fatalf("environments = %v", got.Environments)
	}
}

func TestAnUnselectedCapabilityIsReportedNotOmitted(t *testing.T) {
	// The editor plugin declares a capability no environment binds. It stays
	// in the report as unselected; it is not dropped and is not called unused.
	r := Derive(world())
	got := find(r, "tei.editor.sql", "editor.sqlglot")
	if got == nil || got.Status != Unselected {
		t.Fatalf("editor = %+v", got)
	}
	if len(got.Environments) != 0 {
		t.Fatalf("unselected must name no environment, got %v", got.Environments)
	}
	if r.Unselected != 1 {
		t.Fatalf("unselected count = %d", r.Unselected)
	}
}

func TestTheSameCapabilityInTwoEnvironmentsIsListedOncePerEnvironment(t *testing.T) {
	w := world()
	w.Environments = append(w.Environments, environment.Environment{
		ID: "production", DeploymentExecutors: []environment.DeploymentExecutorBinding{{Action: deployment.ActionDatabaseRelease, Executor: "database.postgres"}},
	})
	r := Derive(w)
	got := find(r, "tei.db.postgres", "database.postgres")
	if got == nil || len(got.Environments) != 2 {
		t.Fatalf("postgres = %+v", got)
	}
	if got.Environments[0] != "development" || got.Environments[1] != "production" {
		t.Fatalf("environments not sorted: %v", got.Environments)
	}
}

func TestAPluginThatDeclaresNothingContributesNothing(t *testing.T) {
	w := world()
	w.Plugins = append(w.Plugins, plugin.Definition{ID: "tei.empty", Version: "1.0.0"})
	r := Derive(w)
	if find(r, "tei.empty", "") != nil {
		t.Fatal("a plugin with no capability produced a row")
	}
	if r.Unselected != 1 {
		t.Fatalf("unselected = %d; an empty plugin must not add one", r.Unselected)
	}
}

func TestADatabaseInstanceSelectsTheProviderByTechnology(t *testing.T) {
	// The instance names "postgres"; the capability ID is "database.postgres".
	// Resolving it must not be a text match, and it must mark the capability
	// selected.
	w := ir.World{
		Plugins: []plugin.Definition{{ID: "tei.db.postgres", Version: "1.0.0", Provides: []plugin.Capability{{
			Kind:              plugin.CapabilityDatabaseProvider,
			ID:                "database.postgres",
			DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres},
		}}}},
		Environments: []environment.Environment{{ID: "development", Instances: []environment.DatabaseInstance{
			{ID: "sales.dev", Provider: environment.ProviderPostgres},
		}}},
	}
	r := Derive(w)
	got := find(r, "tei.db.postgres", "database.postgres")
	if got == nil || got.Status != Selected {
		t.Fatalf("postgres = %+v", got)
	}
	if len(got.Environments) != 1 || got.Environments[0] != "development" {
		t.Fatalf("environments = %v", got.Environments)
	}
}

func TestAnInstanceTechnologyNoCapabilitySupportsStaysUnselected(t *testing.T) {
	w := ir.World{
		Plugins: []plugin.Definition{{ID: "tei.db.postgres", Version: "1.0.0", Provides: []plugin.Capability{{
			Kind:              plugin.CapabilityDatabaseProvider,
			ID:                "database.postgres",
			DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres},
		}}}},
		Environments: []environment.Environment{{ID: "development", Instances: []environment.DatabaseInstance{
			{ID: "sales.dev", Provider: environment.ProviderOracle},
		}}},
	}
	r := Derive(w)
	if got := find(r, "tei.db.postgres", "database.postgres"); got == nil || got.Status != Unselected {
		t.Fatalf("oracle instance selected postgres = %+v", got)
	}
}

func TestReportStatesAuthorityAndIsDeterministic(t *testing.T) {
	a := Derive(world())
	b := Derive(world())
	ja, err := a.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	jb, err := b.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if string(ja) != string(jb) {
		t.Fatal("report is not repeatable")
	}
	if a.RuleFingerprint == "" || a.Authority != Authority || a.Schema != Schema {
		t.Fatalf("bad header: %+v", a)
	}
}

func TestSelectedPluginDoesNotImplySelectedCapability(t *testing.T) {
	// An Environment can select the plugin package without binding its
	// editor capability. Do not elevate package selection to capability use.
	w := world()
	w.Environments[0].Plugins = []environment.ExtensionRequirement{
		{ID: "tei.editor.sql", CompatibleMajor: 1},
	}
	r := Derive(w)
	got := find(r, "tei.editor.sql", "editor.sqlglot")
	if got == nil || got.Status != Unselected || len(got.Environments) != 0 {
		t.Fatalf("package selection must not select capability: %+v", got)
	}
}

func TestOneSelectedCapabilityDoesNotSelectAllOfItsPluginCapabilities(t *testing.T) {
	// The deployment-executor binding uses one offered capability;
	// another offered capability from the same plugin remains unselected.
	w := world()
	w.Plugins[0].Provides = append(w.Plugins[0].Provides,
		capability(plugin.CapabilityEditorIntelligence, "editor.postgres"))
	r := Derive(w)
	chosen := find(r, "tei.db.postgres", "database.postgres")
	unchosen := find(r, "tei.db.postgres", "editor.postgres")
	if chosen == nil || chosen.Status != Selected {
		t.Fatalf("chosen capability = %+v", chosen)
	}
	if unchosen == nil || unchosen.Status != Unselected {
		t.Fatalf("another capability of chosen plugin = %+v", unchosen)
	}
}

func TestDatabaseTechnologyCoverageDoesNotClaimUniqueProviderResolution(t *testing.T) {
	// This report measures which declarations match an Environment's
	// database technology. Two matches can both be reported "selected" here
	// even though an actual Provider Resolver must detect the ambiguity.
	w := ir.World{
		Plugins: []plugin.Definition{
			{ID: "tei.db.first", Version: "1.0.0", Provides: []plugin.Capability{{
				Kind: plugin.CapabilityDatabaseProvider, ID: "database.first",
				DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres},
			}}},
			{ID: "tei.db.second", Version: "1.0.0", Provides: []plugin.Capability{{
				Kind: plugin.CapabilityDatabaseProvider, ID: "database.second",
				DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres},
			}}},
		},
		Environments: []environment.Environment{{ID: "development", Instances: []environment.DatabaseInstance{{
			ID: "sales.dev", Provider: environment.ProviderPostgres,
		}}}},
	}
	r := Derive(w)
	for _, id := range []struct{ pluginID, capabilityID string }{
		{"tei.db.first", "database.first"},
		{"tei.db.second", "database.second"},
	} {
		got := find(r, id.pluginID, id.capabilityID)
		if got == nil || got.Status != Selected {
			t.Fatalf("coverage candidate %s = %+v", id.pluginID, got)
		}
	}
	if r.Authority != "derived-read-only" || r.Unselected != 0 {
		t.Fatalf("report cannot certify a unique provider: %+v", r)
	}
}

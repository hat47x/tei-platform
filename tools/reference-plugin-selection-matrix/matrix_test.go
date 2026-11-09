package selectionmatrix

import (
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
)

func world() ir.World {
	dbCap := plugin.Capability{
		Kind: plugin.CapabilityDatabaseProvider, ID: "database.postgres",
		DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres},
	}
	exec := plugin.Capability{
		Kind: plugin.CapabilityDeploymentExecutor, ID: "migration.atlas",
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	obs := plugin.Capability{
		Kind: plugin.CapabilityObservationProvider, ID: "database.observer",
		Observations: []deployment.ObservationKind{deployment.ObservationDatabaseState},
	}
	mig := plugin.Capability{
		Kind: plugin.CapabilityMigrationProvider, ID: "migration.atlas",
		MigrationTools: []migration.Tool{migration.ToolAtlas},
	}
	return ir.World{
		Plugins: []plugin.Definition{
			{ID: "tei.db.postgres", Version: "1.0.0", Provides: []plugin.Capability{dbCap, exec}},
			{ID: "tei.observer", Version: "1.0.0", Provides: []plugin.Capability{obs}},
			{ID: "tei.migrate", Version: "1.0.0", Provides: []plugin.Capability{mig}},
		},
		Environments: []environment.Environment{{
			ID: "development",
			Instances: []environment.DatabaseInstance{
				{ID: "sales.dev", Provider: environment.ProviderPostgres},
			},
			DeploymentExecutors: []environment.DeploymentExecutorBinding{
				{Action: deployment.ActionDatabaseRelease, Executor: "migration.atlas"},
			},
			ObservationProviders: []environment.ObservationProviderBinding{
				{Kind: deployment.ObservationDatabaseState, Provider: "database.observer"},
			},
			EvidenceProviders: []environment.EvidenceProviderBinding{
				{Kind: deployment.EvidenceRolloutHealth, Provider: "health.reference"},
			},
			Features: []environment.Feature{"editor.sql_analysis"},
		}},
		MigrationProviders: []migration.ProviderConfig{{Database: "sales_core", Tool: migration.ToolAtlas}},
	}
}

func find(r Report, form, binding string) *Selection {
	for i := range r.Selections {
		if r.Selections[i].Form == form && r.Selections[i].Binding == binding {
			return &r.Selections[i]
		}
	}
	return nil
}

func TestEveryBindingFormIsResolvedByItsDeclaredSupportSet(t *testing.T) {
	r := Derive(world())
	cases := []struct{ form, binding, want string }{
		{"deployment executor", "database-release", VerdictSelected},
		{"observation provider", "database-state", VerdictSelected},
		{"database instance", "postgres", VerdictSelected},
		{"environment feature", "editor.sql_analysis", VerdictMissing},
	}
	for _, tc := range cases {
		got := find(r, tc.form, tc.binding)
		if got == nil {
			t.Fatalf("%s %s missing from report", tc.form, tc.binding)
		}
		if got.Verdict != tc.want {
			t.Fatalf("%s %s = %s, want %s", tc.form, tc.binding, got.Verdict, tc.want)
		}
	}
}

func TestAnUnsupportedBindingIsMissingNotGuessed(t *testing.T) {
	// The environment binds an evidence provider that no declared capability
	// supports.
	r := Derive(world())
	got := find(r, "evidence provider", "rollout-health")
	if got == nil || got.Verdict != VerdictMissing {
		t.Fatalf("evidence = %+v", got)
	}
	if r.Missing != 2 {
		t.Fatalf("missing count = %d, want 2 (evidence + feature)", r.Missing)
	}
}

func TestTwoSupportedCapabilitiesAreAmbiguousAndNotChosen(t *testing.T) {
	w := world()
	w.Plugins = append(w.Plugins, plugin.Definition{ID: "tei.db.postgres.alt", Version: "1.0.0",
		Provides: []plugin.Capability{{
			Kind: plugin.CapabilityDatabaseProvider, ID: "database.postgres.alt",
			DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres},
		}}})
	r := Derive(w)
	got := find(r, "database instance", "postgres")
	if got == nil || got.Verdict != VerdictAmbiguous {
		t.Fatalf("postgres = %+v", got)
	}
	if len(got.Capabilities) != 2 {
		t.Fatalf("ambiguous must list both, got %v", got.Capabilities)
	}
}

func TestAMigrationProviderBindsByToolNotByDatabaseName(t *testing.T) {
	r := Derive(world())
	got := find(r, "migration provider", "sales_core")
	// The binding is listed under the database it applies to; the matching is
	// by tool, so it resolves.
	if got == nil || got.Verdict != VerdictSelected {
		t.Fatalf("migration = %+v", got)
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

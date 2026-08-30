package dependency_test

import (
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/database/migration"
	"tei/internal/dependency"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/realization"
)

func TestEnvironmentFeaturesDeriveScopedRequirements(t *testing.T) {
	w := ir.World{
		Plugins: []plugin.Definition{{
			ID: core.MustCanonicalID("tei.editor.sql"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeProcess, Command: "python3 worker"},
			External: []plugin.ExternalRequirement{{Scope: "editor", Kind: "runtime", Name: "python"}, {Scope: "editor", Kind: "language-package", Name: "sqlglot"}},
		}},
		Applications: []application.Application{{ID: core.MustCanonicalID("app")}},
		Environments: []environment.Environment{{
			ID:       core.MustCanonicalID("development"),
			Plugins:  []environment.ExtensionRequirement{{ID: core.MustCanonicalID("tei.editor.sql"), CompatibleMajor: 1}},
			Features: []environment.Feature{environment.FeatureSQLAnalysis},
			Bindings: []environment.ApplicationDatabaseBinding{{Application: core.MustCanonicalID("app")}},
		}},
		Bindings: []realization.CustomGoBinding{{Application: core.MustCanonicalID("app"), Capability: core.MustCanonicalID("app.custom")}},
	}
	got := dependency.Derive(w)
	want := map[string]bool{
		"editor|runtime|python":           false,
		"editor|language-package|sqlglot": false,
		"build|toolchain|go":              false,
	}
	for _, r := range got {
		key := string(r.Scope) + "|" + string(r.Kind) + "|" + r.Name
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for k, ok := range want {
		if !ok {
			t.Fatalf("missing derived requirement %s: %#v", k, got)
		}
	}
}

func TestSameDependencyAggregatesMultipleOrigins(t *testing.T) {
	w := ir.World{
		Applications: []application.Application{{ID: core.MustCanonicalID("app")}},
		Environments: []environment.Environment{{
			ID:       core.MustCanonicalID("development"),
			Bindings: []environment.ApplicationDatabaseBinding{{Application: core.MustCanonicalID("app")}},
		}},
		Bindings: []realization.CustomGoBinding{
			{Application: core.MustCanonicalID("app"), Capability: core.MustCanonicalID("app.a")},
			{Application: core.MustCanonicalID("app"), Capability: core.MustCanonicalID("app.b")},
		},
	}
	got := dependency.Derive(w)
	var goReq *dependency.Requirement
	for i := range got {
		if got[i].Name == "go" {
			goReq = &got[i]
		}
	}
	if goReq == nil || len(goReq.Origins) != 2 {
		t.Fatalf("expected one Go requirement with two origins, got %#v", got)
	}
}

func TestMigrationProviderAddsToolRequirement(t *testing.T) {
	w := ir.World{
		Plugins: []plugin.Definition{{
			ID: core.MustCanonicalID("tei.migration.atlas"), Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
			External: []plugin.ExternalRequirement{{Scope: "migration", Kind: "tool", Name: "atlas"}},
		}},
		Databases:          []database.Database{{ID: core.MustCanonicalID("sales_core"), Kind: "relational", Ownership: database.OwnershipManagedHere}},
		MigrationProviders: []migration.ProviderConfig{{Database: core.MustCanonicalID("sales_core"), Tool: migration.ToolAtlas, Mode: migration.ModeVersioned}},
		Environments:       []environment.Environment{{ID: core.MustCanonicalID("development"), Plugins: []environment.ExtensionRequirement{{ID: core.MustCanonicalID("tei.migration.atlas"), CompatibleMajor: 1}}, Instances: []environment.DatabaseInstance{{ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"), Provider: environment.ProviderPostgres}}}},
	}
	rs := dependency.Derive(w)
	for _, r := range rs {
		if r.Scope == dependency.ScopeMigration && r.Kind == dependency.KindTool && r.Name == "atlas" {
			return
		}
	}
	t.Fatalf("atlas migration requirement not derived: %#v", rs)
}

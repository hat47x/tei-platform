package runtimepackaging

import (
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
)

func TestActualOrderEntryDatabaseProviders(t *testing.T) {
	report, err := ObserveWorkspaceDatabaseProviders("../workspace", "order_entry", "development")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 2 {
		t.Fatalf("expected two explicitly-bound databases, got %+v", report)
	}
	if p := report.Providers[0]; p.Alias != "master" || p.Database != "master_data" ||
		p.Provider != "oracle" || p.Plugin != "tei.database.oracle" || len(p.PluginFingerprint) != 64 {
		t.Fatalf("wrong master provider: %+v", p)
	}
	if p := report.Providers[1]; p.Alias != "sales" || p.Database != "sales_core" ||
		p.Provider != "postgres" || p.Plugin != "tei.database.postgres" || len(p.PluginFingerprint) != 64 {
		t.Fatalf("wrong sales provider: %+v", p)
	}
}

func sampleWorkspace() (ir.World, plugin.Lock) {
	provider := plugin.Definition{
		ID: "tei.demo.postgres", Version: "1.0.0", Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{Kind: plugin.CapabilityDatabaseProvider,
			ID: "database.postgres", DatabaseProviders: []environment.DatabaseProvider{environment.ProviderPostgres}}},
	}
	env := environment.Environment{
		ID: "development",
		Plugins: []environment.ExtensionRequirement{{ID: provider.ID, CompatibleMajor: 1}},
		Instances: []environment.DatabaseInstance{{ID: "db.dev", Definition: "db", Provider: environment.ProviderPostgres}},
		Bindings: []environment.ApplicationDatabaseBinding{{Application: "app", Alias: "primary", Instance: "db.dev"}},
	}
	world := ir.World{
		Applications: []application.Application{{ID: "app",
			Databases: []application.DatabaseRequirement{{Alias: "primary", TargetDatabase: "db"}}}},
		Environments: []environment.Environment{env}, Plugins: []plugin.Definition{provider},
	}
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: world.Plugins}, []environment.Environment{env})
	if err != nil {
		panic(err)
	}
	return world, lock
}

func TestBuiltinProviderNeedsNoInventedArtifactDigest(t *testing.T) {
	world, lock := sampleWorkspace()
	report, err := ObserveDatabaseProviders(world, lock, "app", "development")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Providers) != 1 || report.Providers[0].ArtifactDigest != "" ||
		len(report.Providers[0].PluginFingerprint) != 64 {
		t.Fatalf("builtin must use the actual descriptor fingerprint: %+v", report)
	}
}

func TestRejectsStaleDescriptorLock(t *testing.T) {
	world, lock := sampleWorkspace()
	lock.Environments[0].Plugins[0].Fingerprint = strings.Repeat("0", 64)
	if _, err := ObserveDatabaseProviders(world, lock, "app", "development"); err == nil {
		t.Fatal("expected a lock mismatch error")
	}
}

func TestRejectsMissingAndMismatchedEnvironmentBinding(t *testing.T) {
	world, lock := sampleWorkspace()
	world.Environments[0].Bindings = nil
	if _, err := ObserveDatabaseProviders(world, lock, "app", "development"); err == nil {
		t.Fatal("expected missing database binding")
	}
	world, lock = sampleWorkspace()
	world.Environments[0].Instances[0].Definition = "other_db"
	if _, err := ObserveDatabaseProviders(world, lock, "app", "development"); err == nil {
		t.Fatal("expected binding to a different database definition to fail")
	}
}

func TestRejectsAmbiguousProvider(t *testing.T) {
	world, _ := sampleWorkspace()
	another := world.Plugins[0]
	another.ID = "tei.demo.another"
	world.Plugins = append(world.Plugins, another)
	env := &world.Environments[0]
	env.Plugins = append(env.Plugins, environment.ExtensionRequirement{ID: another.ID, CompatibleMajor: 1})
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: world.Plugins}, world.Environments)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ObserveDatabaseProviders(world, lock, "app", "development")
	if err == nil || !strings.Contains(err.Error(), "2 providers") {
		t.Fatalf("expected ambiguous provider rejection; got %v", err)
	}
}

func TestRejectsDuplicateDatabaseAlias(t *testing.T) {
	world, lock := sampleWorkspace()
	world.Applications[0].Databases = append(world.Applications[0].Databases, world.Applications[0].Databases[0])
	if _, err := ObserveDatabaseProviders(world, lock, "app", "development"); err == nil {
		t.Fatal("duplicate application alias must fail closed")
	}
}

func TestRejectsWrongApplicationAndEnvironment(t *testing.T) {
	world, lock := sampleWorkspace()
	for _, ids := range [][2]string{{"other", "development"}, {"app", "production"}} {
		if _, err := ObserveDatabaseProviders(world, lock, ids[0], ids[1]); err == nil {
			t.Fatalf("wrong scope %v must fail", ids)
		}
	}
}


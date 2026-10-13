package plugin_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestCapabilitySupportSetsRejectDuplicatesInStandalonePackage(t *testing.T) {
	tests := []struct {
		name   string
		change func(*plugin.Capability)
	}{
		{name: "actions", change: func(c *plugin.Capability) {
			c.Actions = []deployment.ActionKind{deployment.ActionTrafficShift, deployment.ActionTrafficShift}
		}},
		{name: "observations", change: func(c *plugin.Capability) {
			c.Observations = []deployment.ObservationKind{deployment.ObservationDatabaseState, deployment.ObservationDatabaseState}
		}},
		{name: "evidence", change: func(c *plugin.Capability) {
			c.Evidence = []deployment.EvidenceKind{deployment.EvidenceRolloutHealth, deployment.EvidenceRolloutHealth}
		}},
		{name: "databaseProviders", change: func(c *plugin.Capability) {
			c.DatabaseProviders = []environment.DatabaseProvider{environment.ProviderSQLite, environment.ProviderSQLite}
		}},
		{name: "migrationTools", change: func(c *plugin.Capability) {
			c.MigrationTools = []migration.Tool{migration.ToolAtlas, migration.ToolAtlas}
		}},
		{name: "features", change: func(c *plugin.Capability) {
			c.Features = []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureSQLAnalysis}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			definition := artifactFreePackageSource()
			tc.change(&definition.Provides[0])
			if err := plugin.ValidateUniqueCapabilitySupports(definition.Provides[0]); err == nil || !strings.Contains(err.Error(), tc.name) {
				t.Fatalf("missing duplicate support diagnostic for %s: %v", tc.name, err)
			}
			if _, err := plugin.MaterializePluginPackage(definition, pluginPackagePlan(definition)); err == nil || !strings.Contains(err.Error(), "more than once") {
				t.Fatalf("direct package accepted duplicate %s: %v", tc.name, err)
			}
		})
	}
}

func TestCapabilitySupportDuplicateDiagnosisDoesNotDependOnOrder(t *testing.T) {
	source := artifactFreePackageSource().Provides[0]
	source.Actions = []deployment.ActionKind{
		deployment.ActionTrafficShift,
		deployment.ActionDatabaseGate,
		deployment.ActionTrafficShift,
		deployment.ActionDatabaseGate,
	}
	reversed := source
	reversed.Actions = []deployment.ActionKind{
		deployment.ActionDatabaseGate,
		deployment.ActionTrafficShift,
		deployment.ActionDatabaseGate,
		deployment.ActionTrafficShift,
	}
	err1 := plugin.ValidateUniqueCapabilitySupports(source)
	err2 := plugin.ValidateUniqueCapabilitySupports(reversed)
	if err1 == nil || err2 == nil || err1.Error() != err2.Error() || !strings.Contains(err1.Error(), "database-gate") {
		t.Fatalf("duplicate diagnostic varies with declaration order: %v vs %v", err1, err2)
	}
}

func TestCapabilitySupportDistinctOrderingPreservesFingerprintAndPackage(t *testing.T) {
	a := artifactFreePackageSource()
	a.Provides[0].MigrationTools = []migration.Tool{migration.ToolLiquibase, migration.ToolAtlas}
	a.Provides[0].Features = []environment.Feature{environment.FeatureDatabaseWorkbench, environment.FeatureSQLAnalysis}
	b := artifactFreePackageSource()
	b.Provides[0].MigrationTools = []migration.Tool{migration.ToolAtlas, migration.ToolLiquibase}
	b.Provides[0].Features = []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureDatabaseWorkbench}
	if plugin.BuildDefinitionFingerprint(a) != plugin.BuildDefinitionFingerprint(b) {
		t.Fatal("reordered distinct support sets changed fingerprint")
	}
	p1, e1 := plugin.MaterializePluginPackage(a, pluginPackagePlan(a))
	p2, e2 := plugin.MaterializePluginPackage(b, pluginPackagePlan(b))
	if e1 != nil || e2 != nil || !bytes.Equal(p1, p2) {
		t.Fatalf("reordered distinct support sets changed package: %v / %v", e1, e2)
	}
}

func TestCapabilitySupportRejectsUnknownMigrationToolAndFeature(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*plugin.Capability)
		reject string
	}{
		{name: "migrationTools", change: func(c *plugin.Capability) {
			c.MigrationTools = []migration.Tool{"unknown"}
		}, reject: "invalid migration tool"},
		{name: "features", change: func(c *plugin.Capability) {
			c.Features = []environment.Feature{"unknown"}
		}, reject: "invalid feature"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := artifactFreePackageSource()
			tc.change(&definition.Provides[0])
			if _, err := plugin.MaterializePluginPackage(definition, pluginPackagePlan(definition)); err == nil || !strings.Contains(err.Error(), tc.reject) {
				t.Fatalf("standalone package failed to reject invalid %s: %v", tc.name, err)
			}
		})
	}
}

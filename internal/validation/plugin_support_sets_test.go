package validation_test

import (
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldRejectsDuplicateCapabilitySupportSetValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*plugin.Capability)
	}{
		{name: "actions", change: func(c *plugin.Capability) {
			c.Actions = []deployment.ActionKind{deployment.ActionDatabaseGate, deployment.ActionDatabaseGate}
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
			c.MigrationTools = []migration.Tool{migration.ToolNative, migration.ToolNative}
		}},
		{name: "features", change: func(c *plugin.Capability) {
			c.Features = []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureSQLAnalysis}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := referenceWorld()
			c := plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.duplicates")}
			tc.change(&c)
			w.Plugins[0].Provides = []plugin.Capability{c}
			found := false
			for _, d := range validation.Validate(w) {
				if d.Code == "DUPLICATE_PLUGIN_CAPABILITY_SUPPORT" && d.Severity == validation.Error {
					found = true
				}
			}
			if !found {
				t.Fatalf("world failed to reject repeated %s value", tc.name)
			}
		})
	}
}

func TestWorldRejectsUnknownMigrationToolsAndFeatures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*plugin.Capability)
		code   string
	}{
		{name: "migrationTools", code: "INVALID_PLUGIN_MIGRATION_TOOL_SUPPORT", change: func(c *plugin.Capability) {
			c.MigrationTools = []migration.Tool{"unrecognized"}
		}},
		{name: "features", code: "INVALID_PLUGIN_FEATURE_SUPPORT", change: func(c *plugin.Capability) {
			c.Features = []environment.Feature{"unrecognized"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := referenceWorld()
			c := plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.unknowns")}
			tc.change(&c)
			w.Plugins[0].Provides = []plugin.Capability{c}
			found := false
			for _, d := range validation.Validate(w) {
				if d.Code == tc.code && d.Severity == validation.Error {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing world diagnostic %s", tc.code)
			}
		})
	}
}

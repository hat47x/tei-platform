package validation_test

import (
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldRejectsRepeatedCapabilitySupportSets(t *testing.T) {
	cases := []struct {
		name   string
		change func(*plugin.Capability)
		code   string
	}{
		{"actions", func(c *plugin.Capability) { c.Actions = []deployment.ActionKind{deployment.ActionTrafficShift, deployment.ActionTrafficShift} }, "DUPLICATE_PLUGIN_SUPPORT"},
		{"observations", func(c *plugin.Capability) { c.Observations = []deployment.ObservationKind{deployment.ObservationDatabaseState, deployment.ObservationDatabaseState} }, "DUPLICATE_PLUGIN_SUPPORT"},
		{"evidence", func(c *plugin.Capability) { c.Evidence = []deployment.EvidenceKind{deployment.EvidenceRolloutHealth, deployment.EvidenceRolloutHealth} }, "DUPLICATE_PLUGIN_SUPPORT"},
		{"database providers", func(c *plugin.Capability) { c.DatabaseProviders = []environment.DatabaseProvider{environment.ProviderSQLite, environment.ProviderSQLite} }, "DUPLICATE_PLUGIN_SUPPORT"},
		{"migration tools", func(c *plugin.Capability) { c.MigrationTools = []migration.Tool{migration.ToolAtlas, migration.ToolAtlas} }, "DUPLICATE_PLUGIN_SUPPORT"},
		{"features", func(c *plugin.Capability) { c.Features = []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureSQLAnalysis} }, "DUPLICATE_PLUGIN_SUPPORT"},
		{"invalid migration tool", func(c *plugin.Capability) { c.MigrationTools = []migration.Tool{"unknown"} }, "INVALID_PLUGIN_MIGRATION_TOOL_SUPPORT"},
		{"invalid feature", func(c *plugin.Capability) { c.Features = []environment.Feature{"editor.unknown"} }, "INVALID_PLUGIN_FEATURE_SUPPORT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := referenceWorld()
			tc.change(&world.Plugins[0].Provides[0])
			for _, diagnostic := range validation.Validate(world) {
				if diagnostic.Code == tc.code && diagnostic.Severity == validation.Error {
					return
				}
			}
			t.Fatalf("world validation did not report %s", tc.code)
		})
	}
}

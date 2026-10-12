package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
)

func TestDirectPackageRejectsDuplicateCapabilitySupportSets(t *testing.T) {
	tests := []struct {
		name string
		set  func(*plugin.Capability)
		reject string
	}{
		{"actions", func(c *plugin.Capability) { c.Actions = []deployment.ActionKind{deployment.ActionTrafficShift, deployment.ActionTrafficShift} }, "more than once"},
		{"observations", func(c *plugin.Capability) { c.Observations = []deployment.ObservationKind{deployment.ObservationDatabaseState, deployment.ObservationDatabaseState} }, "more than once"},
		{"evidence", func(c *plugin.Capability) { c.Evidence = []deployment.EvidenceKind{deployment.EvidenceRolloutHealth, deployment.EvidenceRolloutHealth} }, "more than once"},
		{"database providers", func(c *plugin.Capability) { c.DatabaseProviders = []environment.DatabaseProvider{environment.ProviderSQLite, environment.ProviderSQLite} }, "more than once"},
		{"migration tools", func(c *plugin.Capability) { c.MigrationTools = []migration.Tool{migration.ToolAtlas, migration.ToolAtlas} }, "more than once"},
		{"features", func(c *plugin.Capability) { c.Features = []environment.Feature{environment.FeatureSQLAnalysis, environment.FeatureSQLAnalysis} }, "more than once"},
		{"unknown migration tool", func(c *plugin.Capability) { c.MigrationTools = []migration.Tool{"unknown"} }, "invalid migration tool"},
		{"unknown feature", func(c *plugin.Capability) { c.Features = []environment.Feature{"editor.unknown"} }, "invalid feature"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			definition := artifactFreePackageSource()
			tc.set(&definition.Provides[0])
			_, err := plugin.MaterializePluginPackage(definition, pluginPackagePlan(definition))
			if err == nil || !strings.Contains(err.Error(), tc.reject) {
				t.Fatalf("expected %q, got %v", tc.reject, err)
			}
		})
	}
}

func TestCapabilitySupportDuplicateDiagnosticIgnoresOrder(t *testing.T) {
	a := deployment.ActionApplicationStart
	z := deployment.ActionTrafficShift
	first := []deployment.ActionKind{z, a, z, a}
	second := []deployment.ActionKind{a, z, a, z}
	errA := plugin.ValidateUniqueCapabilitySupports("actions", first)
	errB := plugin.ValidateUniqueCapabilitySupports("actions", second)
	if errA == nil || errB == nil || errA.Error() != errB.Error() || !strings.Contains(errA.Error(), string(a)) {
		t.Fatalf("duplicate target depends on order: %v and %v", errA, errB)
	}
}

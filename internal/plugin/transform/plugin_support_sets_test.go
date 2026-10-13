package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/deployment"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestCapabilitySupportSetsAreRejectedByBothBuildPaths(t *testing.T) {
	tests := []struct {
		name   string
		change func(*plugin.Definition)
	}{
		{name: "actions", change: func(d *plugin.Definition) {
			d.Provides[1].Actions = append(d.Provides[1].Actions, d.Provides[1].Actions[0])
		}},
		{name: "observations", change: func(d *plugin.Definition) {
			d.Provides[0].Observations = append(d.Provides[0].Observations, d.Provides[0].Observations[0])
		}},
		{name: "evidence", change: func(d *plugin.Definition) {
			d.Provides[0].Evidence = append(d.Provides[0].Evidence, d.Provides[0].Evidence[0])
		}},
		{name: "databaseProviders", change: func(d *plugin.Definition) {
			d.Provides[0].DatabaseProviders = append(d.Provides[0].DatabaseProviders, d.Provides[0].DatabaseProviders[0])
		}},
		{name: "migrationTools", change: func(d *plugin.Definition) {
			d.Provides[0].MigrationTools = append(d.Provides[0].MigrationTools, d.Provides[0].MigrationTools[0])
		}},
		{name: "features", change: func(d *plugin.Definition) {
			d.Provides[0].Features = append(d.Provides[0].Features, d.Provides[0].Features[0])
		}},
	}
	source := referencePluginPackageTransformSource(t)
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def, plan := transformPackageFixture()
			tc.change(&def)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(def)
			if _, err := plugin.MaterializePluginPackage(def, plan); err == nil || !strings.Contains(err.Error(), "more than once") {
				t.Fatalf("Go package accepted duplicate %s: %v", tc.name, err)
			}
			_, err := transform.Evaluate(source, map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, def),
			})
			if err == nil || !strings.Contains(err.Error(), "duplicate key") {
				t.Fatalf("TEI package accepted duplicate %s: %v", tc.name, err)
			}
		})
	}
}

func TestMigrationToolAndFeatureValidityMatchesAcrossBuildPaths(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*plugin.Definition)
	}{
		{name: "invalid migration tool", change: func(d *plugin.Definition) {
			d.Provides[0].MigrationTools = []migration.Tool{"unknown"}
		}},
		{name: "invalid feature", change: func(d *plugin.Definition) {
			d.Provides[0].Features = []environment.Feature{"unknown"}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			def, plan := transformPackageFixture()
			tc.change(&def)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(def)
			if _, err := plugin.MaterializePluginPackage(def, plan); err == nil {
				t.Fatal("Go builder accepted unknown support value")
			}
			if _, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, def),
			}); err == nil || !strings.Contains(err.Error(), "assert-enum failed") {
				t.Fatalf("TEI builder accepted unknown support value: %v", err)
			}
		})
	}
}

func TestDistinctReorderedSupportsKeepGoTEIByteIdentity(t *testing.T) {
	source := referencePluginPackageTransformSource(t)
	var baseline []byte
	for _, reverse := range []bool{false, true} {
		def, plan := transformPackageFixture()
		if reverse {
			c := &def.Provides[1]
			c.Actions = []deployment.ActionKind{deployment.ActionApplicationStart, deployment.ActionTrafficShift}
			observation := &def.Provides[0]
			observation.DatabaseProviders[0], observation.DatabaseProviders[1] = observation.DatabaseProviders[1], observation.DatabaseProviders[0]
			observation.MigrationTools[0], observation.MigrationTools[1] = observation.MigrationTools[1], observation.MigrationTools[0]
			observation.Features[0], observation.Features[1] = observation.Features[1], observation.Features[0]
		}
		plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(def)
		want, err := plugin.MaterializePluginPackage(def, plan)
		if err != nil {
			t.Fatalf("Go materializer rejected valid supports: %v", err)
		}
		result, err := transform.Evaluate(source, map[string]transform.Value{
			"buildPlan":        valueFromGoJSON(t, plan),
			"pluginDefinition": valueFromGoJSON(t, def),
		})
		if err != nil || result.Output.Kind != transform.ValueBytes || !bytes.Equal(result.Output.Bytes, want) {
			t.Fatalf("reordered valid support values changed Go/TEI equivalence: %v", err)
		}
		if baseline != nil && !bytes.Equal(want, baseline) {
			t.Fatal("declaration ordering changed valid package bytes")
		}
		baseline = want
	}
}

func TestMissingAndEmptySupportListsRemainValid(t *testing.T) {
	for _, field := range []string{"actions", "observations", "evidence", "databaseProviders", "migrationTools", "features"} {
		t.Run(field, func(t *testing.T) {
			def, plan := transformPackageFixture()
			c := &def.Provides[0]
			switch field {
			case "actions":
				c.Actions = nil
			case "observations":
				c.Observations = nil
			case "evidence":
				c.Evidence = nil
			case "databaseProviders":
				c.DatabaseProviders = nil
			case "migrationTools":
				c.MigrationTools = nil
			case "features":
				c.Features = nil
			}
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(def)
			want, err := plugin.MaterializePluginPackage(def, plan)
			if err != nil {
				t.Fatalf("Go builder rejected missing support field: %v", err)
			}
			result, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, def),
			})
			if err != nil || result.Output.Kind != transform.ValueBytes || !bytes.Equal(want, result.Output.Bytes) {
				t.Fatalf("TEI builder differs on missing support field %s: %v", field, err)
			}
		})
	}
}

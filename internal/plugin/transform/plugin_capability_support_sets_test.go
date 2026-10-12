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

func TestPluginCapabilitySupportSetsGoTEIParity(t *testing.T) {
	cases := []struct {
		name   string
		change func(*plugin.Definition)
		reject string
	}{
		{"duplicate actions", func(d *plugin.Definition) { d.Provides[1].Actions = append(d.Provides[1].Actions, d.Provides[1].Actions[0]) }, "duplicate key"},
		{"duplicate observations", func(d *plugin.Definition) { d.Provides[0].Observations = append(d.Provides[0].Observations, d.Provides[0].Observations[0]) }, "duplicate key"},
		{"duplicate evidence", func(d *plugin.Definition) { d.Provides[0].Evidence = append(d.Provides[0].Evidence, d.Provides[0].Evidence[0]) }, "duplicate key"},
		{"duplicate database provider", func(d *plugin.Definition) { d.Provides[0].DatabaseProviders = append(d.Provides[0].DatabaseProviders, d.Provides[0].DatabaseProviders[0]) }, "duplicate key"},
		{"duplicate migration tool", func(d *plugin.Definition) { d.Provides[0].MigrationTools = append(d.Provides[0].MigrationTools, d.Provides[0].MigrationTools[0]) }, "duplicate key"},
		{"duplicate feature", func(d *plugin.Definition) { d.Provides[0].Features = append(d.Provides[0].Features, d.Provides[0].Features[0]) }, "duplicate key"},
		{"unknown migration tool", func(d *plugin.Definition) { d.Provides[0].MigrationTools = []migration.Tool{"unknown"} }, "assert-enum failed"},
		{"unknown feature", func(d *plugin.Definition) { d.Provides[0].Features = []environment.Feature{"editor.unknown"} }, "assert-enum failed"},
		{"valid reordering", func(d *plugin.Definition) {
			d.Provides[1].Actions[0], d.Provides[1].Actions[1] = d.Provides[1].Actions[1], d.Provides[1].Actions[0]
			d.Provides[0].MigrationTools[0], d.Provides[0].MigrationTools[1] = d.Provides[0].MigrationTools[1], d.Provides[0].MigrationTools[0]
			d.Provides[0].Features[0], d.Provides[0].Features[1] = d.Provides[0].Features[1], d.Provides[0].Features[0]
		}, ""},
		{"nil optional arrays", func(d *plugin.Definition) {
			d.Provides[0].MigrationTools = nil
			d.Provides[0].Features = nil
		}, ""},
		{"empty optional arrays", func(d *plugin.Definition) {
			d.Provides[0].MigrationTools = []migration.Tool{}
			d.Provides[0].Features = []environment.Feature{}
			d.Provides[1].Actions = []deployment.ActionKind{}
		}, ""},
	}
	source := referencePluginPackageTransformSource(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			tc.change(&definition)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			oracle, goErr := plugin.MaterializePluginPackage(definition, plan)
			result, teiErr := transform.Evaluate(source, map[string]transform.Value{
				"buildPlan": valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if tc.reject != "" {
				if goErr == nil || teiErr == nil || !strings.Contains(teiErr.Error(), tc.reject) {
					t.Fatalf("input acceptance diverged: Go=%v, TEI=%v", goErr, teiErr)
				}
				return
			}
			if goErr != nil || teiErr != nil {
				t.Fatalf("valid definition rejected: Go=%v, TEI=%v", goErr, teiErr)
			}
			if result.Output.Kind != transform.ValueBytes || !bytes.Equal(result.Output.Bytes, oracle) {
				t.Fatal("Go/TEI package bytes differ")
			}
		})
	}
}

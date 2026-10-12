package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestRequirementEntryValidationParityBetweenGoAndTEI(t *testing.T) {
	cases := []struct {
		name   string
		change func(*plugin.Definition)
		reject string
	}{
		{name: "zero major", change: func(d *plugin.Definition) { d.Requires[0].CompatibleMajor = 0 }, reject: "positive-integer"},
		{name: "negative major", change: func(d *plugin.Definition) { d.Requires[0].CompatibleMajor = -1 }, reject: "positive-integer"},
		{name: "empty id", change: func(d *plugin.Definition) { d.Requires[0].Plugin = "" }, reject: "canonical-id"},
		{name: "invalid id", change: func(d *plugin.Definition) { d.Requires[0].Plugin = core.CanonicalID("bad id") }, reject: "canonical-id"},
		{name: "whitespace id", change: func(d *plugin.Definition) { d.Requires[0].Plugin = core.CanonicalID(" tei.base ") }, reject: "canonical-id"},
		{name: "valid dependency order", change: func(d *plugin.Definition) {
			d.Requires[0], d.Requires[1] = d.Requires[1], d.Requires[0]
		}},
		{name: "empty dependencies", change: func(d *plugin.Definition) { d.Requires = []plugin.Requirement{} }},
		{name: "nil dependencies", change: func(d *plugin.Definition) { d.Requires = nil }},
	}
	source := referencePluginPackageTransformSource(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			tc.change(&definition)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			want, goErr := plugin.MaterializePluginPackage(definition, plan)
			got, teiErr := transform.Evaluate(source, map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if tc.reject != "" {
				if goErr == nil || teiErr == nil || !strings.Contains(teiErr.Error(), tc.reject) {
					t.Fatalf("invalid requirement accepted or wrong rejection: Go=%v TEI=%v", goErr, teiErr)
				}
				return
			}
			if goErr != nil || teiErr != nil {
				t.Fatalf("valid requirement rejected: Go=%v TEI=%v", goErr, teiErr)
			}
			if got.Output.Kind != transform.ValueBytes || !bytes.Equal(want, got.Output.Bytes) {
				t.Fatal("Go and TEI package bytes differ for valid requirements")
			}
		})
	}
}

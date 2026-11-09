package transform_test

import (
	"bytes"
	"testing"

	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestPluginPackageWithoutCapabilitiesMatchesGo(t *testing.T) {
	var outputs [][]byte
	for _, tc := range []struct {
		name     string
		provides []plugin.Capability
	}{
		{name: "nil capabilities", provides: nil},
		{name: "empty capabilities", provides: []plugin.Capability{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			definition.Provides = tc.provides
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)

			want, err := plugin.MaterializePluginPackage(definition, plan)
			if err != nil {
				t.Fatalf("Go package builder rejected empty capabilities: %v", err)
			}
			got, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if err != nil {
				t.Fatalf("TEI package builder rejected empty capabilities: %v", err)
			}
			if got.Output.Kind != transform.ValueBytes || !bytes.Equal(got.Output.Bytes, want) {
				t.Fatalf("empty capabilities changed Go/TEI package equivalence:\nGo: %s\nTEI: %s", want, got.Output.Bytes)
			}
			if bytes.Contains(want, []byte("\"provides\"")) {
				t.Fatal("omitempty capabilities must remain absent from package bytes")
			}
			outputs = append(outputs, append([]byte(nil), want...))
		})
	}
	if len(outputs) != 2 || !bytes.Equal(outputs[0], outputs[1]) {
		t.Fatal("nil and empty capability slices must produce identical package bytes")
	}
}

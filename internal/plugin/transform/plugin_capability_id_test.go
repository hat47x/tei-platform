package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestCapabilityIDValidationMatchesGoAndTEIBuilders(t *testing.T) {
	source := referencePluginPackageTransformSource(t)
	for _, tc := range []struct {
		name   string
		id     string
		reject bool
	}{
		{name: "valid canonical id", id: "runtime.valid"},
		{name: "empty id", id: "", reject: true},
		{name: "invalid id", id: "not a valid id", reject: true},
		{name: "whitespace-padded id", id: " runtime.valid ", reject: true},
		{name: "uppercase id", id: "Runtime.valid", reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			definition.Provides[0].ID = core.CanonicalID(tc.id)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)

			want, goErr := plugin.MaterializePluginPackage(definition, plan)
			got, teiErr := transform.Evaluate(source, map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if tc.reject {
				if goErr == nil || !strings.Contains(goErr.Error(), "invalid canonical id") {
					t.Fatalf("Go materializer accepted invalid id %q: %v", tc.id, goErr)
				}
				if teiErr == nil || !strings.Contains(teiErr.Error(), "assert-canonical-id") {
					t.Fatalf("TEI Source accepted invalid id %q: %v", tc.id, teiErr)
				}
				return
			}
			if goErr != nil || teiErr != nil {
				t.Fatalf("valid id %q rejected: Go=%v TEI=%v", tc.id, goErr, teiErr)
			}
			if got.Output.Kind != transform.ValueBytes || !bytes.Equal(got.Output.Bytes, want) {
				t.Fatal("Go and TEI materializers disagree for valid Capability ID")
			}
		})
	}
}

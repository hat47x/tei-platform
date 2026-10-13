package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/deployment"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestPluginPackageTransformRejectsRepeatedCapabilityIdentity(t *testing.T) {
	definition, plan := transformPackageFixture()
	duplicate := definition.Provides[0]
	duplicate.Observations = []deployment.ObservationKind{deployment.ObservationApplicationRuntime}
	definition.Provides = append(definition.Provides, duplicate)
	plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)

	if _, err := plugin.MaterializePluginPackage(definition, plan); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("Go package builder accepted repeated capability: %v", err)
	}
	_, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": valueFromGoJSON(t, definition),
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate key") {
		t.Fatalf("TEI package builder accepted repeated capability: %v", err)
	}
}

func TestPluginPackageAllowsSharedCapabilityIDAcrossDifferentKinds(t *testing.T) {
	definition, plan := transformPackageFixture()
	definition.Provides[1].ID = definition.Provides[0].ID
	plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)

	want, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		t.Fatal(err)
	}
	got, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": valueFromGoJSON(t, definition),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Output.Kind != transform.ValueBytes || !bytes.Equal(got.Output.Bytes, want) {
		t.Fatal("valid shared capability ID changed Go/TEI package equivalence")
	}
}

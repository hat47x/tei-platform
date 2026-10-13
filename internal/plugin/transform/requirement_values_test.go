package transform_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestRequirementValuesRejectedByGoAndTEITransform(t *testing.T) {
	tests := []struct {
		name     string
		req      plugin.Requirement
		goError  string
		teiError string
	}{
		{"empty ID", plugin.Requirement{Plugin: "", CompatibleMajor: 1}, "invalid canonical id", "assert-canonical-id"},
		{"invalid ID", plugin.Requirement{Plugin: core.CanonicalID("not a valid id"), CompatibleMajor: 1}, "invalid canonical id", "assert-canonical-id"},
		{"padded ID", plugin.Requirement{Plugin: core.CanonicalID(" tei.base "), CompatibleMajor: 1}, "invalid canonical id", "assert-canonical-id"},
		{"zero major", plugin.Requirement{Plugin: core.MustCanonicalID("tei.base"), CompatibleMajor: 0}, "invalid compatible major", "assert-positive-integer"},
		{"negative major", plugin.Requirement{Plugin: core.MustCanonicalID("tei.base"), CompatibleMajor: -1}, "invalid compatible major", "assert-positive-integer"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			definition.Requires = []plugin.Requirement{tc.req}
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			if _, err := plugin.MaterializePluginPackage(definition, plan); err == nil || !strings.Contains(err.Error(), tc.goError) {
				t.Fatalf("Go materializer: expected %q, got %v", tc.goError, err)
			}
			if _, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			}); err == nil || !strings.Contains(err.Error(), tc.teiError) {
				t.Fatalf("TEI Source materializer: expected %q, got %v", tc.teiError, err)
			}
		})
	}
}

func TestRequirementMajorMustBeIntegerLexemeInTEITransform(t *testing.T) {
	for _, invalid := range []string{"1.0", "1e0", "1e999", "9223372036854775808", "null", "\"1\""} {
		t.Run(invalid, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			definition.Requires = []plugin.Requirement{{Plugin: core.MustCanonicalID("tei.base"), CompatibleMajor: 1}}
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			raw, err := json.Marshal(definition)
			if err != nil {
				t.Fatal(err)
			}
			changed := bytes.Replace(raw, []byte(`"compatibleMajor":1`), []byte(`"compatibleMajor":`+invalid), 1)
			if bytes.Equal(raw, changed) {
				t.Fatal("missing major in source fixture")
			}
			badDefinition, err := transform.ParseJSON(changed)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": badDefinition,
			}); err == nil || !strings.Contains(err.Error(), "assert-positive-integer") {
				t.Fatalf("TEI transform must reject non-integer requirement %s: %v", invalid, err)
			}
		})
	}
}

func TestRequirementValuesPreserveEmptyAndValidPackageBytes(t *testing.T) {
	for _, requires := range [][]plugin.Requirement{
		nil,
		{},
		{{Plugin: core.MustCanonicalID("tei.base"), CompatibleMajor: 1}},
		{{Plugin: core.MustCanonicalID("tei.base"), CompatibleMajor: 1}, {Plugin: core.MustCanonicalID("tei.other"), CompatibleMajor: 2}},
	} {
		definition, plan := transformPackageFixture()
		definition.Requires = requires
		plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
		oracle, err := plugin.MaterializePluginPackage(definition, plan)
		if err != nil {
			t.Fatalf("Go materializer %v: %v", requires, err)
		}
		result, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
			"buildPlan":        valueFromGoJSON(t, plan),
			"pluginDefinition": valueFromGoJSON(t, definition),
		})
		if err != nil {
			t.Fatalf("TEI materializer %v: %v", requires, err)
		}
		if !bytes.Equal(result.Output.Bytes, oracle) {
			t.Fatalf("TEI/Go bytes diverged for %v", requires)
		}
	}
}

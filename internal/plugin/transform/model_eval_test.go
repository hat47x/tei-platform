package transform_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestPluginPackageTransformRejectsAuthorityAndIdentityMismatch(t *testing.T) {
	tests := []struct {
		name string
		edit func(*plugin.Definition, *plugin.BuildPlan)
	}{
		{"runtime output", func(_ *plugin.Definition, plan *plugin.BuildPlan) {
			plan.OutputKind = plugin.BuildOutputRuntimeArtifact
		}},
		{"plugin identity", func(_ *plugin.Definition, plan *plugin.BuildPlan) {
			plan.Plugin.ID = core.MustCanonicalID("tei.other")
		}},
		{"definition fingerprint", func(_ *plugin.Definition, plan *plugin.BuildPlan) {
			plan.PluginDefinitionFingerprint = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
		}},
		{"source artifact", func(definition *plugin.Definition, plan *plugin.BuildPlan) {
			definition.Artifact.Digest = transformDigestA
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(*definition)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			tt.edit(&definition, &plan)
			_, err := transform.Evaluate(transform.PluginPackageSourceV1(), map[string]transform.Value{
				"buildPlan":        valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if err == nil {
				t.Fatal("expected transform evaluation to fail")
			}
		})
	}
}

func TestTransformRejectsUndeclaredInputAndBudgetExhaustion(t *testing.T) {
	source := transform.PluginPackageSourceV1()
	definition, plan := transformPackageFixture()
	inputs := map[string]transform.Value{
		"buildPlan":        valueFromGoJSON(t, plan),
		"pluginDefinition": valueFromGoJSON(t, definition),
		"ambient":          transform.String("not-declared"),
	}
	if _, err := transform.Evaluate(source, inputs); err == nil {
		t.Fatal("expected undeclared input to fail")
	}

	source.Limits.MaxSteps = 1
	delete(inputs, "ambient")
	if _, err := transform.Evaluate(source, inputs); err == nil {
		t.Fatal("expected maxSteps exhaustion")
	}
}

func TestOrderedValuePreservesFieldOrderAndRejectsDuplicateKeys(t *testing.T) {
	value, err := transform.ParseJSON([]byte("{\"b\":1,\"a\":2}"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := value.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{\"b\":1,\"a\":2}" {
		t.Fatalf("field order changed: %s", raw)
	}
	if _, err := transform.ParseJSON([]byte("{\"a\":1,\"a\":2}")); err == nil {
		t.Fatal("expected duplicate JSON object key to fail")
	}
}

func TestTransformSourceFingerprintCanonicalizesLiteralJSON(t *testing.T) {
	a := transform.PluginPackageSourceV1()
	b := transform.PluginPackageSourceV1()
	b.Program.Assertions[0].Expected.Literal = json.RawMessage(" 1 ")

	aFP, err := transform.SourceFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	bFP, err := transform.SourceFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if aFP != bFP {
		t.Fatalf("equivalent literal JSON changed Source fingerprint: %s != %s", aFP, bFP)
	}
}

func TestTransformSourceMaterializationIsDeterministic(t *testing.T) {
	source := transform.PluginPackageSourceV1()
	first, firstDigest, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	second, secondDigest, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) || firstDigest != secondDigest {
		t.Fatal("transform Source realization is not deterministic")
	}
	if len(first) == 0 || first[len(first)-1] != '\n' {
		t.Fatal("transform Source realization must end with newline")
	}
}

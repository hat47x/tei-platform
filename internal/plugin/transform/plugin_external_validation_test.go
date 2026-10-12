package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestGoAndTEIPackageBuildersAgreeOnExternalRequirements(t *testing.T) {
	type testCase struct {
		name string
		change func(*plugin.Definition)
		reject string
	}
	cases := []testCase{
		{name: "valid reordered inputs", change: func(d *plugin.Definition) {
			d.External[0], d.External[1] = d.External[1], d.External[0]
		}},
		{name: "same name in another scope", change: func(d *plugin.Definition) {
			d.External = append(d.External, plugin.ExternalRequirement{Scope: "test", Kind: "toolchain", Name: "a"})
		}},
		{name: "duplicate tuple", change: func(d *plugin.Definition) {
			d.External = append(d.External, d.External[0])
		}, reject: "duplicate key"},
		{name: "unsupported scope", change: func(d *plugin.Definition) {
			d.External[0].Scope = "something-else"
		}, reject: "assert-enum failed"},
		{name: "unsupported kind", change: func(d *plugin.Definition) {
			d.External[0].Kind = "something-else"
		}, reject: "assert-enum failed"},
		{name: "empty name", change: func(d *plugin.Definition) {
			d.External[0].Name = ""
		}, reject: "assert-nonblank failed"},
		{name: "whitespace name", change: func(d *plugin.Definition) {
			d.External[0].Name = " \t "
		}, reject: "assert-nonblank failed"},
	}
	source := referencePluginPackageTransformSource(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			tc.change(&definition)
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			goOutput, goErr := plugin.MaterializePluginPackage(definition, plan)
			got, teiErr := transform.Evaluate(source, map[string]transform.Value{
				"buildPlan": valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if tc.reject != "" {
				if goErr == nil || teiErr == nil || !strings.Contains(teiErr.Error(), tc.reject) {
					t.Fatalf("malformed inputs must fail in both builders (Go %v; TEI %v)", goErr, teiErr)
				}
				return
			}
			if goErr != nil || teiErr != nil {
				t.Fatalf("valid inputs rejected (Go %v; TEI %v)", goErr, teiErr)
			}
			if got.Output.Kind != transform.ValueBytes || !bytes.Equal(got.Output.Bytes, goOutput) {
				t.Fatal("valid external requirements changed package equivalence")
			}
		})
	}
}

func TestAssertNonBlankStringSemantics(t *testing.T) {
	source := uniqueByTestSource()
	source.Inputs = []transform.InputSpec{{Name: "name", Kind: transform.InputString}}
	input := transform.Expr{Op: transform.OpInput, Name: "name"}
	source.Program.Assertions = []transform.Expr{{Op: transform.OpAssertNonBlank, From: &input}}
	source.Program.Output = input

	for _, tc := range []struct {
		value string
		valid bool
	}{
		{value: "a", valid: true},
		{value: " a ", valid: true},
		{value: "", valid: false},
		{value: "\t \n", valid: false},
	} {
		_, err := transform.Evaluate(source, map[string]transform.Value{"name": transform.String(tc.value)})
		if (err == nil) != tc.valid {
			t.Fatalf("assert-nonblank value %q valid=%t, error=%v", tc.value, tc.valid, err)
		}
	}
	invalidSource := source
	invalidSource.Program.Assertions = []transform.Expr{{Op: transform.OpAssertNonBlank}}
	if err := invalidSource.Validate(); err == nil {
		t.Fatal("missing operand was accepted")
	}
}

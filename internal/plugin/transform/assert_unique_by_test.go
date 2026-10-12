package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func uniqueByTestSource() transform.Source {
	array := transform.Expr{Op: transform.OpInput, Name: "dependencies"}
	member := transform.Expr{Op: transform.OpInput, Name: "dependency"}
	return transform.Source{
		FormatVersion: transform.SourceFormatVersion,
		Identity: transform.SourceIdentity{
			ID: core.MustCanonicalID("tei.test.requirement_uniqueness"),
			Version: "1.0.0",
			ContractVersion: "tei.plugin-transform/v1",
		},
		Profile: transform.ProfileV1,
		Limits: transform.Limits{
			MaxSteps: 1000,
			MaxCollection: 100,
			MaxInputNodes: 1000,
			MaxInputBytes: 100000,
			MaxOutputBytes: 100000,
			MaxNestingDepth: 32,
		},
		Inputs: []transform.InputSpec{{Name: "dependencies", Kind: transform.InputArray}},
		Program: transform.Program{
			Assertions: []transform.Expr{{
				Op: transform.OpAssertUniqueBy,
				From: &array,
				Item: "dependency",
				Keys: []transform.Expr{{Op: transform.OpSelectField, From: &member, Field: "plugin"}},
			}},
			Output: array,
		},
	}
}

func TestAssertUniqueByValueAndFailureSemantics(t *testing.T) {
	cases := []struct {
		name string
		records any
		wantError string
	}{
		{name: "distinct keys", records: []any{map[string]any{"plugin": "tei.a"}, map[string]any{"plugin": "tei.b"}}},
		{name: "empty collection", records: []any{}},
		{name: "same key", records: []any{map[string]any{"plugin": "tei.a"}, map[string]any{"plugin": "tei.a"}}, wantError: "duplicate key"},
		{name: "separated duplicate key", records: []any{map[string]any{"plugin": "tei.a"}, map[string]any{"plugin": "tei.b"}, map[string]any{"plugin": "tei.a"}}, wantError: "duplicate key"},
		{name: "nonstring key", records: []any{map[string]any{"plugin": 1}}, wantError: "key must be string"},
		{name: "missing key", records: []any{map[string]any{"other": "tei.a"}}, wantError: "required field"},
		{name: "nonobject entry", records: []any{"tei.a"}, wantError: "select-field requires object"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := valueFromGoJSON(t, tc.records)
			result, err := transform.Evaluate(uniqueByTestSource(), map[string]transform.Value{
				"dependencies": input,
			})
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected %q, got %v", tc.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Output.Kind != transform.ValueArray {
				t.Fatalf("output kind %s, want array", result.Output.Kind)
			}
			got, err := result.Output.JSON()
			if err != nil {
				t.Fatal(err)
			}
			want, err := input.JSON()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("uniqueness assertion mutated a valid input: got %s, want %s", got, want)
			}
		})
	}
}

func TestAssertUniqueByChargesStringHashWork(t *testing.T) {
	source := uniqueByTestSource()
	// A tiny output prevents the output serializer, rather than key hashing,
	// from being responsible for exhausting the budget.
	source.Program.Output = transform.Expr{Op: transform.OpLiteral, Literal: []byte(`"ok"`)}
	source.Limits.MaxSteps = 12
	if _, err := transform.Evaluate(source, map[string]transform.Value{
		"dependencies": valueFromGoJSON(t, []any{map[string]any{"plugin": "a"}}),
	}); err != nil {
		t.Fatalf("short unique key must fit the same step budget: %v", err)
	}
	_, err := transform.Evaluate(source, map[string]transform.Value{
		"dependencies": valueFromGoJSON(t, []any{
			map[string]any{"plugin": strings.Repeat("x", 40<<10)},
		}),
	})
	if err == nil || !strings.Contains(err.Error(), "maxSteps") {
		t.Fatalf("expected oversized key hashing to consume the step budget, got %v", err)
	}
}

func TestPluginPackageWithoutRequirementsMatchesGoOracle(t *testing.T) {
	for _, tc := range []struct {
		name     string
		requires []plugin.Requirement
	}{
		{name: "nil slice", requires: nil},
		{name: "empty slice", requires: []plugin.Requirement{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition, plan := transformPackageFixture()
			definition.Requires = tc.requires
			plan.PluginDefinitionFingerprint = plugin.BuildDefinitionFingerprint(definition)
			want, err := plugin.MaterializePluginPackage(definition, plan)
			if err != nil {
				t.Fatal(err)
			}
			got, err := transform.Evaluate(referencePluginPackageTransformSource(t), map[string]transform.Value{
				"buildPlan": valueFromGoJSON(t, plan),
				"pluginDefinition": valueFromGoJSON(t, definition),
			})
			if err != nil {
				t.Fatalf("valid plugin with no dependencies was rejected: %v", err)
			}
			if got.Output.Kind != transform.ValueBytes || !bytes.Equal(got.Output.Bytes, want) {
				t.Fatalf("valid plugin without requirements diverged from Go package oracle")
			}
		})
	}
}

func TestAssertUniqueByOptionalCollection(t *testing.T) {
	source := uniqueByTestSource()
	definition := transform.Expr{Op: transform.OpInput, Name: "definition"}
	source.Inputs = []transform.InputSpec{{Name: "definition", Kind: transform.InputObject}}
	source.Program.Assertions[0].From = &transform.Expr{
		Op: transform.OpSelectField, From: &definition, Field: "requires", Optional: true,
	}
	source.Program.Assertions[0].Optional = true
	source.Program.Output = definition

	for _, tc := range []struct {
		name       string
		definition map[string]any
		wantError  string
	}{
		{name: "missing requirements", definition: map[string]any{}},
		{name: "null requirements", definition: map[string]any{"requires": nil}},
		{name: "empty requirements", definition: map[string]any{"requires": []any{}}},
		{name: "valid requirement", definition: map[string]any{"requires": []any{map[string]any{"plugin": "tei.a"}}}},
		{name: "nonarray requirement", definition: map[string]any{"requires": "tei.a"}, wantError: "requires array"},
		{name: "duplicate requirement", definition: map[string]any{
			"requires": []any{map[string]any{"plugin": "tei.a"}, map[string]any{"plugin": "tei.a"}},
		}, wantError: "duplicate key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := transform.Evaluate(source, map[string]transform.Value{
				"definition": valueFromGoJSON(t, tc.definition),
			})
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected %q, got %v", tc.wantError, err)
				}
			} else if err != nil {
				t.Fatalf("optional requirements were rejected: %v", err)
			}
		})
	}
}

func TestAssertUniqueByRejectsMalformedBinding(t *testing.T) {
	t.Run("missing key", func(t *testing.T) {
		source := uniqueByTestSource()
		source.Program.Assertions[0].Keys = nil
		if err := source.Validate(); err == nil {
			t.Fatal("assert-unique-by requires one key")
		}
	})
	t.Run("shadowing", func(t *testing.T) {
		source := uniqueByTestSource()
		source.Program.Assertions[0].Item = "dependencies"
		if err := source.Validate(); err == nil {
			t.Fatal("assert-unique-by may not shadow its input")
		}
	})
	t.Run("undeclared key input", func(t *testing.T) {
		source := uniqueByTestSource()
		source.Program.Assertions[0].Keys[0] = transform.Expr{Op: transform.OpInput, Name: "missing"}
		if err := source.Validate(); err == nil {
			t.Fatal("assert-unique-by may not access undeclared inputs")
		}
	})
}

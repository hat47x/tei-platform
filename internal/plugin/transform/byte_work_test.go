package transform_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin/transform"
)

func byteWorkSource(id string, maxSteps int, inputs []transform.InputSpec, output transform.Expr) transform.Source {
	return transform.Source{
		FormatVersion: transform.SourceFormatVersion,
		Identity: transform.SourceIdentity{
			ID:              core.MustCanonicalID(id),
			Version:         "1.0.0",
			ContractVersion: "tei.plugin-transform/v1",
		},
		Profile: transform.ProfileV1,
		Limits: transform.Limits{
			MaxSteps:        maxSteps,
			MaxCollection:   64,
			MaxInputNodes:   256,
			MaxInputBytes:   1 << 20,
			MaxOutputBytes:  1 << 20,
			MaxNestingDepth: 32,
		},
		Inputs:  inputs,
		Program: transform.Program{Output: output},
	}
}

func TestSHA256ConsumesByteLinearWorkBudget(t *testing.T) {
	input := transform.Expr{Op: transform.OpInput, Name: "payload"}
	source := byteWorkSource(
		"tei.builder.sha_work_test",
		6,
		[]transform.InputSpec{{Name: "payload", Kind: transform.InputBytes}},
		transform.Expr{Op: transform.OpSHA256, From: &input},
	)
	if _, err := transform.Evaluate(source, map[string]transform.Value{
		"payload": transform.Bytes(make([]byte, 16<<10)),
	}); err == nil {
		t.Fatal("expected SHA-256 byte work to exceed maxSteps")
	}
}

func TestCanonicalJSONConsumesByteLinearWorkBudget(t *testing.T) {
	input := transform.Expr{Op: transform.OpInput, Name: "value"}
	source := byteWorkSource(
		"tei.builder.json_work_test",
		6,
		[]transform.InputSpec{{Name: "value", Kind: transform.InputObject}},
		transform.Expr{Op: transform.OpCanonicalJSON, From: &input},
	)
	value, err := transform.Object(transform.Field{
		Name:  "payload",
		Value: transform.String(strings.Repeat("x", 8<<10)),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transform.Evaluate(source, map[string]transform.Value{"value": value}); err == nil {
		t.Fatal("expected canonical JSON byte work to exceed maxSteps")
	}
}

func TestAppendBytesConsumesByteLinearWorkBudget(t *testing.T) {
	input := transform.Expr{Op: transform.OpInput, Name: "payload"}
	source := byteWorkSource(
		"tei.builder.append_work_test",
		6,
		[]transform.InputSpec{{Name: "payload", Kind: transform.InputBytes}},
		transform.Expr{
			Op:     transform.OpAppendBytes,
			From:   &input,
			Suffix: strings.Repeat("s", 4<<10),
		},
	)
	if _, err := transform.Evaluate(source, map[string]transform.Value{
		"payload": transform.Bytes(make([]byte, 16<<10)),
	}); err == nil {
		t.Fatal("expected append byte copy work to exceed maxSteps")
	}
}

func TestAssertEqualConsumesByteComparisonWork(t *testing.T) {
	actual := transform.Expr{Op: transform.OpInput, Name: "actual"}
	expected := transform.Expr{Op: transform.OpInput, Name: "expected"}
	source := byteWorkSource(
		"tei.builder.bytes_equal_work_test",
		8,
		[]transform.InputSpec{
			{Name: "actual", Kind: transform.InputBytes},
			{Name: "expected", Kind: transform.InputBytes},
		},
		actual,
	)
	source.Program.Assertions = []transform.Expr{{
		Op:       transform.OpAssertEqual,
		Actual:   &actual,
		Expected: &expected,
	}}
	value := transform.Bytes(make([]byte, 16<<10))
	if _, err := transform.Evaluate(source, map[string]transform.Value{
		"actual":   value,
		"expected": value,
	}); err == nil {
		t.Fatal("expected byte equality work to exceed maxSteps")
	}
}

func TestSortKeyComparisonConsumesStringByteWork(t *testing.T) {
	items := transform.Expr{Op: transform.OpInput, Name: "items"}
	item := transform.Expr{Op: transform.OpInput, Name: "item"}
	source := byteWorkSource(
		"tei.builder.sort_string_work_test",
		8,
		[]transform.InputSpec{{Name: "items", Kind: transform.InputArray}},
		transform.Expr{
			Op:   transform.OpSortArrayBy,
			From: &items,
			Item: "item",
			Keys: []transform.Expr{item},
		},
	)
	if _, err := transform.Evaluate(source, map[string]transform.Value{
		"items": transform.Array(
			transform.String(strings.Repeat("a", 8<<10)),
			transform.String(strings.Repeat("b", 8<<10)),
		),
	}); err == nil {
		t.Fatal("expected long sort key comparison to exceed maxSteps")
	}
}

func TestByteWorkAccountingStillAllowsSmallDeterministicTransform(t *testing.T) {
	input := transform.Expr{Op: transform.OpInput, Name: "payload"}
	source := byteWorkSource(
		"tei.builder.small_byte_work_test",
		32,
		[]transform.InputSpec{{Name: "payload", Kind: transform.InputBytes}},
		transform.Expr{Op: transform.OpSHA256, From: &input},
	)
	result, err := transform.Evaluate(source, map[string]transform.Value{
		"payload": transform.Bytes([]byte("small")),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output.Kind != transform.ValueString || len(result.Output.String) != 64 {
		t.Fatalf("SHA-256 result: %#v", result.Output)
	}
}

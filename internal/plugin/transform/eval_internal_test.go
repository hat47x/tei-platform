package transform

import (
	"encoding/json"
	"testing"

	"tei/internal/core"
)

// This regression suite forward-ports the copy-amplification branch's
// observable ownership requirements to the current resource-metered evaluator.
// It deliberately uses Evaluate rather than retired helper functions.
func copyAmplificationSource(output Expr) Source {
	return Source{
		FormatVersion: SourceFormatVersion,
		Identity: SourceIdentity{
			ID:              core.MustCanonicalID("tei.reference.transform_copy"),
			Version:         "1.0.0",
			ContractVersion: "tei.reference.transform-copy/v1",
		},
		Profile: ProfileV1,
		Limits: Limits{
			MaxSteps:        10000,
			MaxCollection:   100,
			MaxInputNodes:   1000,
			MaxInputBytes:   8192,
			MaxOutputBytes:  8192,
			MaxNestingDepth: 20,
		},
		Inputs:  []InputSpec{{Name: "source", Kind: InputObject}},
		Program: Program{Output: output},
	}
}

func copyAmplificationInput() Value {
	return Value{Kind: ValueObject, Object: []Field{
		{Name: "replace", Value: String("old")},
		{Name: "items", Value: Value{Kind: ValueArray, Array: []Value{
			String("before"),
		}}},
	}}
}

func TestEvaluateSnapshotIsIndependentOfCallerMutation(t *testing.T) {
	input := copyAmplificationInput()
	result, err := Evaluate(
		copyAmplificationSource(Expr{Op: OpInput, Name: "source"}),
		map[string]Value{"source": input},
	)
	if err != nil {
		t.Fatal(err)
	}
	input.Object[1].Value.Array[0].String = "caller-changed"
	if got := result.Output.Object[1].Value.Array[0].String; got != "before" {
		t.Fatalf("result aliases caller-owned input after Evaluate: %q", got)
	}
	result.Output.Object[1].Value.Array[0].String = "result-changed"
	if got := input.Object[1].Value.Array[0].String; got != "caller-changed" {
		t.Fatalf("returned result aliases mutable caller input: %q", got)
	}
}

func TestEvaluateFieldSelectionIsIndependentOfCallerMutation(t *testing.T) {
	input := copyAmplificationInput()
	expr := Expr{
		Op:    OpSelectField,
		From:  &Expr{Op: OpInput, Name: "source"},
		Field: "items",
	}
	result, err := Evaluate(copyAmplificationSource(expr), map[string]Value{"source": input})
	if err != nil {
		t.Fatal(err)
	}
	input.Object[1].Value.Array[0].String = "caller-changed"
	if len(result.Output.Array) != 1 || result.Output.Array[0].String != "before" {
		t.Fatalf("selection leaked caller mutation: %#v", result.Output)
	}
}

func TestEvaluateReplaceFieldKeepsOriginalObjectUnchanged(t *testing.T) {
	input := copyAmplificationInput()
	expr := Expr{
		Op:    OpReplaceField,
		From:  &Expr{Op: OpInput, Name: "source"},
		Field: "replace",
		Value: &Expr{Op: OpLiteral, Literal: json.RawMessage(`"new"`)},
	}
	result, err := Evaluate(copyAmplificationSource(expr), map[string]Value{"source": input})
	if err != nil {
		t.Fatal(err)
	}
	if result.Output.Object[0].Value.String != "new" || input.Object[0].Value.String != "old" {
		t.Fatalf("replace mutated caller or lost replacement: original=%#v result=%#v", input, result.Output)
	}
	result.Output.Object[1].Name = "renamed"
	result.Output.Object[1].Value.Array[0].String = "changed"
	if input.Object[1].Name != "items" || input.Object[1].Value.Array[0].String != "before" {
		t.Fatal("returned replace output mutated caller-owned container or nested data")
	}
}

func TestEvaluateOmitFieldKeepsOriginalObjectUnchanged(t *testing.T) {
	input := copyAmplificationInput()
	expr := Expr{
		Op:    OpOmitField,
		From:  &Expr{Op: OpInput, Name: "source"},
		Field: "replace",
	}
	result, err := Evaluate(copyAmplificationSource(expr), map[string]Value{"source": input})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Output.Object) != 1 || result.Output.Object[0].Name != "items" {
		t.Fatalf("unexpected omit result: %#v", result.Output)
	}
	if len(input.Object) != 2 || input.Object[0].Name != "replace" {
		t.Fatalf("omit mutated caller-owned object: %#v", input)
	}
	result.Output.Object[0].Value.Array[0].String = "changed"
	if input.Object[1].Value.Array[0].String != "before" {
		t.Fatal("returned omit output mutated caller-owned nested data")
	}
}

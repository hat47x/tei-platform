package transform_test

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

func joinArraySource(t *testing.T, separator string, maxOutput int) transform.Source {
	t.Helper()
	return transform.Source{
		FormatVersion: transform.SourceFormatVersion,
		Identity: transform.SourceIdentity{
			ID: "tei.test.join_array", Version: "1.0.0", ContractVersion: "tei.plugin-transform/v1",
		},
		Profile: transform.ProfileV1,
		Limits: transform.Limits{
			MaxSteps: 1000, MaxCollection: 4, MaxInputNodes: 100,
			MaxInputBytes: 4096, MaxOutputBytes: maxOutput, MaxNestingDepth: 8,
		},
		Inputs: []transform.InputSpec{{Name: "items", Kind: transform.InputArray}},
		Program: transform.Program{Output: transform.Expr{
			Op: transform.OpJoinArray, Separator: separator,
			From: &transform.Expr{
				Op: transform.OpMapArray, Item: "item",
				From: &transform.Expr{Op: transform.OpInput, Name: "items"},
				Body: &transform.Expr{Op: transform.OpJoinStrings, Values: []transform.Expr{
					{Op: transform.OpLiteral, Literal: json.RawMessage(`"<"`)},
					{Op: transform.OpInput, Name: "item"},
					{Op: transform.OpLiteral, Literal: json.RawMessage(`">"`)},
				}},
			},
		}},
	}
}

func items(values ...transform.Value) map[string]transform.Value {
	return map[string]transform.Value{"items": transform.Array(values...)}
}

// A declared list of any length becomes text in declared order. This is what a
// fixed join-strings list cannot do: the Source stays the same when the list grows.
func TestJoinArrayJoinsAListOfAnyLength(t *testing.T) {
	source := joinArraySource(t, ",", 64)
	for _, c := range []struct {
		in   []transform.Value
		want string
	}{
		{nil, ""},
		{[]transform.Value{transform.String("a")}, "<a>"},
		{[]transform.Value{transform.String("b"), transform.String("a"), transform.String("")}, "<b>,<a>,<>"},
	} {
		result, err := transform.Evaluate(source, items(c.in...))
		if err != nil {
			t.Fatal(err)
		}
		if result.Output.Kind != transform.ValueString || result.Output.String != c.want {
			t.Fatalf("got %#v, want %q", result.Output, c.want)
		}
	}
}

func TestJoinArrayRejectsWhatItCannotJoin(t *testing.T) {
	cases := map[string]struct {
		source transform.Source
		input  map[string]transform.Value
		want   string
	}{
		"non-string item": {joinArraySource(t, ",", 64), nil, "requires string"},
		"over maxCollection": {joinArraySource(t, ",", 64),
			items(transform.String("1"), transform.String("2"), transform.String("3"), transform.String("4"), transform.String("5")),
			"maxCollection"},
		"over maxOutputBytes": {joinArraySource(t, ",", 8),
			items(transform.String("aaa"), transform.String("bbb")), "maxOutputBytes"},
		"separator over maxOutputBytes": {joinArraySource(t, strings.Repeat("-", 5), 7),
			items(transform.String(""), transform.String("")), "maxOutputBytes"},
	}
	// map-array yields the array join-array reads; give it a non-string body.
	nonString := cases["non-string item"]
	nonString.source.Program.Output.From.Body = &transform.Expr{Op: transform.OpInput, Name: "item"}
	nonString.input = items(transform.Number("1"))
	cases["non-string item"] = nonString
	for name, c := range cases {
		if _, err := transform.Evaluate(c.source, c.input); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Fatalf("%s: error %v does not name %q", name, err, c.want)
		}
	}
}

func TestJoinArrayValidatesItsShape(t *testing.T) {
	source := joinArraySource(t, ",", 64)
	source.Program.Output.From = nil
	if err := source.Validate(); err == nil || !strings.Contains(err.Error(), "join-array source") {
		t.Fatalf("missing source accepted: %v", err)
	}
	source = joinArraySource(t, ",", 64)
	source.Program.Output.Values = []transform.Expr{{Op: transform.OpLiteral, Literal: json.RawMessage(`"x"`)}}
	if err := source.Validate(); err == nil || !strings.Contains(err.Error(), "does not accept field values") {
		t.Fatalf("join-strings field on join-array accepted: %v", err)
	}
	// The serialized form round-trips through the strict Source parser.
	raw, err := json.Marshal(joinArraySource(t, "\n", 64))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transform.ParseSourceJSON(raw); err != nil {
		t.Fatalf("join-array Source does not parse: %v", err)
	}
}

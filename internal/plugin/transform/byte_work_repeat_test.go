package transform_test

import (
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/plugin/transform"
)

// Repeating byte-linear operations against the same bounded input must not
// multiply the effective CPU work without consuming the declared maxSteps.
func TestRepeatedSHA256ChargesWorkForEveryVisit(t *testing.T) {
	input := transform.Expr{Op: transform.OpInput, Name: "payload"}
	hash := transform.Expr{Op: transform.OpSHA256, From: &input}
	data := transform.Bytes(make([]byte, 64<<10))
	values := map[string]transform.Value{"payload": data}

	for _, tc := range []struct {
		name    string
		repeats int
		allowed bool
	}{
		{name: "one hash fits", repeats: 1, allowed: true},
		{name: "three hashes exceed", repeats: 3, allowed: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operations := make([]transform.Expr, tc.repeats)
			for i := range operations {
				operations[i] = hash
			}
			source := byteWorkSource("tei.builder.repeated_hash_work", 40,
				[]transform.InputSpec{{Name: "payload", Kind: transform.InputBytes}},
				transform.Expr{Op: transform.OpJoinStrings, Values: operations})
			result, err := transform.Evaluate(source, values)
			if tc.allowed {
				if err != nil {
					t.Fatalf("single SHA-256 must fit: %v", err)
				}
				if result.Output.Kind != transform.ValueString || len(result.Output.String) != 64 {
					t.Fatalf("unexpected SHA-256 output %#v", result.Output)
				}
			} else if err == nil || !strings.Contains(err.Error(), "maxSteps") {
				t.Fatalf("repeated SHA-256 must hit maxSteps, got %v", err)
			}
		})
	}
}

func TestRepeatedAssertEqualChargesEveryLargeComparison(t *testing.T) {
	actual := transform.Expr{Op: transform.OpInput, Name: "actual"}
	expected := transform.Expr{Op: transform.OpInput, Name: "expected"}
	assertion := transform.Expr{Op: transform.OpAssertEqual, Actual: &actual, Expected: &expected}
	large := transform.Bytes(make([]byte, 16<<10))
	values := map[string]transform.Value{"actual": large, "expected": large}
	for _, tc := range []struct {
		name       string
		assertions int
		allowed    bool
	}{
		{name: "one comparison fits", assertions: 1, allowed: true},
		{name: "four comparisons exceed", assertions: 4, allowed: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := byteWorkSource("tei.builder.repeated_equal_work", 20,
				[]transform.InputSpec{
					{Name: "actual", Kind: transform.InputBytes},
					{Name: "expected", Kind: transform.InputBytes},
				}, actual)
			for i := 0; i < tc.assertions; i++ {
				source.Program.Assertions = append(source.Program.Assertions, assertion)
			}
			_, err := transform.Evaluate(source, values)
			if tc.allowed && err != nil {
				t.Fatalf("single equality comparison must fit: %v", err)
			}
			if !tc.allowed && (err == nil || !strings.Contains(err.Error(), "maxSteps")) {
				t.Fatalf("repeated equality must hit maxSteps, got %v", err)
			}
		})
	}
}

func TestAssertEnumChargesEveryRejectedLargeCandidate(t *testing.T) {
	value := strings.Repeat("x", 16<<10)
	actual := transform.Expr{Op: transform.OpInput, Name: "actual"}
	jsonString := func(s string) json.RawMessage {
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for _, tc := range []struct {
		name    string
		allowed []json.RawMessage
		fits    bool
	}{
		{name: "first candidate matches", allowed: []json.RawMessage{jsonString(value)}, fits: true},
		{name: "several near misses", allowed: []json.RawMessage{
			jsonString(value[:len(value)-1] + "a"),
			jsonString(value[:len(value)-1] + "b"),
			jsonString(value[:len(value)-1] + "c"),
			jsonString(value),
		}, fits: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := byteWorkSource("tei.builder.repeated_enum_work", 20,
				[]transform.InputSpec{{Name: "actual", Kind: transform.InputString}},
				actual)
			source.Program.Assertions = []transform.Expr{{
				Op: transform.OpAssertEnum, Actual: &actual, Allowed: tc.allowed,
			}}
			result, err := transform.Evaluate(source, map[string]transform.Value{
				"actual": transform.String(value),
			})
			if tc.fits {
				if err != nil {
					t.Fatalf("matching candidate must fit: %v", err)
				}
				if result.Output.Kind != transform.ValueString || result.Output.String != value {
					t.Fatal("accepted value was changed")
				}
			} else if err == nil || !strings.Contains(err.Error(), "maxSteps") {
				t.Fatalf("repeated enum comparisons must hit maxSteps, got %v", err)
			}
		})
	}
}

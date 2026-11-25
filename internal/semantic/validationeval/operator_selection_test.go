package validationeval_test

import (
	"math/big"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

func TestBuiltinImplementationsDecideByTheirDefinitions(t *testing.T) {
	// Expected values are the definitions written out for five numbers.
	singleInputWant := map[string]map[string]bool{
		"tei.number.non-negative": {"-3": false, "-1/100": false, "0": true, "1/100": true, "3": true},
		"tei.number.non-zero":     {"-3": true, "-1/100": true, "0": false, "1/100": true, "3": true},
	}
	// The row-scoped implementation is checked over rows, with the expected
	// three-valued answer read from its definition: every row at most the
	// limit, a row above it, and a row whose value was not read.
	rowInputWant := map[string][]struct {
		name    string
		rows    map[string][]validationeval.NumberReading
		want    validationeval.Tri
		wantErr bool
	}{
		"tei.number.every-at-most": {
			{name: "every row at most", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1", "0.5", "0"),
				"limit": presentRows("1", "1", "1"),
			}, want: validationeval.TriYes},
			{name: "one row above", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1", "2"),
				"limit": presentRows("1", "1"),
			}, want: validationeval.TriNo},
			{name: "a row with no value", rows: map[string][]validationeval.NumberReading{
				"value": {{State: validationeval.ValueNull}},
				"limit": presentRows("1"),
			}, want: validationeval.TriMaybe},
			{name: "no value input", rows: map[string][]validationeval.NumberReading{
				"limit": presentRows("1"),
			}, wantErr: true},
			{name: "no limit input", rows: map[string][]validationeval.NumberReading{
				"value": presentRows("1"),
			}, wantErr: true},
		},
	}
	multiInputWant := map[string][]struct {
		name    string
		values  map[string]*big.Rat
		want    bool
		wantErr bool
	}{
		"tei.number.within": {
			{name: "in-range", values: map[string]*big.Rat{"value": big.NewRat(0, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: true},
			{name: "at-low", values: map[string]*big.Rat{"value": big.NewRat(0, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: true},
			{name: "at-high", values: map[string]*big.Rat{"value": big.NewRat(100, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: true},
			{name: "below-low", values: map[string]*big.Rat{"value": big.NewRat(-1, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: false},
			{name: "above-high", values: map[string]*big.Rat{"value": big.NewRat(101, 1), "low": big.NewRat(0, 1), "high": big.NewRat(100, 1)}, want: false},
			// low > high is a wrong range specification: it is reported, never
			// answered as not-satisfied, and the bounds are not swapped.
			{name: "low-greater-than-high", values: map[string]*big.Rat{"value": big.NewRat(50, 1), "low": big.NewRat(100, 1), "high": big.NewRat(0, 1)}, wantErr: true},
		},
	}
	seen := map[string]bool{}
	for _, implementation := range validationeval.BuiltinImplementations() {
		if seen[implementation.ID] {
			t.Fatalf("built-in implementation %s is listed twice", implementation.ID)
		}
		seen[implementation.ID] = true
		if implementation.Number != nil {
			expected, known := singleInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			if implementation.Input != semantic.OperatorInputNumber {
				t.Fatalf("%s input=%q", implementation.ID, implementation.Input)
			}
			for text, satisfied := range expected {
				value, _ := new(big.Rat).SetString(text)
				got, err := implementation.Number(value)
				if err != nil || got != satisfied {
					t.Fatalf("%s(%s)=%v err=%v, want %v", implementation.ID, text, got, err, satisfied)
				}
			}
		} else if implementation.Numbers != nil {
			cases, known := multiInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			for _, tc := range cases {
				got, err := implementation.Numbers(tc.values)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("%s(%s)=%v err=nil, want an error", implementation.ID, tc.name, got)
					}
					continue
				}
				if err != nil {
					t.Fatalf("%s(%s)=%v err=%v, want %v", implementation.ID, tc.name, got, err, tc.want)
				}
				if got != tc.want {
					t.Fatalf("%s(%s)=%v, want %v", implementation.ID, tc.name, got, tc.want)
				}
			}
		} else if implementation.NumberRows != nil {
			cases, known := rowInputWant[implementation.ID]
			if !known {
				t.Fatalf("built-in implementation %s has no expected values in this test", implementation.ID)
			}
			for _, tc := range cases {
				got, err := implementation.NumberRows(tc.rows)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("%s(%s)=%d err=nil, want an error", implementation.ID, tc.name, int(got))
					}
					continue
				}
				if err != nil {
					t.Fatalf("%s(%s)=%d err=%v, want %d", implementation.ID, tc.name, int(got), err, int(tc.want))
				}
				if got != tc.want {
					t.Fatalf("%s(%s)=%d, want %d", implementation.ID, tc.name, int(got), int(tc.want))
				}
			}
		} else {
			t.Fatalf("built-in implementation %s has no decision function", implementation.ID)
		}
	}
	if len(seen) != len(singleInputWant)+len(multiInputWant)+len(rowInputWant) {
		t.Fatalf("built-in implementations=%v, want exactly %d", seen, len(singleInputWant)+len(multiInputWant))
	}
}

func TestResolveImplementationsReturnsOnlyWhatWasSelected(t *testing.T) {
	var calls []string
	record := func(name string, result bool) validationeval.NamedImplementation {
		return validationeval.NamedImplementation{
			ID: name, Input: semantic.OperatorInputNumber,
			Number: func(*big.Rat) (bool, error) { calls = append(calls, name); return result, nil },
		}
	}
	a, b := record("example.a", true), record("example.b", false)
	selection := []validationeval.ImplementationSelection{{Operator: "sales.non_negative", Implementation: "example.b"}}

	for _, available := range [][]validationeval.NamedImplementation{{a, b}, {b, a}} {
		calls = nil
		var observed []validationeval.ImplementationCall
		resolved, err := validationeval.ResolveImplementations(selection, available, func(call validationeval.ImplementationCall) {
			observed = append(observed, call)
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(resolved) != 1 || resolved[0].Operator != "sales.non_negative" || resolved[0].Input != semantic.OperatorInputNumber {
			t.Fatalf("resolved=%+v", resolved)
		}
		if len(observed) != 0 {
			t.Fatalf("resolving reported a call before anything was decided: %+v", observed)
		}
		got, err := resolved[0].Number(big.NewRat(1, 1))
		if err != nil || got {
			t.Fatalf("selected B must answer false: %v %v", got, err)
		}
		if !reflect.DeepEqual(calls, []string{"example.b"}) {
			t.Fatalf("called %v, want only the selected implementation", calls)
		}
		wantCall := []validationeval.ImplementationCall{{Operator: "sales.non_negative", Implementation: "example.b"}}
		if !reflect.DeepEqual(observed, wantCall) {
			t.Fatalf("observed=%+v, want %+v", observed, wantCall)
		}
	}
}

func TestResolveImplementationsRefusals(t *testing.T) {
	one := validationeval.NamedImplementation{ID: "example.a", Input: semantic.OperatorInputNumber,
		Number: func(*big.Rat) (bool, error) { return true, nil }}
	cases := []struct {
		name      string
		selected  []validationeval.ImplementationSelection
		available []validationeval.NamedImplementation
		want      string
	}{
		{"name nobody carries",
			[]validationeval.ImplementationSelection{{Operator: "sales.op", Implementation: "example.missing"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE"},
		{"name carried twice",
			[]validationeval.ImplementationSelection{{Operator: "sales.op", Implementation: "example.a"}},
			[]validationeval.NamedImplementation{one, one}, "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS"},
		{"operator selected twice",
			[]validationeval.ImplementationSelection{
				{Operator: "sales.op", Implementation: "example.a"}, {Operator: "sales.op", Implementation: "example.a"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_AMBIGUOUS"},
		{"selection without an implementation",
			[]validationeval.ImplementationSelection{{Operator: "sales.op"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_INVALID"},
		{"selection without an operator",
			[]validationeval.ImplementationSelection{{Implementation: "example.a"}},
			[]validationeval.NamedImplementation{one}, "TEI_OPERATOR_IMPLEMENTATION_INVALID"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := validationeval.ResolveImplementations(tc.selected, tc.available, nil)
			if err == nil || !strings.Contains(err.Error(), tc.want) || resolved != nil {
				t.Fatalf("resolved=%+v err=%v, want %q", resolved, err, tc.want)
			}
		})
	}
	// The unavailable message names what can be selected instead.
	_, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: "sales.op", Implementation: "example.missing"}},
		validationeval.BuiltinImplementations(), nil)
	if err == nil || !strings.Contains(err.Error(), "tei.number.non-negative, tei.number.non-zero") {
		t.Fatalf("message does not list the available implementations: %v", err)
	}
}

// presentRows builds the readings of one column whose rows all hold a value.
func presentRows(values ...string) []validationeval.NumberReading {
	out := make([]validationeval.NumberReading, 0, len(values))
	for _, text := range values {
		value, ok := new(big.Rat).SetString(text)
		if !ok {
			panic("not a number: " + text)
		}
		out = append(out, validationeval.NumberReading{State: validationeval.ValuePresent, Value: value})
	}
	return out
}

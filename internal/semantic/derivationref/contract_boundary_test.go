package derivationref

import (
	"errors"
	"testing"
)

func TestValidateGraphRejectsInvalidRules(t *testing.T) {
	tests := []struct {
		name string
		defs []Definition
		deps map[string][]string
		want error
	}{
		{"duplicate", []Definition{{ID: "x", Pure: true}, {ID: "x", Pure: true}}, nil, nil},
		{"empty identity", []Definition{{ID: "", Pure: true}}, nil, nil},
		{"self cycle", []Definition{{ID: "x", Pure: true}}, map[string][]string{"x": {"x"}}, ErrCycle},
		{"deep cycle", []Definition{{ID: "x", Pure: true}, {ID: "y", Pure: true}, {ID: "z", Pure: true}}, map[string][]string{"x": {"y"}, "y": {"z"}, "z": {"x"}}, ErrCycle},
		{"unknown dependency", []Definition{{ID: "x", Pure: true}}, map[string][]string{"x": {"unknown"}}, nil},
		{"impure", []Definition{{ID: "x", Pure: false}}, nil, ErrImpure},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateGraph(tc.defs, tc.deps)
			if err == nil {
				t.Fatal("invalid graph was accepted")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("got %v; want %v", err, tc.want)
			}
		})
	}
}

func TestMultiplyInputErrorsAreNotNull(t *testing.T) {
	invalid := []Input{
		{State: Missing}, {State: AbsentRelation}, {State: MultipleRelations}, {State: "unknown"},
	}
	for _, in := range invalid {
		result, err := Multiply(Input{State: Null}, in)
		if !errors.Is(err, ErrInput) {
			t.Fatalf("input %q: error %v", in.State, err)
		}
		if result.State == Null {
			t.Fatalf("input %q: incorrectly returned NULL", in.State)
		}
	}
}

func TestMultiplyExactSignedDecimals(t *testing.T) {
	tests := []struct{ a, b, want string }{
		{"-1.20", "2.5", "-3"},
		{"0.0001", "0.0002", "0.00000002"},
		{"1.5", "0.2", "0.3"},
	}
	for _, tc := range tests {
		got, err := Multiply(Input{State: Present, Decimal: tc.a}, Input{State: Present, Decimal: tc.b})
		if err != nil {
			t.Fatal(err)
		}
		if got.Decimal != tc.want {
			t.Fatalf("%s * %s = %s, want %s", tc.a, tc.b, got.Decimal, tc.want)
		}
	}
}

package validationeval_test

import (
	"math/big"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A row-scoped contract's example is checked against the selected
// implementation before any real row is read (section 8.1). The expectations
// below are written from that statement, not from the code.
func rowExampleContract(cases ...semantic.OperatorRowCase) semantic.Operator {
	return semantic.Operator{
		ID:    "sales.every_line",
		Input: semantic.OperatorInputNumber,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputNumber, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
		RowCases:          cases,
	}
}

// everyAtMostRows answers TriNo when some row's value is above its own limit,
// whatever other rows hold, and TriYes otherwise.
func everyAtMostRows(rows map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
	values, limits := rows["value"], rows["limit"]
	if len(values) != len(limits) {
		return validationeval.TriMaybe, nil
	}
	for i := range values {
		if values[i].State != validationeval.ValuePresent || limits[i].State != validationeval.ValuePresent {
			continue
		}
		if values[i].Value.Cmp(limits[i].Value) > 0 {
			return validationeval.TriNo, nil
		}
	}
	return validationeval.TriYes, nil
}

func rowExampleImplementation() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator:   "sales.every_line",
		Input:      semantic.OperatorInputNumber,
		NumberRows: everyAtMostRows,
	}
}

// The example reaches the implementation through the decision, so a contract
// whose example the implementation disagrees with stops the rule.
// The three-valued table of section 8.1 is pinned against the shipped
// implementation, in both row orders: a row that holds no value, or was not
// read, does not hide a counterexample found in another row, and does not let
// the rule hold either. The expectations are the design's, not the code's.
func TestEveryAtMostAnswersSectionEightTable(t *testing.T) {
	var rows func(map[string][]validationeval.NumberReading) (validationeval.Tri, error)
	for _, available := range validationeval.BuiltinImplementations() {
		if available.ID == "tei.number.every-at-most" {
			rows = available.NumberRows
		}
	}
	if rows == nil {
		t.Fatal("the shipped every-at-most implementation was not found")
	}
	number := func(text string) validationeval.NumberReading {
		value, err := semantic.ParseOperatorCaseValue(text)
		if err != nil {
			t.Fatalf("not a number: %q", text)
		}
		return validationeval.NumberReading{State: validationeval.ValuePresent, Value: value}
	}
	table := []struct {
		name   string
		value  []validationeval.NumberReading
		limit  []validationeval.NumberReading
		answer validationeval.Tri
	}{
		{"(2,10) (5,5)", []validationeval.NumberReading{number("2"), number("5")}, []validationeval.NumberReading{number("10"), number("5")}, validationeval.TriYes},
		{"(2,10) (5,4)", []validationeval.NumberReading{number("2"), number("5")}, []validationeval.NumberReading{number("10"), number("4")}, validationeval.TriNo},
		{"(NULL,10) (5,4)", []validationeval.NumberReading{{State: validationeval.ValueNull}, number("5")}, []validationeval.NumberReading{number("10"), number("4")}, validationeval.TriNo},
		{"(5,4) (NULL,10)", []validationeval.NumberReading{number("5"), {State: validationeval.ValueNull}}, []validationeval.NumberReading{number("4"), number("10")}, validationeval.TriNo},
		{"(未読,10) (5,4)", []validationeval.NumberReading{{State: validationeval.ValueNotFetched}, number("5")}, []validationeval.NumberReading{number("10"), number("4")}, validationeval.TriNo},
		{"(5,4) (未読,10)", []validationeval.NumberReading{number("5"), {State: validationeval.ValueNotFetched}}, []validationeval.NumberReading{number("4"), number("10")}, validationeval.TriNo},
		{"(未読,10) (5,5)", []validationeval.NumberReading{{State: validationeval.ValueNotFetched}, number("5")}, []validationeval.NumberReading{number("10"), number("5")}, validationeval.TriMaybe},
		{"(NULL,10) (5,5)", []validationeval.NumberReading{{State: validationeval.ValueNull}, number("5")}, []validationeval.NumberReading{number("10"), number("5")}, validationeval.TriMaybe},
	}
	for _, tc := range table {
		t.Run(tc.name, func(t *testing.T) {
			got, err := rows(map[string][]validationeval.NumberReading{"value": tc.value, "limit": tc.limit})
			if err != nil {
				t.Fatalf("err=%v", err)
			}
			if got != tc.answer {
				t.Fatalf("got=%d, want %d", int(got), int(tc.answer))
			}
		})
	}
}

func TestRowExampleIsCheckedAgainstTheImplementation(t *testing.T) {
	attribute, err := core.ParseCanonicalID("sales.order_line.quantity")
	if err != nil {
		t.Fatal(err)
	}
	price, err := core.ParseCanonicalID("sales.order_line.unit_price")
	if err != nil {
		t.Fatal(err)
	}
	source := &rowsSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.NumberReading{
			attribute: {{State: validationeval.ValuePresent, Value: big.NewRat(1, 1)}},
			price:     {{State: validationeval.ValuePresent, Value: big.NewRat(10, 1)}},
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.every_line", Symbol: "every_line",
		Kind:      semantic.ValidationOperator,
		Attribute: attribute, Relation: "sales.order.lines",
		Operator: "sales.every_line",
		Inputs:   []semantic.ValidationInputAssignment{{Name: "limit", Value: "sales.order_line.unit_price"}},
	}
	for _, tc := range []struct {
		name    string
		example semantic.OperatorRowCase
		wantErr string
	}{
		{
			name: "an example that agrees is accepted",
			example: semantic.OperatorRowCase{
				Rows: []map[string]string{
					{"value": "2", "limit": "10"},
					{"value": "5", "limit": "5"},
				},
				Answer: semantic.OperatorRowSatisfied,
			},
		},
		{
			name: "an example whose answer the implementation disagrees with stops the rule",
			example: semantic.OperatorRowCase{
				Rows:   []map[string]string{{"value": "5", "limit": "4"}},
				Answer: semantic.OperatorRowSatisfied,
			},
			wantErr: "TEI_OPERATOR_IMPLEMENTATION_MISMATCH",
		},
		{
			name: "a counterexample after a row that holds no value is still a counterexample",
			example: semantic.OperatorRowCase{
				// The shipped operator reads the first row as undecidable but
				// keeps looking, so the answer is TriNo, not TriMaybe.
				Rows:   []map[string]string{{"value": "?", "limit": "10"}},
				Answer: semantic.OperatorRowNotSatisfied,
			},
			wantErr: "unreadable",
		},
		{
			name: "an example with no rows is refused",
			example: semantic.OperatorRowCase{
				// An empty collection is not the same as one row holding no
				// value, so it is refused rather than read as one.
				Answer: semantic.OperatorRowSatisfied,
			},
			wantErr: "no rows",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract := rowExampleContract(tc.example)
			_, err := validationeval.Decide(rule, source, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{rowExampleImplementation()},
			})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("a valid example stopped the rule: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

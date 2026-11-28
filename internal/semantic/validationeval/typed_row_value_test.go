package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// typedRowsSource is a reader that can read one column of a relation's rows in
// each of the types this program reads, so a row-scoped operator can be
// exercised for each declared type.
type typedRowsSource struct {
	numberRows validationeval.TypedRowReading
	dateRows   validationeval.TypedRowReading
	textRows   validationeval.TypedRowReading
	truthRows  validationeval.TypedRowReading
	anyRows    validationeval.TypedRowReading
}

func (s typedRowsSource) ReadTypedRows(input semantic.OperatorInputDeclaration, _, _ core.CanonicalID) (validationeval.TypedRowReading, error) {
	switch input.Type {
	case semantic.OperatorInputNumber:
		return s.numberRows, nil
	case semantic.OperatorInputDate:
		return s.dateRows, nil
	case semantic.OperatorInputText:
		return s.textRows, nil
	case semantic.OperatorInputTruth:
		return s.truthRows, nil
	default:
		return s.anyRows, nil
	}
}

func (s typedRowsSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
}

func (s typedRowsSource) ReadRelationKeys(_, _ core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func presentText(text string) validationeval.RowValueReading {
	return validationeval.RowValueReading{
		State: validationeval.ValuePresent,
		Value: semantic.OperatorValue{Text: &text},
	}
}

// TestATextRowScopedRuleIsDecidedByItsRows pins decision section 8.4 for the
// row-scoped shape: an input that reads every row of a relation as text is
// decided by an implementation that provides the typed row field, and the rows
// it receives carry the type they were read as. Before this, a row-scoped
// operator could only be decided from numbers.
func TestATextRowScopedRuleIsDecidedByItsRows(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText, OfEachRow: true},
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Relation: "sales.order_line", Attribute: "sales.order_line.note", Operator: "sales.op",
	}

	cases := []struct {
		name  string
		texts []string
		want  validationeval.Tri
	}{
		{"every row holds the same text", []string{"A-100", "A-100"}, validationeval.TriYes},
		{"one row differs", []string{"A-100", "B-200"}, validationeval.TriNo},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := make([]validationeval.RowValueReading, 0, len(tc.texts))
			for _, text := range tc.texts {
				rows = append(rows, presentText(text))
			}
			seen := make(chan map[string][]validationeval.RowValueReading, 1)
			implementation := validationeval.OperatorImplementation{
				Operator: "sales.op",
				Input:    semantic.OperatorInputText,
				ValueRows: func(values map[string][]validationeval.RowValueReading) (validationeval.Tri, error) {
					seen <- values
					return tc.want, nil
				},
			}
			decision, decideErr := validationeval.Decide(rule, typedRowsSource{
				textRows: validationeval.TypedRowReading{
					// The reader states whether these rows are all the rows of
					// the relation, as every reader of a relation's rows does:
					// without the statement a rule over all the rows cannot be
					// answered.
					Coverage: validationeval.CoverageComplete,
					State:    validationeval.ValuePresent,
					Values:   rows,
				},
			}, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: []validationeval.OperatorImplementation{implementation},
			})
			if decideErr != nil {
				t.Fatalf("the row-scoped rule was refused: %v", decideErr)
			}
			if decision.Result != resultFor(tc.want) {
				t.Fatalf("result=%v, want %v", decision.Result, tc.want)
			}
			select {
			case got := <-seen:
				if len(got["value"]) != len(tc.texts) {
					t.Fatalf("the implementation received %d rows, want %d", len(got["value"]), len(tc.texts))
				}
				for index, text := range tc.texts {
					if got["value"][index].Value.Text == nil || *got["value"][index].Value.Text != text {
						t.Fatalf("row %d received %v, want %q", index, got["value"][index].Value.Text, text)
					}
				}
			default:
				t.Fatal("the implementation was not asked about the rows")
			}
		})
	}
}

// TestARowScopedTypedRuleIsRefusedByANumberReader pins the other side: a reader
// that cannot read the declared type over rows is refused, so the rule is not
// decided from numbers the column was never read as.
func TestARowScopedTypedRuleIsRefusedByANumberReader(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.op", Symbol: "op", Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText, OfEachRow: true},
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.probe", Kind: semantic.ValidationOperator,
		Relation: "sales.order_line", Attribute: "sales.order_line.note", Operator: "sales.op",
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.op",
		Input:    semantic.OperatorInputText,
		ValueRows: func(map[string][]validationeval.RowValueReading) (validationeval.Tri, error) {
			t.Fatal("the implementation was asked by a reader that cannot read text rows")
			return validationeval.TriMaybe, nil
		},
	}
	_, err := validationeval.Decide(rule, numbersOnlyRows{}, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err == nil {
		t.Fatal("a text row-scoped rule was decided by a reader that reads numbers")
	}
}

// resultFor maps a Tri answer to the Result the decision carries for a
// row-scoped rule.
func resultFor(tri validationeval.Tri) validationeval.Result {
	if tri == validationeval.TriYes {
		return validationeval.ResultSatisfied
	}
	if tri == validationeval.TriNo {
		return validationeval.ResultNotSatisfied
	}
	return ""
}

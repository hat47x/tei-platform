package validationeval_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// rowValueSource reads one column of typed values over the rows of one
// relation, and records which relation and attribute were asked.
type rowValueSource struct {
	columns map[core.CanonicalID][]validationeval.ValueReading
	asked   []string
}

func (s *rowValueSource) ReadAttempt(core.CanonicalID) (string, bool) { return "one-read", true }

func (s *rowValueSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, errNoSingleRow
}

func (s *rowValueSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s *rowValueSource) ReadRelationValues(relation, attribute core.CanonicalID) (validationeval.ValueRowsReading, error) {
	s.asked = append(s.asked, string(relation)+"/"+string(attribute))
	return validationeval.ValueRowsReading{
		Coverage: validationeval.CoverageComplete,
		Rows:     s.columns[attribute],
	}, nil
}

var errNoSingleRow = errString("this source reads rows, not one record")

// textColumn is the value column of a row-scoped text input.
func textColumn(texts ...string) []validationeval.ValueReading {
	out := make([]validationeval.ValueReading, 0, len(texts))
	for _, text := range texts {
		value := text
		out = append(out, validationeval.ValueReading{
			State: validationeval.ValuePresent,
			Value: &validationeval.OperatorValue{Text: &value},
		})
	}
	return out
}

// A row-scoped operator whose inputs are not numbers is decided over the rows
// it read, and each column is read once, from the one relation the rule names.
func TestARowScopedTypedInputIsDecidedOverTheRows(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.named_lines", Symbol: "named_lines", Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText, OfEachRow: true},
			{Name: "limit", Type: semantic.OperatorInputNumber, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
	}
	source := &rowValueSource{columns: map[core.CanonicalID][]validationeval.ValueReading{
		"sales.order_line.item_name": textColumn("A-100", "B-200"),
		"sales.order_line.quantity": {{
			State: validationeval.ValuePresent,
			Value: &validationeval.OperatorValue{Number: rat("10")},
		}, {
			State: validationeval.ValuePresent,
			Value: &validationeval.OperatorValue{Number: rat("5")},
		}},
	}}
	// The implementation counts the rows whose text is longer than its limit.
	// It is a pure function of the values it is handed and sees no Source.
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.named_lines", Input: semantic.OperatorInputText,
		ValueRows: func(rows map[string][]validationeval.ValueReading) (validationeval.Tri, error) {
			values := rows["value"]
			limits := rows["limit"]
			if len(values) != len(limits) {
				return validationeval.TriMaybe, errString("the two columns are not the same rows")
			}
			for index := range values {
				if values[index].State != validationeval.ValuePresent ||
					limits[index].State != validationeval.ValuePresent {
					return validationeval.TriMaybe, nil
				}
				if values[index].Value.Text == nil || limits[index].Value.Number == nil {
					return validationeval.TriMaybe, errString("a column holds a value of another type")
				}
			}
			return validationeval.TriYes, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order.lines.named", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.item_name", Operator: "sales.named_lines",
		Relation: "sales.order.lines",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "limit", Value: "sales.order_line.quantity"},
		},
	}
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("a row-scoped typed rule was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied", decision.Result)
	}
	if len(source.asked) != 2 {
		t.Fatalf("the reader was asked %v, want the two columns once each", source.asked)
	}
	for _, asked := range source.asked {
		if asked != "sales.order.lines/sales.order_line.item_name" &&
			asked != "sales.order.lines/sales.order_line.quantity" {
			t.Fatalf("the reader was asked %q, which is not a column of the rule's relation", asked)
		}
	}
}

// A row-scoped operator whose implementation decides from numbers is not
// selected for a contract that declares a text column: the numbers would answer
// a question about text.
func TestARowScopedNumericImplementationIsNotSelectedForATypedContract(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.named_lines", Symbol: "named_lines", Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText, OfEachRow: true},
		},
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.named_lines", Input: semantic.OperatorInputText,
		NumberRows: func(map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		},
	}
	if err := validationeval.CheckImplementationMatchesContract(contract, implementation); err == nil {
		t.Fatal("a numeric row implementation was selected for a text column")
	}
}

// A constant assigned to a typed input is read as a value of that input's type,
// so a text constant is quoted and a date constant is a calendar date.
func TestAConstantOfATypedInputIsReadAsThatType(t *testing.T) {
	contract := semantic.Operator{
		ID: "sales.starts_with", Symbol: "starts_with", Input: semantic.OperatorInputText,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputText},
			{Name: "prefix", Type: semantic.OperatorInputText},
		},
	}
	source := &typedSource{cells: map[core.CanonicalID]string{"sales.order_line.item_name": `"A-100"`}}
	var got map[string]validationeval.OperatorValue
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.starts_with", Input: semantic.OperatorInputText,
		Values: func(values map[string]validationeval.OperatorValue) (bool, error) {
			got = values
			return true, nil
		},
	}
	rule := semantic.Validation{
		ID: "sales.order_line.item_prefixed", Kind: semantic.ValidationOperator,
		Attribute: "sales.order_line.item_name", Operator: "sales.starts_with",
		Inputs: []semantic.ValidationInputAssignment{{Name: "prefix", Value: `"A-"`}},
	}
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("a typed constant was refused: %v", err)
	}
	if decision.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied", decision.Result)
	}
	if got["prefix"].Text == nil || *got["prefix"].Text != "A-" {
		t.Fatalf("prefix=%+v, want the text A-", got["prefix"].Text)
	}
	if got["value"].Text == nil || *got["value"].Text != "A-100" {
		t.Fatalf("value=%+v, want the text A-100", got["value"].Text)
	}
	// A constant that is not of the input's type is refused rather than read
	// as something else.
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "prefix", Value: "A-"}}
	if _, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	}); err == nil {
		t.Fatal("an unquoted constant was read as text")
	}
}

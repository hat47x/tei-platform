package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// RowValueReading is one row of one input of a row-scoped operator, in the type
// the declaration states. Exactly one field of Value is set, decided by that
// type: a row read as a number is carried as a number and never as a text
// (decision section 8.4).
//
// The state stays beside the value, so "this row holds no value" and "nobody
// read this row" stay apart rather than both becoming the zero of the type.
type RowValueReading struct {
	State ValueState
	Value semantic.OperatorValue
}

// TypedRowReading is what a reader knows about one input's column over the rows
// of a relation: the rows in the order they were read, each value carrying the
// type the declaration states.
type TypedRowReading struct {
	State  ValueState
	Values []RowValueReading
}

// StatefulTypedRowSource is implemented by a reader that can read one column of
// a relation's rows in the type the declaration states. A rule whose operator
// reads every row of a relation in a type other than number is refused by a
// reader that cannot: it is not decided by reading the column as numbers
// (decision section 8.4).
type StatefulTypedRowSource interface {
	ReadTypedRows(input semantic.OperatorInputDeclaration, relation, attribute core.CanonicalID) (TypedRowReading, error)
}

// readTypedRows reads one input's column over the rows of a relation in the
// type its declaration states. A reader that cannot read that type over rows is
// an error rather than a value of another type: reading a text column as
// numbers would answer a question about a different value.
func readTypedRows(rule semantic.Validation, input semantic.OperatorInputDeclaration, source StatefulSource, relation, attribute core.CanonicalID) (TypedRowReading, error) {
	reader, ok := source.(StatefulTypedRowSource)
	if !ok {
		return TypedRowReading{}, fmt.Errorf("semantic validation %s reads operator %s input %q as %s over the rows of %s, which this reader cannot read", rule.ID, rule.Operator, input.Name, input.Type, relation)
	}
	reading, err := reader.ReadTypedRows(input, relation, attribute)
	if err != nil {
		return TypedRowReading{}, fmt.Errorf("semantic validation %s cannot read attribute %s over the rows of %s: %w", rule.ID, attribute, relation, err)
	}
	return reading, nil
}

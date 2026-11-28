package validationeval

import (
	"fmt"

	"tei/internal/semantic"
)

// checkPresenceRowCases asks a presence-over-rows implementation about every row
// example the contract states, so an implementation that answers one of them
// differently is not an implementation of this operator and is never asked about
// real data (section 8.1).
//
// A row example states, per input, one cell per row: `present` when the row holds
// a value, `absent` when it holds none, or the word for "not read". The last is
// kept as that state rather than read as an answer, because a row nobody read is
// no answer at all (section 5).
func checkPresenceRowCases(contract semantic.Operator, implementation OperatorImplementation, inputs []semantic.OperatorInputDeclaration) error {
	if len(contract.Cases) > 0 {
		return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s reads whether an attribute holds a value in every row, so its meaning is stated by a table of rows; a one-record example is not checked against it", contract.ID)
	}
	for _, example := range contract.RowCases {
		rows := make(map[string][]PresenceReading, len(inputs))
		for _, input := range inputs {
			name := input.Name
			if name == "" {
				name = "value"
			}
			column := make([]PresenceReading, 0, len(example.Rows))
			for index, row := range example.Rows {
				text, stated := row[name]
				if !stated {
					return fmt.Errorf("operator %s row example states no value for input %q on row %d", contract.ID, name, index+1)
				}
				reading, err := presenceRowCell(text)
				if err != nil {
					return fmt.Errorf("operator %s has an unreadable row example (%s, row %d): %w", contract.ID, name, index+1, err)
				}
				column = append(column, reading)
			}
			rows[name] = column
		}
		answer, err := implementation.PresenceRows(rows)
		if err != nil {
			return fmt.Errorf("the implementation of operator %s could not decide a row example: %w", contract.ID, err)
		}
		want, ok := operatorRowExpectTri(example.Answer)
		if !ok {
			return fmt.Errorf("operator %s has a row example whose answer %q is not one this evaluator knows", contract.ID, example.Answer)
		}
		if answer != want {
			return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_MISMATCH: the contract of operator %s states that a row example is %s, but the selected implementation answers %s",
				contract.ID, example.Answer, operatorRowTriWord(answer))
		}
	}
	return nil
}

// presenceRowCell reads one cell of a presence row example. Section 8.3 fixes
// the two words, and the shared word for "not read" is the same state the
// numeric and typed row examples keep.
func presenceRowCell(text string) (PresenceReading, error) {
	switch text {
	case semantic.OperatorCasePresencePresent:
		return PresenceReading{State: ValuePresent}, nil
	case semantic.OperatorCasePresenceAbsent:
		return PresenceReading{State: ValueNull}, nil
	case semantic.RowCellNotFetched:
		return PresenceReading{State: ValueNotFetched}, nil
	}
	return PresenceReading{}, fmt.Errorf("%q is not a cell of a presence row example; write %s, %s or %s",
		text, semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent, semantic.RowCellNotFetched)
}

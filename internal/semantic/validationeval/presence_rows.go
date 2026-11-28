package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides a rule whose operator declares a presence input read over
// every row of a relation (decision sections 15.1 and 3):
//
//	operator sales.every_line_has_a_price:
//	  takes value: presence of each row
//	  refutes on some rows
//
// A presence input asks whether a row holds a value, not which value it holds.
// The answer is one of three states per row: the row holds a value, the row
// holds none, or the query did not return the column. Only the first two are
// readings of the row; the third is the read saying it did not look, so it is
// not turned into "no value".
//
// The rows are read once, for the whole relation, so every input of one rule
// comes from one Read Attempt (section 14.2 B-10). The operator is a pure
// function: it receives the three-valued state of each row and answers
// TriYes, TriNo or TriMaybe. The composition with Coverage and the operator's
// `refutes on some rows` statement happens in the one place that decides it
// (decideRowScoped, row_scope.go).

// PresenceRowsReading is what a reader knows about one presence input over the
// rows of a relation: the state of every row, in the order the rows were read,
// and whether those rows are every row of the relation.
type PresenceRowsReading struct {
	Coverage Coverage
	Rows     []PresenceReading
}

// StatefulPresenceRowSource is implemented by a reader that can answer whether
// every row of a relation holds a value for one attribute. A reader that cannot
// is refused rather than asked row by row: a rule whose input reads all the rows
// cannot be answered from one row.
type StatefulPresenceRowSource interface {
	ReadRelationPresence(relation, attribute core.CanonicalID) (PresenceRowsReading, error)
}

// readPresenceRows reads one presence input over the rows of a relation. A
// reader that cannot is an error, not an undecided rule: the reader lacking the
// read is a property of the reader, while an unread row is a property of this
// relation.
func readPresenceRows(rule semantic.Validation, input semantic.OperatorInputDeclaration, source StatefulSource, relation, attribute core.CanonicalID) (PresenceRowsReading, error) {
	reader, ok := source.(StatefulPresenceRowSource)
	if !ok {
		return PresenceRowsReading{}, fmt.Errorf("semantic validation %s reads operator %s input %q as presence over the rows of %s, but this reader cannot answer whether a row holds a value",
			rule.ID, rule.Operator, input.Name, relation)
	}
	reading, err := reader.ReadRelationPresence(relation, attribute)
	if err != nil {
		return PresenceRowsReading{}, fmt.Errorf("semantic validation %s cannot ask whether attribute %s of relation %s holds a value: %w",
			rule.ID, attribute, relation, err)
	}
	return reading, nil
}

// decidePresenceRowScopedOperator decides a rule whose operator reads whether
// every row of a relation holds a value.
//
// The first input's column is the rule's subject attribute. Every later input
// must be assigned by name to an attribute of the same relation. A constant is
// refused rather than expanded to every row: a constant belongs to no record,
// so a presence question about each row cannot be asked of it.
//
// A row that holds no value is a reading of the relation, so it is handed to
// the implementation as it is and the operator's own statement decides what a
// counterexample is worth (section 15.2). A row nobody read is not a reading:
// nothing is known about it, including whether a value is there, so it is
// reported as the rule's undecided reason rather than handed over as "no
// value" (section 8.6's table and section 5).
//
// Every input's column must come from one read of one relation: the origin of
// each attribute input is collected and compared before the implementation is
// asked, so one answer never mixes two reads (section 14.2 B-10).
func decidePresenceRowScopedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}
	if implementation.PresenceRows == nil {
		return Decision{}, fmt.Errorf(
			"operator %s reads whether every row of a relation holds a value, but its selected implementation does not decide over rows",
			rule.Operator)
	}

	rows := make(map[string][]PresenceReading, len(inputs))
	attributes := make(map[string]core.CanonicalID, len(inputs))
	origins := make([]inputOrigin, 0, len(inputs))
	var coverage Coverage
	for index, input := range inputs {
		attribute := rule.Attribute
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the constant %q, but a presence input asks the rows of relation %s whether they hold a value; assign an attribute",
					rule.ID, rule.Operator, input.Name, assignment.Value, rule.Relation)
			}
			attribute = core.CanonicalID(assignment.Value)
		}
		attributes[input.Name] = core.CanonicalID(attribute)
		origins = append(origins, dataOrigin(input.Name, string(attribute), source))
		reading, err := readPresenceRows(rule, input, source, core.CanonicalID(rule.Relation), attribute)
		if err != nil {
			return Decision{}, err
		}
		// Every input of this one rule is read from this one relation, so all of
		// them report the same coverage. Two coverages would mean two reads, and
		// one answer must not mix them.
		if coverage != "" && reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s read relation %s with two coverages (%q and %q); one answer cannot mix two reads",
				rule.ID, rule.Relation, coverage, reading.Coverage)
		}
		coverage = reading.Coverage
		rows[input.Name] = reading.Rows
	}
	// A row nobody read leaves the rule undecided before the implementation is
	// asked: what is unknown could be a value or no value, and telling the
	// implementation that nothing is there would answer a question nobody
	// asked (section 5).
	unknown, err := requireSameAttempt(rule, origins)
	if err != nil {
		return Decision{}, err
	}
	if unknown != nil {
		return Decision{Undecided: unknown}, nil
	}
	if reason := firstNotFetchedPresenceRow(attributes, rows, inputs); reason != nil {
		return Decision{Undecided: reason}, nil
	}

	answer, err := implementation.PresenceRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, nil)
}

// firstNotFetchedPresenceRow reports the first row that was not read, in input
// order and then in read order, so an undecided rule names the attribute it
// stopped at. A row that holds no value is not reported here: it was read and
// states something, and the operator decides what that is worth.
func firstNotFetchedPresenceRow(attributes map[string]core.CanonicalID, rows map[string][]PresenceReading, inputs []semantic.OperatorInputDeclaration) *Undecided {
	for _, input := range inputs {
		for _, row := range rows[input.Name] {
			if row.State == ValueNotFetched {
				return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attributes[input.Name]}
			}
		}
	}
	return nil
}

// checkPresenceRowCases asks a row-scoped presence implementation about every
// row example the contract states and refuses it at the first one it answers
// differently (section 8.6, section 8.1: an implementation that answers an
// example differently is not an implementation of this operator, and real data
// is not read for it).
//
// The example's cells are the states as they would have been read, so the
// implementation is asked exactly the map the rule path would hand it. This is
// not checkOperatorRowCases: that reader reads a cell in the type its input
// declares, and a presence input has no value of any type to read.
func checkPresenceRowCases(contract semantic.Operator, implementation OperatorImplementation) error {
	if len(contract.RowCases) == 0 {
		return nil
	}
	if implementation.PresenceRows == nil {
		return fmt.Errorf("operator %s states row examples, but its selected implementation does not decide over rows", contract.ID)
	}
	inputs := contractInputs(contract)
	for _, example := range contract.RowCases {
		rows, err := presenceRowExampleColumns(contract, inputs, example)
		if err != nil {
			return err
		}
		want, ok := operatorRowExpectTri(example.Answer)
		if !ok {
			return fmt.Errorf("operator %s has a row example whose answer %q is not one this evaluator knows", contract.ID, example.Answer)
		}
		answer, err := implementation.PresenceRows(rows)
		if err != nil {
			return fmt.Errorf("the implementation of operator %s could not decide a row example: %w", contract.ID, err)
		}
		if answer != want {
			return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_MISMATCH: the contract of operator %s states that a row example is %s, but the selected implementation answers %s",
				contract.ID, example.Answer, operatorRowTriWord(answer))
		}
	}
	return nil
}

// presenceRowExampleColumns reads one row example's cells as the presence
// states an implementation that decides over rows is asked about.
//
// Section 8.6: a cell states one of the three words, and the state is handed to
// the implementation as it was written, so the example asks it exactly the
// question a rule read would. A column the row does not state is refused: the
// example would ask about a row nobody wrote, which is the refusal the other
// row-example readers make.
func presenceRowExampleColumns(contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, example semantic.OperatorRowCase) (map[string][]PresenceReading, error) {
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
				return nil, fmt.Errorf("operator %s row example states no cell for input %q on row %d", contract.ID, name, index+1)
			}
			reading, err := parsePresenceRowExampleCell(text)
			if err != nil {
				return nil, fmt.Errorf("operator %s has an unreadable row example (%s, row %d): %w", contract.ID, name, index+1, err)
			}
			column = append(column, reading)
		}
		rows[name] = column
	}
	return rows, nil
}

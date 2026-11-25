package validationeval

import (
	"fmt"
	"math/big"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides an operator rule whose inputs are not all numbers.
//
// The numeric path (reading.go) is unchanged: a contract that declares
// `number` for every input keeps Number, Numbers and NumberRows, and its
// implementation is not asked to change. A contract that declares `date`,
// `text`, `truth` or `any` for any input is decided here, through the values
// section 8.4 fixed, so an implementation is handed the type the declaration
// states and never a number that happens to be there.
//
// The two paths are not merged and one implementation does not serve both: a
// contract declares one kind of input, and an implementation that offered both
// would leave it ambiguous which value it answered from.

// decideTypedOperator decides one operator rule about one record whose inputs
// include a value that is not a number.
func decideTypedOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if implementation.Values == nil {
		return Decision{}, fmt.Errorf(
			"operator %s takes a value that is not a number, but its selected implementation does not decide from typed values",
			rule.Operator)
	}
	// The rule subject supplies the first input, and only it. A rule that
	// also assigns that input by name binds it twice, so the instruction
	// is refused instead of read.
	for _, assignment := range rule.Inputs {
		if assignment.Name == inputs[0].Name {
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q, which the rule subject already supplies", rule.ID, rule.Operator, assignment.Name)
		}
	}
	values := make(map[string]OperatorValue, len(inputs))
	// The subject and every attribute-assigned input must belong to one
	// Read Attempt, exactly as on the numeric path (section 14.2 B-10).
	origins := []inputOrigin{dataOrigin(inputs[0].Name, string(rule.Attribute), source)}
	value, stopped, err := readTypedValue(rule, inputs[0], rule.Attribute, inputs[0].AbsentAs, source, implementation.Absent)
	if err != nil {
		return Decision{}, err
	}
	if stopped != nil {
		return Decision{Undecided: stopped}, nil
	}
	values[inputs[0].Name] = *value
	for _, input := range inputs[1:] {
		// Every input after the first must be assigned by name; one that
		// is not is not silently read as the subject's value or as zero.
		assignment := findInputAssignment(rule, input.Name)
		if assignment == nil {
			return Decision{}, fmt.Errorf("semantic validation %s does not assign operator %s input %q; every input after the first must be assigned", rule.ID, rule.Operator, input.Name)
		}
		if isAttributeReference(assignment.Value) {
			named, aerr := core.ParseCanonicalID(assignment.Value)
			if aerr != nil {
				return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q an unreadable attribute %q", rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			origins = append(origins, dataOrigin(input.Name, assignment.Value, source))
			got, stopped, rerr := readTypedValue(rule, input, named, assignment.AbsentAs, source, implementation.Absent)
			if rerr != nil {
				return Decision{}, rerr
			}
			if stopped != nil {
				return Decision{Undecided: stopped}, nil
			}
			if got == nil {
				return Decision{Undecided: &Undecided{Reason: UndecidedValueNotFetched, Attribute: named}}, nil
			}
			values[input.Name] = *got
			continue
		}
		origins = append(origins, constantOrigin(input.Name))
		constant, cerr := parseTypedConstant(input, assignment.Value)
		if cerr != nil {
			return Decision{}, fmt.Errorf("semantic validation %s assigns operator %s input %q the value %q, which is not one of that input's type: %w", rule.ID, rule.Operator, input.Name, assignment.Value, cerr)
		}
		values[input.Name] = *constant
	}
	unknown, aerr := requireSameAttempt(rule, origins)
	if aerr != nil {
		return Decision{}, aerr
	}
	if unknown != nil {
		return Decision{Undecided: unknown}, nil
	}
	satisfied, err := implementation.Values(values)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decided(satisfiedResult(satisfied))
}

// readTypedValue reads one value of the declared type from one attribute.
//
// It returns (nil, nil, nil) for a value that was not read, so the caller can
// tell "no value" from "the reader failed". The absent mapping is applied only
// where the value is a number: a mapping states how an absent number is read
// (`absent as zero`), and no mapping states how an absent date, text or boolean
// is read, so those absences keep their own reason rather than being read as a
// value nobody stated.
func readTypedValue(rule semantic.Validation, input semantic.OperatorInputDeclaration, attribute core.CanonicalID, mapping string, source StatefulSource, absent func(string) (*big.Rat, bool)) (*OperatorValue, *Undecided, error) {
	switch input.Type {
	case semantic.OperatorInputNumber:
		value, stopped, err := readNumberForAttribute(rule, attribute, source)
		if err != nil {
			return nil, nil, err
		}
		if stopped != nil {
			if stopped.Reason == UndecidedValueNull && mapping != "" && absent != nil {
				if mapped, ok := absent(mapping); ok {
					return &OperatorValue{Number: new(big.Rat).Set(mapped)}, nil, nil
				}
			}
			return nil, stopped, nil
		}
		return &OperatorValue{Number: new(big.Rat).Set(value)}, nil, nil
	case semantic.OperatorInputDate:
		reader, ok := source.(StatefulDateSource)
		if !ok {
			return nil, nil, errNoValueReader(rule, input)
		}
		reading, err := reader.ReadDate(attribute)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation %s cannot read attribute %s: %w", rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			day := reading.Value
			return &OperatorValue{Date: &day}, nil, nil
		case ValueNull, ValueNotFetched:
			return nil, absentReason(reading.State, attribute), nil
		default:
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	case semantic.OperatorInputText:
		reader, ok := source.(StatefulTextSource)
		if !ok {
			return nil, nil, errNoValueReader(rule, input)
		}
		reading, err := reader.ReadText(attribute)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation %s cannot read attribute %s: %w", rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			if reading.Value == nil {
				return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, attribute)
			}
			text := *reading.Value
			return &OperatorValue{Text: &text}, nil, nil
		case ValueNull, ValueNotFetched:
			return nil, absentReason(reading.State, attribute), nil
		default:
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	case semantic.OperatorInputTruth:
		reader, ok := source.(StatefulTruthSource)
		if !ok {
			return nil, nil, errNoValueReader(rule, input)
		}
		reading, err := reader.ReadTruth(attribute)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation %s cannot read attribute %s: %w", rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			if reading.Value == nil {
				return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, attribute)
			}
			truth := *reading.Value
			return &OperatorValue{Truth: &truth}, nil, nil
		case ValueNull, ValueNotFetched:
			return nil, absentReason(reading.State, attribute), nil
		default:
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	case semantic.OperatorInputAny:
		reader, ok := source.(StatefulAnySource)
		if !ok {
			return nil, nil, errNoValueReader(rule, input)
		}
		reading, err := reader.ReadAny(attribute)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation %s cannot read attribute %s: %w", rule.ID, attribute, err)
		}
		switch reading.State {
		case ValuePresent:
			if reading.Value == nil {
				return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, attribute)
			}
			return reading.Value, nil, nil
		case ValueNull, ValueNotFetched:
			return nil, absentReason(reading.State, attribute), nil
		default:
			return nil, nil, fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
		}
	}
	return nil, nil, fmt.Errorf("semantic validation %s reads operator %s input %q as %s, which states whether a value is there and not a value",
		rule.ID, rule.Operator, input.Name, input.Type)
}

// parseTypedConstant reads a literal the rule assigned to an input as a value
// of that input's type.
func parseTypedConstant(input semantic.OperatorInputDeclaration, text string) (*OperatorValue, error) {
	cell, err := semantic.ParseOperatorCaseCell(text)
	if err != nil {
		return nil, err
	}
	value, err := OperatorValueFor(input.Type, cell)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// errNoValueReader is the error a read gives when the reader has no boundary
// for the input's type. requireValueReaders refuses such a rule before any
// read is attempted, so this is a reader that changed under the rule rather
// than a declaration nobody can read.
func errNoValueReader(rule semantic.Validation, input semantic.OperatorInputDeclaration) error {
	return fmt.Errorf("semantic validation %s reads operator %s input %q as %s, but this reader cannot read a value of that type; the %s input has no reader on this path yet",
		rule.ID, rule.Operator, input.Name, input.Type, input.Type)
}

// decideRowScopedValueOperator decides one operator rule over every row of a
// relation whose inputs include a value that is not a number. It mirrors the
// numeric row path: one call per input, one coverage, one answer.
func decideRowScopedValueOperator(rule semantic.Validation, contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, implementation OperatorImplementation, source StatefulSource) (Decision, error) {
	if rule.Relation == "" {
		return Decision{}, fmt.Errorf(
			"semantic validation %s uses operator %s, which reads every row, but the rule names no relation; write \"per <relation>\"",
			rule.ID, rule.Operator)
	}
	reader, ok := source.(StatefulRowValueSource)
	if !ok {
		// A reader that can only read rows as numbers is not asked for a
		// typed column: it would have to invent a value of the declared type.
		return Decision{}, fmt.Errorf(
			"semantic validation %s reads every row of relation %s, but this reader cannot read the rows of a relation as values of the declared types",
			rule.ID, rule.Relation)
	}
	if implementation.ValueRows == nil {
		return Decision{}, fmt.Errorf(
			"operator %s reads every row of a relation, but its selected implementation does not decide over rows",
			rule.Operator)
	}

	rows := make(map[string][]ValueReading, len(inputs))
	var coverage Coverage
	var reason *Undecided
	// A constant states one value for the whole relation. Its column is as
	// long as the rows that were read, so the value is written into every row
	// once the row count is known.
	constantValues := map[string]OperatorValue{}
	var width []ValueReading
	for index, input := range inputs {
		attribute := rule.Attribute
		// The declaration's `absent as <mapping>` applies to the subject; a
		// named input's mapping is the one the rule states for it.
		mapping := ""
		if index == 0 {
			mapping = input.AbsentAs
		}
		if index > 0 {
			assignment := findInputAssignment(rule, input.Name)
			if assignment == nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s does not assign operator %s input %q; every input after the first must be assigned",
					rule.ID, rule.Operator, input.Name)
			}
			if !isAttributeReference(assignment.Value) {
				constant, err := parseTypedConstant(input, assignment.Value)
				if err != nil {
					return Decision{}, fmt.Errorf(
						"semantic validation %s assigns operator %s input %q the value %q, which is not one of that input's type: %w",
						rule.ID, rule.Operator, input.Name, assignment.Value, err)
				}
				constantValues[input.Name] = *constant
				continue
			}
			named, err := core.ParseCanonicalID(assignment.Value)
			if err != nil {
				return Decision{}, fmt.Errorf(
					"semantic validation %s assigns operator %s input %q an unreadable attribute %q",
					rule.ID, rule.Operator, input.Name, assignment.Value)
			}
			attribute = named
			if assignment.AbsentAs != "" {
				mapping = assignment.AbsentAs
			}
		}
		reading, err := reader.ReadRelationValues(rule.Relation, attribute)
		if err != nil {
			return Decision{}, fmt.Errorf(
				"semantic validation %s cannot read attribute %s over relation %s: %w",
				rule.ID, attribute, rule.Relation, err)
		}
		// Every input of this one rule is read from this one relation, so all
		// of them report the same coverage. Two coverages would mean two reads,
		// and one answer must not mix them.
		if coverage != "" && reading.Coverage != coverage {
			return Decision{}, fmt.Errorf(
				"semantic validation %s read relation %s with two coverages (%q and %q); one answer cannot mix two reads",
				rule.ID, rule.Relation, coverage, reading.Coverage)
		}
		coverage = reading.Coverage
		rows[input.Name] = withValueAbsentMapping(reading.Rows, mapping, implementation.Absent)
		if index == 0 {
			width = rows[input.Name]
		}
		// A row that holds no value, or was not read, keeps its own reason. The
		// first such row in read order is reported; the implementation is still
		// asked, so a counterexample found in another row is not thrown away.
		if reason == nil {
			reason = firstRowValueReason(attribute, rows[input.Name])
		}
	}
	for name, value := range constantValues {
		rows[name] = fillValueConstant(width, value)
	}

	answer, err := implementation.ValueRows(rows)
	if err != nil {
		return Decision{}, fmt.Errorf("operator %s could not decide semantic validation %s: %w", rule.Operator, rule.ID, err)
	}
	return decideRowScoped(rule, coverage, contract.RefutesOnSomeRows, answer, reason)
}

// withValueAbsentMapping rewrites the rows that hold no value under the mapping
// the rule named, when the implementation can read that mapping and the value
// is a number. A row whose value is replaced becomes present with the mapped
// value; every other absence keeps its own reason instead of being read as a
// value nobody stated.
func withValueAbsentMapping(rows []ValueReading, mapping string, absent func(string) (*big.Rat, bool)) []ValueReading {
	if mapping == "" || absent == nil {
		return rows
	}
	mapped, ok := absent(mapping)
	if !ok {
		return rows
	}
	out := make([]ValueReading, 0, len(rows))
	for _, row := range rows {
		if row.State == ValueNull && row.Value != nil && row.Value.Number != nil {
			out = append(out, ValueReading{State: ValuePresent, Value: &OperatorValue{Number: new(big.Rat).Set(mapped)}})
			continue
		}
		out = append(out, row)
	}
	return out
}

// fillValueConstant writes one constant value into every row that was read, so
// a constant column is the same length as the columns read from the relation.
func fillValueConstant(width []ValueReading, value OperatorValue) []ValueReading {
	out := make([]ValueReading, 0, len(width))
	for range width {
		row := ValueReading{State: ValuePresent, Value: &value}
		out = append(out, row)
	}
	return out
}

// firstRowValueReason is the reason of the first row that holds no value or was
// not read, in read order.
func firstRowValueReason(attribute core.CanonicalID, rows []ValueReading) *Undecided {
	for _, row := range rows {
		if row.State == ValuePresent {
			if row.Value == nil {
				return &Undecided{Reason: UndecidedValueNull, Attribute: attribute}
			}
			continue
		}
		if row.State == ValueNull {
			return &Undecided{Reason: UndecidedValueNull, Attribute: attribute}
		}
		if row.State == ValueNotFetched {
			return &Undecided{Reason: UndecidedValueNotFetched, Attribute: attribute}
		}
	}
	return nil
}

package validationeval

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/semantic"
)

// OperatorImplementation is one implementation of a declared operator.
//
// An implementation is a pure function of the values the declaration says the
// operator takes. It receives no Source, so it cannot read another attribute,
// another row, a clock, or anything else: what a rule reads is decided by the
// operator declaration, never by the code that implements it.
//
// For a single-input operator, Number is used. For a multi-input operator,
// Numbers is used. Exactly one of them is non-nil for any implementation.
type OperatorImplementation struct {
	// Operator is the declared operator this implements.
	Operator core.CanonicalID
	// Input is the first input's type, kept for backward compatibility with
	// single-input operators.
	Input semantic.OperatorInput
	// Number decides a rule for a single-input operator. It receives a private
	// copy of the value.
	Number func(value *big.Rat) (bool, error)
	// Numbers decides a rule for a multi-input operator. It receives the input
	// name to value map. Values are non-nil only for inputs that are present.
	Numbers func(values map[string]*big.Rat) (bool, error)
	// NumberRows decides a rule whose operator reads every row of a relation.
	// It receives, per input name, the states of the rows that were read in
	// the order they were read, and answers TriYes, TriNo or TriMaybe. It sees
	// no Source, so what a rule reads stays decided by the declaration.
	NumberRows func(rows map[string][]NumberReading) (Tri, error)
	// Values decides a rule for an operator whose inputs are of any type. It
	// receives one value per input name, each carrying the type it was read as
	// (decision section 8.4). A value is present only for an input the reader
	// read, so an implementation never has to tell a value nobody holds from
	// one nobody read.
	//
	// An implementation provides either this field or one of the numeric ones,
	// never both: providing both would leave which value it answers from
	// ambiguous.
	Values func(values map[string]semantic.OperatorValue) (bool, error)
	// ValueRows decides a rule whose operator reads every row of a relation in
	// any type. It receives, per input name, the rows that were read in the
	// order they were read, each value carrying the type it was read as
	// (decision section 8.4).
	//
	// An implementation provides either this field or one of the others, never
	// both: providing both would leave which value it answers from ambiguous.
	ValueRows func(rows map[string][]RowValueReading) (Tri, error)
	// Presence decides a rule whose operator declares an input that reads
	// whether an attribute holds a value rather than which value it holds
	// (decision section 3: "presence reads whether, not what"). It receives,
	// per input name, whether the attribute held a value: true when the record
	// holds one, false when the record holds none. A value nobody read is not
	// passed at all, so an implementation never has to tell "no value" from
	// "not read".
	//
	// A presence value is not a value of any type, so it does not travel in
	// OperatorValue: a truth is what a column says, and presence is whether
	// the column says anything. Giving presence a field there would let an
	// implementation answer a question about a value from a record that holds
	// none.
	Presence func(present map[string]bool) (bool, error)
	// PresenceRows decides a rule whose operator reads whether every row of a
	// relation holds a value (decision section 15.1). It receives, per input
	// name, the state of every row in the order the rows were read and answers
	// TriYes, TriNo or TriMaybe. It sees no Source, so what a rule reads stays
	// decided by the declaration.
	//
	// An implementation provides either this field or one of the others, never
	// both: providing both would leave which value it answers from ambiguous.
	PresenceRows func(rows map[string][]PresenceReading) (Tri, error)
	// Absent reads an absent value under the mapping the rule named. A rule may
	// write `or <name>` after an input, and the declaration may state
	// `absent as <name>`; the mapping says how an absent value is to be read.
	//
	// It is asked only when the input held no value and the rule named a
	// mapping. A nil Absent, or ok=false for that mapping, means this
	// implementation cannot read the absence that way, and the decision stays
	// undecided instead of reading the absence as zero or as present.
	Absent func(mapping string) (*big.Rat, bool)
}

// Operators is what an evaluation knows about declared operators: the
// declarations of the loaded model and the implementations the composition
// selected. Neither list is searched by position; identity decides.
type Operators struct {
	Declared        []semantic.Operator
	Implementations []OperatorImplementation
}

// EvaluateWithOperators evaluates rule like Evaluate and additionally decides
// rules of kind operator.
//
// An operator rule is evaluated only when exactly one declaration and exactly
// one implementation name its operator. A missing, duplicated, or mismatched
// implementation is an error: it is never replaced by another one and never
// reported as satisfied or not satisfied.
func EvaluateWithOperators(rule semantic.Validation, source Source, operators Operators) (Result, error) {
	if rule.Kind != semantic.ValidationOperator {
		return Evaluate(rule, source)
	}
	if source == nil {
		return "", fmt.Errorf("no Source is specified to evaluate semantic validation %s", rule.ID)
	}
	decision, err := decide(rule, legacySource{source}, operators, true)
	if err != nil {
		return "", err
	}
	return decision.Result, nil
}

// selectOperatorImplementation finds the one implementation an operator rule
// is decided with. It reads no real data.
func selectOperatorImplementation(rule semantic.Validation, operators Operators) (OperatorImplementation, error) {
	none := OperatorImplementation{}
	if rule.Operator == "" {
		return none, fmt.Errorf("semantic validation %s names no operator", rule.ID)
	}

	var declared []semantic.Operator
	for _, operator := range operators.Declared {
		if operator.ID == rule.Operator {
			declared = append(declared, operator)
		}
	}
	switch {
	case len(declared) == 0:
		return none, fmt.Errorf("semantic validation %s uses operator %s, which is not declared", rule.ID, rule.Operator)
	case len(declared) > 1:
		return none, fmt.Errorf("semantic validation %s uses operator %s, which is declared %d times", rule.ID, rule.Operator, len(declared))
	}
	contract := declared[0]

	var implementations []OperatorImplementation
	for _, implementation := range operators.Implementations {
		if implementation.Operator == rule.Operator {
			implementations = append(implementations, implementation)
		}
	}
	switch {
	case len(implementations) == 0:
		return none, fmt.Errorf("semantic validation %s uses operator %s, which has no selected implementation", rule.ID, rule.Operator)
	case len(implementations) > 1:
		return none, fmt.Errorf("semantic validation %s uses operator %s, which has %d implementations; exactly one must be selected", rule.ID, rule.Operator, len(implementations))
	}
	implementation := implementations[0]
	if err := CheckImplementationMatchesContract(contract, implementation); err != nil {
		return none, err
	}
	return implementation, nil
}

// CheckImplementationMatchesContract reports whether one selected
// implementation can be the implementation of one declared operator: the same
// kind of input, one decision function of the shape the declaration asks for,
// and examples it does not contradict.
//
// It is exported because validate must refuse a Workspace whose selection this
// binary cannot honour before any rule is read, and the evaluator must refuse
// it when a rule is read. Both ask the same question, so both call this one
// function: a change to what "matches" means cannot make validate accept a
// Workspace the evaluator would refuse, or the other way round.
func CheckImplementationMatchesContract(contract semantic.Operator, implementation OperatorImplementation) error {
	if implementation.Input != contract.Input {
		return fmt.Errorf("operator %s is declared to take %s, but its implementation takes %s", contract.ID, contract.Input, implementation.Input)
	}

	// Reject an implementation that does not match the shape of the declaration.
	// A declaration that reads a column of the rows is decided over those rows;
	// any other declaration is decided from the values of one record, whether it
	// names one input or several.
	inputs := contractInputs(contract)
	rowScoped := false
	for _, input := range inputs {
		if input.OfEachRow {
			rowScoped = true
			break
		}
	}
	hasRows := implementation.NumberRows != nil
	hasMulti := implementation.Numbers != nil
	hasSingle := implementation.Number != nil
	hasTyped := implementation.Values != nil
	hasTypedRows := implementation.ValueRows != nil
	hasPresence := implementation.Presence != nil
	hasPresenceRows := implementation.PresenceRows != nil
	shapes := 0
	for _, present := range []bool{hasRows, hasMulti, hasSingle, hasTyped, hasTypedRows, hasPresence, hasPresenceRows} {
		if present {
			shapes++
		}
	}
	if shapes == 0 {
		return fmt.Errorf("the implementation of operator %s has no decision function", contract.ID)
	}
	if shapes > 1 {
		// Which values an implementation answers about must not be ambiguous.
		// One that provides a number function and a typed one would answer a
		// rule from whichever the evaluator happened to fill.
		return fmt.Errorf("the implementation of operator %s provides more than one kind of decision function", contract.ID)
	}
	// A presence input asks whether an attribute holds a value, and a value
	// input asks which value it holds. An implementation of one kind cannot
	// answer the other: an implementation deciding values would be handed the
	// number a presence rule never asked about, which is the mistake #4359
	// measured (a presence contract answered satisfied from the value 7).
	//
	// A presence input over the rows of a relation is decided by the row-scoped
	// presence field, not by the one-record presence field: a column of rows is
	// not one record's answer (sections 3 and 15.1).
	//
	// Section 8.6 fixes the cells of a presence row-example table: they write
	// whether a row holds a value (present, absent, not-fetched), so a table
	// for a presence input states the operator's meaning and is checked.
	if rowScoped && declaresPresenceInput(inputs) {
		if !hasPresenceRows {
			return fmt.Errorf("operator %s reads whether every row of a relation holds a value, but its implementation does not decide over rows", contract.ID)
		}
		if len(contract.Cases) > 0 {
			return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s reads every row of a relation, so its meaning is stated by a table of rows; a one-record example is not checked against it", contract.ID)
		}
		// A row-example table for a presence input is read: section 8.6 fixes
		// the cells to present, absent and not-fetched, so the example states
		// the rows and is checked against the implementation like any other
		// row example.
		return checkOperatorRowCases(contract, implementation)
	}
	if declaresPresenceInput(inputs) && !hasPresence {
		return fmt.Errorf("operator %s declares the presence input %q, but its implementation decides values; presence asks whether a value is there, not which value it is",
			contract.ID, presenceInputName(inputs))
	}
	if !declaresPresenceInput(inputs) && hasPresence {
		return fmt.Errorf("operator %s declares no presence input, but its implementation decides whether a value is there", contract.ID)
	}
	// A presence implementation answers every input with "does this record
	// hold a value?", so a contract that also declares a value input would have
	// that input answered the same way, its value never read. The two
	// questions cannot be asked in one signature on this path, so it is
	// refused rather than answered from a value nobody read (#4359).
	//
	// The same holds over a relation's rows: the row implementation answers
	// every input with "does every row hold a value?", so a value input beside
	// a row-presence input is refused too. It is checked here, before either
	// path decides a rule.
	if declaresPresenceInput(inputs) {
		for _, input := range inputs {
			if input.Type != semantic.OperatorInputPresence {
				return fmt.Errorf("operator %s declares the %s input %q beside a presence input; a presence implementation answers whether a value is there for every input, so the two cannot share one signature",
					contract.ID, input.Type, input.Name)
			}
		}
	}
	// A row implementation with no presence input beside it is the other
	// mismatch: it answers a question the declaration never asked.
	if !declaresPresenceInput(inputs) && hasPresenceRows {
		return fmt.Errorf("operator %s declares no presence input, but its implementation decides whether every row of a relation holds a value", contract.ID)
	}
	if rowScoped && !hasRows && !hasTypedRows {
		return fmt.Errorf("operator %s reads every row of a relation but its implementation decides from the values of one record", contract.ID)
	}
	if !rowScoped && hasRows {
		return fmt.Errorf("operator %s reads no rows of a relation but its implementation decides over rows", contract.ID)
	}
	if !rowScoped && hasTypedRows {
		return fmt.Errorf("operator %s reads no rows of a relation but its implementation decides over rows of any type", contract.ID)
	}
	// An implementation that reads values of any type (section 8.4) decides
	// both a single-input and a multi-input contract: the declaration's types
	// are what decide which value each input gets, not how many inputs there
	// are. A contract that declares a calendar day is one of those, so it is
	// decided by the typed field and never by a number one.
	if !rowScoped && !hasTyped && !hasPresence {
		multiInput := len(inputs) > 1
		if multiInput && !hasMulti {
			return fmt.Errorf("operator %s declares %d inputs but its implementation takes a single value", contract.ID, len(inputs))
		}
		if !multiInput && hasMulti {
			return fmt.Errorf("operator %s declares one input but its implementation takes multiple values", contract.ID)
		}
		if multiInput && hasSingle {
			return fmt.Errorf("operator %s declares %d inputs but its implementation takes a single value", contract.ID, len(inputs))
		}
	}

	// The contract's own examples come first. An implementation that
	// answers one of them differently is not an implementation of this
	// operator, so it is not asked about real data.
	//
	// An example states the values of one record. A row-scoped operator's
	// meaning is stated by the table of rows (section 8), which is not read
	// yet, so the examples of such a contract are refused rather than checked
	// against an implementation that would be asked a different question.
	if rowScoped {
		if len(contract.Cases) > 0 {
			return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s reads every row of a relation, so its meaning is stated by a table of rows; a one-record example is not checked against it", contract.ID)
		}
		return checkOperatorRowCases(contract, implementation)
	}

	// The contract's own examples come first. An implementation that
	// answers one of them differently is not an implementation of this
	// operator, so it is not asked about real data.
	//
	// A presence implementation is asked about the two words an example
	// states, not about a decimal it never reads: whether a record holds a
	// value is a different question from which value it holds.
	if hasPresence {
		return checkPresenceOperatorCases(contract, implementation, inputs)
	}
	if err := checkOperatorCases(contract, implementation); err != nil {
		return err
	}
	return nil
}

// checkOperatorCases asks the implementation about every example the contract
// states and refuses the implementation at the first different answer.
//
// An example carries one value, so it can state the meaning of a single-input
// operator only. A multi-input contract with examples is refused rather than
// read as if its one value bound the first input: the remaining inputs would
// go unstated and the example would say something the author never wrote.
func checkOperatorCases(contract semantic.Operator, implementation OperatorImplementation) error {
	inputs := contractInputs(contract)
	multi := len(inputs) > 1
	for _, example := range contract.Cases {
		// An example states either one value or one value per declared input.
		// Which form it uses must match the signature: a multi-input example
		// read as a single value would leave the other inputs unstated, and a
		// single-input operator given per-input columns would be read as
		// something the author never wrote.
		if example.Inputs != nil {
			if !multi {
				// The design says the table's columns are the operator's input
				// names, so a single-input operator may write `| <name> | expect |`.
				// That is the single-value form with the input named by its own
				// name, not a multi-input example.
				if len(example.Inputs) == 1 {
					if value, ok := example.Inputs[inputs[0].Name]; ok {
						example = semantic.OperatorCase{Value: value, Satisfied: example.Satisfied}
					} else {
						return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares the input %q, but an example states %s",
							contract.ID, inputs[0].Name, operatorExampleNames(example.Inputs))
					}
				} else {
					return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares one input, but an example states the inputs %s",
						contract.ID, operatorExampleNames(example.Inputs))
				}
			} else {
				for _, input := range inputs {
					if _, stated := example.Inputs[input.Name]; !stated {
						return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s example states %s, but input %q is not stated; an example that leaves an input unread cannot be checked",
							contract.ID, operatorExampleNames(example.Inputs), input.Name)
					}
				}
				for name := range example.Inputs {
					known := false
					for _, input := range inputs {
						if input.Name == name {
							known = true
						}
					}
					if !known {
						return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s example states %q, which is not one of its inputs (%s)",
							contract.ID, name, operatorInputNames(inputs))
					}
				}
			}
		} else if multi {
			return fmt.Errorf("TEI_OPERATOR_EXAMPLE_SHAPE_MISMATCH: operator %s declares %d inputs (%s), but its example carries one value; a multi-input example states one column per input",
				contract.ID, len(inputs), operatorInputNames(inputs))
		}

		// A typed implementation is asked about the values as the
		// declaration says they are read, so a text is a text and a number
		// is a number (section 8.3, 8.4). The example states the same
		// spelling for each input, so the reading here is the one the
		// author wrote.
		if implementation.Values != nil {
			typed := make(map[string]semantic.OperatorValue, len(inputs))
			for _, input := range inputs {
				text := example.Value
				if example.Inputs != nil {
					text = example.Inputs[input.Name]
				}
				value, err := semantic.ParseOperatorInputValue(input.Type, text)
				if err != nil {
					return fmt.Errorf("operator %s has an unreadable example for input %q, which is read as %s: %w", contract.ID, input.Name, input.Type, err)
				}
				typed[input.Name] = value
			}
			got, err := implementation.Values(typed)
			if err != nil {
				return fmt.Errorf("the implementation of operator %s could not decide the contract's example %s: %w", contract.ID, operatorExampleText(example), err)
			}
			if got != example.Satisfied {
				return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_MISMATCH: the contract of operator %s states that %s is %s, but the selected implementation answers %s",
					contract.ID, operatorExampleText(example), operatorCaseWord(example.Satisfied), operatorCaseWord(got))
			}
			continue
		}

		values := map[string]*big.Rat{}
		if example.Inputs != nil {
			for name, text := range example.Inputs {
				value, err := semantic.ParseOperatorCaseValue(text)
				if err != nil {
					return fmt.Errorf("operator %s has an unreadable example (%s): %w", contract.ID, name, err)
				}
				values[name] = value
			}
		} else {
			value, err := semantic.ParseOperatorCaseValue(example.Value)
			if err != nil {
				return fmt.Errorf("operator %s has an unreadable example: %w", contract.ID, err)
			}
			// A single-input operator may name its input (`takes amount: number`).
			// The example's one value belongs to that input, whatever it is
			// called, so the implementation is never handed a nil value.
			values[inputs[0].Name] = value
		}

		var (
			got bool
			err error
		)
		if implementation.Numbers != nil {
			got, err = implementation.Numbers(values)
		} else {
			got, err = implementation.Number(values[inputs[0].Name])
		}
		if err != nil {
			return fmt.Errorf("the implementation of operator %s could not decide the contract's example %s: %w", contract.ID, operatorExampleText(example), err)
		}
		if got != example.Satisfied {
			return fmt.Errorf("TEI_OPERATOR_IMPLEMENTATION_MISMATCH: the contract of operator %s states that %s is %s, but the selected implementation answers %s",
				contract.ID, operatorExampleText(example), operatorCaseWord(example.Satisfied), operatorCaseWord(got))
		}
	}
	return nil
}

// operatorExampleNames lists the inputs an example states, in declaration
// order when the contract names them, for a stable diagnostic.
func operatorExampleNames(stated map[string]string) string {
	names := make([]string, 0, len(stated))
	for name := range stated {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// operatorExampleText renders one example for a diagnostic.
func operatorExampleText(example semantic.OperatorCase) string {
	if example.Inputs == nil {
		return example.Value
	}
	return "(" + operatorExampleNames(example.Inputs) + ")"
}

func operatorCaseWord(satisfied bool) Result {
	if satisfied {
		return ResultSatisfied
	}
	return ResultNotSatisfied
}

// operatorInputNames lists the declared input names in signature order, so a
// diagnostic can name the inputs an example would have to state.
func operatorInputNames(inputs []semantic.OperatorInputDeclaration) string {
	names := make([]string, 0, len(inputs))
	for _, input := range inputs {
		name := input.Name
		if name == "" {
			name = "value"
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// checkOperatorRowCases asks the implementation about every example the
// contract states for an operator that reads every row, and refuses it at the
// first one it answers differently (section 8.1: an implementation that
// answers an example differently is not an implementation of this operator,
// and real data is not read for it).
//
// The example's rows are the values as they would have been read, so the
// implementation is asked exactly the map the rule path would hand it: an
// implementation that decides numbers is asked the numeric columns, and one
// that reads values of any type is asked the columns in the type each input
// declares (section 8.3, 8.4).
func checkOperatorRowCases(contract semantic.Operator, implementation OperatorImplementation) error {
	if len(contract.RowCases) == 0 {
		return nil
	}
	if implementation.NumberRows == nil && implementation.ValueRows == nil && implementation.PresenceRows == nil {
		return fmt.Errorf("operator %s states row examples, but its selected implementation does not decide over rows", contract.ID)
	}
	inputs := contractInputs(contract)
	for _, example := range contract.RowCases {
		var (
			answer Tri
			err    error
		)
		switch {
		case implementation.PresenceRows != nil:
			// A presence example states, per row, whether the row holds a
			// value rather than which value it holds (section 8.6), so its
			// cells are read as the three row states and not as values.
			var rows map[string][]PresenceReading
			if rows, err = rowExamplePresenceColumns(contract, inputs, example); err == nil {
				answer, err = implementation.PresenceRows(rows)
			}
		case implementation.ValueRows != nil:
			var rows map[string][]RowValueReading
			if rows, err = rowExampleValueColumns(contract, inputs, example); err == nil {
				answer, err = implementation.ValueRows(rows)
			}
		default:
			var rows map[string][]NumberReading
			if rows, err = rowExampleNumberColumns(contract, inputs, example); err == nil {
				answer, err = implementation.NumberRows(rows)
			}
		}
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

// rowExamplePresenceColumns reads one example's rows as the presence column an
// implementation that decides whether a row holds a value is asked about.
//
// A presence cell writes whether the row holds a value rather than which value
// it holds (decision section 8.6): `present` and `absent` are the two words,
// the empty cell is refused because it already means "holds no value" for every
// other input, and `not-fetched` keeps the row unread.
func rowExamplePresenceColumns(contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, example semantic.OperatorRowCase) (map[string][]PresenceReading, error) {
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
				return nil, fmt.Errorf("operator %s row example states no presence for input %q on row %d", contract.ID, name, index+1)
			}
			switch text {
			case semantic.RowCellNotFetched:
				column = append(column, PresenceReading{State: ValueNotFetched})
			case semantic.OperatorCasePresencePresent:
				column = append(column, PresenceReading{State: ValuePresent})
			case semantic.OperatorCasePresenceAbsent:
				column = append(column, PresenceReading{State: ValueNull})
			case semantic.RowCellNoValue:
				return nil, fmt.Errorf("operator %s row example leaves input %q of row %d empty; a presence input writes %s or %s, because the empty cell already states that the row holds no value for every other input",
					contract.ID, name, index+1, semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent)
			default:
				return nil, fmt.Errorf("operator %s row example writes %q for the presence input %q on row %d; a presence input writes %s or %s",
					contract.ID, text, name, index+1, semantic.OperatorCasePresencePresent, semantic.OperatorCasePresenceAbsent)
			}
		}
		rows[name] = column
	}
	return rows, nil
}

// rowExampleNumberColumns reads one example's rows as the numeric columns an
// implementation that decides numbers is asked about.
func rowExampleNumberColumns(contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, example semantic.OperatorRowCase) (map[string][]NumberReading, error) {
	// Section 8.1: an example may state the empty relation. Every column is
	// then empty, and the implementation is asked about no rows rather than
	// skipped: whether the empty relation satisfies the operator is the
	// operator's own answer, and it must not depend on the declaration's
	// `refutes on some rows`.
	rows := make(map[string][]NumberReading, len(inputs))
	for _, input := range inputs {
		name := input.Name
		if name == "" {
			name = "value"
		}
		column := make([]NumberReading, 0, len(example.Rows))
		for index, row := range example.Rows {
			text, stated := row[name]
			if !stated {
				return nil, fmt.Errorf("operator %s row example states no value for input %q on row %d", contract.ID, name, index+1)
			}
			// The three states a read row can hold are written in the table: a
			// value of the input's type, the empty text (no value), and the word
			// `not-fetched` (not read). They are handed to the implementation as
			// they are, so the example asks it exactly the question a rule read
			// would.
			switch text {
			case semantic.RowCellNoValue:
				column = append(column, NumberReading{State: ValueNull})
			case semantic.RowCellNotFetched:
				column = append(column, NumberReading{State: ValueNotFetched})
			default:
				// A value is read in the type the declaration gave, so a date is
				// a date and a text is a text (section 8.3). The numeric column is
				// the one this path reads, so an input of another type is refused
				// rather than read as a number.
				if input.Type != semantic.OperatorInputNumber {
					return nil, fmt.Errorf("operator %s row example states a value for input %q, which is read as %s, but this path reads rows as numbers; the %s input has no reader on this path yet",
						contract.ID, name, input.Type, input.Type)
				}
				value, err := semantic.ParseOperatorCaseValue(text)
				if err != nil {
					return nil, fmt.Errorf("operator %s has an unreadable row example (%s, row %d): %w", contract.ID, name, index+1, err)
				}
				column = append(column, NumberReading{State: ValuePresent, Value: value})
			}
		}
		rows[name] = column
	}
	return rows, nil
}

// rowExampleValueColumns reads one example's rows as the columns of typed values
// an implementation that reads values of any type is asked about. Each cell is
// read in the type its input declares, which is the same reading the rule path
// makes, so the example asks the implementation exactly the question a rule
// read would (section 8.3, 8.4).
func rowExampleValueColumns(contract semantic.Operator, inputs []semantic.OperatorInputDeclaration, example semantic.OperatorRowCase) (map[string][]RowValueReading, error) {
	rows := make(map[string][]RowValueReading, len(inputs))
	for _, input := range inputs {
		name := input.Name
		if name == "" {
			name = "value"
		}
		column := make([]RowValueReading, 0, len(example.Rows))
		for index, row := range example.Rows {
			text, stated := row[name]
			if !stated {
				return nil, fmt.Errorf("operator %s row example states no value for input %q on row %d", contract.ID, name, index+1)
			}
			// A row that holds no value and a row nobody read keep their own
			// state, and the value is carried only for a row that holds one
			// (section 8.4).
			switch text {
			case semantic.RowCellNoValue:
				column = append(column, RowValueReading{State: ValueNull})
			case semantic.RowCellNotFetched:
				column = append(column, RowValueReading{State: ValueNotFetched})
			default:
				value, err := semantic.ParseOperatorInputValue(input.Type, text)
				if err != nil {
					return nil, fmt.Errorf("operator %s has an unreadable row example (%s, row %d): %w", contract.ID, name, index+1, err)
				}
				column = append(column, RowValueReading{State: ValuePresent, Value: value})
			}
		}
		rows[name] = column
	}
	return rows, nil
}

// operatorRowExpectTri maps the answer an example states onto the answer an
// implementation gives.
func operatorRowExpectTri(answer semantic.OperatorRowExpect) (Tri, bool) {
	switch answer {
	case semantic.OperatorRowSatisfied:
		return TriYes, true
	case semantic.OperatorRowNotSatisfied:
		return TriNo, true
	case semantic.OperatorRowUndecided:
		return TriMaybe, true
	}
	return 0, false
}

func operatorRowTriWord(answer Tri) string {
	switch answer {
	case TriYes:
		return string(semantic.OperatorRowSatisfied)
	case TriNo:
		return string(semantic.OperatorRowNotSatisfied)
	}
	return "undecided"
}

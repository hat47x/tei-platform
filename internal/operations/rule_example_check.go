package operations

import (
	"fmt"
	"math/big"

	"strings"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/syntax"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// Diagnostic codes of the example check.
const (
	// RuleExampleMismatchCode: a rule answers differently from what the
	// example expects. Either the rule or the example is wrong; the check
	// cannot tell which, and reports the place the expectation is written.
	RuleExampleMismatchCode = "RULE_EXAMPLE_MISMATCH"
	// RuleExampleNotCheckedCode: the rule could not be asked at all, for
	// instance because no implementation is selected for its operator. An
	// example that was not checked is not reported as agreeing.
	RuleExampleNotCheckedCode = "RULE_EXAMPLE_NOT_CHECKED"
	// OperatorSelectionUnresolvedCode: the manifest selects an implementation
	// this running tei cannot honour, or one that does not match the operator
	// it is selected for. The selection is refused whether or not a rule
	// happens to be read in this session: a Workspace whose manifest cannot be
	// honoured is not a Workspace this binary can run.
	OperatorSelectionUnresolvedCode = "OPERATOR_SELECTION_UNRESOLVED"
)

// exampleSource reads one example the way a rule reads records.
//
// For a rule about one record it answers from that row. For a rule about all
// the rows it answers with every row, and says the rows are all there are:
// the author wrote the whole example. An item the table has no column for was
// not written, so it is reported as not fetched, never as empty.
type exampleSource struct {
	example workspace.RuleExample
	row     int
}

// ReadAttempt states that every value of the example came from one read. The
// example is one authoring statement: its rows are written together, so a rule
// that reads two of its items reads them from the same statement rather than
// from two reads that could disagree about what the records held.
//
// It is the answer section 14.2 asks a reader for when a rule reads more than
// one attribute: a reader that states no attempt leaves that rule undecided.
func (s exampleSource) ReadAttempt(attribute core.CanonicalID) (string, bool) {
	return "the example table " + s.example.File, true
}

func (s exampleSource) column(attribute core.CanonicalID) (int, bool) {
	for i, item := range s.example.Items {
		if item.Attribute == attribute {
			return i, true
		}
	}
	return 0, false
}

func (s exampleSource) ReadNumber(attribute core.CanonicalID) (validationeval.NumberReading, error) {
	if s.row < 0 {
		return validationeval.NumberReading{}, fmt.Errorf("a rule about all the rows asked for the single value %s", attribute)
	}
	column, written := s.column(attribute)
	if !written {
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[s.row][column]
	if cell.Text == "" {
		return validationeval.NumberReading{State: validationeval.ValueNull}, nil
	}
	// A cell that writes the word for "not read" is read as not fetched, which
	// is the state the word states, rather than as a decimal it never was.
	if semantic.IsOperatorCaseNotFetchedWord(cell.Text) {
		return validationeval.NumberReading{State: validationeval.ValueNotFetched}, nil
	}
	value, err := semantic.ParseOperatorCaseValue(cell.Text)
	if err != nil {
		return validationeval.NumberReading{}, err
	}
	return validationeval.NumberReading{State: validationeval.ValuePresent, Value: value}, nil
}

func (s exampleSource) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	if s.row < 0 {
		return validationeval.DateReading{}, fmt.Errorf("a rule about all the rows asked for the single value %s", attribute)
	}
	column, written := s.column(attribute)
	if !written {
		return validationeval.DateReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[s.row][column]
	if cell.Text == "" {
		return validationeval.DateReading{State: validationeval.ValueNull}, nil
	}
	// A cell that writes the word for "not read" is read as not fetched, which
	// is the state the word states, rather than as a date it never was.
	if semantic.IsOperatorCaseNotFetchedWord(cell.Text) {
		return validationeval.DateReading{State: validationeval.ValueNotFetched}, nil
	}
	date, err := semantic.ParseCalendarDate(cell.Text)
	if err != nil {
		return validationeval.DateReading{}, err
	}
	return validationeval.DateReading{State: validationeval.ValuePresent, Value: date}, nil
}

// readCellDate reads the date of one cell of one row. A column the example
// does not have was not written, so it is not fetched.
// ReadText reads one text of the example. A column the example does not have
// was not written, so it is not fetched; an empty cell is a record that holds no
// text; the word for "not read" is that state. The cell states the text with
// the quotes section 8.3 fixes, so an empty quoted text is a value of length
// zero and stays apart from a record that holds no value.
func (s exampleSource) ReadText(attribute core.CanonicalID) (validationeval.TextReading, error) {
	if s.row < 0 {
		return validationeval.TextReading{}, fmt.Errorf("a rule about all the rows asked for the single value %s", attribute)
	}
	column, written := s.column(attribute)
	if !written {
		return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[s.row][column]
	if cell.Text == "" {
		return validationeval.TextReading{State: validationeval.ValueNull}, nil
	}
	if semantic.IsOperatorCaseNotFetchedWord(cell.Text) {
		return validationeval.TextReading{State: validationeval.ValueNotFetched}, nil
	}
	value, err := semantic.ParseOperatorTextValue(cell.Text)
	if err != nil {
		return validationeval.TextReading{}, err
	}
	return validationeval.TextReading{State: validationeval.ValuePresent, Value: value}, nil
}

// ReadTruth reads one truth of the example. A column the example does not have
// was not written, so it is not fetched; an empty cell is a record that holds no
// truth; the word for "not read" is that state. The cell states the truth as the
// words true and false, never as a number.
func (s exampleSource) ReadTruth(attribute core.CanonicalID) (validationeval.TruthReading, error) {
	if s.row < 0 {
		return validationeval.TruthReading{}, fmt.Errorf("a rule about all the rows asked for the single value %s", attribute)
	}
	column, written := s.column(attribute)
	if !written {
		return validationeval.TruthReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[s.row][column]
	if cell.Text == "" {
		return validationeval.TruthReading{State: validationeval.ValueNull}, nil
	}
	if semantic.IsOperatorCaseNotFetchedWord(cell.Text) {
		return validationeval.TruthReading{State: validationeval.ValueNotFetched}, nil
	}
	value, err := semantic.ParseOperatorTruthValue(cell.Text)
	if err != nil {
		return validationeval.TruthReading{}, err
	}
	return validationeval.TruthReading{State: validationeval.ValuePresent, Value: value}, nil
}

// ReadAny reads one value of an `any` input of the example. The cell's writing
// decides the type, so an unquoted 100 and a quoted "100" are different values
// and the value carries which one it was.
func (s exampleSource) ReadAny(attribute core.CanonicalID) (validationeval.AnyReading, error) {
	if s.row < 0 {
		return validationeval.AnyReading{}, fmt.Errorf("a rule about all the rows asked for the single value %s", attribute)
	}
	column, written := s.column(attribute)
	if !written {
		return validationeval.AnyReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[s.row][column]
	if cell.Text == "" {
		return validationeval.AnyReading{State: validationeval.ValueNull}, nil
	}
	if semantic.IsOperatorCaseNotFetchedWord(cell.Text) {
		return validationeval.AnyReading{State: validationeval.ValueNotFetched}, nil
	}
	value, err := semantic.ParseOperatorInputValue(semantic.OperatorInputAny, cell.Text)
	if err != nil {
		return validationeval.AnyReading{}, err
	}
	if value.Any == nil {
		return validationeval.AnyReading{}, fmt.Errorf("%q is not a value of any single type", cell.Text)
	}
	return validationeval.AnyReading{State: validationeval.ValuePresent, Value: *value.Any}, nil
}

func (s exampleSource) readCellDate(row int, attribute core.CanonicalID) (validationeval.DateReading, error) {
	column, written := s.column(attribute)
	if !written {
		return validationeval.DateReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[row][column]
	if cell.Text == "" {
		return validationeval.DateReading{State: validationeval.ValueNull}, nil
	}
	date, err := semantic.ParseCalendarDate(cell.Text)
	if err != nil {
		return validationeval.DateReading{}, err
	}
	return validationeval.DateReading{State: validationeval.ValuePresent, Value: date}, nil
}

// ReadPresence reports whether the item holds a value in this row. An item the
// table has no column for was not written, so it is not fetched; an empty cell
// is a record that holds no value.
func (s exampleSource) ReadPresence(attribute core.CanonicalID) (validationeval.PresenceReading, error) {
	if s.row < 0 {
		return validationeval.PresenceReading{}, fmt.Errorf("a rule about all the rows asked for the presence of %s", attribute)
	}
	column, written := s.column(attribute)
	if !written {
		return validationeval.PresenceReading{State: validationeval.ValueNotFetched}, nil
	}
	cell := s.example.Rows[s.row][column]
	// Section 8.3 fixes the two words a presence cell states: `present` means
	// the record holds a value, and `absent` means it holds none. The word has
	// to be read before the fallback below, or `absent` would be answered as a
	// value that is there, which states the opposite of what the author wrote.
	switch cell.Text {
	case semantic.OperatorCasePresencePresent:
		return validationeval.PresenceReading{State: validationeval.ValuePresent}, nil
	case semantic.OperatorCasePresenceAbsent:
		return validationeval.PresenceReading{State: validationeval.ValueNull}, nil
	}
	if cell.Text == "" {
		return validationeval.PresenceReading{State: validationeval.ValueNull}, nil
	}
	// Section 8.3: the writing for "not read" does not depend on the input's
	// type, so a cell that states the word is a row nobody read rather than a
	// row that holds a value. Reading it as present would answer the presence
	// question from a value nobody looked at, which is the same mistake the
	// number path already avoids (it checks this word before parsing a decimal).
	if semantic.IsOperatorCaseNotFetchedWord(cell.Text) {
		return validationeval.PresenceReading{State: validationeval.ValueNotFetched}, nil
	}
	// Any other written cell states that the record holds a value: the table
	// already writes a cell per item, and a presence column's other cells are
	// the author's way of saying the item is there.
	return validationeval.PresenceReading{State: validationeval.ValuePresent}, nil
}

func (s exampleSource) ReadRelationPeriods(relation, start, end, group core.CanonicalID) (validationeval.PeriodRowsReading, error) {
	if s.row >= 0 {
		return validationeval.PeriodRowsReading{}, fmt.Errorf("a rule about one record asked for the rows of relation %s", relation)
	}
	reading := validationeval.PeriodRowsReading{Coverage: validationeval.CoverageComplete}
	groupColumn, groupWritten := s.column(group)
	for i, row := range s.example.Rows {
		var period validationeval.PeriodRow
		var err error
		if period.Start, err = s.readCellDate(i, start); err != nil {
			return validationeval.PeriodRowsReading{}, err
		}
		if period.End, err = s.readCellDate(i, end); err != nil {
			return validationeval.PeriodRowsReading{}, err
		}
		if group != "" {
			switch {
			case !groupWritten:
				period.Group = validationeval.KeyReading{State: validationeval.ValueNotFetched}
			case row[groupColumn].Text == "":
				period.Group = validationeval.KeyReading{State: validationeval.ValueNull}
			default:
				key, err := exampleKey(s.example.Items[groupColumn], row[groupColumn].Text)
				if err != nil {
					return validationeval.PeriodRowsReading{}, err
				}
				period.Group = validationeval.KeyReading{State: validationeval.ValuePresent, Key: key}
			}
		}
		reading.Rows = append(reading.Rows, period)
	}
	return reading, nil
}

// ReadTypedRows reads one input's column over every row of the example in the
// type the declaration states. An example is the whole record set it states, so
// the coverage is complete. A column the example does not have was not written,
// so every row is not fetched; an empty cell is a row that holds no value; the
// word for "not read" is that state. A cell that is not the input's type is
// refused with the text it holds, so the author can find the cell.
//
// A rule about one record cannot ask this: its operator reads one record, and
// the example says which row that is.
func (s exampleSource) ReadTypedRows(input semantic.OperatorInputDeclaration, relation, attribute core.CanonicalID) (validationeval.TypedRowReading, error) {
	if s.row >= 0 {
		return validationeval.TypedRowReading{}, fmt.Errorf("a rule about one record asked for the rows of relation %s", relation)
	}
	column, written := s.column(attribute)
	reading := validationeval.TypedRowReading{
		Coverage: validationeval.CoverageComplete,
		State:    validationeval.ValuePresent,
		Values:   make([]validationeval.RowValueReading, 0, len(s.example.Rows)),
	}
	for _, row := range s.example.Rows {
		switch {
		case !written:
			reading.Values = append(reading.Values, validationeval.RowValueReading{State: validationeval.ValueNotFetched})
		case row[column].Text == "":
			reading.Values = append(reading.Values, validationeval.RowValueReading{State: validationeval.ValueNull})
		case semantic.IsOperatorCaseNotFetchedWord(row[column].Text):
			reading.Values = append(reading.Values, validationeval.RowValueReading{State: validationeval.ValueNotFetched})
		default:
			value, err := semantic.ParseOperatorInputValue(input.Type, row[column].Text)
			if err != nil {
				return validationeval.TypedRowReading{}, err
			}
			reading.Values = append(reading.Values, validationeval.RowValueReading{State: validationeval.ValuePresent, Value: value})
		}
	}
	return reading, nil
}

// ReadRelationNumbers reads one numeric attribute over every row of the
// example. The author wrote the rows the example has, so the coverage is
// complete: an example is the whole record set it states, not a page of a
// larger one. A column the example does not have was not written, so every row
// is not fetched; an empty cell is a row that holds no value.
//
// A rule about one record cannot ask this: its operator reads one record, and
// the example says which row that is.
// ReadRelationPresence reads whether one item holds a value in every row of the
// example table. A row-set example states records, so its cell states the item's
// own value: a written cell is a value the record holds, an empty cell is a
// record that holds none, and the word for "not read" is that state (section
// 8.3). The rows of the example are the rows of the relation the mark names.
func (s exampleSource) ReadRelationPresence(relation, attribute core.CanonicalID) (validationeval.PresenceRowsReading, error) {
	if s.row >= 0 {
		return validationeval.PresenceRowsReading{}, fmt.Errorf("a rule about one record asked for the rows of relation %s", relation)
	}
	column, written := s.column(attribute)
	reading := validationeval.PresenceRowsReading{Coverage: validationeval.CoverageComplete}
	for _, row := range s.example.Rows {
		switch {
		case !written:
			reading.Rows = append(reading.Rows, validationeval.PresenceReading{State: validationeval.ValueNotFetched})
		case row[column].Text == "":
			reading.Rows = append(reading.Rows, validationeval.PresenceReading{State: validationeval.ValueNull})
		case semantic.IsOperatorCaseNotFetchedWord(row[column].Text):
			reading.Rows = append(reading.Rows, validationeval.PresenceReading{State: validationeval.ValueNotFetched})
		default:
			reading.Rows = append(reading.Rows, validationeval.PresenceReading{State: validationeval.ValuePresent})
		}
	}
	return reading, nil
}

func (s exampleSource) ReadRelationNumbers(relation, attribute core.CanonicalID) (validationeval.NumberRowsReading, error) {
	if s.row >= 0 {
		return validationeval.NumberRowsReading{}, fmt.Errorf("a rule about one record asked for the rows of relation %s", relation)
	}
	column, written := s.column(attribute)
	reading := validationeval.NumberRowsReading{Coverage: validationeval.CoverageComplete}
	for _, row := range s.example.Rows {
		switch {
		case !written:
			reading.Rows = append(reading.Rows, validationeval.NumberReading{State: validationeval.ValueNotFetched})
		case row[column].Text == "":
			reading.Rows = append(reading.Rows, validationeval.NumberReading{State: validationeval.ValueNull})
		case semantic.IsOperatorCaseNotFetchedWord(row[column].Text):
			reading.Rows = append(reading.Rows, validationeval.NumberReading{State: validationeval.ValueNotFetched})
		default:
			value, err := semantic.ParseOperatorCaseValue(row[column].Text)
			if err != nil {
				return validationeval.NumberRowsReading{}, err
			}
			reading.Rows = append(reading.Rows, validationeval.NumberReading{State: validationeval.ValuePresent, Value: value})
		}
	}
	return reading, nil
}

func (s exampleSource) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	if s.row >= 0 {
		return validationeval.RelationReading{}, fmt.Errorf("a rule about one record asked for the rows of relation %s", relation)
	}
	reading := validationeval.RelationReading{Coverage: validationeval.CoverageComplete}
	column, written := s.column(attribute)
	for _, row := range s.example.Rows {
		switch {
		case !written:
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNotFetched})
		case row[column].Text == "":
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValueNull})
		default:
			key, err := exampleKey(s.example.Items[column], row[column].Text)
			if err != nil {
				return validationeval.RelationReading{}, err
			}
			reading.Keys = append(reading.Keys, validationeval.KeyReading{State: validationeval.ValuePresent, Key: key})
		}
	}
	return reading, nil
}

// exampleKey is what two cells are compared by. Numbers are compared as
// numbers, so 1 and 1.0 are the same key; anything else is compared as
// written.
func exampleKey(item workspace.RuleExampleItem, text string) (string, error) {
	if !item.Numeric {
		return "text:" + text, nil
	}
	value, err := semantic.ParseOperatorCaseValue(text)
	if err != nil {
		return "", err
	}
	return "number:" + new(big.Rat).Set(value).RatString(), nil
}

// CheckRuleExamples asks every rule an example names about the example's
// records and reports each answer that differs from the expected one.
//
// It reads the examples and the World of one Load and nothing else. A rule
// decided by a declared operator is asked through the implementation the
// Workspace selected among available. The result is empty when every example
// was checked and agrees.
// CheckOperatorSelection reports a manifest implementation selection that this
// running tei cannot honour: a name it does not carry, two entries for one
// operator, or an implementation that does not match the operator it is
// selected for.
//
// It is separate from CheckRuleExamples because a Workspace with no examples
// still states a selection. Before this, a selection that could not be resolved
// was reported only while checking an example, so a Workspace with no example
// tables passed validate and then failed at the first read:
//
//	$ tei validate                    -> ok
//	$ tei view -environment ...       -> TEI_OPERATOR_IMPLEMENTATION_UNAVAILABLE
//
// One manifest statement must get one diagnosis at validate, whether or not a
// rule is read in this session. The reported place is the operator's
// declaration, which is where the selection attaches.
func CheckOperatorSelection(loaded workspace.Result, index sourceindex.Index, available []validationeval.NamedImplementation) []workspace.ProjectedDiagnostic {
	if len(loaded.OperatorImplementations) == 0 {
		return nil
	}
	selections := make([]validationeval.ImplementationSelection, 0, len(loaded.OperatorImplementations))
	for _, selection := range loaded.OperatorImplementations {
		selections = append(selections, validationeval.ImplementationSelection{
			Operator: core.CanonicalID(selection.Operator), Implementation: selection.Implementation,
		})
	}
	implementations, err := validationeval.ResolveImplementations(selections, available, nil)
	if err != nil {
		// The error names the operator and the implementation it could not
		// resolve. The subject is derived from the entry that failed, not from
		// the message: a first selection may resolve while a later one does not.
		for _, selection := range loaded.OperatorImplementations {
			if !strings.Contains(err.Error(), selection.Implementation) {
				continue
			}
			return []workspace.ProjectedDiagnostic{operatorSelectionFinding(index, core.CanonicalID(selection.Operator), err.Error())}
		}
		// No entry could be attributed to the message. Report it against the
		// first one rather than dropping it: a selection this binary cannot
		// honour must never be silently accepted.
		first := core.CanonicalID(loaded.OperatorImplementations[0].Operator)
		return []workspace.ProjectedDiagnostic{operatorSelectionFinding(index, first, err.Error())}
	}
	// Resolving is not the whole check: an implementation that resolves by
	// name can still disagree with the operator it is selected for, and the
	// evaluator refuses that when it is asked. Ask the same function the
	// evaluator asks, for every selected operator, so validate and a read
	// cannot disagree about what "matches" means.
	for _, implementation := range implementations {
		operator, declared := operatorByID(loaded.World, implementation.Operator)
		if !declared {
			continue
		}
		if err := validationeval.CheckImplementationMatchesContract(operator, implementation); err != nil {
			return []workspace.ProjectedDiagnostic{operatorSelectionFinding(index, implementation.Operator, err.Error())}
		}
	}
	return nil
}

// operatorByID finds the declared operator a selection attaches to.
func operatorByID(world ir.World, id core.CanonicalID) (semantic.Operator, bool) {
	for _, namespace := range world.Semantics {
		for _, operator := range namespace.Operators {
			if operator.ID == id {
				return operator, true
			}
		}
	}
	return semantic.Operator{}, false
}

// operatorSelectionFinding projects one selection failure onto the operator's
// declaration. A finding with no resolvable source is still reported; the
// message names what failed either way.
func operatorSelectionFinding(index sourceindex.Index, operator core.CanonicalID, message string) workspace.ProjectedDiagnostic {
	opSubject := subject.Subject{Kind: "semantic-operator", ID: string(operator)}
	finding := workspace.ProjectedDiagnostic{
		Diagnostic: validation.Diagnostic{
			Code: OperatorSelectionUnresolvedCode, Severity: validation.Error,
			Message: message, Subject: &opSubject,
		},
	}
	for _, ref := range index.ReferencesFor(opSubject) {
		if ref.Role == sourceindex.RoleDefinition {
			finding.Sources = append(finding.Sources, ref)
		}
	}
	return finding
}

func CheckRuleExamples(loaded workspace.Result, available []validationeval.NamedImplementation) []workspace.ProjectedDiagnostic {
	if len(loaded.RuleExamples) == 0 {
		return nil
	}
	rules := map[core.CanonicalID]semantic.Validation{}
	operators := validationeval.Operators{}
	for _, namespace := range loaded.World.Semantics {
		for _, rule := range namespace.Validations {
			rules[rule.ID] = rule
		}
		operators.Declared = append(operators.Declared, namespace.Operators...)
	}
	selections := make([]validationeval.ImplementationSelection, 0, len(loaded.OperatorImplementations))
	for _, selection := range loaded.OperatorImplementations {
		selections = append(selections, validationeval.ImplementationSelection{
			Operator: core.CanonicalID(selection.Operator), Implementation: selection.Implementation,
		})
	}
	implementations, selectionErr := validationeval.ResolveImplementations(selections, available, nil)
	if selectionErr == nil {
		operators.Implementations = implementations
	}

	var out []workspace.ProjectedDiagnostic
	report := func(example workspace.RuleExample, expectation workspace.RuleExampleExpectation, code, message string) {
		ruleSubject := subject.Subject{Kind: "semantic-validation", ID: string(expectation.Rule)}
		out = append(out, workspace.ProjectedDiagnostic{
			Diagnostic: validation.Diagnostic{Code: code, Severity: validation.Error, Message: message, Subject: &ruleSubject},
			Sources: []sourceindex.Reference{{
				Subject:      ruleSubject,
				ArtifactPath: example.File,
				Range:        expectation.Range,
				Role:         sourceindex.RoleAuthoring,
			}},
		})
	}
	for _, example := range loaded.RuleExamples {
		for _, expectation := range example.Expectations {
			rule, known := rules[expectation.Rule]
			if !known {
				report(example, expectation, RuleExampleNotCheckedCode,
					fmt.Sprintf("the example expects an answer of rule %s, which the loaded model does not have", expectation.Rule))
				continue
			}
			if rule.Kind == semantic.ValidationOperator && selectionErr != nil {
				report(example, expectation, RuleExampleNotCheckedCode,
					fmt.Sprintf("rule %s was not asked about the example: %v", rule.ID, selectionErr))
				continue
			}
			decision, err := validationeval.Decide(rule, exampleSource{example: example, row: expectation.Row}, operators)
			if err != nil {
				report(example, expectation, RuleExampleNotCheckedCode,
					fmt.Sprintf("rule %s was not asked about the example: %v", rule.ID, err))
				continue
			}
			answer, detail := syntax.RuleExampleExpect(decision.Result), ""
			if decision.Undecided != nil {
				answer, detail = syntax.RuleExampleUndecided, fmt.Sprintf(" (%s)", decision.Undecided)
			}
			if answer != expectation.Expect {
				report(example, expectation, RuleExampleMismatchCode,
					fmt.Sprintf("the example expects %s, but rule %s answers %s%s", expectation.Expect, rule.ID, answer, detail))
			}
		}
	}
	return out
}

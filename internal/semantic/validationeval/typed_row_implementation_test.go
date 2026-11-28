package validationeval_test

import (
	"fmt"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule whose operator reads a column of days is decided by the shipped
// implementation that decides rows of any type (section 8.4). The expectations
// below are written from what the operator means — every row's day is not before
// its deadline's day — and not read back from the code.

// dayRowsSource reads one column of a relation's rows as calendar days, so a
// row-scoped rule about days is answered from the type the declaration states
// and not from a decimal the column was never read as.
type dayRowsSource struct {
	coverage validationeval.Coverage
	columns  map[core.CanonicalID][]validationeval.RowValueReading
	// asked records which attributes were read, so a test can show that each
	// input is read once and not once per row.
	asked []core.CanonicalID
}

func (s *dayRowsSource) ReadTypedRows(input semantic.OperatorInputDeclaration, relation, attribute core.CanonicalID) (validationeval.TypedRowReading, error) {
	s.asked = append(s.asked, attribute)
	column, ok := s.columns[attribute]
	if !ok {
		return validationeval.TypedRowReading{State: validationeval.ValueNotFetched}, nil
	}
	return validationeval.TypedRowReading{
		Coverage: s.coverage,
		State:    validationeval.ValuePresent,
		Values:   column,
	}, nil
}

func (s *dayRowsSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, fmt.Errorf("this reader reads days, not numbers")
}

func (s *dayRowsSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("this reader reads days, not keys")
}

var (
	deliveredOn core.CanonicalID = "sales.order_line.delivered_on"
	deadlineOn  core.CanonicalID = "sales.order_line.deadline_on"
)

// everyNotBeforeRule is a rule about every row of the relation: the subject
// supplies the first input, and the rule names the relation its rows are read
// from.
var everyNotBeforeRule = semantic.Validation{
	ID:        "sales.order.lines.delivered_not_before_deadline",
	Kind:      semantic.ValidationOperator,
	Attribute: deliveredOn,
	Relation:  "sales.order.lines",
	Operator:  "sales.every_not_before",
}

// everyNotBeforeContract declares the shipped operator: every row's day is not
// before its deadline's day. The deadline is one value for the whole relation,
// so a rule may bind it to a constant or to a column of the rows.
func everyNotBeforeContract(refutes bool) semantic.Operator {
	return semantic.Operator{
		ID: "sales.every_not_before", Symbol: "every_not_before",
		Input: semantic.OperatorInputDate,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputDate, OfEachRow: true},
			{Name: "deadline", Type: semantic.OperatorInputDate},
		},
		RefutesOnSomeRows: refutes,
	}
}

// shippedEveryNotBefore resolves the shipped implementation for the contract,
// so the implementation under test is the one this tei carries.
func shippedEveryNotBefore(t *testing.T, contract semantic.Operator) []validationeval.OperatorImplementation {
	t.Helper()
	implementations, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: contract.ID, Implementation: "tei.day.every-not-before"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("the shipped implementation could not be resolved: %v", err)
	}
	return implementations
}

// The shipped implementation decides the rule from the days the reader read, and
// the coverage and the reasons of the rows are composed with its answer in the
// one place that decides them (row_scope.go).
func TestASelectedDayRowImplementationDecidesTheRule(t *testing.T) {
	for _, tc := range []struct {
		name string
		// refutes is the declaration's `refutes on some rows`.
		refutes  bool
		coverage validationeval.Coverage
		value    []validationeval.RowValueReading
		deadline []validationeval.RowValueReading
		want     validationeval.Decision
	}{
		{
			name:     "every day is not before its deadline",
			coverage: validationeval.CoverageComplete,
			value:    dayColumn("2026-04-01", "2026-04-05"),
			deadline: dayColumn("2026-04-01", "2026-04-05"),
			want:     validationeval.Decision{Result: validationeval.ResultSatisfied},
		},
		{
			// A counterexample that was seen is settled by itself: the row
			// nobody read cannot hide it, and the declaration says a
			// counterexample among the rows read is enough.
			name:     "a witnessed counterexample beside a row nobody read",
			refutes:  true,
			coverage: validationeval.CoveragePartial,
			value:    append(dayColumn("2026-03-31"), unreadDay()),
			deadline: append(dayColumn("2026-04-05"), unreadDay()),
			want:     validationeval.Decision{Result: validationeval.ResultNotSatisfied},
		},
		{
			// A row that holds no day cannot rescue a comparison that failed,
			// and cannot break one that held either: the answer is the
			// operator's own.
			name:     "a counterexample beside a row that holds no value",
			refutes:  true,
			coverage: validationeval.CoverageComplete,
			value:    append(dayColumn("2026-03-31"), noDay()),
			deadline: append(dayColumn("2026-04-05"), noDay()),
			want:     validationeval.Decision{Result: validationeval.ResultNotSatisfied},
		},
		{
			// Holding everywhere cannot be claimed while a row that was read
			// holds no value: that row might break the rule. This is so even
			// when the reader states the rows are every row.
			name:     "a row that holds no value",
			coverage: validationeval.CoverageComplete,
			value:    append(dayColumn("2026-04-10"), noDay()),
			deadline: dayColumn("2026-04-05", "2026-04-05"),
			want: validationeval.Decision{Undecided: &validationeval.Undecided{
				Reason: validationeval.UndecidedValueNull, Attribute: deliveredOn,
			}},
		},
		{
			name:     "a row nobody read",
			coverage: validationeval.CoverageComplete,
			value:    append(dayColumn("2026-04-10"), unreadDay()),
			deadline: dayColumn("2026-04-05", "2026-04-05"),
			want: validationeval.Decision{Undecided: &validationeval.Undecided{
				Reason: validationeval.UndecidedValueNotFetched, Attribute: deliveredOn,
			}},
		},
		{
			// A relation that was only partly read cannot be answered
			// satisfied, and without the refutation statement it cannot be
			// answered not-satisfied either.
			name:     "a partly read relation with no counterexample",
			coverage: validationeval.CoveragePartial,
			value:    dayColumn("2026-04-10"),
			deadline: dayColumn("2026-04-05"),
			want: validationeval.Decision{Undecided: &validationeval.Undecided{
				Reason:    validationeval.UndecidedRelationPartiallyRead,
				Attribute: deliveredOn, Relation: "sales.order.lines",
			}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract := everyNotBeforeContract(tc.refutes)
			rule := everyNotBeforeRule
			rule.Inputs = []semantic.ValidationInputAssignment{{Name: "deadline", Value: string(deadlineOn)}}
			source := &dayRowsSource{
				coverage: tc.coverage,
				columns: map[core.CanonicalID][]validationeval.RowValueReading{
					deliveredOn: tc.value,
					deadlineOn:  tc.deadline,
				},
			}
			got, err := validationeval.Decide(rule, source, validationeval.Operators{
				Declared:        []semantic.Operator{contract},
				Implementations: shippedEveryNotBefore(t, contract),
			})
			if err != nil {
				t.Fatalf("the rule was refused: %v", err)
			}
			if got.Result != tc.want.Result {
				t.Fatalf("result=%q, want %q", got.Result, tc.want.Result)
			}
			if tc.want.Undecided == nil {
				if got.Undecided != nil {
					t.Fatalf("undecided=%v, want a decided answer", got.Undecided)
				}
				return
			}
			if got.Undecided == nil {
				t.Fatalf("undecided=nil, want %v", tc.want.Undecided)
			}
			// The reason and the place it stopped at are both part of the
			// answer: a row's own reason names the row, and the relation's own
			// reason names the relation.
			if got.Undecided.Reason != tc.want.Undecided.Reason || got.Undecided.Attribute != tc.want.Undecided.Attribute ||
				got.Undecided.Relation != tc.want.Undecided.Relation {
				t.Fatalf("undecided=%v, want %v", got.Undecided, tc.want.Undecided)
			}
		})
	}
}

// A constant states one value for the whole relation, so every row holds it and
// the operator is asked about one deadline per row rather than a column of one
// row. Before this the constant's column was one row long however many rows the
// relation was read into.
func TestADayRuleWithAConstantDeadlineIsDecidedForEveryRow(t *testing.T) {
	contract := everyNotBeforeContract(false)
	rule := everyNotBeforeRule
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "deadline", Value: "2026-04-05"}}
	source := &dayRowsSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.RowValueReading{
			deliveredOn: dayColumn("2026-03-31", "2026-04-05"),
		},
	}
	got, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: shippedEveryNotBefore(t, contract),
	})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	// The first row is before the constant deadline, so the relation does not
	// hold even though the second row is the deadline's own day.
	if got.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("result=%q, want not-satisfied", got.Result)
	}

	// Every row on or after the same constant holds.
	onTime := &dayRowsSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.RowValueReading{
			deliveredOn: dayColumn("2026-04-05", "2026-04-06"),
		},
	}
	got, err = validationeval.Decide(rule, onTime, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: shippedEveryNotBefore(t, contract),
	})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	if got.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied", got.Result)
	}
}

// Section 8.1: an empty relation is read as no rows, and the implementation is
// asked about them rather than skipped: whether the empty relation satisfies
// the operator is the operator's own answer.
func TestAnEmptyRelationIsAskedOfTheImplementationAsNoRows(t *testing.T) {
	contract := everyNotBeforeContract(false)
	rule := everyNotBeforeRule
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "deadline", Value: "2026-04-05"}}
	source := &dayRowsSource{
		coverage: validationeval.CoverageComplete,
		columns:  map[core.CanonicalID][]validationeval.RowValueReading{deliveredOn: {}},
	}
	var asked map[string][]validationeval.RowValueReading
	shipped := shippedEveryNotBefore(t, contract)[0]
	recording := validationeval.OperatorImplementation{
		Operator: contract.ID,
		Input:    shipped.Input,
		ValueRows: func(rows map[string][]validationeval.RowValueReading) (validationeval.Tri, error) {
			asked = rows
			return shipped.ValueRows(rows)
		},
	}
	got, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{recording},
	})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	if got.Result != validationeval.ResultSatisfied {
		t.Fatalf("result=%q, want satisfied", got.Result)
	}
	if asked == nil {
		t.Fatal("the implementation was not asked about the empty relation")
	}
	if len(asked["value"]) != 0 || len(asked["deadline"]) != 0 {
		t.Fatalf("the implementation was asked about %d value rows and %d deadline rows, want no rows",
			len(asked["value"]), len(asked["deadline"]))
	}
}

// A column of days is not decided by an implementation that decides numbers:
// the two read different values from the same column, and the contract check
// refuses the selection before any rule is read.
func TestADayRowRuleIsRefusedByAnImplementationThatDecidesNumbers(t *testing.T) {
	contract := everyNotBeforeContract(true)
	implementations, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: contract.ID, Implementation: "tei.number.every-at-most"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		// The selection resolves: the name exists. What refuses it is the
		// contract check, which happens before any rule is read.
		t.Fatalf("the number implementation could not be resolved: %v", err)
	}
	err = validationeval.CheckImplementationMatchesContract(contract, implementations[0])
	if err == nil {
		t.Fatal("a number implementation was accepted for a contract that reads days")
	}
	if !strings.Contains(err.Error(), "date") || !strings.Contains(err.Error(), "number") {
		t.Fatalf("the refusal does not name the two types: %v", err)
	}
	// The rule is refused for the same reason, so no row is read for it.
	rule := everyNotBeforeRule
	rule.Inputs = []semantic.ValidationInputAssignment{{Name: "deadline", Value: string(deadlineOn)}}
	source := &dayRowsSource{
		coverage: validationeval.CoverageComplete,
		columns: map[core.CanonicalID][]validationeval.RowValueReading{
			deliveredOn: dayColumn("2026-04-10"),
			deadlineOn:  dayColumn("2026-04-05"),
		},
	}
	if _, err := validationeval.Decide(rule, source, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: implementations,
	}); err == nil {
		t.Fatal("a day rule was decided by an implementation that decides numbers")
	}
}

// An implementation that decides both the numeric column and the column of any
// type leaves which values it answers about ambiguous, so it is refused.
func TestAnImplementationThatDecidesBothColumnsIsRefused(t *testing.T) {
	contract := everyNotBeforeContract(true)
	both := validationeval.NamedImplementation{
		ID:    "test.both_columns",
		Input: semantic.OperatorInputDate,
		NumberRows: func(map[string][]validationeval.NumberReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		},
		ValueRows: func(map[string][]validationeval.RowValueReading) (validationeval.Tri, error) {
			return validationeval.TriYes, nil
		},
	}
	implementations, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: contract.ID, Implementation: "test.both_columns"}},
		[]validationeval.NamedImplementation{both}, nil)
	if err != nil {
		t.Fatalf("the selection could not be resolved: %v", err)
	}
	if len(implementations) != 1 || implementations[0].NumberRows == nil || implementations[0].ValueRows == nil {
		t.Fatalf("resolved=%+v, want one implementation carrying both columns", implementations)
	}
	err = validationeval.CheckImplementationMatchesContract(contract, implementations[0])
	if err == nil || !strings.Contains(err.Error(), "more than one kind of decision function") {
		t.Fatalf("err=%v, want the ambiguity refusal", err)
	}
}

// The contract's own examples are asked of the implementation before any real
// row is read (section 8.1). A table of days states its answer in the mark, and
// the implementation is asked about exactly the rows a rule read would hand it,
// each value carrying the type it was read as (section 8.3, 8.4).
func TestTheRowExamplesOfADayOperatorAreCheckedAgainstTheImplementation(t *testing.T) {
	noDay := semantic.RowCellNoValue
	notFetched := semantic.RowCellNotFetched
	for _, tc := range []struct {
		name    string
		rows    []map[string]string
		answer  semantic.OperatorRowExpect
		wantErr string
	}{
		{
			name: "every day not before its deadline",
			rows: []map[string]string{
				{"value": "2026-04-01", "deadline": "2026-04-01"},
				{"value": "2026-04-05", "deadline": "2026-04-05"},
			},
			answer: semantic.OperatorRowSatisfied,
		},
		{
			name: "one day before its deadline",
			rows: []map[string]string{
				{"value": "2026-04-01", "deadline": "2026-04-05"},
				{"value": "2026-03-31", "deadline": "2026-04-01"},
			},
			answer: semantic.OperatorRowNotSatisfied,
		},
		{
			// The same rows with the answer changed: the example now states
			// something the implementation contradicts, and real data is not
			// read for it.
			name: "the answer alone changed",
			rows: []map[string]string{
				{"value": "2026-04-01", "deadline": "2026-04-05"},
				{"value": "2026-03-31", "deadline": "2026-04-01"},
			},
			answer:  semantic.OperatorRowSatisfied,
			wantErr: "TEI_OPERATOR_IMPLEMENTATION_MISMATCH",
		},
		{
			name: "a row that holds no day and nothing before its deadline",
			rows: []map[string]string{
				{"value": noDay, "deadline": "2026-04-05"},
				{"value": "2026-04-05", "deadline": "2026-04-05"},
			},
			answer: semantic.OperatorRowUndecided,
		},
		{
			name: "a counterexample beside a row nobody read",
			rows: []map[string]string{
				{"value": notFetched, "deadline": "2026-04-05"},
				{"value": "2026-03-31", "deadline": "2026-04-01"},
			},
			answer: semantic.OperatorRowNotSatisfied,
		},
		{
			// Section 8.1: the empty relation is one example, and its answer is
			// the operator's own: no row is before its deadline.
			name:   "no rows",
			rows:   nil,
			answer: semantic.OperatorRowSatisfied,
		},
		{
			// A day is written YYYY-MM-DD. A cell of another shape is refused
			// rather than read as the nearest day.
			name: "a cell that is not a day",
			rows: []map[string]string{
				{"value": "2026/04/01", "deadline": "2026-04-05"},
			},
			answer:  semantic.OperatorRowSatisfied,
			wantErr: "unreadable",
		},
		{
			// A text is written with quotes; a day is not one.
			name: "a day written as a text",
			rows: []map[string]string{
				{"value": `"2026-04-01"`, "deadline": "2026-04-05"},
			},
			answer:  semantic.OperatorRowSatisfied,
			wantErr: "unreadable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract := everyNotBeforeContract(true)
			contract.RowCases = []semantic.OperatorRowCase{{Rows: tc.rows, Answer: tc.answer}}
			err := validationeval.CheckImplementationMatchesContract(contract, shippedEveryNotBefore(t, contract)[0])
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("the example stopped the rule: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("err=%v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

// dayColumn builds one column of days, in the order written.
func dayColumn(days ...string) []validationeval.RowValueReading {
	out := make([]validationeval.RowValueReading, 0, len(days))
	for _, text := range days {
		day, err := semantic.ParseCalendarDate(text)
		if err != nil {
			panic(err)
		}
		out = append(out, validationeval.RowValueReading{
			State: validationeval.ValuePresent,
			Value: semantic.OperatorValue{Date: &day},
		})
	}
	return out
}

func noDay() validationeval.RowValueReading {
	return validationeval.RowValueReading{State: validationeval.ValueNull}
}

func unreadDay() validationeval.RowValueReading {
	return validationeval.RowValueReading{State: validationeval.ValueNotFetched}
}

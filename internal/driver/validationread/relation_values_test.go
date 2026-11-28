package validationread_test

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A column of a relation's rows is read in the type the declaration states, so
// an implementation is never handed a value of a type nobody read (section 8.4).
// The state of every row is carried beside its value, and the value is set only
// for a row that holds one.

// dateInput is one input declared as a calendar day.
var dateInput = semantic.OperatorInputDeclaration{Name: "value", Type: semantic.OperatorInputDate, OfEachRow: true}

// anyInput is one input declared as a value of any type.
var anyInput = semantic.OperatorInputDeclaration{Name: "value", Type: semantic.OperatorInputAny, OfEachRow: true}

// presenceInput reads whether an attribute holds a value, which is not a value
// of any type.
var presenceInput = semantic.OperatorInputDeclaration{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true}

func dayOf(t *testing.T, text string) semantic.OperatorValue {
	t.Helper()
	day, err := semantic.ParseCalendarDate(text)
	if err != nil {
		t.Fatal(err)
	}
	return semantic.OperatorValue{Date: &day}
}

func TestARelationColumnIsReadInTheTypeTheDeclarationStates(t *testing.T) {
	detail := detailOf(t, validationeval.CoveragePartial,
		row(map[core.CanonicalID]any{startsOn: "2026-04-01"}),
		row(map[core.CanonicalID]any{startsOn: nil}),
		row(map[core.CanonicalID]any{personID: "P003"}),
	)
	want := validationeval.TypedRowReading{
		Coverage: validationeval.CoveragePartial,
		State:    validationeval.ValuePresent,
		Values: []validationeval.RowValueReading{
			{State: validationeval.ValuePresent, Value: dayOf(t, "2026-04-01")},
			{State: validationeval.ValueNull},
			{State: validationeval.ValueNotFetched},
		},
	}
	got, err := detail.ReadRelationValues(membersID, startsOn, dateInput)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v err=%v, want %+v", got, err, want)
	}

	// The same rows read as a value of any type keep the type they arrived
	// with: a day is a day, and a text is a text.
	texts := detailOf(t, validationeval.CoverageComplete,
		row(map[core.CanonicalID]any{personID: "P001"}),
		row(map[core.CanonicalID]any{personID: []byte("P002")}),
	)
	any, err := texts.ReadRelationValues(membersID, personID, anyInput)
	if err != nil {
		t.Fatal(err)
	}
	if any.State != validationeval.ValuePresent || len(any.Values) != 2 {
		t.Fatalf("any=%+v", any)
	}
	for index, want := range []string{"P001", "P002"} {
		value := any.Values[index]
		if value.State != validationeval.ValuePresent || value.Value.Any == nil ||
			value.Value.Any.Type != semantic.OperatorInputText || *value.Value.Any.Text != want {
			t.Fatalf("row %d: %+v, want the text %q", index, value, want)
		}
	}

	// One row does not stand for the rows of a relation.
	one, err := detail.Row(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := one.ReadRelationValues(membersID, startsOn, dateInput); err == nil ||
		!strings.Contains(err.Error(), "one row is not the rows of relation") {
		t.Fatalf("err=%v, want the refusal of one row", err)
	}
}

func TestTheTypedColumnReaderRefusesWhatItWasNotGiven(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete,
		row(map[core.CanonicalID]any{startsOn: "2026-04-01", seatID: 3}))
	refusals := []struct {
		name    string
		err     func() error
		mention string
	}{
		{"another relation", func() error {
			_, err := detail.ReadRelationValues("organization.department.heads", startsOn, dateInput)
			return err
		}, "not of organization.department.heads"},
		{"an attribute of another data", func() error {
			_, err := detail.ReadRelationValues(membersID, "organization.department.name", dateInput)
			return err
		}, "is not an item of organization.membership"},
		{"a number column read as a day", func() error {
			_, err := detail.ReadRelationValues(membersID, seatID, dateInput)
			return err
		}, "is not declared as a calendar date"},
		{"a value that is not a day", func() error {
			_, err := detailOf(t, "", row(map[core.CanonicalID]any{startsOn: "2026/04/01"})).
				ReadRelationValues(membersID, startsOn, dateInput)
			return err
		}, "not a calendar date"},
		{"an input that reads no value", func() error {
			_, err := detail.ReadRelationValues(membersID, startsOn, presenceInput)
			return err
		}, "does not read a value of a row as"},
	}
	for _, tc := range refusals {
		if err := tc.err(); err == nil || !strings.Contains(err.Error(), tc.mention) {
			t.Fatalf("%s: err=%v, want an error mentioning %q", tc.name, err, tc.mention)
		}
	}
}

// The rows of one relation are asked of the whole relation, so a rule whose
// operator reads a column of days is decided once for the relation and not once
// per row. The implementation under test is the one this tei ships.
var everyNotBefore = semantic.Operator{
	ID: "organization.every_membership_not_before", Symbol: "every_membership_not_before",
	Input: semantic.OperatorInputDate,
	Inputs: []semantic.OperatorInputDeclaration{
		{Name: "value", Type: semantic.OperatorInputDate, OfEachRow: true},
		{Name: "deadline", Type: semantic.OperatorInputDate},
	},
}

var everyNotBeforeRule = semantic.Validation{
	ID:        "organization.department.members.starts_not_before_end",
	Kind:      semantic.ValidationOperator,
	Attribute: startsOn,
	Relation:  membersID,
	Operator:  everyNotBefore.ID,
	Inputs:    []semantic.ValidationInputAssignment{{Name: "deadline", Value: string(endsOn)}},
}

func decideEveryNotBefore(t *testing.T, detail *validationread.Detail) validationeval.Decision {
	t.Helper()
	implementations, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: everyNotBefore.ID, Implementation: "tei.day.every-not-before"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("the shipped implementation could not be resolved: %v", err)
	}
	decisions, err := detail.Decide(everyNotBeforeRule, validationeval.Operators{
		Declared:        []semantic.Operator{everyNotBefore},
		Implementations: implementations,
	})
	if err != nil {
		t.Fatalf("the rule was refused: %v", err)
	}
	if decisions.Relation == nil {
		t.Fatalf("the rule was asked row by row: %+v", decisions)
	}
	return *decisions.Relation
}

func TestARowScopedDayRuleIsDecidedFromTheRowsOfTheRelation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		coverage validationeval.Coverage
		rows     []driver.ContractQueryRow
		want     validationeval.Decision
	}{
		{
			name:     "every membership starts on or after it ends",
			coverage: validationeval.CoverageComplete,
			rows: []driver.ContractQueryRow{
				row(map[core.CanonicalID]any{startsOn: "2026-04-01", endsOn: "2026-04-01"}),
				row(map[core.CanonicalID]any{startsOn: "2026-05-01", endsOn: "2026-04-10"}),
			},
			want: validationeval.Decision{Result: validationeval.ResultSatisfied},
		},
		{
			name:     "one membership starts before it ends",
			coverage: validationeval.CoverageComplete,
			rows: []driver.ContractQueryRow{
				row(map[core.CanonicalID]any{startsOn: "2026-04-01", endsOn: "2026-04-10"}),
				row(map[core.CanonicalID]any{startsOn: "2026-05-01", endsOn: "2026-04-10"}),
			},
			want: validationeval.Decision{Result: validationeval.ResultNotSatisfied},
		},
		{
			// The row's own reason is reported although the reader states these
			// are every row of the relation: a row that holds no day might
			// break the rule.
			name:     "a row that holds no day",
			coverage: validationeval.CoverageComplete,
			rows: []driver.ContractQueryRow{
				row(map[core.CanonicalID]any{startsOn: "2026-04-01", endsOn: nil}),
			},
			want: validationeval.Decision{Undecided: &validationeval.Undecided{
				Reason: validationeval.UndecidedValueNull, Attribute: endsOn,
			}},
		},
		{
			// A relation that may not be every row cannot be answered
			// satisfied, and the declaration states no refutation, so it cannot
			// be answered not-satisfied either.
			name:     "a partly read relation",
			coverage: validationeval.CoveragePartial,
			rows: []driver.ContractQueryRow{
				row(map[core.CanonicalID]any{startsOn: "2026-04-01", endsOn: "2026-04-01"}),
			},
			want: validationeval.Decision{Undecided: &validationeval.Undecided{
				Reason:    validationeval.UndecidedRelationPartiallyRead,
				Attribute: startsOn, Relation: membersID,
			}},
		},
		{
			// Section 8.1: no rows is no counterexample, so the relation holds.
			name:     "no rows",
			coverage: validationeval.CoverageComplete,
			rows:     nil,
			want:     validationeval.Decision{Result: validationeval.ResultSatisfied},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := decideEveryNotBefore(t, detailOf(t, tc.coverage, tc.rows...))
			if got.Result != tc.want.Result {
				t.Fatalf("result=%q, want %q", got.Result, tc.want.Result)
			}
			if tc.want.Undecided == nil {
				if got.Undecided != nil {
					t.Fatalf("undecided=%v, want a decided answer", got.Undecided)
				}
				return
			}
			if got.Undecided == nil || got.Undecided.Reason != tc.want.Undecided.Reason ||
				got.Undecided.Attribute != tc.want.Undecided.Attribute || got.Undecided.Relation != tc.want.Undecided.Relation {
				t.Fatalf("undecided=%v, want %v", got.Undecided, tc.want.Undecided)
			}
		})
	}
}

// The caller states nothing about the rows, so a rule over all of them is
// refused: the rows are never taken to be complete by default.
func TestARowScopedDayRuleIsRefusedWhenNothingIsStatedAboutTheRows(t *testing.T) {
	detail := detailOf(t, "", row(map[core.CanonicalID]any{startsOn: "2026-04-01", endsOn: "2026-04-01"}))
	implementations, err := validationeval.ResolveImplementations(
		[]validationeval.ImplementationSelection{{Operator: everyNotBefore.ID, Implementation: "tei.day.every-not-before"}},
		validationeval.BuiltinImplementations(), nil)
	if err != nil {
		t.Fatalf("the shipped implementation could not be resolved: %v", err)
	}
	if _, err := detail.Decide(everyNotBeforeRule, validationeval.Operators{
		Declared:        []semantic.Operator{everyNotBefore},
		Implementations: implementations,
	}); err == nil || !strings.Contains(err.Error(), "unknown coverage") {
		t.Fatalf("err=%v, want the coverage refusal", err)
	}
}

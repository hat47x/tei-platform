package validationread_test

import (
	"math/big"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule whose operator reads every row of a relation in a type other than
// number is decided by the reader that understands that type over rows
// (decision section 8.4). Before this, no reader implemented
// StatefulTypedRowSource, so such a rule could only be decided by a test that
// built its own reader: the shipped reader could not answer it at all.
//
// The expectations below are written from the meaning: the column is present
// only when every row was read and holds a value, and the value of each row
// carries the type the declaration states, so a day is a day and a text is a
// text rather than the number either would parse as.

const (
	shippedAt  core.CanonicalID = "sales.order_line.shipped_at"
	deadlineAt core.CanonicalID = "sales.order_line.deadline_at"
	quantity   core.CanonicalID = "sales.order_line.quantity"
	itemCode   core.CanonicalID = "sales.order_line.item_code"
	urgent     core.CanonicalID = "sales.order_line.urgent"
	orderLines core.CanonicalID = "sales.order.lines"
)

var shippedSemantics = []semantic.Namespace{
	{ID: "sales", Data: []semantic.Data{{ID: "sales.order_line", Attributes: []semantic.Attribute{
		{ID: shippedAt, Symbol: "shippedAt", TypeRef: "std.date"},
		{ID: deadlineAt, Symbol: "deadlineAt", TypeRef: "std.date"},
		{ID: quantity, Symbol: "quantity", TypeRef: "std.integer"},
		{ID: itemCode, Symbol: "itemCode", TypeRef: "std.text"},
		{ID: urgent, Symbol: "urgent", TypeRef: "std.truth"},
	}}}},
	{ID: "std", Types: []semantic.Type{
		{ID: "std.integer", Kind: semantic.TypePrimitive},
		{ID: "std.date", Kind: semantic.TypePrimitive},
		{ID: "std.text", Kind: semantic.TypePrimitive},
		{ID: "std.truth", Kind: semantic.TypePrimitive},
	}},
}

var shippedRelation = semantic.Relation{
	ID: orderLines, Symbol: "lines", Target: "sales.order_line", Cardinality: semantic.CardinalityMany,
}

var shippedData = semantic.Data{ID: "sales.order_line", Attributes: []semantic.Attribute{
	{ID: shippedAt, Symbol: "shippedAt", TypeRef: "std.date"},
	{ID: deadlineAt, Symbol: "deadlineAt", TypeRef: "std.date"},
	{ID: quantity, Symbol: "quantity", TypeRef: "std.integer"},
	{ID: itemCode, Symbol: "itemCode", TypeRef: "std.text"},
	{ID: urgent, Symbol: "urgent", TypeRef: "std.truth"},
}}

func shippedDetail(t *testing.T, coverage validationeval.Coverage, rows ...driver.ContractQueryRow) *validationread.Detail {
	t.Helper()
	detail, err := validationread.NewDetail(shippedSemantics, shippedRelation, shippedData, rows, coverage)
	if err != nil {
		t.Fatalf("cannot build the detail: %v", err)
	}
	return detail
}

func shippedRow(values map[core.CanonicalID]any) driver.ContractQueryRow {
	return driver.ContractQueryRow{Attributes: values}
}

func mustDay(t *testing.T, text string) semantic.CalendarDate {
	t.Helper()
	day, err := semantic.ParseCalendarDate(text)
	if err != nil {
		t.Fatalf("%q is not a calendar date: %v", text, err)
	}
	return day
}

func readShippedTypedRows(t *testing.T, detail *validationread.Detail, inputType semantic.OperatorInput, attribute core.CanonicalID) validationeval.TypedRowReading {
	t.Helper()
	reading, err := detail.ReadTypedRows(semantic.OperatorInputDeclaration{
		Name: "value", Type: inputType,
	}, orderLines, attribute)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	return reading
}

// A column of calendar days is read as days, so an implementation that decides
// days receives days and not the numbers of the same column.
func TestAColumnOfCalendarDatesIsReadAsDays(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{shippedAt: "2026-04-06"}),
		shippedRow(map[core.CanonicalID]any{shippedAt: "2026-04-07"}),
	)
	reading := readShippedTypedRows(t, detail, semantic.OperatorInputDate, shippedAt)
	if reading.State != validationeval.ValuePresent || len(reading.Values) != 2 {
		t.Fatalf("reading=%+v, want two present rows", reading)
	}
	for index, text := range []string{"2026-04-06", "2026-04-07"} {
		want := mustDay(t, text)
		got := reading.Values[index].Value.Date
		if got == nil || got.Compare(want) != 0 {
			t.Fatalf("row %d holds %v, want %s", index, got, text)
		}
	}
}

// A row the query did not return keeps its own state beside its value, so the
// row is not read as the zero of its type and a counterexample in another row is
// still visible (decision section 6: a counterexample that was seen settles the
// rule whatever else is unknown).
func TestAColumnWithARowNobodyReadIsNotRead(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{shippedAt: "2026-04-06"}),
		shippedRow(map[core.CanonicalID]any{}),
	)
	reading := readShippedTypedRows(t, detail, semantic.OperatorInputDate, shippedAt)
	if len(reading.Values) != 2 {
		t.Fatalf("values=%d, want one per row of the relation", len(reading.Values))
	}
	if reading.Values[0].State != validationeval.ValuePresent {
		t.Fatalf("row 0 state=%q, want present", reading.Values[0].State)
	}
	if reading.Values[1].State != validationeval.ValueNotFetched {
		t.Fatalf("row 1 state=%q, want not-fetched", reading.Values[1].State)
	}
	if reading.Values[1].Value.Kind() != "" {
		t.Fatalf("row 1 carries %v, want no value for a row nobody read", reading.Values[1].Value)
	}
}

// A row that holds no value keeps that state beside its value, so it is not read
// as the zero of its type and the rest of the column stays readable.
func TestAColumnWithARowThatHoldsNoValueIsNotRead(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{shippedAt: "2026-04-06"}),
		shippedRow(map[core.CanonicalID]any{shippedAt: nil}),
	)
	reading := readShippedTypedRows(t, detail, semantic.OperatorInputDate, shippedAt)
	if len(reading.Values) != 2 {
		t.Fatalf("values=%d, want one per row of the relation", len(reading.Values))
	}
	if reading.Values[1].State != validationeval.ValueNull {
		t.Fatalf("row 1 state=%q, want null", reading.Values[1].State)
	}
	if reading.Values[1].Value.Kind() != "" {
		t.Fatalf("row 1 carries %v, want no value", reading.Values[1].Value)
	}
}

// A column of numbers is read as numbers by the same reader, so a rule that
// declares a number input over rows keeps working.
func TestAColumnOfNumbersIsReadAsNumbers(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{quantity: big.NewRat(2, 1)}),
		shippedRow(map[core.CanonicalID]any{quantity: big.NewRat(5, 1)}),
	)
	reading := readShippedTypedRows(t, detail, semantic.OperatorInputNumber, quantity)
	if reading.State != validationeval.ValuePresent || len(reading.Values) != 2 {
		t.Fatalf("reading=%+v, want two present rows", reading)
	}
	for index, want := range []*big.Rat{big.NewRat(2, 1), big.NewRat(5, 1)} {
		got := reading.Values[index].Value.Number
		if got == nil || got.Cmp(want) != 0 {
			t.Fatalf("row %d holds %v, want %v", index, got, want)
		}
	}
}

// A text is read as a text, so a column of codes reaches an implementation that
// decides texts as texts and not as the numbers they would parse as.
func TestAColumnOfTextsIsReadAsTexts(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{itemCode: "A-100"}),
		shippedRow(map[core.CanonicalID]any{itemCode: "B-200"}),
	)
	reading := readShippedTypedRows(t, detail, semantic.OperatorInputText, itemCode)
	if reading.State != validationeval.ValuePresent || len(reading.Values) != 2 {
		t.Fatalf("reading=%+v, want two present rows", reading)
	}
	for index, want := range []string{"A-100", "B-200"} {
		got := reading.Values[index].Value.Text
		if got == nil || *got != want {
			t.Fatalf("row %d holds %v, want %s", index, got, want)
		}
	}
}

// A truth is read as a truth, so a column of flags reaches an implementation as
// flags and not as the numbers they would parse as.
func TestAColumnOfTruthsIsReadAsTruths(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{urgent: true}),
		shippedRow(map[core.CanonicalID]any{urgent: false}),
	)
	reading := readShippedTypedRows(t, detail, semantic.OperatorInputTruth, urgent)
	if reading.State != validationeval.ValuePresent || len(reading.Values) != 2 {
		t.Fatalf("reading=%+v, want two present rows", reading)
	}
	for index, want := range []bool{true, false} {
		got := reading.Values[index].Value.Truth
		if got == nil || *got != want {
			t.Fatalf("row %d holds %v, want %v", index, got, want)
		}
	}
}

// A calendar date is not read as a text, because the two ask different questions
// of the same column (section 7).
func TestAColumnOfDatesIsNotReadAsTexts(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{shippedAt: "2026-04-06"}),
	)
	if _, err := detail.ReadTypedRows(semantic.OperatorInputDeclaration{
		Name: "value", Type: semantic.OperatorInputText,
	}, orderLines, shippedAt); err == nil {
		t.Fatal("a column of calendar dates was read as texts")
	}
}

// Rows of another relation are refused rather than answered from the rows of
// this one.
func TestTypedRowsOfAnotherRelationAreRefused(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{shippedAt: "2026-04-06"}),
	)
	_, err := detail.ReadTypedRows(semantic.OperatorInputDeclaration{
		Name: "value", Type: semantic.OperatorInputDate,
	}, "sales.order.headers", shippedAt)
	if err == nil {
		t.Fatal("rows of another relation were read")
	}
}

// The row-scoped decision asks this reader, so a rule over a column of days is
// decided from the days the reader read.
func TestARuleOverAColumnOfDaysIsDecidedFromTheReader(t *testing.T) {
	detail := shippedDetail(t, validationeval.CoverageComplete,
		shippedRow(map[core.CanonicalID]any{
			shippedAt:  "2026-04-06",
			deadlineAt: "2026-04-05",
		}),
		shippedRow(map[core.CanonicalID]any{
			shippedAt:  "2026-04-01",
			deadlineAt: "2026-04-05",
		}),
	)
	rule := semantic.Validation{
		ID: "sales.order.lines.shipped_not_before_deadline", Symbol: "shipped_not_before_deadline",
		Kind:      semantic.ValidationOperator,
		Attribute: shippedAt,
		Relation:  orderLines,
		Operator:  "sales.every_day_at_least",
		Inputs: []semantic.ValidationInputAssignment{
			{Name: "deadline", Value: string(deadlineAt)},
		},
	}
	contract := semantic.Operator{
		ID: "sales.every_day_at_least", Symbol: "every_day_at_least",
		Input: semantic.OperatorInputDate,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputDate, OfEachRow: true},
			{Name: "deadline", Type: semantic.OperatorInputDate, OfEachRow: true},
		},
		RefutesOnSomeRows: true,
	}
	implementation := validationeval.OperatorImplementation{
		Operator: "sales.every_day_at_least",
		Input:    semantic.OperatorInputDate,
		ValueRows: func(rows map[string][]validationeval.RowValueReading) (validationeval.Tri, error) {
			values, deadlines := rows["value"], rows["deadline"]
			if len(values) != len(deadlines) {
				return validationeval.TriMaybe, nil
			}
			for index := range values {
				if values[index].Value.Date == nil || deadlines[index].Value.Date == nil {
					return validationeval.TriMaybe, nil
				}
				if values[index].Value.Date.Compare(*deadlines[index].Value.Date) < 0 {
					return validationeval.TriNo, nil
				}
			}
			return validationeval.TriYes, nil
		},
	}
	decision, err := validationeval.Decide(rule, detail, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{implementation},
	})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	// 2026-04-01 is before the deadline 2026-04-05, and the contract says a
	// witnessed counterexample among the rows read is enough.
	if decision.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("decision=%+v, want not-satisfied", decision)
	}
}

package validationread_test

import (
	"math/big"
	"reflect"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// These tests read rows built by hand. The expected readings are written from
// the rule in the package comment: a value is present, an explicit nil is
// null, and an attribute the row does not hold was not fetched.

const (
	membersID  core.CanonicalID = "organization.department.members"
	personID   core.CanonicalID = "organization.membership.person_id"
	seatID     core.CanonicalID = "organization.membership.seat_number"
	allocation core.CanonicalID = "organization.membership.allocation"
	startsOn   core.CanonicalID = "organization.membership.starts_on"
	endsOn     core.CanonicalID = "organization.membership.ends_on"
	registered core.CanonicalID = "organization.membership.registered_at"
)

var membershipData = semantic.Data{
	ID: "organization.membership",
	Attributes: []semantic.Attribute{
		{ID: personID, Symbol: "personId", TypeRef: "std.identifier"},
		{ID: seatID, Symbol: "seatNumber", TypeRef: "std.integer"},
		// An alias of a number type is a number.
		{ID: allocation, Symbol: "allocation", TypeRef: "organization.share"},
		{ID: startsOn, Symbol: "startsOn", TypeRef: "std.date"},
		{ID: endsOn, Symbol: "endsOn", TypeRef: "std.date"},
		{ID: registered, Symbol: "registeredAt", TypeRef: "std.datetime"},
	},
}

var membersRelation = semantic.Relation{ID: membersID, Symbol: "members", Target: "organization.membership", Cardinality: semantic.CardinalityMany}

var testSemantics = []semantic.Namespace{
	{ID: "organization",
		Types: []semantic.Type{{ID: "organization.share", Symbol: "Share", Kind: semantic.TypeAlias, Underlying: "std.decimal"}},
		Data:  []semantic.Data{membershipData}},
	{ID: "std", Types: []semantic.Type{
		{ID: "std.identifier", Kind: semantic.TypePrimitive}, {ID: "std.integer", Kind: semantic.TypePrimitive},
		{ID: "std.decimal", Kind: semantic.TypePrimitive}, {ID: "std.date", Kind: semantic.TypePrimitive},
		{ID: "std.datetime", Kind: semantic.TypePrimitive},
	}},
}

func row(values map[core.CanonicalID]any) driver.ContractQueryRow {
	return driver.ContractQueryRow{Attributes: values}
}

func detailOf(t *testing.T, coverage validationeval.Coverage, rows ...driver.ContractQueryRow) *validationread.Detail {
	t.Helper()
	detail, err := validationread.NewDetail(testSemantics, membersRelation, membershipData, rows, coverage)
	if err != nil {
		t.Fatal(err)
	}
	return detail
}

func ratOf(text string) *big.Rat {
	value, ok := new(big.Rat).SetString(text)
	if !ok {
		panic(text)
	}
	return value
}

func TestARowSaysWhetherAValueIsThereAndWhyNot(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete, row(map[core.CanonicalID]any{
		allocation: "0.40", // present
		endsOn:     nil,    // the record holds no value
		// startsOn is not held: the query did not return it
	}))
	one, err := detail.Row(0)
	if err != nil {
		t.Fatal(err)
	}

	number, err := one.ReadNumber(allocation)
	if err != nil || number.State != validationeval.ValuePresent || number.Value.Cmp(ratOf("2/5")) != 0 {
		t.Fatalf("number=%+v err=%v", number, err)
	}
	end, err := one.ReadDate(endsOn)
	if err != nil || end != (validationeval.DateReading{State: validationeval.ValueNull}) {
		t.Fatalf("end=%+v err=%v", end, err)
	}
	start, err := one.ReadDate(startsOn)
	if err != nil || start != (validationeval.DateReading{State: validationeval.ValueNotFetched}) {
		t.Fatalf("start=%+v err=%v", start, err)
	}
	// The same three states for a number.
	other := detailOf(t, validationeval.CoverageComplete, row(map[core.CanonicalID]any{allocation: nil}), row(map[core.CanonicalID]any{}))
	for index, want := range []validationeval.ValueState{validationeval.ValueNull, validationeval.ValueNotFetched} {
		reader, _ := other.Row(index)
		got, err := reader.ReadNumber(allocation)
		if err != nil || got.State != want || got.Value != nil {
			t.Fatalf("row %d: got=%+v err=%v, want %s", index, got, err, want)
		}
	}
}

func TestNumbersAreReadOnlyFromExactRepresentations(t *testing.T) {
	accepted := []struct {
		value any
		want  string
	}{
		{int64(3), "3"}, {int32(-2), "-2"}, {7, "7"}, {"0.40", "2/5"}, {[]byte("12.5"), "25/2"},
		{ratOf("1/3"), "1/3"}, {"-0", "0"},
	}
	for _, tc := range accepted {
		reader, _ := detailOf(t, "", row(map[core.CanonicalID]any{allocation: tc.value})).Row(0)
		got, err := reader.ReadNumber(allocation)
		if err != nil || got.State != validationeval.ValuePresent || got.Value.Cmp(ratOf(tc.want)) != 0 {
			t.Fatalf("%#v: got=%+v err=%v, want %s", tc.value, got, err, tc.want)
		}
	}
	// The value handed out is a copy: a rule cannot change the row.
	held := ratOf("5")
	reader, _ := detailOf(t, "", row(map[core.CanonicalID]any{allocation: held})).Row(0)
	got, _ := reader.ReadNumber(allocation)
	got.Value.SetInt64(0)
	if held.Cmp(ratOf("5")) != 0 {
		t.Fatal("reading a number changed the row")
	}

	refused := []struct {
		value   any
		mention string
	}{
		{0.4, "binary floating point"}, {float32(1), "binary floating point"},
		{"1e2", "not a decimal number"}, {"1,000", "not a decimal number"}, {" 3", "not a decimal number"}, {"", "not a decimal number"},
		{true, "is not a number"}, {time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), "is not a number"},
		{(*big.Rat)(nil), "is not a number"},
	}
	for _, tc := range refused {
		reader, _ := detailOf(t, "", row(map[core.CanonicalID]any{allocation: tc.value})).Row(0)
		got, err := reader.ReadNumber(allocation)
		if err == nil || !strings.Contains(err.Error(), tc.mention) || !strings.Contains(err.Error(), string(allocation)) || got.State != "" {
			t.Fatalf("%#v: got=%+v err=%v, want an error mentioning %q", tc.value, got, err, tc.mention)
		}
	}
}

func TestDatesAreReadOnlyAsWholeDays(t *testing.T) {
	want := semantic.CalendarDate{Year: 2026, Month: 4, Day: 1}
	tokyo := time.FixedZone("JST", 9*60*60)
	accepted := []any{
		"2026-04-01", []byte("2026-04-01"), want,
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		// Midnight in its own zone is that zone's day; it is not moved to UTC,
		// where it would be the day before.
		time.Date(2026, 4, 1, 0, 0, 0, 0, tokyo),
	}
	for _, value := range accepted {
		reader, _ := detailOf(t, "", row(map[core.CanonicalID]any{startsOn: value})).Row(0)
		got, err := reader.ReadDate(startsOn)
		if err != nil || got != (validationeval.DateReading{State: validationeval.ValuePresent, Value: want}) {
			t.Fatalf("%#v: got=%+v err=%v", value, got, err)
		}
	}
	refused := []struct {
		value   any
		mention string
	}{
		{time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC), "has a time of day"},
		{time.Date(2026, 4, 1, 0, 0, 0, 1, time.UTC), "has a time of day"},
		{"2026/04/01", "not a calendar date"}, {"2026-04-01T00:00:00Z", "not a calendar date"}, {"2026-02-30", "not a calendar date"},
		{int64(20260401), "is not a calendar date"}, {semantic.CalendarDate{}, "zero calendar date"},
	}
	for _, tc := range refused {
		reader, _ := detailOf(t, "", row(map[core.CanonicalID]any{startsOn: tc.value})).Row(0)
		got, err := reader.ReadDate(startsOn)
		if err == nil || !strings.Contains(err.Error(), tc.mention) || got.State != "" {
			t.Fatalf("%#v: got=%+v err=%v, want an error mentioning %q", tc.value, got, err, tc.mention)
		}
	}
}

func TestKeysCompareValuesByTheDeclaredType(t *testing.T) {
	keysOf := func(attribute core.CanonicalID, values ...any) []validationeval.KeyReading {
		t.Helper()
		rows := make([]driver.ContractQueryRow, 0, len(values))
		for _, value := range values {
			rows = append(rows, row(map[core.CanonicalID]any{attribute: value}))
		}
		reading, err := detailOf(t, validationeval.CoverageComplete, rows...).ReadRelationKeys(membersID, attribute)
		if err != nil {
			t.Fatal(err)
		}
		if reading.Coverage != validationeval.CoverageComplete {
			t.Fatalf("coverage=%q", reading.Coverage)
		}
		return reading.Keys
	}
	same := func(keys []validationeval.KeyReading) bool {
		for _, key := range keys[1:] {
			if key.State != validationeval.ValuePresent || key.Key != keys[0].Key {
				return false
			}
		}
		return keys[0].State == validationeval.ValuePresent
	}
	// One number, however it was returned.
	if keys := keysOf(seatID, int64(12), "12", "12.0", []byte("12"), ratOf("12")); !same(keys) {
		t.Fatalf("one seat number has several keys: %+v", keys)
	}
	// One day, however it was returned.
	if keys := keysOf(startsOn, "2026-04-01", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)); !same(keys) {
		t.Fatalf("one day has several keys: %+v", keys)
	}
	// Texts are compared as observed; an integer is not the text of its digits.
	different := keysOf(personID, "P001", "p001", "P001 ", int64(1), "1")
	seen := map[string]bool{}
	for _, key := range different {
		if key.State != validationeval.ValuePresent || seen[key.Key] {
			t.Fatalf("different values share a key: %+v", different)
		}
		seen[key.Key] = true
	}
	// Null and not fetched stay what they are, and carry no key.
	rows := []driver.ContractQueryRow{row(map[core.CanonicalID]any{personID: nil}), row(map[core.CanonicalID]any{})}
	reading, err := detailOf(t, validationeval.CoveragePartial, rows...).ReadRelationKeys(membersID, personID)
	want := validationeval.RelationReading{Coverage: validationeval.CoveragePartial, Keys: []validationeval.KeyReading{
		{State: validationeval.ValueNull}, {State: validationeval.ValueNotFetched},
	}}
	if err != nil || !reflect.DeepEqual(reading, want) {
		t.Fatalf("reading=%+v err=%v", reading, err)
	}
	// A value that cannot be compared is refused, not turned into a text.
	if _, err := detailOf(t, "", row(map[core.CanonicalID]any{personID: 1.5})).ReadRelationKeys(membersID, personID); err == nil {
		t.Fatal("a floating point value was used as a key")
	}
	if _, err := detailOf(t, "", row(map[core.CanonicalID]any{seatID: 12.0})).ReadRelationKeys(membersID, seatID); err == nil {
		t.Fatal("a floating point seat number was used as a key")
	}
}

func TestPeriodsAreReadWithTheirStates(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete,
		row(map[core.CanonicalID]any{personID: "P001", startsOn: "2026-04-01", endsOn: "2026-09-30"}),
		row(map[core.CanonicalID]any{personID: nil, startsOn: "2026-10-01", endsOn: nil}),
		row(map[core.CanonicalID]any{startsOn: "2026-10-01"}),
	)
	day := func(text string) validationeval.DateReading {
		value, err := semantic.ParseCalendarDate(text)
		if err != nil {
			t.Fatal(err)
		}
		return validationeval.DateReading{State: validationeval.ValuePresent, Value: value}
	}
	got, err := detail.ReadRelationPeriods(membersID, startsOn, endsOn, personID)
	want := validationeval.PeriodRowsReading{Coverage: validationeval.CoverageComplete, Rows: []validationeval.PeriodRow{
		{Start: day("2026-04-01"), End: day("2026-09-30"), Group: validationeval.KeyReading{State: validationeval.ValuePresent, Key: "text:P001"}},
		{Start: day("2026-10-01"), End: validationeval.DateReading{State: validationeval.ValueNull}, Group: validationeval.KeyReading{State: validationeval.ValueNull}},
		{Start: day("2026-10-01"), End: validationeval.DateReading{State: validationeval.ValueNotFetched}, Group: validationeval.KeyReading{State: validationeval.ValueNotFetched}},
	}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	// Without a grouping attribute the group is not read at all.
	ungrouped, err := detail.ReadRelationPeriods(membersID, startsOn, endsOn, "")
	if err != nil || ungrouped.Rows[0].Group != (validationeval.KeyReading{}) {
		t.Fatalf("ungrouped=%+v err=%v", ungrouped, err)
	}
}

// A row-scoped input reads one attribute over every row of the relation, in the
// order the rows were read, and reports whether those are all the rows. Reading
// one attribute of one row is a different question and must not be answered.
func TestARowScopedInputReadsEveryRowWithItsCoverage(t *testing.T) {
	detail := detailOf(t, validationeval.CoveragePartial,
		row(map[core.CanonicalID]any{allocation: 3}),
		row(map[core.CanonicalID]any{allocation: nil}),
		row(map[core.CanonicalID]any{startsOn: "2026-04-01"}),
	)
	want := validationeval.NumberRowsReading{
		Coverage: validationeval.CoveragePartial,
		Rows: []validationeval.NumberReading{
			{State: validationeval.ValuePresent, Value: ratOf("3")},
			{State: validationeval.ValueNull},
			{State: validationeval.ValueNotFetched},
		},
	}
	got, err := detail.ReadRelationNumbers(membersID, allocation)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v err=%v, want %+v", got, err, want)
	}

	// A value the reader cannot turn into a number is the reader's error, not a
	// value that was absent: recording it as "not fetched" would answer a
	// question about the data with a statement about the read.
	unreadable := detailOf(t, validationeval.CoverageComplete,
		row(map[core.CanonicalID]any{allocation: "not a number"}),
	)
	if _, err := unreadable.ReadRelationNumbers(membersID, allocation); err == nil {
		t.Fatal("an unreadable numeric row was reported as a reading")
	}

	// One row does not stand for the rows of a relation.
	one, err := detail.Row(0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := one.ReadRelationNumbers(membersID, allocation); err == nil {
		t.Fatal("a single row answered as the rows of a relation")
	}
}

func TestTheReaderRefusesWhatItWasNotGiven(t *testing.T) {
	detail := detailOf(t, validationeval.CoverageComplete, row(map[core.CanonicalID]any{allocation: "1", startsOn: "2026-04-01"}))
	one, _ := detail.Row(0)
	refusals := []struct {
		name    string
		err     func() error
		mention string
	}{
		{"another relation", func() error { _, err := detail.ReadRelationKeys("organization.department.heads", personID); return err }, "not of organization.department.heads"},
		{"periods of another relation", func() error {
			_, err := detail.ReadRelationPeriods("organization.person.memberships", startsOn, endsOn, "")
			return err
		}, "not of organization.person.memberships"},
		{"an attribute of another data", func() error { _, err := detail.ReadRelationKeys(membersID, "organization.department.name"); return err }, "is not an item of organization.membership"},
		{"a number from a date", func() error { _, err := one.ReadNumber(startsOn); return err }, "is not declared as a number"},
		{"a date from a number", func() error { _, err := one.ReadDate(allocation); return err }, "is not declared as a calendar date"},
		{"a date from a moment in time", func() error { _, err := one.ReadDate(registered); return err }, "is not declared as a calendar date"},
		{"a single value from all the rows", func() error { _, err := detail.ReadNumber(allocation); return err }, "reads one row"},
		{"a single date from all the rows", func() error { _, err := detail.ReadDate(startsOn); return err }, "reads one row"},
		{"all the rows from one row", func() error { _, err := one.ReadRelationKeys(membersID, personID); return err }, "reads the Detail"},
		{"a row that does not exist", func() error { _, err := detail.Row(1); return err }, "does not exist"},
		{"a row before the first", func() error { _, err := detail.Row(-1); return err }, "does not exist"},
	}
	for _, tc := range refusals {
		if err := tc.err(); err == nil || !strings.Contains(err.Error(), tc.mention) {
			t.Fatalf("%s: err=%v, want an error mentioning %q", tc.name, err, tc.mention)
		}
	}
	if detail.Len() != 1 {
		t.Fatalf("len=%d", detail.Len())
	}

	// Rows that cannot be the rows of the relation are refused when the
	// reader is made.
	if _, err := validationread.NewDetail(testSemantics, membersRelation, membershipData,
		[]driver.ContractQueryRow{row(map[core.CanonicalID]any{"organization.department.name": "営業部"})}, validationeval.CoverageComplete); err == nil ||
		!strings.Contains(err.Error(), "is not an item of organization.membership") {
		t.Fatalf("err=%v", err)
	}
	if _, err := validationread.NewDetail(testSemantics, membersRelation, membershipData, nil, "all"); err == nil || !strings.Contains(err.Error(), "unknown coverage") {
		t.Fatalf("err=%v", err)
	}
	other := semantic.Data{ID: "organization.leave"}
	if _, err := validationread.NewDetail(testSemantics, membersRelation, other, nil, validationeval.CoverageComplete); err == nil ||
		!strings.Contains(err.Error(), "leads to organization.membership") {
		t.Fatalf("err=%v", err)
	}
}

func TestFromHeaderDetailReadsTheRowsOfThePatternRelation(t *testing.T) {
	execution := driver.HeaderDetailContractQueryExecution{
		Resolution: driver.ResolvedHeaderDetailContractQuery{
			Relation: driver.ResolvedRelation{Relation: membersRelation, Target: membershipData},
			Query:    database.QueryContract{Name: "members_of_department", Result: "organization.membership", ResultCardinality: database.QueryResultMany},
		},
		Result: driver.ContractQueryResult{Rows: []driver.ContractQueryRow{row(map[core.CanonicalID]any{personID: "P001"})}},
	}
	detail, err := validationread.FromHeaderDetail(testSemantics, execution, validationeval.CoverageComplete)
	if err != nil {
		t.Fatal(err)
	}
	reading, err := detail.ReadRelationKeys(membersID, personID)
	if err != nil || len(reading.Keys) != 1 || reading.Keys[0].Key != "text:P001" {
		t.Fatalf("reading=%+v err=%v", reading, err)
	}

	wrongResult := execution
	wrongResult.Resolution.Query.Result = "organization.leave"
	if _, err := validationread.FromHeaderDetail(testSemantics, wrongResult, validationeval.CoverageComplete); err == nil ||
		!strings.Contains(err.Error(), "returns organization.leave") {
		t.Fatalf("err=%v", err)
	}
	if _, err := validationread.FromHeaderDetail(testSemantics, driver.HeaderDetailContractQueryExecution{}, validationeval.CoverageComplete); err == nil ||
		!strings.Contains(err.Error(), "names no relation") {
		t.Fatalf("err=%v", err)
	}
}

// When the caller states nothing about coverage, a rule over all the rows is
// refused. The rows are not taken to be complete.
func TestRowsAreNotCompleteUnlessTheCallerSaysSo(t *testing.T) {
	unique := semantic.Validation{ID: "organization.department.members.person_unique", Kind: semantic.ValidationRelationUnique,
		Attribute: personID, Relation: membersID}
	rows := []driver.ContractQueryRow{row(map[core.CanonicalID]any{personID: "P001"}), row(map[core.CanonicalID]any{personID: "P002"})}

	got, err := validationeval.Decide(unique, detailOf(t, "", rows...), validationeval.Operators{})
	if err == nil || !strings.Contains(err.Error(), "unknown coverage") || got.Result != "" || got.Undecided != nil {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	got, err = validationeval.Decide(unique, detailOf(t, validationeval.CoveragePartial, rows...), validationeval.Operators{})
	if err != nil || got.Undecided == nil || got.Undecided.Reason != validationeval.UndecidedRelationPartiallyRead {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	got, err = validationeval.Decide(unique, detailOf(t, validationeval.CoverageComplete, rows...), validationeval.Operators{})
	if err != nil || got.Result != validationeval.ResultSatisfied {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

package validationread_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tei/internal/core"
	"tei/internal/database"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/driver/validationread"
	"tei/internal/environment"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// These tests read rules' input from a real database. A SQLite file is
// created and filled with SQL, the product SQL Executor runs a Contract Query
// against it through an explicit SQL realization, the product mapping turns
// the result columns into semantic rows, and this package reads those rows.
// Nothing between the database and the rule is a stub except the few lines
// that hand a SQL text to database/sql.
//
// The expected answers are written by hand from the rows inserted below.

type sqliteRunner struct{ db *sql.DB }

func (r sqliteRunner) ExecuteSQL(ctx context.Context, source string, parameters map[string]any) (sqlquery.RawExecution, error) {
	args := make([]any, 0, len(parameters))
	for name, value := range parameters {
		args = append(args, sql.Named(name, value))
	}
	rows, err := r.db.QueryContext(ctx, source, args...)
	if err != nil {
		return sqlquery.RawExecution{}, err
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return sqlquery.RawExecution{}, err
	}
	raw := queryresult.RawResult{Columns: columns}
	for rows.Next() {
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return sqlquery.RawExecution{}, err
		}
		raw.Rows = append(raw.Rows, values)
	}
	return sqlquery.RawExecution{Result: raw}, rows.Err()
}

const membershipSchema = `
create table membership (
  department_id text not null,
  person_id     text,
  seat_number   integer,
  allocation    text,   -- an exact decimal, kept as text
  load_factor   real,   -- a binary floating point column
  starts_on     text,
  ends_on       text
);
insert into membership values
  ('D1', 'P001', 1, '1',   1.0, '2026-04-01', '2026-09-30'),
  ('D1', 'P001', 2, '0.5', 0.5, '2026-10-01', null),
  ('D1', 'P002', 3, '0',   0.0, '2026-04-01', null),
  ('D1', 'P003', 4, null,  null, '2026-05-01', '2026-04-30'),
  ('D2', 'P010', 1, '1',   1.0, '2026-04-01', '2026-09-30'),
  ('D2', 'P010', 1, '1',   1.0, '2026-09-30', null);
`

func membersTarget(queryName string) realization.ContractQueryTarget {
	return realization.ContractQueryTarget{Database: "organization_db", Contract: "organization_db.membership", Query: core.LocalName(queryName)}
}

func membersContract(queryName string) database.QueryContract {
	return database.QueryContract{
		Name:              queryName,
		Inputs:            []database.QueryInput{{Name: "department", Type: "std.identifier"}},
		Result:            "organization.membership",
		ResultCardinality: database.QueryResultMany,
	}
}

func membersRealization(queryName, selectList, tail string, columns ...query.ResultColumnMapping) realization.ContractQuerySQLRealization {
	return realization.ContractQuerySQLRealization{
		Target:        membersTarget(queryName),
		Provider:      environment.ProviderSQLite,
		Source:        "select " + selectList + " from membership where department_id = :department_id order by seat_number, starts_on" + tail,
		Inputs:        []realization.ContractQueryInputParameterMapping{{Input: "department", Parameter: "department_id"}},
		ResultColumns: columns,
	}
}

var (
	columnPerson = query.ResultColumnMapping{Column: "person_id", Attribute: personID}
	columnSeat   = query.ResultColumnMapping{Column: "seat_number", Attribute: seatID}
	columnShare  = query.ResultColumnMapping{Column: "allocation", Attribute: allocation}
	columnStart  = query.ResultColumnMapping{Column: "starts_on", Attribute: startsOn}
	columnEnd    = query.ResultColumnMapping{Column: "ends_on", Attribute: endsOn}
)

func membershipExecutor(t *testing.T) sqlquery.Executor {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "organization.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(membershipSchema); err != nil {
		t.Fatal(err)
	}
	return sqlquery.Executor{
		Provider: environment.ProviderSQLite,
		Runner:   sqliteRunner{db: db},
		Realizations: []realization.ContractQuerySQLRealization{
			membersRealization("members", "person_id, seat_number, allocation, starts_on, ends_on", "",
				columnPerson, columnSeat, columnShare, columnStart, columnEnd),
			// The same rows without two of the columns.
			membersRealization("members_narrow", "person_id, seat_number, starts_on", "", columnPerson, columnSeat, columnStart),
			// The first two rows only.
			membersRealization("members_first_page", "person_id, seat_number, allocation, starts_on, ends_on", " limit 2",
				columnPerson, columnSeat, columnShare, columnStart, columnEnd),
			// The share taken from a floating point column.
			membersRealization("members_float", "person_id, load_factor", "", columnPerson,
				query.ResultColumnMapping{Column: "load_factor", Attribute: allocation}),
		},
	}
}

// readMembers runs one Contract Query for one department and reads its rows
// as the members of that department.
func readMembers(t *testing.T, executor sqlquery.Executor, queryName, department string, coverage validationeval.Coverage) *validationread.Detail {
	t.Helper()
	contract := membersContract(queryName)
	result, err := executor.ExecuteContractQuery(context.Background(), membersTarget(queryName), contract, map[string]any{"department": department})
	if err != nil {
		t.Fatalf("the Contract Query did not run: %v", err)
	}
	execution := driver.HeaderDetailContractQueryExecution{
		Resolution: driver.ResolvedHeaderDetailContractQuery{
			Relation: driver.ResolvedRelation{Relation: membersRelation, Target: membershipData},
			Query:    contract,
		},
		Result: result,
	}
	detail, err := validationread.FromHeaderDetail(testSemantics, execution, coverage)
	if err != nil {
		t.Fatal(err)
	}
	return detail
}

var (
	allocationPositive = semantic.Validation{ID: "organization.membership.allocation_positive", Kind: semantic.ValidationPositive, Attribute: allocation}
	personUnique       = semantic.Validation{ID: "organization.department.members.person_unique", Kind: semantic.ValidationRelationUnique, Attribute: personID, Relation: membersID}
	seatUnique         = semantic.Validation{ID: "organization.department.members.seat_unique", Kind: semantic.ValidationRelationUnique, Attribute: seatID, Relation: membersID}
	periodInOrder      = semantic.Validation{ID: "organization.membership.period_in_order", Kind: semantic.ValidationNotAfter, Attribute: startsOn, Compared: endsOn, IfPresent: true}
	periodsApart       = semantic.Validation{ID: "organization.department.members.periods_apart", Kind: semantic.ValidationNoOverlap,
		Attribute: startsOn, Compared: endsOn, Relation: membersID, GroupBy: personID, OpenEnded: true}
)

// answer is a decision written the way the cases below write it: a result,
// or "undecided: <reason>".
func answer(t *testing.T, rule semantic.Validation, source validationeval.StatefulSource) string {
	t.Helper()
	decision, err := validationeval.Decide(rule, source, validationeval.Operators{})
	if err != nil {
		t.Fatalf("%s: %v", rule.ID, err)
	}
	if decision.Undecided != nil {
		return "undecided: " + string(decision.Undecided.Reason) + " " + string(decision.Undecided.Attribute)
	}
	return string(decision.Result)
}

func perRow(t *testing.T, rule semantic.Validation, detail *validationread.Detail) []string {
	t.Helper()
	out := make([]string, 0, detail.Len())
	for index := 0; index < detail.Len(); index++ {
		one, err := detail.Row(index)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, answer(t, rule, one))
	}
	return out
}

func sameAnswers(t *testing.T, what string, got, want []string) {
	t.Helper()
	if strings.Join(got, " | ") != strings.Join(want, " | ") {
		t.Fatalf("%s:\n got  %v\n want %v", what, got, want)
	}
}

// What the database driver hands over is observed here, so the reader's
// rules are checked against real values and not against assumed ones.
func TestWhatSQLiteReturns(t *testing.T) {
	executor := membershipExecutor(t)
	result, err := executor.ExecuteContractQuery(context.Background(), membersTarget("members"), membersContract("members"), map[string]any{"department": "D1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 4 {
		t.Fatalf("rows=%d", len(result.Rows))
	}
	types := func(row driver.ContractQueryRow) map[core.CanonicalID]string {
		out := map[core.CanonicalID]string{}
		for attribute, value := range row.Attributes {
			out[attribute] = fmt.Sprintf("%T", value)
		}
		return out
	}
	first, last := types(result.Rows[0]), types(result.Rows[3])
	if first[seatID] != "int64" || first[allocation] != "string" || first[startsOn] != "string" || first[personID] != "string" {
		t.Fatalf("first row types=%v", first)
	}
	// SQL NULL arrives as an attribute the row holds with nil.
	if value, held := result.Rows[3].Attributes[allocation]; !held || value != nil || last[allocation] != "<nil>" {
		t.Fatalf("NULL arrived as held=%v value=%#v", held, value)
	}
	// A column the query did not select is not in the row at all.
	narrow, err := executor.ExecuteContractQuery(context.Background(), membersTarget("members_narrow"), membersContract("members_narrow"), map[string]any{"department": "D1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, held := narrow.Rows[0].Attributes[endsOn]; held {
		t.Fatalf("an unselected column is in the row: %#v", narrow.Rows[0].Attributes)
	}
	float, err := executor.ExecuteContractQuery(context.Background(), membersTarget("members_float"), membersContract("members_float"), map[string]any{"department": "D1"})
	if err != nil {
		t.Fatal(err)
	}
	if kind := fmt.Sprintf("%T", float.Rows[0].Attributes[allocation]); kind != "float64" {
		t.Fatalf("a REAL column arrived as %s", kind)
	}
}

func TestRulesAreDecidedFromRowsOfARealDatabase(t *testing.T) {
	executor := membershipExecutor(t)

	// Department D1: P001 left and came back, P002 has share 0, P003 has no
	// share and a period that ends before it starts.
	d1 := readMembers(t, executor, "members", "D1", validationeval.CoverageComplete)
	sameAnswers(t, "share is positive, per row", perRow(t, allocationPositive, d1), []string{
		"satisfied", "satisfied", "not-satisfied", "undecided: value-null " + string(allocation),
	})
	sameAnswers(t, "start is not after end if present, per row", perRow(t, periodInOrder, d1), []string{
		"satisfied", "satisfied", "satisfied", "not-satisfied",
	})
	sameAnswers(t, "rules over all the rows of D1", []string{
		answer(t, personUnique, d1), answer(t, seatUnique, d1), answer(t, periodsApart, d1),
	}, []string{"not-satisfied", "satisfied", "satisfied"})

	// Department D2: P010 twice, on the same seat, sharing 2026-09-30.
	d2 := readMembers(t, executor, "members", "D2", validationeval.CoverageComplete)
	sameAnswers(t, "rules over all the rows of D2", []string{
		answer(t, personUnique, d2), answer(t, seatUnique, d2), answer(t, periodsApart, d2),
	}, []string{"not-satisfied", "not-satisfied", "not-satisfied"})

	// A department without members: nothing can collide.
	none := readMembers(t, executor, "members", "D9", validationeval.CoverageComplete)
	if none.Len() != 0 {
		t.Fatalf("rows=%d", none.Len())
	}
	sameAnswers(t, "rules over no rows", []string{answer(t, personUnique, none), answer(t, periodsApart, none)}, []string{"satisfied", "satisfied"})
}

// The narrow query returns neither the share nor the end. Those values were
// not fetched; they are not empty, and no rule treats them as empty.
func TestAColumnTheQueryDidNotReturnIsNotFetched(t *testing.T) {
	executor := membershipExecutor(t)
	narrow := readMembers(t, executor, "members_narrow", "D1", validationeval.CoverageComplete)
	sameAnswers(t, "share is positive", perRow(t, allocationPositive, narrow), []string{
		"undecided: value-not-fetched " + string(allocation), "undecided: value-not-fetched " + string(allocation),
		"undecided: value-not-fetched " + string(allocation), "undecided: value-not-fetched " + string(allocation),
	})
	// With the full query the second row has no end and satisfies the rule.
	// Here the end is unknown, so `if present` cannot apply.
	sameAnswers(t, "start is not after end if present", perRow(t, periodInOrder, narrow)[1:2], []string{
		"undecided: value-not-fetched " + string(endsOn),
	})
	// P001 has two rows whose ends are unknown: they might share a day.
	sameAnswers(t, "rules over all the rows", []string{
		answer(t, periodsApart, narrow), answer(t, seatUnique, narrow), answer(t, personUnique, narrow),
	}, []string{"undecided: value-not-fetched " + string(endsOn), "satisfied", "not-satisfied"})
}

// The first page holds two of the four rows of D1. The caller says so.
func TestAPageIsNotTheWholeRelation(t *testing.T) {
	executor := membershipExecutor(t)
	page := readMembers(t, executor, "members_first_page", "D1", validationeval.CoveragePartial)
	if page.Len() != 2 {
		t.Fatalf("rows=%d", page.Len())
	}
	sameAnswers(t, "rules over the first page", []string{
		// Both rows of the page are P001: a duplicate that was seen.
		answer(t, personUnique, page),
		// Seats 1 and 2 differ, but the other rows were not read.
		answer(t, seatUnique, page),
		answer(t, periodsApart, page),
	}, []string{"not-satisfied", "undecided: relation-partially-read " + string(seatID), "undecided: relation-partially-read " + string(startsOn)})

	// The same page read as if it were everything answers "satisfied" for
	// the seats. That statement is the caller's, and it is wrong here: the
	// reader cannot check it, which is why it is never assumed.
	claimed := readMembers(t, executor, "members_first_page", "D1", validationeval.CoverageComplete)
	if got := answer(t, seatUnique, claimed); got != "satisfied" {
		t.Fatalf("got %s", got)
	}
	// Stating nothing refuses the rule.
	unstated := readMembers(t, executor, "members_first_page", "D1", "")
	if _, err := validationeval.Decide(seatUnique, unstated, validationeval.Operators{}); err == nil || !strings.Contains(err.Error(), "unknown coverage") {
		t.Fatalf("err=%v", err)
	}
}

// A share taken from a REAL column is refused. 0.5 happens to be exact in
// binary; the reader does not look at the value to decide, because 0.1 is
// not, and a rule must not depend on which of the two a row holds.
func TestAFloatingPointColumnIsRefused(t *testing.T) {
	executor := membershipExecutor(t)
	float := readMembers(t, executor, "members_float", "D1", validationeval.CoverageComplete)
	one, err := float.Row(1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := validationeval.Decide(allocationPositive, one, validationeval.Operators{})
	if err == nil || !strings.Contains(err.Error(), "binary floating point") || got.Result != "" || got.Undecided != nil {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	// NULL in the same column is still just null.
	empty, _ := float.Row(3)
	if answer(t, allocationPositive, empty) != "undecided: value-null "+string(allocation) {
		t.Fatal("NULL in a floating point column was not read as null")
	}
}

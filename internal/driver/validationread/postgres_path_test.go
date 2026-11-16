package validationread_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/internal/core"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/driver/validationread"
	"tei/internal/environment"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/semantic/validationeval"
)

// These tests run against a PostgreSQL server and are opt-in. They need
// TEI_VALIDATION_READ_PG_DSN to point to a disposable database named
// tei_validation_read_e2e, and TEI_VALIDATION_READ_PG_CONFIRM_ISOLATED set to
// that name. They create and drop public.order_lines and public.membership.
//
// Nothing on the path is a stand-in: the reference Workspace's definitions,
// the product SQL Executor, the product PostgreSQL Runner, the pgx driver and
// the server. The expected answers are written by hand from the rows
// inserted below.

const validationReadDatabase = "tei_validation_read_e2e"

func openValidationReadPostgres(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEI_VALIDATION_READ_PG_DSN")
	if dsn == "" {
		t.Skip("rules from PostgreSQL rows NOT RUN: TEI_VALIDATION_READ_PG_DSN unset")
	}
	if os.Getenv("TEI_VALIDATION_READ_PG_CONFIRM_ISOLATED") != validationReadDatabase {
		t.Fatal("refusing to create tables without explicit disposable database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize the PostgreSQL driver")
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var name string
	if err := db.QueryRowContext(ctx, "select current_database()").Scan(&name); err != nil || name != validationReadDatabase {
		t.Fatalf("refusing to create tables in an unexpected database: %q", name)
	}
	return db
}

func createTables(t *testing.T, db *sql.DB, statements string, tables string) {
	t.Helper()
	if _, err := db.Exec("drop table if exists " + tables); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(statements); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists " + tables); err != nil {
			t.Errorf("tables were not dropped: %v", err)
		}
	})
}

// The table follows the persistence mapping of the reference Workspace:
// int64 columns are bigint, decimal columns are numeric, all required.
const orderLinesSchema = `
create table public.order_lines (
  order_line_id bigint primary key,
  order_id      bigint not null,
  line_number   bigint not null,
  item_code     varchar not null,
  quantity      numeric(12,3) not null,
  unit_price    numeric(12,2) not null
);
insert into public.order_lines values
  (11, 1001, 1, 'ITEM-A', 2.5, 1000),
  (12, 1001, 2, 'ITEM-B', 1,   500.5),
  (13, 1001, 3, 'ITEM-C', 0,   10),
  (21, 1002, 1, 'ITEM-A', 1,   100),
  (22, 1002, 1, 'ITEM-B', 2,   100),
  (23, 1002, 0, 'ITEM-C', 3,   100),
  (31, 1003, 7, 'ITEM-A', 0.001, 100);
`

func TestOrderLinesOfTheReferenceWorkspaceAreReadFromPostgreSQL(t *testing.T) {
	db := openValidationReadPostgres(t)
	createTables(t, db, orderLinesSchema, "public.order_lines")
	world := referenceWorld(t)
	runner := postgresquery.Runner{DB: db}

	// What the driver hands over for the reference columns.
	observed := executeOrderLines(t, world, runner, 1001).Execution.Result.Rows
	if len(observed) != 3 {
		t.Fatalf("rows=%d", len(observed))
	}
	types := map[core.CanonicalID]string{}
	for attribute, value := range observed[0].Attributes {
		types[attribute] = fmt.Sprintf("%T", value)
	}
	if types["sales.order_line.order_line_id"] != "int64" || types["sales.order_line.line_number"] != "int64" ||
		types["sales.order_line.item_code"] != "string" ||
		types["sales.order_line.quantity"] != "string" || types["sales.order_line.unit_price"] != "string" {
		t.Fatalf("PostgreSQL handed over %v", types)
	}
	if text := observed[0].Attributes["sales.order_line.quantity"]; text != "2.500" {
		t.Fatalf("numeric(12,3) 2.5 arrived as %q", text)
	}

	sameAnswers(t, "order 1001", answers(t, world, readOrderLines(t, world, runner, 1001, validationeval.CoverageComplete)), []string{
		"sales.order_line.quantity_positive: rows satisfied, satisfied, not-satisfied",
		"sales.order.lines.line_number_unique: relation satisfied",
		"sales.order_line.line_number_positive: rows satisfied, satisfied, satisfied",
	})
	// The reference SQL orders by line number, so the line with number 0
	// comes first.
	sameAnswers(t, "order 1002", answers(t, world, readOrderLines(t, world, runner, 1002, validationeval.CoverageComplete)), []string{
		"sales.order_line.quantity_positive: rows satisfied, satisfied, satisfied",
		"sales.order.lines.line_number_unique: relation not-satisfied",
		"sales.order_line.line_number_positive: rows not-satisfied, satisfied, satisfied",
	})
	// The smallest quantity the column can hold is positive.
	sameAnswers(t, "order 1003", answers(t, world, readOrderLines(t, world, runner, 1003, validationeval.CoverageComplete)), []string{
		"sales.order_line.quantity_positive: rows satisfied",
		"sales.order.lines.line_number_unique: relation satisfied",
		"sales.order_line.line_number_positive: rows satisfied",
	})
	sameAnswers(t, "an order without lines", answers(t, world, readOrderLines(t, world, runner, 9999, validationeval.CoverageComplete)), []string{
		"sales.order_line.quantity_positive: rows ",
		"sales.order.lines.line_number_unique: relation satisfied",
		"sales.order_line.line_number_positive: rows ",
	})
}

// The reference Workspace has no date and no empty value in its order lines.
// The membership rows of sqlite_path_test.go are read from PostgreSQL here,
// with the column types a PostgreSQL schema would use: date, numeric, double
// precision.
const membershipPostgresSchema = `
create table public.membership (
  department_id text not null,
  person_id     text,
  seat_number   integer,
  allocation    numeric(5,2),
  load_factor   double precision,
  starts_on     date,
  ends_on       date,
  registered_at timestamptz
);
insert into public.membership values
  ('D1', 'P001', 1, 1,    1.0, '2026-04-01', '2026-09-30', '2026-03-31 15:00:00+00'),
  ('D1', 'P001', 2, 0.5,  0.5, '2026-10-01', null,         null),
  ('D1', 'P002', 3, 0,    0.0, '2026-04-01', null,         null),
  ('D1', 'P003', 4, null, null, '2026-05-01', '2026-04-30', null),
  ('D2', 'P010', 1, 1,    1.0, '2026-04-01', '2026-09-30', null),
  ('D2', 'P010', 1, 1,    1.0, '2026-09-30', null,         null);
`

func postgresMembers(db *sql.DB) sqlquery.Executor {
	build := func(name, selectList, tail string, columns ...query.ResultColumnMapping) realization.ContractQuerySQLRealization {
		return realization.ContractQuerySQLRealization{
			Target:        membersTarget(name),
			Provider:      environment.ProviderPostgres,
			Source:        "select " + selectList + " from public.membership where department_id = :department_id order by seat_number, starts_on" + tail,
			Inputs:        []realization.ContractQueryInputParameterMapping{{Input: "department", Parameter: "department_id"}},
			ResultColumns: columns,
		}
	}
	return sqlquery.Executor{
		Provider: environment.ProviderPostgres,
		Runner:   postgresquery.Runner{DB: db},
		Realizations: []realization.ContractQuerySQLRealization{
			build("members", "person_id, seat_number, allocation, starts_on, ends_on", "",
				columnPerson, columnSeat, columnShare, columnStart, columnEnd),
			build("members_narrow", "person_id, seat_number, starts_on", "", columnPerson, columnSeat, columnStart),
			build("members_first_page", "person_id, seat_number, allocation, starts_on, ends_on", " limit 2",
				columnPerson, columnSeat, columnShare, columnStart, columnEnd),
			build("members_float", "person_id, load_factor", "", columnPerson,
				query.ResultColumnMapping{Column: "load_factor", Attribute: allocation}),
			// A moment in time put where a calendar date is declared.
			build("members_moment", "person_id, registered_at", "", columnPerson,
				query.ResultColumnMapping{Column: "registered_at", Attribute: startsOn}),
		},
	}
}

func readPostgresMembers(t *testing.T, executor sqlquery.Executor, queryName, department string, coverage validationeval.Coverage) (*validationread.Detail, driver.ContractQueryResult) {
	t.Helper()
	contract := membersContract(queryName)
	result, err := executor.ExecuteContractQuery(context.Background(), membersTarget(queryName), contract, map[string]any{"department": department})
	if err != nil {
		t.Fatalf("the Contract Query did not run: %v", err)
	}
	detail, err := validationread.NewDetail(testSemantics, membersRelation, membershipData, result.Rows, coverage)
	if err != nil {
		t.Fatal(err)
	}
	return detail, result
}

func TestDatesAndEmptyValuesAreReadFromPostgreSQL(t *testing.T) {
	db := openValidationReadPostgres(t)
	// The session zone is set away from UTC on purpose: a date must stay the
	// day that was stored, whatever zone the connection is in. The driver
	// hands a date over as midnight UTC.
	if _, err := db.Exec("set time zone 'Asia/Tokyo'"); err != nil {
		t.Fatal(err)
	}
	createTables(t, db, membershipPostgresSchema, "public.membership")
	executor := postgresMembers(db)

	d1, raw := readPostgresMembers(t, executor, "members", "D1", validationeval.CoverageComplete)
	first := raw.Rows[0].Attributes
	if day, isTime := first[startsOn].(time.Time); !isTime || day.Format(time.RFC3339) != "2026-04-01T00:00:00Z" {
		t.Fatalf("a date column arrived as %T %v", first[startsOn], first[startsOn])
	}
	if kind := fmt.Sprintf("%T %T %T", first[seatID], first[allocation], first[personID]); kind != "int64 string string" {
		t.Fatalf("integer, numeric, text arrived as %s", kind)
	}
	if value, held := raw.Rows[3].Attributes[allocation]; !held || value != nil {
		t.Fatalf("NULL arrived as held=%v value=%#v", held, value)
	}

	// The same answers as from SQLite: the rows are the same rows.
	sameAnswers(t, "share is positive, per row", perRow(t, allocationPositive, d1), []string{
		"satisfied", "satisfied", "not-satisfied", "undecided: value-null " + string(allocation),
	})
	sameAnswers(t, "start is not after end if present, per row", perRow(t, periodInOrder, d1), []string{
		"satisfied", "satisfied", "satisfied", "not-satisfied",
	})
	sameAnswers(t, "rules over all the rows of D1", []string{
		answer(t, personUnique, d1), answer(t, seatUnique, d1), answer(t, periodsApart, d1),
	}, []string{"not-satisfied", "satisfied", "satisfied"})

	// P010 ends on 2026-09-30 and starts again on 2026-09-30. Read as dates,
	// the two periods share that day. A reader that moved the dates to
	// another zone would see 09-29 and 09-30 and miss it.
	d2, _ := readPostgresMembers(t, executor, "members", "D2", validationeval.CoverageComplete)
	sameAnswers(t, "rules over all the rows of D2", []string{
		answer(t, personUnique, d2), answer(t, seatUnique, d2), answer(t, periodsApart, d2),
	}, []string{"not-satisfied", "not-satisfied", "not-satisfied"})

	narrow, _ := readPostgresMembers(t, executor, "members_narrow", "D1", validationeval.CoverageComplete)
	sameAnswers(t, "a query without the end", []string{
		perRow(t, periodInOrder, narrow)[1], answer(t, periodsApart, narrow),
	}, []string{"undecided: value-not-fetched " + string(endsOn), "undecided: value-not-fetched " + string(endsOn)})

	page, _ := readPostgresMembers(t, executor, "members_first_page", "D1", validationeval.CoveragePartial)
	sameAnswers(t, "the first page", []string{answer(t, personUnique, page), answer(t, seatUnique, page)},
		[]string{"not-satisfied", "undecided: relation-partially-read " + string(seatID)})

	// double precision is refused, as REAL is from SQLite.
	float, _ := readPostgresMembers(t, executor, "members_float", "D1", validationeval.CoverageComplete)
	one, _ := float.Row(1)
	if _, err := validationeval.Decide(allocationPositive, one, validationeval.Operators{}); err == nil || !strings.Contains(err.Error(), "binary floating point") {
		t.Fatalf("err=%v", err)
	}

	// A timestamptz where a calendar date is declared. The driver hands a
	// timestamptz over in the zone of this process, whatever the zone of the
	// session, and a date as midnight UTC. The stored moment 2026-03-31
	// 15:00 UTC is therefore a midnight on a machine nine hours ahead and a
	// time of day on a machine in UTC: the reader takes the first as
	// 2026-04-01 and refuses the second. The same row, two answers. The
	// reader cannot see which column type the value came from.
	processZone := time.Local
	t.Cleanup(func() { time.Local = processZone })

	time.Local = time.FixedZone("nine hours ahead", 9*60*60)
	moment, rawMoment := readPostgresMembers(t, executor, "members_moment", "D1", validationeval.CoverageComplete)
	handed, _ := rawMoment.Rows[0].Attributes[startsOn].(time.Time)
	if handed.Format(time.RFC3339) != "2026-04-01T00:00:00+09:00" {
		t.Fatalf("timestamptz arrived as %s", handed.Format(time.RFC3339))
	}
	row, _ := moment.Row(0)
	if reading, err := row.ReadDate(startsOn); err != nil || reading.Value.String() != "2026-04-01" {
		t.Fatalf("a midnight was not read as its own day: %+v err=%v", reading, err)
	}

	time.Local = time.UTC
	moment, _ = readPostgresMembers(t, executor, "members_moment", "D1", validationeval.CoverageComplete)
	row, _ = moment.Row(0)
	if reading, err := row.ReadDate(startsOn); err == nil || !strings.Contains(err.Error(), "time of day") {
		t.Fatalf("a moment with a time of day was read as a date: %+v err=%v", reading, err)
	}
}

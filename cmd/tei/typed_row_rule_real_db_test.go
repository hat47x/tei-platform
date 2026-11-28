package main

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// A rule whose operator reads every row of a relation in a type other than
// number has to be decided from rows that actually came from PostgreSQL, and the
// product entry point `tei view -environment` has to report the answer in
// perRelation.
//
// #6043 added the row-scoped typed reader and #6052 shipped tei.day.every-not-before,
// but no test drove such a rule through the shipped CLI against a database: the
// real-PostgreSQL checks in cmd/tei all use number rules, and the typed ones
// decide one record at a time.
//
// The expected answers are written from testdata/destination-order-change/seed.sql
// and from "every line was promised on or after the order's deadline", not read
// back from the CLI:
//
//	order 1001: promised 2026-04-01, 2026-04-02, 2026-03-01 (deadline 2026-04-01)
//	            2026-03-01 is before the deadline, so the rule does not hold
//	order 1002: promised 2026-04-10, 2026-04-11 (deadline 2026-04-10)
//	            every day is on or after the deadline, so the rule holds
//
// The column is a separate table because the reference Workspace declares no
// std.date attribute on sales.order_line: a typed column has to be read from an
// attribute the Workspace already declares, so the promised day is stored beside
// the line and the reader asks the relation for it.

const typedRowDatabase = "tei_destination_order_e2e"

const typedRowDocument = "```tei\n" +
	"namespace sales\n" +
	"\n" +
	"operator sales.every_line_not_before:\n" +
	"  takes value: date of each row\n" +
	"  takes deadline: date of each row\n" +
	"  refutes on some rows\n" +
	"\n" +
	"rule sales.order.lines.promised_not_before_deadline:\n" +
	"  sales.order_line.promised_on is sales.every_line_not_before deadline: sales.order_line.deadline_on per sales.order.lines\n" +
	"```\n"

type typedRowReport struct {
	OK     bool             `json:"ok"`
	Result *typedRowResult  `json:"result"`
	Error  *typedRowFailure `json:"error"`
}

type typedRowResult struct {
	Coverage             string            `json:"coverage"`
	PerRelation          map[string]string `json:"perRelation"`
	PerRelationUndecided map[string]string `json:"perRelationUndecided"`
}

type typedRowFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// typedRowFixture creates the two attributes the rule reads and seeds their
// days from the seed's order lines. The days themselves are written here, by
// hand, so the expected answers below are not read back from the database.
func typedRowFixture(t *testing.T) (binary, dsn string) {
	t.Helper()
	dsn = os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL typed row rule not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != typedRowDatabase {
		t.Fatalf("refusing to modify the database without the disposable-database confirmation %q", typedRowDatabase)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal("cannot initialize the isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != typedRowDatabase {
		t.Fatalf("expected %q, got %q: %v", typedRowDatabase, actual, err)
	}
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null,
		promised_on varchar(10),
		deadline_on varchar(10)
	)`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
			t.Errorf("cannot clean up disposable order lines: %v", err)
		}
	})
	seed, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatalf("cannot seed order lines: %v", err)
	}
	// The days each line was promised on, and the deadline of the line's order.
	// They are the values the expected answers below are written from.
	if _, err := db.Exec(`update public.order_lines set
		promised_on = case order_line_id
			when 1 then '2026-04-01'
			when 2 then '2026-04-02'
			when 3 then '2026-03-01'
			when 4 then '2026-04-10'
			when 5 then '2026-04-11'
		end,
		deadline_on = case order_id
			when 1001 then '2026-04-01'
			when 1002 then '2026-04-10'
		end`); err != nil {
		t.Fatal(err)
	}
	return buildCLIForLifecycle(t), dsn
}

// typedRowWorkspace copies the reference Workspace, adds the two day attributes
// the rule reads, the operator and rule, and selects the shipped implementation.
func typedRowWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)

	manifest := filepath.Join(root, "tei.workspace.yaml")
	content, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "languages:") {
		t.Fatal("the reference Workspace already selects a language")
	}
	additions := "languages:\n  - tei.authoring/v1\n" +
		"implementations:\n  - operator: sales.every_line_not_before\n    use: tei.day.every-not-before\n"
	if err := os.WriteFile(manifest, append(content, []byte(additions)...), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "zz-promised-not-before.tei.md"),
		[]byte(typedRowDocument), 0o644); err != nil {
		t.Fatal(err)
	}
	// The two day attributes the rule reads. They are declared here, beside the
	// rule, so the reference Workspace itself is untouched.
	orderLine := filepath.Join(root, "semantic", "sales", "order.tei")
	content, err = os.ReadFile(orderLine)
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    attribute unitPrice type std.money\n"
	if strings.Count(string(content), anchor) != 1 {
		t.Fatalf("order.tei does not hold %q exactly once", anchor)
	}
	edited := strings.Replace(string(content), anchor,
		anchor+"    attribute promisedOn type std.date\n    attribute deadlineOn type std.date\n", 1)
	if err := os.WriteFile(orderLine, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	// The reader asks the relation for the two attributes, so the persistence
	// mapping has to name the columns that hold them.
	persistence := filepath.Join(root, "databases", "sales-core", "persistence", "order-line.tei")
	content, err = os.ReadFile(persistence)
	if err != nil {
		t.Fatal(err)
	}
	const columnAnchor = "    column sales.order_line.unit_price -> unit_price storage decimal required\n"
	if strings.Count(string(content), columnAnchor) != 1 {
		t.Fatalf("the persistence mapping does not hold %q exactly once", columnAnchor)
	}
	edited = strings.Replace(string(content), columnAnchor,
		columnAnchor+"    column sales.order_line.promised_on -> promised_on storage varchar optional\n"+
			"    column sales.order_line.deadline_on -> deadline_on storage varchar optional\n", 1)
	if err := os.WriteFile(persistence, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	// The reader reads the two days from the Contract Query's result, so the
	// query has to return them: the columns are added to the SQL and to the
	// result block that names which attribute each column is.
	sqlFile := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
	content, err = os.ReadFile(sqlFile)
	if err != nil {
		t.Fatal(err)
	}
	const selectAnchor = "    unit_price\n"
	if strings.Count(string(content), selectAnchor) != 1 {
		t.Fatalf("the query does not select %q exactly once", selectAnchor)
	}
	editedSQL := strings.Replace(string(content), selectAnchor,
		"    unit_price,\n    promised_on,\n    deadline_on\n", 1)
	if err := os.WriteFile(sqlFile, []byte(editedSQL), 0o644); err != nil {
		t.Fatal(err)
	}
	realization := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.tei")
	content, err = os.ReadFile(realization)
	if err != nil {
		t.Fatal(err)
	}
	const resultAnchor = "        column unit_price -> sales.order_line.unit_price\n"
	if strings.Count(string(content), resultAnchor) != 1 {
		t.Fatalf("the query realization does not name %q exactly once", resultAnchor)
	}
	editedRealization := strings.Replace(string(content), resultAnchor,
		resultAnchor+
			"        column promised_on -> sales.order_line.promised_on\n"+
			"        column deadline_on -> sales.order_line.deadline_on\n", 1)
	if err := os.WriteFile(realization, []byte(editedRealization), 0o644); err != nil {
		t.Fatal(err)
	}
	application := filepath.Join(root, "applications", "order-entry", "application.tei")
	content, err = os.ReadFile(application)
	if err != nil {
		t.Fatal(err)
	}
	const appAnchor = "    uses validation sales.order_line.quantity_positive\n"
	if strings.Count(string(content), appAnchor) != 1 {
		t.Fatalf("application.tei does not hold %q exactly once", appAnchor)
	}
	edited = strings.Replace(string(content), appAnchor,
		appAnchor+"    uses validation sales.order.lines.promised_not_before_deadline\n", 1)
	if err := os.WriteFile(application, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func readTypedRowRule(t *testing.T, binary, root, orderID string) (typedRowReport, cliProcessStreams) {
	t.Helper()
	got := runCLIProcessWithStreams(t, binary,
		"view", "-workspace", root, "-application", "order_entry",
		"-pattern", "order_entry.order_lines",
		"-environment", "development", "-input", "order_id="+orderID,
		"-format", "json",
	)
	var report typedRowReport
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
		t.Fatalf("invalid CLI JSON: %v (%s)", err, got.stdout)
	}
	return report, got
}

// plan/NOW.md road item 5's next product change: a rule that reads a column of
// days is answered for rows that came from PostgreSQL, in perRelation, and the
// answer matches the days written in the seed rather than the CLI's own output.
func TestATypedRowRuleIsAnsweredPerRelationFromPostgres(t *testing.T) {
	binary, dsn := typedRowFixture(t)
	root := typedRowWorkspace(t)
	if validated := runCLIProcessWithStreams(t, binary, "validate", "-workspace", root); validated.code != 0 {
		t.Fatalf("the Workspace did not validate: code=%d stdout=%s stderr=%s",
			validated.code, validated.stdout, validated.stderr)
	}

	for _, tc := range []struct {
		order string
		want  string
		why   string
	}{
		{
			order: "1001",
			want:  "not-satisfied",
			why:   "one line was promised on 2026-03-01, before the deadline 2026-04-01",
		},
		{
			order: "1002",
			want:  "satisfied",
			why:   "every line was promised on or after the deadline 2026-04-10",
		},
	} {
		t.Run("order "+tc.order, func(t *testing.T) {
			report, got := readTypedRowRule(t, binary, root, tc.order)
			if got.code != 0 || !report.OK || report.Error != nil {
				t.Fatalf("CLI failed: code=%d stdout=%s stderr=%s", got.code, got.stdout, got.stderr)
			}
			if strings.Contains(got.stdout, dsn) {
				t.Fatal("the database connection string appeared in the JSON output")
			}
			read := report.Result
			if read == nil || read.Coverage != "complete" {
				t.Fatalf("the read did not cover the relation: %+v", read)
			}
			answer, present := read.PerRelation["sales.order.lines.promised_not_before_deadline"]
			if !present {
				t.Fatalf("the typed row rule is absent from perRelation: %+v", read.PerRelation)
			}
			if answer != tc.want {
				t.Fatalf("order=%s perRelation=%q (undecided=%v), independently expected %q (%s)", tc.order, answer, read.PerRelationUndecided, tc.want, tc.why)
			}
			if len(read.PerRelationUndecided) != 0 {
				t.Fatalf("undecided relation answers where the rows hold days: %v", read.PerRelationUndecided)
			}
		})
	}
}

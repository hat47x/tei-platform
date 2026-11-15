package main

// Real-PostgreSQL verification of the reference browse-read entry.
//
// It runs only when a disposable database is explicitly confirmed:
//
//	TEI_BROWSE_READ_PG_DSN                 owner DSN of an EMPTY disposable database
//	TEI_BROWSE_READ_PG_CONFIRM_ISOLATED    must be tei_browse_read_e2e (and equal current_database())
//	TEI_BROWSE_READ_PG_OTHER_DSN           optional: a second empty disposable database (same server)
//	TEI_BROWSE_READ_PG_READER_DSN          optional: a login role named tei_browse_reader, created here
//
// Skipped parts are NOT RUN, not passed. Expected values below are written by hand from
// examples/data-source-workbench (PQ-01..PQ-04) and from an independent reader that
// shares no SQL or code with the Query under test; nothing is generated from the CLI output.

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"tei/tools/internal/postgresdb"
)

const confirmName = "tei_browse_read_e2e"

type cliCell struct {
	State string `json:"state"`
	Value any    `json:"value"`
}

type cliResponse struct {
	OK    bool `json:"ok"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Context *struct {
		Source struct {
			SHA string `json:"querySourceSha256"`
		} `json:"source"`
		Binding struct {
			Instance  string `json:"instance"`
			Assertion string `json:"assertion"`
		} `json:"binding"`
		Connection struct {
			Database string `json:"database"`
			User     string `json:"user"`
			ReadOnly bool   `json:"readOnly"`
			Snapshot string `json:"snapshot"`
		} `json:"connection"`
		Parameters []struct {
			Name            string `json:"name"`
			DeclaredByQuery bool   `json:"declaredByQuery"`
		} `json:"parameters"`
		Rowset struct {
			RowCount int `json:"rowCount"`
		} `json:"rowset"`
		Rows  [][]cliCell `json:"rows"`
		Grain struct {
			Identity string `json:"identity"`
			Dup      int    `json:"exactDuplicateRows"`
		} `json:"grain"`
		Coverage struct {
			Status         string `json:"status"`
			EmptyJustified bool   `json:"emptyJustified"`
			Visibility     string `json:"visibility"`
		} `json:"coverage"`
		NullOrigin string `json:"nullOrigin"`
	} `json:"context"`
}

type env struct {
	root, bin, dsn, otherDSN, readerDSN string
}

func repoRoot(t *testing.T) string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func (e env) cli(t *testing.T, dsn, workspace, environment string, extra ...string) (cliResponse, string) {
	t.Helper()
	args := []string{"-workspace", workspace, "-environment", environment, "-application", "order_entry",
		"-pattern", "order_entry.recent_orders", "-dsn-env", "RBR_DSN", "-confirm-isolated", confirmName}
	args = append(args, extra...)
	cmd := exec.Command(e.bin, args...)
	cmd.Env = append(os.Environ(), "RBR_DSN="+dsn)
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run() // a refusal exits 1 and still prints the envelope
	var r cliResponse
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("CLI output is not JSON: %v\n%s", err, out.String())
	}
	if dsn != "" && strings.Contains(out.String(), dsn) {
		t.Fatal("the DSN appeared in the CLI output")
	}
	return r, out.String()
}

func from(s string) []string { return []string{"-param", "from=time:" + s} }

func rowsOf(r cliResponse) [][]string {
	var out [][]string
	for _, row := range r.Context.Rows {
		var cells []string
		for _, c := range row {
			if c.State == "null" {
				cells = append(cells, "<NULL>")
			} else {
				cells = append(cells, fmt.Sprint(c.Value))
			}
		}
		out = append(out, cells)
	}
	return out
}

// independentRead reads the whole relation with its own SQL and filters and sorts in Go.
func independentRead(t *testing.T, db *sql.DB, since time.Time) [][]string {
	t.Helper()
	rows, err := db.Query(`SELECT order_id, customer_id, ordered_at, status, COALESCE(total_amount::text, '<NULL>') FROM public.orders`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	type rec struct {
		at   time.Time
		cols []string
	}
	var all []rec
	for rows.Next() {
		var id, cust, status, amount string
		var at time.Time
		if err := rows.Scan(&id, &cust, &at, &status, &amount); err != nil {
			t.Fatal(err)
		}
		if !at.Before(since) {
			all = append(all, rec{at, []string{id, cust, at.UTC().Format(time.RFC3339Nano), status, amount}})
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].at.After(all[j].at) })
	var out [][]string
	for _, r := range all {
		out = append(out, r.cols)
	}
	return out
}

func mustExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(p)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestBrowseReadAgainstDisposablePostgreSQL(t *testing.T) {
	dsn := os.Getenv("TEI_BROWSE_READ_PG_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL browse read NOT RUN: TEI_BROWSE_READ_PG_DSN unset")
	}
	if os.Getenv("TEI_BROWSE_READ_PG_CONFIRM_ISOLATED") != confirmName {
		t.Fatal("refusing to create fixtures without explicit disposable database confirmation")
	}
	e := env{root: repoRoot(t), dsn: dsn, otherDSN: os.Getenv("TEI_BROWSE_READ_PG_OTHER_DSN"), readerDSN: os.Getenv("TEI_BROWSE_READ_PG_READER_DSN")}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	var relations int
	if err := db.QueryRow(`SELECT current_database()`).Scan(&name); err != nil || name != confirmName {
		t.Fatalf("refusing database %q", name)
	}
	if err := db.QueryRow(`SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE c.relkind IN ('r','p','v','m','S','f') AND n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%'`).Scan(&relations); err != nil || relations != 0 {
		t.Fatalf("refusing a database that already has %d user relations (%v)", relations, err)
	}

	e.bin = filepath.Join(t.TempDir(), "reference-browse-read")
	build := exec.Command("go", "build", "-o", e.bin, "./tools/reference-browse-read")
	build.Dir = e.root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	seed, err := os.ReadFile(filepath.Join(e.root, "examples", "data-source-workbench", "recent_orders_product_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, db, string(seed))
	ws := filepath.Join(e.root, "reference", "workspace")
	const dev = "development"

	t.Run("A normal read matches the hand-fixed rows and the independent reader", func(t *testing.T) {
		r, _ := e.cli(t, dsn, ws, dev, from("2026-11-07T09:00:00Z")...)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		want := [][]string{
			{"O-003", "C-202", "2026-11-07T10:10:00Z", "accepted", "5.25"},
			{"O-002", "C-101", "2026-11-07T10:05:00Z", "pending", "0.00"},
			{"O-001", "C-101", "2026-11-07T10:00:00Z", "accepted", "120.50"},
		}
		if got := rowsOf(r); !reflect.DeepEqual(got, want) {
			t.Fatalf("CLI rows\n got %v\nwant %v", got, want)
		}
		if got := independentRead(t, db, time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)); !reflect.DeepEqual(rowsOf(r), got) {
			t.Fatalf("CLI rows differ from the independent reader\n cli %v\n ind %v", rowsOf(r), got)
		}
		c := r.Context
		if c.Binding.Instance != "sales_core.dev" || c.Binding.Assertion != "host-asserted" ||
			c.Connection.Database != confirmName || !c.Connection.ReadOnly || c.Connection.Snapshot == "" {
			t.Fatalf("binding/connection=%+v %+v", c.Binding, c.Connection)
		}
		if len(c.Parameters) != 1 || c.Parameters[0].Name != "from" || !c.Parameters[0].DeclaredByQuery {
			t.Fatalf("the declared :from input must be reflected in the observed Context: %+v", c.Parameters)
		}
		if c.Coverage.Status != "unestablished" || c.Coverage.EmptyJustified || c.Coverage.Visibility != "unobserved" || c.Grain.Identity != "undeclared" {
			t.Fatalf("claims raised above the ceiling: %+v %+v", c.Coverage, c.Grain)
		}
	})

	t.Run("B boundary is inclusive and one microsecond later excludes", func(t *testing.T) {
		for _, tc := range []struct {
			from string
			ids  []string
		}{
			{"2026-11-07T10:05:00Z", []string{"O-003", "O-002"}},
			{"2026-11-07T10:05:00.000001Z", []string{"O-003"}},
			{"2026-11-07T10:10:00Z", []string{"O-003"}},
		} {
			r, _ := e.cli(t, dsn, ws, dev, from(tc.from)...)
			if !r.OK {
				t.Fatalf("%s: %+v", tc.from, r.Error)
			}
			var ids []string
			for _, row := range rowsOf(r) {
				ids = append(ids, row[0])
			}
			if !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("from=%s ids=%v want %v", tc.from, ids, tc.ids)
			}
		}
	})

	t.Run("C empty result never becomes absence", func(t *testing.T) {
		r, _ := e.cli(t, dsn, ws, dev, from("2026-11-07T10:10:01Z")...)
		if !r.OK || r.Context.Rowset.RowCount != 0 {
			t.Fatalf("%+v %+v", r.Error, r.Context)
		}
		if r.Context.Coverage.Status != "unestablished" || r.Context.Coverage.EmptyJustified {
			t.Fatalf("0 rows raised the ceiling: %+v", r.Context.Coverage)
		}
		if e.otherDSN == "" {
			t.Log("R3/R4a other-database counterexample NOT RUN: TEI_BROWSE_READ_PG_OTHER_DSN unset")
			return
		}
		odb, err := postgresdb.OpenSQL(e.otherDSN)
		if err != nil {
			t.Fatal(err)
		}
		defer odb.Close()
		var oname string
		if err := odb.QueryRow(`SELECT current_database()`).Scan(&oname); err != nil || oname == confirmName || oname == "postgres" {
			t.Fatalf("the second database must be another disposable database: %q", oname)
		}
		mustExec(t, odb, string(seed))
		mustExec(t, odb, `DELETE FROM public.orders; INSERT INTO public.orders VALUES ('T2-001','C-900','2026-11-07T10:20:00Z','accepted',999.00)`)
		// Same workspace, same Query bytes, same Environment/Instance, same parameter.
		other := cliWithConfirm(t, e, e.otherDSN, oname, ws, dev, from("2026-11-07T10:10:01Z")...)
		if !other.OK || other.Context.Rowset.RowCount != 1 {
			t.Fatalf("%+v", other)
		}
		if other.Context.Source.SHA != r.Context.Source.SHA || other.Context.Binding.Instance != r.Context.Binding.Instance {
			t.Fatal("the premise failed: Source digest and Binding must be identical")
		}
		if other.Context.Connection.Database == r.Context.Connection.Database {
			t.Fatal("the Context must expose that a different database was read")
		}
	})

	t.Run("D refusals happen before any read", func(t *testing.T) {
		code := func(r cliResponse) string {
			if r.OK || r.Error == nil {
				return "OK"
			}
			return r.Error.Code
		}
		cases := []struct {
			name, dsn, workspace, environment string
			extra                             []string
			want                              string
		}{
			{"unknown environment, DSN variable empty", "", ws, "staging", from("2026-11-07T09:00:00Z"), "BINDING_REFUSED"},
			{"workspace without environments", dsn, filepath.Join(e.root, "examples", "xtea-lowcode"), dev, nil, "BINDING_REFUSED"},
			{"missing parameter", dsn, ws, dev, nil, "READ_FAILED"},
			{"extra parameter", dsn, ws, dev, append(from("2026-11-07T09:00:00Z"), "-param", "status=text:accepted"), "READ_FAILED"},
			{"untagged parameter", dsn, ws, dev, []string{"-param", "from=2026-11-07T09:00:00Z"}, "INVALID_PARAMETER"},
			{"integer for a timestamp", dsn, ws, dev, []string{"-param", "from=int:42"}, "READ_FAILED"},
		}
		for _, tc := range cases {
			r, _ := e.cli(t, tc.dsn, tc.workspace, tc.environment, tc.extra...)
			if got := code(r); got != tc.want {
				t.Errorf("%s: code=%s want %s (%+v)", tc.name, got, tc.want, r.Error)
			}
		}
		wrong := cliWithConfirm(t, e, dsn, "some_other_database", ws, dev, from("2026-11-07T09:00:00Z")...)
		if code(wrong) != "DATABASE_NOT_CONFIRMED" {
			t.Errorf("a wrong -confirm-isolated was accepted: %+v", wrong.Error)
		}
	})

	t.Run("E NULL is null; a missing column is refused, not nulled", func(t *testing.T) {
		mustExec(t, db, `ALTER TABLE public.orders ALTER COLUMN total_amount DROP NOT NULL; UPDATE public.orders SET total_amount = NULL WHERE order_id='O-002'`)
		r, _ := e.cli(t, dsn, ws, dev, from("2026-11-07T10:05:00Z")...)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		if got := rowsOf(r); got[1][0] != "O-002" || got[1][4] != "<NULL>" || got[0][4] != "5.25" {
			t.Fatalf("rows=%v", got)
		}
		if !reflect.DeepEqual(rowsOf(r), independentRead(t, db, time.Date(2026, 11, 7, 10, 5, 0, 0, time.UTC))) {
			t.Fatal("NULL rendering differs from the independent reader")
		}
		if r.Context.NullOrigin != "unestablished" {
			t.Fatalf("nullOrigin=%s", r.Context.NullOrigin)
		}
		mustExec(t, db, `ALTER TABLE public.orders DROP COLUMN total_amount`)
		missing, _ := e.cli(t, dsn, ws, dev, from("2026-11-07T09:00:00Z")...)
		if missing.OK || missing.Error.Code != "READ_FAILED" || !strings.Contains(missing.Error.Message, "42703") {
			t.Fatalf("schema mismatch must stop the read: %+v", missing.Error)
		}
		mustExec(t, db, `ALTER TABLE public.orders ADD COLUMN total_amount NUMERIC(12,2); UPDATE public.orders SET total_amount = 0.00 WHERE order_id='O-002'; UPDATE public.orders SET total_amount = 120.50 WHERE order_id='O-001'; UPDATE public.orders SET total_amount = 5.25 WHERE order_id='O-003'`)
	})

	t.Run("G changed Source bytes change the digest and the Rowset under the same Query ID", func(t *testing.T) {
		before, _ := e.cli(t, dsn, ws, dev, from("2026-11-07T09:00:00Z")...)
		mustExec(t, db, `CREATE TABLE public.order_lines(order_id text, line_no int, PRIMARY KEY(order_id,line_no));
			INSERT INTO public.order_lines VALUES ('O-001',1),('O-001',2),('O-003',1)`)
		copied := filepath.Join(t.TempDir(), "workspace")
		copyTree(t, ws, copied)
		sqlPath := filepath.Join(copied, "applications", "order-entry", "queries", "recent_orders.sql")
		join := "SELECT o.order_id, o.customer_id, o.ordered_at, o.status, o.total_amount\nFROM public.orders AS o JOIN public.order_lines AS l ON l.order_id = o.order_id\nWHERE o.ordered_at >= :from ORDER BY o.ordered_at DESC"
		if err := os.WriteFile(sqlPath, []byte(join), 0o644); err != nil {
			t.Fatal(err)
		}
		after, _ := e.cli(t, dsn, copied, dev, from("2026-11-07T09:00:00Z")...)
		if !after.OK {
			t.Fatalf("%+v", after.Error)
		}
		if after.Context.Source.SHA == before.Context.Source.SHA {
			t.Fatal("different SQL bytes produced the same digest")
		}
		// R5: independent expectation written by hand from the line seed: O-001 has 2 lines,
		// O-002 none, O-003 one -> 3 joined rows, one exact duplicate, O-002 absent.
		var ids []string
		for _, row := range rowsOf(after) {
			ids = append(ids, row[0])
		}
		if !reflect.DeepEqual(ids, []string{"O-003", "O-001", "O-001"}) {
			t.Fatalf("joined ids=%v", ids)
		}
		if after.Context.Grain.Dup != 1 || after.Context.Grain.Identity != "undeclared" {
			t.Fatalf("multiplicity not reported: %+v", after.Context.Grain)
		}
		var joined int
		if err := db.QueryRow(`SELECT count(*) FROM public.order_lines`).Scan(&joined); err != nil || joined != 3 {
			t.Fatalf("independent line count=%d (%v)", joined, err)
		}
		if after.Context.Coverage.Status != "unestablished" {
			t.Fatal("coverage raised")
		}
	})

	if e.readerDSN == "" {
		t.Log("F RLS / column privilege / masking view NOT RUN: TEI_BROWSE_READ_PG_READER_DSN unset")
		return
	}
	t.Run("F RLS and column privilege: the visible set is never a complete set", func(t *testing.T) {
		mustExec(t, db, `DROP TABLE public.order_lines`)
		mustExec(t, db, `DROP ROLE IF EXISTS tei_browse_reader; CREATE ROLE tei_browse_reader LOGIN PASSWORD 'tei_browse_reader';
			GRANT SELECT ON public.orders TO tei_browse_reader;
			ALTER TABLE public.orders ENABLE ROW LEVEL SECURITY;
			CREATE POLICY accepted_only ON public.orders FOR SELECT TO tei_browse_reader USING (status = 'accepted')`)
		r, _ := e.cli(t, e.readerDSN, ws, dev, from("2026-11-07T09:00:00Z")...)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		var ids []string
		for _, row := range rowsOf(r) {
			ids = append(ids, row[0])
		}
		if !reflect.DeepEqual(ids, []string{"O-003", "O-001"}) { // hand-fixed: only 'accepted' rows are visible
			t.Fatalf("visible ids=%v", ids)
		}
		if n := len(independentRead(t, db, time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC))); n != 3 {
			t.Fatalf("the owner sees %d rows, want 3", n)
		}
		if r.Context.Connection.User != "tei_browse_reader" || r.Context.Coverage.Visibility != "unobserved" ||
			r.Context.Coverage.Status != "unestablished" || r.Context.Coverage.EmptyJustified {
			t.Fatalf("a partial visible set was presented as more: %+v %+v", r.Context.Connection, r.Context.Coverage)
		}
		mustExec(t, db, `ALTER TABLE public.orders DISABLE ROW LEVEL SECURITY; REVOKE SELECT ON public.orders FROM tei_browse_reader;
			GRANT SELECT (order_id, customer_id, ordered_at, status) ON public.orders TO tei_browse_reader`)
		denied, _ := e.cli(t, e.readerDSN, ws, dev, from("2026-11-07T09:00:00Z")...)
		if denied.OK || denied.Error.Code != "READ_FAILED" || !strings.Contains(denied.Error.Message, "42501") {
			t.Fatalf("a missing column privilege must stop the read: %+v", denied.Error)
		}
		mustExec(t, db, `ALTER TABLE public.orders RENAME TO orders_base;
			CREATE VIEW public.orders AS SELECT order_id, customer_id, ordered_at, status, NULL::numeric(12,2) AS total_amount FROM public.orders_base;
			GRANT SELECT ON public.orders TO tei_browse_reader`)
		masked, _ := e.cli(t, e.readerDSN, ws, dev, from("2026-11-07T09:00:00Z")...)
		if !masked.OK {
			t.Fatalf("%+v", masked.Error)
		}
		for _, row := range rowsOf(masked) {
			if row[4] != "<NULL>" {
				t.Fatalf("masking view did not hide the amount: %v", row)
			}
		}
		// The false success this test pins: the Context cannot distinguish hidden from stored NULL.
		if masked.Context.NullOrigin != "unestablished" || masked.Context.Source.SHA == "" {
			t.Fatalf("null origin claimed: %+v", masked.Context)
		}
		mustExec(t, db, `DROP VIEW public.orders; ALTER TABLE public.orders_base RENAME TO orders; DROP OWNED BY tei_browse_reader; DROP ROLE tei_browse_reader`)
	})
}

func cliWithConfirm(t *testing.T, e env, dsn, confirm, workspace, environment string, extra ...string) cliResponse {
	t.Helper()
	args := []string{"-workspace", workspace, "-environment", environment, "-application", "order_entry",
		"-pattern", "order_entry.recent_orders", "-dsn-env", "RBR_DSN", "-confirm-isolated", confirm}
	args = append(args, extra...)
	cmd := exec.CommandContext(context.Background(), e.bin, args...)
	cmd.Env = append(os.Environ(), "RBR_DSN="+dsn)
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	var r cliResponse
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.String())
	}
	return r
}

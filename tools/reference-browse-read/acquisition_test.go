package main

// Real-PostgreSQL verification of the acquisition observation (W1-W3): what the Context says
// about coverage when the read was restricted in ways the plan does or does not show.
//
// It runs only against its OWN empty disposable database, so it does not share state with
// TestBrowseReadAgainstDisposablePostgreSQL:
//
//	TEI_BROWSE_READ_PG_ACQ_DSN             owner DSN of an EMPTY disposable database tei_browse_read_acq
//	TEI_BROWSE_READ_PG_ACQ_CONFIRM         must be tei_browse_read_acq (and equal current_database())
//	TEI_BROWSE_READ_PG_ACQ_READER_DSN      login role tei_acq_reader, created here (optional; RLS case)
//
// Unset means NOT RUN, not passed. Expected rows are written by hand from the 3-row seed and
// from an independent reader; nothing is generated from the CLI output.

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/tools/internal/postgresdb"
)

const acqName = "tei_browse_read_acq"

type acqResponse struct {
	OK    bool `json:"ok"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Context *struct {
		Rows  [][]cliCell `json:"rows"`
		Grain struct {
			Scan        string `json:"scan"`
			DeclaredKey string `json:"declaredKey"`
			Dup         int    `json:"exactDuplicateRows"`
		} `json:"grain"`
		Coverage struct {
			Status         string   `json:"status"`
			EmptyJustified bool     `json:"emptyJustified"`
			Visibility     string   `json:"visibility"`
			Reasons        []string `json:"reasons"`
			Open           []string `json:"open"`
		} `json:"coverage"`
		Acquisition struct {
			Observed bool `json:"observed"`
			Declared struct {
				Status string   `json:"status"`
				Tables []string `json:"tables"`
			} `json:"declaredRelations"`
			Observation struct {
				SystemIdentifier string   `json:"systemIdentifier"`
				LockedTables     []string `json:"lockedTables"`
				LockNote         string   `json:"lockNote"`
				PlanRelations    []string `json:"planRelations"`
				PlanNodes        []string `json:"planNodes"`
				PlanNote         string   `json:"planNote"`
			} `json:"observation"`
		} `json:"acquisition"`
	} `json:"context"`
}

func acqRun(t *testing.T, bin, dsn, workspace string) acqResponse {
	t.Helper()
	cmd := exec.Command(bin, "-workspace", workspace, "-environment", "development", "-application", "order_entry",
		"-pattern", "order_entry.recent_orders", "-dsn-env", "RBR_DSN", "-confirm-isolated", acqName,
		"-param", "from=time:2026-11-07T09:00:00Z")
	cmd.Env = append(os.Environ(), "RBR_DSN="+dsn)
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()
	var r acqResponse
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out.String())
	}
	return r
}

func acqIDs(r acqResponse) []string {
	var ids []string
	for _, row := range r.Context.Rows {
		ids = append(ids, row[0].Value.(string))
	}
	return ids
}

func acqHas(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func acqExec(t *testing.T, db *sql.DB, q string) {
	t.Helper()
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("%v\n%s", err, q)
	}
}

func acqCopy(t *testing.T, src, dst string) {
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

// acqWorkspaceWithSQL copies the Reference Workspace with a different recent_orders.sql.
func acqWorkspaceWithSQL(t *testing.T, root, sqlText string) string {
	t.Helper()
	copied := filepath.Join(t.TempDir(), "workspace")
	acqCopy(t, filepath.Join(root, "reference", "workspace"), copied)
	path := filepath.Join(copied, "applications", "order-entry", "queries", "recent_orders.sql")
	if err := os.WriteFile(path, []byte(sqlText), 0o644); err != nil {
		t.Fatal(err)
	}
	return copied
}

const acqSelect = "SELECT o.order_id, o.customer_id, o.ordered_at, o.status, o.total_amount\n"

func TestAcquisitionObservationAgainstDisposablePostgreSQL(t *testing.T) {
	dsn := os.Getenv("TEI_BROWSE_READ_PG_ACQ_DSN")
	if dsn == "" {
		t.Skip("acquisition observation NOT RUN: TEI_BROWSE_READ_PG_ACQ_DSN unset")
	}
	if os.Getenv("TEI_BROWSE_READ_PG_ACQ_CONFIRM") != acqName {
		t.Fatal("refusing to create fixtures without explicit disposable database confirmation")
	}
	readerDSN := os.Getenv("TEI_BROWSE_READ_PG_ACQ_READER_DSN")
	root := repoRoot(t)
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var name string
	var relations int
	if err := db.QueryRow(`SELECT current_database()`).Scan(&name); err != nil || name != acqName {
		t.Fatalf("refusing database %q", name)
	}
	if err := db.QueryRow(`SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
		WHERE c.relkind IN ('r','p','v','m','S','f') AND n.nspname NOT IN ('pg_catalog','information_schema') AND n.nspname NOT LIKE 'pg_toast%'`).Scan(&relations); err != nil || relations != 0 {
		t.Fatalf("refusing a database that already has %d user relations (%v)", relations, err)
	}
	bin := filepath.Join(t.TempDir(), "reference-browse-read")
	build := exec.Command("go", "build", "-o", bin, "./tools/reference-browse-read")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	seed, err := os.ReadFile(filepath.Join(root, "examples", "data-source-workbench", "recent_orders_product_seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	acqExec(t, db, string(seed))
	ws := filepath.Join(root, "reference", "workspace")
	ownerCount := func() int {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM public.orders`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("baseline: clean observations, coverage stays unestablished", func(t *testing.T) {
		r := acqRun(t, bin, dsn, ws)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		c, o := r.Context, r.Context.Acquisition.Observation
		if !c.Acquisition.Observed || c.Acquisition.Declared.Status != "resolved" || !reflect.DeepEqual(c.Acquisition.Declared.Tables, []string{"public.orders"}) {
			t.Fatalf("declared=%+v", c.Acquisition)
		}
		if !reflect.DeepEqual(o.LockedTables, []string{"public.orders"}) || o.LockNote != "" || o.SystemIdentifier == "" || o.PlanNote != "" {
			t.Fatalf("observation=%+v", o)
		}
		if !reflect.DeepEqual(o.PlanRelations, []string{"public.orders"}) || acqHas(o.PlanNodes, "Limit") {
			t.Fatalf("plan=%v %v", o.PlanRelations, o.PlanNodes)
		}
		if c.Coverage.Status != "unestablished" || len(c.Coverage.Reasons) != 0 || c.Coverage.EmptyJustified ||
			c.Coverage.Visibility != "rls-inactive" || !acqHas(c.Coverage.Open, "native-sql-predicate-not-closed") {
			t.Fatalf("coverage=%+v", c.Coverage)
		}
		if c.Grain.Scan != "single-relation" || c.Grain.DeclaredKey != "unique" {
			t.Fatalf("grain=%+v", c.Grain)
		}
		if !reflect.DeepEqual(acqIDs(r), []string{"O-003", "O-002", "O-001"}) {
			t.Fatalf("ids=%v", acqIDs(r))
		}
	})

	t.Run("PINNED LIMITATION: a function that drops rows leaves every observation clean", func(t *testing.T) {
		acqExec(t, db, `CREATE FUNCTION public.visible_orders() RETURNS SETOF public.orders LANGUAGE sql STABLE
			AS 'select * from public.orders where status = ''accepted'''`)
		defer acqExec(t, db, `DROP FUNCTION public.visible_orders()`)
		function := acqWorkspaceWithSQL(t, root, acqSelect+"FROM public.visible_orders() AS o\nWHERE o.ordered_at >= :from ORDER BY o.ordered_at DESC")
		r := acqRun(t, bin, dsn, function)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		// Hand-fixed: only the two 'accepted' rows come back although the table holds three.
		if got := acqIDs(r); !reflect.DeepEqual(got, []string{"O-003", "O-001"}) || ownerCount() != 3 {
			t.Fatalf("ids=%v table rows=%d", got, ownerCount())
		}
		c := r.Context
		if !reflect.DeepEqual(c.Acquisition.Observation.PlanRelations, []string{"public.orders"}) || c.Coverage.Visibility != "rls-inactive" {
			t.Fatalf("the observations were expected to stay clean: %+v", c.Acquisition.Observation)
		}
		// The ceiling is the whole point: clean observations must NOT read as complete.
		if c.Coverage.Status != "unestablished" || len(c.Coverage.Reasons) != 0 || c.Coverage.EmptyJustified ||
			!acqHas(c.Coverage.Open, "native-sql-predicate-not-closed") {
			t.Fatalf("an undetectable restriction was reported as more than unestablished: %+v", c.Coverage)
		}
	})

	t.Run("limit is seen and lowers coverage to partial", func(t *testing.T) {
		limited := acqWorkspaceWithSQL(t, root, acqSelect+"FROM public.orders AS o\nWHERE o.ordered_at >= :from ORDER BY o.ordered_at DESC LIMIT 2")
		r := acqRun(t, bin, dsn, limited)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		if got := acqIDs(r); !reflect.DeepEqual(got, []string{"O-003", "O-002"}) || ownerCount() != 3 {
			t.Fatalf("ids=%v table rows=%d", got, ownerCount())
		}
		if r.Context.Coverage.Status != "partial" || !acqHas(r.Context.Coverage.Reasons, "limit-in-plan") {
			t.Fatalf("coverage=%+v", r.Context.Coverage)
		}
	})

	t.Run("join is seen: another relation, joined scan, duplicated key", func(t *testing.T) {
		acqExec(t, db, `CREATE TABLE public.order_lines(order_id text, line_no int, PRIMARY KEY(order_id,line_no));
			INSERT INTO public.order_lines VALUES ('O-001',1),('O-001',2),('O-003',1)`)
		defer acqExec(t, db, `DROP TABLE public.order_lines`)
		joined := acqWorkspaceWithSQL(t, root, acqSelect+"FROM public.orders AS o JOIN public.order_lines AS l ON l.order_id = o.order_id\nWHERE o.ordered_at >= :from ORDER BY o.ordered_at DESC")
		r := acqRun(t, bin, dsn, joined)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		// Hand-fixed from the line seed: O-001 has 2 lines, O-002 none, O-003 one.
		if got := acqIDs(r); !reflect.DeepEqual(got, []string{"O-003", "O-001", "O-001"}) {
			t.Fatalf("ids=%v", got)
		}
		c := r.Context
		if c.Coverage.Status != "partial" || !acqHas(c.Coverage.Reasons, "relation-not-declared:public.order_lines") {
			t.Fatalf("coverage=%+v", c.Coverage)
		}
		if c.Grain.Scan != "multi-relation" || c.Grain.DeclaredKey != "duplicate" || c.Grain.Dup != 1 {
			t.Fatalf("grain=%+v", c.Grain)
		}
	})

	if readerDSN == "" {
		t.Log("RLS case NOT RUN: TEI_BROWSE_READ_PG_ACQ_READER_DSN unset")
	} else {
		t.Run("row level security is seen and lowers coverage to partial", func(t *testing.T) {
			acqExec(t, db, `DROP ROLE IF EXISTS tei_acq_reader; CREATE ROLE tei_acq_reader LOGIN PASSWORD 'tei_acq_reader';
				GRANT SELECT ON public.orders TO tei_acq_reader;
				ALTER TABLE public.orders ENABLE ROW LEVEL SECURITY;
				CREATE POLICY accepted_only ON public.orders FOR SELECT TO tei_acq_reader USING (status = 'accepted')`)
			defer acqExec(t, db, `DROP OWNED BY tei_acq_reader; DROP ROLE tei_acq_reader`)
			defer acqExec(t, db, `ALTER TABLE public.orders DISABLE ROW LEVEL SECURITY; DROP POLICY accepted_only ON public.orders`)
			r := acqRun(t, bin, readerDSN, ws)
			if !r.OK {
				t.Fatalf("%+v", r.Error)
			}
			if got := acqIDs(r); !reflect.DeepEqual(got, []string{"O-003", "O-001"}) || ownerCount() != 3 {
				t.Fatalf("ids=%v table rows=%d", got, ownerCount())
			}
			c := r.Context
			if c.Coverage.Status != "partial" || !acqHas(c.Coverage.Reasons, "row-security-active:public.orders") || c.Coverage.Visibility != "rls-active" {
				t.Fatalf("coverage=%+v", c.Coverage)
			}
			// The owner is not subject to the policy: the same SQL, a different Role, a clean report.
			owner := acqRun(t, bin, dsn, ws)
			if !owner.OK || owner.Context.Coverage.Visibility != "rls-inactive" || len(owner.Context.Rows) != 3 {
				t.Fatalf("owner=%+v", owner.Context.Coverage)
			}
		})
	}

	t.Run("a view under the declared name is seen", func(t *testing.T) {
		acqExec(t, db, `ALTER TABLE public.orders RENAME TO orders_base;
			CREATE VIEW public.orders AS SELECT order_id, customer_id, ordered_at, status, NULL::numeric(12,2) AS total_amount FROM public.orders_base`)
		defer acqExec(t, db, `DROP VIEW public.orders; ALTER TABLE public.orders_base RENAME TO orders`)
		r := acqRun(t, bin, dsn, ws)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		c := r.Context
		for _, want := range []string{"relation-not-declared:public.orders_base", "declared-relation-not-read:public.orders"} {
			if !acqHas(c.Coverage.Reasons, want) {
				t.Errorf("reasons lack %s: %v", want, c.Coverage.Reasons)
			}
		}
		if c.Coverage.Status != "partial" {
			t.Fatalf("coverage=%+v", c.Coverage)
		}
	})

	t.Run("a partitioned table under the declared name is seen", func(t *testing.T) {
		acqExec(t, db, `ALTER TABLE public.orders RENAME TO orders_old;
			CREATE TABLE public.orders (order_id text, customer_id text, ordered_at timestamptz, status text, total_amount numeric(12,2)) PARTITION BY RANGE (ordered_at);
			CREATE TABLE public.orders_p1 PARTITION OF public.orders FOR VALUES FROM ('2026-01-01') TO ('2026-11-07T10:06:00Z');
			CREATE TABLE public.orders_p2 PARTITION OF public.orders FOR VALUES FROM ('2026-11-07T10:06:00Z') TO ('2027-01-01');
			INSERT INTO public.orders SELECT * FROM public.orders_old`)
		defer acqExec(t, db, `DROP TABLE public.orders; ALTER TABLE public.orders_old RENAME TO orders`)
		r := acqRun(t, bin, dsn, ws)
		if !r.OK {
			t.Fatalf("%+v", r.Error)
		}
		c := r.Context
		if c.Coverage.Status != "partial" || !acqHas(c.Coverage.Reasons, "append-or-partitioned") ||
			!acqHas(c.Coverage.Reasons, "declared-relation-not-read:public.orders") {
			t.Fatalf("coverage=%+v", c.Coverage)
		}
		if !strings.Contains(strings.Join(c.Acquisition.Observation.PlanRelations, ","), "public.orders_p1") {
			t.Fatalf("plan relations=%v", c.Acquisition.Observation.PlanRelations)
		}
	})
}

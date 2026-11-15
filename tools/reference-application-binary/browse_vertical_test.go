package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tei/tools/reference-application-binary/internal/browsegen"
)

const (
	b1App     = "order_entry"
	b1Pattern = "order_entry.recent_orders"
)

func b1Workspace(t *testing.T) string {
	return filepath.Join(p0aRepoRoot(t), "reference", "workspace")
}

// b1CopyWorkspace returns a private copy so a test can edit the Source.
func b1CopyWorkspace(t *testing.T) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "ws")
	src := b1Workspace(t)
	err := filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dst, rel), b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func b1SQLPath(ws string) string {
	return filepath.Join(ws, "applications", "order-entry", "queries", "recent_orders.sql")
}

func b1Generate(t *testing.T, ws string) (*browsegen.Plan, string) {
	t.Helper()
	plan, err := browsegen.NewPlan(ws, b1App, b1Pattern, browsegen.TargetPostgres)
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	out := t.TempDir()
	if _, err := browsegen.Generate(plan, out); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return plan, out
}

func b1Read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// T1: the generated code must react to the Source.
func TestB1SourceSensitivity(t *testing.T) {
	base := b1Workspace(t)
	planA, outA := b1Generate(t, base)

	ws := b1CopyWorkspace(t)
	raw, _ := os.ReadFile(b1SQLPath(ws))
	mut := strings.Replace(string(raw), "o.ordered_at >= :from", "o.ordered_at < :from", 1)
	mut = strings.Replace(mut, "            o.status,\n", "", 1)
	if mut == string(raw) {
		t.Fatal("mutation did not apply")
	}
	if err := os.WriteFile(b1SQLPath(ws), []byte(mut), 0o644); err != nil {
		t.Fatal(err)
	}
	planB, outB := b1Generate(t, ws)
	if b1Read(t, outA, "main.go") == b1Read(t, outB, "main.go") {
		t.Fatal("main.go identical after the Source SQL changed")
	}
	if planA.SourceRevision == planB.SourceRevision || planA.ModelDigest == planB.ModelDigest {
		t.Fatalf("revisions did not change: %s/%s %s/%s", planA.SourceRevision, planB.SourceRevision, planA.ModelDigest, planB.ModelDigest)
	}
}

// T2: the same Source generates byte-identical output.
func TestB1Deterministic(t *testing.T) {
	_, a := b1Generate(t, b1Workspace(t))
	_, b := b1Generate(t, b1Workspace(t))
	for _, f := range []string{"main.go", "go.mod", "manifest.json"} {
		if b1Read(t, a, f) != b1Read(t, b, f) {
			t.Errorf("%s differs between two generations of the same Source", f)
		}
	}
}

// T7: a browse Function carries no write, no other Function, no migration.
func TestB1NoForeignCode(t *testing.T) {
	_, out := b1Generate(t, b1Workspace(t))
	src := b1Read(t, out, "main.go")
	for _, bad := range []string{"INSERT", "UPDATE", "DELETE", "submit_order", "applyMigration", "customer_reference", "sqlite"} {
		if strings.Contains(src, bad) {
			t.Errorf("generated main.go contains %q", bad)
		}
	}
	if !strings.Contains(src, "ReadOnly: true") || !strings.Contains(src, "public.orders") || !strings.Contains(src, "$1") {
		t.Error("generated main.go lacks the READ ONLY transaction or the Source SQL")
	}
	// Mutant: the check must notice a generator that drops READ ONLY.
	if strings.Contains(strings.Replace(src, "ReadOnly: true", "ReadOnly: false", 1), "ReadOnly: true") {
		t.Error("the read-only check cannot tell a mutant from the original")
	}
}

// T8: an unsupported Target fails before anything is generated.
func TestB1TargetCapability(t *testing.T) {
	_, err := browsegen.NewPlan(b1Workspace(t), b1App, b1Pattern, "sqlite")
	if !errors.Is(err, browsegen.ErrTargetDialectUnsupported) {
		t.Fatalf("want TARGET_DIALECT_UNSUPPORTED, got %v", err)
	}
	if _, err := browsegen.NewPlan(b1Workspace(t), b1App, "order_entry.submit_order", browsegen.TargetPostgres); err == nil {
		t.Fatal("a command Pattern must not be accepted by the browse generator")
	}
}

// No silent omission: what the Application declares and was not generated is listed.
func TestB1OmissionsListed(t *testing.T) {
	plan, _ := b1Generate(t, b1Workspace(t))
	got := map[string]bool{}
	for _, o := range plan.Omissions {
		got[o.Kind+"/"+o.ID+"="+o.Reason] = true
	}
	for _, want := range []string{
		"validation/sales.order_line.quantity_positive=not-selected",
		"validation/sales.order.lines.line_number_unique=not-selected",
		"validation/sales.order_line.line_number_positive=not-selected",
		"pattern/order_entry.submit_order=not-selected",
		"database/master=not-selected",
	} {
		if !got[want] {
			t.Errorf("omission %q not reported (have %v)", want, plan.Omissions)
		}
	}
	if plan.InputDeclaration != "undeclared" || len(plan.Params) != 1 || plan.Params[0] != "from" {
		t.Errorf("input handling: declaration=%s params=%v", plan.InputDeclaration, plan.Params)
	}
}

func b1BuildBinary(t *testing.T, out string) (*browsegen.Receipt, string) {
	t.Helper()
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("native build only")
	}
	bin := filepath.Join(t.TempDir(), "recent-orders")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	rc, err := browsegen.Build(out, bin)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return rc, bin
}

func b1Run(bin string, env []string, args ...string) (string, int) {
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), env...)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	err := cmd.Run()
	code := 0
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		code = -1
	}
	return strings.TrimSpace(o.String()) + "|" + strings.TrimSpace(e.String()), code
}

// T5 (pre-DB part), T9: no database needed. The DSN points at a closed port, so
// only a rejection that happens before the database is contacted gives exit 2.
func TestB1BinaryRejectsBeforeDatabaseAndDetectsStale(t *testing.T) {
	ws := b1CopyWorkspace(t)
	_, out := b1Generate(t, ws)
	rc, bin := b1BuildBinary(t, out)
	if err := browsegen.Verify(out, bin, rc); err != nil {
		t.Fatalf("fresh artifact must verify: %v", err)
	}
	dead := []string{"TEI_DSN=postgres://x:x@127.0.0.1:1/none?sslmode=disable&connect_timeout=1"}
	cases := map[string][]string{
		"no argument":    {"query", b1Pattern},
		"extra argument": {"query", b1Pattern, "--from=2026-11-07T09:00:00Z", "--status=accepted"},
		"duplicate":      {"query", b1Pattern, "--from=a", "--from=b"},
		"wrong function": {"query", "order_entry.customer_reference", "--from=a"},
		"positional":     {"query", b1Pattern, "2026-11-07T09:00:00Z"},
	}
	for name, args := range cases {
		if msg, code := b1Run(bin, dead, args...); code != 2 {
			t.Errorf("%s: exit %d, want 2 (rejected before the database): %s", name, code, msg)
		}
	}
	if msg, code := b1Run(bin, dead, "query", b1Pattern, "--from=2026-11-07T09:00:00Z"); code != 1 {
		t.Errorf("valid arguments to a dead database: exit %d, want 1: %s", code, msg)
	}
	// T9: change the Source and regenerate without rebuilding.
	raw, _ := os.ReadFile(b1SQLPath(ws))
	if err := os.WriteFile(b1SQLPath(ws), []byte(strings.Replace(string(raw), "DESC", "ASC", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := browsegen.NewPlan(ws, b1App, b1Pattern, browsegen.TargetPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := browsegen.Generate(plan, out); err != nil {
		t.Fatal(err)
	}
	if err := browsegen.Verify(out, bin, rc); err == nil || !strings.Contains(err.Error(), "STALE_BINARY") {
		t.Fatalf("a regenerated Source with the old binary must fail Verify, got %v", err)
	}
}

type b1Expected struct {
	Columns []string                     `json:"columns"`
	Rows    map[string]map[string]string `json:"rows"`
	Cases   []struct {
		Name     string   `json:"name"`
		From     string   `json:"from"`
		OrderIDs []string `json:"orderIds"`
	} `json:"cases"`
}

// T3-T6: real PostgreSQL. Opt-in: TEI_TASK9_PG_DSN must name a disposable
// database named tei_task9_probe that already holds recent_orders_product_seed.sql.
func TestB1AgainstRealPostgres(t *testing.T) {
	dsn := os.Getenv("TEI_TASK9_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_TASK9_PG_DSN not set (NOT RUN)")
	}
	if !strings.Contains(dsn, "/tei_task9_probe") {
		t.Fatal("refusing a database not named tei_task9_probe")
	}
	var exp b1Expected
	raw, err := os.ReadFile(filepath.Join("testdata", "browse_recent_orders_expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &exp); err != nil {
		t.Fatal(err)
	}
	_, out := b1Generate(t, b1Workspace(t))
	rc, bin := b1BuildBinary(t, out)
	if err := browsegen.Verify(out, bin, rc); err != nil {
		t.Fatal(err)
	}
	env := []string{"TEI_DSN=" + dsn}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	query := func(from string) ([]map[string]any, string, int) {
		msg, code := b1Run(bin, env, "query", b1Pattern, "--from="+from)
		if code != 0 {
			return nil, msg, code
		}
		var rows []map[string]any
		body, _, _ := strings.Cut(msg, "|")
		if err := json.Unmarshal([]byte(body), &rows); err != nil {
			t.Fatalf("output is not a JSON array: %v: %s", err, body)
		}
		return rows, msg, 0
	}
	for _, c := range exp.Cases {
		rows, msg, code := query(c.From)
		if code != 0 {
			t.Errorf("%s: exit %d: %s", c.Name, code, msg)
			continue
		}
		var ids []string
		for _, r := range rows {
			id, _ := r["order_id"].(string)
			ids = append(ids, id)
			for col, wv := range exp.Rows[id] {
				if got, _ := r[col].(string); got != wv {
					t.Errorf("%s %s.%s = %v, want %q", c.Name, id, col, r[col], wv)
				}
			}
			if len(r) != len(exp.Columns) {
				t.Errorf("%s: %d columns, want %d", c.Name, len(r), len(exp.Columns))
			}
		}
		if strings.Join(ids, ",") != strings.Join(c.OrderIDs, ",") {
			t.Errorf("%s: order %v, want %v", c.Name, ids, c.OrderIDs)
		}
		// T4: independent reader, never the generated binary.
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM public.orders WHERE ordered_at >= $1::timestamptz`, c.From).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != len(c.OrderIDs) {
			t.Errorf("%s: independent count %d, expected file says %d", c.Name, n, len(c.OrderIDs))
		}
	}
	// T5: input that must be refused, not turned into an empty result.
	if _, msg, code := query("42"); code == 0 {
		t.Errorf("from=42 accepted: %s", msg)
	}
	// T6: NULL stays null.
	if _, err := db.Exec(`ALTER TABLE public.orders ALTER COLUMN total_amount DROP NOT NULL`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE public.orders SET total_amount = NULL WHERE order_id = 'O-002'`); err != nil {
		t.Fatal(err)
	}
	rows, msg, code := query("2026-11-07T09:00:00Z")
	if code != 0 {
		t.Fatalf("NULL case: %s", msg)
	}
	for _, r := range rows {
		if r["order_id"] == "O-002" {
			if v, present := r["total_amount"]; !present || v != nil {
				t.Errorf("NULL total_amount came out as %#v (present=%v)", v, present)
			}
		}
	}
	// T5: a dropped column stops the Function; it is not an empty result.
	if _, err := db.Exec(`ALTER TABLE public.orders DROP COLUMN status`); err != nil {
		t.Fatal(err)
	}
	if _, msg, code := query("2026-11-07T09:00:00Z"); code == 0 {
		t.Errorf("dropped column accepted: %s", msg)
	}
}

// b1ReferenceRows runs the existing Environment-bound reference Consumer
// (tools/reference-browse-read, a hand-wired path that is not generated) against
// the same database and returns its rows as column -> rendered value.
func b1ReferenceRows(t *testing.T, refBin, dsn, from string) []map[string]any {
	t.Helper()
	cmd := exec.Command(refBin, "-workspace", b1Workspace(t), "-environment", "development",
		"-application", b1App, "-pattern", b1Pattern, "-dsn-env", "RBR_DSN",
		"-confirm-isolated", "tei_task9_probe", "-param", "from=time:"+from)
	cmd.Env = append(os.Environ(), "RBR_DSN="+dsn)
	var o bytes.Buffer
	cmd.Stdout = &o
	_ = cmd.Run()
	var resp struct {
		OK      bool `json:"ok"`
		Context *struct {
			Rowset struct {
				RawColumns []string `json:"rawColumns"`
			} `json:"rowset"`
			Rows [][]struct {
				State string `json:"state"`
				Value any    `json:"value"`
			} `json:"rows"`
		} `json:"context"`
		Error *struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(o.Bytes(), &resp); err != nil || !resp.OK || resp.Context == nil {
		t.Fatalf("reference Consumer failed: %v %s", err, o.String())
	}
	var rows []map[string]any
	for _, r := range resp.Context.Rows {
		m := map[string]any{}
		for i, c := range r {
			if c.State == "null" {
				m[resp.Context.Rowset.RawColumns[i]] = nil
			} else {
				m[resp.Context.Rowset.RawColumns[i]] = c.Value
			}
		}
		rows = append(rows, m)
	}
	return rows
}

// B0 comparison: the generated binary and the hand-wired reference Consumer must
// return the same rows, in the same order, for the same Source and database.
func TestB1MatchesHandWiredReference(t *testing.T) {
	dsn := os.Getenv("TEI_TASK9_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_TASK9_PG_DSN not set (NOT RUN)")
	}
	refBin := filepath.Join(t.TempDir(), "reference-browse-read")
	build := exec.Command("go", "build", "-o", refBin, "./tools/reference-browse-read")
	build.Dir = p0aRepoRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build reference Consumer: %v %s", err, out)
	}
	_, out := b1Generate(t, b1Workspace(t))
	_, bin := b1BuildBinary(t, out)
	for _, from := range []string{"2026-11-07T09:00:00Z", "2026-11-07T10:05:00Z", "2026-11-07T10:10:01Z"} {
		ref := b1ReferenceRows(t, refBin, dsn, from)
		msg, code := b1Run(bin, []string{"TEI_DSN=" + dsn}, "query", b1Pattern, "--from="+from)
		if code != 0 {
			t.Fatalf("generated binary: %s", msg)
		}
		body, _, _ := strings.Cut(msg, "|")
		var got []map[string]any
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != len(ref) {
			t.Fatalf("from=%s: generated %d rows, reference %d", from, len(got), len(ref))
		}
		for i := range ref {
			for col, rv := range ref[i] {
				if gv, ok := got[i][col]; !ok || strings.TrimSpace(sprint(gv)) != strings.TrimSpace(sprint(rv)) {
					t.Errorf("from=%s row %d %s: generated %v, reference %v", from, i, col, got[i][col], rv)
				}
			}
		}
	}
}

func sprint(v any) string {
	if v == nil {
		return "<null>"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

package orderrollouthost

// Real PostgreSQL harness. The tests need one dedicated, disposable database
// named tei_orhc (normally started by tools/verify-order-rollout-host-continuity.py).
// Without TEI_ORHC_ADMIN_DSN they skip; a skip is not evidence that anything
// passed. Each scenario creates its own control and target databases
// (tei_orhc_<scenario>_ctl, tei_orhc_<scenario>_sales) and two roles.
//
// Host A and host B run as separate OS processes (the test binary acting as
// the CLI). The migration tool runs as a child process of host A (the test
// binary started under the name "atlas").

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	adminDSNEnv       = "TEI_ORHC_ADMIN_DSN"
	confirmEnv        = "TEI_ORHC_CONFIRM_ISOLATED"
	outDirEnv         = "TEI_ORHC_OUT_DIR"
	confirmValue      = "tei_orhc"
	observedLogPrefix = "ORHC-OBSERVED"
)

func TestMain(m *testing.M) {
	if raw := os.Getenv(ChildArgsEnv); raw != "" {
		var args []string
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			fmt.Fprintln(os.Stderr, "child args")
			os.Exit(2)
		}
		os.Exit(Main(args, os.Stdin, os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

func observed(t *testing.T, name string, v any) {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"test": t.Name(), "name": name, "value": v})
	t.Logf("%s %s", observedLogPrefix, raw)
	if dir := os.Getenv(outDirEnv); dir != "" {
		f, err := os.OpenFile(filepath.Join(dir, "observed.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			f.Write(append(raw, '\n'))
			f.Close()
		}
	}
}

type scenario struct {
	t        *testing.T
	ctx      context.Context
	name     string
	repo     string
	exe      string
	binDir   string
	outDir   string
	control  *pgxpool.Pool // admin role on the control database
	sales    *pgxpool.Pool // admin role on the target database
	appDSN   string        // order_entry 3.7.0 role on the target database
	sec      HostSecrets
	secrets  map[string]string // every concrete value that must not leak
	start    Start
	children []*hostProc
}

func dsnFor(cfg *pgx.ConnConfig, db, user, pw string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", url.PathEscape(user), url.PathEscape(pw), cfg.Host, cfg.Port, db)
}

func newScenario(t *testing.T, name string) *scenario {
	t.Helper()
	dsn := os.Getenv(adminDSNEnv)
	if dsn == "" {
		t.Skip(adminDSNEnv + " is not set; a skip is not evidence")
	}
	if os.Getenv(confirmEnv) != confirmValue {
		t.Fatalf("%s must be %s", confirmEnv, confirmValue)
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Database != confirmValue {
		t.Fatalf("admin DSN must name the database %s", confirmValue)
	}
	ctx := context.Background()
	admin, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal("admin connection failed")
	}
	defer admin.Close(ctx)
	ctl, salesDB := "tei_orhc_"+name+"_ctl", "tei_orhc_"+name+"_sales"
	mig, app := "orhc_mig_"+name, "orhc_app_"+name
	migPw, appPw := "mig-"+randomHex(12), "app-"+randomHex(12)
	for _, q := range []string{
		"DROP DATABASE IF EXISTS " + ctl + " WITH (FORCE)", "DROP DATABASE IF EXISTS " + salesDB + " WITH (FORCE)",
		"DROP ROLE IF EXISTS " + mig, "DROP ROLE IF EXISTS " + app,
		"CREATE ROLE " + mig + " LOGIN PASSWORD '" + migPw + "'", "CREATE ROLE " + app + " LOGIN PASSWORD '" + appPw + "'",
		"CREATE DATABASE " + ctl, "CREATE DATABASE " + salesDB,
		// A host stopped inside a transaction must not keep row locks
		// that the takeover host needs.
		"ALTER DATABASE " + ctl + " SET idle_in_transaction_session_timeout = '2s'",
	} {
		if _, err := admin.Exec(ctx, q); err != nil {
			t.Fatalf("setup %q: %v", strings.SplitN(q, " PASSWORD", 2)[0], err)
		}
	}
	s := &scenario{t: t, ctx: ctx, name: name, repo: repoRoot(t)}
	s.exe, err = os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	s.control, err = pgxpool.New(ctx, dsnFor(cfg, ctl, cfg.User, cfg.Password))
	if err != nil {
		t.Fatal(err)
	}
	s.sales, err = pgxpool.New(ctx, dsnFor(cfg, salesDB, cfg.User, cfg.Password))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			s.diagnose()
		}
		for _, c := range s.children {
			c.kill()
		}
		s.control.Close()
		s.sales.Close()
		if a, err := pgx.ConnectConfig(context.Background(), cfg); err == nil {
			for _, q := range []string{"DROP DATABASE IF EXISTS " + ctl + " WITH (FORCE)", "DROP DATABASE IF EXISTS " + salesDB + " WITH (FORCE)", "DROP ROLE IF EXISTS " + mig, "DROP ROLE IF EXISTS " + app} {
				_, _ = a.Exec(context.Background(), q)
			}
			a.Close(context.Background())
		}
	})
	if err := ApplyControlSchema(ctx, s.control); err != nil {
		t.Fatal(err)
	}
	if err := ApplySalesSchema(ctx, s.sales); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		"ALTER TABLE public.orders OWNER TO " + mig, "ALTER SCHEMA tei_atlas OWNER TO " + mig,
		"ALTER TABLE tei_atlas.job OWNER TO " + mig, "ALTER TABLE tei_atlas.revision OWNER TO " + mig,
		// Creating the index of the expand needs CREATE on the schema.
		"GRANT CREATE ON SCHEMA public TO " + mig,
		"GRANT SELECT, INSERT ON public.orders TO " + app,
		// 4.0.0 rows written earlier by 3.7.0; some without status.
		`INSERT INTO public.orders(order_id,customer_id,ordered_at,status,total_amount)
		 SELECT i, 100 + i % 7, timestamptz '2026-08-01 09:00:00+09' + i * interval '1 hour', CASE WHEN i % 4 = 0 THEN NULL ELSE 'OPEN' END, i * 10.5 FROM generate_series(1,20) i`,
	} {
		if _, err := s.sales.Exec(ctx, q); err != nil {
			t.Fatalf("sales setup: %v", err)
		}
	}
	s.start = loadStart(t)
	base := t.TempDir()
	s.outDir = base
	if dir := os.Getenv(outDirEnv); dir != "" {
		s.outDir = filepath.Join(dir, name)
		if err := os.MkdirAll(s.outDir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s.binDir = filepath.Join(base, "bin")
	if err := os.MkdirAll(s.binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(s.exe, filepath.Join(s.binDir, "atlas")); err != nil {
		t.Fatal(err)
	}
	s.sec = HostSecrets{ControlDSN: dsnFor(cfg, ctl, cfg.User, cfg.Password), SecretHandle: "secret-manager://orhc/" + randomHex(8),
		SecretValue: dsnFor(cfg, salesDB, mig, migPw), Endpoint: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		MountHandle: "mount-source://orhc/" + randomHex(8), Workdir: filepath.Join(base, "work-"+randomHex(4))}
	if err := os.MkdirAll(s.sec.Workdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteMigrationBundle(s.sec.Workdir, s.start); err != nil {
		t.Fatal(err)
	}
	s.appDSN = dsnFor(cfg, salesDB, app, appPw)
	s.secrets = map[string]string{"admin-dsn": dsn, "admin-password": cfg.Password, "control-dsn": s.sec.ControlDSN, "secret-handle": s.sec.SecretHandle,
		"credential": s.sec.SecretValue, "migrator-password": migPw, "endpoint": s.sec.Endpoint, "mount-source": s.sec.MountHandle,
		"workdir": s.sec.Workdir, "app-dsn": s.appDSN, "app-password": appPw}
	return s
}

type hostProc struct {
	name, report, stdout, stderr string
	cmd                          *exec.Cmd
	done                         chan struct{}
	err                          error
	once                         sync.Once
}

func (s *scenario) spawn(role, owner string, extra ...string) *hostProc {
	s.t.Helper()
	h := &hostProc{name: owner, report: filepath.Join(s.outDir, owner+".report.json"), stdout: filepath.Join(s.outDir, owner+".stdout"), stderr: filepath.Join(s.outDir, owner+".stderr"), done: make(chan struct{})}
	args := append([]string{"host", "-repo", s.repo, "-role", role, "-owner", owner, "-out", h.report, "-bin-dir", s.binDir}, extra...)
	raw, _ := json.Marshal(args)
	h.cmd = exec.Command(s.exe)
	h.cmd.Env = append([]string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), ChildArgsEnv + "=" + string(raw)}, s.sec.Env()...)
	so, err := os.Create(h.stdout)
	if err != nil {
		s.t.Fatal(err)
	}
	se, err := os.Create(h.stderr)
	if err != nil {
		s.t.Fatal(err)
	}
	h.cmd.Stdout, h.cmd.Stderr = so, se
	if err := h.cmd.Start(); err != nil {
		s.t.Fatal(err)
	}
	go func() {
		h.err = h.cmd.Wait()
		so.Close()
		se.Close()
		close(h.done)
	}()
	s.children = append(s.children, h)
	return h
}

func (h *hostProc) kill() {
	h.once.Do(func() {
		select {
		case <-h.done:
			return
		default:
		}
		_ = ContinueProcess(h.cmd.Process.Pid)
		_ = h.cmd.Process.Kill()
		<-h.done
	})
}

func (s *scenario) waitExit(h *hostProc, timeout time.Duration) {
	s.t.Helper()
	select {
	case <-h.done:
		if h.err != nil {
			raw, _ := os.ReadFile(h.stderr)
			s.t.Fatalf("%s exited with %v: %s", h.name, h.err, raw)
		}
	case <-time.After(timeout):
		s.t.Fatalf("%s did not exit within %s", h.name, timeout)
	}
}

func (s *scenario) waitFor(what string, timeout time.Duration, cond func() (bool, error)) {
	s.t.Helper()
	end := time.Now().Add(timeout)
	for {
		ok, err := cond()
		if err != nil {
			s.t.Fatalf("waiting for %s: %v", what, err)
		}
		if ok {
			return
		}
		if time.Now().After(end) {
			s.t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *scenario) count(pool *pgxpool.Pool, q string, args ...any) func() (bool, error) {
	return func() (bool, error) {
		var n int64
		err := pool.QueryRow(s.ctx, q, args...).Scan(&n)
		return n > 0, err
	}
}

func (s *scenario) scalar(pool *pgxpool.Pool, q string, args ...any) int64 {
	s.t.Helper()
	var n int64
	if err := pool.QueryRow(s.ctx, q, args...).Scan(&n); err != nil {
		s.t.Fatalf("query %q: %v", q, err)
	}
	return n
}

func readReport[T any](t *testing.T, path string) (T, string) {
	t.Helper()
	var doc struct {
		Report T      `json:"report"`
		Error  string `json:"error"`
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Report, doc.Error
}

// legacyWriter is order_entry 3.7.0: it writes orders through the legacy
// contract (the 4.0.0 columns only) with its own role, and records the
// phase of the migration it saw just before each write.
type legacyWrite struct {
	ID        int64
	Customer  int64
	OrderedAt time.Time
	Status    string
	Total     string
	Phase     string
	Err       string
	Latency   time.Duration
}

type legacyWriter struct {
	s      *scenario
	mu     sync.Mutex
	writes []legacyWrite
	stop   chan struct{}
	done   chan struct{}
}

func (s *scenario) startWriter() *legacyWriter {
	s.t.Helper()
	w := &legacyWriter{s: s, stop: make(chan struct{}), done: make(chan struct{})}
	if _, err := s.control.Exec(s.ctx, `INSERT INTO orhc.app_instance VALUES ('order-entry-370-a',$1,$2,true)`, OrderApplication, LegacyVersion); err != nil {
		s.t.Fatal(err)
	}
	conn, err := pgx.Connect(s.ctx, s.appDSN)
	if err != nil {
		s.t.Fatal("writer connection failed")
	}
	go func() {
		defer close(w.done)
		defer conn.Close(context.Background())
		id := int64(1000)
		for {
			select {
			case <-w.stop:
				return
			case <-time.After(20 * time.Millisecond):
			}
			phase := "before-apply"
			var lc string
			if err := s.sales.QueryRow(s.ctx, `SELECT lifecycle FROM tei_atlas.job ORDER BY accepted_at LIMIT 1`).Scan(&lc); err == nil {
				switch lc {
				case "applying":
					phase = "applying"
				case "succeeded":
					phase = "after-apply"
				}
			}
			id++
			wr := legacyWrite{ID: id, Customer: 100 + id%7, OrderedAt: time.Date(2026, 8, 27, 18, 30, 0, 0, time.UTC).Add(time.Duration(id) * time.Second), Status: "OPEN", Total: fmt.Sprintf("%d.%02d", id%1000, id%100), Phase: phase}
			began := time.Now()
			_, err := conn.Exec(s.ctx, `INSERT INTO public.orders(order_id,customer_id,ordered_at,status,total_amount) VALUES ($1,$2,$3,$4,$5::numeric)`, wr.ID, wr.Customer, wr.OrderedAt, wr.Status, wr.Total)
			wr.Latency = time.Since(began)
			if err != nil {
				wr.Err = err.Error()
			}
			w.mu.Lock()
			w.writes = append(w.writes, wr)
			w.mu.Unlock()
		}
	}()
	return w
}

func (w *legacyWriter) n() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.writes)
}

// finish stops the writer and checks every write against the database.
func (w *legacyWriter) finish() map[string]any {
	close(w.stop)
	<-w.done
	s := w.s
	if _, err := s.control.Exec(s.ctx, `UPDATE orhc.app_instance SET active=false`); err != nil {
		s.t.Fatal(err)
	}
	phases := map[string]int{}
	var errs []string
	var maxLatency time.Duration
	lost := 0
	for _, wr := range w.writes {
		phases[wr.Phase]++
		if wr.Err != "" {
			errs = append(errs, wr.Err)
			continue
		}
		maxLatency = max(maxLatency, wr.Latency)
		var cust int64
		var at time.Time
		var status, total string
		err := s.sales.QueryRow(s.ctx, `SELECT customer_id, ordered_at, status, total_amount::text FROM public.orders WHERE order_id=$1`, wr.ID).Scan(&cust, &at, &status, &total)
		if err != nil || cust != wr.Customer || !at.Equal(wr.OrderedAt) || status != wr.Status || total != wr.Total {
			lost++
		}
	}
	sort.Strings(errs)
	return map[string]any{"writes": len(w.writes), "phases": phases, "errors": len(errs), "errorSamples": errs[:min(3, len(errs))], "lostOrChanged": lost, "maxLatencyMs": maxLatency.Milliseconds()}
}

// leakText collects every output of steps 1-5 of the scenario: the host
// reports and their standard output and error, every row of the control
// records and of the tool registry, and the given texts.
func (s *scenario) leakText(extra ...string) string {
	s.t.Helper()
	var b strings.Builder
	entries, _ := os.ReadDir(s.outDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(s.outDir, e.Name()))
		if err == nil {
			b.Write(raw)
			b.WriteByte('\n')
		}
	}
	ctl, err := DumpTables(s.ctx, s.control, ControlTables)
	if err != nil {
		s.t.Fatal(err)
	}
	sales, err := DumpTables(s.ctx, s.sales, SalesRecordTables)
	if err != nil {
		s.t.Fatal(err)
	}
	b.WriteString(ctl)
	b.WriteString(sales)
	for _, x := range extra {
		b.WriteString(x)
		b.WriteByte('\n')
	}
	return b.String()
}

func (s *scenario) writeOut(name string, v any) string {
	s.t.Helper()
	raw, _ := json.MarshalIndent(v, "", "  ")
	if err := os.WriteFile(filepath.Join(s.outDir, name), append(raw, '\n'), 0o644); err != nil {
		s.t.Fatal(err)
	}
	return string(raw)
}

// diagnose logs the records of a failed scenario. The rows contain no
// concrete values (that is what the leak check verifies).
func (s *scenario) diagnose() {
	ctx := context.Background()
	for _, q := range []struct {
		pool *pgxpool.Pool
		sql  string
	}{
		{s.sales, `SELECT to_jsonb(j)::text FROM tei_atlas.job j`},
		{s.sales, `SELECT to_jsonb(r)::text FROM tei_atlas.revision r`},
		{s.sales, `SELECT pid::text || ' ' || coalesce(wait_event_type,'') || ' ' || coalesce(wait_event,'') || ' ' || state || ' ' || left(query, 80) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid()`},
		{s.control, `SELECT to_jsonb(e)::text FROM orhc.call_event e ORDER BY seq`},
		{s.control, `SELECT op || ' ' || actor || ' ' || accepted::text || ' ' || count(*)::text FROM orhc.ownership_event GROUP BY op, actor, accepted ORDER BY 1`},
		{s.control, `SELECT op || ' ' || actor || ' ' || accepted::text || ' ' || coalesce(message,'') FROM orhc.lease_event ORDER BY seq DESC LIMIT 12`},
	} {
		rows, err := q.pool.Query(ctx, q.sql)
		if err != nil {
			s.t.Logf("diagnose: %v", err)
			continue
		}
		for rows.Next() {
			var line string
			_ = rows.Scan(&line)
			s.t.Logf("diagnose: %s", line)
		}
		rows.Close()
	}
	for _, c := range s.children {
		raw, _ := os.ReadFile(c.stderr)
		s.t.Logf("diagnose: %s stderr: %s", c.name, raw)
		if rep, err := os.ReadFile(c.report); err == nil {
			s.t.Logf("diagnose: %s report: %s", c.name, rep)
		}
	}
}

package orderassistoffline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	offlinestorage "tei/tools/reference-offline-storage"
)

// Shared test setup. The realizations need Go and Node.js; the PostgreSQL
// tests need one dedicated, disposable database named tei_oapo (normally
// started by tools/verify-order-assist-provider-offline.py). Without the
// tools or TEI_OAPO_ADMIN_DSN the tests skip; a skip is not a pass.

const (
	adminDSNEnv       = "TEI_OAPO_ADMIN_DSN"
	confirmEnv        = "TEI_OAPO_CONFIRM_ISOLATED"
	outDirEnv         = "TEI_OAPO_OUT_DIR"
	childArgsEnv      = "TEI_OAPO_CHILD_ARGS"
	confirmValue      = "tei_oapo"
	observedLogPrefix = "OAPO-OBSERVED"
)

var (
	realOnce sync.Once
	realSet  *RealizationSet
	realErr  error
	realDir  string
)

func TestMain(m *testing.M) {
	if raw := os.Getenv(childArgsEnv); raw != "" {
		var args []string
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		os.Exit(Main(args, os.Stdout, os.Stderr))
	}
	code := m.Run()
	if realDir != "" {
		_ = os.RemoveAll(realDir)
	}
	os.Exit(code)
}

func repoRoot(t testing.TB) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// realizations builds the six realizations once per test binary.
func realizations(t testing.TB) *RealizationSet {
	t.Helper()
	realOnce.Do(func() {
		realDir, realErr = os.MkdirTemp("", "tei-oapo-realizations-")
		if realErr != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		realSet, realErr = BuildRealizations(ctx, repoRoot(t), realDir)
	})
	if errors.Is(realErr, ErrToolMissing) {
		t.Skipf("realizations need Go and Node.js (%v); skip is not a pass", realErr)
	}
	if realErr != nil {
		t.Fatal(realErr)
	}
	return realSet
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

// serverDistribution is the server side of the rollout once it is complete:
// the native executable on the order server and the WASI module for the
// nightly review, both generation 2.
func serverDistribution(set *RealizationSet) Distribution {
	srv, batch := set.Get(2, RuntimeGoNative), set.Get(2, RuntimeWASI)
	return Distribution{TargetGeneration: 2, Sites: []Deployment{
		{Site: SiteServer, Realization: srv.Name, Generation: 2, ArtifactDigest: srv.ArtifactDigest},
		{Site: SiteBatch, Realization: batch.Name, Generation: 2, ArtifactDigest: batch.ArtifactDigest},
	}}
}

type harness struct {
	t          *testing.T
	ctx        context.Context
	admin      *pgx.Conn
	real       *RealizationSet
	server     *Server
	http       *httptest.Server
	credit     *CreditClient
	creditURL  string
	creditPool *pgxpool.Pool
	serverDSN  string
	dir        string
	policyV1   Policy
	policyV2   Policy
	migration  Migration
}

func roleDSN(t testing.TB, admin, role string) string {
	t.Helper()
	u, err := url.Parse(admin)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.User(role)
	return u.String()
}

func newHarness(t *testing.T, opts ServerOptions) *harness {
	t.Helper()
	dsn := os.Getenv(adminDSNEnv)
	if dsn == "" {
		t.Skip("requires a dedicated PostgreSQL database (TEI_OAPO_ADMIN_DSN); skip is not a pass")
	}
	if os.Getenv(confirmEnv) != confirmValue {
		t.Fatalf("%s must be %q to confirm the database is disposable", confirmEnv, confirmValue)
	}
	set := realizations(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	t.Cleanup(cancel)
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(context.Background()) })
	var db string
	if err := admin.QueryRow(ctx, `SELECT current_database()`).Scan(&db); err != nil || db != confirmValue {
		t.Fatalf("connected database is %q, want %q (%v)", db, confirmValue, err)
	}
	if err := ResetSchemas(ctx, admin); err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, ctx: ctx, admin: admin, real: set, dir: t.TempDir()}
	h.serverDSN = roleDSN(t, dsn, RoleServer)
	pool, err := pgxpool.New(ctx, h.serverDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h.server = &Server{Pool: pool, Real: set, Dist: serverDistribution(set), Opts: opts}
	if err := h.server.Init(ctx, "2026-10-01", 1, 2); err != nil {
		t.Fatal(err)
	}
	h.http = httptest.NewServer(h.server.Handler())
	t.Cleanup(h.http.Close)
	if h.policyV1, err = LoadPolicy("testdata/approval-policy-v1.json"); err != nil {
		t.Fatal(err)
	}
	if h.policyV2, err = LoadPolicy("testdata/approval-policy-v2.json"); err != nil {
		t.Fatal(err)
	}
	if h.migration, err = LoadMigration("testdata/policy-migration-v1-v2.json"); err != nil {
		t.Fatal(err)
	}
	if err := h.server.InstallPolicy(ctx, h.policyV1, true); err != nil {
		t.Fatal(err)
	}
	for id, role := range map[string]string{"sales-a": "sales", "sales-b": "sales", "mgr-kato": "manager", "head-sato": "sales-head",
		"deputy-ito": "sales", "dir-mori": "director", "fin-ueda": "finance"} {
		if err := h.server.AddPrincipal(ctx, id, role); err != nil {
			t.Fatal(err)
		}
	}
	h.startCreditProvider(dsn)
	return h
}

func (h *harness) child(args ...string) *exec.Cmd {
	raw, _ := json.Marshal(args)
	cmd := exec.CommandContext(h.ctx, os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), childArgsEnv+"="+string(raw))
	return cmd
}

func (h *harness) startCreditProvider(adminDSN string) {
	t := h.t
	creditDSN := roleDSN(t, adminDSN, RoleCredit)
	ready := filepath.Join(h.dir, "credit-ready.json")
	cmd := h.child("credit-provider", "-dsn", creditDSN, "-ready-file", ready)
	logPath := filepath.Join(h.dir, "credit-provider.log")
	logf, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = logf, logf
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan struct{})
		go func() { _ = cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			_ = cmd.Process.Kill()
			<-done
		}
		logf.Close()
	})
	start := time.Now()
	for {
		raw, err := os.ReadFile(ready)
		if err == nil {
			var r struct{ URL string }
			if json.Unmarshal(raw, &r) == nil && r.URL != "" {
				h.creditURL = r.URL
				break
			}
		}
		if time.Since(start) > 2*time.Minute {
			log, _ := os.ReadFile(logPath)
			t.Fatalf("credit provider did not become ready: %s", log)
		}
		time.Sleep(50 * time.Millisecond)
	}
	h.credit = NewCreditClient(h.creditURL)
	pool, err := pgxpool.New(h.ctx, creditDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	h.creditPool = pool
	if err := (&CreditProvider{Pool: pool}).SeedAccount(h.ctx, "C-100", 1_000_000); err != nil {
		t.Fatal(err)
	}
}

func (h *harness) setCreditFault(op string, remaining int) {
	h.t.Helper()
	if _, err := h.creditPool.Exec(h.ctx, `INSERT INTO oapo_credit.fault VALUES ($1,'drop-after-commit',$2)
		ON CONFLICT (op) DO UPDATE SET remaining=EXCLUDED.remaining`, op, remaining); err != nil {
		h.t.Fatal(err)
	}
}

// providerCounts reads the provider's own records (through the provider's
// role, as an auditor of the external system would).
type providerCounts struct {
	Reservations   int   `json:"reservations"`
	ReleaseCount   int   `json:"releaseCount"`
	UsedMinor      int64 `json:"usedMinor"`
	ReserveCalls   int   `json:"reserveCalls"`
	ReleaseCalls   int   `json:"releaseCalls"`
	InquireCalls   int   `json:"inquireCalls"`
	ConsumedEvents int   `json:"consumedEvents"`
	ReleasedEvents int   `json:"releasedEvents"`
}

func (h *harness) provider(key string) providerCounts {
	h.t.Helper()
	var c providerCounts
	err := h.creditPool.QueryRow(h.ctx, `SELECT
		(SELECT count(*) FROM oapo_credit.reservation WHERE idempotency_key=$1 OR ($1='' AND idempotency_key IS NULL)),
		(SELECT coalesce(sum(release_count),0) FROM oapo_credit.reservation WHERE idempotency_key=$1 OR ($1='' AND idempotency_key IS NULL)),
		(SELECT used_minor FROM oapo_credit.account WHERE customer_code='C-100'),
		(SELECT count(*) FROM oapo_credit.request_log WHERE op='reserve' AND coalesce(idempotency_key,'')=$1),
		(SELECT count(*) FROM oapo_credit.request_log WHERE op='release' AND coalesce(idempotency_key,'')=$1),
		(SELECT count(*) FROM oapo_credit.request_log WHERE op='inquire' AND coalesce(idempotency_key,'')=$1),
		(SELECT count(*) FROM oapo_credit.request_log WHERE op='reserve' AND coalesce(idempotency_key,'')=$1 AND outcome='consumed'),
		(SELECT count(*) FROM oapo_credit.request_log WHERE op='release' AND coalesce(idempotency_key,'')=$1 AND outcome='released')`,
		key).Scan(&c.Reservations, &c.ReleaseCount, &c.UsedMinor, &c.ReserveCalls, &c.ReleaseCalls, &c.InquireCalls, &c.ConsumedEvents, &c.ReleasedEvents)
	if err != nil {
		h.t.Fatal(err)
	}
	return c
}

func (h *harness) terminal(id string, gen int, naive bool) *Terminal {
	h.t.Helper()
	store, err := offlinestorage.Open(filepath.Join(h.dir, "terminal-"+id))
	if err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(func() { store.Close() })
	return &Terminal{ID: id, Generation: gen, Realization: h.real.Get(gen, RuntimeNodeJS), Store: store,
		ServerURL: h.http.URL, HTTP: &http.Client{Timeout: 5 * time.Minute}, NaiveDecode: naive}
}

func (h *harness) saga() *Saga { return &Saga{Pool: h.server.Pool, Credit: h.credit} }

// sagaProcess runs one saga worker as its own OS process. With a fault it
// must stop with FaultExitCode.
func (h *harness) sagaProcess(caseID, fault string) (int, string) {
	h.t.Helper()
	args := []string{"saga", "-dsn", h.serverDSN, "-credit-url", h.creditURL, "-case", caseID}
	if fault != "" {
		args = append(args, "-fault", fault)
	}
	cmd := h.child(args...)
	out, err := cmd.CombinedOutput()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		h.t.Fatal(err)
	}
	return code, string(out)
}

func (h *harness) quote(id string) QuoteState {
	h.t.Helper()
	st, err := h.server.Quote(h.ctx, id)
	if err != nil {
		h.t.Fatal(err)
	}
	return st
}

func (h *harness) caseState(id string) CaseState {
	h.t.Helper()
	c, err := h.server.Case(h.ctx, id)
	if err != nil {
		h.t.Fatal(err)
	}
	return c
}

func (h *harness) mustDecide(d Decision) {
	h.t.Helper()
	if err := h.server.Decide(h.ctx, d); err != nil {
		h.t.Fatalf("decision %+v: %v", d, err)
	}
}

func (h *harness) countEvents(kind, caseID string) int {
	h.t.Helper()
	var n int
	if err := h.server.Pool.QueryRow(h.ctx, `SELECT count(*) FROM oapo.event WHERE kind=$1 AND coalesce(case_id,'')=$2`, kind, caseID).Scan(&n); err != nil {
		h.t.Fatal(err)
	}
	return n
}

// op unwraps a terminal operation result.
func (h *harness) op(op Op, err error) Op {
	h.t.Helper()
	if err != nil {
		h.t.Fatal(err)
	}
	return op
}

func rawString(s string) json.RawMessage { raw, _ := json.Marshal(s); return raw }

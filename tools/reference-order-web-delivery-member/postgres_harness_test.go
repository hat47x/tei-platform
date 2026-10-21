package orderwebmember

// Real PostgreSQL tests. They need one dedicated, disposable database named
// tei_owdm (normally started by tools/verify-order-web-delivery-member.py)
// with track_commit_timestamp=on. Without TEI_OWDM_ADMIN_DSN they skip; a
// skip is not evidence that anything passed.
//
// The member and staff units run as separate OS processes (the test binary
// acting as the CLI), each connecting as its own DB role.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	adminDSNEnv       = "TEI_OWDM_ADMIN_DSN"
	confirmEnv        = "TEI_OWDM_CONFIRM_ISOLATED"
	outDirEnv         = "TEI_OWDM_OUT_DIR"
	childArgsEnv      = "TEI_OWDM_CHILD_ARGS"
	confirmValue      = "tei_owdm"
	observedLogPrefix = "OWDM-OBSERVED"
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

type harness struct {
	t                  *testing.T
	ctx                context.Context
	adminDSN           string
	admin              *pgx.Conn
	ws                 string
	member             Projection
	staff              Projection
	limits             RoleLimits
	leaks              atomic.Int64 // member responses containing an internal sentinel
	retried            atomic.Int64 // requests resent after 429 by must
	monoStart, dbStart time.Time
	scanned            atomic.Int64 // member response bodies scanned
	unitsMu            sync.Mutex
	units              []*unitProc
	dir                string
}

var (
	projOnce              sync.Once
	projMember, projStaff Projection
	projErr               error
)

func newHarness(t *testing.T) *harness {
	t.Helper()
	dsn := os.Getenv(adminDSNEnv)
	if dsn == "" {
		t.Skip("requires a dedicated PostgreSQL database (TEI_OWDM_ADMIN_DSN); skip is not a pass")
	}
	if os.Getenv(confirmEnv) != confirmValue {
		t.Fatalf("%s must be %q to confirm the database is disposable", confirmEnv, confirmValue)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	t.Cleanup(cancel)
	ws, err := filepath.Abs(workspaceRoot())
	if err != nil {
		t.Fatal(err)
	}
	projOnce.Do(func() {
		var loaded, err = LoadWorkspace(ws)
		if err != nil {
			projErr = err
			return
		}
		if projMember, projErr = DeriveProjection(loaded, KindMember); projErr != nil {
			return
		}
		projStaff, projErr = DeriveProjection(loaded, KindStaff)
	})
	if projErr != nil {
		t.Fatal(projErr)
	}
	h := &harness{t: t, ctx: ctx, adminDSN: dsn, ws: ws, member: projMember, staff: projStaff, limits: DefaultRoleLimits(), dir: t.TempDir()}
	if h.admin, err = pgx.Connect(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.admin.Close(context.Background()) })
	var tct string
	if err := h.admin.QueryRow(ctx, `SHOW track_commit_timestamp`).Scan(&tct); err != nil || tct != "on" {
		t.Fatalf("track_commit_timestamp must be on (got %q, %v)", tct, err)
	}
	if err := ResetSchema(ctx, h.admin, h.member, h.limits); err != nil {
		t.Fatal(err)
	}
	h.monoStart, h.dbStart = time.Now(), h.dbNow()
	t.Cleanup(h.stopAll)
	return h
}

func (h *harness) roleDSN(role string) string {
	u, err := url.Parse(h.adminDSN)
	if err != nil {
		h.t.Fatal(err)
	}
	u.User = url.User(role)
	return u.String()
}

func (h *harness) dbNow() time.Time {
	var now time.Time
	if err := h.admin.QueryRow(h.ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		h.t.Fatal(err)
	}
	return now
}

// waitDBClock waits, by observing the DB clock, until it has passed target.
func (h *harness) waitDBClock(target time.Time) {
	for h.dbNow().Before(target) {
		select {
		case <-h.ctx.Done():
			h.t.Fatal("context ended while waiting for the DB clock")
		case <-time.After(20 * time.Millisecond):
		}
	}
}

type unitProc struct {
	h        *harness
	name     string
	unit     string
	cmd      *exec.Cmd
	ready    Ready
	base     string
	stderr   *bytes.Buffer
	done     chan error
	client   *http.Client
	stopOnce sync.Once
}

type unitOptions struct {
	dsn    string // default: the unit's own role
	expect string // default: the derived projection digest of the unit kind
	args   []string
}

func (h *harness) startUnit(name, kind string, opt unitOptions) *unitProc {
	h.t.Helper()
	up, err := h.tryStartUnit(name, kind, opt)
	if err != nil {
		h.t.Fatal(err)
	}
	return up
}

func (h *harness) tryStartUnit(name, kind string, opt unitOptions) (*unitProc, error) {
	dsn := opt.dsn
	if dsn == "" {
		role := MemberRole
		if kind == KindStaff {
			role = StaffRole
		}
		dsn = h.roleDSN(role)
	}
	expect := opt.expect
	if expect == "" {
		expect = h.member.Digest
		if kind == KindStaff {
			expect = h.staff.Digest
		}
	}
	readyFile := filepath.Join(h.dir, name+".ready.json")
	args := append([]string{"serve", "-unit", kind, "-dsn", dsn, "-workspace", h.ws, "-ready-file", readyFile,
		"-expect-projection-digest", expect}, opt.args...)
	raw, _ := json.Marshal(args)
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), childArgsEnv+"="+string(raw))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	up := &unitProc{h: h, name: name, unit: kind, cmd: cmd, stderr: &stderr, done: make(chan error, 1),
		client: &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 512, MaxIdleConns: 1024}}}
	go func() { up.done <- cmd.Wait() }()
	deadline := time.Now().Add(90 * time.Second)
	for {
		if raw, err := os.ReadFile(readyFile); err == nil {
			if err := json.Unmarshal(raw, &up.ready); err != nil {
				return nil, err
			}
			break
		}
		select {
		case err := <-up.done:
			return nil, fmt.Errorf("unit %s exited before serving: %v: %s", name, err, stderr.String())
		case <-time.After(20 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			<-up.done
			return nil, fmt.Errorf("unit %s did not become ready: %s", name, stderr.String())
		}
	}
	up.base = "http://" + up.ready.Addr
	h.unitsMu.Lock()
	h.units = append(h.units, up)
	h.unitsMu.Unlock()
	return up, nil
}

func (u *unitProc) stop() {
	u.stopOnce.Do(func() {
		_ = u.cmd.Process.Kill()
		select {
		case <-u.done:
			// A race detected inside a unit process (go test -race) is a failure.
			if strings.Contains(u.stderr.String(), "DATA RACE") {
				u.h.t.Errorf("unit %s reported a data race: %s", u.name, u.stderr.String())
			}
		case <-time.After(10 * time.Second):
			u.h.t.Errorf("unit %s did not exit", u.name)
		}
		if u.h.t.Failed() && u.stderr.Len() > 0 {
			tail := u.stderr.String()
			if len(tail) > 4000 {
				tail = tail[len(tail)-4000:]
			}
			u.h.t.Logf("unit %s stderr:\n%s", u.name, tail)
		}
		u.client.CloseIdleConnections()
	})
}

func (h *harness) stopAll() {
	h.unitsMu.Lock()
	defer h.unitsMu.Unlock()
	for _, u := range h.units {
		u.stop()
	}
}

type resp struct {
	status int
	header http.Header
	body   []byte
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %d %s: %v", r.status, r.body, err)
	}
}

var sentinels = []string{SentinelCost, SentinelCredit, SentinelMemo}

func (u *unitProc) do(method, path, principal string, body any, header map[string]string) (resp, error) {
	var rd io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(u.h.ctx, method, u.base+path, rd)
	if err != nil {
		return resp{}, err
	}
	if principal != "" {
		req.Header.Set("Authorization", "Bearer "+TokenOf(principal))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := u.client.Do(req)
	if err != nil {
		return resp{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return resp{}, err
	}
	if u.unit == KindMember {
		u.h.scanned.Add(1)
		for _, s := range sentinels {
			if bytes.Contains(raw, []byte(s)) {
				u.h.leaks.Add(1)
			}
		}
	}
	return resp{status: res.StatusCode, header: res.Header, body: raw}, nil
}

// must sends a request and, like a well-behaved client, honours 429 with
// Retry-After (at most five times). Polls that measure time use do instead.
func (u *unitProc) must(method, path, principal string, body any, header map[string]string) resp {
	u.h.t.Helper()
	for attempt := 0; ; attempt++ {
		r, err := u.do(method, path, principal, body, header)
		if err != nil {
			u.h.t.Fatalf("%s %s: %v", method, path, err)
		}
		if r.status != http.StatusTooManyRequests || attempt == 5 {
			return r
		}
		u.h.retried.Add(1)
		ra, err := strconv.Atoi(r.header.Get("Retry-After"))
		if err != nil || ra < 1 {
			u.h.t.Fatalf("429 without a valid Retry-After: %q", r.header.Get("Retry-After"))
		}
		time.Sleep(time.Duration(ra) * time.Second)
	}
}

func (u *unitProc) stats() map[string]any {
	var m map[string]any
	r := u.must("GET", "/stats", "", nil, nil)
	r.json(u.h.t, &m)
	return m
}

// commitTime returns the commit timestamp of the transaction that last wrote
// the principal row.
func (h *harness) principalCommit(principal string) time.Time {
	var ts time.Time
	if err := h.admin.QueryRow(h.ctx, `SELECT pg_xact_commit_timestamp(xmin) FROM owdm.principal WHERE principal_id = $1`, principal).Scan(&ts); err != nil {
		h.t.Fatal(err)
	}
	return ts
}

func (h *harness) orderRow(id int64) map[string]any {
	var m map[string]any
	if err := h.admin.QueryRow(h.ctx, `SELECT to_jsonb(o) FROM owdm.orders o WHERE order_id = $1`, id).Scan(&m); err != nil {
		h.t.Fatal(err)
	}
	return m
}

func percentile(ds []time.Duration, p float64) time.Duration {
	if len(ds) == 0 {
		return 0
	}
	s := append([]time.Duration(nil), ds...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	idx := int(float64(len(s))*p+0.999999) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }

func keysOf(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		if strings.Contains(s, x) {
			return true
		}
	}
	return false
}

func pgxConnect(h *harness, dsn string) (*pgx.Conn, error) { return pgx.Connect(h.ctx, dsn) }

func sqlState(err error) string {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return pe.Code
	}
	return ""
}

// clockDrift is how far the DB wall clock moved beyond the test's monotonic
// clock since the harness started (positive: the wall clock jumped forward).
func (h *harness) clockDrift() float64 {
	db := h.dbNow().Sub(h.dbStart)
	return ms(db - time.Since(h.monoStart))
}

// waitDBIdle waits, by observing pg_stat_activity, until no other client
// backend is running a query (queries of a killed unit keep running until
// they finish). It returns the time waited in ms.
func (h *harness) waitDBIdle() float64 {
	start := time.Now()
	for {
		var active int
		if err := h.admin.QueryRow(h.ctx, `SELECT count(*) FROM pg_stat_activity
			WHERE datname = current_database() AND backend_type = 'client backend'
			  AND state <> 'idle' AND pid <> pg_backend_pid()`).Scan(&active); err != nil {
			h.t.Fatal(err)
		}
		if active == 0 {
			return ms(time.Since(start))
		}
		if time.Since(start) > time.Minute {
			h.t.Fatalf("%d client backends still active after a minute", active)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

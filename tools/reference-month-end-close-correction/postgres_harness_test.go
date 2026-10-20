//go:build unix

package monthendclose

// Real PostgreSQL tests. They need the dedicated, disposable database
// tei_mecc, normally started by tools/verify-month-end-close-correction.py.
// Without the DSN they skip; a skip is not evidence that anything passed.

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	childArgsEnv      = "TEI_MECC_CHILD_ARGS"
	confirmEnv        = "TEI_MECC_CONFIRM_ISOLATED"
	outDirEnv         = "TEI_MECC_OUT_DIR"
	confirmValue      = "tei_mecc"
	observedLogPrefix = "MECC-OBSERVED"
	month             = "2026-09"
)

// TestMain lets the test binary act as the CLI so a test can start, kill
// and stop a separate process running exactly the CLI code path.
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

type harness struct {
	t    *testing.T
	ctx  context.Context
	dsn  string
	conn *pgx.Conn
	dir  string
	spec *FileSpec
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{t: t, dir: t.TempDir(), dsn: os.Getenv(DSNEnv)}
	if h.dsn == "" {
		t.Skip("requires the dedicated PostgreSQL database (TEI_MECC_DSN); skip is not a pass")
	}
	if os.Getenv(confirmEnv) != confirmValue {
		t.Fatalf("%s must be %q to confirm the database is disposable", confirmEnv, confirmValue)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	t.Cleanup(cancel)
	h.ctx = ctx
	var err error
	if h.conn, err = Connect(ctx, h.dsn, 10*time.Second); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.conn.Close(context.Background()) })
	if err := ResetSchema(ctx, h.conn); err != nil {
		t.Fatal(err)
	}
	h.spec = testSpec(t)
	if _, err := LoadCalendar(ctx, h.conn, testCalendar(t)); err != nil {
		t.Fatal(err)
	}
	for _, p := range syntheticPayables() {
		h.exec(`INSERT INTO mecc.payable VALUES ($1,$2::text::date)`, p.ID, p.NominalDue)
	}
	return h
}

func (h *harness) exec(sql string, args ...any) {
	h.t.Helper()
	if _, err := h.conn.Exec(h.ctx, sql, args...); err != nil {
		h.t.Fatalf("%v: %s", err, sql)
	}
}

func (h *harness) int(sql string, args ...any) int64 {
	h.t.Helper()
	var n int64
	if err := h.conn.QueryRow(h.ctx, sql, args...).Scan(&n); err != nil {
		h.t.Fatalf("%v: %s", err, sql)
	}
	return n
}

func (h *harness) str(sql string, args ...any) string {
	h.t.Helper()
	var s string
	if err := h.conn.QueryRow(h.ctx, sql, args...).Scan(&s); err != nil {
		h.t.Fatalf("%v: %s", err, sql)
	}
	return s
}

func (h *harness) decide(version int, evaluatedAt string) DecideResult {
	h.t.Helper()
	res, err := Decide(h.ctx, h.conn, month, "jp-business", version, testRule(h.t), mustTime(h.t, evaluatedAt))
	if err != nil {
		h.t.Fatal(err)
	}
	return res
}

func (h *harness) revise(from int, date, recordedAt string) Calendar {
	h.t.Helper()
	c, err := ReviseCalendar(h.ctx, h.conn, "jp-business", from, []Holiday{{date, "臨時休業"}}, recordedAt)
	if err != nil {
		h.t.Fatal(err)
	}
	return c
}

func (h *harness) write(name string, b []byte) string {
	h.t.Helper()
	p := filepath.Join(h.dir, name)
	if err := os.WriteFile(p, b, 0o644); err != nil {
		h.t.Fatal(err)
	}
	return p
}

func (h *harness) writeJSON(name string, v any) string {
	h.t.Helper()
	raw, _ := json.MarshalIndent(v, "", "  ")
	return h.write(name, raw)
}

const receivedBeforeCutoff = "2026-09-30T16:30:00+09:00"

// seedEntries stores the September orders, the boundary orders and some
// August and October orders through the online path.
func (h *harness) seedEntries(d Decision) map[string]Entry {
	h.t.Helper()
	out := map[string]Entry{}
	put := func(no, at string, amount int64, awaiting bool) {
		e, err := PlaceEntry(h.ctx, h.conn, d, EntryInput{OrderNo: no, AcceptedAt: mustTime(h.t, at), Amount: amount, Awaiting: awaiting})
		if err != nil {
			h.t.Fatal(err)
		}
		out[no] = e
	}
	for i := 1; i <= 53; i++ {
		put(fmt.Sprintf("SO-9%03d", i), fmt.Sprintf("2026-09-%02dT%02d:15:00+09:00", 1+i%30, 8+i%10), int64(1000+113*i), false)
	}
	put("SO-AUG-LAST", "2026-08-31T23:59:59+09:00", 7777, false)
	put("SO-SEP-FIRST", "2026-08-31T15:00:00Z", 1111, false)    // 00:00 JST on 1 September
	put("SO-SEP-LAST", "2026-09-30T23:59:59+09:00", 9999, true) // left pending allocation
	put("SO-OCT-FIRST", "2026-09-30T15:00:00Z", 2222, false)    // 00:00 JST on 1 October
	return out
}

type proc struct {
	cmd    *exec.Cmd
	out    bytes.Buffer
	errMu  sync.Mutex
	errb   bytes.Buffer
	marker chan string
	done   chan struct{}
	code   int
}

// start runs the CLI in a separate process with extra environment.
func (h *harness) start(env map[string]string, args ...string) *proc {
	h.t.Helper()
	raw, _ := json.Marshal(args)
	p := &proc{marker: make(chan string, 4), done: make(chan struct{})}
	p.cmd = exec.Command(os.Args[0], "-test.run=^$")
	p.cmd.Env = append(os.Environ(), childArgsEnv+"="+string(raw), DSNEnv+"="+h.dsn)
	for k, v := range env {
		p.cmd.Env = append(p.cmd.Env, k+"="+v)
	}
	p.cmd.Stdout = &p.out
	pipe, err := p.cmd.StderrPipe()
	if err != nil {
		h.t.Fatal(err)
	}
	if err := p.cmd.Start(); err != nil {
		h.t.Fatal(err)
	}
	h.t.Cleanup(func() {
		if p.cmd.ProcessState == nil {
			_ = p.cmd.Process.Signal(syscall.SIGCONT)
			_ = p.cmd.Process.Kill()
			<-p.done
		}
	})
	go func() {
		sc := bufio.NewScanner(pipe)
		for sc.Scan() {
			line := sc.Text()
			p.errMu.Lock()
			p.errb.WriteString(line + "\n")
			p.errMu.Unlock()
			if strings.HasPrefix(line, StopMarker+" ") || strings.HasPrefix(line, CrashMarker+" ") {
				p.marker <- line
			}
		}
		_ = p.cmd.Wait()
		p.code = p.cmd.ProcessState.ExitCode()
		close(p.done)
	}()
	return p
}

func (p *proc) stderr() string {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return p.errb.String()
}

func (h *harness) wait(p *proc) *proc {
	h.t.Helper()
	select {
	case <-p.done:
	case <-time.After(3 * time.Minute):
		h.t.Fatalf("child did not finish\nstderr:\n%s", p.stderr())
	}
	return p
}

func (h *harness) waitMarker(p *proc, prefix string) string {
	h.t.Helper()
	select {
	case m := <-p.marker:
		if !strings.HasPrefix(m, prefix) {
			h.t.Fatalf("marker %q, want %s", m, prefix)
		}
		return m
	case <-p.done:
		h.t.Fatalf("child ended before %s (exit %d)\nstderr:\n%s", prefix, p.code, p.stderr())
	case <-time.After(2 * time.Minute):
		h.t.Fatalf("no %s marker\nstderr:\n%s", prefix, p.stderr())
	}
	return ""
}

// run runs a child to its end and checks the exit code.
func (h *harness) run(wantCode int, env map[string]string, args ...string) *proc {
	h.t.Helper()
	p := h.wait(h.start(env, args...))
	if p.code != wantCode {
		h.t.Fatalf("%v exit %d, want %d\nstdout:\n%s\nstderr:\n%s", args, p.code, wantCode, p.out.String(), p.stderr())
	}
	return p
}

// killed runs a child that must die at the crash point.
func (h *harness) killed(crashAt string, args ...string) *proc {
	h.t.Helper()
	p := h.start(map[string]string{CrashEnv: crashAt}, args...)
	h.waitMarker(p, CrashMarker+" "+strings.SplitN(crashAt, ":", 2)[0])
	h.wait(p)
	if p.code == 0 {
		h.t.Fatalf("child with %s exited 0", crashAt)
	}
	return p
}

func refusedWith(t *testing.T, p *proc, want string) {
	t.Helper()
	var r struct {
		Refused string `json:"refused"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(p.out.Bytes(), &r); err != nil {
		t.Fatalf("refusal output %q: %v", p.out.String(), err)
	}
	if !strings.Contains(r.Refused+" "+r.Error, want) {
		t.Fatalf("refusal %q / %q, want %q", r.Refused, r.Error, want)
	}
}

func decodeOut[T any](t *testing.T, p *proc) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(p.out.Bytes(), &v); err != nil {
		t.Fatalf("output %q: %v", p.out.String(), err)
	}
	return v
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

// closeInvariants checks that the close output is exactly one copy of the
// cut and that the cut equals the entries posted to the month.
func (h *harness) closeInvariants() map[string]int64 {
	h.t.Helper()
	m := map[string]int64{
		"cutEntries":     h.int(`SELECT cut_entries FROM mecc.close_run WHERE month=$1`, month),
		"cutAmount":      h.int(`SELECT cut_amount::bigint FROM mecc.close_run WHERE month=$1`, month),
		"detailRows":     h.int(`SELECT count(*) FROM mecc.close_detail WHERE month=$1`, month),
		"detailAmount":   h.int(`SELECT coalesce(sum(amount),0)::bigint FROM mecc.close_detail WHERE month=$1`, month),
		"chunkEntries":   h.int(`SELECT coalesce(sum(entries),0) FROM mecc.close_chunk WHERE month=$1`, month),
		"totalRows":      h.int(`SELECT count(*) FROM mecc.close_total WHERE month=$1`, month),
		"totalEntries":   h.int(`SELECT entries FROM mecc.close_total WHERE month=$1`, month),
		"totalAmount":    h.int(`SELECT amount::bigint FROM mecc.close_total WHERE month=$1`, month),
		"postedEntries":  h.int(`SELECT count(*) FROM mecc.sales_entry WHERE posting_month=$1`, month),
		"postedAmount":   h.int(`SELECT coalesce(sum(amount),0)::bigint FROM mecc.sales_entry WHERE posting_month=$1`, month),
		"missingDetails": h.int(`SELECT count(*) FROM mecc.sales_entry s WHERE posting_month=$1 AND NOT EXISTS (SELECT 1 FROM mecc.close_detail d WHERE d.month=$1 AND d.entry_id=s.entry_id)`, month),
		"foreignDetails": h.int(`SELECT count(*) FROM mecc.close_detail d JOIN mecc.sales_entry s USING (entry_id) WHERE d.month=$1 AND s.posting_month<>$1`, month),
	}
	if m["totalRows"] != 1 || m["missingDetails"] != 0 || m["foreignDetails"] != 0 ||
		m["cutEntries"] != m["detailRows"] || m["detailRows"] != m["chunkEntries"] || m["detailRows"] != m["totalEntries"] || m["totalEntries"] != m["postedEntries"] ||
		m["cutAmount"] != m["detailAmount"] || m["detailAmount"] != m["totalAmount"] || m["totalAmount"] != m["postedAmount"] {
		h.t.Fatalf("close is not exactly one copy of the posted entries: %v", m)
	}
	return m
}

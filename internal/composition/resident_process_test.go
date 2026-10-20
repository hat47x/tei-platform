package composition

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestResidentProcessFixture runs only in a separate OS process. Its text
// protocol is test-only and does not define a product Subscription or ABI.
func TestResidentProcessFixture(t *testing.T) {
	mode := os.Getenv("TEI_RESIDENT_REFERENCE")
	if mode == "" {
		return
	}
	in := bufio.NewScanner(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	reply := func(value string) {
		fmt.Fprintln(out, value, os.Getpid())
		out.Flush()
	}
	if mode == "reject" {
		reply("NOT_READY")
	} else {
		reply("READY")
	}
	held := false
	for in.Scan() {
		switch in.Text() {
		case "HOLD":
			held = true
			reply("HELD")
		case "RELEASE":
			if !held {
				os.Exit(3)
			}
			held = false
			reply("DONE")
		case "PING":
			reply("PONG")
		case "STOP":
			return
		default:
			os.Exit(4)
		}
	}
	if in.Err() != nil {
		os.Exit(5)
	}
}

type residentProcess struct {
	cmd   *exec.Cmd
	input *bufio.Writer
	out   *bufio.Scanner
	pid   int
	done  bool
}

func startResidentProcess(t *testing.T, ctx context.Context, mode string) *residentProcess {
	t.Helper()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestResidentProcessFixture$")
	cmd.Env = append(os.Environ(), "TEI_RESIDENT_REFERENCE="+mode)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p := &residentProcess{cmd: cmd, input: bufio.NewWriter(stdin), out: bufio.NewScanner(stdout), pid: cmd.Process.Pid}
	t.Cleanup(func() {
		if !p.done {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			p.done = true
		}
	})
	return p
}

func (p *residentProcess) expect(expected string) error {
	if !p.out.Scan() {
		return fmt.Errorf("process %d ended before %s: %v", p.pid, expected, p.out.Err())
	}
	got := strings.Fields(p.out.Text())
	if len(got) != 2 || got[0] != expected || got[1] != strconv.Itoa(p.pid) {
		return fmt.Errorf("process %d: got %q, want %s", p.pid, p.out.Text(), expected)
	}
	return nil
}

func (p *residentProcess) exchange(command, expected string) error {
	if p.done {
		return errors.New("process already stopped")
	}
	if _, err := fmt.Fprintln(p.input, command); err != nil {
		return err
	}
	if err := p.input.Flush(); err != nil {
		return err
	}
	return p.expect(expected)
}

func (p *residentProcess) stop() error {
	if p.done {
		return nil
	}
	p.done = true
	if _, err := fmt.Fprintln(p.input, "STOP"); err != nil {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		return err
	}
	if err := p.input.Flush(); err != nil {
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		return err
	}
	return p.cmd.Wait()
}

// Real resident child processes demonstrate two distinct PIDs surviving the
// cutover. The host guard is synthetic; this is not a production Process ABI.
func TestResidentProcessesKeepOldInvocationThroughCutover(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	p1 := startResidentProcess(t, ctx, "ready")
	if err := p1.expect("READY"); err != nil {
		t.Fatal(err)
	}
	scope := NewScope()
	oldStops := 0
	if err := scope.Track(func() error { oldStops++; return p1.stop() }); err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(b(1, "v1"), scope)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewCoordinator(router, allow)
	if err != nil {
		t.Fatal(err)
	}
	a, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Release()
	if err := p1.exchange("HOLD", "HELD"); err != nil {
		t.Fatal(err)
	}
	var p2 *residentProcess
	newStops := 0
	lc := Lifecycle{
		Prepare: func(ctx context.Context, scope *Scope) error {
			p2 = startResidentProcess(t, ctx, "ready")
			return scope.Track(func() error { newStops++; return p2.stop() })
		},
		Ready: func(context.Context, Binding) error { return p2.expect("READY") },
	}
	if err := c.Replace(ctx, 1, b(2, "v2"), lc); err != nil {
		t.Fatal(err)
	}
	if p1.pid == p2.pid || p1.cmd.ProcessState != nil {
		t.Fatal("old process was not resident at cutover")
	}
	bLease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if a.Binding.ProviderID != "v1" || bLease.Binding.ProviderID != "v2" {
		t.Fatal("generation changed for in-flight invocation")
	}
	if err := p2.exchange("PING", "PONG"); err != nil {
		t.Fatal(err)
	}
	bLease.Release()
	drainCtx, stopDrain := context.WithCancel(ctx)
	stopDrain()
	if err := c.Reap(drainCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("active old lease did not block retirement: %v", err)
	}
	if oldStops != 0 || c.PendingRetirements() != 1 {
		t.Fatal("old process cleaned before its request completed")
	}
	if err := p1.exchange("RELEASE", "DONE"); err != nil {
		t.Fatal(err)
	}
	a.Release()
	if err := c.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if oldStops != 1 || newStops != 0 || !p1.done {
		t.Fatalf("incorrect cleanup: old=%d new=%d", oldStops, newStops)
	}
	if err := c.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if err := c.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if newStops != 1 || !p2.done {
		t.Fatal("new process was not cleaned")
	}
}

// A failed Ready must not publish a new generation or stop the old PID.
func TestResidentProcessReadyFailureRetainsOldProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	p1 := startResidentProcess(t, ctx, "ready")
	if err := p1.expect("READY"); err != nil {
		t.Fatal(err)
	}
	scope := NewScope()
	if err := scope.Track(p1.stop); err != nil {
		t.Fatal(err)
	}
	router, err := NewRouter(b(1, "v1"), scope)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewCoordinator(router, allow)
	if err != nil {
		t.Fatal(err)
	}
	var rejected *residentProcess
	cleanups := 0
	err = c.Replace(ctx, 1, b(2, "v2"), Lifecycle{
		Prepare: func(ctx context.Context, s *Scope) error {
			rejected = startResidentProcess(t, ctx, "reject")
			return s.Track(func() error { cleanups++; return rejected.stop() })
		},
		Ready: func(context.Context, Binding) error {
			if err := rejected.expect("NOT_READY"); err != nil {
				return err
			}
			return errors.New("provider did not become ready")
		},
	})
	if err == nil || cleanups != 1 || !rejected.done {
		t.Fatalf("failed stage was not cleaned: err=%v cleanup=%d", err, cleanups)
	}
	lease, err := c.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if lease.Binding.ProviderID != "v1" {
		t.Fatal("failed stage replaced old provider")
	}
	if err := p1.exchange("PING", "PONG"); err != nil {
		t.Fatal(err)
	}
	lease.Release()
	if c.PendingRetirements() != 0 {
		t.Fatal("failed stage retired old process")
	}
	if err := c.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := c.Reap(ctx); err != nil {
		t.Fatal(err)
	}
}

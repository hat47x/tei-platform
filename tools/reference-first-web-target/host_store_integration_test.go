//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceacceptance/hoststore"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// The Store is the Reader here, so these tests exercise a real, re-reading,
// file-backed Reader instead of a one-off synthetic publication. What stays
// synthetic: the issuer (the test verifier accepts everything, which is NOT a
// human Acceptance) and the effect Grant (syntheticEffectReader).

// Re-exec the test binary as the Web process. The child only reads; its
// verifier is the same test-only issuer policy as the parent's (accept every
// Decision), which is NOT a human Acceptance.
func TestCombinedWebTargetHostStoreChild(t *testing.T) {
	dir := os.Getenv("TEI_TEST_HOSTSTORE_DIR")
	if dir == "" {
		return
	}
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 {
		os.Exit(3)
	}
	store, err := hoststore.Open(dir, func(context.Context, hoststore.Decision) error { return nil })
	if err != nil {
		fmt.Fprintln(os.Stderr, "store open failed")
		os.Exit(3)
	}
	err = runCombinedWebTargetWithEffectGrant(context.Background(), store, &syntheticEffectReader{current: true}, os.Args[separator+1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

type storeWorld struct {
	root  string
	index sourceindex.Index
}

func loadStoreWorld(t *testing.T, root string) storeWorld {
	t.Helper()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	return storeWorld{root: root, index: index}
}

func storeDecision(ref string, world storeWorld) hoststore.Decision {
	return hoststore.Decision{
		Issuer: "test-only-issuer",
		Record: sourceacceptance.Record{
			ApplicationID: "order_entry", DecisionRef: ref, AuthorityRevision: "test-only-" + ref,
			ModelRevision: sourceindex.ModelRevision(world.index), Artifacts: world.index.Artifacts,
		},
	}
}

type storeChild struct {
	base, output string
	stop         func()
	log          *os.File
}

func freeAddress(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func storeChildArgs(root, address string) []string {
	return []string{"-test.run=^TestCombinedWebTargetHostStoreChild$", "--",
		"-workspace", root, "-application", "order_entry",
		"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
		"-target-revision", testTargetRevision, "-listen", address}
}

func storeChildCommand(ctx context.Context, t *testing.T, dir, root, address string) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, executable, storeChildArgs(root, address)...)
	command.Dir = filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	command.Env = append(os.Environ(), "TEI_TEST_HOSTSTORE_DIR="+dir)
	return command
}

// expectStartupRefusal runs the Web process and requires an exit before HTTP
// publication, with a public error that does not disclose the Store location.
func expectStartupRefusal(t *testing.T, dir, root, code string) {
	t.Helper()
	address := freeAddress(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := storeChildCommand(ctx, t, dir, root, address)
	// Source refusal must precede loading any Runner credential fixture.
	command.Args = append(command.Args, "-legacy-users", "does-not-exist", "-legacy-authorization", "does-not-exist")
	output, err := command.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 || !bytes.Contains(output, []byte(code)) ||
		bytes.Contains(output, []byte("TEI first Web Target projection:")) ||
		bytes.Contains(output, []byte(dir)) {
		t.Fatalf("expected %s before HTTP; err=%v output=%s", code, err, output)
	}
	client := &http.Client{Timeout: 300 * time.Millisecond}
	if response, err := client.Get("http://" + address + "/healthz"); err == nil {
		response.Body.Close()
		t.Fatal("refused Source published HTTP")
	}
}

func startStoreChild(t *testing.T, dir, root string) storeChild {
	t.Helper()
	address := freeAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	command := storeChildCommand(ctx, t, dir, root, address)
	logfile, err := os.CreateTemp(t.TempDir(), "child-*.log")
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	command.Stdout, command.Stderr = logfile, logfile
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	stop := func() { cancel(); <-done; logfile.Close() }
	t.Cleanup(stop)
	base := "http://" + address
	client := &http.Client{Timeout: 500 * time.Millisecond}
	for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(50 * time.Millisecond) {
		if response, err := client.Get(base + "/healthz"); err == nil {
			response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return storeChild{base: base, stop: stop, log: logfile}
			}
		}
		if time.Now().After(deadline) {
			raw, _ := os.ReadFile(logfile.Name())
			t.Fatalf("Web process did not publish HTTP: %s", raw)
		}
	}
}

const submitBody = `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`

func (c storeChild) submit(t *testing.T) (int, string) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Post(c.base+"/scenario/submit", "application/json", strings.NewReader(submitBody))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body)
}

func requireCompletedCommand(t *testing.T, status int, body string) {
	t.Helper()
	var observed struct {
		Result         commandScenarioResult `json:"result"`
		ExecutionError string                `json:"executionError"`
	}
	if err := json.Unmarshal([]byte(body), &observed); err != nil {
		t.Fatalf("response is not the Command result JSON: status=%d err=%v", status, err)
	}
	if status != http.StatusOK || observed.Result.Stage != "completed" || observed.Result.Command == nil ||
		observed.ExecutionError != "" || strings.Contains(body, "editor-secret") {
		t.Fatalf("expected a completed Command: status=%d stage=%q err=%q", status, observed.Result.Stage, observed.ExecutionError)
	}
}

func requireRefusedBeforeDispatch(t *testing.T, status int, body string) {
	t.Helper()
	if status != http.StatusServiceUnavailable || !strings.Contains(body, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE") ||
		strings.Contains(body, `"command"`) || strings.Contains(body, `"observation"`) || strings.Contains(body, "editor-secret") {
		t.Fatalf("expected a 503 before any Command: status=%d body=%s", status, body)
	}
}

func TestHostStoreBacksTheRealWebProcess(t *testing.T) {
	r1Root := workspaceWithUnitPriceRule(t)
	r0Root := filepath.Join(t.TempDir(), "r0")
	if err := os.CopyFS(r0Root, os.DirFS(referenceWorkspacePath(t))); err != nil {
		t.Fatal(err)
	}
	r2Root := filepath.Join(t.TempDir(), "r2")
	if err := os.CopyFS(r2Root, os.DirFS(r1Root)); err != nil {
		t.Fatal(err)
	}
	appFile := filepath.Join(r2Root, "applications", "order-entry", "application.tei")
	file, err := os.OpenFile(appFile, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString("\n")
	if closeErr := file.Close(); writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
	r0, r1, r2 := loadStoreWorld(t, r0Root), loadStoreWorld(t, r1Root), loadStoreWorld(t, r2Root)
	if sourceindex.ModelRevision(r1.index) == sourceindex.ModelRevision(r2.index) ||
		sourceindex.ModelRevision(r1.index) == sourceindex.ModelRevision(r0.index) {
		t.Fatal("R0/R1/R2 controls are not distinguishable")
	}

	issuer := func(context.Context, hoststore.Decision) error { return nil }
	newStore := func(t *testing.T) (*hoststore.Store, string) {
		dir := filepath.Join(t.TempDir(), "host-store")
		store, err := hoststore.Open(dir, issuer)
		if err != nil {
			t.Fatal(err)
		}
		return store, dir
	}

	t.Run("R1 runs without a synthetic Reader", func(t *testing.T) {
		store, dir := newStore(t)
		if err := store.Ingest(context.Background(), storeDecision("d1", r1)); err != nil {
			t.Fatal(err)
		}
		child := startStoreChild(t, dir, r1.root)
		status, body := child.submit(t)
		requireCompletedCommand(t, status, body)
	})

	t.Run("R0 and meaning-equivalent R2 are refused before HTTP", func(t *testing.T) {
		store, dir := newStore(t)
		if err := store.Ingest(context.Background(), storeDecision("d1", r1)); err != nil {
			t.Fatal(err)
		}
		expectStartupRefusal(t, dir, r0.root, "SOURCE_FINGERPRINT_MISMATCH")
		expectStartupRefusal(t, dir, r2.root, "SOURCE_FINGERPRINT_MISMATCH")
	})

	t.Run("revoke while running refuses the next request before dispatch", func(t *testing.T) {
		store, dir := newStore(t)
		if err := store.Ingest(context.Background(), storeDecision("d1", r1)); err != nil {
			t.Fatal(err)
		}
		child := startStoreChild(t, dir, r1.root)
		status, body := child.submit(t)
		requireCompletedCommand(t, status, body)
		if err := store.Revoke("order_entry", "d1"); err != nil {
			t.Fatal(err)
		}
		status, body = child.submit(t)
		requireRefusedBeforeDispatch(t, status, body)
		expectStartupRefusal(t, dir, r1.root, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE")
	})

	t.Run("replace refuses the running old World and starts the new one", func(t *testing.T) {
		store, dir := newStore(t)
		if err := store.Ingest(context.Background(), storeDecision("d1", r1)); err != nil {
			t.Fatal(err)
		}
		old := startStoreChild(t, dir, r1.root)
		status, body := old.submit(t)
		requireCompletedCommand(t, status, body)
		// A different Source is accepted; the running process holds the old
		// World and must not upgrade itself or keep dispatching.
		if err := store.Ingest(context.Background(), storeDecision("d2", r0)); err != nil {
			t.Fatal(err)
		}
		status, body = old.submit(t)
		requireRefusedBeforeDispatch(t, status, body)
		expectStartupRefusal(t, dir, r1.root, "SOURCE_FINGERPRINT_MISMATCH")
		fresh := startStoreChild(t, dir, r0.root)
		client := &http.Client{Timeout: time.Second}
		response, err := client.Get(fresh.base + "/healthz")
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("new current Source was refused: %d", response.StatusCode)
		}
	})

	t.Run("a new Decision for identical Source still stops the running process", func(t *testing.T) {
		store, dir := newStore(t)
		if err := store.Ingest(context.Background(), storeDecision("d1", r1)); err != nil {
			t.Fatal(err)
		}
		child := startStoreChild(t, dir, r1.root)
		if err := store.Ingest(context.Background(), storeDecision("d1b", r1)); err != nil {
			t.Fatal(err)
		}
		status, body := child.submit(t)
		requireRefusedBeforeDispatch(t, status, body)
	})

	t.Run("a corrupt or missing Store fails closed without disclosing its path", func(t *testing.T) {
		store, dir := newStore(t)
		expectStartupRefusal(t, dir, r1.root, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE")
		if err := store.Ingest(context.Background(), storeDecision("d1", r1)); err != nil {
			t.Fatal(err)
		}
		child := startStoreChild(t, dir, r1.root)
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 1 {
			t.Fatalf("expected exactly one Store file: %v %d", err, len(entries))
		}
		if err := os.WriteFile(filepath.Join(dir, entries[0].Name()), []byte("{broken"), 0o600); err != nil {
			t.Fatal(err)
		}
		status, body := child.submit(t)
		requireRefusedBeforeDispatch(t, status, body)
		if strings.Contains(body, dir) || strings.Contains(body, entries[0].Name()) {
			t.Fatalf("Store location leaked: %s", body)
		}
		expectStartupRefusal(t, dir, r1.root, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE")
	})
}

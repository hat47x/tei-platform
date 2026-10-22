//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// These tests run the SHIPPED binaries: the Web Target CLI and the issuer tool.
// The signing key is generated per test run; no key is stored in the repository.
// The shipped CLI has no effect Grant Reader, so an accepted Source still
// refuses to dispatch a Command.

func buildShipped(t *testing.T, repoRoot, pkg, name string) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), name)
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, pkg)
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, output)
	}
	return binary
}

func runTool(t *testing.T, binary string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	output, err := exec.CommandContext(ctx, binary, args...).CombinedOutput()
	return string(output), err
}

func TestShippedCLIServesASignedAcceptedSourceButDispatchesNoCommand(t *testing.T) {
	repoRoot := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	web := buildShipped(t, repoRoot, "./tools/reference-first-web-target", "first-web-target")
	admin := buildShipped(t, repoRoot, "./tools/reference-host-acceptance", "host-acceptance")

	r1 := workspaceWithUnitPriceRule(t)
	r0 := filepath.Join(t.TempDir(), "r0")
	if err := os.CopyFS(r0, os.DirFS(referenceWorkspacePath(t))); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	key, trust, store := filepath.Join(dir, "issuer.key"), filepath.Join(dir, "trust.json"), filepath.Join(dir, "store")
	decision := filepath.Join(dir, "decision.json")

	trustJSON, err := runTool(t, admin, "keygen", "-private-key", key, "-issuer", "test-issuer", "-key-id", "k1", "-application", "order_entry")
	if err != nil {
		t.Fatalf("keygen: %v %s", err, trustJSON)
	}
	if err := os.WriteFile(trust, []byte(trustJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := runTool(t, admin, "keygen", "-private-key", key, "-issuer", "x", "-key-id", "x", "-application", "x"); err == nil {
		t.Fatalf("keygen overwrote an existing key: %s", out)
	}
	if out, err := runTool(t, admin, "sign", "-workspace", r1, "-application", "order_entry", "-decision-ref", "d1",
		"-authority-revision", "test-only-issuer-rev-1", "-issuer", "test-issuer", "-key-id", "k1", "-private-key", key, "-out", decision); err != nil {
		t.Fatalf("sign: %v %s", err, out)
	}
	if out, err := runTool(t, admin, "ingest", "-store", store, "-trust", trust, "-decision", decision); err != nil {
		t.Fatalf("ingest: %v %s", err, out)
	}
	raw, err := os.ReadFile(decision)
	if err != nil || bytes.Contains(raw, []byte(strings.TrimSpace(string(mustRead(t, key))))) {
		t.Fatal("the signed Decision contains the private key")
	}

	webArgs := func(root, address string, extra ...string) []string {
		return append([]string{"-workspace", root, "-application", "order_entry", "-baseline-id", "B",
			"-baseline-revision", testBaselineRevision, "-target-revision", testTargetRevision, "-listen", address}, extra...)
	}
	refused := func(t *testing.T, code string, args ...string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, web, args...)
		command.Dir = repoRoot
		output, err := command.CombinedOutput()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || !bytes.Contains(output, []byte(code)) ||
			bytes.Contains(output, []byte("TEI first Web Target projection:")) || bytes.Contains(output, []byte(dir)) {
			t.Fatalf("expected %s before HTTP: err=%v output=%s", code, err, output)
		}
	}

	t.Run("signed R1 serves HTTP and still refuses the Command without an effect Grant", func(t *testing.T) {
		address := freeAddress(t)
		ctx, cancel := context.WithCancel(context.Background())
		command := exec.CommandContext(ctx, web, webArgs(r1, address, "-host-acceptance-store", store, "-host-acceptance-trust", trust)...)
		command.Dir = repoRoot
		logfile, err := os.CreateTemp(t.TempDir(), "web-*.log")
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
		defer func() { cancel(); <-done; logfile.Close() }()
		base := "http://" + address
		client := &http.Client{Timeout: time.Second}
		for deadline := time.Now().Add(15 * time.Second); ; time.Sleep(50 * time.Millisecond) {
			if response, err := client.Get(base + "/healthz"); err == nil {
				response.Body.Close()
				if response.StatusCode == http.StatusOK {
					break
				}
			}
			if time.Now().After(deadline) {
				t.Fatalf("shipped CLI did not serve the signed R1: %s", mustRead(t, logfile.Name()))
			}
		}
		response, err := (&http.Client{Timeout: 5 * time.Second}).Post(base+"/scenario/submit", "application/json", strings.NewReader(submitBody))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		if response.StatusCode != http.StatusUnprocessableEntity || !bytes.Contains(body, []byte("HOST_EFFECT_GRANT_UNAVAILABLE")) ||
			bytes.Contains(body, []byte(`"command": {`)) || bytes.Contains(body, []byte("editor-secret")) {
			t.Fatalf("shipped CLI dispatched a Command without an effect Grant: status=%d body=%s", response.StatusCode, body)
		}
		// Revoking through the issuer tool stops the running process.
		if out, err := runTool(t, admin, "revoke", "-store", store, "-trust", trust, "-application", "order_entry", "-decision-ref", "d1"); err != nil {
			t.Fatalf("revoke: %v %s", err, out)
		}
		after, err := client.Get(base + "/healthz")
		if err != nil {
			t.Fatal(err)
		}
		after.Body.Close()
		if after.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("revoked Source still served: %d", after.StatusCode)
		}
		// Re-accept for the following subtests with a new Decision.
		if out, err := runTool(t, admin, "sign", "-workspace", r1, "-application", "order_entry", "-decision-ref", "d2",
			"-authority-revision", "test-only-issuer-rev-2", "-issuer", "test-issuer", "-key-id", "k1", "-private-key", key, "-out", decision); err != nil {
			t.Fatalf("sign d2: %v %s", err, out)
		}
		if out, err := runTool(t, admin, "ingest", "-store", store, "-trust", trust, "-decision", decision); err != nil {
			t.Fatalf("ingest d2: %v %s", err, out)
		}
	})

	t.Run("R0 is refused before HTTP", func(t *testing.T) {
		refused(t, "SOURCE_FINGERPRINT_MISMATCH", webArgs(r0, freeAddress(t),
			"-host-acceptance-store", store, "-host-acceptance-trust", trust, "-legacy-users", "does-not-exist")...)
	})

	t.Run("a trust file without the issuer's key is refused", func(t *testing.T) {
		otherKey := filepath.Join(t.TempDir(), "other.key")
		otherTrust, err := runTool(t, admin, "keygen", "-private-key", otherKey, "-issuer", "test-issuer", "-key-id", "k1", "-application", "order_entry")
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(t.TempDir(), "other-trust.json")
		if err := os.WriteFile(path, []byte(otherTrust), 0o600); err != nil {
			t.Fatal(err)
		}
		refused(t, "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE", webArgs(r1, freeAddress(t),
			"-host-acceptance-store", store, "-host-acceptance-trust", path, "-legacy-users", "does-not-exist")...)
	})

	t.Run("flags are required together and cannot be combined with diagnostics", func(t *testing.T) {
		refused(t, "HOST_ACCEPTANCE_STORE_FLAGS", webArgs(r1, freeAddress(t), "-host-acceptance-store", store)...)
		refused(t, "SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT", webArgs(r1, freeAddress(t),
			"-host-acceptance-store", store, "-host-acceptance-trust", trust, "-diagnostic-unaccepted-source")...)
		refused(t, "HOST_ACCEPTANCE_STORE_UNAVAILABLE", webArgs(r1, freeAddress(t),
			"-host-acceptance-store", filepath.Join(dir, "missing"), "-host-acceptance-trust", trust)...)
	})
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runtimesnapshot "tei/tools/reference-runtime-snapshot"
)

// Opt-in: build and launch the actual combined and application-specific First
// Web Target binaries, then exercise the capture implementation over loopback.
// This is a reference/synthetic-Workspace observation, NOT an accepted B/R/T
// or a business/authorization/second-Target acceptance receipt.
func TestActualFirstWebTargetCaptureAcrossApplications(t *testing.T) {
	if os.Getenv("TEI_E1_LIVE_MANIFEST_CAPTURE") != "1" {
		t.Skip("set TEI_E1_LIVE_MANIFEST_CAPTURE=1 to build real First Web binaries and capture live manifests")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	requireClean := func() string {
		t.Helper()
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = root
		head, err := cmd.Output()
		if err != nil {
			t.Fatalf("identify source HEAD: %v", err)
		}
		cmd = exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all", "--ignored=matching")
		cmd.Dir = root
		status, err := cmd.Output()
		if err != nil || len(status) != 0 {
			t.Fatalf("source checkout is not clean: err=%v entries=%d", err, len(status))
		}
		return strings.TrimSpace(string(head))
	}
	start := requireClean()
	if len(start) != 40 {
		t.Fatalf("unexpected Git HEAD identity: %q", start)
	}
	build := func(tag string) string {
		t.Helper()
		target := filepath.Join(t.TempDir(), "first-web")
		args := []string{"build", "-trimpath", "-ldflags=-buildid=", "-o", target}
		if tag != "" {
			args = append(args, "-tags="+tag)
		}
		args = append(args, "./tools/reference-first-web-target")
		command := exec.Command("go", args...)
		command.Dir = root
		command.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=0")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("build First Web Target with tags %q: %v\n%s", tag, err, output)
		}
		return target
	}
	binaries := map[string]string{
		"combined": build(""),
		"order":    build("tei_order_only"),
		"repair":   build("tei_repair_only"),
	}
	const baseline = "B-capture-diagnostic"
	const revision = "R-capture-diagnostic"
	for _, tc := range []struct{ app, tagged string }{
		{"order_entry", "order"}, {"repair_intake", "repair"},
	} {
		t.Run(tc.app, func(t *testing.T) {
			observed := make(map[string][]byte)
			for _, mode := range []string{"combined", tc.tagged} {
				t.Run(mode, func(t *testing.T) {
					base := launchActualFirstWebForCapture(t, root, binaries[mode], tc.app, baseline, revision, start)
					args := []string{
						"-url", base + "/manifest.json", "-application", tc.app,
						"-baseline-id", baseline, "-baseline-revision", revision,
						"-target-revision", start,
					}
					var stdout, stderr bytes.Buffer
					if err := runCapture(args, &stdout, &stderr); err != nil {
						t.Fatalf("capture live first Web manifest: %v", err)
					}
					// Independently read the same live response, to confirm
					// capture preserves HTTP bytes. This is NOT a signed pin.
					response, err := (&http.Client{Timeout: 5 * time.Second}).Get(base + "/manifest.json")
					if err != nil {
						t.Fatal(err)
					}
					raw, readErr := io.ReadAll(io.LimitReader(response.Body, maxCaptureBytes+1))
					_ = response.Body.Close()
					if readErr != nil || response.StatusCode != http.StatusOK || len(raw) > maxCaptureBytes {
						t.Fatalf("read independently observed live manifest: HTTP %d err=%v bytes=%d", response.StatusCode, readErr, len(raw))
					}
					if !bytes.Equal(stdout.Bytes(), raw) {
						t.Fatalf("capture changed actual First Web JSON bytes (captured=%d raw=%d)", stdout.Len(), len(raw))
					}
					digest := sha256.Sum256(raw)
					hash := hex.EncodeToString(digest[:])
					if !strings.Contains(stderr.String(), "OBSERVED SHA-256: "+hash) || !strings.Contains(stderr.String(), "not an approved") {
						t.Fatal("capture observation was not correctly labeled")
					}
					snapshot, err := runtimesnapshot.Accept(stdout.Bytes(), runtimesnapshot.Identity{
						ApplicationID: tc.app, HandoverBaselineID: baseline,
						BaselineRepositoryRevision: revision, TargetBuildRevision: start,
						ManifestSHA256: hash,
					})
					if err != nil {
						t.Fatalf("read-only replay rejected observed manifest: %v", err)
					}
					for _, route := range []struct {
						method, path string
						status       int
					}{
						{http.MethodGet, "/manifest.json", http.StatusOK},
						{http.MethodGet, "/healthz", http.StatusNoContent},
						{http.MethodPost, "/scenario/submit", http.StatusNotFound},
					} {
						req, err := http.NewRequest(route.method, "http://example.invalid"+route.path, nil)
						if err != nil {
							t.Fatal(err)
						}
						recorder := httptest.NewRecorder()
						snapshot.Handler().ServeHTTP(recorder, req)
						if recorder.Code != route.status {
							t.Errorf("frozen route %s %s returned %d, want %d", route.method, route.path, recorder.Code, route.status)
						}
						if route.path == "/manifest.json" && !bytes.Equal(recorder.Body.Bytes(), raw) {
							t.Fatal("frozen route failed to preserve live manifest bytes")
						}
					}
					badArgs := append([]string(nil), args...)
					for i := 0; i < len(badArgs)-1; i++ {
						if badArgs[i] == "-application" {
							if tc.app == "order_entry" {
								badArgs[i+1] = "repair_intake"
							} else {
								badArgs[i+1] = "order_entry"
							}
						}
					}
					var failed bytes.Buffer
					if err := runCapture(badArgs, &failed, io.Discard); err == nil || failed.Len() != 0 {
						t.Fatal("wrong-Application capture failed to close before output")
					}
					observed[mode] = append([]byte(nil), raw...)
				})
			}
			if !bytes.Equal(observed["combined"], observed[tc.tagged]) {
				t.Fatalf("same-B/R/T %s manifest drifted between combined and tagged Go binaries", tc.app)
			}
		})
	}
	if end := requireClean(); end != start {
		t.Fatalf("source checkout changed during live capture: %s -> %s", start, end)
	}
}

func launchActualFirstWebForCapture(t *testing.T, root, binary, app, baseline, revision, target string) string {
	t.Helper()
	sock, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := sock.Addr().String()
	if err := sock.Close(); err != nil {
		t.Fatal(err)
	}
	logfile, err := os.CreateTemp(t.TempDir(), "first-web-capture-*.log")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logfile.Close() })
	command := exec.Command(binary,
		"-workspace", "reference/workspace",
		"-application", app, "-baseline-id", baseline,
		"-baseline-revision", revision, "-target-revision", target,
		"-legacy-users", "reference/legacy-auth/users.json",
		"-legacy-authorization", "reference/legacy-auth/authorization.json",
		"-listen", address,
	)
	command.Dir = root
	command.Stdout, command.Stderr = logfile, logfile
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	client := &http.Client{Timeout: 200 * time.Millisecond}
	base := "http://" + address
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		reply, err := client.Get(base + "/healthz")
		if err == nil {
			_ = reply.Body.Close()
			if reply.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	// Process startup can fail due to an unavailable/invalid reference fixture.
	// Logs are local-only and can contain runtime details; report no raw log data.
	t.Fatalf("%s First Web binary for %s was not healthy within 15s (log: %s)", filepath.Base(binary), app, filepath.Base(logfile.Name()))
	return ""
}

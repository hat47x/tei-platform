//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// The real combined Web listener and its Command runner must stop when the
// embedding Host lifetime ends. Reader/Decision in this test are synthetic;
// successful HTTP startup is not a real Human Acceptance or Host grant.
func TestCombinedWebHostLifetimeStopsListener(t *testing.T) {
	root := referenceWorkspacePath(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	accepted := sourceacceptance.Record{
		ApplicationID: "order_entry", DecisionRef: "test-only-host-runtime",
		AuthorityRevision: "test-only-host-lifetime",
		ModelRevision: sourceindex.ModelRevision(index), Artifacts: index.Artifacts,
	}
	reader := &requestSourceReader{record: accepted}
	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := reservation.Addr().String()
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Dir(filepath.Dir(root))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- runCombinedWebTarget(ctx, reader, []string{
			"-workspace", root, "-application", "order_entry",
			"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
			"-target-revision", testTargetRevision, "-listen", addr,
			"-legacy-users", filepath.Join(repoRoot, "reference", "legacy-auth", "users.json"),
			"-legacy-authorization", filepath.Join(repoRoot, "reference", "legacy-auth", "authorization.json"),
		})
	}()
	client := &http.Client{Timeout: 300 * time.Millisecond}
	url := "http://" + addr + "/healthz"
	var opened bool
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		select {
		case err := <-done:
			t.Fatalf("Host-bound Web exited before HTTP became ready: %v", err)
		default:
		}
		res, err := client.Get(url)
		if err == nil {
			res.Body.Close()
			if res.StatusCode == http.StatusOK {
				opened = true
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
	}
	if !opened {
		t.Fatal("Host-bound Web did not begin serving an accepted test Source")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Host cancellation must terminate listener with its cause: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Host cancellation did not stop the running Web listener")
	}
	res, err := client.Get(url)
	if err == nil {
		res.Body.Close()
		t.Fatalf("Host-terminated Web endpoint remained available: status %d", res.StatusCode)
	}
}

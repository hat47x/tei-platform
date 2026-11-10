package compositionprocessbridge

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func TestPersistentWorkerStartsPinnedBytesAfterSourceReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	dir := t.TempDir()
	old, oldDigest := buildPinnedFixture(t, dir, "pinned-old")
	next, nextDigest := buildPinnedFixture(t, dir, "path-replacement")
	if oldDigest == nextDigest {
		t.Fatal("reference releases have identical bytes")
	}
	args := []string{referenceArgvMarker, "--tei-reference-persistent"}
	cmd, closeImage, err := preparePersistentCommand(ctx, old, oldDigest, args)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = closeImage() })
	fd := cmd.ExtraFiles[0]
	if err := os.Rename(next, old); err != nil {
		t.Fatal(err)
	}
	worker, err := startPersistentCommand(ctx, cmd, "pinned-old")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = worker.stop() })
	if err := closeImage(); err != nil {
		t.Fatal(err)
	}
	if _, err := fd.Stat(); !errors.Is(err, os.ErrClosed) {
		t.Fatal("host retained its executable descriptor")
	}
	req := pruntime.Request{ProtocolVersion: 1, RequestID: "pinned-call", Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}, CapabilityID: core.MustCanonicalID("process.reference"), DescriptorFingerprint: "persistent-pinned-reference", Operation: "execute", Payload: json.RawMessage(`{"id":"intent","actionId":"reference/apply","idempotencyKey":"reference-key"}`)}
	response, err := worker.invoke(ctx, req, false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if response.Status != pruntime.ResponseOK || !strings.Contains(string(response.Payload), "revision=pinned-old") {
		t.Fatalf("replacement bytes executed: %+v", response)
	}
	if _, _, err := preparePersistentCommand(ctx, old, oldDigest, args); err == nil {
		t.Fatal("new preparation accepted replaced source against old digest")
	}
	if err := worker.stop(); err != nil {
		t.Fatal(err)
	}
	if worker.alive() {
		t.Fatal("child exit not confirmed")
	}
	t.Logf("verified image %s remained active at PID %d after source became %s", oldDigest, worker.cmd.Process.Pid, nextDigest)
}

func TestPersistentPinRejectsUnboundedAndNonExecutableSourceBeforeLaunch(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	path := filepath.Join(dir, "over-budget")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(pinnedReferenceMaxBytes + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	_ = f.Close()
	for _, source := range []string{path, dir} {
		cmd, closeImage, err := preparePersistentCommand(ctx, source, "sha256:"+strings.Repeat("0", 64), nil)
		if err == nil {
			if closeImage != nil {
				_ = closeImage()
			}
			t.Fatal("invalid source admitted")
		}
		if cmd != nil {
			t.Fatal("invalid source produced a launch command")
		}
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := preparePersistentCommand(cancelled, path, "sha256:"+strings.Repeat("0", 64), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled pin: %v", err)
	}
}

package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func TestPersistentProcessReplacementDrainFailureAndExplicitRecovery(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("verified persistent reference launch requires Linux")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	build := func(revision string) (string, string) {
		t.Helper()
		path := filepath.Join(t.TempDir(), "worker-"+revision)
		if runtime.GOOS == "windows" {
			path += ".exe"
		}
		cmd := exec.CommandContext(ctx, "go", "build", "-ldflags", "-X main.referenceRevision="+revision, "-o", path, "../../plugins/reference-process-executor")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("worker build: %v %s", err, out)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return path, fmt.Sprintf("sha256:%x", sha256.Sum256(b))
	}
	oldPath, oldDigest := build("old")
	newPath, newDigest := build("new")
	if oldDigest == newDigest {
		t.Fatal("release bytes are identical")
	}
	old, err := startPersistentWorker(ctx, oldPath, oldDigest, "old", false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = old.stop() })
	contract := composition.Contract{ID: "reference.persistent.deployment-intent", Version: "1", Shape: "existing-process-request-response; accepted-intent-not-durable; bounded-serial-reference-framing"}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{ID: "worker", Scope: "reference", Offers: contract}}, nil); err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("worker", "reference")
	if err != nil {
		t.Fatal(err)
	}
	binding := func(digest string, generation uint64) composition.Binding {
		return composition.Binding{ContractID: contract.ID, ContractVersion: "1", ProviderID: "worker", ArtifactDigest: digest, Generation: generation}
	}
	scope := composition.NewScope()
	if err := scope.Track(old.stop); err != nil {
		t.Fatal(err)
	}
	var revoked atomic.Bool
	coordinator, err := composition.NewSelectedCoordinator(registry, selected, binding(oldDigest, 1), scope, func(b composition.Binding) error {
		if revoked.Load() || b.ContractID != contract.ID || b.ContractVersion != "1" || b.ProviderID != "worker" || (b.ArtifactDigest != oldDigest && b.ArtifactDigest != newDigest) {
			return errors.New("host permission denied")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	request := func(id string) pruntime.Request {
		return pruntime.Request{ProtocolVersion: 1, RequestID: id, Plugin: plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}, CapabilityID: core.MustCanonicalID("process.reference"), DescriptorFingerprint: "persistent-reference-descriptor", Operation: "execute", Payload: json.RawMessage(`{"id":"reference-intent","actionId":"reference/apply","idempotencyKey":"reference-key"}`)}
	}
	assertResponse := func(response pruntime.Response, revision string) {
		t.Helper()
		if response.Status != pruntime.ResponseOK || !strings.Contains(string(response.Payload), "revision="+revision) || !strings.Contains(string(response.Payload), `"outcome":"accepted"`) {
			t.Fatalf("wrong response: %+v", response)
		}
	}
	lease, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		defer lease.Release()
		response, err := old.invoke(ctx, request("held-old"), true, func() { close(started) }, release)
		if err == nil && !strings.Contains(string(response.Payload), "revision=old") {
			err = errors.New("old request changed release")
		}
		finished <- err
	}()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("old child did not start held request")
	}
	var next *persistentWorker
	lifecycle := composition.Lifecycle{Prepare: func(ctx context.Context, s *composition.Scope) error {
		var err error
		next, err = startPersistentWorker(ctx, newPath, newDigest, "new", false)
		if err != nil {
			return err
		}
		t.Cleanup(func() { _ = next.stop() })
		return s.Track(next.stop)
	}, Ready: func(context.Context, composition.Binding) error { return nil }}
	if err := coordinator.ReplaceSelected(ctx, registry, selected, 1, binding(newDigest, 2), lifecycle); err != nil {
		t.Fatal(err)
	}
	if old.cmd.Process.Pid == next.cmd.Process.Pid {
		t.Fatal("new release reused old PID")
	}
	newLease, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	response, err := next.invoke(ctx, request("new-call"), false, nil, nil)
	newLease.Release()
	if err != nil {
		t.Fatal(err)
	}
	assertResponse(response, "new")
	drainCtx, stopDrain := context.WithTimeout(ctx, 30*time.Millisecond)
	if err := coordinator.Reap(drainCtx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("held drain incorrectly succeeded: %v", err)
	}
	stopDrain()
	if !old.alive() || coordinator.PendingRetirements() != 1 {
		t.Fatal("drain failure lost old process ownership")
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if old.alive() || coordinator.PendingRetirements() != 0 {
		t.Fatal("old PID was not reaped after completion")
	}
	t.Logf("old PID %d drained; new PID %d continued", old.cmd.Process.Pid, next.cmd.Process.Pid)
	bad := composition.Lifecycle{Prepare: func(ctx context.Context, s *composition.Scope) error {
		_, err := startPersistentWorker(ctx, newPath, newDigest, "new", true)
		return err
	}, Ready: func(context.Context, composition.Binding) error { return nil }}
	if err := coordinator.ReplaceSelected(ctx, registry, selected, 2, binding(newDigest, 3), bad); err == nil {
		t.Fatal("failed readiness published")
	}
	unchanged, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Binding.Generation != 2 {
		t.Fatal("failed readiness replaced active generation")
	}
	unchanged.Release()
	response, err = next.invoke(ctx, request("after-rejection"), false, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertResponse(response, "new")
	revoked.Store(true)
	if lease, err := coordinator.Acquire(); err == nil {
		lease.Release()
		t.Fatal("revoked host admitted call")
	}
	revoked.Store(false)
	// Cancel an actual held exchange. Its stream is disposed; no late response
	// can be mistaken for the next invocation and there is no automatic retry.
	callCtx, cancelCall := context.WithCancel(ctx)
	cancelLease, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	cancelStarted := make(chan struct{})
	cancelDone := make(chan error, 1)
	go func() {
		defer cancelLease.Release()
		_, err := next.invoke(callCtx, request("cancelled"), true, func() { close(cancelStarted) }, make(chan struct{}))
		cancelDone <- err
	}()
	select {
	case <-cancelStarted:
	case <-ctx.Done():
		t.Fatal("cancel probe did not enter child")
	}
	cancelCall()
	if err := <-cancelDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if next.alive() {
		t.Fatal("cancelled stream remained usable")
	}
	if _, err := next.invoke(ctx, request("late-retry"), false, nil, nil); err == nil {
		t.Fatal("dead process automatically retried")
	}
	if err := coordinator.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Reap(ctx); err == nil {
		t.Fatal("failed child exit was reported as clean retirement")
	}
	if coordinator.PendingRetirements() != 0 {
		t.Fatal("reported cleanup failure retained an already completed retirement")
	}
	if lease, err := coordinator.Acquire(); err == nil {
		lease.Release()
		t.Fatal("withdrawn process admitted call")
	}
	var restored *persistentWorker
	restore := composition.Lifecycle{Prepare: func(ctx context.Context, s *composition.Scope) error {
		var err error
		restored, err = startPersistentWorker(ctx, newPath, newDigest, "new", false)
		if err != nil {
			return err
		}
		t.Cleanup(func() { _ = restored.stop() })
		return s.Track(restored.stop)
	}, Ready: func(context.Context, composition.Binding) error { return nil }}
	if err := coordinator.RestoreSelected(ctx, registry, selected, 2, binding(newDigest, 3), restore); err != nil {
		t.Fatal(err)
	}
	if restored.cmd.Process.Pid == next.cmd.Process.Pid {
		t.Fatal("recovery reused terminated process")
	}
	recovered, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	response, err = restored.invoke(ctx, request("explicit-recovery"), false, nil, nil)
	recovered.Release()
	if err != nil {
		t.Fatal(err)
	}
	assertResponse(response, "new")
	if err := restored.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-restored.done:
	case <-ctx.Done():
		t.Fatal("crashed child exit was not confirmed")
	}
	crashLease, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	_, err = restored.invoke(ctx, request("after-crash"), false, nil, nil)
	crashLease.Release()
	if err == nil {
		t.Fatal("crashed child silently retried")
	}
	if err := coordinator.Withdraw(3); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Reap(ctx); err == nil {
		t.Fatal("unexpected child termination was reported as clean shutdown")
	}
	if restored.alive() {
		t.Fatal("restored PID leaked")
	}
	t.Logf("cancelled PID %d reported failed exit; explicit recovery PID %d reaped", next.cmd.Process.Pid, restored.cmd.Process.Pid)
}

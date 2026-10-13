package compositionprocessbridge

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// A selected generation must retain the exact executable descriptor while an
// old invocation is pinned; swapping a file name must not swap either image.
// This is reference-only Linux behavior, not production authority or sandboxing.
func TestLinuxPinnedGenerationKeepsOldExecutableUntilDrain(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	dir := t.TempDir()
	oldPath, oldDigest := buildPinnedFixture(t, ctx, dir, "old")
	newPath, newDigest := buildPinnedFixture(t, ctx, dir, "new")
	if oldDigest == newDigest {
		t.Fatal("two different executable revisions have the same digest")
	}
	oldHost, err := pinReferenceExecutable(ctx, oldPath, oldDigest)
	if err != nil {
		t.Fatal(err)
	}
	defer oldHost.Close()

	contract := composition.Contract{
		ID: "std.deployment.executor", Version: "1.0.0",
		Shape: "reference-execute-request-result-v1",
	}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "pinned-old", Scope: "reference", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("pinned-old", "reference")
	if err != nil {
		t.Fatal(err)
	}
	oldBinding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "pinned-old", ArtifactDigest: oldDigest, Generation: 1,
	}
	newBinding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "pinned-new", ArtifactDigest: newDigest, Generation: 2,
	}
	allowed := true
	denied := errors.New("Host permission revoked")
	guard := func(b composition.Binding) error {
		if !allowed || (b != oldBinding && b != newBinding) {
			return denied
		}
		return nil
	}
	oldScope := composition.NewScope()
	oldCleanups, newCleanups := 0, 0
	if err := oldScope.Track(func() error {
		oldCleanups++
		return oldHost.Close()
	}); err != nil {
		t.Fatal(err)
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, oldBinding, oldScope, guard)
	if err != nil {
		t.Fatal(err)
	}

	call := func(h *pinnedReferenceHost, binding composition.Binding, revision string) {
		t.Helper()
		lease, err := coordinator.Acquire()
		if err != nil {
			t.Fatal(err)
		}
		if lease.Binding != binding {
			lease.Release()
			t.Fatalf("unexpected generation binding: %+v", lease.Binding)
		}
		lease.Release()
		out := runPinnedReference(t, ctx, h, h.source, binding.ArtifactDigest)
		if !strings.Contains(out.Message, "revision="+revision) {
			t.Fatalf("wrong executable for %s: %+v", revision, out)
		}
	}
	call(oldHost, oldBinding, "old")

	// This lease represents an old request whose invocation has started.
	pinnedOld, err := coordinator.Acquire()
	if err != nil || pinnedOld.Binding != oldBinding {
		t.Fatalf("could not hold old in-flight invocation: %v", err)
	}
	snapshot, err = registry.Apply(snapshot.Revision, []composition.Provider{{
		ID: "pinned-new", Scope: "reference", Offers: contract,
	}}, []string{"pinned-old"})
	if err != nil {
		t.Fatal(err)
	}
	oldCalls := oldHost.launches
	if _, err := coordinator.Acquire(); !errors.Is(err, composition.ErrStaleRegistry) {
		t.Fatalf("stale registry admitted a fresh call: %v", err)
	}
	if oldHost.launches != oldCalls {
		t.Fatal("stale generation launched a process")
	}
	next, err := registry.Select("pinned-new", "reference")
	if err != nil || next.Revision != snapshot.Revision {
		t.Fatalf("cannot select next provider: %v", err)
	}

	var newHost *pinnedReferenceHost
	lifecycle := composition.Lifecycle{
		Prepare: func(ctx context.Context, scope *composition.Scope) error {
			var err error
			newHost, err = pinReferenceExecutable(ctx, newPath, newDigest)
			if err != nil {
				return err
			}
			return scope.Track(func() error {
				newCleanups++
				return newHost.Close()
			})
		},
		Ready: func(ctx context.Context, b composition.Binding) error {
			if b != newBinding || newHost == nil {
				return errors.New("new binding does not match prepared image")
			}
			v, err := newHost.Verify(ctx, plugin.Identity{}, plugin.Artifact{Digest: newDigest})
			if err != nil || !v.DigestVerified || v.SignatureVerified || v.ProvenanceChecked {
				return errors.New("candidate image failed digest verification")
			}
			return nil
		},
	}
	if err := coordinator.ReplaceSelected(ctx, registry, next, 1, newBinding, lifecycle); err != nil {
		t.Fatalf("cannot switch verified executable: %v", err)
	}
	if oldCleanups != 0 || newCleanups != 0 {
		t.Fatal("cutover cleaned a still-pinned generation")
	}
	// Both original paths now refer to different objects (or no object), but
	// both previously verified executable descriptors retain their own bytes.
	if err := os.Rename(newPath, oldPath); err != nil {
		t.Fatal(err)
	}
	call(newHost, newBinding, "new")
	retiring := runPinnedReference(t, ctx, oldHost, oldPath, oldDigest)
	if pinnedOld.Binding != oldBinding || !strings.Contains(retiring.Message, "revision=old") {
		t.Fatalf("old in-flight execution changed image: %+v", retiring)
	}
	if oldCleanups != 0 {
		t.Fatal("old descriptor closed while an old lease remained held")
	}
	pinnedOld.Release()
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if oldCleanups != 1 || newCleanups != 0 || oldHost.executable != nil {
		t.Fatalf("incorrect retiring cleanup: old=%d new=%d", oldCleanups, newCleanups)
	}
	oldCalls = oldHost.launches
	if _, err := oldHost.Invoke(ctx, pruntime.ProcessCommand{Executable: oldPath}, pruntime.Request{}); err == nil || oldHost.launches != oldCalls {
		t.Fatal("retired executable still launched after closing its descriptor")
	}

	allowed = false
	newCalls := newHost.launches
	if _, err := coordinator.Acquire(); !errors.Is(err, denied) || newHost.launches != newCalls {
		t.Fatalf("Host authority revocation bypassed: %v", err)
	}
	allowed = true
	call(newHost, newBinding, "new")
	if err := coordinator.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if _, err := coordinator.Acquire(); !errors.Is(err, composition.ErrUnbound) {
		t.Fatalf("withdrawn generation remained available: %v", err)
	}
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if oldCleanups != 1 || newCleanups != 1 || newHost.executable != nil {
		t.Fatalf("incorrect final cleanup: old=%d new=%d", oldCleanups, newCleanups)
	}
}

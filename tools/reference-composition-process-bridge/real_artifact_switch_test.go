package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/executor"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// This test checks distinct executable bytes, not merely distinct provider IDs.
// Its host adapter uses the existing process wire format as a reference only.
func TestSelectedCompositionSwitchesDistinctProcessArtifacts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference stdio process launcher forbids spaces in executable paths on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()

	dir := t.TempDir()
	type release struct {
		name, revision, path, digest string
		identity                     plugin.Identity
		plan                         pruntime.EnvironmentPlan
		adapter                      pruntime.DeploymentExecutor
	}
	launcher := &countingLauncher{}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	makeRelease := func(revision, version string) release {
		t.Helper()
		path := filepath.Join(dir, "process-"+revision)
		cmd := exec.CommandContext(ctx, "go", "build",
			"-ldflags", "-X main.referenceRevision="+revision,
			"-o", path, "../../plugins/reference-process-executor")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s executable: %v\n%s", revision, err, output)
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(bytes)
		digest := "sha256:" + hex.EncodeToString(sum[:])
		identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: version}
		plan := pruntime.EnvironmentPlan{
			Environment: core.MustCanonicalID("reference"),
			PluginLockFingerprint: "reference-artifact-switch-lock-" + revision,
			Plugins: []pruntime.PluginRuntime{{
				Plugin: identity, DescriptorFingerprint: "reference-process-descriptor-" + revision,
				Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
				Command: path, Args: []string{referenceArgvMarker},
				SandboxPolicy: plugin.SandboxNone,
				Artifact: plugin.Artifact{Digest: digest},
			}},
			Capabilities: []pruntime.CapabilityRuntime{{
				Plugin: identity, Capability: capability, Boundary: pruntime.BoundaryProcess,
			}},
		}
		plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
		return release{
			revision: revision, path: path, digest: digest, identity: identity, plan: plan,
			adapter: pruntime.DeploymentExecutor{
				Plan: plan, Registry: pruntime.Registry{
					Processes: launcher, Verifier: fileVerifier{path: path},
				}, CapabilityID: capabilityID,
			},
		}
	}
	old := makeRelease("old", "1.0.0")
	next := makeRelease("new", "1.0.1")
	if old.digest == next.digest {
		t.Fatal("different revisions unexpectedly produced the same executable bytes")
	}

	contract := composition.Contract{
		ID: "std.deployment.executor", Version: "1.0.0", Shape: "reference-execute-request-result-v1",
	}
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "provider-old", Scope: "reference", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("provider-old", "reference")
	if err != nil {
		t.Fatal(err)
	}
	allow := true
	denied := errors.New("host rejected artifact binding")
	allowedDigest := map[string]string{"provider-old": old.digest, "provider-new": next.digest}
	guard := func(b composition.Binding) error {
		if !allow || b.ContractID != contract.ID || b.ContractVersion != contract.Version ||
			allowedDigest[b.ProviderID] != b.ArtifactDigest {
			return denied
		}
		return nil
	}
	oldBinding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "provider-old", ArtifactDigest: old.digest, Generation: 1,
	}
	oldScope := composition.NewScope()
	oldCleanups, nextCleanups := 0, 0
	if err := oldScope.Track(func() error {
		oldCleanups++
		return os.Remove(old.path)
	}); err != nil {
		t.Fatal(err)
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, oldBinding, oldScope, guard)
	if err != nil {
		t.Fatal(err)
	}
	command := executor.Command{
		ID: "composition-artifact-update", ActionID: "sales_expand/apply",
		IdempotencyKey: "artifact-switch-key", Kind: string(deployment.ActionDatabaseRelease),
		Database: core.MustCanonicalID("sales_core"), Version: "1.0.0",
	}
	assertAccepted := func(got executor.Result, rel release) {
		t.Helper()
		if got.Outcome != executor.OutcomeAccepted ||
			got.ActionID != command.ActionID || got.IdempotencyKey != command.IdempotencyKey ||
			!strings.Contains(got.Message, "revision="+rel.revision) {
			t.Fatalf("unexpected response for %s: %+v", rel.revision, got)
		}
		pv := got.PluginInvocation
		if pv == nil || pv.PluginID != rel.identity.ID || pv.PluginVersion != rel.identity.Version ||
			pv.ArtifactDigest != rel.digest || !pv.DigestVerified || pv.SignatureVerified || pv.Sandboxed {
			t.Fatalf("incorrect artifact provenance for %s: %+v", rel.revision, pv)
		}
		if err := pruntime.ValidateDeploymentInvocation(rel.plan, capabilityID, pv); err != nil {
			t.Fatalf("invalid %s invocation evidence: %v", rel.revision, err)
		}
	}
	first, err := invokeSelected(ctx, coordinator, old.adapter, command)
	if err != nil {
		t.Fatal(err)
	}
	assertAccepted(first, old)

	// Keep the old lease while the registry and active generation change.
	pinned, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if pinned.Binding != oldBinding {
		t.Fatal("old in-flight invocation was bound to the wrong executable")
	}
	snapshot, err = registry.Apply(snapshot.Revision, []composition.Provider{{
		ID: "provider-new", Scope: "reference", Offers: contract,
	}}, []string{"provider-old"})
	if err != nil {
		t.Fatal(err)
	}
	before := launcher.calls
	if _, err := invokeSelected(ctx, coordinator, old.adapter, command); !errors.Is(err, composition.ErrStaleRegistry) {
		t.Fatalf("stale registry accepted a new invocation: %v", err)
	}
	if launcher.calls != before {
		t.Fatal("stale selection started a child process")
	}
	nextSelection, err := registry.Select("provider-new", "reference")
	if err != nil || nextSelection.Revision != snapshot.Revision {
		t.Fatalf("new provider selection failed: %v", err)
	}
	newBinding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "provider-new", ArtifactDigest: next.digest, Generation: 2,
	}
	stages := composition.Lifecycle{
		Prepare: func(_ context.Context, scope *composition.Scope) error {
			return scope.Track(func() error {
				nextCleanups++
				return os.Remove(next.path)
			})
		},
		Ready: func(ctx context.Context, b composition.Binding) error {
			if b != newBinding {
				return errors.New("staged binding does not match the next locked executable")
			}
			_, err := (fileVerifier{path: next.path}).Verify(ctx, next.identity, next.plan.Plugins[0].Artifact)
			return err
		},
	}
	if err := coordinator.ReplaceSelected(ctx, registry, nextSelection, 1, newBinding, stages); err != nil {
		t.Fatalf("replace selected executable: %v", err)
	}
	if oldCleanups != 0 || nextCleanups != 0 {
		t.Fatal("replacement disposed a generation with outstanding leases")
	}

	// A fresh invocation must neither fall back to the old executable nor
	// accept an old adapter under the new generation's lock.
	before = launcher.calls
	if _, err := invokeSelected(ctx, coordinator, old.adapter, command); err == nil {
		t.Fatal("old artifact was selected for a fresh new-generation invocation")
	}
	if launcher.calls != before {
		t.Fatal("mismatched old artifact reached the process launcher")
	}
	second, err := invokeSelected(ctx, coordinator, next.adapter, command)
	if err != nil {
		t.Fatal(err)
	}
	assertAccepted(second, next)
	if first.PluginInvocation.ArtifactDigest == second.PluginInvocation.ArtifactDigest {
		t.Fatal("replacement invoked the same executable bytes")
	}

	// The invocation holding an old lease can still use its old executable;
	// the old file is removed only after that lease has been released.
	retiring, err := old.adapter.Execute(ctx, command)
	if err != nil {
		t.Fatalf("in-flight old invocation lost its executable: %v", err)
	}
	assertAccepted(retiring, old)
	pinned.Release()
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if oldCleanups != 1 || nextCleanups != 0 {
		t.Fatalf("incorrect release counts: old=%d new=%d", oldCleanups, nextCleanups)
	}
	if _, err := os.Stat(old.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("retired executable was not removed: %v", err)
	}
	before = launcher.calls
	if _, err := old.adapter.Execute(ctx, command); err == nil || launcher.calls != before {
		t.Fatal("retired artifact was invoked after cleanup")
	}

	// The host retains authority after a successful binary replacement.
	allow = false
	before = launcher.calls
	if _, err := invokeSelected(ctx, coordinator, next.adapter, command); !errors.Is(err, denied) ||
		launcher.calls != before {
		t.Fatalf("host denial did not stop the new executable: %v", err)
	}
	allow = true
	bad := next.adapter
	bad.Plan = next.plan
	bad.Plan.Plugins = append([]pruntime.PluginRuntime(nil), next.plan.Plugins...)
	bad.Plan.Plugins[0].Artifact.Digest = old.digest
	if _, err := bad.Execute(ctx, command); err == nil || launcher.calls != before {
		t.Fatal("wrong artifact bytes passed locked digest verification")
	}
	if err := coordinator.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeSelected(ctx, coordinator, next.adapter, command); !errors.Is(err, composition.ErrUnbound) {
		t.Fatalf("withdrawn new provider accepted invocation: %v", err)
	}
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if nextCleanups != 1 {
		t.Fatalf("new generation cleanup ran %d times", nextCleanups)
	}
	// Both acknowledgments are only intent acceptance, not external state proof.
}

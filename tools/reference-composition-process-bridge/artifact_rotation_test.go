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

// TestDistinctProcessArtifactsRotateWithoutStaleDispatch exercises the same
// host-side reference bridge with two different executable files. A logical
// generation change with the same binary is insufficient to prove that stale
// plans cannot dispatch an older artifact after cutover.
func TestDistinctProcessArtifactsRotateWithoutStaleDispatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the existing reference process launcher does not support executable paths containing spaces on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	build := func(source, basename string) (string, string) {
		t.Helper()
		binary := filepath.Join(t.TempDir(), basename)
		cmd := exec.CommandContext(ctx, "go", "build", "-o", binary, source)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", source, err, output)
		}
		data, err := os.ReadFile(binary)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		return binary, "sha256:" + hex.EncodeToString(sum[:])
	}
	legacyBinary, legacyDigest := build("../reference-process-plugin", "legacy")
	nextBinary, nextDigest := build("../../plugins/reference-process-executor", "independent")
	if legacyDigest == nextDigest {
		t.Fatal("different reference implementation sources yielded the same binary digest")
	}

	launcher := &countingLauncher{}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind:    plugin.CapabilityDeploymentExecutor,
		ID:      capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	adapter := func(binary, digest, version string) pruntime.DeploymentExecutor {
		identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: version}
		plan := pruntime.EnvironmentPlan{
			Environment:              core.MustCanonicalID("reference"),
			PluginLockFingerprint:    "reference-process-lock",
			RuntimePolicyFingerprint: "reference-process-policy",
			Plugins: []pruntime.PluginRuntime{{
				Plugin:                identity,
				DescriptorFingerprint: "reference-process-descriptor",
				Boundary:              pruntime.BoundaryProcess,
				Trust:                 pruntime.TrustExternalProcess,
				Command:               binary,
				Args:                  []string{referenceArgvMarker},
				SandboxPolicy:         plugin.SandboxNone,
				Artifact:              plugin.Artifact{Digest: digest},
			}},
			Capabilities: []pruntime.CapabilityRuntime{{
				Plugin: identity, Capability: capability, Boundary: pruntime.BoundaryProcess,
			}},
		}
		return pruntime.DeploymentExecutor{
			Plan: plan,
			Registry: pruntime.Registry{
				Processes: launcher,
				Verifier:  fileVerifier{path: binary},
			},
			CapabilityID: capabilityID,
		}
	}
	legacy := adapter(legacyBinary, legacyDigest, "1.0.0")
	next := adapter(nextBinary, nextDigest, "2.0.0")
	command := executor.Command{
		ID:             "artifact-rotation",
		ActionID:       "sales_expand/apply",
		IdempotencyKey: "artifact-rotation-once",
		Kind:           string(deployment.ActionDatabaseRelease),
		Database:       core.MustCanonicalID("sales_core"),
		Version:        "1.0.0",
	}
	contract := composition.Contract{
		ID:      "std.deployment.executor",
		Version: "1.0.0",
		Shape:   "reference-execute-request-result-v1",
	}
	registry := composition.NewRegistry()
	snap, err := registry.Apply(0, []composition.Provider{{
		ID: "legacy", Scope: "reference", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	firstSelection, err := registry.Select("legacy", "reference")
	if err != nil {
		t.Fatal(err)
	}
	denied := errors.New("host denies this binding")
	allowed := true
	expectedDigests := map[string]string{
		"legacy": legacyDigest, "independent": nextDigest,
	}
	guard := func(binding composition.Binding) error {
		if !allowed || binding.ContractID != contract.ID ||
			binding.ContractVersion != contract.Version ||
			expectedDigests[binding.ProviderID] != binding.ArtifactDigest {
			return denied
		}
		return nil
	}
	original := composition.Binding{
		ContractID:      contract.ID,
		ContractVersion: contract.Version,
		ProviderID:      "legacy",
		ArtifactDigest:  legacyDigest,
		Generation:      1,
	}
	oldScope := composition.NewScope()
	oldCleanups := 0
	if err := oldScope.Track(func() error { oldCleanups++; return nil }); err != nil {
		t.Fatal(err)
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, firstSelection, original, oldScope, guard)
	if err != nil {
		t.Fatal(err)
	}

	checkResult := func(out executor.Result, plan pruntime.EnvironmentPlan, digest string) {
		t.Helper()
		if out.Outcome != executor.OutcomeAccepted ||
			out.ActionID != command.ActionID || out.IdempotencyKey != command.IdempotencyKey {
			t.Fatalf("bad executor acknowledgment: %+v", out)
		}
		if out.PluginInvocation == nil ||
			out.PluginInvocation.ArtifactDigest != digest ||
			!out.PluginInvocation.DigestVerified ||
			out.PluginInvocation.SignatureVerified ||
			out.PluginInvocation.Sandboxed {
			t.Fatalf("incorrect plugin provenance: %+v", out.PluginInvocation)
		}
		if err := pruntime.ValidateDeploymentInvocation(plan, capabilityID, out.PluginInvocation); err != nil {
			t.Fatalf("invalid locked invocation: %v", err)
		}
	}
	before, err := invokeSelected(ctx, coordinator, legacy, command)
	if err != nil {
		t.Fatalf("legacy plugin invocation: %v", err)
	}
	checkResult(before, legacy.Plan, legacyDigest)
	pinned, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Apply(snap.Revision, []composition.Provider{{
		ID: "independent", Scope: "reference", Offers: contract,
	}}, []string{"legacy"}); err != nil {
		t.Fatal(err)
	}
	afterSelection, err := registry.Select("independent", "reference")
	if err != nil {
		t.Fatal(err)
	}
	dispatched := launcher.calls
	if _, err := invokeSelected(ctx, coordinator, legacy, command); !errors.Is(err, composition.ErrStaleRegistry) {
		t.Fatalf("registry changed but old selection invoked: %v", err)
	}
	if launcher.calls != dispatched {
		t.Fatal("stale selection launched a child process")
	}

	nextBinding := original
	nextBinding.ProviderID = "independent"
	nextBinding.ArtifactDigest = nextDigest
	nextBinding.Generation = 2
	abortedCleanups := 0
	untrusted := nextBinding
	untrusted.ArtifactDigest = "sha256:" + strings.Repeat("0", 64)
	badLifecycle := composition.Lifecycle{
		Prepare: func(_ context.Context, scope *composition.Scope) error {
			return scope.Track(func() error { abortedCleanups++; return nil })
		},
		Ready: func(ctx context.Context, binding composition.Binding) error {
			_, err := (fileVerifier{path: nextBinary}).Verify(ctx, next.Plan.Plugins[0].Plugin, plugin.Artifact{Digest: binding.ArtifactDigest})
			return err
		},
	}
	if err := coordinator.ReplaceSelected(ctx, registry, afterSelection, 1, untrusted, badLifecycle); err == nil {
		t.Fatal("staged generation accepted an unverified executable digest")
	}
	if abortedCleanups != 1 || coordinator.PendingRetirements() != 0 || launcher.calls != dispatched {
		t.Fatal("failed replacement leaked a scope or dispatched a process")
	}

	newCleanups := 0
	lifecycle := composition.Lifecycle{
		Prepare: func(_ context.Context, scope *composition.Scope) error {
			return scope.Track(func() error { newCleanups++; return nil })
		},
		Ready: func(ctx context.Context, binding composition.Binding) error {
			if binding != nextBinding {
				return errors.New("unexpected generation binding")
			}
			verified, err := (fileVerifier{path: nextBinary}).Verify(ctx, next.Plan.Plugins[0].Plugin, next.Plan.Plugins[0].Artifact)
			if err != nil {
				return err
			}
			if !verified.DigestVerified {
				return errors.New("next executable digest was not verified")
			}
			return nil
		},
	}
	if err := coordinator.ReplaceSelected(ctx, registry, afterSelection, 1, nextBinding, lifecycle); err != nil {
		t.Fatalf("verified generation replacement failed: %v", err)
	}

	if _, err := invokeSelected(ctx, coordinator, legacy, command); err == nil {
		t.Fatal("old artifact dispatched after generation replacement")
	}
	if launcher.calls != dispatched {
		t.Fatal("replaced generation dispatched stale executable bytes")
	}
	after, err := invokeSelected(ctx, coordinator, next, command)
	if err != nil {
		t.Fatalf("independent plugin invocation: %v", err)
	}
	checkResult(after, next.Plan, nextDigest)
	if launcher.calls != dispatched+1 {
		t.Fatalf("expected only replacement process to be launched, calls=%d", launcher.calls)
	}
	if pinned.Binding.ArtifactDigest != legacyDigest || oldCleanups != 0 {
		t.Fatal("in-flight old generation was mutated or cleaned too early")
	}
	pinned.Release()
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if oldCleanups != 1 || newCleanups != 0 {
		t.Fatalf("bad generation cleanup after release: old=%d new=%d", oldCleanups, newCleanups)
	}

	allowed = false
	dispatched = launcher.calls
	if _, err := invokeSelected(ctx, coordinator, next, command); !errors.Is(err, denied) {
		t.Fatalf("host denied invocation still accepted: %v", err)
	}
	if launcher.calls != dispatched {
		t.Fatal("host-denied invocation launched an external process")
	}
	if err := coordinator.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeSelected(ctx, coordinator, next, command); !errors.Is(err, composition.ErrUnbound) {
		t.Fatalf("withdrawn provider accepted invocation: %v", err)
	}
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if newCleanups != 1 || launcher.calls != dispatched {
		t.Fatalf("withdrawal cleanup or process dispatch mismatch: cleanup=%d calls=%d", newCleanups, launcher.calls)
	}
}

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

// TestRejectedProcessArtifactStageRetainsVerifiedProvider covers a failed
// cutover, which a successful two-binary rotation cannot demonstrate.
// A failed readiness check must dispose only the candidate's resources and
// must leave the previously verified provider callable.
func TestRejectedProcessArtifactStageRetainsVerifiedProvider(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference launcher does not support executable paths containing spaces on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	build := func(revision string) (string, string) {
		t.Helper()
		path := filepath.Join(t.TempDir(), "process-"+revision)
		cmd := exec.CommandContext(ctx, "go", "build", "-ldflags", "-X main.referenceRevision="+revision,
			"-o", path, "../../plugins/reference-process-executor")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", revision, err, out)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(content)
		return path, "sha256:" + hex.EncodeToString(digest[:])
	}
	oldPath, oldDigest := build("active")
	candidatePath, candidateDigest := build("candidate")
	if oldDigest == candidateDigest {
		t.Fatal("reference builds were not distinct")
	}

	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := pruntime.EnvironmentPlan{
		Environment:              core.MustCanonicalID("reference"),
		PluginLockFingerprint:    "reference-staging-lock",
		RuntimePolicyFingerprint: "reference-staging-policy",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: "reference-process-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: oldPath, Args: []string{referenceArgvMarker},
			SandboxPolicy: plugin.SandboxNone,
			Artifact:      plugin.Artifact{Digest: oldDigest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity, Capability: capability, Boundary: pruntime.BoundaryProcess,
		}},
	}
	launcher := &countingLauncher{}
	adapter := pruntime.DeploymentExecutor{
		Plan: plan,
		Registry: pruntime.Registry{
			Processes: launcher, Verifier: fileVerifier{path: oldPath},
		},
		CapabilityID: capabilityID,
	}
	contract := composition.Contract{
		ID: "std.deployment.executor", Version: "1.0.0",
		Shape: "reference-execute-request-result-v1",
	}
	registry := composition.NewRegistry()
	_, err := registry.Apply(0, []composition.Provider{{
		ID: "stable", Scope: "reference", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selection, err := registry.Select("stable", "reference")
	if err != nil {
		t.Fatal(err)
	}
	oldBinding := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "stable", ArtifactDigest: oldDigest, Generation: 1,
	}
	cleanedOld := 0
	oldScope := composition.NewScope()
	if err := oldScope.Track(func() error { cleanedOld++; return nil }); err != nil {
		t.Fatal(err)
	}
	guard := func(b composition.Binding) error {
		if b != oldBinding {
			return errors.New("host rejected unknown artifact binding")
		}
		return nil
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selection, oldBinding, oldScope, guard)
	if err != nil {
		t.Fatal(err)
	}
	command := executor.Command{
		ID: "staging-rejection", ActionID: "sales_expand/apply",
		IdempotencyKey: "staging-rejection-key", Kind: string(deployment.ActionDatabaseRelease),
		Database: core.MustCanonicalID("sales_core"), Version: "1.0.0",
	}
	assertOldCallable := func() {
		t.Helper()
		result, err := invokeSelected(ctx, coordinator, adapter, command, oldBinding)
		if err != nil {
			t.Fatal(err)
		}
		if result.Outcome != executor.OutcomeAccepted || result.PluginInvocation == nil ||
			result.PluginInvocation.ArtifactDigest != oldDigest ||
			!result.PluginInvocation.DigestVerified || result.PluginInvocation.SignatureVerified {
			t.Fatalf("old provider lost verified invocation identity: %+v", result)
		}
		if err := pruntime.ValidateDeploymentInvocation(plan, capabilityID, result.PluginInvocation); err != nil {
			t.Fatal(err)
		}
	}
	assertOldCallable()
	held, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	untrusted := oldBinding
	untrusted.ArtifactDigest = "sha256:" + strings.Repeat("0", 64)
	untrusted.Generation = 2
	cleanupCandidate := 0
	badStage := composition.Lifecycle{
		Prepare: func(_ context.Context, scope *composition.Scope) error {
			return scope.Track(func() error {
				cleanupCandidate++
				return os.Remove(candidatePath)
			})
		},
		Ready: func(ctx context.Context, b composition.Binding) error {
			if b != untrusted {
				return errors.New("unexpected candidate binding")
			}
			_, err := (fileVerifier{path: candidatePath}).Verify(ctx, identity, plugin.Artifact{Digest: b.ArtifactDigest})
			return err
		},
	}
	count := launcher.calls
	if err := coordinator.ReplaceSelected(ctx, registry, selection, 1, untrusted, badStage); err == nil {
		t.Fatal("unverified candidate was published")
	}
	if cleanupCandidate != 1 || cleanedOld != 0 ||
		coordinator.PendingRetirements() != 0 || launcher.calls != count {
		t.Fatalf("failed staging leaked or dispatched: candidate cleanup=%d old cleanup=%d calls=%d", cleanupCandidate, cleanedOld, launcher.calls)
	}
	if _, err := os.Stat(candidatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected candidate was not removed: %v", err)
	}
	stillCurrent, err := coordinator.Acquire()
	if err != nil {
		t.Fatalf("rejected stage interrupted old serving: %v", err)
	}
	if stillCurrent.Binding != oldBinding || held.Binding != oldBinding {
		t.Fatal("failed stage changed an in-flight or active binding")
	}
	stillCurrent.Release()
	assertOldCallable()
	if launcher.calls != count+1 {
		t.Fatal("failed cutover did not preserve verified process invocation")
	}
	held.Release()
	if err := coordinator.Withdraw(1); err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if cleanedOld != 1 || cleanupCandidate != 1 {
		t.Fatalf("incorrect cleanup counts after withdrawal: old=%d rejected=%d", cleanedOld, cleanupCandidate)
	}
}

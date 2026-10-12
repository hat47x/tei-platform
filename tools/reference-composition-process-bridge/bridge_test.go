package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
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

const referenceArgvMarker = "--tei-reference-marker=argv value preserved"

// fileVerifier is reference-only: it checks the binary bytes at invocation
// time. It does not attest provenance, signatures or a production trust root.
type fileVerifier struct{ path string }

func (v fileVerifier) Verify(_ context.Context, _ plugin.Identity, artifact plugin.Artifact) (pruntime.ArtifactVerification, error) {
	f, err := os.Open(v.path)
	if err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	defer f.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, f); err != nil {
		return pruntime.ArtifactVerification{}, err
	}
	if got := "sha256:" + hex.EncodeToString(digest.Sum(nil)); got != artifact.Digest {
		return pruntime.ArtifactVerification{}, errors.New("plugin binary bytes do not match the locked digest")
	}
	return pruntime.ArtifactVerification{VerifierID: "reference-local-file-sha256", DigestVerified: true}, nil
}

type countingLauncher struct {
	calls    int
	delegate pruntime.StdioProcessLauncher
}

func (l *countingLauncher) Invoke(ctx context.Context, command pruntime.ProcessCommand, request pruntime.Request) (pruntime.Response, error) {
	l.calls++
	return l.delegate.Invoke(ctx, command, request)
}

// invokeSelected is deliberately a host-side reference adapter, not a second
// implementation of the plugin protocol or a production permission authority.
func invokeSelected(ctx context.Context, coordinator *composition.Coordinator, adapter pruntime.DeploymentExecutor, command executor.Command) (executor.Result, error) {
	lease, err := coordinator.Acquire()
	if err != nil {
		return executor.Result{}, err
	}
	defer lease.Release()
	if len(adapter.Plan.Plugins) != 1 || lease.Binding.ArtifactDigest != adapter.Plan.Plugins[0].Artifact.Digest {
		return executor.Result{}, errors.New("selected generation does not match locked plugin artifact")
	}
	return adapter.Execute(ctx, command)
}

func TestSelectedCompositionInvokesExistingProcessPlugin(t *testing.T) {
	for _, fixture := range []struct {
		name   string
		source string
	}{
		{name: "legacy", source: "../reference-process-plugin"},
		{name: "independent", source: "../../plugins/reference-process-executor"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			testSelectedCompositionInvokesProcessPlugin(t, fixture.source)
		})
	}
}

func testSelectedCompositionInvokesProcessPlugin(t *testing.T, source string) {
	if runtime.GOOS == "windows" {
		t.Skip("reference stdio process launcher forbids spaces in executable paths on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	binaryPath := filepath.Join(t.TempDir(), "tei-process-reference")
	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, source)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build existing process plugin: %v\n%s", err, output)
	}
	bytes, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	artifactDigest := "sha256:" + hex.EncodeToString(sum[:])
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := pruntime.EnvironmentPlan{
		Environment:              core.MustCanonicalID("reference"),
		PluginLockFingerprint:    "reference-process-lock",
		RuntimePolicyFingerprint: "reference-process-policy",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: "reference-process-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: binaryPath, Args: []string{referenceArgvMarker},
			SandboxPolicy: plugin.SandboxNone,
			Artifact:      plugin.Artifact{Digest: artifactDigest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity, Capability: capability, Boundary: pruntime.BoundaryProcess,
		}},
	}
	launcher := &countingLauncher{}
	adapter := pruntime.DeploymentExecutor{
		Plan: plan, Registry: pruntime.Registry{
			Processes: launcher, Verifier: fileVerifier{path: binaryPath},
		},
		CapabilityID: capabilityID,
	}
	contract := composition.Contract{
		ID: "std.deployment.executor", Version: "1.0.0", Shape: "reference-execute-request-result-v1",
	}
	registry := composition.NewRegistry()
	snap, err := registry.Apply(0, []composition.Provider{{
		ID: "process-v1", Scope: "reference", Offers: contract,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	firstSelection, err := registry.Select("process-v1", "reference")
	if err != nil {
		t.Fatal(err)
	}
	allowed := true
	denied := errors.New("host permission revoked")
	guard := func(b composition.Binding) error {
		if !allowed || b.ContractID != contract.ID || b.ArtifactDigest != artifactDigest {
			return denied
		}
		return nil
	}
	original := composition.Binding{
		ContractID: contract.ID, ContractVersion: contract.Version,
		ProviderID: "process-v1", ArtifactDigest: artifactDigest, Generation: 1,
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
	command := executor.Command{
		ID: "composition-process-1", ActionID: "sales_expand/apply",
		IdempotencyKey: "composition-process-key",
		Kind:           string(deployment.ActionDatabaseRelease),
		Database:       core.MustCanonicalID("sales_core"), Version: "1.0.0",
	}
	assertAccepted := func(result executor.Result) {
		t.Helper()
		if result.Outcome != executor.OutcomeAccepted ||
			result.ActionID != command.ActionID || result.IdempotencyKey != command.IdempotencyKey ||
			!strings.Contains(result.Message, referenceArgvMarker) {
			t.Fatalf("unexpected process executor result: %+v", result)
		}
		if result.PluginInvocation == nil ||
			result.PluginInvocation.ArtifactDigest != artifactDigest ||
			!result.PluginInvocation.DigestVerified ||
			result.PluginInvocation.SignatureVerified ||
			result.PluginInvocation.Boundary != string(pruntime.BoundaryProcess) ||
			result.PluginInvocation.Sandboxed {
			t.Fatalf("incorrect reference runtime provenance: %+v", result.PluginInvocation)
		}
		if err := pruntime.ValidateDeploymentInvocation(plan, capabilityID, result.PluginInvocation); err != nil {
			t.Fatalf("invalid locked plugin invocation: %v", err)
		}
	}
	first, err := invokeSelected(ctx, coordinator, adapter, command)
	if err != nil {
		t.Fatalf("invoke existing process provider: %v", err)
	}
	assertAccepted(first)
	pinned, err := coordinator.Acquire()
	if err != nil {
		t.Fatal(err)
	}
	nextSnapshot, err := registry.Apply(snap.Revision, []composition.Provider{{
		ID: "process-v2", Scope: "reference", Offers: contract,
	}}, []string{"process-v1"})
	if err != nil {
		t.Fatal(err)
	}
	before := launcher.calls
	if _, err := invokeSelected(ctx, coordinator, adapter, command); !errors.Is(err, composition.ErrStaleRegistry) {
		t.Fatalf("stale selected provider was not rejected: %v", err)
	}
	if launcher.calls != before {
		t.Fatal("stale selection dispatched an external process")
	}
	nextSelection, err := registry.Select("process-v2", "reference")
	if err != nil {
		t.Fatal(err)
	}
	if nextSelection.Revision != nextSnapshot.Revision {
		t.Fatal("selected an unexpected registry revision")
	}
	nextBinding := original
	nextBinding.ProviderID = "process-v2"
	nextBinding.Generation = 2
	nextCleanups := 0
	stages := composition.Lifecycle{
		Prepare: func(_ context.Context, s *composition.Scope) error {
			return s.Track(func() error { nextCleanups++; return nil })
		},
		Ready: func(ctx context.Context, b composition.Binding) error {
			if b != nextBinding {
				return errors.New("staged binding differs from accepted plan")
			}
			_, err := (fileVerifier{path: binaryPath}).Verify(ctx, identity, plan.Plugins[0].Artifact)
			return err // local byte check only; never dispatch business intent in Ready
		},
	}
	if err := coordinator.ReplaceSelected(ctx, registry, nextSelection, 1, nextBinding, stages); err != nil {
		t.Fatal(err)
	}
	second, err := invokeSelected(ctx, coordinator, adapter, command)
	if err != nil {
		t.Fatalf("invoke replacement process provider: %v", err)
	}
	assertAccepted(second)
	if pinned.Binding.ProviderID != "process-v1" || oldCleanups != 0 {
		t.Fatal("in-flight old generation was replaced or prematurely disposed")
	}
	pinned.Release()
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if oldCleanups != 1 || nextCleanups != 0 {
		t.Fatalf("incorrect retiring generation cleanup: old=%d new=%d", oldCleanups, nextCleanups)
	}

	before = launcher.calls
	allowed = false
	if _, err := invokeSelected(ctx, coordinator, adapter, command); !errors.Is(err, denied) {
		t.Fatalf("host guard was bypassed: %v", err)
	}
	if launcher.calls != before {
		t.Fatal("host denial dispatched an external process")
	}
	allowed = true
	badPlan := plan
	badPlan.Plugins = append([]pruntime.PluginRuntime(nil), plan.Plugins...)
	badPlan.Plugins[0].Artifact.Digest = "sha256:" + strings.Repeat("0", 64)
	badAdapter := adapter
	badAdapter.Plan = badPlan
	if _, err := badAdapter.Execute(ctx, command); err == nil {
		t.Fatal("incorrect process artifact digest was accepted")
	}
	if launcher.calls != before {
		t.Fatal("unverified process artifact was launched")
	}
	if err := coordinator.Withdraw(2); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeSelected(ctx, coordinator, adapter, command); !errors.Is(err, composition.ErrUnbound) {
		t.Fatalf("withdrawn provider accepted invocation: %v", err)
	}
	if err := coordinator.Reap(ctx); err != nil {
		t.Fatal(err)
	}
	if nextCleanups != 1 || launcher.calls != before {
		t.Fatalf("withdrawal cleanup or process dispatch mismatch: cleanup=%d launched=%d expected=%d", nextCleanups, launcher.calls, before)
	}
	// A process acknowledgement is not durable proof of the requested external
	// database state. The reference never performs the actual release.
}

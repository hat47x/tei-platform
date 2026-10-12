package compositionprocessbridge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

// referenceContractGate models an independently accepted Consumer contract.
// Registry eligibility is a declaration, not the Host's authority to accept
// new contract semantics. This is a test-local preflight, not a second resolver.
func referenceContractGate(selected composition.Selection, accepted composition.Contract, scope string) error {
	if selected.Contract != accepted || selected.Scope != scope {
		return errors.New("host has not accepted this exact contract and scope")
	}
	return nil
}

func TestProcessHostDoesNotInheritApprovalAcrossContractShapeDrift(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("reference stdio process launcher forbids spaces in executable paths on Windows")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	binary := filepath.Join(t.TempDir(), "contract-bound-process")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../plugins/reference-process-executor")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build independent process plugin: %v\n%s", err, output)
	}
	raw, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	digestBytes := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(digestBytes[:])

	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	capabilityID := core.MustCanonicalID("process.reference")
	capability := plugin.Capability{
		Kind: plugin.CapabilityDeploymentExecutor, ID: capabilityID,
		Actions: []deployment.ActionKind{deployment.ActionDatabaseRelease},
	}
	plan := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "reference-contract-gate-lock",
		Plugins: []pruntime.PluginRuntime{{
			Plugin: identity, DescriptorFingerprint: "reference-process-descriptor",
			Boundary: pruntime.BoundaryProcess, Trust: pruntime.TrustExternalProcess,
			Command: binary, Args: []string{referenceArgvMarker},
			SandboxPolicy: plugin.SandboxNone,
			Artifact:      plugin.Artifact{Digest: digest},
		}},
		Capabilities: []pruntime.CapabilityRuntime{{
			Plugin: identity, Capability: capability, Boundary: pruntime.BoundaryProcess,
		}},
	}
	plan.RuntimePolicyFingerprint = pruntime.PolicyFingerprint(plan)
	launcher := &countingLauncher{}
	adapter := pruntime.DeploymentExecutor{
		Plan: plan,
		Registry: pruntime.Registry{
			Processes: launcher, Verifier: fileVerifier{path: binary},
		},
		CapabilityID: capabilityID,
	}
	accepted := composition.Contract{
		ID: "std.deployment.executor", Version: "1.0.0", Shape: "reference-execute-request-result-v1",
	}
	const acceptedScope = "runtime-reference"
	registry := composition.NewRegistry()
	snapshot, err := registry.Apply(0, []composition.Provider{{
		ID: "contract-provider", Scope: acceptedScope, Offers: accepted,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := registry.Select("contract-provider", acceptedScope)
	if err != nil {
		t.Fatal(err)
	}
	hostAllowed := true
	denied := errors.New("host permission revoked")
	guard := func(b composition.Binding) error {
		if !hostAllowed || b.ContractID != accepted.ID ||
			b.ContractVersion != accepted.Version || b.ArtifactDigest != digest ||
			b.ProviderID != "contract-provider" {
			return denied
		}
		return nil
	}
	binding := composition.Binding{
		ContractID: accepted.ID, ContractVersion: accepted.Version,
		ProviderID: "contract-provider", ArtifactDigest: digest, Generation: 1,
	}
	coordinator, err := composition.NewSelectedCoordinator(registry, selected, binding, composition.NewScope(), guard)
	if err != nil {
		t.Fatal(err)
	}
	command := executor.Command{
		ID: "reference-contract-gate", ActionID: "sales_expand/apply",
		IdempotencyKey: "contract-gate-key", Kind: string(deployment.ActionDatabaseRelease),
		Database: core.MustCanonicalID("sales_core"), Version: "1.0.0",
	}
	invoke := func(s composition.Selection) (executor.Result, error) {
		if err := referenceContractGate(s, accepted, acceptedScope); err != nil {
			return executor.Result{}, err
		}
		return invokeSelected(ctx, coordinator, adapter, command, binding)
	}
	first, err := invoke(selected)
	if err != nil {
		t.Fatalf("accepted contract could not dispatch: %v", err)
	}
	if first.Outcome != executor.OutcomeAccepted || first.PluginInvocation == nil ||
		first.PluginInvocation.ArtifactDigest != digest || !first.PluginInvocation.DigestVerified {
		t.Fatalf("wrong accepted process invocation: %+v", first)
	}
	if err := pruntime.ValidateDeploymentInvocation(plan, capabilityID, first.PluginInvocation); err != nil {
		t.Fatal(err)
	}
	if launcher.calls != 1 {
		t.Fatal("accepted contract did not invoke exactly one process")
	}

	// The provider can publish the same ID and version with a different
	// request/result shape, even though the executable bytes are unchanged.
	// Neither a stale Selection nor a fresh one gains Consumer acceptance.
	changed := accepted
	changed.Shape = "incompatible-new-result-shape"
	snapshot, err = registry.Apply(snapshot.Revision, []composition.Provider{{
		ID: "contract-provider", Scope: acceptedScope, Offers: changed,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	before := launcher.calls
	if _, err := invoke(selected); !errors.Is(err, composition.ErrStaleRegistry) {
		t.Fatalf("old selection survived contract change: %v", err)
	}
	fresh, err := registry.Select("contract-provider", acceptedScope)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Contract != changed || fresh.Contract.ID != accepted.ID ||
		fresh.Contract.Version != accepted.Version {
		t.Fatal("test did not produce a same-identity contract shape drift")
	}
	if _, err := invoke(fresh); err == nil || !strings.Contains(err.Error(), "exact contract") {
		t.Fatalf("Host reused prior approval after shape drift: %v", err)
	}
	if launcher.calls != before {
		t.Fatal("unaccepted contract change started an external process")
	}

	// A provider in another scope cannot reuse the accepted contract and
	// executable. This applies even when the contract bytes are identical.
	snapshot, err = registry.Apply(snapshot.Revision, []composition.Provider{{
		ID: "other-scope-provider", Scope: "other-tenant", Offers: accepted,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Select("other-scope-provider", acceptedScope); !errors.Is(err, composition.ErrSelectionMismatch) {
		t.Fatalf("cross-scope provider was selectable: %v", err)
	}
	otherScope, err := registry.Select("other-scope-provider", "other-tenant")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := invoke(otherScope); err == nil || !strings.Contains(err.Error(), "exact contract") {
		t.Fatalf("Host used another scope's selected provider: %v", err)
	}
	if launcher.calls != before {
		t.Fatal("cross-scope selection started an external process")
	}

	// Once the accepted shape is offered again, the Host may stage a new
	// generation with explicit readiness and current permission checks.
	snapshot, err = registry.Apply(snapshot.Revision, []composition.Provider{{
		ID: "contract-provider", Scope: acceptedScope, Offers: accepted,
	}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	renewed, err := registry.Select("contract-provider", acceptedScope)
	if err != nil {
		t.Fatal(err)
	}
	if err := referenceContractGate(renewed, accepted, acceptedScope); err != nil {
		t.Fatal(err)
	}
	next := binding
	next.Generation = 2
	if err := coordinator.ReplaceSelected(ctx, registry, renewed, binding.Generation, next, composition.Lifecycle{
		Prepare: func(_ context.Context, s *composition.Scope) error {
			return s.Track(func() error { return nil })
		},
		Ready: func(ctx context.Context, b composition.Binding) error {
			if b != next {
				return fmt.Errorf("unexpected staged binding: %+v", b)
			}
			_, err := (fileVerifier{path: binary}).Verify(ctx, identity, plan.Plugins[0].Artifact)
			return err
		},
	}); err != nil {
		t.Fatalf("current accepted contract could not be restored: %v", err)
	}
	// Advance the Host-accepted invocation binding only after successful
	// readiness validation and generation publication.
	binding = next
	second, err := invoke(renewed)
	if err != nil || second.Outcome != executor.OutcomeAccepted || launcher.calls != before+1 {
		t.Fatalf("accepted fresh generation did not dispatch: result=%+v err=%v calls=%d", second, err, launcher.calls)
	}
	hostAllowed = false
	before = launcher.calls
	if _, err := invoke(renewed); !errors.Is(err, denied) || launcher.calls != before {
		t.Fatalf("current host revocation did not stop dispatch: %v", err)
	}
	// Signature, namespace publication and semantic compatibility evidence
	// are out of scope: these are separately managed Host authorities.
}

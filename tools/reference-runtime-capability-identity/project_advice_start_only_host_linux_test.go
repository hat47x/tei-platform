//go:build linux

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"tei/internal/core"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// This is a TEST Host's own current Environment Authority, not a new
// production Grant store. The SAME mutex protects the snapshot read,
// revocation completion, and physical OS Start admission.
type projectStartOnlyHost struct {
	mu          sync.RWMutex
	environment core.CanonicalID
	authority   RuntimeEnvironmentAuthority
	starts      atomic.Int32
}

func (h *projectStartOnlyHost) current(
	ctx context.Context, env core.CanonicalID,
) (RuntimeEnvironmentAuthority, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeEnvironmentAuthority{}, err
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if env != h.environment {
		return RuntimeEnvironmentAuthority{}, fmt.Errorf("foreign Host Environment")
	}
	return h.authority, nil
}

func (h *projectStartOnlyHost) startAdmission(
	plan RuntimeEnvironmentPlan, profile ExecutionProfile,
) pruntime.LinuxHostStartAdmission {
	// Plan and profile are bound by the Host, not extracted from a Process
	// Request. The callback checks the CURRENT Authority when Start is ready.
	return func(ctx context.Context, start func() error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		h.mu.RLock()
		defer h.mu.RUnlock()
		if err := verifyRuntimeEnvironmentPlan(
			plan, plan.RuntimeCapability, profile, h.authority,
		); err != nil {
			return fmt.Errorf("Host OS Start admission rejected current Environment Grant: %w", err)
		}
		h.starts.Add(1)
		return start()
	}
}

func (h *projectStartOnlyHost) revoke() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.authority.ExecutionGrants = nil
}

func (h *projectStartOnlyHost) restore(a RuntimeEnvironmentAuthority) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.authority = a
}

// The wrapper proves the old E39 read→Verify gap can be closed by RECHECKING
// inside the actual OS Start gate, without holding the Host Grant lock until
// the response completes. The independent compiled Go executable is invoked,
// and denied starts do not call the physical Start callback.
func TestProjectAdviceHostStartOnlyAdmissionClosesOldReadVerifyGap(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	planA, realization, _, pinned := projectAdvicePinnedFixture(t, path, digest)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	grantA := referenceAuthority(planA.RuntimeCapability, nil)
	a := &projectStartOnlyHost{environment: planA.Environment, authority: grantA}
	envB := core.MustCanonicalID("reference_runtime_b")
	grantB := referenceAuthority(planA.RuntimeCapability, nil)
	grantB.Environment = envB
	planB, err := buildRuntimeEnvironmentPlan(planA.RuntimeCapability, profile, grantB)
	if err != nil {
		t.Fatal(err)
	}
	b := &projectStartOnlyHost{environment: envB, authority: grantB}

	hostFor := func(t *testing.T, scope *projectStartOnlyHost, plan RuntimeEnvironmentPlan) (
		*pruntime.HostAdmittedPinnedLinuxProcess, pruntime.HostSandboxLauncher,
	) {
		t.Helper()
		admitted, err := pruntime.NewHostAdmittedPinnedLinuxProcess(
			pinned, scope.startAdmission(plan, profile),
		)
		if err != nil {
			t.Fatal(err)
		}
		return admitted, pruntime.HostSandboxLauncher{
			Network:    &recordingNetworkEnforcer{},
			Filesystem: &recordingFilesystemEnforcer{},
			Runner:     admitted,
		}
	}
	runnerA, sandboxA := hostFor(t, a, planA)
	runnerB, sandboxB := hostFor(t, b, planB)
	invoke := func(scope *projectStartOnlyHost, plan RuntimeEnvironmentPlan,
		verifier pruntime.ArtifactVerifier, sandbox pruntime.SandboxLauncher, id string,
	) (GenericInvocation, error) {
		return invokeWithCurrentHostEnvironment(
			context.Background(), scope.environment, plan, profile, scope.current,
			realization, verifier, nil, sandbox,
			"evaluate-advice", id,
			customeradvice.Input{CustomerCode: "VIP-123"},
		)
	}

	initial, err := invoke(a, planA, runnerA, sandboxA, "start-only-a-before-revoke")
	if err != nil {
		t.Fatal(err)
	}
	var advice customeradvice.Result
	if err := json.Unmarshal(initial.Response.Payload, &advice); err != nil {
		t.Fatal(err)
	}
	if advice.Kind != "review_suggested" || a.starts.Load() != 1 ||
		initial.Provenance.ArtifactVerification == nil ||
		!initial.Provenance.ArtifactVerification.DigestVerified {
		t.Fatalf("independent project Go executable was not admitted: %+v / %+v", advice, initial.Provenance)
	}

	// Deterministic revocation AFTER the first current read and inside the
	// artifact verifier, but BEFORE physical OS Start. The old E39 path
	// launched here; this gate MUST deny before invoking cmd.Start.
	hooked := &revokingProjectAdviceVerifier{
		held: pinned,
		onVerify: func() {
			a.revoke()
		},
	}
	if _, err := invoke(a, planA, hooked, sandboxA, "start-only-a-revoke-after-read"); err == nil ||
		!strings.Contains(err.Error(), "OS Start admission rejected") {
		t.Fatalf("physical Start accepted stale A Grant: %v", err)
	}
	if hooked.calls != 1 || a.starts.Load() != 1 {
		t.Fatalf("stale Plan reached OS Start: verifier calls=%d starts=%d", hooked.calls, a.starts.Load())
	}

	// B's separate Environment has the SAME signed/unsigned binary image
	// and exact Capability Ref; A's Grant removal must not revoke B.
	surviving, err := invoke(b, planB, runnerB, sandboxB, "start-only-b-independent")
	if err != nil {
		t.Fatalf("unrelated Host Environment B lost its Grant: %v", err)
	}
	if surviving.Provenance.Environment != envB || b.starts.Load() != 1 {
		t.Fatalf("independent Environment B did not pass its OWN start gate: %+v", surviving.Provenance)
	}

	// The next A call fails at current-plan read, even earlier than Start.
	before := a.starts.Load()
	if _, err := invoke(a, planA, runnerA, sandboxA, "start-only-a-next"); err == nil ||
		!strings.Contains(err.Error(), "Execution Grant") || a.starts.Load() != before {
		t.Fatalf("revoked A unexpectedly reached Start: %v", err)
	}
	a.restore(grantA)
	if _, err := invoke(a, planA, runnerA, sandboxA, "start-only-a-grant-restored"); err != nil {
		t.Fatalf("explicit Host Grant restoration failed: %v", err)
	}
	if a.starts.Load() != before+1 {
		t.Fatal("restored Host Grant did not enter OS Start")
	}
}

// A raw/closed image or absent Host callback cannot become an admitted runner.
// This does not grant a process authority merely because it has a valid hash.
func TestHostAdmittedPinnedLinuxProcessNeedsHostGate(t *testing.T) {
	if _, err := pruntime.NewHostAdmittedPinnedLinuxProcess(nil, nil); err == nil {
		t.Fatal("missing image/gate became an executable adapter")
	}
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	_, _, _, pinned := projectAdvicePinnedFixture(t, path, digest)
	if _, err := pruntime.NewHostAdmittedPinnedLinuxProcess(pinned, nil); err == nil {
		t.Fatal("missing trusted Host Start admission was accepted")
	}
}

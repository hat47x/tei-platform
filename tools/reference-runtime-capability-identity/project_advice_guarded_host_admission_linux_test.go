//go:build linux

package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// guardedProjectHost is a TEST-ONLY Host-owned Authority source and admission
// critical section. No Plugin, caller, or runtime Plan may mutate its Grant.
// Holding RLock through Invoke is deliberately conservative: unlike a
// production Start-only gate it also delays revocation while the child runs.
// It proves one correct ordering using the existing Host reader/invocation,
// not a new production permission store or a proposed long-running strategy.
type guardedProjectHost struct {
	mu          sync.RWMutex
	environment core.CanonicalID
	authority   RuntimeEnvironmentAuthority
}

func (h *guardedProjectHost) invoke(
	ctx context.Context,
	plan RuntimeEnvironmentPlan,
	profile ExecutionProfile,
	realization RuntimeProcessRealization,
	verifier pruntime.ArtifactVerifier,
	sandbox pruntime.SandboxLauncher,
	requestID string,
) (GenericInvocation, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return invokeWithCurrentHostEnvironment(
		ctx, h.environment, plan, profile,
		func(_ context.Context, env core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
			if env != h.environment {
				return RuntimeEnvironmentAuthority{}, errors.New("Host Environment differs from admitted scope")
			}
			return h.authority, nil
		},
		realization, verifier, nil, sandbox,
		"evaluate-advice", requestID,
		customeradvice.Input{CustomerCode: "VIP-123"},
	)
}

func (h *guardedProjectHost) revoke() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.authority.ExecutionGrants = nil
}

// admissionPauseVerifier is a deterministic test barrier after the Host
// Authority read but before Artifact verification and real OS process launch.
type admissionPauseVerifier struct {
	held    *pruntime.PinnedLinuxProcess
	entered chan struct{}
	release chan struct{}
}

func (v *admissionPauseVerifier) Verify(
	ctx context.Context, identity plugin.Identity, artifact plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	close(v.entered)
	select {
	case <-v.release:
	case <-ctx.Done():
		return pruntime.ArtifactVerification{}, ctx.Err()
	}
	return v.held.Verify(ctx, identity, artifact)
}

type guardedProjectResult struct {
	inv GenericInvocation
	err error
}

func awaitProjectAdmission(t *testing.T, description string, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

// #1845 showed a revoked Grant could be read and then a child started.
// With Host-owned serialized admission, revocation is never COMPLETE while
// an admitted request may still start. A different Environment has a
// separate gate. This reference holds the gate for the full child lifetime;
// it does NOT demonstrate an optimized Start-only admission implementation.
func TestProjectAdviceGuardedHostRevocationCannotCompleteBeforeRealProcess(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	plan, realization, host, pinned := projectAdvicePinnedFixture(t, path, digest)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	scopeA := &guardedProjectHost{
		environment: plan.Environment,
		authority:   referenceAuthority(plan.RuntimeCapability, nil),
	}
	scopeB := &guardedProjectHost{
		environment: core.MustCanonicalID("reference_runtime_b"),
		authority:   referenceAuthority(plan.RuntimeCapability, nil),
	}
	scopeB.authority.Environment = scopeB.environment
	planB, err := buildRuntimeEnvironmentPlan(plan.RuntimeCapability, profile, scopeB.authority)
	if err != nil {
		t.Fatal(err)
	}

	paused := &admissionPauseVerifier{
		held:    pinned,
		entered: make(chan struct{}),
		release: make(chan struct{}),
	}
	result := make(chan guardedProjectResult, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	go func() {
		inv, err := scopeA.invoke(ctx, plan, profile, realization, paused, host, "admitted-before-revoke")
		result <- guardedProjectResult{inv: inv, err: err}
	}()
	awaitProjectAdmission(t, "first request reaching artifact verification", paused.entered)

	revocationAttempted := make(chan struct{})
	revocationDone := make(chan struct{})
	go func() {
		close(revocationAttempted)
		scopeA.revoke()
		close(revocationDone)
	}()
	awaitProjectAdmission(t, "Environment A revocation attempt", revocationAttempted)
	// The admitted A request still holds the read-side critical section.
	select {
	case <-revocationDone:
		t.Fatal("Environment A revocation completed while the admitted request was paused")
	default:
	}

	// B has its OWN Host Environment gate. It can be revoked without waiting
	// for the unrelated A process to finish.
	scopeBRevoked := make(chan struct{})
	go func() {
		scopeB.revoke()
		close(scopeBRevoked)
	}()
	awaitProjectAdmission(t, "independent Environment B revocation", scopeBRevoked)
	if _, err := scopeB.invoke(ctx, planB, profile, realization, pinned, host, "revoked-scope-b"); err == nil ||
		!strings.Contains(err.Error(), "Execution Grant") {
		t.Fatalf("Environment B revocation was not independent: %v", err)
	}
	if _, err := scopeB.invoke(ctx, plan, profile, realization, pinned, host, "wrong-host-scope"); err == nil ||
		!strings.Contains(err.Error(), "trusted Host Environment") {
		t.Fatalf("foreign Environment unexpectedly reused A Plan: %v", err)
	}
	// Complete the admitted A request; OS Start and process Wait run BEFORE
	// release of the conservative reference admission lock.
	close(paused.release)
	var got guardedProjectResult
	select {
	case got = <-result:
	case <-time.After(15 * time.Second):
		t.Fatal("admitted real project process did not finish")
	}
	if got.err != nil {
		t.Fatalf("request admitted before revocation failed: %v", got.err)
	}
	var advice customeradvice.Result
	if err := json.Unmarshal(got.inv.Response.Payload, &advice); err != nil {
		t.Fatal(err)
	}
	if advice.Kind != "review_suggested" ||
		got.inv.Provenance.ArtifactVerification == nil ||
		!got.inv.Provenance.ArtifactVerification.DigestVerified {
		t.Fatalf("admitted request did not execute exact independent image: %+v %+v", advice, got.inv.Provenance)
	}
	awaitProjectAdmission(t, "Environment A revocation completion", revocationDone)

	// Any request after completed revocation fails BEFORE Artifact/Process.
	denied := &projectAdviceTestBinaryVerifier{path: path}
	_, err := scopeA.invoke(ctx, plan, profile, realization, denied, host, "after-revocation")
	if err == nil || !strings.Contains(err.Error(), "Execution Grant") {
		t.Fatalf("old Grant was accepted after revocation completed: %v", err)
	}
	if denied.calls != 0 {
		t.Fatal("revoked request reached Artifact Verifier")
	}
}

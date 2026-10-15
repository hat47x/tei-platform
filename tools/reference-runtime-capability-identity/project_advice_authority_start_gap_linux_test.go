//go:build linux

package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// revokingProjectAdviceVerifier is a deterministic diagnostic hook inserted
// AFTER the current Environment reader but BEFORE the real process launch.
// The held image is still checked by the existing PinnedLinuxProcess.
// It must not be used as a production verifier or a permission authority.
type revokingProjectAdviceVerifier struct {
	held     *pruntime.PinnedLinuxProcess
	onVerify func()
	calls    int
}

func (v *revokingProjectAdviceVerifier) Verify(
	ctx context.Context,
	identity plugin.Identity,
	artifact plugin.Artifact,
) (pruntime.ArtifactVerification, error) {
	v.calls++
	v.onVerify()
	return v.held.Verify(ctx, identity, artifact)
}

// #1835 rereads the current Host Environment Grant before artifact verification.
// This test intentionally revokes that Grant in the next synchronous phase,
// without sleeps or racing map accesses. A real independent Go process STILL
// executes under the old Plan because the reference reader does not retain an
// admission generation gate until OS process Start.
//
// A passing test records a KNOWN GAP, not a desirable or accepted runtime
// behavior. It does not disprove the separate Linux signer start gate (#1370).
func TestProjectAdviceCurrentGrantRevokedAfterReadStillReachesStart(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	plan, realization, host, held := projectAdvicePinnedFixture(t, path, digest)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	hostEnv := plan.Environment
	current := referenceAuthority(plan.RuntimeCapability, nil)
	readerCalls := 0
	readCurrent := func(_ context.Context, env core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
		readerCalls++
		if env != hostEnv {
			t.Fatal("Host reader was asked to read a foreign Environment")
		}
		return current, nil
	}
	verifier := &revokingProjectAdviceVerifier{
		held: held,
		onVerify: func() {
			// The earlier reader returned an independent snapshot of the
			// ExecutionGrant slice. Revoke the current Host grant here.
			current.ExecutionGrants = nil
		},
	}
	inv, err := invokeWithCurrentHostEnvironment(
		context.Background(), hostEnv, plan, profile, readCurrent,
		realization, verifier, nil, host,
		"evaluate-advice", "grant-revoked-between-read-and-start",
		customeradvice.Input{CustomerCode: "VIP-123"},
	)
	if err != nil {
		t.Fatalf("expected known reference race to be observable: %v", err)
	}
	if readerCalls != 1 || verifier.calls != 1 ||
		len(current.ExecutionGrants) != 0 {
		t.Fatalf("reference did not revoke precisely between the two phases: readers=%d verifications=%d", readerCalls, verifier.calls)
	}
	var result customeradvice.Result
	if err := json.Unmarshal(inv.Response.Payload, &result); err != nil {
		t.Fatal(err)
	}
	if result.Kind != "review_suggested" ||
		inv.Response.Status != pruntime.ResponseOK ||
		inv.Provenance.RuntimeCapability.Fingerprint != plan.RuntimeCapability.Fingerprint ||
		inv.Provenance.ArtifactVerification == nil ||
		!inv.Provenance.ArtifactVerification.DigestVerified {
		t.Fatalf("test did not reach the exact real process after current grant revocation: result=%+v provenance=%+v", result, inv.Provenance)
	}

	// The NEXT invocation does notice revocation and fails closed before the
	// artifact verifier or Host Sandbox. That does not heal the first gap.
	verifier.onVerify = func() {
		t.Fatal("revoked Environment reached artifact verification again")
	}
	_, err = invokeWithCurrentHostEnvironment(
		context.Background(), hostEnv, plan, profile, readCurrent,
		realization, verifier, nil, host,
		"evaluate-advice", "next-request-revoked",
		customeradvice.Input{CustomerCode: "VIP-123"},
	)
	if err == nil || !strings.Contains(err.Error(), "Execution Grant") ||
		readerCalls != 2 || verifier.calls != 1 {
		t.Fatalf("next invocation was not rejected before execution: err=%v readers=%d verifications=%d", err, readerCalls, verifier.calls)
	}
}

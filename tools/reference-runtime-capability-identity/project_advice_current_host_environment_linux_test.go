//go:build linux

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// Two project-facing Hosts may use the SAME locked Go artifact in distinct
// trusted Environments. A Scope selection from #1016 does not create either
// Environment Grant; the current Host reader owns that information.
func TestProjectAdviceCurrentHostEnvironmentSeparatesIndependentGrants(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	planA, realization, host, pinned := projectAdvicePinnedFixture(t, path, digest)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	envA := planA.Environment
	envB := core.MustCanonicalID("reference_runtime_b")
	grantA := referenceAuthority(planA.RuntimeCapability, nil)
	grantB := referenceAuthority(planA.RuntimeCapability, nil)
	grantB.Environment = envB
	planB, err := buildRuntimeEnvironmentPlan(planA.RuntimeCapability, profile, grantB)
	if err != nil {
		t.Fatal(err)
	}
	if planA.RuntimeCapability.Fingerprint != planB.RuntimeCapability.Fingerprint ||
		planA.RuntimePolicyFingerprint == planB.RuntimePolicyFingerprint {
		t.Fatal("same exact Plugin/Contract must allow independently bound Environments")
	}

	current := map[core.CanonicalID]RuntimeEnvironmentAuthority{
		envA: grantA,
		envB: grantB,
	}
	reads := []core.CanonicalID{}
	read := func(ctx context.Context, env core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
		reads = append(reads, env)
		value, ok := current[env]
		if !ok {
			return RuntimeEnvironmentAuthority{}, errors.New("not found")
		}
		return value, nil
	}
	call := func(env core.CanonicalID, plan RuntimeEnvironmentPlan, requestID string) (GenericInvocation, error) {
		return invokeWithCurrentHostEnvironment(
			context.Background(), env, plan, profile, read,
			realization, pinned, nil, host,
			"evaluate-advice", requestID,
			customeradvice.Input{CustomerCode: "VIP-123"},
		)
	}

	for _, tc := range []struct {
		env core.CanonicalID
		plan RuntimeEnvironmentPlan
		id string
	}{
		{envA, planA, "project-a"},
		{envB, planB, "project-b"},
	} {
		inv, err := call(tc.env, tc.plan, tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if inv.Provenance.Environment != tc.env ||
			inv.Provenance.RuntimeCapability.Fingerprint != planA.RuntimeCapability.Fingerprint ||
			inv.Provenance.ArtifactVerification == nil ||
			!inv.Provenance.ArtifactVerification.DigestVerified ||
			inv.Response.Status != pruntime.ResponseOK {
			t.Fatalf("unexpected Host provenance %+v", inv.Provenance)
		}
	}
	if len(reads) != 2 || reads[0] != envA || reads[1] != envB {
		t.Fatalf("Host Environment reader was bypassed: %+v", reads)
	}

	// Remove A's grant AFTER initial Plan construction. Reusing A's old
	// Plan must fail; B's current grant and the shared real process survive.
	revoked := grantA
	revoked.ExecutionGrants = nil
	current[envA] = revoked
	if _, err := call(envA, planA, "revoked-a"); err == nil ||
		!strings.Contains(err.Error(), "Execution Grant") {
		t.Fatalf("revoked Environment A reused stale Plan: %v", err)
	}
	if _, err := call(envB, planB, "surviving-b"); err != nil {
		t.Fatalf("unrelated Environment B was revoked: %v", err)
	}

	// Grant generation rotation also invalidates previously cached plans.
	rotated := grantB
	rotated.ExecutionGrants = append([]RuntimeExecutionGrant(nil), grantB.ExecutionGrants...)
	rotated.ExecutionGrants[0].AuthorityEvidenceFingerprint = fingerprint("rotated-B-authority")
	current[envB] = rotated
	if _, err := call(envB, planB, "rotated-b"); err == nil ||
		!strings.Contains(err.Error(), "current Environment Authority") {
		t.Fatalf("stale Authority generation was accepted: %v", err)
	}
}

// Pre-dispatch denials must not contact the binary verifier, Sandbox policy
// adapters, process runner or Resource Broker. These are *reference* counters,
// not proof of atomic revocation after this current Authority read.
func TestProjectAdviceCurrentHostEnvironmentFailsClosedBeforeArtifactDispatch(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	plan, realization, _, pinned := projectAdvicePinnedFixture(t, path, digest)
	_ = pinned // PinLinuxProcess closes with t.Cleanup; never used for these failures.
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	env := plan.Environment
	grant := referenceAuthority(plan.RuntimeCapability, nil)
	other := core.MustCanonicalID("reference_runtime_other")
	reader := func(_ context.Context, id core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
		if id != env {
			return RuntimeEnvironmentAuthority{}, errors.New("unauthorized environment")
		}
		return grant, nil
	}
	changed := grant
	changed.ExecutionGrants = nil
	duplicate := grant
	duplicate.ExecutionGrants = append(
		append([]RuntimeExecutionGrant(nil), grant.ExecutionGrants...),
		grant.ExecutionGrants[0],
	)
	wrong := grant
	wrong.Environment = other

	cases := []struct {
		name string
		hostEnv core.CanonicalID
		plan RuntimeEnvironmentPlan
		reader currentRuntimeEnvironmentAuthorityReader
		want string
	}{
		{"wrong-host-environment", other, plan, reader, "trusted Host Environment"},
		{"reader-absent", env, plan, nil, "reader is missing"},
		{"reader-unavailable", env, plan, func(context.Context, core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
			return RuntimeEnvironmentAuthority{}, errors.New("current source unavailable")
		}, "current source unavailable"},
		{"reader-wrong-environment", env, plan, func(context.Context, core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
			return wrong, nil
		}, "different Host Environment"},
		{"grant-revoked", env, plan, func(context.Context, core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
			return changed, nil
		}, "Execution Grant"},
		{"grant-duplicated", env, plan, func(context.Context, core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
			return duplicate, nil
		}, "exactly one"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verifier := &projectAdviceTestBinaryVerifier{path: path}
			network := &recordingNetworkEnforcer{}
			filesystem := &recordingFilesystemEnforcer{}
			runner := &recordingSandboxRunner{}
			host := pruntime.HostSandboxLauncher{
				Network: network, Filesystem: filesystem, Runner: runner,
			}
			_, err := invokeWithCurrentHostEnvironment(
				context.Background(), tc.hostEnv, tc.plan, profile, tc.reader,
				realization, verifier, nil, host,
				"evaluate-advice", "denied-"+tc.name,
				customeradvice.Input{CustomerCode: "VIP-123"},
			)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("bad current Host Authority was accepted: %v", err)
			}
			if verifier.calls != 0 || network.Calls != 0 ||
				filesystem.Calls != 0 || runner.Calls != 0 {
				t.Fatal("denied Host Environment reached artifact verification or execution")
			}
		})
	}
}

func TestProjectAdviceCurrentHostEnvironmentContextCancellationFailsClosed(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reads := 0
	_, err := invokeWithCurrentHostEnvironment(
		ctx, core.MustCanonicalID("reference_runtime"),
		RuntimeEnvironmentPlan{Environment: core.MustCanonicalID("reference_runtime")},
		ExecutionProfile{},
		func(context.Context, core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
			reads++
			return RuntimeEnvironmentAuthority{}, nil
		},
		RuntimeProcessRealization{Plugin: plugin.Identity{}},
		nil, nil, nil, "evaluate-advice", "cancelled", nil,
	)
	if !errors.Is(err, context.Canceled) || reads != 0 {
		t.Fatalf("cancelled call consulted Authority reader or dispatched: %v, reads=%d", err, reads)
	}
}

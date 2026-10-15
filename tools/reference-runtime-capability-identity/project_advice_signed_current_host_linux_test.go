//go:build linux

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// This exercises ONE exact, independently built project image across two
// Host Environments with separate grant lifetimes and a shared Host signer.
// Both the service-contract candidate and the two execution grants remain
// explicitly SYNTHETIC. The signer is generated in this test; it is not an
// issuer endorsement or a real deployment policy.
func TestProjectAdviceSignedImageAndCurrentEnvironmentAreIndependentGates(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	candidate.ArtifactDigest = digest
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	dispatch, err := genericProcessDispatchID(ref)
	if err != nil {
		t.Fatal(err)
	}

	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const signatureRef = "test-host/project-advice-signing-key"
	trust := pruntime.NewLinuxHostSignerTrust()
	if err := trust.Allow(signatureRef, public); err != nil {
		t.Fatal(err)
	}
	message, err := pruntime.LinuxPinnedSignatureMessage(candidate.Plugin, digest, signatureRef)
	if err != nil {
		t.Fatal(err)
	}
	proof := pruntime.LinuxPinnedSignatureProof{
		SignatureRef: signatureRef,
		Signature:    ed25519.Sign(private, message),
	}
	transport := pruntime.StdioProcessLauncher{Env: []string{}}
	pin := func(t *testing.T) *pruntime.PinnedLinuxProcess {
		t.Helper()
		pinned, err := pruntime.PinSignedLinuxProcess(
			context.Background(), trust, candidate.Plugin, path, digest, proof, transport,
		)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := pinned.Close(); err != nil {
				t.Error(err)
			}
		})
		return pinned
	}
	pinned := pin(t)
	realization := RuntimeProcessRealization{
		RuntimeCapabilityFingerprint: ref.Fingerprint,
		Plugin:                       candidate.Plugin,
		Artifact: plugin.Artifact{
			Digest:       digest,
			SignatureRef: signatureRef,
		},
		Command: pruntime.ProcessCommand{
			Executable: path,
			Args:       []string{"--tei-expected-dispatch=" + string(dispatch)},
		},
	}
	grantA := referenceAuthority(ref, nil)
	envA := grantA.Environment
	envB := core.MustCanonicalID("reference_runtime_b")
	grantB := referenceAuthority(ref, nil)
	grantB.Environment = envB
	planA, err := buildRuntimeEnvironmentPlan(ref, profile, grantA)
	if err != nil {
		t.Fatal(err)
	}
	planB, err := buildRuntimeEnvironmentPlan(ref, profile, grantB)
	if err != nil {
		t.Fatal(err)
	}
	if planA.RuntimeCapability.Fingerprint != planB.RuntimeCapability.Fingerprint ||
		planA.RuntimePolicyFingerprint == planB.RuntimePolicyFingerprint {
		t.Fatal("shared exact runtime identity lost independent Environment binding")
	}
	current := map[core.CanonicalID]RuntimeEnvironmentAuthority{
		envA: grantA,
		envB: grantB,
	}
	read := func(_ context.Context, env core.CanonicalID) (RuntimeEnvironmentAuthority, error) {
		a, ok := current[env]
		if !ok {
			return RuntimeEnvironmentAuthority{}, context.Canceled
		}
		return a, nil
	}
	makeHost := func(p *pruntime.PinnedLinuxProcess) pruntime.HostSandboxLauncher {
		return pruntime.HostSandboxLauncher{
			Network:    &recordingNetworkEnforcer{},
			Filesystem: &recordingFilesystemEnforcer{},
			Runner:     p,
		}
	}
	invoke := func(env core.CanonicalID, plan RuntimeEnvironmentPlan, p *pruntime.PinnedLinuxProcess, id string) (GenericInvocation, error) {
		return invokeWithCurrentHostEnvironment(
			context.Background(), env, plan, profile, read,
			realization, p, nil, makeHost(p),
			"evaluate-advice", id,
			customeradvice.Input{CustomerCode: "VIP-123"},
		)
	}
	check := func(env core.CanonicalID, plan RuntimeEnvironmentPlan, p *pruntime.PinnedLinuxProcess, id string) {
		t.Helper()
		inv, err := invoke(env, plan, p, id)
		if err != nil {
			t.Fatal(err)
		}
		var result customeradvice.Result
		if err := json.Unmarshal(inv.Response.Payload, &result); err != nil {
			t.Fatal(err)
		}
		if result.Kind != "review_suggested" ||
			inv.Provenance.Environment != env ||
			inv.Provenance.ArtifactVerification == nil ||
			!inv.Provenance.ArtifactVerification.DigestVerified ||
			!inv.Provenance.ArtifactVerification.SignatureVerified ||
			inv.Provenance.ArtifactVerification.ProvenanceChecked ||
			inv.Provenance.RuntimeCapability.ArtifactDigest != digest {
			t.Fatalf("wrong signed project result or provenance: %+v / %+v", result, inv.Provenance)
		}
	}
	check(envA, planA, pinned, "signed-a")
	check(envB, planB, pinned, "signed-b")

	// Environment revocation is scoped; signature trust survives for B.
	revokedA := grantA
	revokedA.ExecutionGrants = nil
	current[envA] = revokedA
	if _, err := invoke(envA, planA, pinned, "signed-a-revoked"); err == nil ||
		!strings.Contains(err.Error(), "Execution Grant") {
		t.Fatalf("A reused revoked Environment grant: %v", err)
	}
	check(envB, planB, pinned, "signed-b-survives-a")

	// Host signer trust is shared by both Environments; revoking it denies
	// subsequent execution even if B still has a current Environment grant.
	if !trust.Revoke(signatureRef) {
		t.Fatal("test Host signer was not revoked")
	}
	if _, err := invoke(envB, planB, pinned, "signed-b-revoked-key"); err == nil ||
		!strings.Contains(err.Error(), "Host signer trust was revoked") {
		t.Fatalf("B used a revoked signed image: %v", err)
	}
	// Reinstalling the same public key advances the signer generation.
	// The previously pinned image MUST NOT regain permission implicitly.
	if err := trust.Allow(signatureRef, public); err != nil {
		t.Fatal(err)
	}
	if _, err := invoke(envB, planB, pinned, "signed-b-old-generation"); err == nil ||
		!strings.Contains(err.Error(), "Host signer trust was revoked") {
		t.Fatalf("old pinned signer epoch was revived: %v", err)
	}
	fresh := pin(t)
	check(envB, planB, fresh, "signed-b-repinned")
}

func TestProjectAdviceSignedImageRejectsUntrustedOrMismatchedProof(t *testing.T) {
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	trust := pruntime.NewLinuxHostSignerTrust()
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const ref = "test-host/project-advice"
	message, err := pruntime.LinuxPinnedSignatureMessage(candidate.Plugin, digest, ref)
	if err != nil {
		t.Fatal(err)
	}
	proof := pruntime.LinuxPinnedSignatureProof{
		SignatureRef: ref,
		Signature:    ed25519.Sign(private, message),
	}
	pin := func(identity plugin.Identity, evidence pruntime.LinuxPinnedSignatureProof) error {
		p, err := pruntime.PinSignedLinuxProcess(
			context.Background(), trust, identity, path, digest, evidence,
			pruntime.StdioProcessLauncher{Env: []string{}},
		)
		if p != nil {
			_ = p.Close()
			t.Fatal("untrusted evidence unexpectedly produced a runnable image")
		}
		return err
	}
	if err := pin(candidate.Plugin, proof); err == nil ||
		!strings.Contains(err.Error(), "not trusted") {
		t.Fatalf("unknown issuer was accepted: %v", err)
	}
	if err := trust.Allow(ref, public); err != nil {
		t.Fatal(err)
	}
	corrupt := proof
	corrupt.Signature = append([]byte(nil), proof.Signature...)
	corrupt.Signature[0] ^= 0xff
	if err := pin(candidate.Plugin, corrupt); err == nil ||
		!strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("modified signature was accepted: %v", err)
	}
	foreign := plugin.Identity{
		ID:      core.MustCanonicalID("tei.reference.other_project"),
		Version: candidate.Plugin.Version,
	}
	if err := pin(foreign, proof); err == nil ||
		!strings.Contains(err.Error(), "signature verification failed") {
		t.Fatalf("cross-Plugin signature was accepted: %v", err)
	}
}

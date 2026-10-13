//go:build linux

package compositionprocessbridge

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func TestProductPinnedLinuxEd25519SignatureBindsExecutedBytes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dir := t.TempDir()
	approved, digest := buildPinnedFixture(t, ctx, dir, "approved")
	replacement, replacementDigest := buildPinnedFixture(t, ctx, dir, "replaced")
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	const ref = "host-trusted-signer/2026-10"
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	message, err := pruntime.LinuxPinnedSignatureMessage(identity, digest, ref)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(private, message)
	trust := pruntime.NewLinuxHostSignerTrust()
	if err := trust.Allow(ref, public); err != nil {
		t.Fatal(err)
	}
	proof := pruntime.LinuxPinnedSignatureProof{
		SignatureRef: ref, Signature: signature,
	}
	host, err := pruntime.PinSignedLinuxProcess(ctx, trust, identity, approved, digest, proof, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatalf("valid Host-selected signer was rejected: %v", err)
	}
	defer host.Close()

	accepted, err := invokeProductPinned(t, ctx, host, approved, digest, ref, "", plugin.SandboxNone)
	if err != nil || !strings.Contains(accepted.Message, "revision=approved") {
		t.Fatalf("approved signer and image were not executed: %+v %v", accepted, err)
	}
	if accepted.PluginInvocation == nil || !accepted.PluginInvocation.DigestVerified ||
		!accepted.PluginInvocation.SignatureVerified || accepted.PluginInvocation.Sandboxed {
		t.Fatalf("signature evidence was not propagated accurately: %+v", accepted.PluginInvocation)
	}
	if err := os.Rename(replacement, approved); err != nil {
		t.Fatal(err)
	}
	// The name now points at different bytes, but both the signed digest and
	// the launcher still refer to the originally pinned executable.
	next, err := invokeProductPinned(t, ctx, host, approved, digest, ref, "", plugin.SandboxNone)
	if err != nil || !strings.Contains(next.Message, "revision=approved") ||
		strings.Contains(next.Message, "revision=replaced") {
		t.Fatalf("signed execution changed after source replacement: %+v %v", next, err)
	}
	if _, err := invokeProductPinned(t, ctx, host, approved, digest, "host-trusted-signer/other", "", plugin.SandboxNone); err == nil {
		t.Fatal("incorrect signature reference accepted")
	}
	if _, err := invokeProductPinned(t, ctx, host, approved, digest, "", "", plugin.SandboxNone); err == nil {
		t.Fatal("signature requirement silently downgraded")
	}
	if _, err := invokeProductPinned(t, ctx, host, approved, digest, ref, "attestation-without-host-validation", plugin.SandboxNone); err == nil {
		t.Fatal("signed digest incorrectly treated as verified provenance")
	}
	if _, err := invokeProductPinned(t, ctx, host, approved, replacementDigest, ref, "", plugin.SandboxNone); err == nil {
		t.Fatal("different artifact digest was accepted")
	}
}

func TestProductPinnedLinuxEd25519SignatureRejectsForgedProofs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	binary, digest := buildPinnedFixture(t, ctx, t.TempDir(), "approved")
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	otherPublic, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const ref = "host-trusted-signer/2026-10"
	message, err := pruntime.LinuxPinnedSignatureMessage(identity, digest, ref)
	if err != nil {
		t.Fatal(err)
	}
	trust := pruntime.NewLinuxHostSignerTrust()
	if err := trust.Allow(ref, public); err != nil {
		t.Fatal(err)
	}
	otherTrust := pruntime.NewLinuxHostSignerTrust()
	if err := otherTrust.Allow(ref, otherPublic); err != nil {
		t.Fatal(err)
	}
	proof := pruntime.LinuxPinnedSignatureProof{SignatureRef: ref, Signature: ed25519.Sign(private, message)}
	modifiedSignature := append([]byte(nil), proof.Signature...)
	modifiedSignature[0] ^= 0x01
	for _, tc := range []struct {
		name     string
		identity plugin.Identity
		trust    *pruntime.LinuxHostSignerTrust
		proof    pruntime.LinuxPinnedSignatureProof
		digest   string
	}{
		{name: "wrong-host-trust-key", identity: identity, trust: otherTrust, proof: proof, digest: digest},
		{name: "modified-detached-signature", identity: identity, trust: trust,
			proof: pruntime.LinuxPinnedSignatureProof{SignatureRef: ref, Signature: modifiedSignature}, digest: digest},
		{name: "other-signature-reference", identity: identity, trust: trust,
			proof: pruntime.LinuxPinnedSignatureProof{SignatureRef: "host-trusted-signer/other", Signature: proof.Signature}, digest: digest},
		{name: "different-plugin-version", identity: plugin.Identity{ID: identity.ID, Version: "2.0.0"},
			trust: trust, proof: proof, digest: digest},
		{name: "different-plugin-identity", identity: plugin.Identity{ID: core.MustCanonicalID("tei.process.other"), Version: identity.Version},
			trust: trust, proof: proof, digest: digest},
		{name: "empty-signer-reference", identity: identity, trust: trust,
			proof: pruntime.LinuxPinnedSignatureProof{Signature: proof.Signature}, digest: digest},
		{name: "wrong-executable-digest", identity: identity, trust: trust,
			proof: proof, digest: "sha256:" + strings.Repeat("0", 64)},
		{name: "missing-Host-trust", identity: identity, proof: proof, digest: digest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pinned, err := pruntime.PinSignedLinuxProcess(ctx, tc.trust, tc.identity, binary, tc.digest, tc.proof, pruntime.StdioProcessLauncher{})
			if err == nil {
				pinned.Close()
				t.Fatal("untrusted signature or executable was allowed to launch")
			}
		})
	}
	if err := trust.Allow(ref, public[:2]); err == nil {
		t.Fatal("invalid Host trust key was accepted")
	}
	if err := trust.Allow("", public); err == nil {
		t.Fatal("blank Host trust reference was accepted")
	}
	if _, err := pruntime.LinuxPinnedSignatureMessage(identity, digest, " "); err == nil {
		t.Fatal("whitespace signer reference became signable")
	}
	if _, err := pruntime.LinuxPinnedSignatureMessage(identity, "sha256:"+strings.Repeat("a", 63)+"g", ref); err == nil {
		t.Fatal("invalid digest became signable")
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	if _, err := pruntime.PinSignedLinuxProcess(stopped, trust, identity, binary, digest, proof, pruntime.StdioProcessLauncher{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled signature preparation was not refused: %v", err)
	}
}

func TestProductPinnedLinuxSignerTrustRevocationAndRotation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	binary, digest := buildPinnedFixture(t, ctx, t.TempDir(), "approved")
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	const ref = "host-trusted-signer/2026-10"
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	message, err := pruntime.LinuxPinnedSignatureMessage(identity, digest, ref)
	if err != nil {
		t.Fatal(err)
	}
	proof := pruntime.LinuxPinnedSignatureProof{SignatureRef: ref, Signature: ed25519.Sign(private, message)}
	trust := pruntime.NewLinuxHostSignerTrust()
	if err := trust.Allow(ref, public); err != nil {
		t.Fatal(err)
	}
	host, err := pruntime.PinSignedLinuxProcess(ctx, trust, identity, binary, digest, proof, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer host.Close()
	if got, err := invokeProductPinned(t, ctx, host, binary, digest, ref, "", plugin.SandboxNone); err != nil || !strings.Contains(got.Message, "revision=approved") {
		t.Fatalf("Host-trusted signer was rejected: result=%+v error=%v", got, err)
	}
	// Updating the original caller-owned public key must not modify the Host's
	// defensive copy of the trust anchor.
	public[0] ^= 0xff
	if _, err := invokeProductPinned(t, ctx, host, binary, digest, ref, "", plugin.SandboxNone); err != nil {
		t.Fatalf("caller mutated Host-owned signer key: %v", err)
	}
	if !trust.Revoke(ref) || trust.Revoke(ref) {
		t.Fatal("revocation result was not stable")
	}
	if _, err := invokeProductPinned(t, ctx, host, binary, digest, ref, "", plugin.SandboxNone); err == nil {
		t.Fatal("revoked signer kept authorizing Registry calls")
	}
	if _, err := host.Invoke(ctx, pruntime.ProcessCommand{Executable: binary}, pruntime.Request{}); err == nil {
		t.Fatal("revoked signer allowed a direct process start")
	}
	if _, err := pruntime.PinSignedLinuxProcess(ctx, trust, identity, binary, digest, proof, pruntime.StdioProcessLauncher{}); err == nil {
		t.Fatal("revoked signer could still pin a new executable")
	}

	newPublic, newPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := trust.Allow(ref, newPublic); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeProductPinned(t, ctx, host, binary, digest, ref, "", plugin.SandboxNone); err == nil {
		t.Fatal("recreated signer reference restored a stale image approval")
	}
	if _, err := pruntime.PinSignedLinuxProcess(ctx, trust, identity, binary, digest, proof, pruntime.StdioProcessLauncher{}); err == nil {
		t.Fatal("old signature was accepted under a rotated Host key")
	}
	renewed := pruntime.LinuxPinnedSignatureProof{SignatureRef: ref, Signature: ed25519.Sign(newPrivate, message)}
	newHost, err := pruntime.PinSignedLinuxProcess(ctx, trust, identity, binary, digest, renewed, pruntime.StdioProcessLauncher{})
	if err != nil {
		t.Fatal(err)
	}
	defer newHost.Close()
	if got, err := invokeProductPinned(t, ctx, newHost, binary, digest, ref, "", plugin.SandboxNone); err != nil || !strings.Contains(got.Message, "revision=approved") {
		t.Fatalf("reverified signer cannot start approved image: result=%+v error=%v", got, err)
	}
	// Re-registering the same public key still revokes its old epoch.
	if err := trust.Allow(ref, newPublic); err != nil {
		t.Fatal(err)
	}
	if _, err := invokeProductPinned(t, ctx, newHost, binary, digest, ref, "", plugin.SandboxNone); err == nil {
		t.Fatal("re-registering a key resurrected its prior epoch")
	}
}

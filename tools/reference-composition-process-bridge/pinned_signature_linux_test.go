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
	proof := pruntime.LinuxPinnedSignatureProof{
		SignatureRef: ref, TrustedPublicKey: public, Signature: signature,
	}
	host, err := pruntime.PinSignedLinuxProcess(ctx, identity, approved, digest, proof, pruntime.StdioProcessLauncher{})
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
	dir := t.TempDir()
	binary, digest := buildPinnedFixture(t, ctx, dir, "approved")
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
	proof := pruntime.LinuxPinnedSignatureProof{
		SignatureRef: ref, TrustedPublicKey: public, Signature: ed25519.Sign(private, message),
	}

	for _, tc := range []struct {
		name     string
		identity plugin.Identity
		proof    pruntime.LinuxPinnedSignatureProof
		digest   string
	}{
		{name: "wrong-host-trust-key", identity: identity,
			proof: pruntime.LinuxPinnedSignatureProof{SignatureRef: ref, TrustedPublicKey: otherPublic, Signature: proof.Signature}, digest: digest},
		{name: "modified-detached-signature", identity: identity,
			proof: pruntime.LinuxPinnedSignatureProof{SignatureRef: ref, TrustedPublicKey: public, Signature: append([]byte(nil), proof.Signature...)}, digest: digest},
		{name: "other-signature-reference", identity: identity,
			proof: pruntime.LinuxPinnedSignatureProof{SignatureRef: "host-trusted-signer/other", TrustedPublicKey: public, Signature: proof.Signature}, digest: digest},
		{name: "different-plugin-version", identity: plugin.Identity{ID: identity.ID, Version: "2.0.0"},
			proof: proof, digest: digest},
		{name: "different-plugin-identity", identity: plugin.Identity{ID: core.MustCanonicalID("tei.process.other"), Version: identity.Version},
			proof: proof, digest: digest},
		{name: "invalid-trusted-key", identity: identity,
			proof: pruntime.LinuxPinnedSignatureProof{SignatureRef: ref, TrustedPublicKey: public[:2], Signature: proof.Signature}, digest: digest},
		{name: "empty-signer-reference", identity: identity,
			proof: pruntime.LinuxPinnedSignatureProof{TrustedPublicKey: public, Signature: proof.Signature}, digest: digest},
		{name: "wrong-executable-digest", identity: identity,
			proof: proof, digest: "sha256:" + strings.Repeat("0", 64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "modified-detached-signature" {
				tc.proof.Signature[0] ^= 0x01
			}
			result, err := pruntime.PinSignedLinuxProcess(ctx, tc.identity, binary, tc.digest, tc.proof, pruntime.StdioProcessLauncher{})
			if err == nil {
				result.Close()
				t.Fatal("invalid signature or image was allowed to become executable")
			}
		})
	}

	// Signature construction must not accept ambiguous/unknown artifact
	// references even if callers offer an otherwise valid signing key.
	if _, err := pruntime.LinuxPinnedSignatureMessage(identity, digest, " "); err == nil {
		t.Fatal("whitespace signer reference accepted")
	}
	if _, err := pruntime.LinuxPinnedSignatureMessage(identity, "sha256:"+strings.Repeat("a", 63)+"g", ref); err == nil {
		t.Fatal("invalid digest became signable")
	}
	stopped, stop := context.WithCancel(ctx)
	stop()
	if _, err := pruntime.PinSignedLinuxProcess(stopped, identity, binary, digest, proof, pruntime.StdioProcessLauncher{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled signature preparation was not rejected: %v", err)
	}
}

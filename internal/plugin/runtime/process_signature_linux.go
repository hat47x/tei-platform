//go:build linux

package runtime

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tei/internal/plugin"
)

// LinuxPinnedSignatureProof provides a detached Ed25519 signature. Its
// PublicKey must come from an independently managed Host trust policy, never
// from the untrusted plugin artifact or its own metadata. SignatureRef is
// the Host-authorized reference to this proof, not a trust anchor by itself.
type LinuxPinnedSignatureProof struct {
	SignatureRef     string
	TrustedPublicKey ed25519.PublicKey
	Signature        []byte
}

// LinuxPinnedSignatureMessage defines the bytes that an authorized signer
// signs. The domain, plugin identity, exact artifact digest, and signature
// reference are all signed to prevent cross-plugin or cross-version reuse.
//
// This is a Linux Runtime-specific proof format, not a Core ABI, a software
// supply-chain provenance statement, or an endorsement of the signer.
func LinuxPinnedSignatureMessage(identity plugin.Identity, digest, signatureRef string) ([]byte, error) {
	if identity.ID == "" || identity.Version == "" {
		return nil, errors.New("Linux pinned signature requires plugin identity and version")
	}
	if len(digest) != len("sha256:")+64 || !strings.HasPrefix(digest, "sha256:") {
		return nil, errors.New("Linux pinned signature requires a SHA-256 digest")
	}
	for _, c := range strings.TrimPrefix(digest, "sha256:") {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return nil, errors.New("Linux pinned signature digest must use lowercase hex")
		}
	}
	if signatureRef == "" || strings.TrimSpace(signatureRef) != signatureRef || len(signatureRef) > 512 {
		return nil, errors.New("Linux pinned signature requires a bounded nonblank reference")
	}
	// Encoding a fixed-field struct avoids concatenation ambiguity. A
	// versioned domain prevents reuse as another protocol's signature.
	return json.Marshal(struct {
		Domain       string `json:"domain"`
		PluginID     string `json:"pluginId"`
		Version      string `json:"version"`
		ArtifactHash string `json:"artifactSha256"`
		SignatureRef string `json:"signatureRef"`
	}{
		Domain:       "tei-linux-pinned-image-ed25519-v1",
		PluginID:     string(identity.ID),
		Version:      identity.Version,
		ArtifactHash: digest,
		SignatureRef: signatureRef,
	})
}

// PinSignedLinuxProcess pins and SHA-256-verifies the exact Linux process
// image, then checks a detached Ed25519 signature over that image's digest
// and plugin identity. Failure closes the pinned descriptor and rejects launch.
//
// The caller MUST establish the trusted key and allowed signature reference
// from independent Host policy. No key supplied by a plugin may become a trust
// root merely by appearing in a proof. A valid signature proves possession of
// the selected private key, not provenance, review, sandboxing, or key validity
// beyond the Host's policy. This function never marks ProvenanceChecked=true.
func PinSignedLinuxProcess(ctx context.Context, identity plugin.Identity, source, digest string, proof LinuxPinnedSignatureProof, transport StdioProcessLauncher) (*PinnedLinuxProcess, error) {
	if len(proof.TrustedPublicKey) != ed25519.PublicKeySize || len(proof.Signature) != ed25519.SignatureSize {
		return nil, errors.New("Linux pinned signature requires a trusted Ed25519 public key and signature")
	}
	message, err := LinuxPinnedSignatureMessage(identity, digest, proof.SignatureRef)
	if err != nil {
		return nil, err
	}
	pinned, err := PinLinuxProcess(ctx, identity, source, digest, transport)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(proof.TrustedPublicKey, message, proof.Signature) {
		_ = pinned.Close()
		return nil, fmt.Errorf("pinned image signature verification failed for %s@%s", identity.ID, identity.Version)
	}
	// Not published before these immutable fields are set. Both Verify and
	// Invoke subsequently operate on this same already-checked image handle.
	pinned.signatureRef = proof.SignatureRef
	pinned.signatureVerified = true
	return pinned, nil
}

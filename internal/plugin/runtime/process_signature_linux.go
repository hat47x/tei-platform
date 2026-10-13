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

// LinuxPinnedSignatureProofは署名参照と署名値だけを保持する。
// 信頼する公開鍵は必ず別のHost管理のLinuxHostSignerTrustから取得する。
type LinuxPinnedSignatureProof struct {
	SignatureRef string
	Signature    []byte
}

func validateLinuxSignatureRef(ref string) error {
	if ref == "" || strings.TrimSpace(ref) != ref || len(ref) > 512 {
		return errors.New("Linux pinned signature requires a bounded nonblank reference")
	}
	return nil
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
	if err := validateLinuxSignatureRef(signatureRef); err != nil {
		return nil, err
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
func PinSignedLinuxProcess(ctx context.Context, trust *LinuxHostSignerTrust, identity plugin.Identity, source, digest string, proof LinuxPinnedSignatureProof, transport StdioProcessLauncher) (*PinnedLinuxProcess, error) {
	if len(proof.Signature) != ed25519.SignatureSize {
		return nil, errors.New("Linux pinned signature requires an Ed25519 signature")
	}
	message, err := LinuxPinnedSignatureMessage(identity, digest, proof.SignatureRef)
	if err != nil {
		return nil, err
	}
	public, epoch, err := trust.resolve(proof.SignatureRef)
	if err != nil {
		return nil, err
	}
	pinned, err := PinLinuxProcess(ctx, identity, source, digest, transport)
	if err != nil {
		return nil, err
	}
	if !ed25519.Verify(public, message, proof.Signature) || !trust.current(proof.SignatureRef, epoch) {
		_ = pinned.Close()
		return nil, fmt.Errorf("pinned image signature verification failed or signer trust changed for %s@%s", identity.ID, identity.Version)
	}
	// Not published before these immutable fields are set. Both Verify and
	// Invoke subsequently check the same Host trust generation and image.
	pinned.signatureRef = proof.SignatureRef
	pinned.signatureVerified = true
	pinned.signerTrust = trust
	pinned.signerEpoch = epoch
	return pinned, nil
}

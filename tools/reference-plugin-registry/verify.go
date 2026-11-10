package registryreference

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Reference-only public keys. These are NOT a production trust root.
const publisherPubHex = "03a107bff3ce10be1d70dd18e74bc09967e4d6309ba50d5f1ddc8664125531b8"
const verifierPubHex = "29acbae141bccaf0b22e1a94d34d0bc7361e526d0bfe12c89794bc9322966dd7"

type Artifact struct {
	Role        string
	BytesBase64 string
}
type Attestation struct {
	Subject         string
	ProfileDigest   string
	Target          string
	Result          string
	SignatureBase64 string
}
type Fixture struct {
	ReleaseText              string
	PublisherSignatureBase64 string
	ProfileText              string
	Artifacts                []Artifact
	Attestation              Attestation
}
type Policy struct {
	ExpectedPlugin   string
	ExpectedVersion  string
	OwnerEpoch       int
	ProfileDigest    string
	Target           string
	Fresh            bool
	PublisherTrusted bool
	VerifierTrusted  bool
}

func Digest(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func ReadFixture(path string) (Fixture, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Fixture{}, err
	}
	return ParseFixture(raw)
}

func DefaultPolicy() Policy {
	return Policy{ExpectedPlugin: "tei:plugin:fixture-alpha", ExpectedVersion: "1.0.0", OwnerEpoch: 3,
		ProfileDigest: "sha256:ee8702820103a27f205b511d8dfb25c339bddfd55c63326e0fafc2f9effa12ea", Target: "linux-amd64", Fresh: true, PublisherTrusted: true, VerifierTrusted: true}
}

func verifySignature(pubHex, signatureB64 string, message []byte) error {
	pub, err := hex.DecodeString(pubHex)
	if err != nil {
		return err
	}
	sig, err := decodeCanonicalBase64(signatureB64)
	if err != nil {
		return err
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(pub), message, sig) {
		return errors.New("invalid signature")
	}
	return nil
}

// Verify validates a fixed, deliberately small reference claim. It does not
// resolve contracts, establish current Trust, or authorize execution.
func Verify(f Fixture, p Policy) error {
	if !p.Fresh || !p.PublisherTrusted || !p.VerifierTrusted {
		return errors.New("trust policy not currently satisfied")
	}
	lines := strings.Split(f.ReleaseText, "\n")
	if len(lines) != 7 || lines[0] != "TEI-REF-RELEASE-V0" || lines[6] != "" {
		return errors.New("release framing invalid")
	}
	keys := []string{"plugin=", "version=", "epoch=", "artifact.core=", "artifact.ui="}
	values := make([]string, len(keys))
	for i, key := range keys {
		if !strings.HasPrefix(lines[i+1], key) || len(lines[i+1]) <= len(key) {
			return fmt.Errorf("invalid release field %d", i)
		}
		values[i] = strings.TrimPrefix(lines[i+1], key)
	}
	if values[0] != p.ExpectedPlugin || values[1] != p.ExpectedVersion {
		return errors.New("unexpected plugin/version")
	}
	if values[2] != fmt.Sprint(p.OwnerEpoch) {
		return errors.New("owner epoch mismatch")
	}
	expected := map[string]string{"core": values[3], "ui": values[4]}
	if len(f.Artifacts) != len(expected) {
		return errors.New("required artifact count mismatch")
	}
	seen := map[string]bool{}
	for _, a := range f.Artifacts {
		want, ok := expected[a.Role]
		if !ok || seen[a.Role] {
			return errors.New("unknown or duplicate artifact role")
		}
		seen[a.Role] = true
		data, err := decodeCanonicalBase64(a.BytesBase64)
		if err != nil {
			return err
		}
		if Digest(data) != want {
			return fmt.Errorf("artifact %q digest mismatch", a.Role)
		}
	}
	pubMessage := append([]byte("TEI-REF-PUBLISH-V0\x00"), []byte(f.ReleaseText)...)
	if err := verifySignature(publisherPubHex, f.PublisherSignatureBase64, pubMessage); err != nil {
		return fmt.Errorf("publisher: %w", err)
	}

	subject := Digest([]byte(f.ReleaseText))
	if f.Attestation.Subject != subject {
		return errors.New("attestation subject mismatch")
	}
	if Digest([]byte(f.ProfileText)) != f.Attestation.ProfileDigest || f.Attestation.ProfileDigest != p.ProfileDigest {
		return errors.New("attestation profile mismatch")
	}
	if f.Attestation.Target != p.Target || f.Attestation.Result != "passed" {
		return errors.New("attestation scope/result mismatch")
	}
	predicate := "TEI-REF-ATTEST-V0\n" + f.Attestation.Subject + "\n" + f.Attestation.ProfileDigest + "\n" + f.Attestation.Target + "\n" + f.Attestation.Result + "\n"
	if err := verifySignature(verifierPubHex, f.Attestation.SignatureBase64, []byte(predicate)); err != nil {
		return fmt.Errorf("verifier: %w", err)
	}
	return nil
}

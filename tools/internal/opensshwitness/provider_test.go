package opensshwitness

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// providerBinary locates a real ssh-keygen. Tests skip (not pass) when none is
// installed, exactly like the Reference pressure_test.go, so a missing
// provider can never be read as a qualified witness. Set
// TEI_OPENSSH_WITNESS_SKIP=1 to disable the provider-facing tests entirely.
var sshKeygenPathValue string

// TestMain caches the provider location so every test and the sign helper use
// the same binary.
func TestMain(m *testing.M) {
	if path, err := exec.LookPath("ssh-keygen"); err == nil {
		sshKeygenPathValue = path
	}
	os.Exit(m.Run())
}

func providerBinary(t *testing.T) string {
	t.Helper()
	if os.Getenv("TEI_OPENSSH_WITNESS_SKIP") == "1" {
		t.Skip("witness provider exercise disabled by TEI_OPENSSH_WITNESS_SKIP")
	}
	path, err := exec.LookPath("ssh-keygen")
	if err != nil {
		t.Skip("ssh-keygen unavailable; real OpenSSH witness not qualified")
	}
	return path
}

type testProvider struct {
	keyFile    string
	allowed    string
	expired    string
	future     string
	revocation string
}

// newTestProvider builds a temporary ed25519 key and several allowed-signers
// policies with a real provider, mirroring the pressure harness. The accept
// policy permits only the accept namespace of the exact scope digest, so an
// accept signature cannot decide a reject and no other scope can ride the
// entry.
func newTestProvider(t *testing.T) *testProvider {
	t.Helper()
	sshKeygen := providerBinary(t)
	// A provider build without -O verify-time cannot pin freshness. The
	// capability-aware skip mirrors the pressure harness: an unprovable
	// environment is not a pass.
	if ok, _ := verifyTimeSupported(sshKeygen, "20261007120000Z"); !ok {
		t.Skip("provider build lacks -O verify-time; freshness cannot be pinned for the witness")
	}
	work := t.TempDir()
	key := filepath.Join(work, "reviewer")
	if out, err := exec.Command(sshKeygen, "-q", "-t", "ed25519", "-N", "", "-f", key).
		CombinedOutput(); err != nil {
		t.Skipf("ed25519 key generation failed; real provider not exercisable: %v: %s", err, out)
	}
	publicKey, err := os.ReadFile(key + ".pub")
	if err != nil {
		t.Fatalf("read public key: %v", err)
	}
	identity := "reviewer@example.test"
	payload := validPayload()
	payload.WitnessNonce = ""
	namespace, err := payload.Namespace()
	if err != nil {
		t.Fatalf("namespace: %v", err)
	}
	write := func(name, content string) string {
		path := filepath.Join(work, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		return path
	}
	allowed := write("allowed_signers",
		identity+" namespaces=\""+namespace+"\",valid-after=\"20200101Z\",valid-before=\"20990101Z\" "+strings.TrimSpace(string(publicKey))+"\n")
	expired := write("allowed_signers_expired",
		identity+" namespaces=\""+namespace+"\",valid-before=\"20200101Z\" "+strings.TrimSpace(string(publicKey))+"\n")
	future := write("allowed_signers_future",
		identity+" namespaces=\""+namespace+"\",valid-after=\"20990101Z\" "+strings.TrimSpace(string(publicKey))+"\n")
	revocation := filepath.Join(work, "revoked.krl")
	if out, err := exec.Command(sshKeygen, "-q", "-k", "-f", revocation, key+".pub").
		CombinedOutput(); err != nil {
		t.Fatalf("krl generation: %v: %s", err, out)
	}
	return &testProvider{
		keyFile:    key,
		allowed:    allowed,
		expired:    expired,
		future:     future,
		revocation: revocation,
	}
}

func (p *testProvider) options(allowed string) ProviderOptions {
	return ProviderOptions{
		SSHKeygen:      sshKeygenPathValue,
		AllowedSigners: allowed,
		// Matches the payload's fixed witnessIssuedAt and the validity
		// windows of the policies above.
		VerifyTime: "20261007120000Z",
	}
}

func signPayload(t *testing.T, p *testProvider, payload Payload) []byte {
	t.Helper()
	signature, err := Sign(SSHKeygenOption(sshKeygenPathValue), p.keyFile, payload)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signature
}

func rejectPayload(base Payload) Payload {
	mutated := base
	mutated.Decision = DecisionReject
	return mutated
}

func TestExactPrincipalNamespaceAndPolicyQualify(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	evidence, err := Verify(p.options(p.allowed), "reviewer@example.test", payload, signature)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if evidence.GateStatus != GateAuthorityQualified {
		t.Fatalf("expected authority-qualified, got %s (%s %q)",
			evidence.GateStatus, evidence.Diagnosis, evidence.ProviderReason)
	}
	if evidence.SignerIdentity != "reviewer@example.test" {
		t.Fatalf("signer identity mismatch: %q", evidence.SignerIdentity)
	}
	if evidence.KeyFingerprint == "" {
		t.Fatalf("key fingerprint must bind the finding: %q", evidence.ProviderReason)
	}
	if evidence.RevocationChecked {
		t.Fatalf("revocation must be marked checked only when a KRL is supplied")
	}
	if evidence.EvidenceFingerprint == "" {
		t.Fatalf("evidence fingerprint must be computed")
	}
	if evidence.MessageDigest == "" || evidence.SignatureDigest == "" {
		t.Fatalf("message and signature digests must be recorded")
	}
}

func TestEvidenceFingerprintIsDeterministic(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	first, err := Verify(p.options(p.allowed), "reviewer@example.test", payload, signature)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	second, err := Verify(p.options(p.allowed), "reviewer@example.test", payload, signature)
	if err != nil {
		t.Fatalf("verify again: %v", err)
	}
	if first.EvidenceFingerprint != second.EvidenceFingerprint {
		t.Fatalf("equal verifications must share one evidence fingerprint: %s vs %s",
			first.EvidenceFingerprint, second.EvidenceFingerprint)
	}
}

func TestWrongPrincipalCannotQualify(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	evidence, err := Verify(p.options(p.allowed), "other@example.test", payload, signature)
	if err != nil {
		t.Fatalf("verify ran into an environment failure: %v", err)
	}
	if evidence.GateStatus != GateAuthorityUnresolved || evidence.Diagnosis == "" {
		t.Fatalf("wrong principal must stay unresolved with a diagnosis: %+v", evidence)
	}
}

func TestAcceptSignatureCannotDecideReject(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	// Verify the accept signature for a reject decision: the policy permits
	// only the accept namespace, so the accept signature must not decide a
	// reject.
	evidence, err := Verify(p.options(p.allowed), "reviewer@example.test", rejectPayload(payload), signature)
	if err != nil {
		t.Fatalf("unexpected environment failure: %v", err)
	}
	if evidence.GateStatus != GateAuthorityUnresolved {
		t.Fatalf("accept signature must not qualify a reject decision: %+v", evidence)
	}
}

func TestTamperedPayloadCannotQualify(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	tampered := payload
	tampered.CandidateFingerprint = "sha256:" + strings.Repeat("f", 64)
	evidence, err := Verify(p.options(p.allowed), "reviewer@example.test", tampered, signature)
	if err != nil {
		t.Fatalf("unexpected environment failure: %v", err)
	}
	if evidence.GateStatus != GateAuthorityUnresolved {
		t.Fatalf("tampered candidate fingerprint must fail closed: %+v", evidence)
	}
}

func TestExpiredPolicyCannotQualify(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	evidence, err := Verify(p.options(p.expired), "reviewer@example.test", payload, signature)
	if err != nil {
		t.Fatalf("unexpected environment failure: %v", err)
	}
	if evidence.GateStatus != GateAuthorityUnresolved {
		t.Fatalf("expired allowed signer must stay unresolved: %+v", evidence)
	}
	if evidence.ProviderReason == "" {
		t.Fatalf("expired policy must keep the provider reason for the record")
	}
}

func TestFuturePolicyCannotQualify(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	evidence, err := Verify(p.options(p.future), "reviewer@example.test", payload, signature)
	if err != nil {
		t.Fatalf("unexpected environment failure: %v", err)
	}
	if evidence.GateStatus != GateAuthorityUnresolved {
		t.Fatalf("not-yet-valid allowed signer must stay unresolved: %+v", evidence)
	}
}

func TestRevokedKeyCannotQualify(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	options := p.options(p.allowed)
	options.Revocation = p.revocation
	evidence, err := Verify(options, "reviewer@example.test", payload, signature)
	if err != nil {
		t.Fatalf("unexpected environment failure: %v", err)
	}
	if evidence.GateStatus != GateAuthorityUnresolved {
		t.Fatalf("revoked signing key must fail closed: %+v", evidence)
	}
	if !evidence.RevocationChecked {
		t.Fatalf("revocation must be marked checked")
	}
}

func TestMissingProviderCannotClaimAnything(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	signature := signPayload(t, p, payload)
	options := p.options(p.allowed)
	options.SSHKeygen = "tei-no-openssh-provider"
	_, err := Verify(options, "reviewer@example.test", payload, signature)
	if !errors.Is(err, ErrProviderUnavailable) {
		t.Fatalf("missing provider must return ErrProviderUnavailable, got %v", err)
	}
}

func TestInvalidPayloadNeverReachesTheProvider(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	payload.Decision = "maybe"
	if _, err := Verify(p.options(p.allowed), "reviewer@example.test", payload, []byte("sig")); err == nil {
		t.Fatalf("invalid payload must be refused before provider contact")
	}
}

func TestVerifyTimeIsMandatory(t *testing.T) {
	p := newTestProvider(t)
	payload := validPayload()
	options := p.options(p.allowed)
	options.VerifyTime = "not-a-time"
	if _, err := Verify(options, "reviewer@example.test", payload, []byte("sig")); err == nil {
		t.Fatalf("verify without a pinned time must be refused")
	}
	if _, err := Sign(SSHKeygenOption(sshKeygenPathValue), p.keyFile, payload); err != nil {
		t.Fatalf("sign must not require verify-time: %v", err)
	}
}

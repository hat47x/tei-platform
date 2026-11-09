package opensshwitness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// EvidenceSchema is the Evidence record schema produced by Verify. It is a
// Reference candidate, not a production acceptance schema.
const EvidenceSchema = "tei.authoring.witness-evidence/v0"

// VerifyTimePattern is the compact time form OpenSSH accepts for allowed
// signer validity and for -O verify-time.
var verifyTimePattern = regexp.MustCompile(`^\d{8}\d{6}Z$`)

var keyFingerprintPattern = regexp.MustCompile(`with ED25519 key SHA256:([A-Za-z0-9/+]{43})\b`)

var (
	// ErrProviderUnavailable reports that the signing provider cannot be
	// located. No result is claimed, and callers must not fall back to
	// self-asserted actor strings.
	ErrProviderUnavailable = errors.New("openssh witness provider unavailable; no result is claimed")
	// ErrVerifyTimeUnsupported reports a provider without -O verify-time
	// support. Freshness would be unchecked, so the verification is not
	// performed and no result is claimed.
	ErrVerifyTimeUnsupported = errors.New("openssh provider cannot pin the verification time; no result is claimed")
)

// GateStatus mirrors the gate states in
// plan/research/AUTHORING_DECISION_AUTHORITY_EVIDENCE_GATE_2026-10.md §5. The
// reference library can only ever reach authority-qualified; every other
// outcome stays unresolved so that callers cannot upgrade it.
type GateStatus string

const (
	GateAuthorityQualified  GateStatus = "authority-qualified"
	GateAuthorityUnresolved GateStatus = "authority-unresolved"
)

// Evidence is the provider's finding for one exact witness message. It is
// evidence about identity, scope, freshness and binding; it is not a decision
// record and never mutates any workspace state.
type Evidence struct {
	Schema              string     `json:"schema"`
	WorkspaceIdentity   string     `json:"workspaceIdentity"`
	Decision            string     `json:"decision"`
	Namespace           string     `json:"namespace"`
	SignerIdentity      string     `json:"signerIdentity"`
	KeyFingerprint      string     `json:"keyFingerprint,omitempty"`
	RevocationChecked   bool       `json:"revocationChecked"`
	VerifyTime          string     `json:"verifyTime"`
	WitnessIssuedAt     string     `json:"witnessIssuedAt"`
	GateStatus          GateStatus `json:"gateStatus"`
	Diagnosis           string     `json:"diagnosis,omitempty"`
	ProviderReason      string     `json:"providerReason,omitempty"`
	MessageDigest       string     `json:"messageDigest"`
	SignatureDigest     string     `json:"signatureDigest"`
	EvidenceFingerprint string     `json:"evidenceFingerprint"`
}

// ProviderOptions selects the policy input for one verification. AllowedSigners
// and Revocation must come from a managed configuration boundary; callers may
// not derive them from the candidate or the workspace (policy source rule).
type ProviderOptions struct {
	// SSHKeygen is the provider binary. Empty means "first ssh-keygen on PATH".
	SSHKeygen string
	// AllowedSigners is the managed allowed-signers policy file.
	AllowedSigners string
	// Revocation is an optional KRL or revoked-keys file.
	Revocation string
	// VerifyTime pins the decision verification time (YYYYMMDDHHMMSSZ). It is
	// required: without it, allowed-signer freshness could not be checked and
	// the library would refuse to run.
	VerifyTime string
}

// fingerprint returns the sha256:... digest of arbitrary bytes.
func fingerprint(value []byte) string {
	sum := sha256.Sum256(value)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// evidenceFingerprint canonicalises the evidence with sorted JSON keys so that
// two findings with the same content share one identity.
func evidenceFingerprint(evidence Evidence) string {
	// Evidence carries its own fingerprint; drop it for the digest input.
	evidence.EvidenceFingerprint = ""
	canonical, err := json.Marshal(evidence)
	if err != nil {
		return ""
	}
	return fingerprint(canonical)
}

// resolveSSHKeygen returns the provider binary, probing PATH when no explicit
// path was given.
func resolveSSHKeygen(o ProviderOptions) (string, error) {
	if o.SSHKeygen != "" {
		return o.SSHKeygen, nil
	}
	return exec.LookPath("ssh-keygen")
}

// checkOptions enforces the pinned-time requirement of verification.
func checkOptions(o ProviderOptions) error {
	if !verifyTimePattern.MatchString(o.VerifyTime) {
		return errors.New("openssh witness verification requires verifyTime as YYYYMMDDHHMMSSZ")
	}
	return nil
}

// capabilityProbeName is the placeholder identity of a verification of files
// that do not exist. The probe never touches a real key, policy or signature;
// it only checks whether the provider knows the verify-time option.
const capabilityProbeName = "tei-openssh-witness-probe"

// verifyTimeSupported reports whether this provider build accepts the
// verify-time option. Distinct shapes of refusal stay false negatives: the
// library then refuses to verify rather than run without pinned freshness.
func verifyTimeSupported(sshKeygen string, verifyTime string) (bool, error) {
	cmd := exec.Command(sshKeygen, "-Y", "verify",
		"-O", "verify-time="+verifyTime,
		"-f", capabilityProbeName+"-policy",
		"-I", capabilityProbeName,
		"-n", capabilityProbeName+"-namespace",
		"-s", capabilityProbeName+"-signature")
	cmd.Stdin = strings.NewReader("")
	output, _ := cmd.CombinedOutput()
	return !strings.Contains(string(output), "verify-time"), nil
}

// Sign signs the canonical message of payload with the private key at
// privateKeyFile and returns the raw SSHSIG signature. The namespace is
// derived from the payload; the caller cannot pick an authority scope by
// naming a namespace. Sign is for test and Reference harness keys only; a
// production flow would ask the operator's own SSH agent to sign.
func Sign(provider SSHKeygenOption, privateKeyFile string, payload Payload) ([]byte, error) {
	message, err := payload.Message()
	if err != nil {
		return nil, err
	}
	namespace, err := payload.Namespace()
	if err != nil {
		return nil, err
	}
	sshKeygen, err := resolveSSHKeygen(ProviderOptions{SSHKeygen: string(provider)})
	if err != nil {
		return nil, ErrProviderUnavailable
	}
	work, err := os.MkdirTemp("", "openssh-witness-sign-*")
	if err != nil {
		return nil, fmt.Errorf("witness sign scratch dir: %w", err)
	}
	defer os.RemoveAll(work)
	messageFile := filepath.Join(work, "witness.txt")
	if err := os.WriteFile(messageFile, []byte(message), 0o600); err != nil {
		return nil, fmt.Errorf("witness sign message write: %w", err)
	}
	sign := exec.Command(sshKeygen, "-q", "-Y", "sign",
		"-f", privateKeyFile,
		"-n", namespace,
		messageFile)
	if output, err := sign.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("openssh witness sign: %v: %s", err, strings.TrimSpace(string(output)))
	}
	signature, err := os.ReadFile(messageFile + ".sig")
	if err != nil {
		return nil, fmt.Errorf("openssh witness sign output: %w", err)
	}
	return signature, nil
}

// SSHKeygenOption carries just the provider binary. Signing does not consult
// the allowed-signers policy, so it does not take the full ProviderOptions.
type SSHKeygenOption string

// scratchDir holds the verifier's scratch files in a directory that clean
// removes.
type scratchDir struct {
	root string
}

func newScratchDir() (*scratchDir, error) {
	root, err := os.MkdirTemp("", "openssh-witness-verify-*")
	if err != nil {
		return nil, fmt.Errorf("witness verify scratch dir: %w", err)
	}
	return &scratchDir{root: root}, nil
}

func (s *scratchDir) file(name string, content []byte) (string, error) {
	path := filepath.Join(s.root, name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return "", fmt.Errorf("witness scratch write %s: %w", name, err)
	}
	return path, nil
}

func (s *scratchDir) clean() { os.RemoveAll(s.root) }

// verifyDiagnostics classifies provider failure text into fail-closed
// diagnoses. Classification is best-effort; the raw reason is always kept, and
// a classification never upgrades a failed verification.
var verifyDiagnostics = []struct {
	match     string
	diagnosis string
}{
	{"namespace does not match", "namespace-mismatch"},
	{"principal not allowed", "principal-not-allowed"},
	{"revoked", "signing-key-revoked"},
	{"expired", "allowed-signers-expired"},
	{"not yet valid", "allowed-signers-not-yet-valid"},
	{"usage restricted", "namespace-restricted"},
	{"Could not verify signature", "verification-refused"},
	{"Couldn't verify signature", "verification-refused"},
}

func classify(reason string) string {
	lower := strings.ToLower(reason)
	for _, d := range verifyDiagnostics {
		if strings.Contains(lower, strings.ToLower(d.match)) {
			return d.diagnosis
		}
	}
	return "verification-refused"
}

const maxProviderReason = 512

// truncateReason keeps provider output small enough for an evidence ledger.
func truncateReason(reason string) string {
	if len(reason) <= maxProviderReason {
		return reason
	}
	return reason[:maxProviderReason] + "…(truncated)"
}

// Verify checks one signature against the exact canonical message of payload
// under the namespace derived from the payload. It returns evidence with
// GateStatus authority-qualified only when the provider accepted
//
//	exactly this principal, exactly this namespace (workspace identity,
//	decision semantics and scope digest), at exactly the pinned verification
//	time, with the optional revocation list applied.
//
// A verified signature upgrades nothing else: callers must still re-check the
// candidate, the review projection and the base revision through a Shared
// Acceptance boundary before any source mutation.
func Verify(o ProviderOptions, signerIdentity string, payload Payload, signature []byte) (Evidence, error) {
	if signerIdentity == "" {
		return Evidence{}, errors.New("openssh witness verification requires the signer identity to test")
	}
	if o.AllowedSigners == "" {
		return Evidence{}, errors.New("openssh witness verification requires a managed allowed-signers policy")
	}
	if err := checkOptions(o); err != nil {
		return Evidence{}, err
	}
	if len(signature) == 0 {
		return Evidence{}, errors.New("openssh witness verification requires a signature")
	}
	// An invalid payload is a caller defect: refuse before touching the
	// provider so that a malformed witness cannot be partially verified.
	if err := payload.validate(); err != nil {
		return Evidence{}, err
	}
	namespace, err := payload.Namespace()
	if err != nil {
		return Evidence{}, err
	}
	// Defense in depth: the namespace's embedded scope must equal the
	// payload's scope exactly, so no alternate scope can ride the namespace.
	_, _, scopeHex, err := parseNamespace(namespace)
	if err != nil {
		return Evidence{}, err
	}
	if scopeHex != payload.scopeFingerprintHex() {
		return Evidence{}, errors.New("witness namespace scope does not match the payload scope")
	}
	message, err := payload.Message()
	if err != nil {
		return Evidence{}, err
	}
	sshKeygen, err := resolveSSHKeygen(o)
	if err != nil {
		return Evidence{}, ErrProviderUnavailable
	}
	// A named but missing provider binary must read as "provider unavailable",
	// never as an unresolved verification result.
	if _, err := exec.LookPath(sshKeygen); err != nil {
		return Evidence{}, ErrProviderUnavailable
	}
	supported, err := verifyTimeSupported(sshKeygen, o.VerifyTime)
	if err != nil {
		return Evidence{}, ErrProviderUnavailable
	}
	if !supported {
		return Evidence{}, ErrVerifyTimeUnsupported
	}
	work, err := newScratchDir()
	if err != nil {
		return Evidence{}, err
	}
	defer work.clean()
	signatureFile, err := work.file("witness.sig", signature)
	if err != nil {
		return Evidence{}, err
	}
	args := []string{"-Y", "verify",
		"-O", "verify-time=" + o.VerifyTime,
		"-f", o.AllowedSigners,
		"-I", signerIdentity,
		"-n", namespace,
		"-s", signatureFile}
	if o.Revocation != "" {
		args = append(args, "-r", o.Revocation)
	}
	cmd := exec.Command(sshKeygen, args...)
	cmd.Stdin = strings.NewReader(message)
	output, runErr := cmd.CombinedOutput()
	var lookupErr *exec.Error
	if errors.As(runErr, &lookupErr) {
		// The provider vanished between the probe and the verification run.
		// No result is claimed.
		return Evidence{}, ErrProviderUnavailable
	}
	reason := strings.TrimSpace(string(output))
	evidence := Evidence{
		Schema:            EvidenceSchema,
		WorkspaceIdentity: payload.WorkspaceIdentity,
		Decision:          payload.Decision,
		Namespace:         namespace,
		SignerIdentity:    signerIdentity,
		RevocationChecked: o.Revocation != "",
		VerifyTime:        o.VerifyTime,
		WitnessIssuedAt:   payload.WitnessIssuedAt,
		MessageDigest:     fingerprint([]byte(message)),
		SignatureDigest:   fingerprint(signature),
	}
	if m := keyFingerprintPattern.FindStringSubmatch(reason); m != nil {
		evidence.KeyFingerprint = m[1]
	}
	if runErr != nil {
		evidence.GateStatus = GateAuthorityUnresolved
		evidence.Diagnosis = classify(reason)
		evidence.ProviderReason = truncateReason(reason)
		evidence.EvidenceFingerprint = evidenceFingerprint(evidence)
		return evidence, nil
	}
	if !strings.Contains(reason, "Good ") {
		evidence.GateStatus = GateAuthorityUnresolved
		evidence.Diagnosis = classify(reason)
		evidence.ProviderReason = truncateReason(reason)
		evidence.EvidenceFingerprint = evidenceFingerprint(evidence)
		return evidence, nil
	}
	// The provider affirmed exactly this principal for exactly this namespace
	// and message. The full provider line stays as provenance.
	evidence.GateStatus = GateAuthorityQualified
	evidence.ProviderReason = truncateReason(reason)
	evidence.EvidenceFingerprint = evidenceFingerprint(evidence)
	return evidence, nil
}

// Now returns the current UTC time in the witness issuedAt form.
func Now() string {
	return time.Now().UTC().Format("20060102T150405Z")
}

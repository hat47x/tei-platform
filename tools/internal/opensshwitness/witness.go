// Package opensshwitness binds a provable authoring decision witness to an
// OpenSSH signing provider for ISSUE-000974. It extracts the provider-specific
// parts seen in tools/reference-authoring-witness-openssh/pressure-test.sh into
// a typed library so that the OpenSSH provider can be verified in one place
// before any Shared Acceptance integration.
//
// This package is Reference-only pressure, not a production identity provider
// and not part of the Shared Acceptance contract. It never mutates workspace
// state and never decides whether a source change may be written. A verified
// evidence is one input that a future Shared Acceptance integration may check;
// provider success alone does not authorize a source commit.
package opensshwitness

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// WitnessSchemaVersion is the reference witness message schema. It matches
// tools/reference-authoring-witness-openssh/fixture/witness-v0.txt and is not a
// production proposal.
const WitnessSchemaVersion = "tei-authoring-witness/reference-v0"

// Decision semantics supported by the reference witness. Accept and reject are
// separate namespaces at the provider level, so reusing an accept signature for
// a reject decision must never verify.
const (
	DecisionAccept = "accept"
	DecisionReject = "reject"
)

const (
	workspacePatternLength = 64
	nonceMaxLength         = 64
)

var (
	fingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
	workspacePattern   = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]{0,63})$`)
	issuedAtPattern    = regexp.MustCompile(`^\d{8}T\d{6}Z$`)
	noncePattern       = regexp.MustCompile(`^[0-9A-Za-z._:-]{1,64}$`)
)

var (
	errInvalidFingerprint  = errors.New("witness fingerprint must be sha256:<64 hex>")
	errInvalidWorkspace    = errors.New("witness workspace identity must be a lowercase token (digits, letters, dots, hyphens)")
	errInvalidDecision     = fmt.Errorf("witness decision must be %q or %q", DecisionAccept, DecisionReject)
	errInvalidIssuedAt     = errors.New("witness issuedAt must match YYYYMMDDTHHMMSSZ")
	errInvalidNonce        = errors.New("witness nonce must be 1..64 chars of [0-9A-Za-z._:-] or empty")
	errEmptyWorkspace      = errors.New("witness workspace identity is required")
	errEmptyDecision       = errors.New("witness decision is required")
	errEmptyScopeDigest    = errors.New("witness scope digest is required")
	errEmptyDecisionReason = errors.New("witness decision reason digest is required")
	errEmptyBaseModelRev   = errors.New("witness base model revision is required")
	errWorkspaceTooLong    = fmt.Errorf("witness workspace identity must be at most %d chars", workspacePatternLength)
	errNonceTooLong        = fmt.Errorf("witness nonce must be at most %d chars", nonceMaxLength)
)

// Payload is the exact set of facts a decision witness binds. The fields enact
// the E1 exact decision target and E4 scope requirements of
// plan/research/AUTHORING_DECISION_AUTHORITY_EVIDENCE_GATE_2026-10.md:
//
//   - candidate identity and the exact review projection identity are explicit,
//     so a witness cannot be recomputed or inferred from the candidate alone;
//   - base model revision is explicit, so a stale acceptance cannot masquerade
//     as a current one;
//   - the decision semantics are explicit, so accept and reject cannot be
//     conflated;
//   - the scope digest is explicit, so a witness is not a global boolean.
//
// candidate.origin.actor (proposer provenance) is deliberately absent: the
// decision actor is established by the provider, never by copying origin
// metadata.
type Payload struct {
	WorkspaceIdentity    string
	CandidateFingerprint string
	ReviewFingerprint    string
	BaseModelRevision    string
	Decision             string
	DecisionReasonDigest string
	ScopeDigest          string
	WitnessIssuedAt      string
	WitnessNonce         string
}

// validate reports every field-level defect. It is called before signing,
// before namespace derivation, and again inside the verifier so that a
// malformed payload can never enter a signature check.
func (p Payload) validate() error {
	switch {
	case p.WorkspaceIdentity == "":
		return errEmptyWorkspace
	case len(p.WorkspaceIdentity) > workspacePatternLength:
		return errWorkspaceTooLong
	case !workspacePattern.MatchString(p.WorkspaceIdentity):
		return errInvalidWorkspace
	case !fingerprintPattern.MatchString(p.CandidateFingerprint):
		return errInvalidFingerprint
	case !fingerprintPattern.MatchString(p.ReviewFingerprint):
		return errInvalidFingerprint
	case p.BaseModelRevision == "":
		return errEmptyBaseModelRev
	case !fingerprintPattern.MatchString(p.BaseModelRevision):
		return errInvalidFingerprint
	case p.Decision == "":
		return errEmptyDecision
	case p.Decision != DecisionAccept && p.Decision != DecisionReject:
		return errInvalidDecision
	case !fingerprintPattern.MatchString(p.DecisionReasonDigest):
		return errInvalidFingerprint
	case p.ScopeDigest == "":
		return errEmptyScopeDigest
	case !fingerprintPattern.MatchString(p.ScopeDigest):
		return errInvalidFingerprint
	case p.WitnessIssuedAt == "":
		return errInvalidIssuedAt
	case !issuedAtPattern.MatchString(p.WitnessIssuedAt):
		return errInvalidIssuedAt
	case p.WitnessNonce == "":
		return nil
	case len(p.WitnessNonce) > nonceMaxLength:
		return errNonceTooLong
	case !noncePattern.MatchString(p.WitnessNonce):
		return errInvalidNonce
	}
	return nil
}

// scopeFingerprintHex returns the scope digest's hex part, which is the only
// part embedded in the namespace (the namespace is length-sensitive).
func (p Payload) scopeFingerprintHex() string {
	return strings.TrimPrefix(p.ScopeDigest, "sha256:")
}

// Validate exposes the field-level validation, so callers can reject an
// unverifiable payload before building a signature request around it.
func (p Payload) Validate() error { return p.validate() }

// Message returns the canonical signing message. The field order is fixed and
// the nonce line is omitted only when empty; two payloads that agree on every
// present field produce byte-identical messages, so a signature binds the
// whole payload exactly.
func (p Payload) Message() (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	lines := []string{
		"schemaVersion=" + WitnessSchemaVersion,
		"workspaceIdentity=" + p.WorkspaceIdentity,
		"candidateFingerprint=" + p.CandidateFingerprint,
		"reviewFingerprint=" + p.ReviewFingerprint,
		"baseModelRevision=" + p.BaseModelRevision,
		"decision=" + p.Decision,
		"decisionReasonDigest=" + p.DecisionReasonDigest,
		"scopeDigest=" + p.ScopeDigest,
		"witnessIssuedAt=" + p.WitnessIssuedAt,
	}
	if p.WitnessNonce != "" {
		lines = append(lines, "witnessNonce="+p.WitnessNonce)
	}
	return strings.Join(lines, "\n") + "\n", nil
}

// Namespace derives the provider namespace deterministically from the payload,
// so neither the signer nor the caller can choose an authority scope by
// picking a namespace string. The decision semantics and scope digest reach
// the allowed-signer scope policy exactly as in the Reference fixture.
func (p Payload) Namespace() (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	parts := []string{
		"tei-authoring-v1",
		p.WorkspaceIdentity,
		p.Decision,
		"scope-" + p.scopeFingerprintHex(),
	}
	return strings.Join(parts, ".") + "@tei-platform.invalid", nil
}

// namespacePattern decodes what Namespace produces, including the reserved
// TLD marker that the Reference fixture requires.
var namespacePattern = regexp.MustCompile(
	`^tei-authoring-v1\.([a-z0-9][a-z0-9.-]{0,63})\.(accept|reject)\.scope-([0-9a-f]{64})@tei-platform\.invalid$`,
)

// parseNamespace exposes the workspace identity, decision semantics and scope
// digest hex that were embedded in a namespace string. It returns an error for
// namespaces that are not reference-shaped, so a provider cannot smuggle an
// arbitrary scope through the verification layer.
func parseNamespace(namespace string) (workspace, decision, scopeHex string, err error) {
	m := namespacePattern.FindStringSubmatch(namespace)
	if m == nil {
		return "", "", "", fmt.Errorf("namespace %q is not a reference witness namespace", namespace)
	}
	return m[1], m[2], m[3], nil
}

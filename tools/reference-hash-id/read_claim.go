package hashid

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ReadClaim is an untrusted description of one source read attempt supplied
// by an external Reader/Consumer. It is NOT a TEI Source Index, a Revision
// authority, or an accepted Source record. In particular, SourceScope is the
// source's scope, not the Scheme.Scope used as a hash domain separator.
type ReadClaim struct {
	QualifiedSubject string
	SourceScope      string
	SourceRevision   string
	ReadAttemptID    string
	SourceSHA256     string
}

// ReadClaimDiagnostic is intentionally a non-authoritative, local comparison
// result. SourceReaderVerified and SourceAccepted are always false: observing
// matching caller labels cannot establish the origin or freshness of a read.
type ReadClaimDiagnostic struct {
	Status               string
	Authority            string
	SourceReaderVerified bool
	SourceAccepted       bool
	SourceSHA256         string
}

// CompareUntrustedReadClaims rejects obvious cross-source, cross-scope,
// cross-revision, and cross-attempt label mismatches, then compares the
// declared SHA-256 to the actual bytes supplied by the Consumer.
//
// This operation does NOT read from a workspace or a real tool, authenticate
// either claim, establish that either claim describes the actual consumption
// attempt, or prove Source Acceptance. In particular A -> B -> A with forged
// or reused labels remains indistinguishable: only an authoritative Reader
// and same-attempt consumption evidence can close that gap.
func CompareUntrustedReadClaims(declared, consumed ReadClaim, consumedBytes []byte) (ReadClaimDiagnostic, error) {
	if err := validateReadClaim(declared); err != nil {
		return ReadClaimDiagnostic{}, fmt.Errorf("declared read claim: %w", err)
	}
	if err := validateReadClaim(consumed); err != nil {
		return ReadClaimDiagnostic{}, fmt.Errorf("consumed read claim: %w", err)
	}
	for _, f := range []struct{ name, before, after string }{
		{"qualified subject", declared.QualifiedSubject, consumed.QualifiedSubject},
		{"source scope", declared.SourceScope, consumed.SourceScope},
		{"source revision", declared.SourceRevision, consumed.SourceRevision},
		{"read attempt", declared.ReadAttemptID, consumed.ReadAttemptID},
		{"source digest", declared.SourceSHA256, consumed.SourceSHA256},
	} {
		if f.before != f.after {
			return ReadClaimDiagnostic{}, fmt.Errorf("%s differs between declared and consumed claims", f.name)
		}
	}
	sum := sha256.Sum256(consumedBytes)
	if hex.EncodeToString(sum[:]) != consumed.SourceSHA256 {
		return ReadClaimDiagnostic{}, errors.New("consumed bytes differ from declared source SHA-256")
	}
	return ReadClaimDiagnostic{
		Status:               "caller-claims-consistent-only",
		Authority:            "none",
		SourceReaderVerified: false,
		SourceAccepted:       false,
		SourceSHA256:         consumed.SourceSHA256,
	}, nil
}

func validateReadClaim(c ReadClaim) error {
	for _, f := range []struct{ name, value string }{
		{"qualified subject", c.QualifiedSubject},
		{"source scope", c.SourceScope},
		{"source revision", c.SourceRevision},
		{"read attempt", c.ReadAttemptID},
	} {
		if f.value == "" || len(f.value) > 2048 || !utf8.ValidString(f.value) ||
			strings.TrimSpace(f.value) != f.value {
			return fmt.Errorf("%s must be a nonblank, bounded UTF-8 label", f.name)
		}
		for _, r := range f.value {
			if r < 0x20 || r == 0x7f {
				return fmt.Errorf("%s contains a control character", f.name)
			}
		}
	}
	if len(c.SourceSHA256) != 64 || strings.ToLower(c.SourceSHA256) != c.SourceSHA256 {
		return errors.New("source digest must be exactly 64 lowercase hexadecimal characters")
	}
	if _, err := hex.DecodeString(c.SourceSHA256); err != nil {
		return fmt.Errorf("source digest is not canonical SHA-256: %w", err)
	}
	return nil
}

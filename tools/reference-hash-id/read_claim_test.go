package hashid

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func referenceReadClaim(input []byte) ReadClaim {
	sum := sha256.Sum256(input)
	return ReadClaim{
		QualifiedSubject: "workspace:order/customer-1",
		SourceScope:      "workspace:a",
		SourceRevision:   "revision:7",
		ReadAttemptID:    "attempt:42",
		SourceSHA256:     hex.EncodeToString(sum[:]),
	}
}

func TestReadClaimsAgreeOnlyAsUntrustedCallerLabels(t *testing.T) {
	input := []byte("same raw source")
	declared := referenceReadClaim(input)
	observed := declared
	diagnostic, err := CompareUntrustedReadClaims(declared, observed, input)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostic.Status != "caller-claims-consistent-only" || diagnostic.Authority != "none" ||
		diagnostic.SourceReaderVerified || diagnostic.SourceAccepted ||
		diagnostic.SourceSHA256 != declared.SourceSHA256 {
		t.Fatalf("caller-supplied equality was promoted to Source authority: %+v", diagnostic)
	}
	// SourceScope and the hash derivation Scheme.Scope are intentionally
	// distinct. Neither may be inferred from the other or from the bytes.
	s := fixtureScheme()
	if s.Scope == declared.SourceScope {
		t.Fatal("test fixture accidentally conflates distinct scope roles")
	}
}

func TestReadClaimsRejectCrossSubjectScopeRevisionAndAttempt(t *testing.T) {
	input := []byte("same bytes across source versions")
	declared := referenceReadClaim(input)
	cases := []struct {
		name string
		edit func(*ReadClaim)
	}{
		{"different qualified subject", func(c *ReadClaim) { c.QualifiedSubject = "workspace:order/customer-2" }},
		{"different source scope", func(c *ReadClaim) { c.SourceScope = "workspace:b" }},
		{"different source revision", func(c *ReadClaim) { c.SourceRevision = "revision:8" }},
		{"different read attempt", func(c *ReadClaim) { c.ReadAttemptID = "attempt:43" }},
		{"different source digest", func(c *ReadClaim) { c.SourceSHA256 = strings.Repeat("0", 64) }},
		{"missing source revision", func(c *ReadClaim) { c.SourceRevision = "" }},
		{"missing attempt", func(c *ReadClaim) { c.ReadAttemptID = " " }},
		{"unbounded identity", func(c *ReadClaim) { c.QualifiedSubject = strings.Repeat("x", 2049) }},
		{"control character", func(c *ReadClaim) { c.SourceScope = "scope\nchanged" }},
		{"invalid utf8", func(c *ReadClaim) { c.ReadAttemptID = string([]byte{0xff}) }},
		{"uppercase digest", func(c *ReadClaim) { c.SourceSHA256 = strings.ToUpper(declared.SourceSHA256) }},
		{"short digest", func(c *ReadClaim) { c.SourceSHA256 = "abc" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consumed := declared
			tc.edit(&consumed)
			got, err := CompareUntrustedReadClaims(declared, consumed, input)
			if err == nil || got.Status != "" {
				t.Fatalf("inconsistent caller labels were accepted: diagnostic=%+v err=%v", got, err)
			}
		})
	}
	if got, err := CompareUntrustedReadClaims(declared, declared, []byte("not the same bytes")); err == nil || got.Status != "" {
		t.Fatalf("reported source digest mismatch was accepted: %+v %v", got, err)
	}
}

func TestReadClaimCannotPreventABAWithoutTrustedReaderRevision(t *testing.T) {
	inputA, inputB := []byte("A"), []byte("B")
	claimedBefore := referenceReadClaim(inputA)
	afterB := claimedBefore
	afterB.SourceRevision = "revision:8"
	afterB.ReadAttemptID = "attempt:43"
	afterB.SourceSHA256 = referenceReadClaim(inputB).SourceSHA256
	if _, err := CompareUntrustedReadClaims(claimedBefore, afterB, inputB); err == nil {
		t.Fatal("new revision/attempt must not match original claim")
	}
	afterABA := claimedBefore
	afterABA.SourceRevision = "revision:9"
	afterABA.ReadAttemptID = "attempt:44"
	if _, err := CompareUntrustedReadClaims(claimedBefore, afterABA, inputA); err == nil {
		t.Fatal("same bytes after ABA are not the same revision and read attempt")
	}
	// A malicious/stale caller can reuse old labels after the bytes return
	// to A. This function cannot identify the real reader or observe the
	// intermediate revision. It must never return SourceAccepted=true.
	unknown, err := CompareUntrustedReadClaims(claimedBefore, claimedBefore, inputA)
	if err != nil || unknown.SourceAccepted || unknown.SourceReaderVerified ||
		unknown.Status != "caller-claims-consistent-only" {
		t.Fatalf("caller labels incorrectly established Source Acceptance: %+v %v", unknown, err)
	}
}

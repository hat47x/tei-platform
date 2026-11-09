package opensshwitness

import (
	"strings"
	"testing"
)

func validPayload() Payload {
	return Payload{
		WorkspaceIdentity:    "ws-ref-001",
		CandidateFingerprint: "sha256:" + repeatHex("c"),
		ReviewFingerprint:    "sha256:" + repeatHex("d"),
		BaseModelRevision:    "sha256:" + repeatHex("b"),
		Decision:             DecisionAccept,
		DecisionReasonDigest: "sha256:" + repeatHex("e"),
		ScopeDigest:          "sha256:" + repeatHex("a"),
		WitnessIssuedAt:      "20261007T120000Z",
		WitnessNonce:         "nonce-001",
	}
}

// repeatHex returns a 64-char hex string of the given digit.
func repeatHex(digit string) string {
	return strings.Repeat(digit, 64)
}

func TestMessageIsCanonicalAndDeterministic(t *testing.T) {
	payload := validPayload()
	first, err := payload.Message()
	if err != nil {
		t.Fatalf("payload message: %v", err)
	}
	second, err := payload.Message()
	if err != nil {
		t.Fatalf("payload message again: %v", err)
	}
	if first != second {
		t.Fatalf("message is not deterministic:\n%s\n---\n%s", first, second)
	}
	// The canonical order mirrors the Reference fixture; any reordering would
	// break byte equality with the pressure fixture style.
	want := "schemaVersion=" + WitnessSchemaVersion + "\n" +
		"workspaceIdentity=ws-ref-001\n" +
		"candidateFingerprint=" + "sha256:" + repeatHex("c") + "\n" +
		"reviewFingerprint=sha256:" + repeatHex("d") + "\n" +
		"baseModelRevision=sha256:" + repeatHex("b") + "\n" +
		"decision=accept\n" +
		"decisionReasonDigest=sha256:" + repeatHex("e") + "\n" +
		"scopeDigest=sha256:" + repeatHex("a") + "\n" +
		"witnessIssuedAt=20261007T120000Z\n" +
		"witnessNonce=nonce-001\n"
	if first != want {
		t.Fatalf("canonical message mismatch:\n%q\nwant:\n%q", first, want)
	}
}

func TestMessageOmitsEmptyNonce(t *testing.T) {
	payload := validPayload()
	payload.WitnessNonce = ""
	message, err := payload.Message()
	if err != nil {
		t.Fatalf("payload message: %v", err)
	}
	if strings.Contains(message, "witnessNonce=") {
		t.Fatalf("empty nonce must be omitted:\n%s", message)
	}
	if !strings.HasSuffix(message, "witnessIssuedAt=20261007T120000Z\n") {
		t.Fatalf("message must end after issuedAt:\n%s", message)
	}
}

func TestNamespaceEmbedsScopeAndDecidesSemantics(t *testing.T) {
	payload := validPayload()
	namespace, err := payload.Namespace()
	if err != nil {
		t.Fatalf("namespace: %v", err)
	}
	wantScopeHex := strings.Repeat("a", 64)
	want := "tei-authoring-v1.ws-ref-001.accept.scope-" + wantScopeHex + "@tei-platform.invalid"
	if namespace != want {
		t.Fatalf("namespace mismatch:\n got %s\nwant %s", namespace, want)
	}
	payload.Decision = DecisionReject
	rejectNamespace, err := payload.Namespace()
	if err != nil {
		t.Fatalf("reject namespace: %v", err)
	}
	if !strings.Contains(rejectNamespace, ".reject.scope-") {
		t.Fatalf("reject namespace must embed the reject semantics: %s", rejectNamespace)
	}
}

func TestParseNamespaceRoundTrip(t *testing.T) {
	payload := validPayload()
	namespace, err := payload.Namespace()
	if err != nil {
		t.Fatalf("namespace: %v", err)
	}
	workspace, decision, scopeHex, err := parseNamespace(namespace)
	if err != nil {
		t.Fatalf("parse namespace: %v", err)
	}
	if workspace != payload.WorkspaceIdentity || decision != payload.Decision || scopeHex != payload.scopeFingerprintHex() {
		t.Fatalf("round trip mismatch: %q %q %q", workspace, decision, scopeHex)
	}
}

func TestParseNamespaceRejectsForeignShapes(t *testing.T) {
	for _, bad := range []string{
		"",
		"some-org@EXAMPLE:namespace",
		"tei-authoring-v1.ws-ref-001.accept.scope-66@tei-platform.invalid",
		"tei-authoring-v1.ws-ref-001.allow.scope-" + strings.Repeat("a", 64) + "@tei-platform.invalid",
		"tei-authoring-v1.ws-ref-001.accept.scope-" + strings.Repeat("a", 63) + "@tei-platform.invalid",
		"tei-authoring-v1.ws-ref-001.accept.scope-" + strings.Repeat("a", 64), // missing domain marker
	} {
		if _, _, _, err := parseNamespace(bad); err == nil {
			t.Fatalf("namespace %q must be rejected", bad)
		}
	}
}

func TestPayloadValidationRejectsIncompleteAndMalformed(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(p *Payload)
	}{
		{"empty workspace", func(p *Payload) { p.WorkspaceIdentity = "" }},
		{"uppercase workspace", func(p *Payload) { p.WorkspaceIdentity = "Ws-Ref-001" }},
		{"workspace with space", func(p *Payload) { p.WorkspaceIdentity = "ws ref" }},
		{"workspace too long", func(p *Payload) { p.WorkspaceIdentity = strings.Repeat("w", 65) }},
		{"bad candidate fingerprint", func(p *Payload) { p.CandidateFingerprint = "md5:zz" }},
		{"empty review fingerprint", func(p *Payload) { p.ReviewFingerprint = "" }},
		{"empty base revision", func(p *Payload) { p.BaseModelRevision = "" }},
		{"unknown decision", func(p *Payload) { p.Decision = "auto-accept" }},
		{"empty decision", func(p *Payload) { p.Decision = "" }},
		{"bad reason digest", func(p *Payload) { p.DecisionReasonDigest = "sha256:too" }},
		{"empty scope digest", func(p *Payload) { p.ScopeDigest = "" }},
		{"bad issuedAt", func(p *Payload) { p.WitnessIssuedAt = "2026-10-07T12:00:00Z" }},
		{"issuedAt without T", func(p *Payload) { p.WitnessIssuedAt = "20261007 120000Z" }},
		{"empty issuedAt", func(p *Payload) { p.WitnessIssuedAt = "" }},
		{"nonce with pipe", func(p *Payload) { p.WitnessNonce = "a|b" }},
		{"nonce too long", func(p *Payload) { p.WitnessNonce = strings.Repeat("n", 65) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := validPayload()
			tc.mutate(&payload)
			if err := payload.Validate(); err == nil {
				t.Fatalf("payload with %s must be rejected", tc.name)
			}
			if _, err := payload.Message(); err == nil {
				t.Fatalf("message for payload with %s must fail", tc.name)
			}
			if _, err := payload.Namespace(); err == nil {
				t.Fatalf("namespace for payload with %s must fail", tc.name)
			}
		})
	}
}

func TestValidateAcceptsMinimalPayload(t *testing.T) {
	payload := validPayload()
	payload.WitnessNonce = ""
	if err := payload.Validate(); err != nil {
		t.Fatalf("minimal payload must validate: %v", err)
	}
}

package hoststore

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// trustDoc builds a trust file as JSON so the tests need no Go API for the
// optional fields; extra members are merged into the single issuer entry.
func trustDoc(t *testing.T, k issuerKey, extra map[string]any) []byte {
	t.Helper()
	entry := map[string]any{
		"issuer": k.issuer, "keyId": k.keyID,
		"publicKey":    base64.StdEncoding.EncodeToString(k.public),
		"applications": []string{"order_entry"},
	}
	for name, value := range extra {
		entry[name] = value
	}
	raw, err := json.Marshal(map[string]any{"schema": trustSchema, "issuers": []any{entry}})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func rfc(d time.Duration) string { return time.Now().UTC().Add(d).Format(time.RFC3339) }

func verifyWith(t *testing.T, k issuerKey, extra map[string]any) error {
	t.Helper()
	policy, err := ParseTrustPolicy(trustDoc(t, k, extra))
	if err != nil {
		t.Fatalf("trust file refused: %v", err)
	}
	return policy.Verifier()(context.Background(), signed(t, k, "d1", "application order_entry {}\n"))
}

func wantErr(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || err.Error() != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}

func TestVerifierRefusesExpiredKey(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	wantErr(t, verifyWith(t, k, map[string]any{"notAfter": rfc(-time.Hour)}), "HOST_STORE_KEY_EXPIRED")
}

func TestVerifierRefusesRevokedKey(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	wantErr(t, verifyWith(t, k, map[string]any{"revoked": true}), "HOST_STORE_KEY_REVOKED")
}

func TestVerifierRefusesNotYetValidKey(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	wantErr(t, verifyWith(t, k, map[string]any{"notBefore": rfc(time.Hour)}), "HOST_STORE_KEY_NOT_YET_VALID")
}

func TestRealClockAcceptsKeyInsideWindowAndOmittedFields(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	for name, extra := range map[string]map[string]any{
		"none":        nil,
		"window":      {"notBefore": rfc(-time.Hour), "notAfter": rfc(time.Hour)},
		"only after":  {"notAfter": rfc(time.Hour)},
		"null":        {"notBefore": nil, "notAfter": nil},
		"not revoked": {"revoked": false},
	} {
		if err := verifyWith(t, k, extra); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestParseTrustPolicyRejectsMalformedKeyState(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	pub := base64.StdEncoding.EncodeToString(k.public)
	cases := map[string]map[string]any{
		"garbage":           {"notAfter": "SECRET-GARBAGE"},
		"date only":         {"notAfter": "2030-01-01"},
		"lowercase z":       {"notAfter": "2030-01-01T00:00:00z"},
		"offset":            {"notAfter": "2030-01-01T09:00:00+09:00"},
		"empty":             {"notBefore": ""},
		"whitespace":        {"notAfter": "  "},
		"after before":      {"notBefore": "2030-01-02T00:00:00Z", "notAfter": "2030-01-01T00:00:00Z"},
		"after equals":      {"notBefore": "2030-01-01T00:00:00Z", "notAfter": "2030-01-01T00:00:00Z"},
		"revoked string":    {"revoked": "true"},
		"notAfter number":   {"notAfter": 20300101},
		"notBefore boolean": {"notBefore": true},
	}
	for name, extra := range cases {
		_, err := ParseTrustPolicy(trustDoc(t, k, extra))
		if err == nil || err.Error() != "HOST_STORE_TRUST_INVALID" {
			t.Errorf("%s: got %v", name, err)
			continue
		}
		for _, leak := range []string{pub, "SECRET-GARBAGE", "2030"} {
			if strings.Contains(err.Error(), leak) {
				t.Errorf("%s: error echoes %q", name, leak)
			}
		}
	}
}

func TestTrustedIssuerWithoutKeyStateMarshalsWithoutNewMembers(t *testing.T) {
	raw, err := json.Marshal(TrustedIssuer{Issuer: "i", KeyID: "k", PublicKey: "p", Applications: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"notBefore", "notAfter", "revoked"} {
		if strings.Contains(string(raw), name) {
			t.Errorf("old-style entry emits %s: %s", name, raw)
		}
	}
}

const goldenPayloadSHA256 = "8dccefab636df0629008fd3374318bd36e2896c859fa33e89a12a66b86551657"

// The signed bytes must not change, or existing signed Decisions stop verifying.
func TestSigningPayloadGolden(t *testing.T) {
	d := decision("d1", members(t, "application order_entry {}\n"))
	payload, err := SigningPayload(d)
	if err != nil {
		t.Fatal(err)
	}
	const golden = `{"schema":"tei.host-acceptance-decision/v0","issuer":"test-issuer","record":`
	if !strings.HasPrefix(string(payload), golden) {
		t.Fatalf("payload head changed: %s", payload)
	}
	sum := sha256.Sum256(payload)
	if got := hex.EncodeToString(sum[:]); got != goldenPayloadSHA256 {
		t.Fatalf("signed payload bytes changed: sha256=%s", got)
	}
	if strings.Contains(string(payload), "notBefore") || strings.Contains(string(payload), "revoked") {
		t.Fatalf("key state leaked into the payload: %s", payload)
	}
}

func trustDocEntries(t *testing.T, top map[string]any, entries ...map[string]any) []byte {
	t.Helper()
	doc := map[string]any{"schema": trustSchema, "issuers": entries}
	for name, value := range top {
		doc[name] = value
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func entryFor(k issuerKey, issuer, keyID string, extra map[string]any) map[string]any {
	entry := map[string]any{
		"issuer": issuer, "keyId": keyID,
		"publicKey":    base64.StdEncoding.EncodeToString(k.public),
		"applications": []string{"order_entry"},
	}
	for name, value := range extra {
		entry[name] = value
	}
	return entry
}

// The same public key under two keyIds or issuers would let a holder of a
// revoked key sign with the unrevoked entry.
func TestParseTrustPolicyRejectsSamePublicKeyUnderTwoEntries(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	cases := map[string][]map[string]any{
		"same issuer, other keyId, one revoked": {
			entryFor(k, "issuer-a", "k1", map[string]any{"revoked": true}),
			entryFor(k, "issuer-a", "k2", nil),
		},
		"other issuer": {
			entryFor(k, "issuer-a", "k1", map[string]any{"revoked": true}),
			entryFor(k, "issuer-b", "k1", nil),
		},
		"other applications": {
			entryFor(k, "issuer-a", "k1", nil),
			entryFor(k, "issuer-a", "k2", map[string]any{"applications": []string{"other"}}),
		},
	}
	for name, entries := range cases {
		_, err := ParseTrustPolicy(trustDocEntries(t, nil, entries...))
		if err == nil || err.Error() != "HOST_STORE_TRUST_INVALID" {
			t.Errorf("%s: got %v", name, err)
		}
	}
	// Distinct keys still parse.
	other := newIssuer(t, "issuer-a", "k2")
	if _, err := ParseTrustPolicy(trustDocEntries(t, nil, entryFor(k, "issuer-a", "k1", nil), entryFor(other, "issuer-a", "k2", nil))); err != nil {
		t.Fatalf("distinct keys refused: %v", err)
	}
}

// A misspelled restriction must not be silently ignored.
func TestParseTrustPolicyRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	for name, raw := range map[string][]byte{
		"entry revoke":      trustDoc(t, k, map[string]any{"revoke": true}),
		"entry not_after":   trustDoc(t, k, map[string]any{"not_after": "2020-01-01T00:00:00Z"}),
		"entry expires":     trustDoc(t, k, map[string]any{"expires": "2020-01-01T00:00:00Z"}),
		"entry Revoked key": trustDoc(t, k, map[string]any{"extra": 1}),
		"top level":         trustDocEntries(t, map[string]any{"revoked": []string{"k1"}}, entryFor(k, "issuer-a", "k1", nil)),
		"trailing object":   append(trustDoc(t, k, nil), []byte(` {"schema":"x"}`)...),
		"trailing garbage":  append(trustDoc(t, k, nil), []byte(` x`)...),
	} {
		if _, err := ParseTrustPolicy(raw); err == nil || err.Error() != "HOST_STORE_TRUST_INVALID" {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestParseTrustPolicyStillAcceptsOldTrustFiles(t *testing.T) {
	k := newIssuer(t, "issuer-a", "k1")
	old := `{"schema":"` + trustSchema + `","issuers":[{"issuer":"issuer-a","keyId":"k1","publicKey":"` +
		base64.StdEncoding.EncodeToString(k.public) + `","applications":["order_entry"]}]}` + "\n  "
	policy, err := ParseTrustPolicy([]byte(old))
	if err != nil {
		t.Fatalf("old trust file refused: %v", err)
	}
	if err := policy.Verifier()(context.Background(), signed(t, k, "d1", "application order_entry {}\n")); err != nil {
		t.Fatalf("old trust file no longer verifies: %v", err)
	}
}

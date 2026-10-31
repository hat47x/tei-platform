package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceacceptance/hoststore"
)

func keygenArgs(key string, extra ...string) []string {
	return append([]string{"keygen", "-private-key", key, "-issuer", "iss", "-key-id", "k1", "-application", "order_entry"}, extra...)
}

func runKeygen(t *testing.T, extra ...string) (string, string, error) {
	t.Helper()
	key := filepath.Join(t.TempDir(), "issuer.key")
	var out bytes.Buffer
	err := run(keygenArgs(key, extra...), &out)
	return out.String(), key, err
}

func TestKeygenWithoutNewFlagsKeepsTheExistingOutputShape(t *testing.T) {
	out, key, err := runKeygen(t)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := hoststore.ParseTrustPolicy([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	golden := "{\n  \"schema\": \"tei.host-acceptance-trust/v0\",\n  \"issuers\": [\n    {\n      \"issuer\": \"iss\",\n      \"keyId\": \"k1\",\n      \"publicKey\": \"" +
		policy.Issuers[0].PublicKey + "\",\n      \"applications\": [\n        \"order_entry\"\n      ]\n    }\n  ]\n}\n"
	if out != golden {
		t.Fatalf("output changed:\n%s", out)
	}
	if _, err := os.Stat(key); err != nil {
		t.Fatal(err)
	}
}

func TestKeygenEmitsWindowAndRevokedOnlyWhenAsked(t *testing.T) {
	out, _, err := runKeygen(t, "-not-before", "2030-01-01T00:00:00Z", "-not-after", "2031-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	policy, err := hoststore.ParseTrustPolicy([]byte(out))
	if err != nil {
		t.Fatalf("output does not re-parse: %v", err)
	}
	i := policy.Issuers[0]
	if i.NotBefore == nil || *i.NotBefore != "2030-01-01T00:00:00Z" || i.NotAfter == nil || *i.NotAfter != "2031-01-01T00:00:00Z" || i.Revoked {
		t.Fatalf("unexpected entry: %+v", i)
	}
	if strings.Contains(out, "revoked") {
		t.Fatal("revoked emitted without the flag")
	}

	out, _, err = runKeygen(t, "-revoked")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"revoked": true`) || strings.Contains(out, "notBefore") || strings.Contains(out, "notAfter") {
		t.Fatalf("unexpected output: %s", out)
	}
	if _, err := hoststore.ParseTrustPolicy([]byte(out)); err != nil {
		t.Fatal(err)
	}
}

func TestKeygenRejectsBadWindowsBeforeCreatingTheKey(t *testing.T) {
	for name, extra := range map[string][]string{
		"garbage":       {"-not-after", "tomorrow"},
		"offset":        {"-not-before", "2030-01-01T09:00:00+09:00"},
		"empty":         {"-not-before", ""},
		"after<=before": {"-not-before", "2030-01-02T00:00:00Z", "-not-after", "2030-01-01T00:00:00Z"},
		"equal":         {"-not-before", "2030-01-01T00:00:00Z", "-not-after", "2030-01-01T00:00:00Z"},
	} {
		out, key, err := runKeygen(t, extra...)
		if err == nil {
			t.Errorf("%s: accepted", name)
		}
		if out != "" {
			t.Errorf("%s: printed output", name)
		}
		if _, statErr := os.Stat(key); statErr == nil {
			t.Errorf("%s: private key file was created", name)
		}
	}
}

func TestKeygenNeverOverwritesAnExistingKey(t *testing.T) {
	key := filepath.Join(t.TempDir(), "issuer.key")
	if err := os.WriteFile(key, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(keygenArgs(key, "-not-after", "2030-01-01T00:00:00Z"), &out); err == nil {
		t.Fatal("overwrote an existing key")
	}
	if raw, _ := os.ReadFile(key); string(raw) != "keep" {
		t.Fatal("existing key modified")
	}
}

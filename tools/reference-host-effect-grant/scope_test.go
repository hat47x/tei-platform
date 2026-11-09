package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKeygenScopeIsWrittenShownAndEnforced(t *testing.T) {
	dir := t.TempDir()
	e := env{dir: dir, key: filepath.Join(dir, "issuer.key"), trust: filepath.Join(dir, "trust.json"), grant: filepath.Join(dir, "grant.json")}
	var out bytes.Buffer
	if err := run([]string{"keygen", "-private-key", e.key, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry",
		"-allow-capability", "order_entry.submit_order", "-allow-resource", "order/*"}, &out, clock); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"capabilities"`) || !strings.Contains(out.String(), `"resourcePatterns"`) {
		t.Fatalf("%s", out.String())
	}
	if err := os.WriteFile(e.trust, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(e.signArgs(), &bytes.Buffer{}, clock); err != nil {
		t.Fatal(err)
	}
	var shown bytes.Buffer
	if err := run([]string{"show", "-grant", e.grant, "-trust", e.trust}, &shown, clock); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(shown.String(), "capabilities=order_entry.submit_order") || !strings.Contains(shown.String(), "resourcePatterns=order/*") {
		t.Fatalf("%s", shown.String())
	}
	// A Grant outside the scope is signed (the issuer may sign anything) but
	// refused by show.
	outside := filepath.Join(dir, "grant2.json")
	args := e.signArgs("-resource", "invoice/1")
	for i, a := range args {
		if a == "-out" {
			args[i+1] = outside
		}
	}
	if err := run(args, &bytes.Buffer{}, clock); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"show", "-grant", outside, "-trust", e.trust}, &bytes.Buffer{}, clock)
	if err == nil || err.Error() != "PROVISIONAL_GRANT_OUT_OF_TRUST_SCOPE" {
		t.Fatalf("%v", err)
	}
}

func TestKeygenUnscopedShowsUnrestrictedAndBadScopeLeavesNoKey(t *testing.T) {
	e := setup(t)
	if err := run(e.signArgs(), &bytes.Buffer{}, clock); err != nil {
		t.Fatal(err)
	}
	var shown bytes.Buffer
	if err := run([]string{"show", "-grant", e.grant, "-trust", e.trust}, &shown, clock); err != nil || !strings.Contains(shown.String(), "UNRESTRICTED") {
		t.Fatalf("%v %s", err, shown.String())
	}
	for _, bad := range [][]string{{"-allow-resource", "*"}, {"-allow-resource", "a*b"}, {"-allow-resource", "order/../x"}, {"-allow-capability", ""}} {
		key := filepath.Join(t.TempDir(), "k")
		args := append([]string{"keygen", "-private-key", key, "-issuer", "dev", "-key-id", "k1", "-application", "order_entry"}, bad...)
		if err := run(args, &bytes.Buffer{}, clock); err == nil {
			t.Fatalf("accepted %v", bad)
		}
		if _, err := os.Stat(key); err == nil {
			t.Fatalf("key left behind for %v", bad)
		}
	}
}

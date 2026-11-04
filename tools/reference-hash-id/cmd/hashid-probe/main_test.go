package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	hashid "tei/tools/reference-hash-id"
)

func fixture(t *testing.T, scheme hashid.Scheme, input []byte) (string, string, []byte) {
	t.Helper()
	dir := t.TempDir()
	schemePath := filepath.Join(dir, "scheme.json")
	inputPath := filepath.Join(dir, "source.bin")
	schemeWire, err := hashid.MarshalSchemeJSON(scheme)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemePath, schemeWire, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, input, 0600); err != nil {
		t.Fatal(err)
	}
	return schemePath, inputPath, schemeWire
}

func fixtureScheme() hashid.Scheme {
	return hashid.Scheme{
		SchemeID:         "tei.std.hashId.v1",
		Purpose:          "example.record",
		Scope:            "example.application",
		InputKind:        "bytes",
		Canonicalization: "raw-bytes/v1",
		Algorithm:        hashid.AlgorithmSHA256,
		Mode:             hashid.ModeEncodedDigest,
		OutputBits:       256,
		Encoding:         hashid.EncodingHex,
	}
}

func decimalScheme() hashid.Scheme {
	s := fixtureScheme()
	s.Algorithm = hashid.AlgorithmSHAKE256
	s.Mode = hashid.ModeExactCharacters
	s.OutputBits = 0
	s.Encoding = ""
	s.Alphabet = "0123456789"
	s.Length = 24
	return s
}

func targetSnapshot(t *testing.T, path string, existing []bindingWire) string {
	t.Helper()
	if existing == nil {
		existing = []bindingWire{} // canonical empty snapshot must encode [] rather than null
	}
	doc := targetProbeWire{
		Format: targetWireV1,
		Profile: targetProfileWire{
			Namespace:       "example.external-records",
			ProfileVersion:  "decimal24-v1",
			AllowedAlphabet: "0123456789",
			MinLength:       0,
			MaxLength:       0,
			ExactLength:     24,
			CaseFoldASCII:   false,
			Reserved:        []string{},
		},
		Bindings: existing,
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func runProbe(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestHeadlessDeriveOnlyReportsNoAuthority(t *testing.T) {
	input := []byte("hello TEI")
	s := fixtureScheme()
	schemePath, inputPath, wire := fixture(t, s, input)
	before, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runProbe(t, "-scheme", schemePath, "-input", inputPath, "-subject", "qualified:customer-one")
	if code != 0 || stderr != "" {
		t.Fatalf("derive failed: code=%d stderr=%q", code, stderr)
	}
	var result report
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(input)
	if result.Status != "derived-only" || result.Authority != "none" || result.StorageWrite ||
		result.Subject != "qualified:customer-one" || result.SchemeJSON != string(wire) ||
		result.SourceByteCount != len(input) || result.SourceSHA256 != hex.EncodeToString(sum[:]) ||
		result.HashIDText != "01e745ee1728403d83ebe8ccd903b375f40d40fc1754fd0bc41ca249142f838d" ||
		result.DigestHex != result.HashIDText || result.TargetNamespace != "" {
		t.Fatalf("derive result changed: %+v", result)
	}
	after, err := os.ReadFile(inputPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("preview changed source bytes")
	}
}

func TestHeadlessTargetPreflightAndSnapshotCollision(t *testing.T) {
	input := []byte("source A")
	s := decimalScheme()
	schemePath, inputPath, wire := fixture(t, s, input)
	dir := t.TempDir()
	path := filepath.Join(dir, "snapshot.json")
	result, err := hashid.Derive(s, input)
	if err != nil {
		t.Fatal(err)
	}
	originalSnapshot := targetSnapshot(t, path, nil)
	args := []string{"-scheme", schemePath, "-input", inputPath, "-subject", "qualified:a", "-target-probe", path}
	code, stdout, stderr := runProbe(t, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("preflight failed: code=%d stderr=%q", code, stderr)
	}
	var got report
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "snapshot-preflight-only" || got.Authority != "none" || got.StorageWrite ||
		got.NormalizedTargetKey != result.Text || got.TargetNamespace != "example.external-records" ||
		got.DigestHex != "" {
		t.Fatalf("unexpected preflight status: %+v", got)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != originalSnapshot {
		t.Fatal("preflight must not modify target snapshot")
	}

	entry := bindingWire{
		Namespace:      "example.external-records",
		ProfileVersion: "decimal24-v1",
		Subject:        "qualified:a",
		SchemeJSON:     string(wire),
		Text:           result.Text,
	}
	targetSnapshot(t, path, []bindingWire{entry})
	code, _, stderr = runProbe(t, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("idempotent snapshot rejected: code=%d stderr=%q", code, stderr)
	}
	// Different Subject with an identical physical ID is a collision.
	args[5] = "qualified:b"
	code, stdout, stderr = runProbe(t, args...)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "collision") {
		t.Fatalf("collision was not rejected: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	// A different Scheme cannot silently replace an accepted allocation.
	args[5] = "qualified:a"
	otherScheme := s
	otherScheme.Purpose = "example.another"
	otherWire, err := hashid.MarshalSchemeJSON(otherScheme)
	if err != nil {
		t.Fatal(err)
	}
	entry.SchemeJSON = string(otherWire)
	targetSnapshot(t, path, []bindingWire{entry})
	code, stdout, stderr = runProbe(t, args...)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "different hash scheme") {
		t.Fatalf("scheme migration was not rejected: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestHeadlessRejectsStaleTargetRulesAndDuplicateBindings(t *testing.T) {
	input := []byte("source A")
	s := decimalScheme()
	schemePath, inputPath, schemeJSON := fixture(t, s, input)
	targetPath := filepath.Join(t.TempDir(), "target.json")
	result, err := hashid.Derive(s, input)
	if err != nil {
		t.Fatal(err)
	}
	existing := bindingWire{
		Namespace:      "example.external-records",
		ProfileVersion: "decimal24-v1",
		Subject:        "qualified:a",
		SchemeJSON:     string(schemeJSON),
		Text:           result.Text,
	}
	args := []string{
		"-scheme", schemePath, "-input", inputPath, "-subject", "qualified:a", "-target-probe", targetPath,
	}
	for _, tc := range []struct {
		name     string
		bindings []bindingWire
		edit     func(*targetProbeWire)
	}{
		{
			name:     "duplicate source snapshot",
			bindings: []bindingWire{existing, existing},
		},
		{
			name:     "stale profile version",
			bindings: []bindingWire{existing},
			edit: func(doc *targetProbeWire) {
				doc.Profile.ProfileVersion = "decimal24-v2"
			},
		},
		{
			name:     "unknown target character set",
			bindings: []bindingWire{},
			edit: func(doc *targetProbeWire) {
				doc.Profile.AllowedAlphabet = "012345678"
			},
		},
		{
			name:     "reserved derived value",
			bindings: []bindingWire{},
			edit: func(doc *targetProbeWire) {
				doc.Profile.Reserved = []string{result.Text}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			targetSnapshot(t, targetPath, tc.bindings)
			raw, err := os.ReadFile(targetPath)
			if err != nil {
				t.Fatal(err)
			}
			var doc targetProbeWire
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			if tc.edit != nil {
				tc.edit(&doc)
			}
			canonical, err := json.Marshal(doc)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(targetPath, canonical, 0600); err != nil {
				t.Fatal(err)
			}
			code, stdout, _ := runProbe(t, args...)
			if code != 1 || stdout != "" {
				t.Fatal("invalid target policy/snapshot was accepted")
			}
		})
	}
}

func TestHeadlessRejectsUntrustedInputAndMalformedSnapshot(t *testing.T) {
	s := decimalScheme()
	schemePath, inputPath, wire := fixture(t, s, []byte("source A"))
	targetPath := filepath.Join(t.TempDir(), "snapshot.json")
	original := targetSnapshot(t, targetPath, nil)
	valid := []string{"-scheme", schemePath, "-input", inputPath, "-subject", "qualified:a", "-target-probe", targetPath}
	for _, invalid := range []string{
		" " + original,
		strings.Replace(original, `"format":`, `"format":"bad","format":`, 1),
		strings.Replace(original, `"bindings":[]`, `"bindings":null`, 1),
		strings.Replace(original, `"bindings":[]`, `"bindings":[],"unknown":true`, 1),
		strings.Replace(original, `"profile":`, `"profile":null,"profile":`, 1),
	} {
		if err := os.WriteFile(targetPath, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		code, stdout, _ := runProbe(t, valid...)
		if code != 1 || stdout != "" {
			t.Fatalf("bad snapshot accepted: code=%d text=%q", code, invalid)
		}
	}
	if err := os.WriteFile(targetPath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(schemePath, append(wire, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runProbe(t, valid...)
	if code != 1 || stdout != "" {
		t.Fatal("noncanonical scheme JSON accepted")
	}
	if err := os.WriteFile(schemePath, wire, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(inputPath, bytes.Repeat([]byte{'x'}, maxInputBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = runProbe(t, valid...)
	if code != 1 || stdout != "" {
		t.Fatal("oversized source accepted")
	}
}

func TestHeadlessUsageFailsClosed(t *testing.T) {
	code, stdout, _ := runProbe(t, "-unknown")
	if code != 2 || stdout != "" {
		t.Fatal("unknown command flag did not fail")
	}
	schemePath, inputPath, _ := fixture(t, fixtureScheme(), []byte("source"))
	for _, args := range [][]string{
		{"-scheme", schemePath, "-input", inputPath},
		{"-scheme", schemePath, "-input", inputPath, "-subject", "  "},
		{"-scheme", schemePath, "-input", inputPath, "-subject", "qualified:a", "unexpected"},
	} {
		code, stdout, _ := runProbe(t, args...)
		if code != 2 || stdout != "" {
			t.Fatalf("invalid invocation accepted: args=%q", args)
		}
	}
}

func TestCommittedHeadlessFixture(t *testing.T) {
	// This end-to-end reference fixture intentionally uses only public,
	// non-accepted raw bytes and a non-authoritative empty snapshot.
	dir := filepath.Join("..", "..", "examples")
	args := []string{
		"-scheme", filepath.Join(dir, "decimal24.scheme.json"),
		"-input", filepath.Join(dir, "input.raw"),
		"-subject", "example:record-one",
		"-target-probe", filepath.Join(dir, "decimal24.target-probe.json"),
	}
	code, stdout, stderr := runProbe(t, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("committed reference fixture failed: code=%d stderr=%q", code, stderr)
	}
	secondCode, secondOutput, secondError := runProbe(t, args...)
	if secondCode != 0 || secondError != "" || secondOutput != stdout {
		t.Fatal("same committed Scheme + raw source + snapshot did not reproduce exact JSON")
	}
	var actual report
	if err := json.Unmarshal([]byte(stdout), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.Status != "snapshot-preflight-only" || actual.Authority != "none" ||
		actual.StorageWrite || actual.TargetNamespace != "example.records" ||
		actual.SourceByteCount != len("hello TEI") ||
		actual.HashIDText != actual.NormalizedTargetKey ||
		actual.HashIDText != "590268890628724110017033" || // independent Python SHAKE256 framing golden
		actual.SourceSHA256 != "a53b0563c14046eb542a3aa0eeeb7b3d502a935477b474fbe78d81bf33fb93c4" ||
		len(actual.HashIDText) != 24 {
		t.Fatalf("unexpected reference fixture result: %+v", actual)
	}
	for _, c := range actual.HashIDText {
		if c < '0' || c > '9' {
			t.Fatalf("nondecimal output %q", actual.HashIDText)
		}
	}
}

func TestRealHeadlessCLIExecutableSmoke(t *testing.T) {
	// Unlike the direct run() tests, this uses the actual Go command startup
	// and flag parsing on the committed public fixture, with no writes.
	if _, err := exec.LookPath("go"); err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("Go toolchain required for actual headless CLI smoke in CI")
		}
		t.Skip("Go command unavailable")
	}
	dir := filepath.Join("..", "..", "examples")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	args := []string{
		"run", ".",
		"-scheme", filepath.Join(dir, "decimal24.scheme.json"),
		"-input", filepath.Join(dir, "input.raw"),
		"-subject", "example:record-one",
		"-target-probe", filepath.Join(dir, "decimal24.target-probe.json"),
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("actual headless CLI failed: %v: %s", err, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected CLI stderr %q", stderr.String())
	}
	var got report
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "snapshot-preflight-only" || got.StorageWrite || got.Authority != "none" ||
		got.HashIDText == "" || got.HashIDText != got.NormalizedTargetKey {
		t.Fatalf("unexpected actual CLI output: %+v", got)
	}
}

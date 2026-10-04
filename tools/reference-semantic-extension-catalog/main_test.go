package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeReferenceJSON(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSemanticNamespaceAuthoritiesUsesStrictJSON(t *testing.T) {
	valid := writeReferenceJSON(
		t,
		"authority.json",
		`{
			"namespace":"std.validation",
			"extension":"tei.std.validation",
			"evidenceFingerprint":"sha256:`+strings.Repeat("a", 64)+`"
		}`,
	)
	values, err := loadSemanticNamespaceAuthorities([]string{valid})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 ||
		values[0].Namespace != "std.validation" ||
		values[0].Extension != "tei.std.validation" {
		t.Fatalf("namespace authority=%+v", values)
	}

	unknown := writeReferenceJSON(
		t,
		"unknown.json",
		`{
			"namespace":"std.validation",
			"extension":"tei.std.validation",
			"evidenceFingerprint":"sha256:`+strings.Repeat("b", 64)+`",
			"unexpected":true
		}`,
	)
	if _, err := loadSemanticNamespaceAuthorities([]string{unknown}); err == nil {
		t.Fatal("unknown namespace authority field was accepted")
	}

	trailing := writeReferenceJSON(
		t,
		"trailing.json",
		`{
			"namespace":"std.validation",
			"extension":"tei.std.validation",
			"evidenceFingerprint":"sha256:`+strings.Repeat("c", 64)+`"
		}
		{"namespace":"std.data"}`,
	)
	if _, err := loadSemanticNamespaceAuthorities([]string{trailing}); err == nil {
		t.Fatal("trailing namespace authority JSON was accepted")
	}
}

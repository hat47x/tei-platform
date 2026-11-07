package plancheck

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

// Test-only raw-source witness for OKF K1. It does NOT parse YAML, validate
// OKF v0.2 conformance, qualify TEI meaning, or read actual external sources.
type okfK1RawExpected struct {
	Fixture             string            `json:"fixture"`
	RequiredFiles       []string          `json:"required_files"`
	ConceptCount        int               `json:"concept_count"`
	IndexCount          int               `json:"index_count"`
	RawSHA256           map[string]string `json:"raw_sha256_utf8"`
	OracleQualification string            `json:"oracle_qualification"`
	ValidationStatus    struct {
		OKFFormalConformance string `json:"okf_formal_conformance"`
		OwnerReview          string `json:"owner_review"`
	} `json:"validation_status"`
}

func okfK1Fixture(t *testing.T) (string, okfK1RawExpected) {
	t.Helper()
	root := filepath.Join(repositoryRoot(t), "testdata", "okf-k1-v02")
	var expected okfK1RawExpected
	raw, err := os.ReadFile(filepath.Join(repositoryRoot(t), "testdata", "okf-k1-v02.expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatal(err)
	}
	if expected.Fixture != "okf-k1-v02" || expected.ConceptCount != 3 || expected.IndexCount != 1 {
		t.Fatalf("K1 expected scope changed without qualification: %s, concepts=%d, indexes=%d",
			expected.Fixture, expected.ConceptCount, expected.IndexCount)
	}
	if len(expected.RequiredFiles) != 4 || len(expected.RawSHA256) != 4 {
		t.Fatalf("K1 required set must have 4 pinned files, got required=%d digests=%d",
			len(expected.RequiredFiles), len(expected.RawSHA256))
	}
	if !strings.HasPrefix(expected.OracleQualification, "PROVISIONAL") {
		t.Fatalf("same-attempt oracle must remain provisional: %s", expected.OracleQualification)
	}
	if expected.ValidationStatus.OKFFormalConformance != "NOT RUN" ||
		expected.ValidationStatus.OwnerReview != "NOT RUN" {
		t.Fatal("raw-byte test cannot promote OKF conformance or owner review")
	}
	return root, expected
}

func TestOKFK1RawSourceWitness(t *testing.T) {
	root, expected := okfK1Fixture(t)
	var gotPaths []string
	seen := map[string]bool{}
	for _, rel := range expected.RequiredFiles {
		if !fs.ValidPath(rel) || seen[rel] {
			t.Fatalf("unsafe or repeated K1 path: %q", rel)
		}
		seen[rel] = true
		gotPaths = append(gotPaths, rel)
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) == 0 || !utf8.Valid(raw) || raw[len(raw)-1] != '\n' || strings.Contains(string(raw), "\r") {
			t.Fatalf("fixture must remain nonempty UTF-8 LF with terminal newline: %q", rel)
		}
		actual := sha256.Sum256(raw)
		got := hex.EncodeToString(actual[:])
		if got != expected.RawSHA256[rel] {
			t.Errorf("K1 raw digest mismatch %s: got %s want %s", rel, got, expected.RawSHA256[rel])
		}
		copyRaw := append([]byte(nil), raw...)
		copyRaw[len(copyRaw)-1] ^= 1 // byte mutation must be observable
		mutant := sha256.Sum256(copyRaw)
		if mutant == actual {
			t.Errorf("byte mutant unexpectedly retained digest: %s", rel)
		}
	}
	sort.Strings(gotPaths)
	for _, p := range []string{"business/submit.md", "data/order.md", "functions/submit.md", "index.md"} {
		if !seen[p] {
			t.Errorf("missing required K1 file: %s", p)
		}
	}
	if len(gotPaths) != len(expected.RawSHA256) {
		t.Fatal("required/observed file count mismatch")
	}
}

func TestOKFK1NegativeLinksStayInTemporaryCopies(t *testing.T) {
	root, _ := okfK1Fixture(t)
	for _, entry := range []string{"functions/submit.md", "business/submit.md"} {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry)))
		if err != nil {
			t.Fatal(err)
		}
		// The TEI repository-wide Markdown link check requires valid
		// repository-relative links in committed .md sources.
		if strings.Contains(string(raw), "](/") || strings.Contains(string(raw), "](../missing") {
			t.Fatalf("OKF-specific link mutant leaked into committed baseline: %s", entry)
		}
		if !strings.Contains(string(raw), "](../data/order.md)") {
			t.Fatalf("expected resolvable baseline link missing: %s", entry)
		}
		if _, err := os.Stat(filepath.Join(root, "data", "order.md")); err != nil {
			t.Fatalf("baseline target missing: %v", err)
		}
		mutatedCopy := append([]byte(nil), raw...)
		mutatedCopy = append(mutatedCopy, []byte("\nSee [unresolved](/business/missing.md) and [bundle-root](/data/order.md).\n")...)
		if string(mutatedCopy) == string(raw) ||
			!strings.Contains(string(mutatedCopy), "](/business/missing.md)") ||
			!strings.Contains(string(mutatedCopy), "](/data/order.md)") {
			t.Fatal("synthetic mutation not distinguishable from baseline")
		}
		// We deliberately make no conformance claim here: a real OKF YAML
		// reader must run the temporary mutant to judge F0 versus F1.
	}
}

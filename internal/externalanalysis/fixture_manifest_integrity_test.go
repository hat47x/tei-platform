package externalanalysis

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The synthetic delivery manifest describes raw receipt/view bytes. It must not
// silently inherit a graph-bearing captured fixture from another consumer.
func TestSyntheticFixtureManifestRawIntegrity(t *testing.T) {
	type run struct {
		Kind    string
		Version string
		Stdout  string
		Bytes   int
		SHA256  string
		Lines   int
	}
	var manifest struct {
		Runs []run
	}
	manifestBytes, err := os.ReadFile(filepath.Join("testdata", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("decode fixture manifest: %v", err)
	}
	if len(manifest.Runs) == 0 {
		t.Fatal("synthetic fixture manifest is empty")
	}
	seen := make(map[string]bool, len(manifest.Runs))
	covered := make(map[string]bool, len(manifest.Runs))
	for _, entry := range manifest.Runs {
		entry := entry
		covered[entry.Version+"/"+entry.Kind] = true
		t.Run(entry.Version+"/"+entry.Kind, func(t *testing.T) {
			rel := filepath.Clean(entry.Stdout)
			if rel == "." || filepath.IsAbs(rel) || rel == ".." ||
				strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				t.Fatalf("invalid synthetic fixture path: %q", entry.Stdout)
			}
			if seen[rel] {
				t.Fatalf("duplicate manifest fixture: %s", rel)
			}
			seen[rel] = true
			if entry.Kind != "analyze" && entry.Kind != "view" {
				t.Fatalf("unknown fixture kind %q", entry.Kind)
			}
			raw, err := os.ReadFile(filepath.Join("testdata", rel))
			if err != nil {
				t.Fatalf("read fixture %q: %v", rel, err)
			}
			if len(raw) != entry.Bytes {
				t.Fatalf("%s: manifest bytes=%d, actual bytes=%d", rel, entry.Bytes, len(raw))
			}
			actualDigest := fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
			if actualDigest != entry.SHA256 {
				t.Fatalf("%s: manifest digest=%s, raw digest=%s", rel, entry.SHA256, actualDigest)
			}
			if entry.Kind == "view" {
				scanner := bufio.NewScanner(strings.NewReader(string(raw)))
				lines := 0
				for scanner.Scan() {
					if strings.TrimSpace(scanner.Text()) != "" {
						lines++
					}
				}
				if err := scanner.Err(); err != nil {
					t.Fatal(err)
				}
				if lines != entry.Lines {
					t.Fatalf("%s: manifest lines=%d, actual lines=%d", rel, entry.Lines, lines)
				}
			}
		})
	}
	// A manifest cannot hide a missing baseline by dropping its own row.
	required := []string{"6.7.0/analyze", "6.7.0/view", "6.8.0/analyze", "6.8.0/view"}
	for _, key := range required {
		if !covered[key] {
			t.Errorf("baseline synthetic fixture missing from manifest: %s", key)
		}
	}
}

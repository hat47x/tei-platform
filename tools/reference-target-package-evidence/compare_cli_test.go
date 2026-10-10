package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadEvidenceFileRejectsUnknownAndTrailingJSON(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "report.json")
	valid := comparisonFixture("tei/cmd/one", SourcePackage{
		ImportPath: "tei/cmd/one", GoFiles: []string{"main.go"},
	})
	raw, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, content string
		valid         bool
	}{
		{"valid", string(raw), true},
		{"unknown-field", string(raw[:len(raw)-1]) + `,"claimBinaryMinimal":true}`, false},
		{"duplicate-json-objects", string(raw) + string(raw), false},
		{"trailing-non-json", string(raw) + " garbage", false},
		{"truncated", string(raw[:len(raw)-1]), false},
		{"duplicate-root-property", string(raw[:len(raw)-1]) + `,"entry":"tei/cmd/other"}`, false},
		{"escaped-duplicate-root-property", string(raw[:len(raw)-1]) + `,"\u0065ntry":"tei/cmd/other"}`, false},
		{"duplicate-nested-package-property", strings.Replace(string(raw), `"goFiles":["main.go"]`, `"goFiles":["main.go"],"goFiles":["evil.go"]`, 1), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(file, []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := readEvidenceFile(file)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, error=%v", tc.valid, err)
			}
			if tc.valid && got.Entry != valid.Entry {
				t.Fatalf("entry corrupted: %+v", got)
			}
		})
	}
}

func TestReadEvidenceRejectsDuplicateJSONKeysAtAnyDepth(t *testing.T) {
	valid := comparisonFixture("tei/cmd/one", SourcePackage{
		ImportPath: "tei/cmd/one", GoFiles: []string{"main.go"},
	})
	raw, err := json.Marshal(valid)
	if err != nil {
		t.Fatal(err)
	}
	for _, ambiguous := range []string{
		string(raw[:len(raw)-1]) + `,"entry":"tei/cmd/other"}`,
		string(raw[:len(raw)-1]) + `,"\u0065ntry":"tei/cmd/other"}`,
		strings.Replace(string(raw), `"goFiles":["main.go"]`,
			`"goFiles":["main.go"],"goFiles":["evil.go"]`, 1),
	} {
		err := rejectAmbiguousJSON([]byte(ambiguous))
		if err == nil || !strings.Contains(err.Error(), "duplicate JSON property") {
			t.Fatalf("ambiguous JSON was not specifically rejected: %v", err)
		}
	}
}

func TestReadEvidenceRejectsOversizedOrDeepJSON(t *testing.T) {
	file := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(file, []byte(strings.Repeat(" ", maxEvidenceFileBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readEvidenceFile(file); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversized evidence was accepted: %v", err)
	}
	nested := strings.Repeat("[", 66) + "1" + strings.Repeat("]", 66)
	if err := rejectAmbiguousJSON([]byte(nested)); err == nil || !strings.Contains(err.Error(), "nesting") {
		t.Fatalf("unbounded JSON recursion was accepted: %v", err)
	}
}

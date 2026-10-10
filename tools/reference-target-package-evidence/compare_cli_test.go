package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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

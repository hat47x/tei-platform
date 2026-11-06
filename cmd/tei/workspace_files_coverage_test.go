package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceFilesReportsLoaderConsumption(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, contents := range map[string]string{
		"tei.workspace.yaml":   "workspace: coverage\nformatVersion: 1\nsources:\n  - semantic\n",
		"semantic/model.tei":   "namespace coverage\n",
		"semantic/ignored.TEI": "not parsed\n",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	code, err := workspaceFilesJSON(&out, root, "", 0)
	if err != nil || code != 0 {
		t.Fatalf("workspace-files: code=%d err=%v output=%s", code, err, &out)
	}
	var result struct {
		OK     bool `json:"ok"`
		Result []struct {
			Path   string `json:"path"`
			Parsed *bool  `json:"parsedByLoader"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"semantic/model.tei": true, "semantic/ignored.TEI": false}
	if !result.OK || len(result.Result) != len(want) {
		t.Fatalf("unexpected result: %s", &out)
	}
	for _, file := range result.Result {
		parsed, ok := want[file.Path]
		if !ok || file.Parsed == nil || *file.Parsed != parsed {
			t.Errorf("unexpected file result: %+v", file)
		}
	}
}

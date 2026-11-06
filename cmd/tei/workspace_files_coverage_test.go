package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
			Parsed *bool  `json:"readByLoader"`
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

func TestWorkspaceFilesLoaderConsumptionAtProcessBoundary(t *testing.T) {
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
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	cmd := exec.Command(binary, "workspace-files", "-workspace", root)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil || stderr.Len() != 0 {
		t.Fatalf("workspace-files: err=%v stderr=%q stdout=%q", err, &stderr, &stdout)
	}
	var result struct {
		SchemaVersion string `json:"schemaVersion"`
		OK            bool   `json:"ok"`
		Result        []struct {
			Path string `json:"path"`
			Read *bool  `json:"readByLoader"`
		} `json:"result"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"semantic/model.tei": true, "semantic/ignored.TEI": false}
	if !result.OK || result.SchemaVersion != "tei.cli.workspace-files/v0" || len(result.Result) != len(want) {
		t.Fatalf("unexpected result: %s", &stdout)
	}
	for _, file := range result.Result {
		read, ok := want[file.Path]
		if !ok || file.Read == nil || *file.Read != read {
			t.Errorf("unexpected file result: %+v", file)
		}
		delete(want, file.Path)
	}
	if len(want) != 0 {
		t.Fatalf("missing files: %v", want)
	}
}

func TestWorkspaceFilesPreservesEmptyInventoryJSON(t *testing.T) {
	root := t.TempDir()
	manifest := "workspace: empty\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	code, err := workspaceFilesJSON(&out, root, "", 0)
	if err != nil || code != 0 {
		t.Fatalf("workspace-files: code=%d err=%v output=%s", code, err, &out)
	}
	var result map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if string(result["result"]) != "null" {
		t.Fatalf("empty inventory changed: %s", &out)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The complete set of independent files must be diagnosed without claiming
// semantic validation or a source fingerprint for a partially parsed World.
func TestValidateReportsSyntaxErrorsInIndependentFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "plugins", "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"tei.workspace.yaml":        "workspace: syntax\nformatVersion: 1\nsources:\n  - semantic\n  - plugins\n",
		"semantic/a.tei":            "namespace example\ndata Broken { attribute x type }\n",
		"semantic/b.tei":            "namespace example\ndata Broken2 { frobnicate }\n",
		"semantic/c.tei":            "namespace example\ntype Bad from\n",
		"plugins/broken/plugin.tei": "plugin {\n",
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil || code != 1 {
		t.Fatalf("validate: code=%d err=%v output=%s", code, err, &out)
	}
	var report struct {
		OK         bool `json:"ok"`
		Incomplete *struct {
			Reason string `json:"reason"`
		} `json:"incomplete"`
		WorldFingerprint       string `json:"worldFingerprint"`
		SourceIndexFingerprint string `json:"sourceIndexFingerprint"`
		Diagnostics            []struct {
			Diagnostic struct {
				Code string `json:"code"`
			} `json:"diagnostic"`
			Sources []struct {
				ArtifactPath string `json:"artifactPath"`
			} `json:"sources"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.OK || report.Incomplete == nil || report.Incomplete.Reason != "syntax-coverage-unverified" || report.WorldFingerprint != "" || report.SourceIndexFingerprint != "" {
		t.Fatalf("partial World must not be qualified: %s", &out)
	}
	want := map[string]bool{"semantic/a.tei": true, "semantic/b.tei": true, "semantic/c.tei": true, "plugins/broken/plugin.tei": true}
	for _, diagnostic := range report.Diagnostics {
		if diagnostic.Diagnostic.Code != "PARSE_ERROR" || len(diagnostic.Sources) != 1 || !want[diagnostic.Sources[0].ArtifactPath] {
			t.Fatalf("unexpected diagnostic: %s", &out)
		}
		delete(want, diagnostic.Sources[0].ArtifactPath)
	}
	if len(want) != 0 {
		t.Fatalf("unreported syntax errors: %v; output: %s", want, &out)
	}
	var textOut bytes.Buffer
	textCode, textErr := validateWorkspace(&textOut, root)
	if textErr != nil || textCode != 1 {
		t.Fatalf("text validate: code=%d err=%v output=%s", textCode, textErr, &textOut)
	}
	for _, path := range []string{"semantic/a.tei", "semantic/b.tei", "semantic/c.tei", "plugins/broken/plugin.tei"} {
		if !strings.Contains(filepath.ToSlash(textOut.String()), path) {
			t.Errorf("text diagnostic missing %s: %s", path, &textOut)
		}
	}
	if !strings.Contains(textOut.String(), "意味検証は実行していません") {
		t.Fatalf("text output must state that semantic validation was not run: %s", &textOut)
	}
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	for _, args := range [][]string{{"validate", "-workspace", root}, {"validate", "-workspace", root, "-json"}} {
		cmd := exec.Command(binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 || stderr.Len() != 0 {
			t.Fatalf("CLI %v: err=%v stderr=%s stdout=%s", args, err, &stderr, &stdout)
		}
		for _, path := range []string{"semantic/a.tei", "semantic/b.tei", "semantic/c.tei", "plugins/broken/plugin.tei"} {
			if !strings.Contains(filepath.ToSlash(stdout.String()), path) {
				t.Errorf("CLI %v missing %s: %s", args, path, &stdout)
			}
		}
	}
}

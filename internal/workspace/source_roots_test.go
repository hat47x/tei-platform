package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func writeRootFixture(t *testing.T, root, manifest string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// TestLoadReportsAMissingSourceRoot pins #2823 reproduction 1: a declared root
// that does not exist must not leave validate with only "ok".
func TestLoadReportsAMissingSourceRoot(t *testing.T) {
	root := t.TempDir()
	writeRootFixture(t, root, "workspace: missing_root\nformatVersion: 1\nsources:\n  - nonexistent\n  - semantic\n", map[string]string{
		"semantic/a.txt": "garbage: [\n",
		"semantic/b.tei": "namespace sales\n",
	})
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var missing bool
	for _, finding := range loaded.SourceRootFindings {
		if finding.Root == "nonexistent" && finding.Severity == "error" && strings.Contains(finding.Message, "does not exist") {
			missing = true
		}
	}
	if !missing {
		t.Fatalf("missing source root was not reported: %+v", loaded.SourceRootFindings)
	}
}

// TestLoadReportsAMisspelledSourceRoot pins #2823 reproduction 2: writing
// `semantics` instead of `semantic` must name the misspelling itself. An
// existing but unknown root stays a warning: forward-compatible layout is
// accepted, but it is explicitly not interpreted.
func TestLoadReportsAMisspelledSourceRoot(t *testing.T) {
	root := t.TempDir()
	writeRootFixture(t, root, "workspace: typo_root\nformatVersion: 1\nsources:\n  - semantics\n", map[string]string{
		"semantics/a.tei": "namespace sales\n",
	})
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	var warned bool
	for _, finding := range loaded.SourceRootFindings {
		if finding.Root == "semantics" && finding.Severity == "warning" {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("misspelled source root was not reported: %+v", loaded.SourceRootFindings)
	}
	// Its file is managed by scanManagedFiles, but no semantic declaration was
	// parsed: the World has no sales namespace from it.
	for _, ns := range loaded.World.Semantics {
		if string(ns.ID) == "sales" {
			t.Fatal("uninterpreted root was silently parsed")
		}
	}
}

// TestLoadReportsManagedButUnreadFiles pins #2823 reproduction 3: a managed
// file whose near-miss name the Loader does not read must be named.
func TestLoadReportsManagedButUnreadFiles(t *testing.T) {
	root := t.TempDir()
	writeRootFixture(t, root, "workspace: near_names\nformatVersion: 1\nsources:\n  - semantic\n", map[string]string{
		"semantic/a.txt":     "garbage: [\n",
		"semantic/b.tei.bak": "namespace sales\n",
		"semantic/c.tei":     "namespace sales\n",
	})
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, finding := range loaded.SourceRootFindings {
		if finding.File != "" && finding.Severity == "warning" {
			seen[finding.File] = true
		}
	}
	if !seen["semantic/b.tei.bak"] {
		t.Fatalf("near-miss file not reported: %+v findings=%+v", seen, loaded.SourceRootFindings)
	}
}

// TestReferenceWorkspaceHasNoSourceRootFindings is the positive control: the
// known-good reference Workspace must not gain spurious findings.
func TestReferenceWorkspaceHasNoSourceRootFindings(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join(repoRoot(t), "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.SourceRootFindings) != 0 {
		t.Fatalf("reference workspace gained findings: %+v", loaded.SourceRootFindings)
	}
}

// TestManagedFilesReportReadByLoader pins #2823 requirement 2: the file list
// must distinguish a file the Loader read from one it only inventoried.
func TestManagedFilesReportReadByLoader(t *testing.T) {
	root := t.TempDir()
	writeRootFixture(t, root, "workspace: read_status\nformatVersion: 1\nsources:\n  - semantic\n", map[string]string{
		"semantic/order.tei": "namespace sales\n",
		"semantic/notes.md":  "# managed, not interpreted\n",
	})
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]bool{}
	for _, file := range loaded.ManagedFiles {
		status[file.Path] = file.ReadByLoader
	}
	if !status["semantic/order.tei"] {
		t.Fatalf("read semantic file was not marked read: %+v", loaded.ManagedFiles)
	}
	if status["semantic/notes.md"] {
		t.Fatalf("uninterpreted file was marked read: %+v", loaded.ManagedFiles)
	}
}

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// This test exercises the actual "go list" selection and Git-tracked input
// checks, rather than supplying a hand-written source closure to a BuildPlan.
func TestResolvedGoSourceClosureTracksSelectedCompilerAndEmbedInputs(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain is unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git is unavailable: %v", err)
	}

	repo := t.TempDir()
	stateDir := t.TempDir()
	runSnapshotGit(t, repo, "init")
	runSnapshotGit(t, repo, "config", "user.name", "TEI Go Closure Test")
	runSnapshotGit(t, repo, "config", "user.email", "tei-closure@example.invalid")
	runSnapshotGit(t, repo, "config", "commit.gpgsign", "false")

	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(message string) {
		t.Helper()
		runSnapshotGit(t, repo, "add", "-A")
		runSnapshotGit(t, repo, "commit", "-m", message)
	}
	selected := "cmd/demo/platform_" + runtime.GOOS + ".go"
	otherOS := "windows"
	if runtime.GOOS == otherOS {
		otherOS = "linux"
	}
	excluded := "cmd/demo/platform_" + otherOS + ".go"
	write("go.mod", "module example.invalid/closure\n\ngo 1.23.2\n")
	write("cmd/demo/main.go", "package main\n\nimport _ \"embed\"\n\n//go:embed message.txt\nvar embedded string\n\nfunc main() { println(selected() + embedded) }\n")
	write(selected, "package main\nfunc selected() string { return \"selected\" }\n")
	write(excluded, "package main\nfunc selected() string { return \"excluded\" }\n")
	write("cmd/demo/message.txt", "embedded-before\n")
	write("docs/notes.md", "# untouched compiler inputs\n")
	commit("initial Go compiler and embed inputs")

	resolve := func() ([]goSourceMember, string) {
		t.Helper()
		members, digest, err := resolveGoPackageSourceClosure(repo, goPath, "./cmd/demo", stateDir)
		if err != nil {
			t.Fatal(err)
		}
		return members, digest
	}
	members, before := resolve()
	seen := make(map[string]bool, len(members))
	for _, member := range members {
		if seen[member.Path] {
			t.Fatalf("duplicate source closure member: %s", member.Path)
		}
		seen[member.Path] = true
	}
	for _, required := range []string{
		"go.mod", "cmd/demo/main.go", selected, "cmd/demo/message.txt",
	} {
		if !seen[required] {
			t.Errorf("compiler or embed input missing from closure: %s", required)
		}
	}
	if seen[excluded] || seen["docs/notes.md"] {
		t.Fatalf("unselected platform or documentation entered build closure: %#v", seen)
	}

	snapshotBefore, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	write("docs/notes.md", "# docs-only Git revision\n")
	commit("docs only")
	snapshotDocs, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	_, afterDocs := resolve()
	if snapshotBefore == snapshotDocs || before != afterDocs {
		t.Fatal("docs-only commit must alter provenance but not resolved build sources")
	}

	write(excluded, "package main\nfunc selected() string { return \"excluded-edited\" }\n")
	commit("unselected target changed")
	_, afterExcluded := resolve()
	if afterDocs != afterExcluded {
		t.Fatal("unselected platform source invalidated the selected target closure")
	}

	write(selected, "package main\nfunc selected() string { return \"selected-edited\" }\n")
	commit("selected target changed")
	_, afterSelected := resolve()
	if afterSelected == afterExcluded {
		t.Fatal("selected compiler source change did not invalidate source closure")
	}

	write("cmd/demo/message.txt", "embedded-after\n")
	commit("embedded source changed")
	_, afterEmbed := resolve()
	if afterEmbed == afterSelected {
		t.Fatal("embedded file change did not invalidate source closure")
	}

	// The resolver must reject an input selected by the compiler when it is
	// present on disk but absent from Git's tracked input set.
	write("cmd/demo/untracked.go", "package main\nvar untrackedSource = true\n")
	_, _, err = resolveGoPackageSourceClosure(repo, goPath, "./cmd/demo", stateDir)
	if err == nil || !strings.Contains(err.Error(), "not Git-tracked") {
		t.Fatalf("untracked selected Go source must fail closed, got %v", err)
	}
	if err := os.Remove(filepath.Join(repo, "cmd", "demo", "untracked.go")); err != nil {
		t.Fatal(err)
	}

	// The Git index alone must not authorize a symlink as a build input.
	embedPath := filepath.Join(repo, "cmd", "demo", "message.txt")
	if err := os.Remove(embedPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing-embedded-input.txt", embedPath); err != nil {
		t.Logf("skipping symlink check on this platform: %v", err)
		return
	}
	if _, _, err := resolveGoPackageSourceClosure(repo, goPath, "./cmd/demo", stateDir); err == nil {
		t.Fatal("selected symlink input must not be accepted as a regular build source")
	}
}

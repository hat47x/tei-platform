package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A module resolved by "go list" must not silently enter the build source
// closure when its actual module root is outside the tracked repository.
func TestResolvedGoSourceClosureRejectsLocalReplaceOutsideRepository(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain is unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git is unavailable: %v", err)
	}

	root := t.TempDir()
	repo := filepath.Join(root, "repository")
	dependency := filepath.Join(root, "external")
	for _, dir := range []string{repo, dependency} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, body string) {
		t.Helper()
		fullPath := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("external/go.mod", "module example.invalid/external\n\ngo 1.23.2\n")
	write("external/external.go", "package external\nfunc Message() string { return \"outside\" }\n")
	write("repository/go.mod", "module example.invalid/closure\n\ngo 1.23.2\n\nrequire example.invalid/external v0.0.0\n\nreplace example.invalid/external => ../external\n")
	write("repository/cmd/demo/main.go", "package main\nimport \"example.invalid/external\"\nfunc main() { println(external.Message()) }\n")

	runSnapshotGit(t, repo, "init")
	runSnapshotGit(t, repo, "config", "user.name", "TEI External Module Test")
	runSnapshotGit(t, repo, "config", "user.email", "tei-external@example.invalid")
	runSnapshotGit(t, repo, "config", "commit.gpgsign", "false")
	runSnapshotGit(t, repo, "add", "-A")
	runSnapshotGit(t, repo, "commit", "-m", "tracked module with an external local replacement")

	// Go must first be able to resolve the package. A failure here would
	// otherwise allow an ordinary "go list" error to masquerade as denial.
	goList := exec.Command(goPath, "list", "-deps", "-mod=readonly", "./cmd/demo")
	goList.Dir = repo
	goList.Env = buildEnvironment(os.Environ(), t.TempDir())
	out, err := goList.CombinedOutput()
	if err != nil {
		t.Fatalf("test setup cannot resolve external module: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "example.invalid/external") {
		t.Fatalf("external module was not selected by the Go resolver:\n%s", out)
	}

	_, _, err = resolveGoPackageSourceClosure(repo, goPath, "./cmd/demo", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "contains external module") {
		t.Fatalf("repository-external module must be denied explicitly, got %v", err)
	}
}

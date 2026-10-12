package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The actual Go resolver must select only the host architecture's Go and
// assembly inputs. Changes to unrelated architectures must not invalidate
// an artifact bound to the current target.
func TestResolvedGoSourceClosureSelectsArchitectureAndAssemblyInputs(t *testing.T) {
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
	runSnapshotGit(t, repo, "config", "user.name", "TEI Target Closure Test")
	runSnapshotGit(t, repo, "config", "user.email", "tei-target@example.invalid")
	runSnapshotGit(t, repo, "config", "commit.gpgsign", "false")

	write := func(rel, body string) {
		t.Helper()
		fullPath := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(message string) {
		t.Helper()
		runSnapshotGit(t, repo, "add", "-A")
		runSnapshotGit(t, repo, "commit", "-m", message)
	}

	otherArch := "amd64"
	if runtime.GOARCH == otherArch {
		otherArch = "arm64"
	}
	selectedGo := "cmd/demo/arch_" + runtime.GOARCH + ".go"
	unselectedGo := "cmd/demo/arch_" + otherArch + ".go"
	selectedAsm := "cmd/demo/assembly_" + runtime.GOARCH + ".s"
	unselectedAsm := "cmd/demo/assembly_" + otherArch + ".s"

	write("go.mod", "module example.invalid/target-closure\n\ngo 1.23.2\n")
	write("cmd/demo/main.go", "package main\nfunc main() { println(architecture()) }\n")
	write(selectedGo, "package main\nfunc architecture() string { return \"selected\" }\n")
	write(unselectedGo, "package main\nfunc architecture() string { return \"unselected\" }\n")
	write(selectedAsm, "// selected architecture assembly input\n")
	write(unselectedAsm, "// unselected architecture assembly input\n")
	write("docs/notes.md", "# unrelated documentation\n")
	commit("initial target-specific Go and assembly inputs")

	resolve := func() (map[string]bool, string) {
		t.Helper()
		members, digest, err := resolveGoPackageSourceClosure(repo, goPath, "./cmd/demo", stateDir)
		if err != nil {
			t.Fatal(err)
		}
		seen := make(map[string]bool, len(members))
		for _, member := range members {
			if seen[member.Path] {
				t.Fatalf("duplicate source member: %s", member.Path)
			}
			seen[member.Path] = true
		}
		return seen, digest
	}

	seen, initial := resolve()
	for _, path := range []string{"go.mod", "cmd/demo/main.go", selectedGo, selectedAsm} {
		if !seen[path] {
			t.Errorf("selected target build input missing: %s", path)
		}
	}
	for _, path := range []string{unselectedGo, unselectedAsm, "docs/notes.md"} {
		if seen[path] {
			t.Errorf("unselected input entered the target closure: %s", path)
		}
	}

	beforeSnapshot, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	write("docs/notes.md", "# changed documentation\n")
	write(unselectedGo, "package main\nfunc architecture() string { return \"not-selected\" }\n")
	write(unselectedAsm, "// changed unselected architecture assembly\n")
	commit("change only documentation and unselected architecture inputs")
	afterSnapshot, err := hashRepositorySnapshotTree(repo)
	if err != nil {
		t.Fatal(err)
	}
	if afterSnapshot == beforeSnapshot {
		t.Fatal("committed changes did not update repository provenance")
	}
	_, afterUnselected := resolve()
	if afterUnselected != initial {
		t.Fatal("unselected target and docs changes invalidated the current build closure")
	}

	write(selectedAsm, "// changed selected architecture assembly\n")
	commit("change selected architecture assembly input")
	_, afterAsm := resolve()
	if afterAsm == afterUnselected {
		t.Fatal("selected assembly input change was omitted from closure digest")
	}

	write(selectedGo, "package main\nfunc architecture() string { return \"selected-changed\" }\n")
	commit("change selected Go source")
	_, afterGo := resolve()
	if afterGo == afterAsm {
		t.Fatal("selected Go source change was omitted from closure digest")
	}

	// Callers cannot silently change the target via environment variables:
	// the reference builder selects the same host target as the executable.
	otherOS := "windows"
	if runtime.GOOS == otherOS {
		otherOS = "linux"
	}
	t.Setenv("GOOS", otherOS)
	t.Setenv("GOARCH", otherArch)
	afterOverrideMembers, afterOverride := resolve()
	if afterOverride != afterGo || !afterOverrideMembers[selectedGo] || afterOverrideMembers[unselectedGo] {
		t.Fatalf("caller GOOS/GOARCH environment changed the fixed reference target closure: %s", strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "/"))
	}
}

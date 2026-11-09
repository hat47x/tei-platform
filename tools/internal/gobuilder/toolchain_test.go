package gobuilder

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSelectExecutableDoesNotUsePATH(t *testing.T) {
	dir := t.TempDir()
	goPath := filepath.Join(dir, "selected-go")
	if err := os.WriteFile(goPath, []byte("exact Go binary fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(t.TempDir(), "unrelated"))
	got, err := SelectExecutable(goPath, "/not-selected", "linux")
	if err != nil {
		t.Fatal(err)
	}
	canonicalPath, err := filepath.EvalSymlinks(goPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != canonicalPath {
		t.Fatalf("explicit Go executable changed after canonical path resolution: got %q want %q", got, canonicalPath)
	}
	for _, invalid := range []string{t.TempDir(), filepath.Join(dir, "missing")} {
		if _, err := SelectExecutable(invalid, "", "linux"); err == nil {
			t.Fatalf("invalid Go executable was accepted: %s", invalid)
		}
	}
	if _, err := SelectExecutable("", "", "linux"); err == nil {
		t.Fatal("missing embedded Go root must require an explicit Go executable")
	}
}

func TestSelectExecutableUsesEmbeddedGoRootWithoutPATH(t *testing.T) {
	goRoot := t.TempDir()
	bin := filepath.Join(goRoot, "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	goPath := filepath.Join(bin, "go")
	if err := os.WriteFile(goPath, []byte("pinned"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	got, err := SelectExecutable("", goRoot, "linux")
	if err != nil {
		t.Fatal(err)
	}
	canonicalPath, err := filepath.EvalSymlinks(goPath)
	if err != nil {
		t.Fatal(err)
	}
	if got != canonicalPath {
		t.Fatalf("embedded Go root did not select its canonical exact binary: got %q want %q", got, canonicalPath)
	}
	if goExecutableName("windows") != "go.exe" || goExecutableName("linux") != "go" {
		t.Fatal("target-specific Go executable naming regressed")
	}
}

func TestHashToolchainClosesSelectedRootAndDetectsChanges(t *testing.T) {
	root := t.TempDir()
	write := func(rel, data string) {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("VERSION", "go1.23.2\n")
	write("bin/go", "binary-v1\n")
	write("pkg/tool/linux_amd64/compile", "compiler-v1\n")
	write("src/runtime/runtime.go", "package runtime\n")
	original, err := HashToolchain(root, "linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(original, "sha256:") || len(original) != 71 {
		t.Fatalf("unexpected toolchain digest %q", original)
	}
	same, err := HashToolchain(root, "linux", "amd64")
	if err != nil || same != original {
		t.Fatalf("unchanged Go toolchain changed digest: %s != %s (%v)", original, same, err)
	}
	// A file outside the selected Go root is not an artifact-affecting input.
	if err := os.WriteFile(filepath.Join(t.TempDir(), "unrelated-doc"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	unrelated, err := HashToolchain(root, "linux", "amd64")
	if err != nil || unrelated != original {
		t.Fatalf("unrelated input changed Go toolchain digest: %s (%v)", unrelated, err)
	}
	write("pkg/tool/linux_amd64/compile", "compiler-v2\n")
	updated, err := HashToolchain(root, "linux", "amd64")
	if err != nil || updated == original {
		t.Fatalf("compiler change did not invalidate toolchain digest: %s (%v)", updated, err)
	}
	write("pkg/tool/linux_amd64/compile", "compiler-v1\n")
	write("src/runtime/added.go", "package runtime\n")
	added, err := HashToolchain(root, "linux", "amd64")
	if err != nil || added == original {
		t.Fatalf("source closure addition did not invalidate toolchain digest: %s (%v)", added, err)
	}
}

func TestHashToolchainRejectsMissingRootAndSymlinks(t *testing.T) {
	if _, err := HashToolchain("", "linux", "amd64"); err == nil {
		t.Fatal("empty toolchain root accepted")
	}
	if _, err := HashToolchain(t.TempDir(), "linux", "amd64"); err == nil {
		t.Fatal("incomplete toolchain root accepted")
	}
	if _, err := HashToolchain(filepath.Join(t.TempDir(), "missing"), "linux", "amd64"); err == nil {
		t.Fatal("nonexistent toolchain root accepted")
	}
	if strings.EqualFold(os.Getenv("OS"), "Windows_NT") {
		t.Skip("symlink permission differs on Windows")
	}
	root := t.TempDir()
	for _, path := range []string{"VERSION", "bin/go", "pkg/tool/linux_amd64/compile", "src/runtime.go"} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(root, "src", "runtime.go")
	link := filepath.Join(root, "src", "linked.go")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := HashToolchain(root, "linux", "amd64"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("toolchain closure silently accepted symlink: %v", err)
	}
}

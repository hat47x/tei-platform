package gobuilder

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Build the repository's actual process-plugin source with the shared Go
// adapter. The isolated copies contain its Go dependency closure, not a full
// checkout or an authenticated self-build provenance packet.
func TestReferenceProcessPluginActualAdapterRelocation(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the actual reference provider four times with isolated caches")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repo := filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
	goexe, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	goexe, err = filepath.Abs(goexe)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(goexe, "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{end}}", "./tools/reference-process-plugin")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOFLAGS=")
	listed, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list provider dependencies: %v\n%s", err, listed)
	}
	base := t.TempDir()
	a, b := filepath.Join(base, "workspace-a"), filepath.Join(base, "workspace-b")
	copyFile := func(src, dst string) {
		t.Helper()
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, dest := range []string{a, b} {
		for _, name := range []string{"go.mod", "go.sum"} {
			copyFile(filepath.Join(repo, name), filepath.Join(dest, name))
		}
		for _, dir := range strings.Split(strings.TrimSpace(string(listed)), "\n") {
			dir = strings.TrimSpace(dir)
			if dir == "" {
				continue
			}
			rel, err := filepath.Rel(repo, dir)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				t.Fatalf("nonlocal dependency %q: %v", dir, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
					copyFile(filepath.Join(dir, entry.Name()), filepath.Join(dest, rel, entry.Name()))
				}
			}
		}
	}
	build := func(repo, label string) []byte {
		t.Helper()
		output := filepath.Join(base, "out-"+label)
		if runtime.GOOS == "windows" {
			output += ".exe"
		}
		if err := BuildPackage(repo, goexe, "./tools/reference-process-plugin", output, filepath.Join(base, "state-"+label)); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	first := build(a, "first")
	otherRoot := build(b, "other-root")
	if !bytes.Equal(first, otherRoot) {
		t.Fatal("workspace relocation changed provider bytes under this adapter")
	}
	providerDir := filepath.Join(b, "tools", "reference-process-plugin")
	oldPath, newPath := filepath.Join(providerDir, "main.go"), filepath.Join(providerDir, "relocated.go")
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	otherFile := build(b, "other-file")
	source, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(source, []byte("reference process accepted deployment intent"), []byte("reference process changed deployment intent"), 1)
	if bytes.Equal(source, changed) {
		t.Fatal("expected provider message not found")
	}
	if err := os.WriteFile(newPath, changed, 0o644); err != nil {
		t.Fatal(err)
	}
	modified := build(b, "modified")
	if bytes.Equal(otherFile, modified) {
		t.Fatal("provider source change did not change artifact")
	}
	t.Logf("actual adapter/provider: workspace move preserves bytes; source filename move changes bytes=%t; provider content change changes bytes", !bytes.Equal(first, otherFile))
}

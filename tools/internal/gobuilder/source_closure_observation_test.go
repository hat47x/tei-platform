package gobuilder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fixtureSupply returns a module download-cache directory when the host has one.
func fixtureSupply(t *testing.T) string {
	t.Helper()
	output, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go env unavailable: %v", err)
	}
	cache := filepath.Join(strings.TrimSpace(string(output)), "cache", "download")
	if _, err := os.Stat(cache); err != nil {
		t.Skipf("host download cache unavailable: %v", err)
	}
	return cache
}

// writableTempDir returns a temporary directory whose cleanup can delete
// read-only module-cache files. t.TempDir() fails on those, which would report a
// passing observation as a failed test.
func writableTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "tei-observation-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := removeWritableTree(dir); err != nil {
			t.Errorf("remove isolated state %s: %v", dir, err)
		}
	})
	return dir
}

func removeWritableTree(root string) error {
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0o700)
		}
		return os.Chmod(path, 0o600)
	}); err != nil {
		return err
	}
	return os.RemoveAll(root)
}

// This file pins the observation boundary: an observed closure lists external
// module members, but never reports itself as qualified. The existing resolver
// keeps refusing external module roots.
func TestObserveSourceClosureSeparatesExternalMembers(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("Git needed for the repository fixture: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	observation, err := ObserveSourceClosure(repo, goPath, "./cmd/tei", fixtureSupply(t), writableTempDir(t))
	if err == nil {
		t.Cleanup(func() {})
	}
	if err != nil {
		t.Skipf("the target needs a module supply this environment cannot provide: %v", err)
	}
	if observation.Qualified {
		t.Fatal("observation claimed closure qualification")
	}
	if observation.QualificationWithheld == "" {
		t.Fatal("observation did not state why qualification is withheld")
	}
	if len(observation.ExternalModules) == 0 {
		t.Fatal("cmd/tei selected no external module; the observation case is stale")
	}
	if len(observation.ExternalMembers) == 0 {
		t.Fatal("external module members were not listed")
	}
	if observation.RepositoryDigest == "" || observation.ObservedDigest == "" {
		t.Fatal("digest fields are empty")
	}
	if observation.RepositoryDigest == observation.ObservedDigest {
		t.Fatal("observed digest ignored external members")
	}
	for _, module := range observation.ExternalModules {
		if !strings.Contains(module, "@") {
			t.Fatalf("external module is not version pinned: %s", module)
		}
	}
	// Names must be cache-location independent: module identity plus a
	// module-relative path, never a cache directory.
	for _, member := range observation.ExternalMembers {
		if !strings.HasPrefix(member.Name, "module:") {
			t.Fatalf("external member name is not namespaced: %s", member.Name)
		}
		if strings.Contains(member.Name, "/mnt/") || strings.Contains(member.Name, ":/") {
			t.Fatalf("external member name leaks a cache path: %s", member.Name)
		}
	}
}

func TestObserveSourceClosureReportsSelfContainedTarget(t *testing.T) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("exact Go binary unavailable: %v", err)
	}
	repo, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	observation, err := ObserveSourceClosure(repo, goPath, "./tools/reference-build-authoring-intent", "", writableTempDir(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(observation.ExternalMembers) != 0 || len(observation.ExternalModules) != 0 {
		t.Fatalf("self-contained target reported external members: %+v", observation.ExternalMembers)
	}
	if observation.Qualified {
		t.Fatal("observation claimed qualification even for a self-contained target")
	}
	if len(observation.RepositoryMembers) == 0 {
		t.Fatal("repository members were not listed")
	}
}

func TestObserveFromListKeepsExternalIdentityStable(t *testing.T) {
	root := writableTempDir(t)
	moduleDir := filepath.Join(writableTempDir(t), "github.com", "x", "y@v1.2.3")
	for _, path := range []string{
		filepath.Join(root, "pkg", "main.go"),
		filepath.Join(root, "go.mod"),
		filepath.Join(moduleDir, "pkg", "y.go"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	raw := fmt.Sprintf(`{"Dir":%q,"ImportPath":"example/cmd","Name":"main","Module":{"Path":"example","Dir":%q},"GoFiles":["main.go"]}
{"Dir":%q,"ImportPath":"github.com/x/y/pkg","Name":"pkg","Module":{"Path":"github.com/x/y","Version":"v1.2.3","Dir":%q},"GoFiles":["y.go"]}
`, filepath.Join(root, "pkg"), root, filepath.Join(moduleDir, "pkg"), moduleDir)

	if err := exec.Command("git", "-C", root, "init").Run(); err != nil {
		t.Skipf("git init failed: %v", err)
	}
	if err := exec.Command("git", "-C", root, "add", "pkg/main.go", "go.mod").Run(); err != nil {
		t.Skipf("git add failed: %v", err)
	}
	observation, err := observeFromList(root, "./pkg", []byte(raw))
	if err != nil {
		t.Fatalf("observation failed: %v", err)
	}
	if len(observation.ExternalMembers) != 1 {
		t.Fatalf("external member count: %+v", observation.ExternalMembers)
	}
	if observation.ExternalMembers[0].Name != "module:github.com/x/y@v1.2.3/pkg/y.go" {
		t.Fatalf("external member name is not module-relative: %s", observation.ExternalMembers[0].Name)
	}
	if runtime.GOOS == "windows" && strings.Contains(observation.ExternalMembers[0].Name, `\`) {
		t.Fatalf("member name used a Windows separator: %s", observation.ExternalMembers[0].Name)
	}
}

package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

// makeDirAlias makes link resolve to target: a junction on Windows (needs no
// privilege) and a symlink elsewhere. It skips when the alias cannot be made,
// so a skipped run proves nothing about the race.
func makeDirAlias(t *testing.T, link, target string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
		if err != nil {
			t.Skipf("directory junction unavailable: %v %s", err, out)
		}
		return
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
}

func writeListingWorkspace(t *testing.T, root, environmentDir string, environment []byte) {
	t.Helper()
	dir := filepath.Join(root, "environments", environmentDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: listing_race\nformatVersion: 1\nsources:\n  - environments\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "environment.tei"), environment, 0o644); err != nil {
		t.Fatal(err)
	}
}

// The Workspace root is given as an alias. After Load pinned the real
// directory, the alias is retargeted to a different directory whose
// environment lives under another name. The domain listing must still come
// from the pinned (original) directory: a pathname-based listing would see the
// other directory and then fail to read its names through the pinned Root.
func TestLoadListsDomainSourcesFromPinnedRootWhenAliasIsRetargeted(t *testing.T) {
	environment, err := os.ReadFile(filepath.Join("..", "..", "reference", "workspace", "environments", "production", "environment.tei"))
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	original := filepath.Join(base, "original")
	replacement := filepath.Join(base, "replacement")
	writeListingWorkspace(t, original, "pinned-name", environment)
	writeListingWorkspace(t, replacement, "swapped-name", environment)
	alias := filepath.Join(base, "alias")
	makeDirAlias(t, alias, original)

	retargeted := false
	result, err := loadWithStackHooks(alias, map[string]bool{}, loadHooks{afterPin: func() {
		if err := os.Remove(alias); err != nil {
			t.Fatalf("remove alias: %v", err)
		}
		makeDirAlias(t, alias, replacement)
		retargeted = true
	}})
	if !retargeted {
		t.Fatal("the alias was not retargeted; the test proves nothing")
	}
	if err != nil {
		t.Fatalf("Load must list from the pinned original directory, got error: %v", err)
	}
	var files []string
	for key, location := range result.Sources {
		if len(key) > len("environment:") && key[:len("environment:")] == "environment:" {
			files = append(files, location.File)
		}
	}
	sort.Strings(files)
	want := []string{"environments/pinned-name/environment.tei"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("environment sources = %v, want %v", files, want)
	}
}

// The pinned listing helpers must keep filepath.Glob + sort.Strings semantics
// for every pattern shape the loaders use (for ordinary names), including
// directories that match a file pattern, missing directories and case.
func TestPinnedListingMatchesPathnameGlobOnSyntheticTree(t *testing.T) {
	root := t.TempDir()
	files := []string{
		"applications/b/application.tei", "applications/a/application.tei",
		"applications/c/other.tei", "applications/a/patterns/z.tei", "applications/a/patterns/a.tei",
		"applications/a/patterns/notes.txt", "applications/a/realizations/header-detail/h.tei",
		"applications/A/Application.tei", "applications/loose.tei",
		"databases/d/migrations/2/migration.tei", "databases/d/migrations/10/migration.tei",
		"releases/applications/x/1.tei", "releases/solutions/s/2.tei",
	}
	for _, f := range files {
		path := filepath.Join(root, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// A directory whose name matches a file pattern is still returned by Glob.
	if err := os.MkdirAll(filepath.Join(root, "applications", "a", "patterns", "dir.tei"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "applications", "b", "application.tei.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	abs := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	cases := []struct {
		pattern string
		list    func() ([]string, error)
	}{
		{"applications/*/application.tei", func() ([]string, error) {
			return listPinnedUnitFiles(root, &Result{}, abs("applications"), nil, "application.tei")
		}},
		{"applications/a/patterns/*.tei", func() ([]string, error) {
			return listPinnedFiles(root, &Result{}, abs("applications/a/patterns"), "*.tei")
		}},
		{"applications/*/realizations/header-detail/*.tei", func() ([]string, error) {
			return listPinnedUnitFiles(root, &Result{}, abs("applications"), []string{"realizations", "header-detail"}, "*.tei")
		}},
		{"databases/d/migrations/*/migration.tei", func() ([]string, error) {
			return listPinnedUnitFiles(root, &Result{}, abs("databases/d/migrations"), nil, "migration.tei")
		}},
		{"releases/applications/*/*.tei", func() ([]string, error) {
			return listPinnedUnitFiles(root, &Result{}, abs("releases/applications"), nil, "*.tei")
		}},
		{"releases/solutions/*/*.tei", func() ([]string, error) {
			return listPinnedUnitFiles(root, &Result{}, abs("releases/solutions"), nil, "*.tei")
		}},
		{"environments/*/environment.tei", func() ([]string, error) { // missing directory
			return listPinnedUnitFiles(root, &Result{}, abs("environments"), nil, "environment.tei")
		}},
		{"applications/a/queries/*.tei", func() ([]string, error) { // missing directory
			return listPinnedFiles(root, &Result{}, abs("applications/a/queries"), "*.tei")
		}},
	}
	for _, tc := range cases {
		got, err := tc.list()
		if err != nil {
			t.Fatalf("%s: %v", tc.pattern, err)
		}
		want, err := filepath.Glob(abs(tc.pattern))
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(want)
		if len(got) == 0 && len(want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: got %v, want %v", tc.pattern, got, want)
		}
	}
}

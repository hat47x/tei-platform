package okf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mutateFixture copies the K1 fixture into a temporary directory, applies the
// listed replacements, and returns the copy's path. Keys are bundle-relative
// paths. An empty replacement deletes the file. The committed fixture is never
// modified: the repository's Markdown link check would reject a broken link in a
// managed file, and the fixture's bytes are pinned by expected.json.
func mutateFixture(t *testing.T, replacements map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, fixtureBundle, "", dir, replacements)
	createNewFiles(t, dir, replacements)
	return dir
}

// copyTree copies src into dst, applying replacements keyed by the bundle-relative
// path of each file. Files named only in replacements are created afterwards.
func copyTree(t *testing.T, src, rel, dst string, replacements map[string]string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("list %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	for _, entry := range entries {
		name := entry.Name()
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		source := filepath.Join(src, name)
		target := filepath.Join(dst, name)
		if entry.IsDir() {
			copyTree(t, source, childRel, target, replacements)
			continue
		}
		raw, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read %s: %v", source, err)
		}
		if replacement, ok := replacements[childRel]; ok {
			if replacement == "" {
				continue // delete
			}
			raw = []byte(replacement)
		}
		writeFile(t, target, string(raw))
	}
}

// createNewFiles creates the replacement entries that name a path the fixture does
// not contain.
func createNewFiles(t *testing.T, dir string, replacements map[string]string) {
	t.Helper()
	for name, content := range replacements {
		if content == "" {
			continue
		}
		path := filepath.Join(dir, filepath.FromSlash(name))
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create %s: %v", filepath.Dir(path), err)
		}
		writeFile(t, path, content)
	}
}

// replaceInFile reads a fixture-relative file and replaces old with new. It fails
// when the old text is absent, so a mutation cannot silently become a no-op.
func replaceInFile(t *testing.T, rel, old, new string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fixtureBundle, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	text := string(raw)
	if old == "" {
		return text
	}
	if index := strings.Index(text, old); index >= 0 {
		return text[:index] + new + text[index+len(old):]
	}
	t.Fatalf("%s does not contain %q; the mutation would be a no-op", rel, old)
	return ""
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// osSymlink wraps os.Symlink so the test file needs no extra import.
func osSymlink(oldname, newname string) error {
	return os.Symlink(oldname, newname)
}

// osMkdirAll wraps os.MkdirAll so callers need no extra import.
func osMkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

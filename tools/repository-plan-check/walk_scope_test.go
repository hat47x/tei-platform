package plancheck

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWalkMarkdownSkipsGitIgnoredLocalState pins the scope of the checks that
// walk the repository: git-ignored local state under .local/ holds archived
// checkouts whose own links point outside this repository, so including them
// makes the result depend on what a given machine happens to store there.
func TestWalkMarkdownSkipsGitIgnoredLocalState(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"plan", filepath.Join(".local", "archives")} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join("plan", "kept.md"), "# kept\n")
	write(filepath.Join(".local", "archives", "old.md"), "# old\n")

	var visited []string
	if err := walkMarkdown(root, func(rel, _ string) { visited = append(visited, rel) }); err != nil {
		t.Fatal(err)
	}
	if len(visited) != 1 || visited[0] != "plan/kept.md" {
		t.Fatalf("visited %v, want only plan/kept.md", visited)
	}
}

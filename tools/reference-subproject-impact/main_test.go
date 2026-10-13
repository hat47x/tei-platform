package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSubprojectClassificationKeepsProductionChangesSeparate(t *testing.T) {
	paths := []string{
		"plugins/beta/main.go",
		"plugins/alpha/README.md",
		"plugins/alpha/main.go",
		"plugins/alpha/main_test.go",
		"internal/composition/registry.go",
		"internal/composition/registry_test.go",
		"internal/permission/model.go",
		"core/README.md",
		"internal/plugin/runtime/adapters.go",
		"docs/design.md",
	}
	result := measure(strings.Repeat("1", 40), strings.Repeat("2", 40), paths)
	if result.ChangedPaths != len(paths) {
		t.Fatalf("wrong changed path count: %+v", result)
	}
	if !reflect.DeepEqual(result.PluginSubprojects, []string{"alpha", "beta"}) {
		t.Fatalf("plugin subprojects: %+v", result.PluginSubprojects)
	}
	if !reflect.DeepEqual(result.PluginSourcePaths, []string{"plugins/alpha/main.go", "plugins/beta/main.go"}) {
		t.Fatalf("plugin production sources: %+v", result.PluginSourcePaths)
	}
	if !reflect.DeepEqual(result.CoreCandidateSources, []string{
		"internal/composition/registry.go", "internal/permission/model.go",
	}) {
		t.Fatalf("core candidate sources: %+v", result.CoreCandidateSources)
	}
	if !reflect.DeepEqual(result.CoreCandidateTests, []string{"internal/composition/registry_test.go"}) {
		t.Fatalf("core candidate tests: %+v", result.CoreCandidateTests)
	}
	if result.OtherPaths != 3 {
		t.Fatalf("other paths were not accounted for: %d", result.OtherPaths)
	}
}

func TestExactCommitImpactUsesGitContentAndCountsCoreMoves(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	repo := t.TempDir()
	gitTest(t, repo, "init", "-q")
	gitTest(t, repo, "config", "user.name", "TEI Reference Probe")
	gitTest(t, repo, "config", "user.email", "tei-reference@example.invalid")
	gitTest(t, repo, "config", "commit.gpgsign", "false")
	write := func(rel, body string) {
		t.Helper()
		full := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(message string) string {
		t.Helper()
		gitTest(t, repo, "add", "-A")
		gitTest(t, repo, "commit", "-qm", message)
		return gitTest(t, repo, "rev-parse", "HEAD")
	}
	write("internal/composition/registry.go", "package composition\n")
	write("docs/readme.md", "before\n")
	base := commit("baseline")
	write("plugins/order-quote/main.go", "package main\nfunc main() {}\n")
	write("plugins/order-quote/README.md", "readme\n")
	write("docs/readme.md", "after\n")
	pluginHead := commit("add independent plugin without changing existing core")
	first, err := report(repo, base, pluginHead)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.PluginSubprojects) != 1 || first.PluginSubprojects[0] != "order-quote" ||
		len(first.PluginSourcePaths) != 1 || len(first.CoreCandidateSources) != 0 ||
		first.ChangedPaths != 3 || first.OtherPaths != 1 {
		t.Fatalf("independent plugin change was misclassified: %+v", first)
	}

	gitTest(t, repo, "mv", "internal/composition/registry.go", "plugins/order-quote/registry.go")
	coreMoved := commit("relocate a core file into a plugin")
	moved, err := report(repo, pluginHead, coreMoved)
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.CoreCandidateSources) != 1 ||
		moved.CoreCandidateSources[0] != "internal/composition/registry.go" ||
		len(moved.PluginSourcePaths) != 1 ||
		moved.PluginSourcePaths[0] != "plugins/order-quote/registry.go" ||
		moved.ChangedPaths != 2 {
		t.Fatalf("core file move disappeared through rename detection: %+v", moved)
	}

	empty, err := report(repo, coreMoved, coreMoved)
	if err != nil || empty.ChangedPaths != 0 || len(empty.PluginSubprojects) != 0 {
		t.Fatalf("empty exact commit diff changed files: %+v, %v", empty, err)
	}
	if _, err := report(repo, "main", coreMoved); err == nil || !strings.Contains(err.Error(), "exact") {
		t.Fatalf("moving branch names must not be accepted: %v", err)
	}
	if _, err := report(repo, strings.Repeat("0", 40), coreMoved); err == nil {
		t.Fatal("an unavailable exact Git commit was accepted")
	}
}

func gitTest(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

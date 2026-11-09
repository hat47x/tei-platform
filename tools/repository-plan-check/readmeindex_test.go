package plancheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The indexes that state they list every document in their directory. A new
// document in one of them fails here until the index names it, which is what
// keeps a stated "all" from drifting (#3683).
func TestPlanReadmesMentionEveryEntry(t *testing.T) {
	for _, dir := range []string{"design", "evidence", "research", "workstreams"} {
		problems, err := CheckReadmeMentionsEntries(filepath.Join("..", "..", "plan", dir))
		if err != nil {
			t.Fatal(err)
		}
		if len(problems) > 0 {
			t.Errorf("plan/%s/README.md が載せていないエントリがある:\n%s", dir, strings.Join(problems, "\n"))
		}
	}
}

// tools/README.md states that its table lists every directory here and every
// executable file directly under tools/. The claim drifted twice (#2333 then
// #3683) because nothing checked it, so it is checked now.
func TestToolsReadmeMentionsEveryEntry(t *testing.T) {
	problems, err := CheckReadmeMentionsEntries(filepath.Join("..", "..", "tools"))
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) > 0 {
		t.Errorf("tools/README.md が載せていないエントリがある:\n%s", strings.Join(problems, "\n"))
	}
}

// The check has to fail when an entry is dropped, otherwise it cannot keep the
// statement above true.
func TestToolsReadmeCheckDetectsAMissingEntry(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "reference-new-tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# 索引\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := CheckReadmeMentionsEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(problems) != 1 || !strings.Contains(problems[0], "reference-new-tool") {
		t.Fatalf("a directory absent from the README was not reported: %v", problems)
	}
}

func TestReadmeMentionCheckReportsMissingEntries(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"A.md", "B.md", ".hidden"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("[a](A.md) [sub](sub/)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	problems, err := CheckReadmeMentionsEntries(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(problems, "\n")
	if got != "B.md: README.md に載っていない\nother: README.md に載っていない" {
		t.Fatalf("problems = %q", got)
	}
}

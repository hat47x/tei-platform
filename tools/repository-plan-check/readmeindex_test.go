package plancheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanReadmesMentionEveryEntry(t *testing.T) {
	for _, dir := range []string{"design", "evidence"} {
		problems, err := CheckReadmeMentionsEntries(filepath.Join("..", "..", "plan", dir))
		if err != nil {
			t.Fatal(err)
		}
		if len(problems) > 0 {
			t.Errorf("plan/%s/README.md が載せていないエントリがある:\n%s", dir, strings.Join(problems, "\n"))
		}
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

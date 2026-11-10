package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The check must run from the repository root via the health script, so it is
// verified against the real checkout this test lives in.
func TestAdrIndexCheckAcceptsCurrentRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("reads the working tree")
	}
	if _, err := os.Stat(filepath.Join("..", "..", "plan", "adr", "README.md")); err != nil {
		t.Skip("plan/adr/README.md is absent here")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "run", "tei/tools/reference-adr-index-check", root)
	command.Dir = filepath.Join("..", "..")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("adr-index-check rejected the current repository: %v\n%s", err, output)
	}
}

func TestAdrIndexCheckRejectsBrokenIndex(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a scratch repository")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "plan", "adr")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("0001-adr-adopted.md", "# ADR-0001\n\n- 状態: 採用\n")
	write("0002-adr-proposed.md", "# ADR-0002\n\n- 状態: 提案中\n")
	write("0003-adr-unindexed.md", "# ADR-0003\n\n- 状態: 採用\n")
	write("README.md", strings.Join([]string{
		"- [ADR-0001: adopted](0001-adr-adopted.md)",
		"- [ADR-0002: proposed](0002-adr-proposed.md)",
		"- ADR-0004: deleted but not marked",
		"",
	}, "\n"))

	command := exec.Command("go", "run", "tei/tools/reference-adr-index-check", root)
	command.Dir = filepath.Join("..", "..")
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("broken index was accepted:\n%s", output)
	}
	report := string(output)
	for _, want := range []string{
		"index lacks a status note for 0002",
		"ADR file missing from the index: 0003",
		"unlinked index entry must state 削除: 0004",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q:\n%s", want, report)
		}
	}
}

func TestAdrIndexCheckAcceptsFixedScratchRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a scratch repository")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "plan", "adr")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("0001-adr-adopted.md", "# ADR-0001\n\n- 状態: 採用\n")
	write("README.md", strings.Join([]string{
		"- [ADR-0001: adopted](0001-adr-adopted.md)",
		"- ADR-0002: removed from the public tree（公開前に削除）",
		"",
	}, "\n"))

	command := exec.Command("go", "run", "tei/tools/reference-adr-index-check", root)
	command.Dir = filepath.Join("..", "..")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("well-formed index was rejected: %v\n%s", err, output)
	}
}

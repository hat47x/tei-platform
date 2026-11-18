//go:build !tei_order_only && !tei_repair_only

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The setup printed in the first-change guide must stop when either the
// generated SQL or psql fails. It also must commit the schema and seed
// together. Substitute executable stubs only for the external tools; execute
// the guide's actual shell lines without rewriting them.
func TestFirstChangeGuideSetupFailsClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the guide requires a POSIX shell")
	}
	raw, err := os.ReadFile(filepath.Join("..", "..", "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	begin := strings.Index(text, "## 0. 表を作り、明細を入れる")
	end := strings.Index(text, "## 1. 作業領域の写しを作る")
	if begin < 0 || end <= begin {
		t.Fatal("the guide has no SQL setup section")
	}
	section := text[begin:end]
	markdownFence := strings.Repeat(string(rune(96)), 3)
	start := strings.Index(section, markdownFence+"bash\n")
	if start < 0 {
		t.Fatal("the SQL setup section has no bash block")
	}
	after := section[start+len(markdownFence+"bash\n"):]
	stop := strings.Index(after, markdownFence)
	if stop < 0 {
		t.Fatal("the SQL setup block is not closed")
	}
	script := after[:stop]
	for _, required := range []string{
		"set -eu",
		"schema_sql=\"$(mktemp)\"",
		"trap 'rm -f \"$schema_sql\"' 0",
		"./tei desired-schema -workspace reference/workspace",
		"> \"$schema_sql\"",
		"psql \"$TEI_VIEW_PG_DSN\" -v ON_ERROR_STOP=1 -1",
		"-f \"$schema_sql\" -f testdata/destination-order-change/seed.sql",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("the setup script lacks the failure/transaction guard %q", required)
		}
	}
	if strings.Contains(script, "| psql") {
		t.Fatal("the CLI failure can be hidden by a successful pipeline's final psql")
	}

	for _, tc := range []struct {
		name       string
		ddlExit    string
		psqlExit   string
		fail       bool
		psqlCalled bool
	}{
		{"DDL failure prevents psql", "9", "0", true, false},
		{"psql failure stops setup", "0", "7", true, true},
		{"both steps succeed", "0", "0", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			marker := filepath.Join(root, "psql.called")
			ddlTool := "#!/bin/sh\nprintf 'CREATE TABLE sample(x integer);\\n'\nexit \"$TEI_TEST_DDL_EXIT\"\n"
			psqlTool := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$TEI_TEST_PSQL_MARKER\"\nexit \"$TEI_TEST_PSQL_EXIT\"\n"
			for name, data := range map[string]string{"tei": ddlTool, "psql": psqlTool} {
				if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			fixture := filepath.Join(root, "testdata", "destination-order-change")
			if err := os.MkdirAll(fixture, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(fixture, "seed.sql"), []byte("INSERT INTO sample VALUES (1);\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("sh", "-eu", "-c", script)
			cmd.Dir = root
			cmd.Env = append(os.Environ(),
				"PATH="+root+":"+os.Getenv("PATH"),
				"TMPDIR="+root,
				"TEI_VIEW_PG_DSN=postgres://example.invalid/scratch",
				"TEI_TEST_DDL_EXIT="+tc.ddlExit,
				"TEI_TEST_PSQL_EXIT="+tc.psqlExit,
				"TEI_TEST_PSQL_MARKER="+marker,
			)
			output, runErr := cmd.CombinedOutput()
			if (runErr != nil) != tc.fail {
				t.Fatalf("shell failure=%v want=%v: %s", runErr, tc.fail, output)
			}
			args, readErr := os.ReadFile(marker)
			if tc.psqlCalled {
				if readErr != nil {
					t.Fatalf("the setup never reached psql: %v", readErr)
				}
				for _, required := range []string{"-v ON_ERROR_STOP=1", "-1 -f ", "-f testdata/destination-order-change/seed.sql"} {
					if !strings.Contains(string(args), required) {
						t.Fatalf("psql was not called in one fail-fast transaction: %q", args)
					}
				}
			} else if !os.IsNotExist(readErr) {
				t.Fatalf("psql ran despite failed DDL generation: %v", readErr)
			}
			leftovers, err := filepath.Glob(filepath.Join(root, "tmp.*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("temporary SQL was not removed: files=%v err=%v", leftovers, err)
			}
		})
	}
}

//go:build !tei_order_only && !tei_repair_only

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/tools/internal/postgresdb"
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
		"-f \"$schema_sql\"",
		"-f testdata/destination-order-change/seed-orders.sql",
		"-f testdata/destination-order-change/seed.sql",
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
			if err := os.WriteFile(filepath.Join(fixture, "seed-orders.sql"), []byte("INSERT INTO sample VALUES (2);\n"), 0o644); err != nil {
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
				for _, required := range []string{"-v ON_ERROR_STOP=1", "-1 -f ", "-f testdata/destination-order-change/seed-orders.sql", "-f testdata/destination-order-change/seed.sql"} {
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

// TestFirstChangeGuideSQLSetupIsAtomicInPostgres runs the documented step-0
// shell block with the actual generated CLI, psql and isolated PostgreSQL.
// Stubbed command exits prove control flow; this checks that a late seed
// failure also rolls back the earlier DDL and inserted rows.
func TestFirstChangeGuideSQLSetupIsAtomicInPostgres(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the guide requires a POSIX shell and psql")
	}
	dsn := webReadPostgresDSN(t)
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatal("refusing to edit PostgreSQL without its isolated database confirmation")
	}
	if _, err := exec.LookPath("psql"); err != nil {
		t.Fatal("psql is missing from the real-database acceptance runner")
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal("cannot open the isolated PostgreSQL database")
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := requireIsolatedWebReadDatabase(t.Context(), db, readPostgresDatabase); err != nil {
		t.Fatal(err)
	}
	binary := buildCLIForGuide(t)
	document, err := os.ReadFile(filepath.Join(repoRootForGuide(t), "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(document)
	from := strings.Index(text, "## 0. 表を作り、明細を入れる")
	to := strings.Index(text, "## 1. 作業領域の写しを作る")
	if from < 0 || to <= from {
		t.Fatal("the guide does not delimit the SQL setup step")
	}
	section := text[from:to]
	fence := strings.Repeat(string(rune(96)), 3)
	open := strings.Index(section, fence+"bash\n")
	if open < 0 {
		t.Fatal("the guide has no shell setup block")
	}
	block := section[open+len(fence+"bash\n"):]
	close := strings.Index(block, fence)
	if close < 0 {
		t.Fatal("the guide's shell setup block is unterminated")
	}
	script := block[:close]
	if !strings.Contains(script, "-v ON_ERROR_STOP=1 -1") {
		t.Fatal("the guide no longer asks psql for an atomic transaction")
	}

	originalSeed, err := os.ReadFile(filepath.Join(destinationFixtureRoot(t), "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	parentSeed, err := os.ReadFile(filepath.Join(destinationFixtureRoot(t), "seed-orders.sql"))
	if err != nil {
		t.Fatal(err)
	}
	clearTables := func() {
		t.Helper()
		for _, table := range []string{"public.order_lines", "public.orders"} {
			if _, err := db.Exec("DROP TABLE IF EXISTS " + table); err != nil {
				t.Fatalf("cannot reset the isolated PostgreSQL table %s: %v", table, err)
			}
		}
	}
	t.Cleanup(clearTables)

	for _, tc := range []struct {
		name       string
		breakSeed  bool
		wantRows   int
		wantFailed bool
	}{
		{name: "valid seed commits rows", wantRows: 5},
		{name: "late seed error rolls back DDL and rows", breakSeed: true, wantFailed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearTables()
			root := t.TempDir()
			if err := os.Symlink(binary, filepath.Join(root, "tei")); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, "reference"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(repoRootForGuide(t), "reference", "workspace"),
				filepath.Join(root, "reference", "workspace")); err != nil {
				t.Fatal(err)
			}
			seedDir := filepath.Join(root, "testdata", "destination-order-change")
			if err := os.MkdirAll(seedDir, 0o755); err != nil {
				t.Fatal(err)
			}
			// The guide runs the parent seed before the detail seed.
			// Missing parent input is not a legitimate rollback test.
			if err := os.WriteFile(filepath.Join(seedDir, "seed-orders.sql"), parentSeed, 0o644); err != nil {
				t.Fatal(err)
			}
			seed := append([]byte(nil), originalSeed...)
			if tc.breakSeed {
				// A deliberate SQL failure after the original INSERT proves
				// that previously applied DDL and rows are rolled back.
				seed = append(seed, []byte("\nSELECT 1 / 0;\n")...)
			}
			if err := os.WriteFile(filepath.Join(seedDir, "seed.sql"), seed, 0o644); err != nil {
				t.Fatal(err)
			}

			command := exec.Command("sh", "-eu", "-c", script)
			command.Dir = root
			command.Env = append(os.Environ(), "TMPDIR="+root, "TEI_VIEW_PG_DSN="+dsn)
			// Output can carry a DSN. Do not echo it into CI failure logs.
			_, runErr := command.CombinedOutput()
			if failed := runErr != nil; failed != tc.wantFailed {
				t.Fatalf("step-0 shell failed=%v, want failure=%v", failed, tc.wantFailed)
			}
			var detailsExist, parentsExist bool
			if err := db.QueryRowContext(t.Context(),
				"SELECT to_regclass('public.order_lines') IS NOT NULL, to_regclass('public.orders') IS NOT NULL").
				Scan(&detailsExist, &parentsExist); err != nil {
				t.Fatal("cannot inspect the isolated parent and detail tables")
			}
			if tc.wantFailed {
				if detailsExist || parentsExist {
					t.Fatal("the guide left a parent or detail table after a failed detail seed")
				}
			} else {
				if !detailsExist || !parentsExist {
					t.Fatal("the guide did not commit both generated tables")
				}
				var parentCount int
				if err := db.QueryRowContext(t.Context(),
					"SELECT count(*) FROM public.orders WHERE order_id IN (1001, 1002, 1003)").
					Scan(&parentCount); err != nil {
					t.Fatal("cannot read committed parent orders")
				}
				if parentCount != 3 {
					t.Fatalf("committed parent orders=%d, want 3", parentCount)
				}
				var count int
				if err := db.QueryRowContext(t.Context(),
					"SELECT count(*) FROM public.order_lines").Scan(&count); err != nil {
					t.Fatal("cannot inspect the committed seed rows")
				}
				if count != tc.wantRows {
					t.Fatalf("the guide committed %d rows, want %d", count, tc.wantRows)
				}
			}
			leftovers, err := filepath.Glob(filepath.Join(root, "tmp.*"))
			if err != nil || len(leftovers) != 0 {
				t.Fatalf("the guide left generated SQL tempfiles: count=%d err=%v", len(leftovers), err)
			}
		})
	}
}

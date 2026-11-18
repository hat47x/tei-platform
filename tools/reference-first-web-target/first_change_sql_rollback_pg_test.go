//go:build !tei_order_only && !tei_repair_only

package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/tools/internal/postgresdb"
)

// TestFirstChangeGuideAppliesSchemaAndSeedAtomicallyInPostgres executes the
// exact shell block in the first-use guide against the isolated PostgreSQL
// database. Only the DDL producer is replaced by a disposable executable.
// Unlike the stub-only shell test, the psql -1 transaction and rollback are
// observed in the database itself.
func TestFirstChangeGuideAppliesSchemaAndSeedAtomicallyInPostgres(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the first-use guide requires a POSIX shell and psql")
	}
	dsn := os.Getenv(readPostgresDSNEnv)
	if dsn == "" {
		t.Skip(readPostgresDSNEnv + " is required for a real PostgreSQL rollback test")
	}
	if os.Getenv(readPostgresNameEnv) != readPostgresDatabase {
		t.Fatalf("refusing DDL without the isolated PostgreSQL confirmation")
	}
	if _, err := exec.LookPath("psql"); err != nil {
		t.Fatal("psql must be installed for the first-use guide's real PostgreSQL test")
	}
	db, err := postgresdb.OpenSQL(dsn)
	if err != nil {
		t.Fatal("cannot open the isolated PostgreSQL connection")
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := requireIsolatedWebReadDatabase(t.Context(), db, readPostgresDatabase); err != nil {
		t.Fatal(err)
	}

	guide, err := os.ReadFile(filepath.Join("..", "..", "docs", "first-change.md"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(guide)
	begin := strings.Index(body, "## 0. 表を作り、明細を入れる")
	end := strings.Index(body, "## 1. 作業領域の写しを作る")
	if begin < 0 || end <= begin {
		t.Fatal("the first-use guide has no SQL setup section")
	}
	section := body[begin:end]
	fence := strings.Repeat(string(rune(96)), 3)
	start := strings.Index(section, fence+"bash\n")
	if start < 0 {
		t.Fatal("the setup section has no POSIX shell block")
	}
	after := section[start+len(fence+"bash\n"):]
	stop := strings.Index(after, fence)
	if stop < 0 {
		t.Fatal("the setup shell block has no closing fence")
	}
	script := after[:stop]

	for _, tc := range []struct {
		name        string
		table       string
		secondValue int
		shouldFail  bool
	}{
		{"duplicate_seed_rolls_back_DDL", "tei_first_change_atomic_failure_probe", 1, true},
		{"valid_seed_commits_DDL_and_rows", "tei_first_change_atomic_success_probe", 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// This name is a constant chosen by the test, never from a user
			// or a Workspace. Check absence BEFORE creating anything; a
			// previous table must never be silently overwritten.
			var before sql.NullString
			err := db.QueryRowContext(t.Context(),
				"select pg_catalog.to_regclass($1)::text", "public."+tc.table).Scan(&before)
			if err != nil {
				t.Fatal("cannot check the disposable table's initial state")
			}
			if before.Valid {
				t.Fatalf("disposable table %s already exists", tc.table)
			}
			t.Cleanup(func() {
				if _, err := db.ExecContext(t.Context(),
					"drop table if exists public."+tc.table); err != nil {
					t.Errorf("failed to remove disposable table %s", tc.table)
				}
			})

			dir := t.TempDir()
			ddl := fmt.Sprintf("#!/bin/sh\nprintf 'CREATE TABLE public.%s (id integer PRIMARY KEY);\\n'\n", tc.table)
			if err := os.WriteFile(filepath.Join(dir, "tei"), []byte(ddl), 0o755); err != nil {
				t.Fatal(err)
			}
			seedDir := filepath.Join(dir, "testdata", "destination-order-change")
			if err := os.MkdirAll(seedDir, 0o755); err != nil {
				t.Fatal(err)
			}
			seed := fmt.Sprintf("INSERT INTO public.%s (id) VALUES (1);\n"+
				"INSERT INTO public.%s (id) VALUES (%d);\n", tc.table, tc.table, tc.secondValue)
			if err := os.WriteFile(filepath.Join(seedDir, "seed.sql"), []byte(seed), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("sh", "-eu", "-c", script)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(),
				"TMPDIR="+dir,
				"TEI_VIEW_PG_DSN="+dsn,
			)
			// Do not print psql output: a connection error may include
			// connection details. The outcome is independently read below.
			_, runErr := cmd.CombinedOutput()
			if (runErr != nil) != tc.shouldFail {
				t.Fatalf("guide setup failed=%t, expected=%t", runErr != nil, tc.shouldFail)
			}

			var afterTable sql.NullString
			if err := db.QueryRowContext(t.Context(),
				"select pg_catalog.to_regclass($1)::text", "public."+tc.table).
				Scan(&afterTable); err != nil {
				t.Fatal("cannot verify the PostgreSQL transaction's table state")
			}
			if tc.shouldFail {
				// With -1, even the successful CREATE TABLE and first
				// INSERT must roll back after the second INSERT fails.
				if afterTable.Valid {
					t.Fatalf("DDL survived a failed seed: %s", tc.table)
				}
				return
			}
			if !afterTable.Valid {
				t.Fatal("successful schema and seed were not committed")
			}
			var count int
			if err := db.QueryRowContext(t.Context(),
				"select count(*) from public."+tc.table).Scan(&count); err != nil {
				t.Fatal("cannot read the committed seed")
			}
			if count != 2 {
				t.Fatalf("committed seed rows=%d, want two", count)
			}
		})
	}
}

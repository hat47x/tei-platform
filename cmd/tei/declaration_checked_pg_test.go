//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// The declaration the binding makes and the qualification this read performs
// were reported separately, so one answer could say `coverage: complete` and
// `declaration covers ... (unchecked)` at the same time. A reader then cannot
// tell that the declaration was in fact confirmed.
//
// This test pins both directions against a real PostgreSQL:
//   - a binding that declares the relation and a Query the qualification
//     matches is reported as checked;
//   - a Query that drops a row leaves the declaration unchecked, because the
//     qualification did not match what the binding declared.
func TestEnvironmentReportsWhetherTheDeclarationWasChecked(t *testing.T) {
	const databaseName = "tei_destination_order_e2e"
	dsn := os.Getenv("TEI_VIEW_PG_DSN")
	if dsn == "" {
		t.Skip("TEI_VIEW_PG_DSN is unset; real PostgreSQL not run")
	}
	if os.Getenv("TEI_VIEW_PG_CONFIRM_ISOLATED") != databaseName {
		t.Fatal("refusing to modify the database without the disposable-database confirmation")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var actual string
	if err := db.QueryRow("select current_database()").Scan(&actual); err != nil || actual != databaseName {
		t.Fatalf("expected %q, got %q: %v", databaseName, actual, err)
	}
	if _, err := db.Exec("drop table if exists public.order_lines"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`create table public.order_lines (
		order_line_id bigint primary key,
		order_id bigint not null,
		line_number bigint not null,
		item_code varchar not null,
		quantity numeric(12,3) not null,
		unit_price numeric(12,2) not null
	)`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec("drop table if exists public.order_lines") })
	seed, err := os.ReadFile(filepath.Join("..", "..", "testdata", "destination-order-change", "seed.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		t.Fatal(err)
	}

	binary := buildCLIForLifecycle(t)
	t.Setenv("TEI_VIEW_PG_DSN", dsn)

	var report struct {
		OK     bool `json:"ok"`
		Result struct {
			Coverage    string `json:"coverage"`
			Declaration *struct {
				CoversRelation string `json:"coversRelation"`
				Checked        bool   `json:"checked"`
			} `json:"declaration"`
		} `json:"result"`
	}
	readReport := func(t *testing.T, stdout string) {
		t.Helper()
		report = struct {
			OK     bool `json:"ok"`
			Result struct {
				Coverage    string `json:"coverage"`
				Declaration *struct {
					CoversRelation string `json:"coversRelation"`
					Checked        bool   `json:"checked"`
				} `json:"declaration"`
			} `json:"result"`
		}{}
		if err := json.Unmarshal([]byte(stdout), &report); err != nil || !report.OK {
			t.Fatalf("cannot read the report: %v\n%s", err, stdout)
		}
		if report.Result.Declaration == nil {
			t.Fatal("the report dropped the declaration")
		}
	}

	// The reference binding declares `covers sales.order.lines` and its accepted
	// SQL reads the mapped table with only the parent condition, so the
	// qualification matches and the declaration is checked.
	t.Run("matching Query is reported as checked", func(t *testing.T) {
		root := t.TempDir()
		copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-environment", "development",
			"-input", "order_id=1002", "-format", "json",
		)
		if got.code != 0 {
			t.Fatalf("view failed: %s%s", got.stdout, got.stderr)
		}
		readReport(t, got.stdout)
		if report.Result.Coverage != "complete" {
			t.Fatalf("coverage=%q; the reference binding should qualify", report.Result.Coverage)
		}
		if !report.Result.Declaration.Checked {
			t.Fatalf("coverage is complete but the declaration is still unchecked: %+v", report.Result.Declaration)
		}
	})

	// The same declaration with a Query that drops a row: the qualification
	// cannot match, so coverage stays partial and the declaration stays
	// unchecked. Unconfirmed is not the same as false.
	t.Run("a Query that drops a row leaves it unchecked", func(t *testing.T) {
		root := t.TempDir()
		copyTreeForDestination(t, filepath.Join("..", "..", "reference", "workspace"), root)
		sourcePath := filepath.Join(root, "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql")
		source, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		const predicate = "WHERE order_id = :order_id"
		if strings.Count(string(source), predicate) != 1 {
			t.Fatalf("the reference Query's parent predicate changed: %s", source)
		}
		edited := strings.Replace(string(source), predicate, predicate+" AND order_line_id <> 3", 1)
		if err := os.WriteFile(sourcePath, []byte(edited), 0o644); err != nil {
			t.Fatal(err)
		}
		got := runCLIProcessWithStreams(t, binary,
			"view", "-workspace", root, "-application", "order_entry",
			"-pattern", "order_entry.order_lines", "-environment", "development",
			"-input", "order_id=1001", "-format", "json",
		)
		if got.code != 0 {
			t.Fatalf("view failed: %s%s", got.stdout, got.stderr)
		}
		readReport(t, got.stdout)
		if report.Result.Coverage != "partial" {
			t.Fatalf("coverage=%q; a Query with an extra condition must stay partial", report.Result.Coverage)
		}
		if report.Result.Declaration.Checked {
			t.Fatal("a partial qualification reported the declaration as checked")
		}
	})
}

// The text form must say the same thing: a confirmed declaration prints
// `(checked)`, otherwise `(unchecked)`.
func TestTextFormSaysCheckedForAConfirmedDeclaration(t *testing.T) {
	var confirmed bytes.Buffer
	if code, err := writeDestinationReadText(&confirmed, destinationRead{
		Application: "order_entry", Pattern: "order_entry.order_lines",
		Environment: "development", Relation: "sales.order.lines",
		Coverage: "complete",
		Declaration: &destinationDeclaration{
			CoversRelation: "sales.order.lines", Checked: true,
		},
	}); err != nil || code != 0 {
		t.Fatalf("text rendering failed: code=%d err=%v", code, err)
	}
	if !strings.Contains(confirmed.String(), "declaration covers sales.order.lines (checked)\n") {
		t.Fatalf("a confirmed declaration did not print (checked):\n%s", confirmed.String())
	}
	if strings.Contains(confirmed.String(), "(unchecked)") {
		t.Fatalf("a confirmed declaration still printed (unchecked):\n%s", confirmed.String())
	}
}

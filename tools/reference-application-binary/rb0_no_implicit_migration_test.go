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

	"github.com/hat47x/tei-platform/tools/reference-application-binary/internal/appgen"
	_ "modernc.org/sqlite"
)

// RBM-01 / RBM-01b: generated normal operations may use an existing DB,
// but must never create one or apply the bundled PostgreSQL migration.
func TestRB0GeneratedBinaryDoesNotImplicitlyMigrate(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("requires native generated-binary execution")
	}

	workspace := filepath.Join(repoRoot(t), "reference", "workspace")
	model, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("extract model: %v", err)
	}
	sourceDir := t.TempDir()
	if _, err := appgen.Generate(model, sourceDir); err != nil {
		t.Fatalf("generate: %v", err)
	}
	migrations, err := os.ReadFile(filepath.Join(sourceDir, "migrations.go"))
	if err != nil {
		t.Fatalf("read generated migration: %v", err)
	}
	if !strings.Contains(string(migrations), "func applyMigrations(") ||
		!strings.Contains(string(migrations), "ON public.orders") {
		t.Fatal("fixture must retain its PostgreSQL-only bundled migration")
	}

	binary := filepath.Join(t.TempDir(), "order-entry")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if _, err := appgen.Build(workspace, "order_entry", sourceDir, binary, ""); err != nil {
		t.Fatalf("build generated binary: %v", err)
	}

	missing := filepath.Join(t.TempDir(), "does-not-exist.db")
	for _, command := range [][]string{
		{"query", "order_entry.recent_orders", "1"},
		{"command", "order_entry.submit_order"},
	} {
		cmd := exec.Command(binary, command...)
		cmd.Env = append(os.Environ(), "TEI_DSN="+missing)
		output, err := cmd.CombinedOutput()
		if err == nil || !strings.Contains(string(output), "database not initialized") {
			t.Fatalf("%v must reject missing DB without creating it: %v, %s", command, err, output)
		}
		if _, err := os.Stat(missing); !os.IsNotExist(err) {
			t.Fatalf("normal operation created DB %q: %v", missing, err)
		}
	}

	// Validation is pure in this generated reference and needs no DB or DSN.
	validate := exec.Command(binary, "validate", "sales.order_line.quantity_positive")
	validate.Env = append(os.Environ(), "TEI_DSN="+missing)
	validate.Stdin = strings.NewReader(`{"quantity":"2","lineNumber":1}`)
	out, err := validate.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "ok" {
		t.Fatalf("DB-independent validation: %v, %s", err, out)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("validation created DB %q: %v", missing, err)
	}

	existing := filepath.Join(t.TempDir(), "orders.db")
	db, err := sql.Open("sqlite", existing)
	if err != nil {
		t.Fatalf("create isolated fixture db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY, customer_id TEXT, amount TEXT, currency TEXT, created_at TEXT);
		INSERT INTO orders VALUES ('o1','c1','100','USD','2026-11-14');`); err != nil {
		t.Fatalf("seed DB: %v", err)
	}
	before := rb0SchemaNames(t, db)

	query := exec.Command(binary, "query", "order_entry.recent_orders", "1")
	query.Env = append(os.Environ(), "TEI_DSN="+existing)
	out, err = query.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "o1") {
		t.Fatalf("query on existing DB should not execute PG migration: %v, %s", err, out)
	}
	// The generated command may write normal business data, but no migration DDL.
	command := exec.Command(binary, "command", "order_entry.submit_order")
	command.Env = append(os.Environ(), "TEI_DSN="+existing)
	command.Stdin = strings.NewReader(`{"id":"o2","customerID":"c1","amount":"200","currency":"USD"}`)
	out, err = command.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "committed") {
		t.Fatalf("business command on existing DB: %v, %s", err, out)
	}
	if after := rb0SchemaNames(t, db); after != before {
		t.Fatalf("normal operations modified SQLite schema: before=%s after=%s", before, after)
	}
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count); err != nil || count != 2 {
		t.Fatalf("normal command effect not independently observed: count=%d err=%v", count, err)
	}

	// A hostile mode=rwc in the DSN must not override the generated read-only
	// connection or initialize a missing database.
	fileURI := fmt.Sprintf("file:%s?mode=rwc", filepath.ToSlash(missing))
	query = exec.Command(binary, "query", "order_entry.recent_orders", "1")
	query.Env = append(os.Environ(), "TEI_DSN="+fileURI)
	out, err = query.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "database not initialized") {
		t.Fatalf("URI with mode=rwc must not initialize: %v, %s", err, out)
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatalf("SQLite URI created DB %q: %v", missing, err)
	}
}

func rb0SchemaNames(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query("SELECT type, name, sql FROM sqlite_schema WHERE name NOT LIKE 'sqlite_%' ORDER BY type, name")
	if err != nil {
		t.Fatalf("read SQLite schema: %v", err)
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var typ, name, sqlText string
		if err := rows.Scan(&typ, &name, &sqlText); err != nil {
			t.Fatalf("scan SQLite schema: %v", err)
		}
		parts = append(parts, typ+"|"+name+"|"+sqlText)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("SQLite schema rows: %v", err)
	}
	return strings.Join(parts, "\n")
}

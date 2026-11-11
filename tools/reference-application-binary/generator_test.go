package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hat47x/tei-platform/tools/reference-application-binary/internal/appgen"
	_ "modernc.org/sqlite"
)

func TestExtractOrderEntryModel(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}
	if ctx == nil {
		t.Fatal("ctx is nil")
	}
	if ctx.ApplicationID != "order_entry" {
		t.Fatalf("application id: %s", ctx.ApplicationID)
	}
	if ctx.ModelRevision == "" {
		t.Fatal("model revision is empty")
	}
	if ctx.SourceRevision == "" {
		t.Fatal("source revision is empty")
	}
	if len(ctx.Queries) == 0 {
		t.Fatal("expected queries")
	}
	if len(ctx.Commands) == 0 {
		t.Fatal("expected commands")
	}
}

func TestGenerateWritesSources(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	out := t.TempDir()
	files, err := appgen.Generate(ctx, out)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("expected generated files")
	}
	for _, p := range files {
		if strings.Contains(p, "..") {
			t.Fatalf("path contains ..: %s", p)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read generated file: %v", err)
		}
		if len(b) == 0 {
			t.Fatalf("empty generated file: %s", p)
		}
	}
}

func TestBuildCreatesRunnableBinary(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry")
	if filepath.Ext(binary) == "" && (os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows") {
		binary += ".exe"
	}

	result, err := appgen.Build(workspace, "order_entry", sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.HasPrefix(result.Digest, "sha256:") {
		t.Fatalf("unexpected digest: %s", result.Digest)
	}
	if result.BinarySize == 0 {
		t.Fatal("expected non-empty binary")
	}
	if result.BuildDuration <= 0 {
		t.Fatalf("unexpected build duration: %v", result.BuildDuration)
	}

	cmd := exec.Command(binary, "provenance")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run provenance: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), ctx.ApplicationID) {
		t.Fatalf("provenance missing application id: %s", out)
	}
}

func TestGeneratedBinaryUnknownCommand(t *testing.T) {
	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry")
	if filepath.Ext(binary) == "" && (os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows") {
		binary += ".exe"
	}

	_, err = appgen.Build(workspace, "order_entry", sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	cmd := exec.Command(binary, "unknown")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error for unknown command, got: %s", out)
	}
	if !strings.Contains(string(out), "unknown command") {
		t.Fatalf("unexpected error message: %s", out)
	}
}

func TestGeneratedBinaryExecutesQueryCommandValidate(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("execution test must run with native GOOS/GOARCH")
	}

	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	migrationsPath := filepath.Join(sourceDir, "migrations.go")
	migrationsSource, err := os.ReadFile(migrationsPath)
	if err == nil && strings.Contains(string(migrationsSource), "public.") {
		t.Skip("workspace migration SQL is not SQLite-compatible")
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry")
	if filepath.Ext(binary) == "" && runtime.GOOS == "windows" {
		binary += ".exe"
	}

	_, err = appgen.Build(workspace, "order_entry", sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY, customer_id TEXT, amount TEXT, currency TEXT, created_at TEXT)`); err != nil {
		t.Fatalf("create orders: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE customers (id TEXT PRIMARY KEY, name TEXT, email TEXT)`); err != nil {
		t.Fatalf("create customers: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO orders VALUES ('1','c1','100','USD','2026-11-10')`); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO customers VALUES ('c1','Alice','alice@example.com')`); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	env := os.Environ()
	env = append(env, fmt.Sprintf("TEI_DSN=%s", dbPath))

	cmd := exec.Command(binary, "query", "order_entry.recent_orders", "1")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run query: %v\n%s", err, out)
	}
	var queryResults []map[string]any
	if err := json.Unmarshal(out, &queryResults); err != nil {
		t.Fatalf("query output not valid json: %v\n%s", err, out)
	}
	if len(queryResults) == 0 {
		t.Fatalf("query returned no rows: %s", out)
	}
	if queryResults[0]["id"] == nil || queryResults[0]["amount"] == nil {
		t.Fatalf("query missing expected fields: %s", out)
	}

	cmd = exec.Command(binary, "query", "order_entry.customer_reference", "c1")
	cmd.Env = env
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run customer query: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Alice") {
		t.Fatalf("customer query missing Alice: %s", out)
	}

	cmd = exec.Command(binary, "command", "order_entry.submit_order")
	cmd.Env = env
	cmd.Stdin = strings.NewReader(`{"id":"2","customerID":"c1","amount":"200","currency":"USD"}`)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run command: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "committed") {
		t.Fatalf("command output missing committed: %s", out)
	}

	cmd = exec.Command(binary, "validate", "sales.order_line.quantity_positive")
	cmd.Env = env
	cmd.Stdin = strings.NewReader(`{"quantity":"2","lineNumber":1}`)
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run validate: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("validate output missing ok: %s", out)
	}
}

func TestGeneratedBinaryMigrationsApplied(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("migration test must run with native GOOS/GOARCH")
	}

	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	migrationsPath := filepath.Join(sourceDir, "migrations.go")
	migrationsSource, err := os.ReadFile(migrationsPath)
	if err != nil {
		t.Fatalf("read migrations.go: %v", err)
	}
	if !strings.Contains(string(migrationsSource), "applyMigrations") {
		t.Fatalf("migrations.go missing applyMigrations: %s", migrationsSource)
	}
	if !strings.Contains(string(migrationsSource), "20260826_001") {
		t.Fatalf("migrations.go missing expected migration id: %s", migrationsSource)
	}
	if strings.Contains(string(migrationsSource), "public.") {
		t.Skip("workspace migration SQL is not SQLite-compatible")
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry")
	if filepath.Ext(binary) == "" && runtime.GOOS == "windows" {
		binary += ".exe"
	}

	_, err = appgen.Build(workspace, "order_entry", sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	dbPath := filepath.Join(t.TempDir(), "app.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE orders (id TEXT PRIMARY KEY, customer_id TEXT, amount TEXT, currency TEXT, created_at TEXT)`); err != nil {
		t.Fatalf("create orders: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE customers (id TEXT PRIMARY KEY, name TEXT, email TEXT)`); err != nil {
		t.Fatalf("create customers: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO orders VALUES ('1','c1','100','USD','2026-11-10')`); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	env := os.Environ()
	env = append(env, fmt.Sprintf("TEI_DSN=%s", dbPath))

	cmd := exec.Command(binary, "query", "order_entry.recent_orders", "1")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run query after migration: %v\n%s", err, out)
	}
	var queryResults []map[string]any
	if err := json.Unmarshal(out, &queryResults); err != nil {
		t.Fatalf("query output not valid json: %v\n%s", err, out)
	}
	if len(queryResults) == 0 {
		t.Fatalf("query returned no rows: %s", out)
	}
}

func TestBuildPerformance(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("performance test must run with native GOOS/GOARCH")
	}

	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	genStart := time.Now()
	_, err = appgen.Generate(ctx, sourceDir)
	genDuration := time.Since(genStart)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry")
	if filepath.Ext(binary) == "" && runtime.GOOS == "windows" {
		binary += ".exe"
	}

	result, err := appgen.Build(workspace, "order_entry", sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	t.Logf("Generation: %v, Build: %v, Binary size: %d bytes", genDuration, result.BuildDuration, result.BinarySize)

	// Performance bounds (not strict, but flag regressions).
	// These are generous bounds for a cold build on a typical dev machine.
	if genDuration > 5*time.Second {
		t.Fatalf("generation too slow: %v", genDuration)
	}
	if result.BuildDuration > 5*time.Minute {
		t.Fatalf("build too slow: %v", result.BuildDuration)
	}
	if result.BinarySize > 50*1024*1024 {
		t.Fatalf("binary too large: %d bytes", result.BinarySize)
	}
}

func TestCrossCompileLinuxAmd64(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("cross-compilation test must run with native GOOS/GOARCH")
	}

	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry-linux-amd64")
	result, err := appgen.Build(workspace, "order_entry", sourceDir, binary, "linux/amd64")
	if err != nil {
		t.Fatalf("Build linux/amd64: %v", err)
	}
	if !strings.HasPrefix(result.Digest, "sha256:") {
		t.Fatalf("unexpected digest: %s", result.Digest)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("linux/amd64 binary missing: %v", err)
	}
}

func TestCrossCompileDarwinAmd64(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("cross-compilation test must run with native GOOS/GOARCH")
	}

	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry-darwin-amd64")
	result, err := appgen.Build(workspace, "order_entry", sourceDir, binary, "darwin/amd64")
	if err != nil {
		t.Fatalf("Build darwin/amd64: %v", err)
	}
	if !strings.HasPrefix(result.Digest, "sha256:") {
		t.Fatalf("unexpected digest: %s", result.Digest)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("darwin/amd64 binary missing: %v", err)
	}
}

func TestCrossCompileDarwinArm64(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("cross-compilation test must run with native GOOS/GOARCH")
	}

	repo := repoRoot(t)
	workspace := filepath.Join(repo, "reference", "workspace")
	ctx, err := appgen.ExtractModel(workspace, "order_entry")
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}

	sourceDir := t.TempDir()
	_, err = appgen.Generate(ctx, sourceDir)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	binary := filepath.Join(t.TempDir(), "tei-app-order-entry-darwin-arm64")
	result, err := appgen.Build(workspace, "order_entry", sourceDir, binary, "darwin/arm64")
	if err != nil {
		t.Fatalf("Build darwin/arm64: %v", err)
	}
	if !strings.HasPrefix(result.Digest, "sha256:") {
		t.Fatalf("unexpected digest: %s", result.Digest)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatalf("darwin/arm64 binary missing: %v", err)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			parent := filepath.Dir(wd)
			if parent != wd && filepath.Base(parent) == "tools" {
				wd = filepath.Dir(parent)
			}
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatal("repo root not found")
		}
		wd = parent
	}
}

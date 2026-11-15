package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/tools/reference-application-binary/internal/appgen"
	"tei/tools/reference-application-binary/internal/teiparse"
)

const (
	p0aMigrationID       = "20260826_001"
	p0aMigrationDatabase = "sales_core"
	p0aMigrationRecovery = "forward-fix"
	p0aSchemaSQL         = "CREATE INDEX orders_customer_idx\nON public.orders (customer_id);"
	p0aDataSQL           = "UPDATE public.orders\nSET status = 'UNKNOWN'\nWHERE status IS NULL;"
	p0aVerifySQL         = "SELECT COUNT(*)\nFROM public.orders\nWHERE status IS NULL;"
	p0aApplicationID     = "order_entry"
	p0aExpectedFiles     = "main.go,config.go,models.go,queries.go,commands.go,validations.go,errors.go,provenance.go,migrations.go"
)

func p0aRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	abs, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(abs, "reference", "workspace")); err != nil {
		t.Fatalf("reference/workspace not found under %s: %v", abs, err)
	}
	return abs
}

func TestP0AStageParse(t *testing.T) {
	repo := p0aRepoRoot(t)
	world, err := teiparse.ParseWorld(filepath.Join(repo, "reference", "workspace"))
	if err != nil {
		t.Fatalf("ParseWorld: %v", err)
	}
	var found *teiparse.Migration
	for _, m := range world.Migrations {
		if m.ID == p0aMigrationID {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("migration %q not found in %d migrations", p0aMigrationID, len(world.Migrations))
	}
	if found.Database != p0aMigrationDatabase {
		t.Errorf("database: got %q, expected %q", found.Database, p0aMigrationDatabase)
	}
	if found.Recovery != p0aMigrationRecovery {
		t.Errorf("recovery: got %q, expected %q", found.Recovery, p0aMigrationRecovery)
	}
	if found.SchemaSQL != p0aSchemaSQL {
		t.Errorf("schema SQL mismatch:\n got: %q\nwant: %q", found.SchemaSQL, p0aSchemaSQL)
	}
	if found.DataSQL != p0aDataSQL {
		t.Errorf("data SQL mismatch:\n got: %q\nwant: %q", found.DataSQL, p0aDataSQL)
	}
	if found.VerifySQL != p0aVerifySQL {
		t.Errorf("verify SQL mismatch:\n got: %q\nwant: %q", found.VerifySQL, p0aVerifySQL)
	}
}

func TestP0AStageModel(t *testing.T) {
	repo := p0aRepoRoot(t)
	ctx, err := appgen.ExtractModel(filepath.Join(repo, "reference", "workspace"), p0aApplicationID)
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}
	if ctx.ApplicationID != p0aApplicationID {
		t.Fatalf("application id: got %q, expected %q", ctx.ApplicationID, p0aApplicationID)
	}
	var found *teiparse.Migration
	for _, m := range ctx.Migrations {
		if m.ID == p0aMigrationID {
			found = m
			break
		}
	}
	if found == nil {
		t.Fatalf("migration %q not carried into GenerationContext (%d migrations)", p0aMigrationID, len(ctx.Migrations))
	}
	if found.SchemaSQL != p0aSchemaSQL {
		t.Errorf("schema SQL mismatch in context:\n got: %q\nwant: %q", found.SchemaSQL, p0aSchemaSQL)
	}
}

func TestP0AStageGenerate(t *testing.T) {
	repo := p0aRepoRoot(t)
	ctx, err := appgen.ExtractModel(filepath.Join(repo, "reference", "workspace"), p0aApplicationID)
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}
	out := t.TempDir()
	files, err := appgen.Generate(ctx, out)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	gotGo := map[string]bool{}
	for _, p := range files {
		base := filepath.Base(p)
		if base == "manifest.json" {
			continue
		}
		gotGo[base] = true
	}
	for _, want := range strings.Split(p0aExpectedFiles, ",") {
		if !gotGo[want] {
			t.Errorf("expected generated file %q missing from %v", want, files)
		}
	}
	b, err := os.ReadFile(filepath.Join(out, "migrations.go"))
	if err != nil {
		t.Fatalf("read migrations.go: %v", err)
	}
	src := string(b)
	if !strings.Contains(src, p0aMigrationID) {
		t.Errorf("migrations.go missing migration id %q", p0aMigrationID)
	}
	if !strings.Contains(src, "func applyMigrations(") {
		t.Errorf("migrations.go missing applyMigrations entrypoint")
	}
	if !strings.Contains(src, "applyMigration"+p0aMigrationID+"(") {
		t.Errorf("migrations.go missing call to applyMigration%s", p0aMigrationID)
	}
	if !strings.Contains(src, "CREATE INDEX orders_customer_idx") {
		t.Errorf("migrations.go missing schema SQL first line")
	}
	if !strings.Contains(src, "ON public.orders (customer_id);") {
		t.Errorf("migrations.go missing schema SQL second line")
	}
	if !strings.Contains(src, "DO NOT EDIT") {
		t.Errorf("migrations.go missing generated-code provenance header")
	}
}

func TestP0AStageBuild(t *testing.T) {
	if os.Getenv("GOOS") != "" || os.Getenv("GOARCH") != "" {
		t.Skip("P0-A build stage must run with native GOOS/GOARCH")
	}
	repo := p0aRepoRoot(t)
	ctx, err := appgen.ExtractModel(filepath.Join(repo, "reference", "workspace"), p0aApplicationID)
	if err != nil {
		t.Fatalf("ExtractModel: %v", err)
	}
	sourceDir := t.TempDir()
	if _, err := appgen.Generate(ctx, sourceDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	binary := filepath.Join(t.TempDir(), "tei-app-"+p0aApplicationID)
	if filepath.Ext(binary) == "" && (os.Getenv("GOOS") == "windows" || os.PathSeparator == '\\') {
		binary += ".exe"
	}
	result, err := appgen.Build(filepath.Join(repo, "reference", "workspace"), p0aApplicationID, sourceDir, binary, "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if result.BinarySize == 0 {
		t.Fatal("expected non-empty binary")
	}
	cmd := exec.Command(binary, "provenance")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run provenance: %v\n%s", err, out)
	}
	var prov map[string]any
	if err := json.Unmarshal(out, &prov); err != nil {
		t.Fatalf("provenance not valid json: %v\n%s", err, out)
	}
	if prov["applicationId"] != p0aApplicationID {
		t.Errorf("provenance applicationId: got %v, expected %q", prov["applicationId"], p0aApplicationID)
	}
	if gen, _ := prov["generator"].(string); gen != "tei.support.application-binary" {
		t.Errorf("provenance generator: got %q, expected %q", gen, "tei.support.application-binary")
	}
}

package workspace_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"tei/internal/ir"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestReferenceWorkspaceNormalizesToGoldenWorld(t *testing.T) {
	root := repoRoot(t)
	got, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatalf("load workspace: %v", err)
	}

	f, err := os.Open(filepath.Join(root, "reference", "world.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var want ir.World
	if err := json.NewDecoder(f).Decode(&want); err != nil {
		t.Fatal(err)
	}
	workspace.Normalize(&want)

	if !reflect.DeepEqual(got.World, want) {
		gj, _ := json.MarshalIndent(got.World, "", "  ")
		wj, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("normalized world mismatch\nGOT:\n%s\nWANT:\n%s", gj, wj)
	}
	if ds := validation.Validate(got.World); validation.HasErrors(ds) {
		t.Fatalf("reference workspace has validation errors: %#v", ds)
	}
}

func TestSourceMapTracksUnitFiles(t *testing.T) {
	root := repoRoot(t)
	got, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"plugin:tei.migration.atlas@1.0.0":          "plugins/tei-migration-atlas/1.0.0/plugin.tei",
		"plugin:tei.database.postgres@1.0.0":        "plugins/tei-database-postgres/1.0.0/plugin.tei",
		"bundle:tei.reference.providers@1.0.0":      "bundles/tei-reference-providers/1.0.0/bundle.tei",
		"semantic-data:sales.order":                 "semantic/sales/order.tei",
		"semantic-attribute:sales.order.status":     "semantic/sales/order.tei",
		"semantic-data:customer.customer":           "semantic/customer/customer.tei",
		"application:order_entry":                   "applications/order-entry/application.tei",
		"database:sales_core":                       "databases/sales-core/database.tei",
		"database-contract:sales_core.order":        "databases/sales-core/contracts/order.tei",
		"persistence:sales_core.order":              "databases/sales-core/persistence/order.tei",
		"migration:sales_core/20260826_001":         "databases/sales-core/migrations/20260826_001/migration.tei",
		"migration-provider:sales_core":             "databases/sales-core/migration-provider.tei",
		"database-release:sales_core/4.0.0":         "releases/databases/sales-core/4.0.0.tei",
		"database-release:sales_core/4.1.0":         "releases/databases/sales-core/4.1.0.tei",
		"database-release:sales_core/4.2.0":         "releases/databases/sales-core/4.2.0.tei",
		"application-release:order_entry/3.7.0":     "releases/applications/order-entry/3.7.0.tei",
		"application-release:order_entry/3.8.0":     "releases/applications/order-entry/3.8.0.tei",
		"solution-release:order_rollout/2026.8.27":  "releases/solutions/order-rollout/2026.8.27.tei",
		"query:order_entry.recent_orders":           "applications/order-entry/queries/recent_orders.tei",
		"binding:order_entry.compose_order_summary": "applications/order-entry/realizations/default.tei",
		"environment:development":                   "environments/development/environment.tei",
		"environment:production":                    "environments/production/environment.tei",
	}
	for key, file := range cases {
		loc, ok := got.Sources[key]
		if !ok {
			t.Errorf("missing source map entry %s", key)
			continue
		}
		if loc.File != file {
			t.Errorf("%s file: got %q want %q", key, loc.File, file)
		}
	}
}

func TestReferenceMigrationPreservesDDLAndDML(t *testing.T) {
	root := repoRoot(t)
	got, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.World.Migrations) != 1 {
		t.Fatalf("expected one migration, got %d", len(got.World.Migrations))
	}
	m := got.World.Migrations[0]
	if len(m.Steps) != 3 {
		t.Fatalf("expected schema/data/verify steps, got %#v", m.Steps)
	}
	if string(m.Steps[0].Kind) != "schema" || string(m.Steps[1].Kind) != "data" || string(m.Steps[2].Kind) != "verify" {
		t.Fatalf("unexpected step kinds: %#v", m.Steps)
	}
	if m.Steps[1].SQL == nil || m.Steps[1].SQL.Source == "" {
		t.Fatal("DML SQL was not preserved")
	}
}

func TestReferenceDatabaseReleaseIsIndependentUnit(t *testing.T) {
	root := repoRoot(t)
	got, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.World.DatabaseReleases) != 3 {
		t.Fatalf("releases=%#v", got.World.DatabaseReleases)
	}
	versions := map[string]bool{}
	for _, r := range got.World.DatabaseReleases {
		if r.Database != "sales_core" {
			t.Fatalf("unexpected database release=%#v", r)
		}
		versions[r.Version] = true
	}
	for _, v := range []string{"4.0.0", "4.1.0", "4.2.0"} {
		if !versions[v] {
			t.Fatalf("missing database release %s", v)
		}
	}
}

func TestWorkspaceManifestAndManagedFilesAreExtensionIndependent(t *testing.T) {
	root := repoRoot(t)
	ws := filepath.Join(root, "reference", "workspace")
	if _, err := os.Stat(filepath.Join(ws, "tei.workspace.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws, "tei.workspace.tei")); !os.IsNotExist(err) {
		t.Fatalf("legacy workspace manifest should not be canonical in reference workspace")
	}
	got, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	want := "databases/sales-core/README.md"
	for _, f := range got.ManagedFiles {
		if f.Path == want && f.Extension == ".md" && f.SourceRoot == "databases" {
			return
		}
	}
	t.Fatalf("native managed file %s not enumerated: %+v", want, got.ManagedFiles)
}

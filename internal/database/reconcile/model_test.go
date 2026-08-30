package reconcile_test

import (
	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/reconcile"
	"tei/internal/database/schemadiff"
	"testing"
)

func TestBuildKeepsHistoryAndSchemaAxesSeparate(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	h := migration.HistoryComparison{Database: db, Entries: []migration.HistoryComparisonEntry{{Database: db, ID: "001", State: migration.RepoApplied}}}
	d := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.AddColumn, Table: "public.orders"}}}
	got := reconcile.Build(db, h, d)
	if len(got.Findings) != 1 || got.Findings[0].Kind != reconcile.FindingSchemaDrift {
		t.Fatalf("got=%#v", got)
	}
}

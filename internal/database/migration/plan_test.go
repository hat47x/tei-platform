package migration_test

import (
	"strings"
	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
	"testing"
)

func TestPlanKeepsDatabaseLifecycleIndependentAndRendersPostgres(t *testing.T) {
	c := physical.Column{Name: "external_id", PhysicalType: "pg_catalog.varchar", Nullable: true}
	d := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.AddColumn, Table: "public.orders", Column: &c}}}
	m, err := migration.Plan("20260827_001", core.MustCanonicalID("sales_core"), "fp-1", d, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if m.Database != core.MustCanonicalID("sales_core") {
		t.Fatal("wrong database")
	}
	if len(m.Realizations) != 1 || len(m.Realizations[0].Statements) != 1 {
		t.Fatalf("unexpected realization: %#v", m.Realizations)
	}
	if !strings.Contains(m.Realizations[0].Statements[0], `ADD COLUMN "external_id"`) {
		t.Fatalf("unexpected sql: %s", m.Realizations[0].Statements[0])
	}
	if m.Steps[len(m.Steps)-1].Kind != migration.StepVerify {
		t.Fatal("migration must end with verification")
	}
}

// MA-08 characterizes the current planning boundary, not a qualified
// apply precondition. The caller can submit identical claimed fingerprints
// for different schema payloads; Plan forwards the string without proof of
// origin, calculation basis, or freshness.
func TestMA08SameClaimedFingerprintCanDescribeDifferentActualSchema(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	old := physical.SchemaSnapshot{
		Database:    db,
		Fingerprint: "claimed-fp1",
		Tables: []physical.Table{{
			Schema: "public",
			Name:   "orders",
			Columns: []physical.Column{{
				Name: "id", PhysicalType: "pg_catalog.int4",
			}},
		}},
	}
	newer := physical.SchemaSnapshot{
		Database:    db,
		Fingerprint: old.Fingerprint,
		Tables: []physical.Table{{
			Schema: "public",
			Name:   "orders",
			Columns: []physical.Column{
				{Name: "id", PhysicalType: "pg_catalog.int4"},
				{Name: "audited", PhysicalType: "pg_catalog.bool", Nullable: true},
			},
		}},
	}
	oldDiff, err := schemadiff.Compare(newer.Normalize(), old.Normalize())
	if err != nil {
		t.Fatal(err)
	}
	newDiff, err := schemadiff.Compare(newer.Normalize(), newer.Normalize())
	if err != nil {
		t.Fatal(err)
	}
	if len(oldDiff.Changes) == 0 || len(newDiff.Changes) != 0 {
		t.Fatalf("fixture must produce different structural differences: old=%#v new=%#v", oldDiff, newDiff)
	}
	oldPlan, err := migration.Plan("m001", db, old.Fingerprint, oldDiff, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	newPlan, err := migration.Plan("m001", db, newer.Fingerprint, newDiff, environment.ProviderPostgres)
	if err != nil {
		t.Fatal(err)
	}
	if oldPlan.BasedOnActualFingerprint != newPlan.BasedOnActualFingerprint ||
		oldPlan.BasedOnActualFingerprint != "claimed-fp1" {
		t.Fatalf("same unverified claim unexpectedly changed: old=%q new=%q", oldPlan.BasedOnActualFingerprint, newPlan.BasedOnActualFingerprint)
	}
	if len(oldPlan.Steps) == len(newPlan.Steps) {
		t.Fatalf("distinct snapshots unexpectedly generated the same step count: old=%#v new=%#v", oldPlan.Steps, newPlan.Steps)
	}
	// Independent Apply Admission Expected: unverified, even though both Plan
	// calls returned successfully with the same claimed fingerprint.
}

// MA-09 characterizes the false-absence boundary: a partial table listing
// is not proof that an omitted table is absent from the live database.
func TestMA09PartialActualTableListIsRenderedAsAbsentTable(t *testing.T) {
	db := core.MustCanonicalID("sales_core")
	orders := physical.Table{Schema: "public", Name: "orders"}
	audit := physical.Table{Schema: "public", Name: "audit_log"}
	desired := physical.SchemaSnapshot{Database: db, Tables: []physical.Table{orders, audit}}
	partial := physical.SchemaSnapshot{Database: db, Tables: []physical.Table{orders}}
	diff, err := schemadiff.Compare(desired.Normalize(), partial.Normalize())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, change := range diff.Changes {
		if change.Kind == schemadiff.CreateTable && change.Table == "public.audit_log" {
			found = true
		}
	}
	if !found {
		t.Fatalf("fixture expected a create-table candidate for the omitted relation: %#v", diff.Changes)
	}
	// Independent Apply Admission Expected: unknown until a complete reader
	// establishes actual absence in the intended target and scope.
}

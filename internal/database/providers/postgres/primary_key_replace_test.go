package postgres_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

// #1803 P5: replacing the primary key of an existing table uses the constraint
// name observed in the catalog, never "<table>_pkey".

func replaceKeyDiff() schemadiff.Diff {
	return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.ChangePrimaryKey, Table: "public.customers", PrimaryKey: []string{"id", "region"}, BeforePrimaryKey: []string{"id"}},
	}}
}

func replaceKeySnapshot() physical.SchemaSnapshot {
	return physical.SchemaSnapshot{Database: "sales_core", Tables: []physical.Table{
		{Schema: "public", Name: "customers", Columns: []physical.Column{{Name: "id", PhysicalType: "text"}, {Name: "region", PhysicalType: "text"}}, PrimaryKey: []string{"id"}},
	}}
}

func cleanReplaceCheck() postgres.ConstraintDataCheck {
	return postgres.ConstraintDataCheck{
		Kind: postgres.ConstraintDataPrimaryKeyReplace, Table: "public.customers",
		Columns: []string{"id", "region"}, BeforeColumns: []string{"id"},
		PrimaryKeyName: "customers_legacy_pk", Observed: true,
	}
}

func TestRenderReplacesPrimaryKeyWithObservedConstraintName(t *testing.T) {
	observation := postgres.SchemaObservation{Snapshot: replaceKeySnapshot(), ConstraintData: []postgres.ConstraintDataCheck{cleanReplaceCheck()}}
	statements, err := postgres.RenderMigrationSQLWithObservation(replaceKeyDiff(), observation)
	if err != nil {
		t.Fatal(err)
	}
	want := `ALTER TABLE "public"."customers" DROP CONSTRAINT "customers_legacy_pk", ADD PRIMARY KEY ("id", "region");`
	if len(statements) != 1 || statements[0] != want {
		t.Fatalf("statements=%#v", statements)
	}
	if strings.Contains(statements[0], "customers_pkey") {
		t.Fatal("the constraint name was assumed")
	}
}

func TestRenderPrimaryKeyReplacementWithoutObservationIsUnresolved(t *testing.T) {
	statements, err := postgres.RenderMigrationSQLWithObservation(replaceKeyDiff(), postgres.SchemaObservation{Snapshot: replaceKeySnapshot()})
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil || blocked.Readiness.Additions[0].Reason != postgres.ConstraintDataUnobserved {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if _, err := postgres.RenderMigrationSQL(replaceKeyDiff()); err == nil || !strings.Contains(err.Error(), "without an existing-data observation") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderPrimaryKeyReplacementRefusals(t *testing.T) {
	cases := map[string]struct {
		mutate func(*postgres.ConstraintDataCheck)
		reason postgres.ConstraintDataReadinessReason
		detail string
	}{
		"constraint name not observed": {func(c *postgres.ConstraintDataCheck) { c.PrimaryKeyName = "" }, postgres.ConstraintDataUnobserved, "never assumed"},
		"not observed":                 {func(c *postgres.ConstraintDataCheck) { c.Observed, c.Reason = false, "no privilege" }, postgres.ConstraintDataUnobserved, "no privilege"},
		"duplicates":                   {func(c *postgres.ConstraintDataCheck) { c.DuplicateKeys, c.DuplicateRows = 2, 5 }, postgres.ConstraintDataViolations, "5 row(s) share 2 duplicated key value(s)"},
		"null key columns":             {func(c *postgres.ConstraintDataCheck) { c.NullKeyRows = 3 }, postgres.ConstraintDataViolations, "3 row(s) have NULL in a key column"},
		"referencing foreign key": {func(c *postgres.ConstraintDataCheck) {
			c.ReferencingForeignKeys = []postgres.ReferencingForeignKey{{Constraint: "orders_customer_fk", Table: "public.orders", Columns: []string{"customer_id"}}}
		}, postgres.ConstraintDataReferencedByForeignKeys, `foreign key "orders_customer_fk" on public.orders (customer_id)`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			check := cleanReplaceCheck()
			tc.mutate(&check)
			observation := postgres.SchemaObservation{Snapshot: replaceKeySnapshot(), ConstraintData: []postgres.ConstraintDataCheck{check}}
			statements, err := postgres.RenderMigrationSQLWithObservation(replaceKeyDiff(), observation)
			var blocked *postgres.ConstraintDataReadinessError
			if !errors.As(err, &blocked) || statements != nil {
				t.Fatalf("statements=%#v err=%v", statements, err)
			}
			if got := blocked.Readiness.Additions[0]; got.Reason != tc.reason || !strings.Contains(got.Detail, tc.detail) {
				t.Fatalf("addition=%#v", got)
			}
		})
	}
}

func TestRenderPrimaryKeyReplacementRefusesNullableNewColumnAndDroppedColumn(t *testing.T) {
	snapshot := replaceKeySnapshot()
	snapshot.Tables[0].Columns[1].Nullable = true
	observation := postgres.SchemaObservation{Snapshot: snapshot, ConstraintData: []postgres.ConstraintDataCheck{cleanReplaceCheck()}}
	if _, err := postgres.RenderMigrationSQLWithObservation(replaceKeyDiff(), observation); err == nil || !strings.Contains(err.Error(), string(postgres.ConstraintDataNullablePrimaryKey)) {
		t.Fatalf("err=%v", err)
	}

	diff := replaceKeyDiff()
	dropped := physical.Column{Name: "id", PhysicalType: "text"}
	diff.Changes = append([]schemadiff.Change{{Kind: schemadiff.DropColumn, Table: "public.customers", BeforeColumn: &dropped}}, diff.Changes...)
	observation = postgres.SchemaObservation{Snapshot: replaceKeySnapshot(), ConstraintData: []postgres.ConstraintDataCheck{cleanReplaceCheck()}}
	if statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation); err == nil || !strings.Contains(err.Error(), "is dropped by the same diff") || statements != nil {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
}

func TestRenderPrimaryKeyReplacementObservationMustMatchTheDiffsCurrentKey(t *testing.T) {
	check := cleanReplaceCheck()
	check.BeforeColumns = []string{"other"}
	observation := postgres.SchemaObservation{Snapshot: replaceKeySnapshot(), ConstraintData: []postgres.ConstraintDataCheck{check}}
	_, err := postgres.RenderMigrationSQLWithObservation(replaceKeyDiff(), observation)
	if err == nil || !strings.Contains(err.Error(), string(postgres.ConstraintDataUnobserved)) {
		t.Fatalf("err=%v", err)
	}
}

type identityCatalog struct {
	countingCatalog
	identity postgres.PrimaryKeyIdentity
	err      error
}

func (c *identityCatalog) PrimaryKeyIdentity(context.Context, postgres.RelationIdentity) (postgres.PrimaryKeyIdentity, error) {
	return c.identity, c.err
}

func TestObserveConstraintDataReadsTheRealPrimaryKeyName(t *testing.T) {
	catalog := &identityCatalog{identity: postgres.PrimaryKeyIdentity{
		Name: "customers_legacy_pk", Columns: []string{"id"},
		ReferencingForeignKeys: []postgres.ReferencingForeignKey{{Constraint: "fk1", Table: "public.orders", Columns: []string{"cid"}}},
	}}
	observation, err := postgres.Provider{Catalog: catalog}.ObserveConstraintData(context.Background(), postgres.SchemaObservation{Snapshot: replaceKeySnapshot()}, replaceKeyDiff())
	if err != nil {
		t.Fatal(err)
	}
	check := observation.ConstraintData[0]
	if !check.Observed || check.PrimaryKeyName != "customers_legacy_pk" || len(check.ReferencingForeignKeys) != 1 || check.DuplicateRows != 2 || strings.Join(catalog.keyCalls, ";") != "customers:id,region" {
		t.Fatalf("check=%#v calls=%v", check, catalog.keyCalls)
	}
}

func TestObserveConstraintDataLeavesPrimaryKeyReplacementUnobserved(t *testing.T) {
	for name, catalog := range map[string]postgres.CatalogReader{
		"reader cannot read constraint": &countingCatalog{},
		"reading fails":                 &identityCatalog{err: errors.New("permission denied")},
		"observed key differs":          &identityCatalog{identity: postgres.PrimaryKeyIdentity{Name: "x", Columns: []string{"id", "region"}}},
		"no constraint name":            &identityCatalog{identity: postgres.PrimaryKeyIdentity{Columns: []string{"id"}}},
	} {
		t.Run(name, func(t *testing.T) {
			observation, err := postgres.Provider{Catalog: catalog}.ObserveConstraintData(context.Background(), postgres.SchemaObservation{Snapshot: replaceKeySnapshot()}, replaceKeyDiff())
			if err != nil {
				t.Fatal(err)
			}
			check := observation.ConstraintData[0]
			if check.Observed || check.Reason == "" || check.PrimaryKeyName != "" {
				t.Fatalf("check=%#v", check)
			}
			if _, err := postgres.RenderMigrationSQLWithObservation(replaceKeyDiff(), observation); err == nil || !strings.Contains(err.Error(), string(postgres.ConstraintDataUnobserved)) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

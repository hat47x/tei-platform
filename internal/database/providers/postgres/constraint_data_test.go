package postgres_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

// existingTablesSnapshot is the observed schema: public.orders without a
// primary key, public.customers with one.
func existingTablesSnapshot() physical.SchemaSnapshot {
	return physical.SchemaSnapshot{Database: "sales_core", Tables: []physical.Table{
		{Schema: "public", Name: "customers", Columns: []physical.Column{{Name: "id", PhysicalType: "text"}}, PrimaryKey: []string{"id"}},
		{Schema: "public", Name: "orders", Columns: []physical.Column{
			{Name: "id", PhysicalType: "text"}, {Name: "code", PhysicalType: "text", Nullable: true},
			{Name: "customer_id", PhysicalType: "text", Nullable: true},
		}},
	}}
}

func existingTableAdditionsDiff() schemadiff.Diff {
	return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.ChangePrimaryKey, Table: "public.orders", PrimaryKey: []string{"id"}},
		{Kind: schemadiff.AddUniqueConstraint, Table: "public.orders", UniqueConstraint: &physical.UniqueConstraint{Name: "orders_code_key", Columns: []string{"code"}}},
		{Kind: schemadiff.AddForeignKey, Table: "public.orders", ForeignKey: &physical.ForeignKey{
			Name: "orders_customer_id_fkey", Columns: []string{"customer_id"}, ReferencedSchema: "public", ReferencedTable: "customers", ReferencedColumns: []string{"id"}}},
	}}
}

func cleanChecks() []postgres.ConstraintDataCheck {
	return []postgres.ConstraintDataCheck{
		{Kind: postgres.ConstraintDataPrimaryKey, Table: "public.orders", Columns: []string{"id"}, Observed: true},
		{Kind: postgres.ConstraintDataUnique, Table: "public.orders", Columns: []string{"code"}, Observed: true},
		{Kind: postgres.ConstraintDataForeignKey, Table: "public.orders", Columns: []string{"customer_id"}, ReferencedTable: "public.customers", ReferencedColumns: []string{"id"}, Observed: true},
	}
}

func TestRenderAddsConstraintsToExistingTablesWhenObservedDataIsClean(t *testing.T) {
	observation := postgres.SchemaObservation{Snapshot: existingTablesSnapshot(), ConstraintData: cleanChecks()}
	statements, err := postgres.RenderMigrationSQLWithObservation(existingTableAdditionsDiff(), observation)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		`ALTER TABLE "public"."orders" ADD PRIMARY KEY ("id");`,
		`ALTER TABLE "public"."orders" ADD CONSTRAINT "orders_code_key" UNIQUE ("code");`,
		`ALTER TABLE "public"."orders" ADD CONSTRAINT "orders_customer_id_fkey" FOREIGN KEY ("customer_id") REFERENCES "public"."customers" ("id");`,
	}
	if !reflect.DeepEqual(statements, want) {
		t.Fatalf("statements=%#v", statements)
	}
}

// EX-02 / DE-10: without an observation the additions are unresolved, never
// treated as having zero violating rows.
func TestRenderRefusesExistingTableConstraintsWithoutObservation(t *testing.T) {
	for name, observation := range map[string]postgres.SchemaObservation{
		"no constraint data": {Snapshot: existingTablesSnapshot()},
		"not observed": {Snapshot: existingTablesSnapshot(), ConstraintData: func() []postgres.ConstraintDataCheck {
			checks := cleanChecks()
			for i := range checks {
				checks[i].Observed = false
				checks[i].Reason = "counting existing rows failed: permission denied"
			}
			return checks
		}()},
	} {
		t.Run(name, func(t *testing.T) {
			statements, err := postgres.RenderMigrationSQLWithObservation(existingTableAdditionsDiff(), observation)
			var blocked *postgres.ConstraintDataReadinessError
			if !errors.As(err, &blocked) || statements != nil {
				t.Fatalf("statements=%#v err=%v", statements, err)
			}
			if len(blocked.Readiness.Additions) != 3 {
				t.Fatalf("additions=%#v", blocked.Readiness.Additions)
			}
			for _, addition := range blocked.Readiness.Additions {
				if addition.Executable || addition.Reason != postgres.ConstraintDataUnobserved {
					t.Fatalf("addition=%#v", addition)
				}
			}
			if !strings.Contains(err.Error(), "not treated as zero") {
				t.Fatalf("err=%v", err)
			}
			if name == "not observed" && !strings.Contains(err.Error(), "permission denied") {
				t.Fatalf("reason is not carried: %v", err)
			}
		})
	}
}

// EX-01: rows that violate the constraint stop rendering with the counts.
func TestRenderRefusesExistingTableConstraintsWithViolationCounts(t *testing.T) {
	checks := cleanChecks()
	checks[0].DuplicateKeys, checks[0].DuplicateRows, checks[0].NullKeyRows = 1, 2, 3
	checks[1].DuplicateKeys, checks[1].DuplicateRows = 2, 5
	checks[2].MissingReferenceRows = 4
	observation := postgres.SchemaObservation{Snapshot: existingTablesSnapshot(), ConstraintData: checks}
	statements, err := postgres.RenderMigrationSQLWithObservation(existingTableAdditionsDiff(), observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	for _, want := range []string{
		"primary-key public.orders (id): existing-data-violations: 2 row(s) share 1 duplicated key value(s), 3 row(s) have NULL in a key column",
		"unique-constraint public.orders (code): existing-data-violations: 5 row(s) share 2 duplicated key value(s)",
		"foreign-key public.orders (customer_id): existing-data-violations: 4 row(s) have no referenced row",
		"no declared handling exists",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in %v", want, err)
		}
	}
}

func TestRenderRefusesWhenOnlySomeAdditionsAreClean(t *testing.T) {
	checks := cleanChecks()
	checks[2].MissingReferenceRows = 1
	observation := postgres.SchemaObservation{Snapshot: existingTablesSnapshot(), ConstraintData: checks}
	readiness, err := observation.AssessConstraintDataReadiness(existingTableAdditionsDiff())
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Executable || !readiness.Additions[0].Executable || !readiness.Additions[1].Executable || readiness.Additions[2].Executable {
		t.Fatalf("readiness=%#v", readiness)
	}
	if statements, err := postgres.RenderMigrationSQLWithObservation(existingTableAdditionsDiff(), observation); err == nil || statements != nil {
		t.Fatalf("partial diff rendered: %#v", statements)
	}
}

func TestRenderRefusesPrimaryKeyOnColumnThatStaysNullable(t *testing.T) {
	snapshot := existingTablesSnapshot()
	snapshot.Tables[1].Columns[0].Nullable = true
	observation := postgres.SchemaObservation{Snapshot: snapshot, ConstraintData: cleanChecks()}
	_, err := postgres.RenderMigrationSQLWithObservation(existingTableAdditionsDiff(), observation)
	if err == nil || !strings.Contains(err.Error(), string(postgres.ConstraintDataNullablePrimaryKey)) {
		t.Fatalf("err=%v", err)
	}
	// The same column made NOT NULL by the diff itself is accepted.
	diff := existingTableAdditionsDiff()
	before, after := snapshot.Tables[1].Columns[0], snapshot.Tables[1].Columns[0]
	after.Nullable = false
	diff.Changes = append([]schemadiff.Change{{Kind: schemadiff.AlterColumn, Table: "public.orders", Column: &after, BeforeColumn: &before}}, diff.Changes...)
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	if statements[1] != `ALTER TABLE "public"."orders" ALTER COLUMN "id" SET NOT NULL;` || statements[2] != `ALTER TABLE "public"."orders" ADD PRIMARY KEY ("id");` {
		t.Fatalf("statements=%#v", statements)
	}
}

func TestRenderRemovesPrimaryKeyWithObservedConstraintName(t *testing.T) {
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.ChangePrimaryKey, Table: "public.customers", BeforePrimaryKey: []string{"id"}},
	}}
	observation := postgres.SchemaObservation{Snapshot: existingTablesSnapshot(), ConstraintData: []postgres.ConstraintDataCheck{{
		Kind: postgres.ConstraintDataPrimaryKeyRemove, Table: "public.customers", BeforeColumns: []string{"id"},
		PrimaryKeyName: "customers_legacy_pk", Observed: true,
	}}}
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	if err != nil {
		t.Fatal(err)
	}
	want := `ALTER TABLE "public"."customers" DROP CONSTRAINT "customers_legacy_pk";`
	if len(statements) != 1 || statements[0] != want {
		t.Fatalf("statements=%#v", statements)
	}
	if strings.Contains(statements[0], "customers_pkey") {
		t.Fatal("the constraint name was assumed")
	}
}

func TestRenderRefusesRemovingPrimaryKeyWithoutObservation(t *testing.T) {
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.ChangePrimaryKey, Table: "public.customers", BeforePrimaryKey: []string{"id"}},
	}}
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, postgres.SchemaObservation{Snapshot: existingTablesSnapshot()})
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil || blocked.Readiness.Additions[0].Reason != postgres.ConstraintDataUnobserved {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
}

func TestRenderRefusesRemovingReferencedPrimaryKey(t *testing.T) {
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.ChangePrimaryKey, Table: "public.customers", BeforePrimaryKey: []string{"id"}},
	}}
	observation := postgres.SchemaObservation{Snapshot: existingTablesSnapshot(), ConstraintData: []postgres.ConstraintDataCheck{{
		Kind: postgres.ConstraintDataPrimaryKeyRemove, Table: "public.customers", BeforeColumns: []string{"id"},
		PrimaryKeyName: "customers_legacy_pk", Observed: true,
		ReferencingForeignKeys: []postgres.ReferencingForeignKey{
			{Constraint: "refs_customer_fk", Table: "public.refs", Columns: []string{"customer_id"}},
		},
	}}}
	statements, err := postgres.RenderMigrationSQLWithObservation(diff, observation)
	var blocked *postgres.ConstraintDataReadinessError
	if !errors.As(err, &blocked) || statements != nil {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if blocked.Readiness.Additions[0].Reason != postgres.ConstraintDataReferencedByForeignKeys ||
		!strings.Contains(err.Error(), `refs_customer_fk`) || !strings.Contains(err.Error(), "not removed") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMigrationSQLWithoutObservationRefusesExistingTablePrimaryKey(t *testing.T) {
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.ChangePrimaryKey, Table: "public.orders", PrimaryKey: []string{"id"}},
	}}
	statements, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "without an existing-data observation") || statements != nil {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
}

func TestConstraintDataObservationMustMatchDiff(t *testing.T) {
	checks := cleanChecks()
	checks[1].Columns = []string{"other"}
	observation := postgres.SchemaObservation{Snapshot: existingTablesSnapshot(), ConstraintData: checks}
	readiness, err := observation.AssessConstraintDataReadiness(existingTableAdditionsDiff())
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Executable || readiness.Additions[1].Reason != postgres.ConstraintDataUnobserved {
		t.Fatalf("readiness=%#v", readiness)
	}
	other := observation
	other.Snapshot.Database = "other"
	if _, err := other.AssessConstraintDataReadiness(existingTableAdditionsDiff()); err == nil {
		t.Fatal("expected database mismatch")
	}
}

type countingCatalog struct {
	fakeCatalog
	keyCalls     []string
	err          error
	missingEmpty []bool
}

func (c *countingCatalog) CountKeyViolations(_ context.Context, table postgres.RelationIdentity, columns []string, countNullKeys bool) (postgres.KeyViolationCounts, error) {
	if c.err != nil {
		return postgres.KeyViolationCounts{}, c.err
	}
	c.keyCalls = append(c.keyCalls, table.Name+":"+strings.Join(columns, ","))
	if countNullKeys {
		return postgres.KeyViolationCounts{DuplicateKeys: 1, DuplicateRows: 2, NullKeyRows: 3}, nil
	}
	return postgres.KeyViolationCounts{}, nil
}

func (c *countingCatalog) CountMissingReferences(_ context.Context, _ postgres.RelationIdentity, _ []string, _ postgres.RelationIdentity, _ []string, referencedEmpty bool) (int64, error) {
	if c.err != nil {
		return 0, c.err
	}
	c.missingEmpty = append(c.missingEmpty, referencedEmpty)
	return 7, nil
}

func TestObserveConstraintDataCountsEachExistingTableAddition(t *testing.T) {
	catalog := &countingCatalog{}
	diff := existingTableAdditionsDiff()
	// A constraint on a table created in the same diff needs no observation.
	diff.Changes = append(diff.Changes,
		schemadiff.Change{Kind: schemadiff.CreateTable, Table: "public.new_table"},
		schemadiff.Change{Kind: schemadiff.AddUniqueConstraint, Table: "public.new_table", UniqueConstraint: &physical.UniqueConstraint{Name: "new_table_x_key", Columns: []string{"x"}}},
	)
	observation, err := postgres.Provider{Catalog: catalog}.ObserveConstraintData(context.Background(), postgres.SchemaObservation{Snapshot: existingTablesSnapshot()}, diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(observation.ConstraintData) != 3 || strings.Join(catalog.keyCalls, ";") != "orders:id;orders:code" || !reflect.DeepEqual(catalog.missingEmpty, []bool{false}) {
		t.Fatalf("checks=%#v calls=%v", observation.ConstraintData, catalog.keyCalls)
	}
	pk, fk := observation.ConstraintData[0], observation.ConstraintData[2]
	if !pk.Observed || pk.DuplicateRows != 2 || pk.NullKeyRows != 3 || !fk.Observed || fk.MissingReferenceRows != 7 {
		t.Fatalf("checks=%#v", observation.ConstraintData)
	}
	if _, err := postgres.RenderMigrationSQLWithObservation(diff, observation); err == nil {
		t.Fatal("violations must refuse rendering")
	}
}

func TestObserveConstraintDataLeavesUncountableChecksUnobserved(t *testing.T) {
	for name, catalog := range map[string]postgres.CatalogReader{
		"reader without counting": fakeCatalog{},
		"count fails":             &countingCatalog{err: errors.New("permission denied for table orders")},
	} {
		t.Run(name, func(t *testing.T) {
			observation, err := postgres.Provider{Catalog: catalog}.ObserveConstraintData(context.Background(), postgres.SchemaObservation{Snapshot: existingTablesSnapshot()}, existingTableAdditionsDiff())
			if err != nil {
				t.Fatal(err)
			}
			for _, check := range observation.ConstraintData {
				if check.Observed || check.Reason == "" || check.DuplicateRows != 0 || check.MissingReferenceRows != 0 {
					t.Fatalf("check=%#v", check)
				}
			}
			if _, err := postgres.RenderMigrationSQLWithObservation(existingTableAdditionsDiff(), observation); err == nil || !strings.Contains(err.Error(), string(postgres.ConstraintDataUnobserved)) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestObserveConstraintDataMarksReferencedTableCreatedInSameDiff(t *testing.T) {
	catalog := &countingCatalog{}
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.CreateTable, Table: "public.regions"},
		{Kind: schemadiff.AddForeignKey, Table: "public.orders", ForeignKey: &physical.ForeignKey{
			Name: "orders_region_fkey", Columns: []string{"region"}, ReferencedSchema: "public", ReferencedTable: "regions", ReferencedColumns: []string{"code"}}},
	}}
	observation, err := postgres.Provider{Catalog: catalog}.ObserveConstraintData(context.Background(), postgres.SchemaObservation{Snapshot: existingTablesSnapshot()}, diff)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(catalog.missingEmpty, []bool{true}) || !observation.ConstraintData[0].ReferencedTableCreated {
		t.Fatalf("checks=%#v", observation.ConstraintData)
	}
}

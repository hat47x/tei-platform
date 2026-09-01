package postgres_test

import (
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

func TestRenderMigrationSQLRendersAdvancedIndexWithoutLosingSemantics(t *testing.T) {
	index := physical.Index{
		Name:   "customers_name_advanced_uidx",
		Unique: true,
		Method: "btree",
		Keys: []physical.IndexKey{
			{
				Column:        "name",
				Collation:     "pg_catalog.C",
				OperatorClass: "pg_catalog.varchar_pattern_ops",
				Order:         physical.IndexOrderDesc,
				NullsOrder:    physical.IndexNullsOrderLast,
			},
			{
				Expression: postgresExpression("lower(email)"),
				Order:      physical.IndexOrderDefault,
				NullsOrder: physical.IndexNullsOrderDefault,
			},
		},
		IncludedColumns:     []string{"external_id"},
		Predicate:           postgresExpression("profile IS NOT NULL"),
		UniqueNullTreatment: physical.UniqueNullTreatmentNotDistinct,
	}
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.AddIndex,
			Table: "public.customers",
			Index: &index,
		}},
	}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 {
		t.Fatalf("statements=%#v", statements)
	}
	want := `CREATE UNIQUE INDEX "customers_name_advanced_uidx" ON "public"."customers" ("name" COLLATE "pg_catalog"."C" "pg_catalog"."varchar_pattern_ops" DESC NULLS LAST, (lower(email))) INCLUDE ("external_id") NULLS NOT DISTINCT WHERE (profile IS NOT NULL);`
	if statements[0] != want {
		t.Fatalf("sql=\n%s\nwant=\n%s", statements[0], want)
	}
	if strings.Contains(statements[0], "CONCURRENTLY") {
		t.Fatalf("ordinary migration rendering must not choose an operational concurrency mode: %s", statements[0])
	}
}

func TestRenderMigrationSQLRendersProviderAccessMethod(t *testing.T) {
	index := physical.Index{
		Name:   "documents_search_idx",
		Method: "gin",
		Keys: []physical.IndexKey{{
			Expression: postgresExpression("to_tsvector('simple', body)"),
			Order:      physical.IndexOrderDefault,
			NullsOrder: physical.IndexNullsOrderDefault,
		}},
	}
	diff := schemadiff.Diff{Database: "search", Changes: []schemadiff.Change{{Kind: schemadiff.AddIndex, Table: "public.documents", Index: &index}}}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err != nil {
		t.Fatal(err)
	}
	want := `CREATE INDEX "documents_search_idx" ON "public"."documents" USING "gin" ((to_tsvector('simple', body)));`
	if len(statements) != 1 || statements[0] != want {
		t.Fatalf("statements=%#v", statements)
	}
}

func TestRenderMigrationSQLSchemaQualifiesDropIndex(t *testing.T) {
	index := physical.Index{Name: "customers_name_idx"}
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.DropIndex, Table: "sales.customers", Index: &index}}}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 || statements[0] != `DROP INDEX "sales"."customers_name_idx";` {
		t.Fatalf("statements=%#v", statements)
	}
}

func TestRenderMigrationSQLKeepsSimpleIndexCompatibility(t *testing.T) {
	index := physical.Index{
		Name:   "orders_customer_id_idx",
		Method: "btree",
		Keys:   []physical.IndexKey{physical.ColumnIndexKey("customer_id")},
	}
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.AddIndex, Table: "public.orders", Index: &index}}}

	statements, err := postgres.RenderMigrationSQL(diff)
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 1 || statements[0] != `CREATE INDEX "orders_customer_id_idx" ON "public"."orders" ("customer_id");` {
		t.Fatalf("statements=%#v", statements)
	}
}

func TestRenderMigrationSQLRejectsExpressionFromAnotherProvider(t *testing.T) {
	index := physical.Index{
		Name:   "bad_expression_idx",
		Method: "btree",
		Keys: []physical.IndexKey{{
			Expression: &physical.ProviderExpression{Provider: "oracle", Dialect: "oracle", Representation: physical.ProviderExpressionAuthored, Source: "lower(name)"},
			Order:      physical.IndexOrderDefault,
			NullsOrder: physical.IndexNullsOrderDefault,
		}},
	}
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.AddIndex, Table: "public.customers", Index: &index}}}

	_, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "is not PostgreSQL") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMigrationSQLRejectsBtreeOrderingOnNonBtreeMethod(t *testing.T) {
	index := physical.Index{
		Name:   "bad_gin_order_idx",
		Method: "gin",
		Keys: []physical.IndexKey{{
			Column:     "tags",
			Order:      physical.IndexOrderDesc,
			NullsOrder: physical.IndexNullsOrderDefault,
		}},
	}
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.AddIndex, Table: "public.documents", Index: &index}}}

	_, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "supported only for PostgreSQL B-tree indexes") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMigrationSQLRejectsUnresolvedComparison(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Unresolved: []schemadiff.UnresolvedComparison{{
			Kind:       schemadiff.UnresolvedProviderIdentifier,
			Table:      "public.customers",
			ObjectKind: "index",
			ObjectName: "customers_name_idx",
			FieldPath:  "index.keys[0].collation",
			Reason:     schemadiff.UnresolvedNormalizationRequired,
		}},
	}

	_, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "fully resolved schema diff") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderMigrationSQLDoesNotTurnIndexStateDriftIntoDDL(t *testing.T) {
	index := physical.Index{Name: "customers_name_idx", State: &physical.IndexState{Valid: false, Ready: false, Live: true}}
	diff := schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{Kind: schemadiff.IndexStateDrift, Table: "public.customers", Index: &index}}}

	_, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "operational recovery decision") {
		t.Fatalf("err=%v", err)
	}
}

func postgresExpression(source string) *physical.ProviderExpression {
	return &physical.ProviderExpression{
		Provider:       "postgres",
		Dialect:        "postgresql",
		Representation: physical.ProviderExpressionAuthored,
		Source:         source,
	}
}

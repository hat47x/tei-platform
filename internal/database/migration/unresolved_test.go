package migration_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func TestRenderRejectsUnresolvedSchemaDiff(t *testing.T) {
	diff := unresolvedDiff()
	_, err := migration.Render(environment.ProviderPostgres, diff)
	if err == nil || !strings.Contains(err.Error(), "fully resolved schema diff") {
		t.Fatalf("err=%v", err)
	}
}

func TestPlanRejectsUnresolvedSchemaDiff(t *testing.T) {
	diff := unresolvedDiff()
	_, err := migration.Plan("20260901_001", core.MustCanonicalID("sales_core"), "fp-1", diff, environment.ProviderPostgres)
	if err == nil || !strings.Contains(err.Error(), "fully resolved schema diff") {
		t.Fatalf("err=%v", err)
	}
}

func unresolvedDiff() schemadiff.Diff {
	return schemadiff.Diff{
		Database: "sales_core",
		Unresolved: []schemadiff.UnresolvedComparison{{
			Kind:       schemadiff.UnresolvedProviderExpression,
			Table:      "public.customers",
			ObjectKind: "index",
			ObjectName: "customers_lower_name_idx",
			FieldPath:  "index.keys[0].expression",
			Reason:     schemadiff.UnresolvedNormalizationRequired,
			DesiredExpression: &physical.ProviderExpression{
				Provider: "postgres",
				Dialect:  "postgresql",
				Source:   "lower(name)",
			},
			ActualExpression: &physical.ProviderExpression{
				Provider: "postgres",
				Dialect:  "postgresql",
				Source:   "lower((name)::text)",
			},
		}},
	}
}

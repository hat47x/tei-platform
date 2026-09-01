package migration_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/database/migration"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func TestPlanRejectsUnresolvedProviderIdentifier(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Unresolved: []schemadiff.UnresolvedComparison{{
			Kind:         schemadiff.UnresolvedProviderIdentifier,
			Table:        "public.customers",
			ObjectKind:   "index",
			ObjectName:   "customers_name_idx",
			FieldPath:    "index.keys[0].collation",
			Reason:       schemadiff.UnresolvedNormalizationRequired,
			DesiredValue: "pg_catalog.default",
		}},
	}

	_, err := migration.Plan("20260901_002", core.MustCanonicalID("sales_core"), "fp-1", diff, environment.ProviderPostgres)
	if err == nil || !strings.Contains(err.Error(), "fully resolved schema diff") {
		t.Fatalf("err=%v", err)
	}
}

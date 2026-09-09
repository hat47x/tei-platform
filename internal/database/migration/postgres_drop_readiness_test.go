package migration_test

import (
	"strings"
	"testing"

	"tei/internal/database/migration"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func TestRenderPostgresDropTableRequiresProviderPartitionObservation(t *testing.T) {
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.DropTable,
			Table: "public.audit_log",
		}},
	}

	statements, err := migration.Render(environment.ProviderPostgres, diff)
	if err == nil || !strings.Contains(err.Error(), "requires complete provider partition observation") {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	if statements != nil {
		t.Fatalf("blocked rendering returned statements=%#v", statements)
	}
}

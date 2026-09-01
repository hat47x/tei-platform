package postgres_test

import (
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

func TestRenderMigrationSQLRejectsDropIndexWithoutSchema(t *testing.T) {
	index := physical.Index{Name: "customers_name_idx"}
	diff := schemadiff.Diff{
		Database: "sales_core",
		Changes: []schemadiff.Change{{
			Kind:  schemadiff.DropIndex,
			Table: "customers",
			Index: &index,
		}},
	}

	_, err := postgres.RenderMigrationSQL(diff)
	if err == nil || !strings.Contains(err.Error(), "not schema-qualified") {
		t.Fatalf("err=%v", err)
	}
}

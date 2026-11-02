package postgres_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

// #1803: a declared constraint name that differs from the observed one is
// renamed from the observed name, only with an observation that finds the
// observed name and a free new name.

func renameUniqueDiff() schemadiff.Diff {
	return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{
		Kind: schemadiff.RenameConstraint, Table: "public.customers", BeforeName: "customers_email_key",
		UniqueConstraint: &physical.UniqueConstraint{Name: "uq_email", Columns: []string{"email"}},
	}}}
}

func renameForeignKeyDiff() schemadiff.Diff {
	return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{{
		Kind: schemadiff.RenameConstraint, Table: "public.customers", BeforeName: "customers_region_fkey",
		ForeignKey: &physical.ForeignKey{Name: "fk_region", Columns: []string{"region"}, ReferencedSchema: "public", ReferencedTable: "regions", ReferencedColumns: []string{"code"}},
	}}}
}

func renameCheck(kind postgres.ConstraintRenameKind, before, after string) postgres.ConstraintDataCheck {
	c := postgres.ConstraintDataCheck{Kind: postgres.ConstraintDataRenameConstraint, Table: "public.customers", RenameKind: kind, BeforeName: before, NewName: after, Observed: true, SourceFound: true}
	if kind == postgres.ConstraintRenameUnique {
		c.Columns = []string{"email"}
	} else {
		c.Columns = []string{"region"}
	}
	return c
}

func observationWith(checks ...postgres.ConstraintDataCheck) postgres.SchemaObservation {
	return postgres.SchemaObservation{Snapshot: physical.SchemaSnapshot{Database: "sales_core"}, ConstraintData: checks}
}

func TestRenderRenamesConstraintsFromTheObservedName(t *testing.T) {
	statements, err := postgres.RenderMigrationSQLWithObservation(renameUniqueDiff(), observationWith(renameCheck(postgres.ConstraintRenameUnique, "customers_email_key", "uq_email")))
	if err != nil || len(statements) != 1 || statements[0] != `ALTER TABLE "public"."customers" RENAME CONSTRAINT "customers_email_key" TO "uq_email";` {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
	statements, err = postgres.RenderMigrationSQLWithObservation(renameForeignKeyDiff(), observationWith(renameCheck(postgres.ConstraintRenameForeignKey, "customers_region_fkey", "fk_region")))
	if err != nil || len(statements) != 1 || statements[0] != `ALTER TABLE "public"."customers" RENAME CONSTRAINT "customers_region_fkey" TO "fk_region";` {
		t.Fatalf("statements=%#v err=%v", statements, err)
	}
}

func TestRenderConstraintRenameRefusals(t *testing.T) {
	cases := map[string]struct {
		observation postgres.SchemaObservation
		reason      postgres.ConstraintDataReadinessReason
	}{
		"no observation": {postgres.SchemaObservation{Snapshot: physical.SchemaSnapshot{Database: "sales_core"}}, postgres.ConstraintDataUnobserved},
		"not observed": {observationWith(func() postgres.ConstraintDataCheck {
			c := renameCheck(postgres.ConstraintRenameUnique, "customers_email_key", "uq_email")
			c.Observed, c.SourceFound, c.Reason = false, false, "no privilege"
			return c
		}()), postgres.ConstraintDataUnobserved},
		"source missing": {observationWith(func() postgres.ConstraintDataCheck {
			c := renameCheck(postgres.ConstraintRenameUnique, "customers_email_key", "uq_email")
			c.SourceFound = false
			return c
		}()), postgres.ConstraintRenameSourceMissing},
		"name taken": {observationWith(func() postgres.ConstraintDataCheck {
			c := renameCheck(postgres.ConstraintRenameUnique, "customers_email_key", "uq_email")
			c.NameOccupiedBy = "relation of kind i"
			return c
		}()), postgres.ConstraintRenameNameTaken},
		"observation for another name": {observationWith(renameCheck(postgres.ConstraintRenameUnique, "other_key", "uq_email")), postgres.ConstraintDataUnobserved},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			statements, err := postgres.RenderMigrationSQLWithObservation(renameUniqueDiff(), tc.observation)
			var blocked *postgres.ConstraintDataReadinessError
			if !errors.As(err, &blocked) || statements != nil || blocked.Readiness.Additions[0].Reason != tc.reason {
				t.Fatalf("statements=%#v err=%v", statements, err)
			}
		})
	}
	if _, err := postgres.RenderMigrationSQL(renameUniqueDiff()); err == nil || !strings.Contains(err.Error(), "without an existing-data observation") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderRefusesRenameTargetClaimedTwice(t *testing.T) {
	diff := renameUniqueDiff()
	diff.Changes = append(diff.Changes, schemadiff.Change{
		Kind: schemadiff.AddUniqueConstraint, Table: "public.customers",
		UniqueConstraint: &physical.UniqueConstraint{Name: "uq_email", Columns: []string{"code"}},
	})
	added := postgres.ConstraintDataCheck{Kind: postgres.ConstraintDataUnique, Table: "public.customers", Columns: []string{"code"}, Observed: true}
	_, err := postgres.RenderMigrationSQLWithObservation(diff, observationWith(renameCheck(postgres.ConstraintRenameUnique, "customers_email_key", "uq_email"), added))
	if err == nil || !strings.Contains(err.Error(), "claimed twice") {
		t.Fatalf("err=%v", err)
	}
}

func TestRenderRefusesMalformedRename(t *testing.T) {
	diff := renameUniqueDiff()
	diff.Changes[0].BeforeName = ""
	if _, err := postgres.RenderMigrationSQLWithObservation(diff, observationWith()); err == nil {
		t.Fatal("a rename without an observed name must be refused")
	}
	diff = renameUniqueDiff()
	diff.Changes[0].ForeignKey = &physical.ForeignKey{}
	if _, err := postgres.RenderMigrationSQLWithObservation(diff, observationWith()); err == nil {
		t.Fatal("a rename carrying two constraints must be refused")
	}
}

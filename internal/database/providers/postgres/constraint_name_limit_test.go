package postgres_test

import (
	"strings"
	"testing"

	"tei/internal/database/physical"
	"tei/internal/database/providers/postgres"
	"tei/internal/database/schemadiff"
)

func uniqueNameDiff(name string) schemadiff.Diff {
	return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.CreateTable, Table: "public.child"},
		{Kind: schemadiff.AddUniqueConstraint, Table: "public.child",
			UniqueConstraint: &physical.UniqueConstraint{Name: name, Columns: []string{"code"}}},
	}}
}

func foreignKeyNameDiff(name string) schemadiff.Diff {
	return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.CreateTable, Table: "public.parent"},
		{Kind: schemadiff.CreateTable, Table: "public.child"},
		{Kind: schemadiff.AddForeignKey, Table: "public.child",
			ForeignKey: &physical.ForeignKey{Name: name, Columns: []string{"parent_code"},
				ReferencedSchema: "public", ReferencedTable: "parent", ReferencedColumns: []string{"code"}}},
	}}
}

func renameNameDiff(name string) schemadiff.Diff {
	return schemadiff.Diff{Database: "sales_core", Changes: []schemadiff.Change{
		{Kind: schemadiff.RenameConstraint, Table: "public.customers", BeforeName: "legacy_name",
			UniqueConstraint: &physical.UniqueConstraint{Name: name, Columns: []string{"email"}}},
	}}
}

func TestRenderAcceptsDeclaredConstraintNameAt63Bytes(t *testing.T) {
	name := strings.Repeat("u", 63)
	statements, err := postgres.RenderMigrationSQL(uniqueNameDiff(name))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(statements, "\n"), `"`+name+`"`) {
		t.Fatalf("statements=%v", statements)
	}
}

func TestRenderRefusesDeclaredConstraintNameOver63Bytes(t *testing.T) {
	cases := map[string]schemadiff.Diff{
		"unique":      uniqueNameDiff(strings.Repeat("u", 64)),
		"foreign key": foreignKeyNameDiff(strings.Repeat("f", 64)),
		"rename":      renameNameDiff(strings.Repeat("r", 64)),
		// 22 three-byte characters are 66 bytes, although only 22 runes.
		"multibyte": uniqueNameDiff(strings.Repeat("あ", 22)),
	}
	for label, diff := range cases {
		t.Run(label, func(t *testing.T) {
			statements, err := postgres.RenderMigrationSQL(diff)
			if err == nil || statements != nil {
				t.Fatalf("statements=%v err=%v", statements, err)
			}
			if !strings.Contains(err.Error(), "63 bytes") || !strings.Contains(err.Error(), "bytes; PostgreSQL truncates") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestRenderRefusesMultibyteNameByBytesNotRunes(t *testing.T) {
	// 21 three-byte characters are exactly 63 bytes and are accepted.
	statements, err := postgres.RenderMigrationSQL(uniqueNameDiff(strings.Repeat("あ", 21)))
	if err != nil {
		t.Fatalf("21 three-byte characters should fit: %v", err)
	}
	if len(statements) == 0 {
		t.Fatal("no statements")
	}
	// 22 are 66 bytes and are refused even though the rune count is small.
	_, err = postgres.RenderMigrationSQL(uniqueNameDiff(strings.Repeat("あ", 22)))
	if err == nil || !strings.Contains(err.Error(), "66 bytes") {
		t.Fatalf("err=%v", err)
	}
}

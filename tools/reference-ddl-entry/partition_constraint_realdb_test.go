package ddlentry

import (
	"context"
	"testing"

	"tei/internal/database/providers/postgres"
)

// A foreign key that references a partitioned table has a top-level constraint
// and an internal child constraint (conparentid != 0) on the referencing
// table. The catalog reader must expose only the top-level one: otherwise the
// observed schema contains the same foreign key twice, and a rename never
// converges. The rename itself works on the top-level constraint.
func TestRealPostgresPartitionReferencedForeignKeyRenameObservedOnce(t *testing.T) {
	db, _ := throwawayDBWithDSN(t, realDSN(t))
	ctx := context.Background()
	for _, statement := range []string{
		`CREATE TABLE "public"."p" (id int NOT NULL, code text, PRIMARY KEY (id)) PARTITION BY RANGE (id)`,
		`CREATE TABLE "public"."p1" PARTITION OF "public"."p" FOR VALUES FROM (0) TO (100)`,
		`CREATE TABLE "public"."refs" (id int, pid int, CONSTRAINT legacy_fk FOREIGN KEY (pid) REFERENCES "public"."p" (id))`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("setup %q: %v", statement, err)
		}
	}
	reader := postgres.SQLCatalogReader{DB: db}
	foreignKeyNames := func() []string {
		fks, err := reader.ForeignKeys(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, fk := range fks {
			if fk.Schema == "public" && fk.Table == "refs" {
				out = append(out, fk.Name)
			}
		}
		return out
	}
	if got := foreignKeyNames(); len(got) != 1 || got[0] != "legacy_fk" {
		t.Fatalf("observed foreign keys = %v; the internal child must not be a second foreign key", got)
	}
	table := postgres.RelationIdentity{Schema: "public", Name: "refs"}
	facts, err := reader.ConstraintRenameFacts(ctx, table, postgres.ConstraintRenameForeignKey, "legacy_fk", "declared_fk")
	if err != nil {
		t.Fatal(err)
	}
	if !facts.SourceFound || facts.NameOccupiedBy != "" {
		t.Fatalf("facts before rename = %+v", facts)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE "public"."refs" RENAME CONSTRAINT legacy_fk TO declared_fk`); err != nil {
		t.Fatal(err)
	}
	if got := foreignKeyNames(); len(got) != 1 || got[0] != "declared_fk" {
		t.Fatalf("observed foreign keys after rename = %v", got)
	}
	// The internal child constraint still exists in the catalog, but it is not
	// an independent foreign key of the table.
	var top, internal int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FILTER (WHERE conparentid = 0), count(*) FILTER (WHERE conparentid <> 0)
		FROM pg_catalog.pg_constraint WHERE conrelid = 'public.refs'::regclass AND contype = 'f'`).Scan(&top, &internal); err != nil {
		t.Fatal(err)
	}
	if top != 1 || internal != 1 {
		t.Fatalf("top-level=%d internal=%d, want 1 and 1", top, internal)
	}
	facts, err = reader.ConstraintRenameFacts(ctx, table, postgres.ConstraintRenameForeignKey, "legacy_fk", "declared_fk")
	if err != nil || facts.SourceFound {
		t.Fatalf("the old top-level name is still a rename source: %+v %v", facts, err)
	}
}

// A unique constraint on a partitioned table is likewise top-level only; the
// partition's copied constraint has conparentid != 0 and must not be observed
// as a second constraint. Renaming the parent's constraint works.
func TestRealPostgresPartitionedUniqueConstraintRenameObservedOnce(t *testing.T) {
	db, _ := throwawayDBWithDSN(t, realDSN(t))
	ctx := context.Background()
	for _, statement := range []string{
		`CREATE TABLE "public"."p" (id int NOT NULL, code text, PRIMARY KEY (id)) PARTITION BY RANGE (id)`,
		`CREATE TABLE "public"."p1" PARTITION OF "public"."p" FOR VALUES FROM (0) TO (100)`,
		`ALTER TABLE "public"."p" ADD CONSTRAINT p_id_code_key UNIQUE (id, code)`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("setup %q: %v", statement, err)
		}
	}
	reader := postgres.SQLCatalogReader{DB: db}
	uniqueNames := func(table string) []string {
		constraints, err := reader.UniqueConstraints(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, constraint := range constraints {
			if constraint.Schema == "public" && constraint.Table == table {
				out = append(out, constraint.Name)
			}
		}
		return out
	}
	if got := uniqueNames("p"); len(got) != 1 || got[0] != "p_id_code_key" {
		t.Fatalf("parent unique constraints = %v", got)
	}
	if got := uniqueNames("p1"); len(got) != 0 {
		t.Fatalf("partition unique constraints = %v; internal copies must not be independent", got)
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE "public"."p" RENAME CONSTRAINT p_id_code_key TO uq_p_id_code`); err != nil {
		t.Fatal(err)
	}
	if got := uniqueNames("p"); len(got) != 1 || got[0] != "uq_p_id_code" {
		t.Fatalf("parent unique constraints after rename = %v", got)
	}
	if got := uniqueNames("p1"); len(got) != 0 {
		t.Fatalf("partition unique constraints after rename = %v", got)
	}
}

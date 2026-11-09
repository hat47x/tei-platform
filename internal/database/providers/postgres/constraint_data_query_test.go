package postgres

import "testing"

// The counting queries quote every identifier and select counts only.
func TestConstraintDataQueriesQuoteIdentifiersAndReturnCountsOnly(t *testing.T) {
	table := RelationIdentity{Schema: "public", Name: `or"ders`}
	got := keyViolationQuery(table, []string{"a", "b"}, true)
	want := `SELECT count(*), COALESCE(sum(d.n), 0)::bigint, (SELECT count(*) FROM "public"."or""ders" WHERE "a" IS NULL OR "b" IS NULL) FROM (SELECT count(*) AS n FROM "public"."or""ders" WHERE "a" IS NOT NULL AND "b" IS NOT NULL GROUP BY "a", "b" HAVING count(*) > 1) AS d`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if got := keyViolationQuery(table, []string{"a"}, false); got != `SELECT count(*), COALESCE(sum(d.n), 0)::bigint, 0::bigint FROM (SELECT count(*) AS n FROM "public"."or""ders" WHERE "a" IS NOT NULL GROUP BY "a" HAVING count(*) > 1) AS d` {
		t.Fatalf("unique query = %s", got)
	}

	parent := RelationIdentity{Schema: "public", Name: "customers"}
	got, err := missingReferenceQuery(table, []string{"x", "y"}, parent, []string{"p", "q"}, false)
	if err != nil {
		t.Fatal(err)
	}
	want = `SELECT count(*) FROM "public"."or""ders" AS child WHERE child."x" IS NOT NULL AND child."y" IS NOT NULL AND NOT EXISTS (SELECT 1 FROM "public"."customers" AS parent WHERE parent."p" = child."x" AND parent."q" = child."y")`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	got, err = missingReferenceQuery(table, []string{"x"}, parent, []string{"p"}, true)
	if err != nil || got != `SELECT count(*) FROM "public"."or""ders" AS child WHERE child."x" IS NOT NULL` {
		t.Fatalf("empty-referenced query = %s err=%v", got, err)
	}
	if _, err := missingReferenceQuery(table, []string{"x"}, parent, nil, false); err == nil {
		t.Fatal("expected column count mismatch")
	}
}

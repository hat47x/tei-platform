package postgres_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"unicode/utf8"

	"tei/internal/database/persistence"
	"tei/internal/database/providers/postgres"
)

func TestConstraintNameFollowsPostgresDefaultForShortNames(t *testing.T) {
	p := postgres.RealizationPolicy{}
	cases := []struct {
		kind    persistence.ConstraintNameKind
		table   string
		columns []string
		want    string
	}{
		// Observed on PostgreSQL 16.15 for unnamed ADD UNIQUE / ADD FOREIGN KEY.
		{persistence.ConstraintNameUnique, "child", []string{"label"}, "child_label_key"},
		{persistence.ConstraintNameUnique, "child", []string{"pb", "pa"}, "child_pb_pa_key"},
		{persistence.ConstraintNameUnique, "child", []string{"Mixed Col"}, "child_Mixed Col_key"},
		{persistence.ConstraintNameForeignKey, "child", []string{"pa", "pb"}, "child_pa_pb_fkey"},
		{persistence.ConstraintNameForeignKey, "child", []string{"pb", "pa"}, "child_pb_pa_fkey"},
		{persistence.ConstraintNameUnique, "表テーブル", []string{"列", "列二"}, "表テーブル_列_列二_key"},
		{persistence.ConstraintNamePrimaryKey, "child", []string{"id"}, "child_pkey"},
	}
	for _, c := range cases {
		if got := p.ConstraintName(c.kind, c.table, c.columns); got != c.want {
			t.Errorf("%s %s %v = %q, want %q", c.kind, c.table, c.columns, got, c.want)
		}
	}
}

func TestConstraintNameKeepsExactly63Bytes(t *testing.T) {
	p := postgres.RealizationPolicy{}
	// 4 + 1 + 54 + 4 = 63 bytes: kept as PostgreSQL keeps it.
	col := strings.Repeat("c", 54)
	got := p.ConstraintName(persistence.ConstraintNameUnique, "tabl", []string{col})
	if want := "tabl_" + col + "_key"; got != want || len(got) != 63 {
		t.Fatalf("got %q (%d bytes)", got, len(got))
	}
}

func TestConstraintNameShortensLongNamesWithDigest(t *testing.T) {
	p := postgres.RealizationPolicy{}
	table := "a_very_long_table_name_for_testing_identifier_limits_xx"
	columns := []string{"first_long_column_name", "second_long_column_name"}
	full := table + "_first_long_column_name_second_long_column_name_key"
	sum := sha256.Sum256([]byte(full))
	want := full[:54] + "_" + hex.EncodeToString(sum[:])[:8]
	got := p.ConstraintName(persistence.ConstraintNameUnique, table, columns)
	if got != want || len(got) != 63 {
		t.Fatalf("got %q (%d bytes), want %q", got, len(got), want)
	}
	// PostgreSQL itself would have chosen a truncated name keeping "_key";
	// the rule deliberately differs (design 4.2, provisional).
	if got == "a_very_long_table_name_for_te_first_long_column_name_second_key" {
		t.Fatal("rule must not rely on PostgreSQL truncation")
	}
	fk := p.ConstraintName(persistence.ConstraintNameForeignKey, table, columns)
	if fk == got || len(fk) != 63 {
		t.Fatalf("foreign key name %q must differ from unique name %q", fk, got)
	}
	if again := p.ConstraintName(persistence.ConstraintNameUnique, table, columns); again != got {
		t.Fatalf("not deterministic: %q != %q", again, got)
	}
}

func TestConstraintNameCutsMultibyteNamesAtCharacterBoundary(t *testing.T) {
	p := postgres.RealizationPolicy{}
	table := strings.Repeat("表", 15) // 45 bytes
	columns := []string{"列あ", "列い"}
	full := table + "_列あ_列い_fkey"
	got := p.ConstraintName(persistence.ConstraintNameForeignKey, table, columns)
	if len(got) > 63 || !utf8.ValidString(got) {
		t.Fatalf("got %q (%d bytes, valid=%v)", got, len(got), utf8.ValidString(got))
	}
	sum := sha256.Sum256([]byte(full))
	suffix := "_" + hex.EncodeToString(sum[:])[:8]
	if !strings.HasSuffix(got, suffix) {
		t.Fatalf("got %q, want suffix %q", got, suffix)
	}
	base := strings.TrimSuffix(got, suffix)
	// Byte 54 falls inside "列" (bytes 53..55); the cut steps back to 53.
	if !strings.HasPrefix(full, base) || len(base) != 53 {
		t.Fatalf("base %q (%d bytes) is not the boundary-clipped prefix of %q", base, len(base), full)
	}
}

func TestConstraintNameCollisionsFollowPostgresNamespaces(t *testing.T) {
	p := postgres.RealizationPolicy{}
	gen := func(name string) persistence.ConstraintNameEntry {
		return persistence.ConstraintNameEntry{Name: name, Generated: true}
	}
	decl := func(name string) persistence.ConstraintNameEntry {
		return persistence.ConstraintNameEntry{Name: name}
	}
	// A unique constraint backs an index: it meets index names of other
	// tables in the same schema.
	got := p.ConstraintNameCollisions([]persistence.ConstraintNameTable{
		{Schema: "public", Table: "a", Unique: []persistence.ConstraintNameEntry{gen("a_x_key")}},
		{Schema: "public", Table: "b", Indexes: []string{"a_x_key"}},
	})
	if len(got) != 1 || got[0].Table != "public.a" || got[0].Name != "a_x_key" || !strings.Contains(got[0].Reason, `index "a_x_key" on public.b`) {
		t.Fatalf("unique vs index: %#v", got)
	}
	// The same names in another schema do not collide.
	if got := p.ConstraintNameCollisions([]persistence.ConstraintNameTable{
		{Schema: "public", Table: "a", Unique: []persistence.ConstraintNameEntry{gen("a_x_key")}},
		{Schema: "other", Table: "b", Indexes: []string{"a_x_key"}},
	}); len(got) != 0 {
		t.Fatalf("different schema: %#v", got)
	}
	// Foreign keys are not relations: the same name on two tables is allowed.
	if got := p.ConstraintNameCollisions([]persistence.ConstraintNameTable{
		{Schema: "public", Table: "a", Foreign: []persistence.ConstraintNameEntry{gen("same_fkey")}},
		{Schema: "public", Table: "b", Foreign: []persistence.ConstraintNameEntry{gen("same_fkey")}},
	}); len(got) != 0 {
		t.Fatalf("foreign keys on different tables: %#v", got)
	}
	// Two generated foreign keys on one table collide (PostgreSQL would add a digit).
	got = p.ConstraintNameCollisions([]persistence.ConstraintNameTable{
		{Schema: "public", Table: "a", Foreign: []persistence.ConstraintNameEntry{gen("a_p_fkey"), gen("a_p_fkey")}},
	})
	if len(got) == 0 || got[0].Table != "public.a" {
		t.Fatalf("two generated foreign keys: %#v", got)
	}
	// A generated foreign key against a declared unique constraint of its table.
	got = p.ConstraintNameCollisions([]persistence.ConstraintNameTable{
		{Schema: "public", Table: "a", Unique: []persistence.ConstraintNameEntry{decl("a_p_fkey")}, Foreign: []persistence.ConstraintNameEntry{gen("a_p_fkey")}},
	})
	if len(got) != 1 || !strings.Contains(got[0].Reason, "(declared)") {
		t.Fatalf("generated vs declared: %#v", got)
	}
	// Collisions between declared names only are not this rule's concern.
	if got := p.ConstraintNameCollisions([]persistence.ConstraintNameTable{
		{Schema: "public", Table: "a", Unique: []persistence.ConstraintNameEntry{decl("dup"), decl("dup")}},
	}); len(got) != 0 {
		t.Fatalf("declared only: %#v", got)
	}
	// A generated unique name meeting the primary key's default name.
	got = p.ConstraintNameCollisions([]persistence.ConstraintNameTable{
		{Schema: "public", Table: "a", PrimaryKey: "a_pkey"},
		{Schema: "public", Table: "b", Unique: []persistence.ConstraintNameEntry{gen("a_pkey")}},
	})
	if len(got) != 1 || got[0].Table != "public.b" {
		t.Fatalf("unique vs primary key: %#v", got)
	}
}

package syntax_test

import (
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/environment"
	"tei/internal/syntax"
)

func TestParseAdvancedIndexAttributeKey(t *testing.T) {
	src := `index customers_name_advanced_idx {
    method btree
    key attribute customer.name {
        collation "pg_catalog.C"
        operator-class "pg_catalog.varchar_pattern_ops"
        order desc
        nulls last
    }
    include customer.external_id
    predicate {
        provider postgres
        dialect postgresql
        source "profile IS NOT NULL"
    }
}`
	index, err := syntax.ParseAdvancedIndex(src)
	if err != nil {
		t.Fatal(err)
	}
	if index.Name != "customers_name_advanced_idx" || index.Unique || index.Method != "btree" {
		t.Fatalf("index=%#v", index)
	}
	if len(index.Keys) != 1 {
		t.Fatalf("keys=%#v", index.Keys)
	}
	key := index.Keys[0]
	if key.Attribute != "customer.name" || key.Expression != nil {
		t.Fatalf("key=%#v", key)
	}
	if key.Collation != "pg_catalog.C" || key.OperatorClass != "pg_catalog.varchar_pattern_ops" || key.Order != persistence.IndexOrderDesc || key.NullsOrder != persistence.IndexNullsOrderLast {
		t.Fatalf("key options=%#v", key)
	}
	if len(index.IncludedAttributes) != 1 || index.IncludedAttributes[0] != "customer.external_id" {
		t.Fatalf("include=%#v", index.IncludedAttributes)
	}
	if index.Predicate == nil || index.Predicate.Provider != environment.ProviderPostgres || index.Predicate.Dialect != "postgresql" || index.Predicate.Source != "profile IS NOT NULL" {
		t.Fatalf("predicate=%#v", index.Predicate)
	}
}

func TestParseAdvancedUniqueExpressionIndex(t *testing.T) {
	src := `unique-index customers_lower_name_uidx {
    key expression {
        provider postgres
        dialect postgresql
        source "lower(name)"
        order asc
        nulls first
    }
    null-treatment not-distinct
}`
	index, err := syntax.ParseAdvancedIndex(src)
	if err != nil {
		t.Fatal(err)
	}
	if !index.Unique || index.UniqueNullTreatment != persistence.UniqueNullTreatmentNotDistinct || len(index.Keys) != 1 {
		t.Fatalf("index=%#v", index)
	}
	key := index.Keys[0]
	if key.Attribute != "" || key.Expression == nil {
		t.Fatalf("key=%#v", key)
	}
	if key.Expression.Provider != environment.ProviderPostgres || key.Expression.Dialect != "postgresql" || key.Expression.Source != "lower(name)" {
		t.Fatalf("expression=%#v", key.Expression)
	}
	if key.Order != persistence.IndexOrderAsc || key.NullsOrder != persistence.IndexNullsOrderFirst {
		t.Fatalf("key=%#v", key)
	}
}

func TestParseAdvancedIndexRejectsInvalidForms(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{name: "no key", src: `index empty_idx { method btree }`},
		{name: "non unique null treatment", src: `index bad_idx { key attribute customer.name null-treatment not-distinct }`},
		{name: "unknown provider", src: `index bad_idx { key expression { provider mystery dialect sql source "x" } }`},
		{name: "missing expression source", src: `index bad_idx { key expression { provider postgres dialect postgresql } }`},
		{name: "unknown order", src: `index bad_idx { key attribute customer.name { order sideways } }`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := syntax.ParseAdvancedIndex(tt.src); err == nil {
				t.Fatalf("expected parse error for %s", tt.src)
			}
		})
	}
}

func TestAdvancedIndexIdentifierAnchor(t *testing.T) {
	src := "unique-index customers_lower_name_uidx {\n    key attribute customer.name\n}\n"
	anchor, err := syntax.AdvancedIndexIdentifierAnchor(src, "customers_lower_name_uidx")
	if err != nil {
		t.Fatal(err)
	}
	got := src[anchor.Start.Offset:anchor.End.Offset]
	if got != "customers_lower_name_uidx" {
		t.Fatalf("anchor=%#v text=%q", anchor, got)
	}
	if anchor.Start.Line != 1 || anchor.Start.Column != 14 {
		t.Fatalf("anchor start=%#v", anchor.Start)
	}
}

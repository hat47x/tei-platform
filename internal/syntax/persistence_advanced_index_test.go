package syntax_test

import (
	"testing"

	"tei/internal/database/persistence"
	"tei/internal/environment"
	"tei/internal/syntax"
)

func TestParsePersistenceSupportsSimpleAndAdvancedIndexes(t *testing.T) {
	src := `persist sales_core.customer {
    database sales_core
    semantic customer.customer
    table public.customers
    column customer.name -> name storage varchar required
    column customer.external_id -> external_id storage varchar required
    column customer.profile -> profile storage json optional

    index customers_external_id_idx on customer.external_id

    unique-index customers_name_advanced_uidx {
        method btree
        key attribute customer.name {
            collation "pg_catalog.C"
            operator-class "pg_catalog.varchar_pattern_ops"
            order desc
            nulls last
        }
        key expression {
            provider postgres
            dialect postgresql
            source "lower(name)"
        }
        include customer.external_id
        predicate {
            provider postgres
            dialect postgresql
            source "profile IS NOT NULL"
        }
        null-treatment not-distinct
    }
}`

	mapping, err := syntax.NewParser(src).ParsePersistence()
	if err != nil {
		t.Fatal(err)
	}
	if len(mapping.Indexes) != 2 {
		t.Fatalf("indexes=%#v", mapping.Indexes)
	}

	simple := mapping.Indexes[0]
	if simple.Name != "customers_external_id_idx" || simple.Unique || len(simple.Attributes) != 1 || simple.Attributes[0] != "customer.external_id" || len(simple.Keys) != 0 {
		t.Fatalf("simple=%#v", simple)
	}

	advanced := mapping.Indexes[1]
	if advanced.Name != "customers_name_advanced_uidx" || !advanced.Unique || advanced.Method != "btree" || len(advanced.Attributes) != 0 || len(advanced.Keys) != 2 {
		t.Fatalf("advanced=%#v", advanced)
	}
	if advanced.Keys[0].Attribute != "customer.name" || advanced.Keys[0].Order != persistence.IndexOrderDesc || advanced.Keys[0].NullsOrder != persistence.IndexNullsOrderLast {
		t.Fatalf("attribute key=%#v", advanced.Keys[0])
	}
	if advanced.Keys[1].Expression == nil || advanced.Keys[1].Expression.Provider != environment.ProviderPostgres || advanced.Keys[1].Expression.Dialect != "postgresql" || advanced.Keys[1].Expression.Source != "lower(name)" {
		t.Fatalf("expression key=%#v", advanced.Keys[1])
	}
	if len(advanced.IncludedAttributes) != 1 || advanced.IncludedAttributes[0] != "customer.external_id" {
		t.Fatalf("include=%#v", advanced.IncludedAttributes)
	}
	if advanced.Predicate == nil || advanced.Predicate.Source != "profile IS NOT NULL" {
		t.Fatalf("predicate=%#v", advanced.Predicate)
	}
	if advanced.UniqueNullTreatment != persistence.UniqueNullTreatmentNotDistinct {
		t.Fatalf("null treatment=%q", advanced.UniqueNullTreatment)
	}
}

func TestPersistenceIndexIdentifierAnchorFindsSimpleAndAdvancedIndex(t *testing.T) {
	src := `persist sales_core.customer {
    database sales_core
    semantic customer.customer
    table public.customers
    column customer.name -> name storage varchar required

    index simple_idx on customer.name

    index advanced_idx {
        key expression {
            provider postgres
            dialect postgresql
            source "jsonb_build_object('index', name)"
        }
    }
}`

	for _, name := range []string{"simple_idx", "advanced_idx"} {
		anchor, err := syntax.PersistenceIndexIdentifierAnchor(src, "sales_core.customer", name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := src[anchor.Start.Offset:anchor.End.Offset]; got != name {
			t.Fatalf("%s anchor text=%q anchor=%#v", name, got, anchor)
		}
	}
}

func TestParsePersistenceRejectsMalformedIndexContinuation(t *testing.T) {
	src := `persist sales_core.customer {
    database sales_core
    semantic customer.customer
    table public.customers
    column customer.name -> name storage varchar required
    index bad_idx customer.name
}`
	if _, err := syntax.NewParser(src).ParsePersistence(); err == nil {
		t.Fatal("expected malformed index continuation to fail")
	}
}

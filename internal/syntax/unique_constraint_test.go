package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

func TestParsePersistenceUniqueConstraint(t *testing.T) {
	mapping, err := syntax.NewParser(`persist sales.customer {
        database sales_core
        semantic sales.customer
        table public.customers
        column sales.customer.id -> id storage int64 required
        column sales.customer.tenant_id -> tenant_id storage int64 required
        column sales.customer.external_id -> external_id storage varchar required
        unique customers_identity_key {
            attribute sales.customer.tenant_id
            attribute sales.customer.external_id
        }
    }`).ParsePersistence()
	if err != nil {
		t.Fatal(err)
	}
	if len(mapping.UniqueConstraints) != 1 {
		t.Fatalf("unique constraints=%#v", mapping.UniqueConstraints)
	}
	constraint := mapping.UniqueConstraints[0]
	if constraint.Name != "customers_identity_key" || len(constraint.Attributes) != 2 || constraint.Attributes[0] != "sales.customer.tenant_id" || constraint.Attributes[1] != "sales.customer.external_id" {
		t.Fatalf("constraint=%#v", constraint)
	}
}

func TestParsePersistenceRejectsEmptyUniqueConstraint(t *testing.T) {
	_, err := syntax.NewParser(`persist sales.customer {
        database sales_core
        semantic sales.customer
        table customers
        column sales.customer.id -> id storage int64 required
        unique customers_empty_key { }
    }`).ParsePersistence()
	if err == nil || !strings.Contains(err.Error(), "unique requires at least one attribute") {
		t.Fatalf("error=%v", err)
	}
}

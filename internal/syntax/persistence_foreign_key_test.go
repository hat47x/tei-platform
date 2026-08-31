package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/syntax"
)

func TestParsePersistenceForeignKey(t *testing.T) {
	m, err := syntax.NewParser(`persist sales_core.order {
    database sales_core
    semantic sales.order
    table public.orders
    column sales.order.tenant_id -> tenant_id storage int64 required
    column sales.order.customer_id -> customer_id storage int64 required
    foreign-key orders_customer_fk {
        source sales.order.tenant_id
        source sales.order.customer_id
        references sales_core.customer
        target sales.customer.tenant_id
        target sales.customer.customer_id
    }
}`).ParsePersistence()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.ForeignKeys) != 1 {
		t.Fatalf("foreign keys=%#v", m.ForeignKeys)
	}
	fk := m.ForeignKeys[0]
	if fk.Name != "orders_customer_fk" || fk.ReferencedMapping != core.MustCanonicalID("sales_core.customer") {
		t.Fatalf("foreign key=%#v", fk)
	}
	if len(fk.Attributes) != 2 || fk.Attributes[0] != "sales.order.tenant_id" || fk.Attributes[1] != "sales.order.customer_id" {
		t.Fatalf("source attributes=%#v", fk.Attributes)
	}
	if len(fk.ReferencedAttributes) != 2 || fk.ReferencedAttributes[0] != "sales.customer.tenant_id" || fk.ReferencedAttributes[1] != "sales.customer.customer_id" {
		t.Fatalf("target attributes=%#v", fk.ReferencedAttributes)
	}
}

func TestParsePersistenceForeignKeyRejectsUnbalancedColumns(t *testing.T) {
	_, err := syntax.NewParser(`persist sales_core.order {
    database sales_core
    semantic sales.order
    table public.orders
    column sales.order.customer_id -> customer_id storage int64 required
    foreign-key orders_customer_fk {
        source sales.order.customer_id
        references sales_core.customer
        target sales.customer.tenant_id
        target sales.customer.customer_id
    }
}`).ParsePersistence()
	if err == nil || !strings.Contains(err.Error(), "source/target attribute count mismatch") {
		t.Fatalf("error=%v", err)
	}
}

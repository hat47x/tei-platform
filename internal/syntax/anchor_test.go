package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/syntax"
)

func TestParseApplicationAnchoredReturnsIdentifierTokenRange(t *testing.T) {
	src := "\n  application order_entry {\n    uses semantic sales\n  }\n"
	got, err := syntax.ParseApplicationAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Value.ID != core.CanonicalID("order_entry") {
		t.Fatalf("application ID=%s", got.Value.ID)
	}
	assertAnchorText(t, src, got.Anchor, "order_entry")
	if got.Anchor.Start.Line != 2 || got.Anchor.Start.Column != 15 {
		t.Fatalf("anchor start=%+v", got.Anchor.Start)
	}
}

func TestParseDatabaseAnchoredKeepsDomainValueSeparateFromAnchor(t *testing.T) {
	src := "database sales_core {\n  kind relational\n  ownership shared\n}\n"
	got, err := syntax.ParseDatabaseAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if got.Value.ID != core.CanonicalID("sales_core") {
		t.Fatalf("database ID=%s", got.Value.ID)
	}
	assertAnchorText(t, src, got.Anchor, "sales_core")
}

func TestParseQueryAnchoredDoesNotTokenizeRawSQLBody(t *testing.T) {
	src := `query order_entry.recent_orders {
    datasource sales
    sql postgres {
        SELECT jsonb_build_object('payload', '{"brace": "}"}')
        FROM public.orders
    }
}
`
	got, err := syntax.ParseQueryAnchored(src, core.CanonicalID("order_entry"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Value.ID != core.CanonicalID("order_entry.recent_orders") {
		t.Fatalf("query ID=%s", got.Value.ID)
	}
	assertAnchorText(t, src, got.Anchor, "order_entry.recent_orders")
	if !strings.Contains(got.Value.Source, "jsonb_build_object") {
		t.Fatalf("SQL source=%q", got.Value.Source)
	}
}

func TestContractQueryDeclarationAnchorFindsQueryNestedInContractWrapper(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    read customer.customer
    query find_by_id {
        input id type customer.customer_id
        result customer.customer zero-or-one
    }
}
`
	anchor, err := syntax.ContractQueryDeclarationAnchor(src, "find_by_id")
	if err != nil {
		t.Fatal(err)
	}
	assertAnchorText(t, src, anchor, "find_by_id")
}

func TestContractQueryDeclarationAnchorSkipsPrecedingQueriesAndLegacyNameOnlyQuery(t *testing.T) {
	src := `contract sales_core.order v4.1 {
    read sales.order
    write sales.order.status
    query find_by_id
    query find_by_customer {
        input customerId type customer.customer_id
        result sales.order many
    }
}
`
	first, err := syntax.ContractQueryDeclarationAnchor(src, "find_by_id")
	if err != nil {
		t.Fatal(err)
	}
	assertAnchorText(t, src, first, "find_by_id")

	second, err := syntax.ContractQueryDeclarationAnchor(src, "find_by_customer")
	if err != nil {
		t.Fatal(err)
	}
	assertAnchorText(t, src, second, "find_by_customer")
}

func TestContractQueryDeclarationAnchorRejectsUnknownQueryName(t *testing.T) {
	src := `contract master_data.customer v2.4 {
    query find_by_id {
        input id type customer.customer_id
        result customer.customer zero-or-one
    }
}
`
	_, err := syntax.ContractQueryDeclarationAnchor(src, "does_not_exist")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeclarationIdentifierAnchorRejectsDifferentSubject(t *testing.T) {
	_, err := syntax.DeclarationIdentifierAnchor("application order_entry {}", "application", "other")
	if err == nil || !strings.Contains(err.Error(), "does not match expected subject") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertAnchorText(t *testing.T, src string, anchor syntax.Anchor, want string) {
	t.Helper()
	if anchor.Start.Offset < 0 || anchor.End.Offset > len(src) || anchor.End.Offset < anchor.Start.Offset {
		t.Fatalf("invalid anchor=%+v for source length %d", anchor, len(src))
	}
	if got := src[anchor.Start.Offset:anchor.End.Offset]; got != want {
		t.Fatalf("anchor text=%q want=%q anchor=%+v", got, want, anchor)
	}
}

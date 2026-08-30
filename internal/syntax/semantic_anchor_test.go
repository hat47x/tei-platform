package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticFileAnchoredReturnsMemberTokenRanges(t *testing.T) {
	src := `namespace sales

type OrderId from std.identifier
type OrderStatus from std.code

data Order {
    attribute orderId type sales.order_id
    attribute customerId type customer.customer_id
    relation customer to customer.customer one
}

capability sales.order.submit {
    input order sales.order
    output sales.order
}
`
	got, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Value.ID) != "sales" {
		t.Fatalf("namespace=%s", got.Value.ID)
	}

	want := map[string]string{
		"semantic:sales":                         "sales",
		"semantic-type:sales.order_id":           "OrderId",
		"semantic-type:sales.order_status":       "OrderStatus",
		"semantic-data:sales.order":              "Order",
		"semantic-attribute:sales.order.order_id": "orderId",
		"semantic-attribute:sales.order.customer_id": "customerId",
		"semantic-relation:sales.order.customer": "customer",
		"semantic-capability:sales.order.submit": "sales.order.submit",
	}
	seen := map[string]bool{}
	for _, item := range got.Anchors {
		key := item.Kind + ":" + item.ID
		text, ok := want[key]
		if !ok {
			t.Fatalf("unexpected anchor subject %s", key)
		}
		if item.Anchor.Start.Offset >= item.Anchor.End.Offset {
			t.Fatalf("anchor %s is not a range: %+v", key, item.Anchor)
		}
		if gotText := src[item.Anchor.Start.Offset:item.Anchor.End.Offset]; gotText != text {
			t.Fatalf("anchor %s text=%q want=%q range=%+v", key, gotText, text, item.Anchor)
		}
		seen[key] = true
	}
	for key := range want {
		if !seen[key] {
			t.Fatalf("missing anchor %s", key)
		}
	}
}

func TestSemanticAnchorsFollowCanonicalSymbolNormalization(t *testing.T) {
	src := `namespace sales

data OrderLine {
    attribute ProductCode type std.code
    relation ParentOrder to sales.order one
}
`
	got, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"semantic-data:sales.order_line":                    "OrderLine",
		"semantic-attribute:sales.order_line.product_code": "ProductCode",
		"semantic-relation:sales.order_line.parent_order":  "ParentOrder",
	}
	for _, item := range got.Anchors {
		key := item.Kind + ":" + item.ID
		if text, ok := want[key]; ok {
			if gotText := src[item.Anchor.Start.Offset:item.Anchor.End.Offset]; gotText != text {
				t.Fatalf("anchor %s text=%q want=%q", key, gotText, text)
			}
			delete(want, key)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing normalized anchors=%v", want)
	}
}

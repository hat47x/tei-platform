package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A role is an option VALUE in the authored Source, not the attribute
// identifier or its TypeRef. Token locations are UTF-8 byte offsets and must
// survive BOM, CRLF, quoted prose, and surrounding non-ASCII text unchanged.
func TestParseSemanticAttributeRoleAnchorIsExactSourceToken(t *testing.T) {
	const src = "\ufeffnamespace sales\r\n\r\ndata Order {\r\n" +
		"  # 日本語の注記: role is not an option here\r\n" +
		"  attribute orderId type std.identifier description \"role inside quotes\" role business-key\r\n" +
		"  attribute note type std.text display \"role is prose\"\r\n}\r\n"
	parsed, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatalf("parse anchored source: %v", err)
	}
	if len(parsed.Value.Data) != 1 || len(parsed.Value.Data[0].Attributes) != 2 ||
		parsed.Value.Data[0].Attributes[0].Role != "business-key" ||
		parsed.Value.Data[0].Attributes[1].Role != "" {
		t.Fatalf("unexpected Role interpretation: %+v", parsed.Value.Data)
	}

	wantOffset := strings.Index(src, "role business-key") + len("role ")
	if wantOffset < len("role ") {
		t.Fatal("fixed input is missing authored Role clause")
	}
	wantEnd := wantOffset + len("business-key")
	seen := map[string]int{}
	for _, item := range parsed.Anchors {
		if item.ID != "sales.order.order_id" && item.ID != "sales.order.note" {
			continue
		}
		seen[item.Kind+":"+item.ID]++
		if item.Kind != "semantic-attribute-role" {
			continue
		}
		if item.ID != "sales.order.order_id" {
			t.Fatalf("Role anchor falsely derived from unrelated attribute: %+v", item)
		}
		start, end := item.Anchor.Start.Offset, item.Anchor.End.Offset
		if start != wantOffset || end != wantEnd ||
			string(src[start:end]) != "business-key" {
			t.Fatalf("Role anchor != independently expected Source bytes: %+v; [%d,%d)", item, wantOffset, wantEnd)
		}
	}
	if seen["semantic-attribute-role:sales.order.order_id"] != 1 ||
		seen["semantic-attribute-role:sales.order.note"] != 0 ||
		seen["semantic-attribute:sales.order.order_id"] != 1 ||
		seen["semantic-attribute-type:sales.order.order_id"] != 1 {
		t.Fatalf("Role/symbol/TypeRef source anchors lack distinct ownership: %+v", seen)
	}
	// No source mutation, unique-writing authorization or Owner acceptance.
}

func TestParseSemanticAttributeRoleAnchorAbsentDoesNotInventOne(t *testing.T) {
	const src = "namespace probe\ndata item {\n  attribute value type std.text description \"role audit-amount\"\n}\n"
	parsed, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range parsed.Anchors {
		if item.Kind == "semantic-attribute-role" {
			t.Fatalf("quoted role prose became a declared Role: %+v", item)
		}
	}
}

package syntax_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/syntax"
)

func TestParseSemanticAttributeRole(t *testing.T) {
	src := `namespace sales

data Order {
    attribute orderId type std.identifier role business-key
}`
	namespace, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(namespace.Data) != 1 || len(namespace.Data[0].Attributes) != 1 {
		t.Fatalf("namespace=%+v", namespace)
	}
	if role := namespace.Data[0].Attributes[0].Role; role != "business-key" {
		t.Fatalf("role=%s", role)
	}
}

// The authored Source must not silently erase an earlier Role option when
// projecting to semantic.Attribute.Role. A duplicate belongs to syntax
// diagnostics, before any Role contract or Owner meaning acceptance.
func TestParseSemanticAttributeRoleRejectDuplicateOption(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{
			name:   "distinct role values",
			source: "namespace sales\ndata Order {\n  attribute orderId type std.identifier role business-key role audit-amount\n}\n",
		},
		{
			name:   "same role value repeated",
			source: "namespace sales\ndata Order {\n  attribute orderId type std.identifier role business-key role business-key\n}\n",
		},
		{
			name:   "comment and newline between options",
			source: "namespace sales\ndata Order {\n  attribute orderId type std.identifier role business-key\n  # source comment\n  role audit-amount\n}\n",
		},
		{
			name:   "quoted role text is not an option",
			source: "namespace sales\ndata Order {\n  attribute orderId type std.identifier role business-key description \"role is documented here\" role audit-amount\n}\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantOffset := strings.LastIndex(tc.source, "role ")
			if wantOffset < 0 {
				t.Fatal("test input has no role option")
			}
			_, err := syntax.NewParser(tc.source).ParseSemanticFile()
			var diagnostic *syntax.ParseError
			if !errors.As(err, &diagnostic) {
				t.Fatalf("expected source-positioned duplicate Role parse error, got %v", err)
			}
			if diagnostic.Pos.Offset != wantOffset ||
				!strings.Contains(diagnostic.Msg, "duplicate role option") {
				t.Fatalf("incorrect second Role diagnostic: %+v; expected offset %d", diagnostic, wantOffset)
			}
			if _, err := syntax.ParseSemanticFileAnchored(tc.source); err == nil {
				t.Fatal("anchored semantic source path silently accepted duplicate Role")
			}
		})
	}
}

func TestParseSemanticAttributeRoleAllowsDistinctAttributesAndQuotedText(t *testing.T) {
	src := "namespace sales\ndata Order {\n" +
		"  attribute orderId type std.identifier description \"role ignored inside quoted text\" role business-key\n" +
		"  attribute note type std.text role audit-amount\n}\n"
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Data) != 1 || len(ns.Data[0].Attributes) != 2 {
		t.Fatalf("unexpected parsed attributes: %+v", ns.Data)
	}
	got := ns.Data[0].Attributes
	if got[0].Role != "business-key" || got[0].Description != "role ignored inside quoted text" ||
		got[1].Role != "audit-amount" {
		t.Fatalf("independent Role options or quoted description changed: %+v", got)
	}
	if _, err := syntax.ParseSemanticFileAnchored(src); err != nil {
		t.Fatalf("valid semantic source anchor parsing failed: %v", err)
	}
}


// Role's source-side value anchor allows navigation, not editing or acceptance.
// Check the exact authored bytes; quoted role-like text and multibyte content
// must not alter this anchor or become an additional Role declaration.
func TestSemanticAttributeRoleSourceAnchorOffsetsAndQuotedLiteral(t *testing.T) {
	src := "namespace sales\r\ndata Order {\r\n" +
		"  attribute total type std.money description \"role 日本語の説明\" role accounting_amount\r\n}\r\n"
	parsed, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Value.Data) != 1 || len(parsed.Value.Data[0].Attributes) != 1 ||
		parsed.Value.Data[0].Attributes[0].Role != "accounting_amount" {
		t.Fatalf("role candidate was not preserved: %+v", parsed.Value.Data)
	}
	wantStart := strings.LastIndex(src, "accounting_amount")
	if wantStart < 0 {
		t.Fatal("source does not contain role value")
	}
	roleCount := 0
	for _, item := range parsed.Anchors {
		if item.Kind != "semantic-attribute-role" {
			continue
		}
		roleCount++
		if item.ID != "sales.order.total" ||
			item.Anchor.Start.Offset != wantStart ||
			item.Anchor.End.Offset != wantStart+len("accounting_amount") {
			t.Fatalf("role value anchor has wrong source identity/range: %+v", item)
		}
		if src[item.Anchor.Start.Offset:item.Anchor.End.Offset] != "accounting_amount" {
			t.Fatalf("role anchor does not identify the original value bytes: %+v", item)
		}
	}
	if roleCount != 1 {
		t.Fatalf("quoted role text was misclassified or Role value anchor missing: count=%d", roleCount)
	}
}

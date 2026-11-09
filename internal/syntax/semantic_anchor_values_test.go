package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// TestParseSemanticFileAnchoredSkipsATypeValuesRestriction pins the Source Index
// side of a value-restricted type alias. `type Status from std.code values { "open" "closed" }`
// parses, but the anchor scan stopped at the `values` token and failed the whole
// Index build with `unexpected semantic declaration "values"`. The scan only
// needs subject identity, so it must consume the restriction and continue.
func TestParseSemanticFileAnchoredSkipsATypeValuesRestriction(t *testing.T) {
	src := `namespace sales

type Status from std.code values { "open" "closed" }

data Order {
    attribute orderId type std.identifier
    attribute status type sales.status
}
`
	got, err := syntax.ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatalf("anchor scan rejected a value-restricted type: %v", err)
	}
	var ids []string
	for _, anchor := range got.Anchors {
		ids = append(ids, anchor.Kind+":"+anchor.ID)
	}
	for _, want := range []string{
		"semantic-type:sales.status",
		"semantic-data:sales.order",
		"semantic-attribute:sales.order.order_id",
		"semantic-attribute:sales.order.status",
	} {
		found := false
		for _, id := range ids {
			if id == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("anchor scan omitted %q: %v", want, ids)
		}
	}
}

// A type without a restriction still anchors, so the skip stays optional.
func TestParseSemanticFileAnchoredWithoutAValuesRestriction(t *testing.T) {
	src := `namespace sales

type Code from std.text

data Order {
    attribute orderId type sales.code
}
`
	if _, err := syntax.ParseSemanticFileAnchored(src); err != nil {
		t.Fatalf("anchor scan rejected a plain type: %v", err)
	}
}

// A broken restriction fails the scan rather than being silently skipped, so a
// malformed source does not reach the Index as if it had been read.
func TestParseSemanticFileAnchoredRejectsMalformedValuesRestriction(t *testing.T) {
	src := `namespace sales

type Status from std.code values { "open"
`
	_, err := syntax.ParseSemanticFileAnchored(src)
	if err == nil {
		t.Fatal("a malformed values restriction was accepted")
	}
	if !strings.Contains(err.Error(), "unexpected token") && !strings.Contains(err.Error(), "semantic type values") {
		t.Fatalf("unexpected error: %v", err)
	}
}

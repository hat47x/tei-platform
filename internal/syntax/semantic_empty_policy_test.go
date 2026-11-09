package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

func TestParseCapabilityEmptyPolicy(t *testing.T) {
	src := `namespace sales

capability submitOrder {
    input order_id sales.order_id
    output sales.order_status
    emptyPolicy requires-evidence
}
`
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if len(ns.Capabilities) != 1 {
		t.Fatalf("capabilities=%d", len(ns.Capabilities))
	}
	if got := string(ns.Capabilities[0].EmptyPolicy); got != "requires-evidence" {
		t.Fatalf("emptyPolicy=%q", got)
	}
}

func TestParseCapabilityRejectsUnknownEmptyPolicy(t *testing.T) {
	src := `namespace sales

capability submitOrder {
    emptyPolicy sometimes
}
`
	_, err := syntax.NewParser(src).ParseSemanticFile()
	if err == nil || !strings.Contains(err.Error(), "unsupported emptyPolicy") {
		t.Fatalf("unknown emptyPolicy was accepted: %v", err)
	}
}

func TestParseCapabilityRejectsDuplicateEmptyPolicy(t *testing.T) {
	src := `namespace sales

capability submitOrder {
    emptyPolicy allow
    emptyPolicy deny
}
`
	_, err := syntax.NewParser(src).ParseSemanticFile()
	if err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("duplicate emptyPolicy was accepted: %v", err)
	}
}

func TestParseCapabilityWithoutEmptyPolicyLeavesItUndeclared(t *testing.T) {
	src := `namespace sales

capability submitOrder {
    input order_id sales.order_id
}
`
	ns, err := syntax.NewParser(src).ParseSemanticFile()
	if err != nil {
		t.Fatal(err)
	}
	if ns.Capabilities[0].EmptyPolicy != "" {
		t.Fatalf("emptyPolicy=%q, want an undeclared value", ns.Capabilities[0].EmptyPolicy)
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The trace response has two fields that both read as "unresolved":
// result.trace.unresolved (rows of the selected subject) and
// result.projectionUnresolved (edges dropped while building the whole graph).
// reference/workspace shows 0 and 17 respectively, so a reader who checks only
// the inner counter concludes "nothing unresolved" (#3906). The contract must
// keep saying which one answers the question.
func TestContractDistinguishesTheTwoUnresolvedFields(t *testing.T) {
	path := filepath.Join("..", "..", "plan", "design", "CLI_JSON_OUTPUT_CONTRACT_2026-10.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the CLI JSON contract: %v", err)
	}
	doc := string(raw)

	for _, want := range []struct{ needle, why string }{
		{"result.trace.unresolved", "must name the row-level counter"},
		{"result.projectionUnresolved", "must name the projection-level list"},
		{"投影した辺そのものから規則を作る", "must say why rule issues cannot occur"},
		{"`projectionUnresolved` で判断する", "must tell a reader which field to trust"},
	} {
		if !strings.Contains(doc, want.needle) {
			t.Fatalf("the contract no longer states %s (%q missing)", want.why, want.needle)
		}
	}

	// The note must also record that the rule-dependent issues are unreachable,
	// so nobody re-adds a claim that they can be returned.
	for _, issue := range []string{"`invalid-edge-type`", "`invalid-rule`", "`conflicting-rule`"} {
		if !strings.Contains(doc, issue) {
			t.Fatalf("the contract no longer names %s as a row issue", issue)
		}
	}
}

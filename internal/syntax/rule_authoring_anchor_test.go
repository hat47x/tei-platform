package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// The rule surface must hand a consumer the value and the anchors from one
// parse. A separate anchor scan over the same bytes is the failure mode this
// slice exists to avoid (the existing semantic path re-lexes today, and a new
// grammar that did the same would double the maintenance surface).
func TestRuleAuthoringAnchoredReturnsValueAndAnchorsTogether(t *testing.T) {
	anchored, err := syntax.ParseRuleAuthoringAnchored(ruleAuthoringBoth)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(anchored.Issues) != 0 {
		t.Fatalf("issues=%v", anchored.Issues)
	}
	if len(anchored.Value.Validations) != 2 {
		t.Fatalf("validations=%d", len(anchored.Value.Validations))
	}
	if anchored.Value.ID != "sales" {
		t.Fatalf("namespace=%q", anchored.Value.ID)
	}

	// Anchor kinds reuse the existing Subject vocabulary. A new kind here would
	// need a Source Index consumer that does not exist.
	kinds := map[string]int{}
	for _, anchor := range anchored.Anchors {
		kinds[anchor.Kind]++
		if anchor.ID == "" {
			t.Fatalf("anchor without identity: %#v", anchor)
		}
	}
	if kinds["semantic"] != 1 {
		t.Fatalf("namespace anchors=%d", kinds["semantic"])
	}
	if kinds["semantic-validation"] != 2 {
		t.Fatalf("validation anchors=%d", kinds["semantic-validation"])
	}
	if len(kinds) != 2 {
		t.Fatalf("unexpected anchor kinds: %v", kinds)
	}

	// Every anchor must slice the authored bytes.
	for _, anchor := range anchored.Anchors {
		start, end := anchor.Anchor.Start.Offset, anchor.Anchor.End.Offset
		if start >= end || end > len(ruleAuthoringBoth) {
			t.Fatalf("bad anchor %#v", anchor)
		}
		if got := ruleAuthoringBoth[start:end]; got == "" {
			t.Fatalf("empty slice for %#v", anchor)
		}
	}
}

// The anchor for a rule identity must point at the authored identity token, not
// at the whole declaration: a rename diagnostic and an editor jump both need the
// narrower span.
func TestRuleAnchorPointsAtTheIdentityToken(t *testing.T) {
	anchored, err := syntax.ParseRuleAuthoringAnchored(ruleAuthoringBoth)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, anchor := range anchored.Anchors {
		if anchor.Kind != "semantic-validation" {
			continue
		}
		if got := ruleAuthoringBoth[anchor.Anchor.Start.Offset:anchor.Anchor.End.Offset]; got != anchor.ID {
			t.Fatalf("anchor slice=%q want %q", got, anchor.ID)
		}
	}
}

// A source with a conversion issue still yields anchors for the rules that
// parsed, so a diagnostic can be placed without a successful conversion.
func TestRuleAnchorsSurviveConversionIssues(t *testing.T) {
	src := `namespace sales
rule sales.a: bare is positive
rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive
`
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(anchored.Issues) != 1 {
		t.Fatalf("issues=%v", anchored.Issues)
	}
	if len(anchored.Anchors) != 3 {
		t.Fatalf("anchors=%d want namespace + two rules", len(anchored.Anchors))
	}
	if _, ok := syntax.ErrorMessage(anchored.Issues[0]); ok {
		t.Fatalf("conversion issue should not be a parse error: %v", anchored.Issues[0])
	}
	if !strings.Contains(anchored.Issues[0].Msg, "not a qualified canonical id") {
		t.Fatalf("message=%q", anchored.Issues[0].Msg)
	}
}

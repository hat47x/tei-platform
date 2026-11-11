package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// A `rule` written inside an ordinary semantic file is a placement mistake, not
// an unknown word. Before the rule surface existed, the parser could only say
// "expected type, data, capability, validation, ...". Naming the declaration and
// where it belongs is the difference between "this token is wrong" and "rules
// live in their own file" (#4169 pattern).
func TestRuleDeclarationIsNamedWhenMisplaced(t *testing.T) {
	src := `namespace sales
data Order {
    attribute order_id type std.identifier
}
rule sales.q: sales.order_id is positive
`
	_, err := syntax.NewParser(src).ParseSemanticFile()
	if err == nil {
		t.Fatal("a rule inside a semantic file was accepted by the existing grammar")
	}
	msg, ok := syntax.ErrorMessage(err)
	if !ok {
		t.Fatalf("no message: %v", err)
	}
	if !strings.Contains(msg, `"rule" starts a rule authoring declaration`) ||
		!strings.Contains(msg, "declare it in its own file") {
		t.Fatalf("message=%q", msg)
	}
	// The range must point at the keyword, not at the end of the previous block.
	start, end, ok := syntax.ErrorRange(err)
	if !ok {
		t.Fatalf("no range: %v", err)
	}
	if got := src[start.Offset:end.Offset]; got != "rule" {
		t.Fatalf("range=%q", got)
	}
}

// The existing declarations keep their own messages: adding a word to the
// placement table must not turn a typo into a placement hint.
func TestUnknownSemanticMemberIsStillATypo(t *testing.T) {
	src := "namespace sales\nfrobnicate sales.q\n"
	_, err := syntax.NewParser(src).ParseSemanticFile()
	if err == nil {
		t.Fatal("an unknown declaration was accepted")
	}
	msg, _ := syntax.ErrorMessage(err)
	if strings.Contains(msg, "declare it in its own file") {
		t.Fatalf("a typo was reported as a placement mistake: %q", msg)
	}
	if !strings.Contains(msg, "expected type, data, capability, validation") {
		t.Fatalf("message=%q", msg)
	}
}

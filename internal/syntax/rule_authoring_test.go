package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/syntax"
)

// These cases pin the candidate rule-authoring surface (tei.rule-authoring/v0).
// They verify source-level structure and byte spans only: no canonical
// acceptance, no evaluation, no provider, no execution authority.

const ruleAuthoringBoth = `namespace sales

rule sales.order_line.quantity_positive:
  sales.order_line.quantity is positive

rule sales.order.lines.line_number_unique:
  sales.order_line.line_number is unique per sales.order.lines
`

func TestParseRuleAuthoringFileReadsBothKinds(t *testing.T) {
	file, err := syntax.ParseRuleAuthoringFile(ruleAuthoringBoth)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if file.Namespace != "sales" {
		t.Fatalf("namespace=%q", file.Namespace)
	}
	if len(file.Rules) != 2 {
		t.Fatalf("rules=%d", len(file.Rules))
	}

	positive := file.Rules[0]
	if positive.ID != "sales.order_line.quantity_positive" {
		t.Fatalf("id=%q", positive.ID)
	}
	if positive.Predicate != syntax.RulePredicatePositive {
		t.Fatalf("predicate=%q", positive.Predicate)
	}
	if positive.Subject.Text != "sales.order_line.quantity" {
		t.Fatalf("subject=%q", positive.Subject.Text)
	}
	if positive.Scope != nil {
		t.Fatalf("positive must not carry a scope: %#v", positive.Scope)
	}

	unique := file.Rules[1]
	if unique.ID != "sales.order.lines.line_number_unique" {
		t.Fatalf("id=%q", unique.ID)
	}
	if unique.Predicate != syntax.RulePredicateUnique {
		t.Fatalf("predicate=%q", unique.Predicate)
	}
	if unique.Subject.Text != "sales.order_line.line_number" {
		t.Fatalf("subject=%q", unique.Subject.Text)
	}
	if unique.Scope == nil || unique.Scope.Text != "sales.order.lines" {
		t.Fatalf("scope=%#v", unique.Scope)
	}
}

// A span must let a caller slice the original bytes. A column that is counted
// in runes, or a span that includes the trailing newline, would produce a
// diagnostic on the wrong text.
func TestRuleAuthoringSpansSliceTheOriginalSource(t *testing.T) {
	file, err := syntax.ParseRuleAuthoringFile(ruleAuthoringBoth)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	check := func(label string, span syntax.Anchor, want string) {
		t.Helper()
		if span.End.Offset > len(ruleAuthoringBoth) || span.Start.Offset >= span.End.Offset {
			t.Fatalf("%s span is not a non-empty byte range: %+v", label, span)
		}
		if got := ruleAuthoringBoth[span.Start.Offset:span.End.Offset]; got != want {
			t.Fatalf("%s slice=%q want %q (%+v)", label, got, want, span)
		}
	}
	check("namespace", file.NamespaceSpan, "sales")
	for _, rule := range file.Rules {
		check("id", rule.IDSpan, string(rule.ID))
		check("subject", rule.SubjectSpan, rule.Subject.Text)
		check("predicate", rule.PredicateSpan, string(rule.Predicate))
	}
	check("declaration", file.Rules[1].DeclarationSpan,
		"rule sales.order.lines.line_number_unique:\n  sales.order_line.line_number is unique per sales.order.lines")
	check("scope", file.Rules[1].ScopeSpan, "sales.order.lines")
}

func TestRuleAuthoringRejectsTheExistingValidationGrammar(t *testing.T) {
	// P-01: the new entry point must not absorb the accepted grammar. The
	// existing parser stays the reader of `validation ... { attribute ... }`.
	src := `namespace sales
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}`
	if _, err := syntax.ParseRuleAuthoringFile(src); err == nil {
		t.Fatal("rule authoring accepted an existing validation declaration")
	}
	if _, err := syntax.NewParser(src).ParseSemanticFile(); err != nil {
		t.Fatalf("existing grammar regressed: %v", err)
	}
}

func TestRuleAuthoringRejections(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"P-02 empty body", "namespace sales\nrule sales.quantity_positive:\n", "expected the constrained attribute"},
		{"P-03 missing predicate", "namespace sales\nrule sales.q: sales.order_line.quantity is\n", "expected a rule predicate"},
		{"P-04 unknown predicate word", "namespace sales\nrule sales.q: sales.order_line.quantity is positive-ish\n", "unknown rule predicate"},
		{"P-05 foreign namespace", "namespace sales\nrule customer.q: sales.order_line.quantity is positive\n", "must belong to the declared namespace"},
		{"P-06 duplicate rule id", "namespace sales\nrule sales.q: a.b is positive\nrule sales.q: c.d is positive\n", "is declared more than once"},
		{"P-07 unique without scope", "namespace sales\nrule sales.q: a.b is unique\n", "requires \"per <relation>\""},
		{"P-08 positive with scope", "namespace sales\nrule sales.q: a.b is positive per a.b\n", "accepts no scope"},
		{"subject is clause word", "namespace sales\nrule sales.q: is positive\n", "cannot be a rule subject"},
		{"missing is", "namespace sales\nrule sales.q: a.b positive\n", "expected \"is\" after the subject reference"},
		{"trailing keyword", "namespace sales\nrule sales.q: a.b is positive\nattribute x\n", "unexpected \"attribute\" after the rule body"},
		{"no declaration", "namespace sales\n", "declares no rule"},
		{"nested namespace", "namespace sales.core\nrule sales.core.q: a.b is positive\n", "must be a top-level canonical id"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := syntax.ParseRuleAuthoringFile(tc.src)
			if err == nil {
				t.Fatalf("accepted: %q", tc.src)
			}
			msg, ok := syntax.ErrorMessage(err)
			if !ok || !strings.Contains(msg, tc.want) {
				t.Fatalf("message=%q want substring %q", msg, tc.want)
			}
		})
	}
}

// Every rejection must point at the authored bytes, not at a re-lexed copy.
func TestRuleAuthoringDiagnosticsCarrySourceRanges(t *testing.T) {
	src := "namespace sales\nrule sales.q: a.b is positive-ish\n"
	_, err := syntax.ParseRuleAuthoringFile(src)
	if err == nil {
		t.Fatal("expected rejection")
	}
	start, end, ok := syntax.ErrorRange(err)
	if !ok {
		t.Fatalf("no range: %T %v", err, err)
	}
	if got := src[start.Offset:end.Offset]; got != "positive-ish" {
		t.Fatalf("range text=%q range=%d:%d", got, start.Offset, end.Offset)
	}
}

// P-09: comments, a leading BOM and CRLF are the existing lexer's concern, and
// the new surface must behave the same way rather than adding its own rule.
func TestRuleAuthoringKeepsLexerBehaviour(t *testing.T) {
	src := "\ufeffnamespace sales\r\n# leading comment\r\nrule sales.q:\r\n  // subject comment\r\n  a.b is positive\r\n"
	file, err := syntax.ParseRuleAuthoringFile(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(file.Rules) != 1 {
		t.Fatalf("rules=%d", len(file.Rules))
	}
	rule := file.Rules[0]
	if got := src[rule.SubjectSpan.Start.Offset:rule.SubjectSpan.End.Offset]; got != "a.b" {
		t.Fatalf("subject slice=%q", got)
	}
	if got := src[rule.PredicateSpan.Start.Offset:rule.PredicateSpan.End.Offset]; got != "positive" {
		t.Fatalf("predicate slice=%q", got)
	}
}

// P-10: a UTF-8 identifier is still rejected by the shared lexer. The rule
// surface must not accept it with a private rule; Unicode identifier policy is
// an Owner decision (#4359), not a side effect of this parser.
func TestRuleAuthoringRejectsUTF8Identifiers(t *testing.T) {
	src := "namespace sales\nrule sales.q: 数量 is positive\n"
	_, err := syntax.ParseRuleAuthoringFile(src)
	if err == nil {
		t.Fatal("accepted a UTF-8 identifier")
	}
	var lexErr *syntax.LexError
	if !asLexError(err, &lexErr) {
		t.Fatalf("expected a lexer rejection, got %T: %v", err, err)
	}
}

func asLexError(err error, target **syntax.LexError) bool {
	for err != nil {
		if e, ok := err.(*syntax.LexError); ok {
			*target = e
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
			continue
		}
		return false
	}
	return false
}

// The predicate words are a closed list that maps onto existing validation
// kinds. This pins the mapping so a later edit cannot quietly rename a word.
func TestRulePredicatesMapToExistingValidationKinds(t *testing.T) {
	for _, predicate := range []syntax.RulePredicate{
		syntax.RulePredicatePositive,
		syntax.RulePredicateUnique,
	} {
		if string(predicate) == "" {
			t.Fatalf("empty predicate word")
		}
	}
	if want := semantic.ValidationPositive; string(want) != string(syntax.RulePredicatePositive) {
		t.Fatalf("positive kind drifted: %q vs %q", want, syntax.RulePredicatePositive)
	}
	if want := semantic.ValidationRelationUnique; !strings.Contains(string(want), string(syntax.RulePredicateUnique)) {
		t.Fatalf("unique kind drifted: %q vs %q", want, syntax.RulePredicateUnique)
	}
}

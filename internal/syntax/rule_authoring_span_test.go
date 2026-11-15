package syntax_test

import (
	"strings"
	"testing"

	"tei/internal/syntax"
)

// S3 cases: every span of the rule surface must slice the authored bytes, not a
// normalised copy. CRLF, tabs, a leading BOM, and non-ASCII comments all move
// the byte offsets without moving the meaning.

func TestRuleSpansUnderCRLFTabsAndBOM(t *testing.T) {
	const src = "\ufeffnamespace sales\r\n\r\n# 受注の数量は正であること\r\nrule sales.order_line.quantity_positive:\r\n\tsales.order_line.quantity is positive\r\n"
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(anchored.Value.Validations) != 1 {
		t.Fatalf("validations=%d", len(anchored.Value.Validations))
	}
	rule := anchored.File.Rules[0]

	// The BOM is not part of the namespace token.
	if got := src[anchored.File.NamespaceSpan.Start.Offset:anchored.File.NamespaceSpan.End.Offset]; got != "sales" {
		t.Fatalf("namespace slice=%q", got)
	}
	// The subject is separated from its clause by a tab; the span covers only
	// the reference.
	if got := src[rule.SubjectSpan.Start.Offset:rule.SubjectSpan.End.Offset]; got != "sales.order_line.quantity" {
		t.Fatalf("subject slice=%q", got)
	}
	if got := src[rule.PredicateSpan.Start.Offset:rule.PredicateSpan.End.Offset]; got != "positive" {
		t.Fatalf("predicate slice=%q", got)
	}
	// Line and column are byte-based, so a CRLF file reports the authored line
	// number rather than a normalised one.
	if rule.SubjectSpan.Start.Line != 5 {
		t.Fatalf("subject line=%d want 5", rule.SubjectSpan.Start.Line)
	}
}

func TestRuleDiagnosticPositionUnderCRLF(t *testing.T) {
	const src = "namespace sales\r\nrule sales.q:\r\n  a.b is nonsense\r\n"
	_, err := syntax.ParseRuleAuthoringFile(src)
	if err == nil {
		t.Fatal("expected rejection")
	}
	start, end, ok := syntax.ErrorRange(err)
	if !ok {
		t.Fatalf("no range: %T %v", err, err)
	}
	if got := src[start.Offset:end.Offset]; got != "nonsense" {
		t.Fatalf("range text=%q", got)
	}
	if start.Line != 3 {
		t.Fatalf("line=%d want 3", start.Line)
	}
	msg, _ := syntax.ErrorMessage(err)
	if !strings.Contains(msg, "nonsense") {
		t.Fatalf("message=%q", msg)
	}
}

// A source with Windows and Unix line endings must produce the same meaning.
func TestRuleMeaningIsIndependentOfLineEndings(t *testing.T) {
	lf := "namespace sales\nrule sales.q:\n  a.b is positive\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")
	for _, src := range []string{lf, crlf} {
		anchored, err := syntax.ParseRuleAuthoringAnchored(src)
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		if len(anchored.Value.Validations) != 1 {
			t.Fatalf("validations=%d", len(anchored.Value.Validations))
		}
	}
}

// The parser takes a string and never rewrites it: the caller keeps the
// authored bytes. This is the surface-level half of "no re-formatting on save";
// the writer half belongs to the formatter issue (#4366).
func TestRuleParseHasNoWriteBackChannel(t *testing.T) {
	const src = "namespace sales\nrule sales.q:   a.b   is   positive\n"
	original := src
	anchored, err := syntax.ParseRuleAuthoringAnchored(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if src != original {
		t.Fatal("the parser changed the caller's source")
	}
	// The unusual spacing survives in the spans: the declaration span starts at
	// the authored keyword, not at a tidied position.
	rule := anchored.File.Rules[0]
	if rule.DeclarationSpan.Start.Offset != strings.Index(src, "rule") {
		t.Fatalf("declaration start=%d want %d", rule.DeclarationSpan.Start.Offset, strings.Index(src, "rule"))
	}
}

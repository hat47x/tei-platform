package syntax

import (
	"fmt"
	"strings"

	"tei/internal/core"
)

// This file adds the rule authoring surface (tei.rule-authoring/v0).
//
// It is a *candidate* parser: it returns source-level structure with exact
// byte spans and never decides canonical meaning, acceptance, or execution
// authority. The existing `validation <id> <kind> { attribute <id> }` grammar
// in parser.go is untouched, so an accepted Source keeps its meaning.
//
// The surface follows the order a domain expert reads in:
//
//	rule sales.order_line.quantity_positive:
//	  sales.order_line.quantity is positive
//
//	rule sales.order.lines.line_number_unique:
//	  sales.order_line.line_number is unique per sales.order.lines
//
// `rule <id>:` names the declaration once, the predicate follows the thing it
// constrains, and a scope is stated only when the predicate needs one.

// RulePredicate is a word of the rule surface. A caller resolves it to an
// existing semantic.ValidationKind; this package does not choose that mapping,
// and an unknown word is rejected instead of being approximated.
type RulePredicate string

const (
	// RulePredicatePositive states that a numeric attribute is greater than
	// zero. It accepts no scope.
	RulePredicatePositive RulePredicate = "positive"
	// RulePredicateUnique states that an attribute repeats no value within the
	// declared scope. It requires an explicit scope.
	RulePredicateUnique RulePredicate = "unique"
)

// rulePredicates lists every accepted predicate word in source order. A word
// outside this list is a diagnostic, not a synonym for a nearby one.
var rulePredicates = []RulePredicate{RulePredicatePositive, RulePredicateUnique}

func (p RulePredicate) valid() bool {
	for _, known := range rulePredicates {
		if p == known {
			return true
		}
	}
	return false
}

// RuleReference is one authored reference before semantic resolution: the
// source text exactly as written plus its byte range. It is deliberately not a
// core.CanonicalID, because resolving a surface name is a separate step with
// its own diagnostics.
type RuleReference struct {
	Text   string
	Anchor Anchor
}

// RuleAuthoring is one parsed `rule` declaration. Spans are UTF-8 byte offsets
// into the original source; they survive into diagnostics and Source Index.
type RuleAuthoring struct {
	// Namespace is the namespace the declaring file belongs to.
	Namespace core.CanonicalID
	// ID is the canonical identity derived from the declaration's own name. It
	// is not a resolved reference: the namespace is authored, and the segment
	// is derived by the same rule the existing semantic parser uses.
	ID     core.CanonicalID
	Symbol string
	// Subject is the attribute the rule constrains.
	Subject RuleReference
	// Predicate is the authored predicate word.
	Predicate RulePredicate
	// Scope is set only when the source wrote `per <relation>`. A nil pointer
	// means the source did not state a scope; it is never defaulted.
	Scope *RuleReference

	// Spans of the parts an editor or diagnostic points at.
	DeclarationSpan Anchor
	IDSpan          Anchor
	SubjectSpan     Anchor
	PredicateSpan   Anchor
	ScopeSpan       Anchor
}

// RuleAuthoringFile is one parsed rule authoring source.
type RuleAuthoringFile struct {
	Namespace     core.CanonicalID
	NamespaceSpan Anchor
	Rules         []RuleAuthoring
}

// ParseRuleAuthoringFile parses a source whose only declarations are rule
// declarations under one namespace. Callers that need the existing grammar
// keep using ParseSemanticFile; this entry point is not wired into
// workspace.Load, so no existing Workspace Source changes interpretation.
func ParseRuleAuthoringFile(src string) (RuleAuthoringFile, error) {
	p := NewParser(src)
	out := RuleAuthoringFile{}
	if _, err := p.expectText("namespace"); err != nil {
		return out, err
	}
	nsTok, err := p.expect(TokenIdent)
	if err != nil {
		return out, err
	}
	ns, err := core.ParseCanonicalID(nsTok.Text)
	if err != nil {
		return out, p.errAt(nsTok, err.Error())
	}
	if strings.Contains(string(ns), ".") {
		return out, p.errAt(nsTok, "rule authoring namespace must be a top-level canonical id")
	}
	out.Namespace = ns
	out.NamespaceSpan = tokenAnchor(nsTok)

	seen := make(map[core.CanonicalID]Anchor, 0)
	for {
		t, err := p.peek()
		if err != nil {
			return out, err
		}
		if t.Kind == TokenEOF {
			_, _ = p.next()
			break
		}
		if t.Kind != TokenIdent || t.Text != "rule" {
			if misplaced := p.misplacedDeclaration(t, "in a rule authoring source"); misplaced != nil {
				return out, misplaced
			}
			return out, p.errAt(t, "expected rule")
		}
		rule, err := p.parseRuleAuthoring(ns)
		if err != nil {
			return out, err
		}
		if prior, duplicate := seen[rule.ID]; duplicate {
			return out, p.errAtSpan(rule.IDSpan, fmt.Sprintf(
				"rule %s is declared more than once; the first declaration is at %d:%d",
				rule.ID, prior.Start.Line, prior.Start.Column))
		}
		seen[rule.ID] = rule.IDSpan
		out.Rules = append(out.Rules, rule)
	}
	if len(out.Rules) == 0 {
		return out, &ParseError{Pos: out.NamespaceSpan.End, End: out.NamespaceSpan.End,
			Msg: "rule authoring source declares no rule"}
	}
	return out, nil
}

func (p *Parser) parseRuleAuthoring(namespace core.CanonicalID) (RuleAuthoring, error) {
	keyword, err := p.next() // rule
	if err != nil {
		return RuleAuthoring{}, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return RuleAuthoring{}, err
	}
	id, symbol, err := ruleAuthoringIdentity(namespace, nameTok.Text)
	if err != nil {
		return RuleAuthoring{}, p.errAt(nameTok, err.Error())
	}
	if _, err := p.expect(TokenColon); err != nil {
		at, peekErr := p.peek()
		if peekErr != nil || at.Kind == TokenEOF {
			at = nameTok
		}
		return RuleAuthoring{}, p.errAt(at, "expected \":\" after the rule name")
	}
	subjectTok, err := p.expectSubjectIdent()
	if err != nil {
		return RuleAuthoring{}, err
	}
	if err := rejectRuleKeywordAsSubject(subjectTok); err != nil {
		return RuleAuthoring{}, p.errAt(subjectTok, err.Error())
	}
	if _, err := p.expectText("is"); err != nil {
		at, peekErr := p.peek()
		if peekErr != nil || at.Kind == TokenEOF {
			at = subjectTok
		}
		return RuleAuthoring{}, p.errAt(at, "expected \"is\" after the subject reference")
	}
	predTok, err := p.expectPredicateIdent()
	if err != nil {
		return RuleAuthoring{}, err
	}
	predicate := RulePredicate(predTok.Text)
	if !predicate.valid() {
		return RuleAuthoring{}, p.errAt(predTok, fmt.Sprintf(
			"unknown rule predicate %q; accepted predicates are %s",
			predTok.Text, validRulePredicatesText()))
	}

	rule := RuleAuthoring{
		Namespace:       namespace,
		ID:              id,
		Symbol:          symbol,
		Subject:         RuleReference{Text: subjectTok.Text, Anchor: tokenAnchor(subjectTok)},
		Predicate:       predicate,
		DeclarationSpan: Anchor{Start: keyword.Start, End: predTok.End},
		IDSpan:          tokenAnchor(nameTok),
		SubjectSpan:     tokenAnchor(subjectTok),
		PredicateSpan:   tokenAnchor(predTok),
	}

	t, err := p.peek()
	if err != nil {
		return RuleAuthoring{}, err
	}
	if t.Kind == TokenIdent && t.Text == "per" {
		_, _ = p.next()
		scopeTok, err := p.expect(TokenIdent)
		if err != nil {
			return RuleAuthoring{}, err
		}
		rule.Scope = &RuleReference{Text: scopeTok.Text, Anchor: tokenAnchor(scopeTok)}
		rule.ScopeSpan = tokenAnchor(scopeTok)
		rule.DeclarationSpan.End = scopeTok.End
	} else if predicate == RulePredicateUnique {
		// "unique" without a scope would silently mean "unique everywhere".
		// The existing relation-unique contract requires an authored relation,
		// so the surface refuses the missing scope instead of inferring one.
		return RuleAuthoring{}, p.errAtSpan(rule.PredicateSpan,
			"predicate \"unique\" requires \"per <relation>\"; no scope is inferred")
	}
	if predicate == RulePredicatePositive && rule.Scope != nil {
		return RuleAuthoring{}, p.errAtSpan(rule.ScopeSpan,
			"predicate \"positive\" accepts no scope; declare the constrained attribute instead")
	}

	// Anything left on the same declaration must be named. A stray keyword is
	// reported as itself, so an author does not have to guess which word broke.
	nt, err := p.peek()
	if err != nil {
		return RuleAuthoring{}, err
	}
	switch {
	case nt.Kind == TokenEOF:
	case nt.Kind == TokenIdent && nt.Text == "rule":
	default:
		return RuleAuthoring{}, p.errAt(nt, fmt.Sprintf(
			"unexpected %q after the rule body; each rule ends its own declaration", nt.Text))
	}
	return rule, nil
}

// ruleAuthoringIdentity mirrors the existing semantic parser: a fully
// qualified name must stay inside the declaring namespace, and a bare symbol is
// converted with the same SymbolToSegment rule used for every other declaration.
// No other derivation (transliteration, translation, fuzzy matching) is added.
func ruleAuthoringIdentity(namespace core.CanonicalID, symbol string) (core.CanonicalID, string, error) {
	if strings.Contains(symbol, ".") {
		id, err := core.ParseCanonicalID(symbol)
		if err != nil {
			return "", "", err
		}
		if !strings.HasPrefix(string(id), string(namespace)+".") {
			return "", "", fmt.Errorf("rule canonical id %s must belong to the declared namespace %s", id, namespace)
		}
		parts := strings.Split(string(id), ".")
		return id, parts[len(parts)-1], nil
	}
	seg, err := core.SymbolToSegment(symbol)
	if err != nil {
		return "", "", err
	}
	id, err := core.ParseCanonicalID(string(namespace) + "." + seg)
	if err != nil {
		return "", "", err
	}
	return id, symbol, nil
}

// rejectRuleKeywordAsSubject stops a subject from being a later clause word.
// Without it, `rule a: is positive` reads "is" as the subject and reports the
// missing predicate one token too late.
func rejectRuleKeywordAsSubject(t Token) error {
	switch t.Text {
	case "is", "per", "rule", "namespace":
		return fmt.Errorf("%q cannot be a rule subject; declare the constrained attribute before \"is\"", t.Text)
	}
	return nil
}

func validRulePredicatesText() string {
	words := make([]string, 0, len(rulePredicates))
	for _, predicate := range rulePredicates {
		words = append(words, string(predicate))
	}
	return strings.Join(words, ", ")
}

// expectSubjectIdent reads the attribute reference after ":" and reports the
// missing reference at the author's own position. Source ends one token early
// for a hand-written rule, and "unexpected token \"\"" says nothing about what
// is missing, so the gap is named here.
func (p *Parser) expectSubjectIdent() (Token, error) {
	t, err := p.next()
	if err != nil {
		return Token{}, err
	}
	if t.Kind != TokenIdent {
		at := t
		if t.Kind == TokenEOF {
			at = p.lastConsumed(t)
		}
		return Token{}, p.errAt(at, "expected the constrained attribute before \"is\"")
	}
	return t, nil
}

// expectPredicateIdent reads the predicate word and names the missing word
// rather than the empty token at end of source.
func (p *Parser) expectPredicateIdent() (Token, error) {
	t, err := p.next()
	if err != nil {
		return Token{}, err
	}
	if t.Kind != TokenIdent {
		at := t
		if t.Kind == TokenEOF {
			at = p.lastConsumed(t)
		}
		return Token{}, p.errAt(at, "expected a rule predicate after \"is\"; accepted predicates are "+validRulePredicatesText())
	}
	return t, nil
}

// lastConsumed keeps the end-of-source position usable for a message. It does
// not invent a token: the caller only reads Start and End.
func (p *Parser) lastConsumed(eof Token) Token {
	return Token{Kind: TokenEOF, Start: eof.Start, End: eof.End}
}

// errAtSpan reports at an already captured span. It keeps every diagnostic on
// the exact authored bytes instead of re-lexing to find the position again.
func (p *Parser) errAtSpan(span Anchor, msg string) error {
	return &ParseError{Pos: span.Start, End: span.End, Msg: msg}
}

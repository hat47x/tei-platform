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
	// RulePredicateOperator is not a word an author writes. It marks a rule
	// whose predicate is a qualified reference to a declared operator, such as
	// `is sales.non_negative`. The reference is kept in RuleAuthoring.Operator.
	RulePredicateOperator RulePredicate = "operator"
	// RulePredicateNotAfter is written as the two words `not after`, followed
	// by the attribute the subject is compared with:
	//
	//	rule organization.membership.period_in_order:
	//	  organization.membership.starts_on is not after organization.membership.ends_on if present
	//
	// `if present` states that a record without the second date satisfies
	// the rule. It is never assumed: without it such a record is not decided.
	RulePredicateNotAfter RulePredicate = "not-after"
)

// OperatorInputWord is a word of an operator declaration after `takes`. A
// caller resolves it to an existing semantic.OperatorInput.
type OperatorInputWord string

// OperatorInputNumberWord declares that the operator reads one numeric value.
const OperatorInputNumberWord OperatorInputWord = "number"

// OperatorAuthoring is one parsed `operator` declaration:
//
//	operator sales.non_negative: takes number
//
// It declares a contract only. The surface has no place to write an
// implementation, an effect, or a second input.
type OperatorAuthoring struct {
	Namespace core.CanonicalID
	ID        core.CanonicalID
	Symbol    string
	Input     OperatorInputWord

	DeclarationSpan Anchor
	IDSpan          Anchor
	InputSpan       Anchor
}

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
	// Operator is set only when the predicate is a qualified operator
	// reference. It is the authored text; resolving it is the validator's job.
	Operator *RuleReference
	// Scope is set only when the source wrote `per <relation>`. A nil pointer
	// means the source did not state a scope; it is never defaulted.
	Scope *RuleReference
	// Compared is set only for `not after`: the attribute the subject is
	// compared with.
	Compared *RuleReference
	// IfPresent is true only when the source wrote `if present` after the
	// compared attribute.
	IfPresent bool

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
	Operators     []OperatorAuthoring
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
	seenOperators := make(map[core.CanonicalID]Anchor, 0)
	for {
		t, err := p.peek()
		if err != nil {
			return out, err
		}
		if t.Kind == TokenEOF {
			_, _ = p.next()
			break
		}
		if t.Kind == TokenIdent && t.Text == "operator" {
			operator, err := p.parseOperatorAuthoring(ns)
			if err != nil {
				return out, err
			}
			if prior, duplicate := seenOperators[operator.ID]; duplicate {
				return out, p.errAtSpan(operator.IDSpan, fmt.Sprintf(
					"operator %s is declared more than once; the first declaration is at %d:%d",
					operator.ID, prior.Start.Line, prior.Start.Column))
			}
			seenOperators[operator.ID] = operator.IDSpan
			out.Operators = append(out.Operators, operator)
			continue
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
	if len(out.Rules) == 0 && len(out.Operators) == 0 {
		return out, &ParseError{Pos: out.NamespaceSpan.End, End: out.NamespaceSpan.End,
			Msg: "rule authoring source declares no rule"}
	}
	return out, nil
}

// parseOperatorAuthoring reads `operator <id>: takes <input>`. The input word
// is a closed list: it decides what a rule using the operator reads, so an
// unknown word is refused instead of being treated as "reads nothing".
func (p *Parser) parseOperatorAuthoring(namespace core.CanonicalID) (OperatorAuthoring, error) {
	keyword, err := p.next() // operator
	if err != nil {
		return OperatorAuthoring{}, err
	}
	nameTok, err := p.expect(TokenIdent)
	if err != nil {
		return OperatorAuthoring{}, err
	}
	id, symbol, err := ruleAuthoringIdentity(namespace, nameTok.Text)
	if err != nil {
		return OperatorAuthoring{}, p.errAt(nameTok, err.Error())
	}
	if _, err := p.expect(TokenColon); err != nil {
		return OperatorAuthoring{}, p.errAt(nameTok, "expected \":\" after the operator name")
	}
	takes, err := p.next()
	if err != nil {
		return OperatorAuthoring{}, err
	}
	if takes.Kind != TokenIdent || takes.Text != "takes" {
		at := takes
		if takes.Kind == TokenEOF {
			at = nameTok
		}
		return OperatorAuthoring{}, p.errAt(at, "expected \"takes\" after the operator name; write what the operator reads, for example \"takes number\"")
	}
	inputTok, err := p.next()
	if err != nil {
		return OperatorAuthoring{}, err
	}
	if inputTok.Kind != TokenIdent {
		at := inputTok
		if inputTok.Kind == TokenEOF {
			at = takes
		}
		return OperatorAuthoring{}, p.errAt(at, "expected what the operator takes; accepted inputs are number")
	}
	if OperatorInputWord(inputTok.Text) != OperatorInputNumberWord {
		return OperatorAuthoring{}, p.errAt(inputTok, fmt.Sprintf(
			"unknown operator input %q; accepted inputs are number", inputTok.Text))
	}
	nt, err := p.peek()
	if err != nil {
		return OperatorAuthoring{}, err
	}
	if !ruleAuthoringDeclarationEnds(nt) {
		return OperatorAuthoring{}, p.errAt(nt, fmt.Sprintf(
			"unexpected %q after the operator declaration; an operator takes exactly one input", nt.Text))
	}
	return OperatorAuthoring{
		Namespace:       namespace,
		ID:              id,
		Symbol:          symbol,
		Input:           OperatorInputWord(inputTok.Text),
		DeclarationSpan: Anchor{Start: keyword.Start, End: inputTok.End},
		IDSpan:          tokenAnchor(nameTok),
		InputSpan:       tokenAnchor(inputTok),
	}, nil
}

// ruleAuthoringDeclarationEnds reports whether t may follow a complete
// declaration: the end of the source or the keyword of the next declaration.
func ruleAuthoringDeclarationEnds(t Token) bool {
	return t.Kind == TokenEOF || t.Kind == TokenIdent && (t.Text == "rule" || t.Text == "operator")
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
	var operator *RuleReference
	var compared *RuleReference
	ifPresent := false
	lastTok := predTok
	if predTok.Text == "not" {
		// `not after <attribute>` and, optionally, `if present`.
		afterTok, err := p.next()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if afterTok.Kind != TokenIdent || afterTok.Text != "after" {
			at := afterTok
			if afterTok.Kind == TokenEOF {
				at = predTok
			}
			return RuleAuthoring{}, p.errAt(at, "expected \"after\" after \"not\"; the predicate is written \"not after <attribute>\"")
		}
		comparedTok, err := p.next()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if comparedTok.Kind != TokenIdent || comparedTok.Text == "if" || ruleAuthoringDeclarationEnds(comparedTok) {
			at := comparedTok
			if comparedTok.Kind != TokenIdent {
				at = afterTok
			}
			return RuleAuthoring{}, p.errAt(at, "expected the attribute to compare with after \"not after\"")
		}
		predicate = RulePredicateNotAfter
		compared = &RuleReference{Text: comparedTok.Text, Anchor: tokenAnchor(comparedTok)}
		lastTok = comparedTok
		ifTok, err := p.peek()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if ifTok.Kind == TokenIdent && ifTok.Text == "if" {
			_, _ = p.next()
			presentTok, err := p.next()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if presentTok.Kind != TokenIdent || presentTok.Text != "present" {
				at := presentTok
				if presentTok.Kind == TokenEOF {
					at = ifTok
				}
				return RuleAuthoring{}, p.errAt(at, "expected \"present\" after \"if\"; the only condition is \"if present\"")
			}
			ifPresent = true
			lastTok = presentTok
		}
	} else if strings.Contains(predTok.Text, ".") {
		// A qualified name after `is` refers to a declared operator. Whether
		// it is declared is not decided here: like an attribute reference, it
		// is resolved against the whole Workspace by the validator.
		predicate = RulePredicateOperator
		operator = &RuleReference{Text: predTok.Text, Anchor: tokenAnchor(predTok)}
	} else if !predicate.valid() || predicate == RulePredicateOperator {
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
		Operator:        operator,
		Compared:        compared,
		IfPresent:       ifPresent,
		DeclarationSpan: Anchor{Start: keyword.Start, End: lastTok.End},
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
	if predicate == RulePredicateNotAfter && rule.Scope != nil {
		return RuleAuthoring{}, p.errAtSpan(rule.ScopeSpan,
			"predicate \"not after\" compares two dates of one record and accepts no scope")
	}
	if predicate == RulePredicateOperator && rule.Scope != nil {
		return RuleAuthoring{}, p.errAtSpan(rule.ScopeSpan,
			"a declared operator takes one value and accepts no scope")
	}

	// Anything left on the same declaration must be named. A stray keyword is
	// reported as itself, so an author does not have to guess which word broke.
	nt, err := p.peek()
	if err != nil {
		return RuleAuthoring{}, err
	}
	switch {
	case ruleAuthoringDeclarationEnds(nt):
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
	case "is", "per", "rule", "namespace", "operator", "takes", "not", "after":
		return fmt.Errorf("%q cannot be a rule subject; declare the constrained attribute before \"is\"", t.Text)
	}
	// A predicate word in the subject position means the clauses were written
	// in the other order (`positive is a.b`). Since a qualified name after
	// `is` is an operator reference, accepting it here would turn a reordered
	// rule into a rule about an operator named like the attribute.
	if RulePredicate(t.Text).valid() {
		return fmt.Errorf("%q is a predicate, not an attribute; write the constrained attribute before \"is\" and the predicate after it", t.Text)
	}
	return nil
}

func validRulePredicatesText() string {
	words := make([]string, 0, len(rulePredicates))
	for _, predicate := range rulePredicates {
		words = append(words, string(predicate))
	}
	words = append(words, "not after <attribute>")
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

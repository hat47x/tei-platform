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
	// RulePredicateNotOverlapping is written as the two words `not
	// overlapping`. Its subject is a period, written `<start> until <end>`:
	//
	//	rule organization.department.members.periods_apart:
	//	  organization.membership.starts_on until organization.membership.ends_on or open
	//	    is not overlapping per organization.department.members
	//	    for each organization.membership.person_id
	//
	// `or open` states that a row without an end is a period with no last
	// day. `for each <attribute>` compares only rows with the same value.
	RulePredicateNotOverlapping RulePredicate = "not-overlapping"
	// RulePredicateRequiredIf is written as `required if <attribute>`:
	//
	//	rule sales.order_line.discount_reason_required:
	//	  sales.order_line.discount_reason is required if sales.order_line.discount
	//
	// It states that the subject holds a value whenever the named attribute
	// holds one. It constrains presence, not the value: whether the subject
	// is required in every record is the attribute definition's business.
	RulePredicateRequiredIf RulePredicate = "required-if"
)

// OperatorInputWord is a word of an operator declaration after `takes`. A
// caller resolves it to an existing semantic.OperatorInput.
type OperatorInputWord string

// OperatorInputNumberWord declares that the operator reads one numeric value.
const OperatorInputNumberWord OperatorInputWord = "number"

// OperatorInputDateWord declares that the operator reads one calendar day,
// written YYYY-MM-DD.
const OperatorInputDateWord OperatorInputWord = "date"

// OperatorInputTextWord declares that the operator reads one string. It is
// written with quotes in an example table so it is never read as a number
// (section 8.3).
const OperatorInputTextWord OperatorInputWord = "text"

// OperatorInputTruthWord declares that the operator reads one truth value,
// written `true` or `false`.
const OperatorInputTruthWord OperatorInputWord = "truth"

// OperatorInputAnyWord declares that the operator reads a value of any of the
// kinds above. Which kind was read travels with the value, so the text "100"
// and the number 100 stay two values.
const OperatorInputAnyWord OperatorInputWord = "any"

// OperatorInputPresenceWord declares that the operator reads whether an
// attribute holds a value, not the value itself.
const OperatorInputPresenceWord OperatorInputWord = "presence"

// OperatorInput is one parsed `takes` line of an operator declaration.
type OperatorInput struct {
	Word          OperatorInputWord
	Name          string
	NameSpan      Anchor
	OfEachRow     bool
	OfEachRowSpan Anchor
	AbsentAs      string
	AbsentAsSpan  Anchor
	Span          Anchor
}

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
	Inputs    []OperatorInput
	Input     OperatorInputWord
	// InputName is the name written before the type, as in
	// `takes low: number`. It is empty for the single-input form
	// `takes number`, which names its one input "value" when it is read.
	InputName string
	// AbsentAs is the name a value-absent input is read as, written
	// `absent as <name>`. Empty means the declaration states no mapping, and
	// then a rule cannot ask for one.
	AbsentAs     string
	AbsentAsSpan Anchor
	// OfEachRow is true when the input reads every row of the relation,
	// written `of each row`. It decides how the rule is asked: once per
	// scope key rather than once per row.
	OfEachRow     bool
	OfEachRowSpan Anchor
	// RefutesOnSomeRows is true when the declaration states that a witnessed
	// counterexample among the rows read is enough to say the rule does not
	// hold, written `refutes on some rows`.
	RefutesOnSomeRows     bool
	RefutesOnSomeRowsSpan Anchor

	DeclarationSpan Anchor
	IDSpan          Anchor
	InputSpan       Anchor
	InputNameSpan   Anchor
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

// RuleInputAssignment is one `name: value` written after an operator
// reference. The value is either a complete attribute id or a constant; which
// one it is stays the reader's decision, so both are kept as authored text.
type RuleInputAssignment struct {
	Name      string
	Value     string
	NameSpan  Anchor
	ValueSpan Anchor
	// AbsentAs is the mapping name written after `or`, asking the operator to
	// read an absent value this way. Empty means the rule stated no mapping.
	AbsentAs     string
	AbsentAsSpan Anchor
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
	// Inputs are the named assignments written after the operator reference,
	// as in `is sales.within low: 0 high: 100`. The subject is the operator's
	// first input and is not repeated here. Empty when none were written.
	Inputs []RuleInputAssignment
	// Scope is set only when the source wrote `per <relation>`. A nil pointer
	// means the source did not state a scope; it is never defaulted.
	Scope *RuleReference
	// Compared is set only for `not after`: the attribute the subject is
	// compared with.
	Compared *RuleReference
	// IfPresent is true only when the source wrote `if present` after the
	// compared attribute.
	IfPresent bool
	// PeriodEnd is set only when the subject is a period: the attribute
	// written after `until`.
	PeriodEnd *RuleReference
	// OpenEnded is true only when the source wrote `or open` after the end
	// of the period.
	OpenEnded bool
	// GroupBy is set only when the source wrote `for each <attribute>`.
	GroupBy *RuleReference

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

// parseOperatorAuthoring reads `operator <id>:` followed by one or more `takes
// <input>` lines. Each input may carry a name, a type, `of each row`, and
// `absent as <name>`. After the inputs, `refutes on some rows` may appear.
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
	var inputs []OperatorInput
	var refutes bool
	var refutesSpan Anchor
	for {
		nt, err := p.peek()
		if err != nil {
			return OperatorAuthoring{}, err
		}
		if nt.Kind == TokenIdent && nt.Text == "takes" {
			input, err := p.parseOperatorInput()
			if err != nil {
				return OperatorAuthoring{}, err
			}
			inputs = append(inputs, input)
		} else if nt.Kind == TokenIdent && nt.Text == "refutes" {
			if _, err := p.next(); err != nil {
				return OperatorAuthoring{}, err
			}
			onTok, err := p.next()
			if err != nil {
				return OperatorAuthoring{}, err
			}
			if onTok.Kind != TokenIdent || onTok.Text != "on" {
				at := onTok
				if onTok.Kind == TokenEOF {
					at = nt
				}
				return OperatorAuthoring{}, p.errAt(at, "expected \"on\" after \"refutes\"; write \"refutes on some rows\"")
			}
			someTok, err := p.next()
			if err != nil {
				return OperatorAuthoring{}, err
			}
			if someTok.Kind != TokenIdent || someTok.Text != "some" {
				at := someTok
				if someTok.Kind == TokenEOF {
					at = onTok
				}
				return OperatorAuthoring{}, p.errAt(at, "expected \"some\" after \"refutes on\"; write \"refutes on some rows\"")
			}
			rowsTok, err := p.next()
			if err != nil {
				return OperatorAuthoring{}, err
			}
			if rowsTok.Kind != TokenIdent || rowsTok.Text != "rows" {
				at := rowsTok
				if rowsTok.Kind == TokenEOF {
					at = someTok
				}
				return OperatorAuthoring{}, p.errAt(at, "expected \"rows\" after \"refutes on some\"; write \"refutes on some rows\"")
			}
			refutes = true
			refutesSpan = Anchor{Start: nt.Start, End: rowsTok.End}
		} else if ruleAuthoringDeclarationEnds(nt) {
			break
		} else {
			return OperatorAuthoring{}, p.errAt(nt, fmt.Sprintf(
				"unexpected %q after the operator name; write \"takes <name>: <type>\" for each input",
				nt.Text))
		}
	}
	if len(inputs) == 0 {
		return OperatorAuthoring{}, p.errAtSpan(tokenAnchor(nameTok), "expected at least one \"takes\" input after the operator name")
	}
	// An unnamed input is addressed as "value". Two unnamed inputs would both
	// answer to that name, so the second could never be assigned and the
	// declaration would look usable while being unreachable.
	if len(inputs) > 1 {
		for i, input := range inputs {
			if input.Name == "" {
				return OperatorAuthoring{}, p.errAtSpan(input.Span, fmt.Sprintf(
					"input %d of operator %s has no name; an operator with more than one input must name them all, because an unnamed input is addressed as \"value\"",
					i+1, id))
			}
		}
	}
	first := inputs[0]
	// The declaration covers every authored line, including the last input's
	// modifiers: otherwise an edit on the second `takes` line would be reported
	// outside the declaration it belongs to.
	last := inputs[len(inputs)-1]
	declarationEnd := last.Span.End
	if last.OfEachRow {
		declarationEnd = last.OfEachRowSpan.End
	}
	if last.AbsentAs != "" {
		declarationEnd = last.AbsentAsSpan.End
	}
	if refutes {
		declarationEnd = refutesSpan.End
	}
	return OperatorAuthoring{
		Namespace:             namespace,
		ID:                    id,
		Symbol:                symbol,
		Inputs:                inputs,
		Input:                 first.Word,
		InputName:             first.Name,
		AbsentAs:              first.AbsentAs,
		DeclarationSpan:       Anchor{Start: keyword.Start, End: declarationEnd},
		IDSpan:                tokenAnchor(nameTok),
		InputSpan:             first.Span,
		InputNameSpan:         first.NameSpan,
		AbsentAsSpan:          first.AbsentAsSpan,
		OfEachRow:             first.OfEachRow,
		OfEachRowSpan:         first.OfEachRowSpan,
		RefutesOnSomeRows:     refutes,
		RefutesOnSomeRowsSpan: refutesSpan,
	}, nil
}

// parseOperatorInput reads one `takes <input>` line: the input type, an
// optional name, optional `of each row`, and optional `absent as <name>`.
func (p *Parser) parseOperatorInput() (OperatorInput, error) {
	takes, err := p.next() // "takes"
	if err != nil {
		return OperatorInput{}, err
	}
	inputTok, err := p.next()
	if err != nil {
		return OperatorInput{}, err
	}
	if inputTok.Kind != TokenIdent {
		at := inputTok
		if inputTok.Kind == TokenEOF {
			at = takes
		}
		return OperatorInput{}, p.errAt(at, "expected what the operator takes; accepted inputs are number, date, text, truth, any, presence")
	}
	// `takes low: number` names the input; `takes number` does not.
	var name string
	var nameSpan Anchor
	nt, err := p.peek()
	if err != nil {
		return OperatorInput{}, err
	}
	if nt.Kind == TokenColon {
		name = inputTok.Text
		nameSpan = tokenAnchor(inputTok)
		if _, err := core.ParseLocalName(name); err != nil {
			return OperatorInput{}, p.errAt(inputTok, fmt.Sprintf(
				"%q is not a usable input name: %v", name, err))
		}
		if _, err := p.next(); err != nil { // the colon
			return OperatorInput{}, err
		}
		typeTok, err := p.next()
		if err != nil {
			return OperatorInput{}, err
		}
		if typeTok.Kind != TokenIdent {
			at := typeTok
			if typeTok.Kind == TokenEOF {
				at = inputTok
			}
			// Only the words a reader exists for are offered. Listing a
			// word this surface then refuses would send an author down a
			// path that ends in a second error.
			return OperatorInput{}, p.errAt(at, fmt.Sprintf(
				"expected the type of input %q after \":\"; accepted types are %s", name, acceptedOperatorInputWordsList()))
		}
		inputTok = typeTok
		nt, err = p.peek()
		if err != nil {
			return OperatorInput{}, err
		}
	}
	if !knownOperatorInputWord(inputTok.Text) {
		return OperatorInput{}, p.errAt(inputTok, fmt.Sprintf(
			"unknown operator input %q; accepted inputs are %s", inputTok.Text, acceptedOperatorInputWordsList()))
	}
	// `of each row` makes the input read every row of the relation.
	var ofEachRow bool
	var ofEachRowSpan Anchor
	if nt.Kind == TokenIdent && nt.Text == "of" {
		if _, err := p.next(); err != nil {
			return OperatorInput{}, err
		}
		eachTok, err := p.next()
		if err != nil {
			return OperatorInput{}, err
		}
		if eachTok.Kind != TokenIdent || eachTok.Text != "each" {
			at := eachTok
			if eachTok.Kind == TokenEOF {
				at = nt
			}
			return OperatorInput{}, p.errAt(at, "expected \"each\" after \"of\"; the range is written \"of each row\"")
		}
		rowTok, err := p.next()
		if err != nil {
			return OperatorInput{}, err
		}
		if rowTok.Kind != TokenIdent || rowTok.Text != "row" {
			at := rowTok
			if rowTok.Kind == TokenEOF {
				at = eachTok
			}
			return OperatorInput{}, p.errAt(at, "expected \"row\" after \"of each\"; the range is written \"of each row\"")
		}
		ofEachRow = true
		ofEachRowSpan = Anchor{Start: nt.Start, End: rowTok.End}
		nt, err = p.peek()
		if err != nil {
			return OperatorInput{}, err
		}
	}
	// `absent as <name>` declares how the input reads when it holds no value.
	var absentAs string
	var absentAsSpan Anchor
	if nt.Kind == TokenIdent && nt.Text == "absent" {
		if _, err := p.next(); err != nil {
			return OperatorInput{}, err
		}
		asTok, err := p.next()
		if err != nil {
			return OperatorInput{}, err
		}
		if asTok.Kind != TokenIdent || asTok.Text != "as" {
			at := asTok
			if asTok.Kind == TokenEOF {
				at = nt
			}
			return OperatorInput{}, p.errAt(at, "expected \"as\" after \"absent\"; the mapping is written \"absent as <name>\"")
		}
		nameTok, err := p.next()
		if err != nil {
			return OperatorInput{}, err
		}
		if nameTok.Kind != TokenIdent || ruleAuthoringDeclarationEnds(nameTok) {
			at := nameTok
			if nameTok.Kind != TokenIdent {
				at = asTok
			}
			return OperatorInput{}, p.errAt(at, "expected the name the absent value is read as after \"absent as\"")
		}
		if _, err := core.ParseLocalName(nameTok.Text); err != nil {
			return OperatorInput{}, p.errAt(nameTok, fmt.Sprintf(
				"%q is not a usable mapping name: %v", nameTok.Text, err))
		}
		absentAs = nameTok.Text
		absentAsSpan = tokenAnchor(nameTok)
		nt, err = p.peek()
		if err != nil {
			return OperatorInput{}, err
		}
	}
	return OperatorInput{
		Word:          OperatorInputWord(inputTok.Text),
		Name:          name,
		NameSpan:      nameSpan,
		OfEachRow:     ofEachRow,
		OfEachRowSpan: ofEachRowSpan,
		AbsentAs:      absentAs,
		AbsentAsSpan:  absentAsSpan,
		Span:          tokenAnchor(inputTok),
	}, nil
}

// acceptedOperatorInputWords is the closed list of type words this surface
// reads. The words with no reader are not in it.
//
// It is one list, not two: the hint that tells an author what to write and the
// check that decides what is accepted both read it, so a word cannot be offered
// and then refused, and a word cannot be accepted and then omitted from the
// hint.
var acceptedOperatorInputWords = []OperatorInputWord{
	OperatorInputNumberWord,
	OperatorInputDateWord,
	OperatorInputTextWord,
	OperatorInputTruthWord,
	OperatorInputAnyWord,
	OperatorInputPresenceWord,
	OperatorInputDateWord,
	OperatorInputTextWord,
	OperatorInputTruthWord,
	OperatorInputAnyWord,
}

// acceptedOperatorInputWordsList is the hint's rendering of that list.
func acceptedOperatorInputWordsList() string {
	words := make([]string, 0, len(acceptedOperatorInputWords))
	for _, word := range acceptedOperatorInputWords {
		words = append(words, string(word))
	}
	return strings.Join(words, ", ")
}

// knownOperatorInputWord reports whether text is an accepted input type word.
func knownOperatorInputWord(text string) bool {
	for _, word := range acceptedOperatorInputWords {
		if string(word) == text {
			return true
		}
	}
	return false
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
	// A subject may be a period: `<start> until <end>`, optionally `or open`.
	var periodEnd *RuleReference
	openEnded := false
	untilTok, err := p.peek()
	if err != nil {
		return RuleAuthoring{}, err
	}
	if untilTok.Kind == TokenIdent && untilTok.Text == "until" {
		_, _ = p.next()
		endTok, err := p.next()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if endTok.Kind != TokenIdent || endTok.Text == "is" || endTok.Text == "or" || ruleAuthoringDeclarationEnds(endTok) {
			at := endTok
			if endTok.Kind != TokenIdent {
				at = untilTok
			}
			return RuleAuthoring{}, p.errAt(at, "expected the attribute that ends the period after \"until\"")
		}
		periodEnd = &RuleReference{Text: endTok.Text, Anchor: tokenAnchor(endTok)}
		orTok, err := p.peek()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if orTok.Kind == TokenIdent && orTok.Text == "or" {
			_, _ = p.next()
			openTok, err := p.next()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if openTok.Kind != TokenIdent || openTok.Text != "open" {
				at := openTok
				if openTok.Kind == TokenEOF {
					at = orTok
				}
				return RuleAuthoring{}, p.errAt(at, "expected \"open\" after \"or\"; a period without an end is written \"or open\"")
			}
			openEnded = true
		}
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
	var inputs []RuleInputAssignment
	var compared *RuleReference
	ifPresent := false
	lastTok := predTok
	overlapping := false
	if predTok.Text == "not" {
		if next, err := p.peek(); err == nil && next.Kind == TokenIdent && next.Text == "overlapping" {
			_, _ = p.next()
			overlapping = true
			predicate = RulePredicateNotOverlapping
			lastTok = next
		}
	}
	if overlapping {
		// The period, the relation and the grouping are checked below.
	} else if predTok.Text == "not" {
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
			return RuleAuthoring{}, p.errAt(at, "expected \"after\" or \"overlapping\" after \"not\"; the predicates are written \"not after <attribute>\" and \"not overlapping per <relation>\"")
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
		// `low: 0 high: 100` assigns the operator's remaining inputs. The
		// subject is the first input and is not repeated.
		for {
			next, err := p.peek()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if next.Kind != TokenIdent {
				break
			}
			after, err := p.peek2()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if after.Kind != TokenColon {
				break
			}
			nameTok, err := p.next()
			if err != nil {
				return RuleAuthoring{}, err
			}
			// Each input has one binding within a rule. Reject a second one
			// at the authored name rather than depending on conversion order.
			for _, previous := range inputs {
				if previous.Name == nameTok.Text {
					return RuleAuthoring{}, p.errAt(nameTok, fmt.Sprintf(
						"operator input %q is assigned more than once", nameTok.Text))
				}
			}
			if _, err := p.next(); err != nil { // the colon
				return RuleAuthoring{}, err
			}
			valueTok, err := p.next()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if valueTok.Kind != TokenIdent {
				at := valueTok
				if valueTok.Kind == TokenEOF {
					at = nameTok
				}
				return RuleAuthoring{}, p.errAt(at, fmt.Sprintf(
					"expected the value of input %q after \":\"; write an attribute id or a constant", nameTok.Text))
			}
			assignment := RuleInputAssignment{
				Name:      nameTok.Text,
				Value:     valueTok.Text,
				NameSpan:  tokenAnchor(nameTok),
				ValueSpan: tokenAnchor(valueTok),
			}
			// A later assignment extends the declaration even when it has no
			// absent-value mapping of its own.
			lastTok = valueTok
			// `or <name>` asks the operator to read an absent value this way.
			// Whether the operator declared that mapping is checked against the
			// declarations, not here.
			afterValue, err := p.peek()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if afterValue.Kind == TokenIdent && afterValue.Text == "or" {
				if _, err := p.next(); err != nil {
					return RuleAuthoring{}, err
				}
				mappingTok, err := p.next()
				if err != nil {
					return RuleAuthoring{}, err
				}
				if mappingTok.Kind != TokenIdent || ruleAuthoringDeclarationEnds(mappingTok) {
					at := mappingTok
					if mappingTok.Kind != TokenIdent {
						at = afterValue
					}
					return RuleAuthoring{}, p.errAt(at, "expected the mapping name after \"or\"")
				}
				assignment.AbsentAs = mappingTok.Text
				assignment.AbsentAsSpan = tokenAnchor(mappingTok)
				lastTok = mappingTok
			}
			inputs = append(inputs, assignment)
		}
	} else if predTok.Text == "required" {
		// `required if <attribute>`.
		ifTok, err := p.next()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if ifTok.Kind != TokenIdent || ifTok.Text != "if" {
			at := ifTok
			if ifTok.Kind == TokenEOF {
				at = predTok
			}
			return RuleAuthoring{}, p.errAt(at, "expected \"if\" after \"required\"; the condition is written \"required if <attribute>\"")
		}
		triggerTok, err := p.next()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if triggerTok.Kind != TokenIdent || ruleAuthoringDeclarationEnds(triggerTok) {
			at := triggerTok
			if triggerTok.Kind != TokenIdent {
				at = ifTok
			}
			return RuleAuthoring{}, p.errAt(at, "expected the attribute whose value makes the subject required after \"required if\"")
		}
		predicate = RulePredicateRequiredIf
		compared = &RuleReference{Text: triggerTok.Text, Anchor: tokenAnchor(triggerTok)}
		lastTok = triggerTok
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
		Inputs:          inputs,
		Compared:        compared,
		IfPresent:       ifPresent,
		PeriodEnd:       periodEnd,
		OpenEnded:       openEnded,
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
	} else if predicate == RulePredicateNotOverlapping {
		return RuleAuthoring{}, p.errAtSpan(rule.PredicateSpan,
			"predicate \"not overlapping\" requires \"per <relation>\"; the rows to compare are never inferred")
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
	if predicate == RulePredicateNotOverlapping {
		if rule.PeriodEnd == nil {
			return RuleAuthoring{}, p.errAtSpan(rule.SubjectSpan,
				"predicate \"not overlapping\" is about a period; write the subject as \"<start> until <end>\"")
		}
		forTok, err := p.peek()
		if err != nil {
			return RuleAuthoring{}, err
		}
		if forTok.Kind == TokenIdent && forTok.Text == "for" {
			_, _ = p.next()
			eachTok, err := p.next()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if eachTok.Kind != TokenIdent || eachTok.Text != "each" {
				at := eachTok
				if eachTok.Kind == TokenEOF {
					at = forTok
				}
				return RuleAuthoring{}, p.errAt(at, "expected \"each\" after \"for\"; rows are grouped with \"for each <attribute>\"")
			}
			groupTok, err := p.next()
			if err != nil {
				return RuleAuthoring{}, err
			}
			if groupTok.Kind != TokenIdent || ruleAuthoringDeclarationEnds(groupTok) {
				at := groupTok
				if groupTok.Kind != TokenIdent {
					at = eachTok
				}
				return RuleAuthoring{}, p.errAt(at, "expected the attribute to group rows by after \"for each\"")
			}
			rule.GroupBy = &RuleReference{Text: groupTok.Text, Anchor: tokenAnchor(groupTok)}
			rule.DeclarationSpan.End = groupTok.End
		}
	} else if rule.PeriodEnd != nil {
		return RuleAuthoring{}, p.errAtSpan(rule.PeriodEnd.Anchor,
			"a period written with \"until\" is the subject of \"not overlapping\" only")
	}
	if predicate == RulePredicateNotAfter && rule.Scope != nil {
		return RuleAuthoring{}, p.errAtSpan(rule.ScopeSpan,
			"predicate \"not after\" compares two dates of one record and accepts no scope")
	}
	// An operator rule may name a relation. Whether that scope is required or
	// forbidden depends on the operator's declaration, which this parser does
	// not read: an operator with an input over every row is asked of the rows of
	// a relation and must name one, and an operator without such an input is
	// asked of one record and must not. The conversion decides which (decision
	// section 15.4); the parser only reads the word.
	if predicate == RulePredicateRequiredIf && rule.Scope != nil {
		return RuleAuthoring{}, p.errAtSpan(rule.ScopeSpan,
			"predicate \"required if\" is about one record and accepts no scope")
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
	case "is", "per", "rule", "namespace", "operator", "takes", "not", "after", "until", "overlapping", "required", "if", "present":
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
	words = append(words, "not after <attribute>", "not overlapping per <relation>", "required if <attribute>")
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

package syntax

import (
	"tei/internal/core"
)

// A rule whose operator reads every row of a relation is asked of that
// relation's rows, so the rule has to name the relation. The scope is written
// `per <relation>` after the predicate, exactly as `unique` and `not
// overlapping` write theirs. Authoring this was unreachable before: the parser
// refused a scope on any operator predicate, and the conversion required one
// for a row-scoped operator, so the rule could enter the meaning by neither
// side.
//
// The file keeps the two checks that make the pair possible in one place:
//   - the parser accepts a scope on an operator rule only when the same source
//     declares that operator with an input read over all rows;
//   - the conversion requires a scope for such a rule and refuses a scope for
//     any other operator rule.
//
// Both report at the same source position, so an author sees one diagnostic
// about one rule rather than two about the same mistake.

// declaredRowScopedOperator reports whether this source declares operator id
// with an input read over all rows. A declaration found in another document is
// not known here: the conversion resolves those after the whole source is read.
func declaredRowScopedOperator(operators []OperatorAuthoring, id core.CanonicalID) bool {
	for _, operator := range operators {
		if operator.ID != id {
			continue
		}
		if operator.OfEachRow {
			return true
		}
		for _, input := range operator.Inputs {
			if input.OfEachRow {
				return true
			}
		}
	}
	return false
}

// operatorReferenceCanonicalID reads the operator reference of a rule as a
// canonical id, when it is written as one. A reference that is not qualified is
// left to the conversion, which reports it with the rule's own position.
func operatorReferenceCanonicalID(rule RuleAuthoring) (core.CanonicalID, bool) {
	if rule.Operator == nil {
		return "", false
	}
	id, err := core.ParseCanonicalID(rule.Operator.Text)
	if err != nil {
		return "", false
	}
	return id, true
}

// scopeIsForARowScopedOperator reports whether a scope on an operator rule is
// allowed at the point the parser reads it: the declared operator it names
// reads every row. A reference this source cannot resolve keeps the old
// refusal, because accepting it would let an unresolved operator decide how a
// rule is asked.
func scopeIsForARowScopedOperator(operators []OperatorAuthoring, rule RuleAuthoring) bool {
	id, ok := operatorReferenceCanonicalID(rule)
	if !ok {
		return false
	}
	return declaredRowScopedOperator(operators, id)
}

// operatorScopeDiagnosticText is the message an operator rule gets when it
// writes a scope its operator does not support.
const operatorScopeDiagnosticText = "a declared operator takes one value and accepts no scope"

// rowScopedOperatorScopeDiagnosticText is the message a row-scoped operator
// rule gets when it names no relation.
const rowScopedOperatorScopeDiagnosticText = "an operator that reads every row is asked of a relation; write \"per <relation>\""

// rowScopedOperatorScopeMessage reports the scope diagnostic a converted rule
// needs, or "" when the rule's scope fits its operator.
//
// rowScoped is true when the operator the rule names declares an input read
// over all rows: such a rule is asked of the rows of one relation and must
// name it. A rule whose operator reads one record takes no scope.
func rowScopedOperatorScopeMessage(rule RuleAuthoring, rowScoped bool, known bool) string {
	if !known {
		return ""
	}
	switch {
	case rowScoped && rule.Scope == nil:
		return rowScopedOperatorScopeDiagnosticText
	case !rowScoped && rule.Scope != nil:
		return operatorScopeDiagnosticText
	}
	return ""
}

// rowScopedOperatorDiagnosticSpan is the position the scope diagnostic is
// reported at: the scope when there is one, the predicate when one is missing.
func rowScopedOperatorDiagnosticSpan(rule RuleAuthoring) Anchor {
	if rule.Scope != nil {
		return rule.ScopeSpan
	}
	return rule.PredicateSpan
}

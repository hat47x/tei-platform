package validationread_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/driver/validationread"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// presenceEvery asks whether every row holds a value, as the shipped
// tei.presence.every-present does. One row that holds nothing refutes it; a row
// nobody read leaves the question open.
func presenceEvery() validationeval.OperatorImplementation {
	return validationeval.OperatorImplementation{
		Operator: "organization.all_hold",
		Input:    semantic.OperatorInputPresence,
		PresenceRows: func(rows map[string][]validationeval.PresenceReading) (validationeval.Tri, error) {
			unknown := false
			for _, row := range rows["value"] {
				switch row.State {
				case validationeval.ValueNull:
					return validationeval.TriNo, nil
				case validationeval.ValueNotFetched:
					unknown = true
				}
			}
			if unknown {
				return validationeval.TriMaybe, nil
			}
			return validationeval.TriYes, nil
		},
	}
}

func presenceRowsRule(attribute core.CanonicalID) (semantic.Validation, semantic.Operator) {
	contract := semantic.Operator{
		ID:    "organization.all_hold",
		Input: semantic.OperatorInputPresence,
		Inputs: []semantic.OperatorInputDeclaration{
			{Name: "value", Type: semantic.OperatorInputPresence, OfEachRow: true},
		},
	}
	rule := semantic.Validation{
		ID:        core.CanonicalID("organization.membership.every_row_holds_") + attribute,
		Kind:      semantic.ValidationOperator,
		Attribute: attribute,
		Relation:  membersID,
		Operator:  "organization.all_hold",
	}
	return rule, contract
}

func presenceRelationAnswer(t *testing.T, detail *validationread.Detail, attribute core.CanonicalID) string {
	t.Helper()
	rule, contract := presenceRowsRule(attribute)
	decisions, err := detail.Decide(rule, validationeval.Operators{
		Declared:        []semantic.Operator{contract},
		Implementations: []validationeval.OperatorImplementation{presenceEvery()},
	})
	if err != nil {
		t.Fatalf("%s: %v", rule.ID, err)
	}
	if decisions.Relation == nil || decisions.Rows != nil {
		t.Fatalf("%s: the answer is not the relation's: %+v", rule.ID, decisions)
	}
	if decisions.Relation.Undecided != nil {
		return "undecided: " + string(decisions.Relation.Undecided.Reason)
	}
	return string(decisions.Relation.Result)
}

// A presence input declared `of each row` is answered from the rows a real query
// returned, not from one record and not from a number. The expectations below
// are decided from the seed rows of the membership table, not from the run:
//
//	D1: allocation '1', '0.5', '0', null
//	D2: allocation '1', '1'
//	every row of D1 and D2 holds a person_id.
func TestARowScopedPresenceRuleIsDecidedFromRealRows(t *testing.T) {
	executor := membershipExecutor(t)

	// Every row of D1 holds a person, so presence holds everywhere.
	d1 := readMembers(t, executor, "members", "D1", validationeval.CoverageComplete)
	if got := presenceRelationAnswer(t, d1, personID); got != "satisfied" {
		t.Fatalf("person_id over D1: got %q, want satisfied", got)
	}

	// D1's allocation column holds one null: the presence question is answered
	// "no" by that row, which refutes the operator even though the relation was
	// read completely.
	if got := presenceRelationAnswer(t, d1, allocation); got != "not-satisfied" {
		t.Fatalf("allocation over D1: got %q, want not-satisfied", got)
	}

	// D2 holds no null, so presence holds for its allocation column too.
	d2 := readMembers(t, executor, "members", "D2", validationeval.CoverageComplete)
	if got := presenceRelationAnswer(t, d2, allocation); got != "satisfied" {
		t.Fatalf("allocation over D2: got %q, want satisfied", got)
	}

	// The narrow query never returned the allocation column, so nobody read it:
	// that is not "the record holds no value", and the rule stays undecided with
	// that reason rather than being refuted by a row nobody looked at.
	narrow := readMembers(t, executor, "members_narrow", "D1", validationeval.CoverageComplete)
	if got := presenceRelationAnswer(t, narrow, allocation); got != "undecided: value-not-fetched" {
		t.Fatalf("allocation over the narrow query: got %q, want undecided value-not-fetched", got)
	}

	// The first page holds the rows '1' and '0.5' and no null, so every row that
	// was read holds a value. A partly read relation cannot be claimed to hold
	// everywhere, so the answer is undecided for that reason rather than
	// satisfied: reading more rows is what could settle it (section 15.2).
	page := readMembers(t, executor, "members_first_page", "D1", validationeval.CoveragePartial)
	if got := presenceRelationAnswer(t, page, allocation); got != "undecided: relation-partially-read" {
		t.Fatalf("allocation over the first page: got %q, want undecided relation-partially-read", got)
	}
}

package validationeval_test

import (
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A rule about presence is a pure read. These tests pin the two properties the
// migration depends on: the evaluation changes nothing it was given, and it
// does not carry an answer from one read into the next.

// countingPresenceSource is a reader that hands out one presence value per
// attribute and records every attribute it was asked about. Mutating the map
// between calls stands for the row being edited.
type countingPresenceSource struct {
	presence map[core.CanonicalID]validationeval.PresenceReading
	reads    int
	calls    []core.CanonicalID
}

func (s *countingPresenceSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, nil
}

func (s *countingPresenceSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, nil
}

func (s *countingPresenceSource) ReadPresence(attribute core.CanonicalID) (validationeval.PresenceReading, error) {
	s.reads++
	s.calls = append(s.calls, attribute)
	return s.presence[attribute], nil
}

// The answer follows the values that are read now. A rule that was satisfied
// because the trigger had a value must not stay satisfied after the trigger is
// emptied, and it must not stay not-satisfied after the reason is filled in.
//
// This is the "do not reuse a stale answer" requirement stated as behaviour: no
// cache keyed by rule identity can pass this test, because the rule is the same
// value both times.
func TestRequiredIfFollowsTheValuesThatAreReadNow(t *testing.T) {
	rule := requiredIfRule()
	source := &countingPresenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
		triggerAttribute:  presenceReading(hasValue),
		requiredAttribute: presenceReading(noValue),
	}}
	first, err := validationeval.Decide(rule, source, validationeval.Operators{})
	if err != nil || first.Result != validationeval.ResultNotSatisfied {
		t.Fatalf("first=%+v err=%v", first, err)
	}

	// The row is edited: a reason is written. The rule value is unchanged.
	source.presence[requiredAttribute] = presenceReading(hasValue)
	second, err := validationeval.Decide(rule, source, validationeval.Operators{})
	if err != nil || second.Result != validationeval.ResultSatisfied {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	if first.Result == second.Result {
		t.Fatal("a changed value did not change the answer")
	}

	// The trigger is emptied. The rule speaks about no record now.
	source.presence[triggerAttribute] = presenceReading(noValue)
	source.presence[requiredAttribute] = presenceReading(noValue)
	third, err := validationeval.Decide(rule, source, validationeval.Operators{})
	if err != nil || third.Result != validationeval.ResultSatisfied {
		t.Fatalf("third=%+v err=%v", third, err)
	}

	// Every decision read the values again rather than answering from the
	// previous one: two presence reads per call, in the same order.
	if source.reads != 6 {
		t.Fatalf("presence reads=%d, want 6 (two per decision)", source.reads)
	}
	wantCalls := []core.CanonicalID{
		triggerAttribute, requiredAttribute,
		triggerAttribute, requiredAttribute,
		triggerAttribute, requiredAttribute,
	}
	if !reflect.DeepEqual(source.calls, wantCalls) {
		t.Fatalf("calls=%v", source.calls)
	}
}

// Deciding a rule changes nothing the caller passed in. The rule is a value, so
// a copy is compared field by field; the source counts its own reads and is
// otherwise untouched.
func TestRequiredIfChangesNothingItWasGiven(t *testing.T) {
	rule := requiredIfRule()
	before := rule
	source := &countingPresenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
		triggerAttribute:  presenceReading(hasValue),
		requiredAttribute: presenceReading(noValue),
	}}
	presenceBefore := make(map[core.CanonicalID]validationeval.PresenceReading, len(source.presence))
	for id, reading := range source.presence {
		presenceBefore[id] = reading
	}
	operators := validationeval.Operators{}

	if _, err := validationeval.Decide(rule, source, operators); err != nil {
		t.Fatal(err)
	}
	if rule != before {
		t.Fatalf("the rule changed: %+v -> %+v", before, rule)
	}
	if !reflect.DeepEqual(source.presence, presenceBefore) {
		t.Fatalf("the reader's values changed: %+v", source.presence)
	}
	if len(operators.Declared) != 0 || len(operators.Implementations) != 0 {
		t.Fatalf("the operator set changed: %+v", operators)
	}
}

// Two decisions of the same rule and the same reading give the same answer.
// This is the property a caller may rely on when comparing runs, and it is what
// makes "a stale answer could have been reused" detectable: the answer would
// differ.
func TestRequiredIfIsDeterministic(t *testing.T) {
	rule := requiredIfRule()
	for _, trigger := range []string{hasValue, noValue, notRead} {
		for _, required := range []string{hasValue, noValue, notRead} {
			first := &countingPresenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
				triggerAttribute:  presenceReading(trigger),
				requiredAttribute: presenceReading(required),
			}}
			second := &countingPresenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
				triggerAttribute:  presenceReading(trigger),
				requiredAttribute: presenceReading(required),
			}}
			got1, err1 := validationeval.Decide(rule, first, validationeval.Operators{})
			got2, err2 := validationeval.Decide(rule, second, validationeval.Operators{})
			if err1 != nil || err2 != nil {
				t.Fatalf("trigger=%s required=%s: %v / %v", trigger, required, err1, err2)
			}
			if !reflect.DeepEqual(got1, got2) {
				t.Fatalf("trigger=%s required=%s: %+v != %+v", trigger, required, got1, got2)
			}
		}
	}
}

// The rule must not be decided from something other than presence. A reader
// that answers presence from a numeric value would make "0" and "no value"
// the same fact, which is the confusion the slice exists to prevent. The
// evaluator reports which attributes it read, so a reader that offers numbers
// only cannot satisfy a presence rule.
func TestRequiredIfNeverDecidesFromANumber(t *testing.T) {
	// statefulSource answers numbers and relation keys, but not presence.
	got, err := validationeval.Decide(requiredIfRule(), &statefulSource{
		number: validationeval.NumberReading{State: validationeval.ValuePresent, Value: rat("0")},
	}, validationeval.Operators{})
	if err == nil {
		t.Fatalf("a presence rule was decided by a reader that cannot read presence: %+v", got)
	}
	if got.Result != "" || got.Undecided != nil {
		t.Fatalf("a refused evaluation produced an answer: %+v", got)
	}
}

// A rule of another kind must not be decided by presence alone. This keeps the
// two directions apart: "positive" reads a number and "required if" reads
// presence, and one cannot stand in for the other.
func TestAPositiveRuleIsNotDecidedByPresence(t *testing.T) {
	positive := semantic.Validation{
		ID: "sales.order_line.quantity_positive", Symbol: "quantity_positive",
		Kind: semantic.ValidationPositive, Attribute: "sales.order_line.quantity",
	}
	source := &countingPresenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
		"sales.order_line.quantity": presenceReading(hasValue),
	}}
	got, err := validationeval.Decide(positive, source, validationeval.Operators{})
	if err == nil {
		t.Fatalf("a positive rule was decided by a presence reader: %+v", got)
	}
	if source.reads != 0 {
		t.Fatalf("a positive rule asked for presence reads: %v", source.calls)
	}
}

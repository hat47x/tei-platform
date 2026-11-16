package validationeval_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
)

// A conditional requirement is about presence, not about a value. The cases
// below write presence as one of three words, and the answers are written by
// hand from the rule's meaning, not read out of the implementation.

const (
	triggerAttribute  = core.CanonicalID("sales.order_line.discount")
	requiredAttribute = core.CanonicalID("sales.order_line.discount_reason")

	hasValue = "present"
	noValue  = "NULL"
	notRead  = "?"
)

func requiredIfRule() semantic.Validation {
	return semantic.Validation{
		ID:        "sales.order_line.discount_reason_required",
		Symbol:    "discount_reason_required",
		Kind:      semantic.ValidationRequiredIf,
		Attribute: requiredAttribute,
		Compared:  triggerAttribute,
	}
}

func presenceReading(text string) validationeval.PresenceReading {
	switch text {
	case noValue:
		return validationeval.PresenceReading{State: validationeval.ValueNull}
	case notRead:
		return validationeval.PresenceReading{State: validationeval.ValueNotFetched}
	case hasValue:
		return validationeval.PresenceReading{State: validationeval.ValuePresent}
	default:
		panic("unknown presence word " + text)
	}
}

// presenceSource records every presence read, so a test can see which
// attributes were asked about and in what order.
type presenceSource struct {
	presence map[core.CanonicalID]validationeval.PresenceReading
	err      error
	calls    []string
}

func (s *presenceSource) ReadNumber(core.CanonicalID) (validationeval.NumberReading, error) {
	return validationeval.NumberReading{}, fmt.Errorf("a rule about presence asked for a number")
}

func (s *presenceSource) ReadRelationKeys(core.CanonicalID, core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("a rule about presence asked for relation keys")
}

func (s *presenceSource) ReadPresence(attribute core.CanonicalID) (validationeval.PresenceReading, error) {
	s.calls = append(s.calls, "ReadPresence("+string(attribute)+")")
	if s.err != nil {
		return validationeval.PresenceReading{}, s.err
	}
	reading, ok := s.presence[attribute]
	if !ok {
		return validationeval.PresenceReading{}, fmt.Errorf("the test stated no presence for %s", attribute)
	}
	return reading, nil
}

// meaning is the rule on a record whose two presence values are fully known:
// either a value is there or it is not. It returns "" when the rule says
// nothing either way, which happens only when a value that must be known is
// not.
//
// The rule reads: the required attribute holds a value whenever the trigger
// attribute holds one. So a trigger without a value satisfies it whatever the
// required attribute holds, and a required attribute with a value satisfies it
// whatever the trigger holds.
func requiredIfMeaning(trigger, required string) validationeval.Result {
	if required == hasValue {
		return validationeval.ResultSatisfied
	}
	if trigger == noValue {
		return validationeval.ResultSatisfied
	}
	if trigger == hasValue {
		return validationeval.ResultNotSatisfied
	}
	return ""
}

// possibleMeanings lists what the rule could mean once every unread presence
// is known. An unread value may turn out to be there or not.
func requiredIfPossibleMeanings(trigger, required string) map[validationeval.Result]bool {
	values := func(text string) []string {
		if text == notRead {
			return []string{hasValue, noValue}
		}
		return []string{text}
	}
	out := map[validationeval.Result]bool{}
	for _, t := range values(trigger) {
		for _, r := range values(required) {
			out[requiredIfMeaning(t, r)] = true
		}
	}
	return out
}

// The evaluator is decided exactly when every completion of what was not read
// gives the same answer. This is the independent definition: it enumerates the
// completions rather than restating the implementation.
func TestRequiredIfIsDecidedExactlyWhenEveryCompletionAgrees(t *testing.T) {
	states := []string{hasValue, noValue, notRead}
	counts := map[string]int{}
	for _, trigger := range states {
		for _, required := range states {
			source := &presenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
				triggerAttribute:  presenceReading(trigger),
				requiredAttribute: presenceReading(required),
			}}
			got, err := validationeval.Decide(requiredIfRule(), source, validationeval.Operators{})
			if err != nil {
				t.Fatalf("trigger=%s required=%s: %v", trigger, required, err)
			}
			possible := requiredIfPossibleMeanings(trigger, required)
			name := fmt.Sprintf("trigger=%s required=%s possible=%v", trigger, required, possible)
			if len(possible) == 1 && !possible[""] {
				for want := range possible {
					counts[string(want)]++
					if got.Result != want || got.Undecided != nil {
						t.Fatalf("%s: got %+v", name, got)
					}
				}
			} else {
				counts["undecided"]++
				if got.Result != "" || got.Undecided == nil {
					t.Fatalf("%s: got %+v", name, got)
				}
			}
			// Both presence values are always asked for, in the same order,
			// whatever the answer turns out to be.
			want := []string{
				"ReadPresence(" + string(triggerAttribute) + ")",
				"ReadPresence(" + string(requiredAttribute) + ")",
			}
			if !reflect.DeepEqual(source.calls, want) {
				t.Fatalf("%s: reads=%v", name, source.calls)
			}
		}
	}
	if counts["satisfied"] == 0 || counts["not-satisfied"] == 0 || counts["undecided"] == 0 {
		t.Fatalf("counts=%v", counts)
	}
	t.Logf("counts=%v", counts)
}

// The answers and the reasons, written by hand. This is the same table the
// independent definition above covers, stated so a reader can check each row
// against the rule's meaning without running the code.
func TestRequiredIfAnswersAndReasons(t *testing.T) {
	cases := []struct {
		name              string
		trigger, required string
		result            validationeval.Result
		reason            validationeval.UndecidedReason
		about             core.CanonicalID
	}{
		{"the trigger has a value and so does the required attribute",
			hasValue, hasValue, validationeval.ResultSatisfied, "", ""},
		{"the trigger has a value and the required attribute does not",
			hasValue, noValue, validationeval.ResultNotSatisfied, "", ""},
		{"the trigger has no value, so the rule says nothing",
			noValue, noValue, validationeval.ResultSatisfied, "", ""},
		{"the trigger has no value and the required attribute does",
			noValue, hasValue, validationeval.ResultSatisfied, "", ""},
		{"neither was read",
			notRead, notRead, "", validationeval.UndecidedValueNotFetched, triggerAttribute},
		{"the trigger was not read and the required attribute has a value",
			notRead, hasValue, validationeval.ResultSatisfied, "", ""},
		{"the trigger was not read and the required attribute has no value",
			notRead, noValue, "", validationeval.UndecidedValueNotFetched, triggerAttribute},
		{"the trigger has a value and the required attribute was not read",
			hasValue, notRead, "", validationeval.UndecidedValueNotFetched, requiredAttribute},
		{"the trigger has no value and the required attribute was not read",
			noValue, notRead, validationeval.ResultSatisfied, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &presenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
				triggerAttribute:  presenceReading(tc.trigger),
				requiredAttribute: presenceReading(tc.required),
			}}
			got, err := validationeval.Decide(requiredIfRule(), source, validationeval.Operators{})
			if err != nil {
				t.Fatal(err)
			}
			if got.Result != tc.result {
				t.Fatalf("result=%q, want %q (undecided=%v)", got.Result, tc.result, got.Undecided)
			}
			if tc.reason == "" {
				if got.Undecided != nil {
					t.Fatalf("undecided=%v", got.Undecided)
				}
				return
			}
			want := validationeval.Undecided{Reason: tc.reason, Attribute: tc.about}
			if got.Undecided == nil || *got.Undecided != want {
				t.Fatalf("undecided=%v, want %v", got.Undecided, want)
			}
		})
	}
}

// The rule is about presence, so it never turns an empty value into zero and
// never decides from a value of another type. This is the separation the
// migration needs: a "quantity is positive" rule and a "reason is required"
// rule read different things and must not be substituted for each other.
func TestRequiredIfRefusesAReaderOrReadingItCannotUse(t *testing.T) {
	t.Run("a reader that cannot read presence", func(t *testing.T) {
		got, err := validationeval.Decide(requiredIfRule(), &statefulSource{}, validationeval.Operators{})
		if err == nil || !strings.Contains(err.Error(), "presence") || got.Result != "" || got.Undecided != nil {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("the older entry without states", func(t *testing.T) {
		got, err := validationeval.Evaluate(requiredIfRule(), &operatorSource{value: rat("1")})
		if err == nil || got != "" {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})
	t.Run("a rule without a trigger attribute", func(t *testing.T) {
		rule := requiredIfRule()
		rule.Compared = ""
		source := &presenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
			triggerAttribute:  presenceReading(hasValue),
			requiredAttribute: presenceReading(hasValue),
		}}
		if _, err := validationeval.Decide(rule, source, validationeval.Operators{}); err == nil {
			t.Fatal("a rule with no trigger was decided")
		}
	})
	t.Run("a reading in an unknown state", func(t *testing.T) {
		source := &presenceSource{presence: map[core.CanonicalID]validationeval.PresenceReading{
			triggerAttribute:  {State: validationeval.ValueState("maybe")},
			requiredAttribute: presenceReading(hasValue),
		}}
		got, err := validationeval.Decide(requiredIfRule(), source, validationeval.Operators{})
		if err == nil || !strings.Contains(err.Error(), "unknown state") || got.Result != "" || got.Undecided != nil {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("a reader that fails", func(t *testing.T) {
		source := &presenceSource{err: fmt.Errorf("connection lost")}
		if _, err := validationeval.Decide(requiredIfRule(), source, validationeval.Operators{}); err == nil ||
			!strings.Contains(err.Error(), "connection lost") {
			t.Fatalf("err=%v", err)
		}
	})
}

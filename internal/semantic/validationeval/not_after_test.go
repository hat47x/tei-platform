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

// A rule that compares two dates of one record is checked against a second
// definition: write out every value a date that was not read could have,
// decide the rule for each, and call the rule decided only if all agree.

type dateSource struct {
	statefulSource
	dates map[core.CanonicalID]validationeval.DateReading
	err   error
}

func (s *dateSource) ReadDate(attribute core.CanonicalID) (validationeval.DateReading, error) {
	s.calls = append(s.calls, "ReadDate("+string(attribute)+")")
	reading, known := s.dates[attribute]
	if !known {
		return validationeval.DateReading{}, fmt.Errorf("no reading prepared for %s", attribute)
	}
	return reading, s.err
}

const (
	startsOn core.CanonicalID = "organization.membership.starts_on"
	endsOn   core.CanonicalID = "organization.membership.ends_on"
)

func periodRule(ifPresent bool) semantic.Validation {
	return semantic.Validation{
		ID: "organization.membership.period_in_order", Symbol: "period_in_order",
		Kind: semantic.ValidationNotAfter, Attribute: startsOn, Compared: endsOn, IfPresent: ifPresent,
	}
}

// The states a date can be read in, as written in the cases below.
const (
	early  = "2026-04-01"
	late   = "2026-09-30"
	none   = "NULL"
	unread = "?"
)

func dateReading(text string) validationeval.DateReading {
	switch text {
	case none:
		return validationeval.DateReading{State: validationeval.ValueNull}
	case unread:
		return validationeval.DateReading{State: validationeval.ValueNotFetched}
	}
	date, err := semantic.ParseCalendarDate(text)
	if err != nil {
		panic(err)
	}
	return validationeval.DateReading{State: validationeval.ValuePresent, Value: date}
}

// meaning is the rule on a record whose two dates are fully known: either a
// date as text, or none. It returns "" when the rule says nothing either way.
func meaning(start, end string, ifPresent bool) validationeval.Result {
	if end == none && ifPresent {
		return validationeval.ResultSatisfied
	}
	if start == none || end == none {
		return ""
	}
	// Dates written YYYY-MM-DD order as texts.
	if start <= end {
		return validationeval.ResultSatisfied
	}
	return validationeval.ResultNotSatisfied
}

// possibleMeanings lists what the rule could mean once every unread date is
// known.
func possibleMeanings(start, end string, ifPresent bool) map[validationeval.Result]bool {
	values := func(text string) []string {
		if text == unread {
			return []string{none, "2025-01-01", early, "2026-06-15", late, "2027-12-31"}
		}
		return []string{text}
	}
	out := map[validationeval.Result]bool{}
	for _, s := range values(start) {
		for _, e := range values(end) {
			out[meaning(s, e, ifPresent)] = true
		}
	}
	return out
}

func TestNotAfterIsDecidedExactlyWhenEveryCompletionAgrees(t *testing.T) {
	states := []string{early, late, none, unread}
	counts := map[string]int{}
	for _, ifPresent := range []bool{false, true} {
		for _, start := range states {
			for _, end := range states {
				source := &dateSource{dates: map[core.CanonicalID]validationeval.DateReading{startsOn: dateReading(start), endsOn: dateReading(end)}}
				got, err := validationeval.Decide(periodRule(ifPresent), source, validationeval.Operators{})
				if err != nil {
					t.Fatalf("start=%s end=%s ifPresent=%v: %v", start, end, ifPresent, err)
				}
				possible := possibleMeanings(start, end, ifPresent)
				name := fmt.Sprintf("start=%s end=%s ifPresent=%v possible=%v", start, end, ifPresent, possible)
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
				// Both dates are always asked for, in the same order.
				if !reflect.DeepEqual(source.calls, []string{"ReadDate(" + string(startsOn) + ")", "ReadDate(" + string(endsOn) + ")"}) {
					t.Fatalf("%s: reads=%v", name, source.calls)
				}
			}
		}
	}
	if counts["satisfied"] == 0 || counts["not-satisfied"] == 0 || counts["undecided"] == 0 {
		t.Fatalf("counts=%v", counts)
	}
	t.Logf("counts=%v", counts)
}

// The answers and the reasons, written by hand.
func TestNotAfterAnswersAndReasons(t *testing.T) {
	cases := []struct {
		name       string
		start, end string
		ifPresent  bool
		result     validationeval.Result
		reason     validationeval.UndecidedReason
		about      core.CanonicalID
	}{
		{"the start is before the end", early, late, false, validationeval.ResultSatisfied, "", ""},
		{"a period of one day", early, early, false, validationeval.ResultSatisfied, "", ""},
		{"the start is after the end", late, early, false, validationeval.ResultNotSatisfied, "", ""},
		{"the start is after the end, if present", late, early, true, validationeval.ResultNotSatisfied, "", ""},
		{"no end, and the rule speaks only of records with one", early, none, true, validationeval.ResultSatisfied, "", ""},
		{"no end, and the rule does not say so", early, none, false, "", validationeval.UndecidedValueNull, endsOn},
		{"no start", none, late, true, "", validationeval.UndecidedValueNull, startsOn},
		{"no start and no end, if present", none, none, true, validationeval.ResultSatisfied, "", ""},
		{"the start was not read and there is no end, if present", unread, none, true, validationeval.ResultSatisfied, "", ""},
		{"the end was not read", early, unread, true, "", validationeval.UndecidedValueNotFetched, endsOn},
		{"the start was not read", unread, late, false, "", validationeval.UndecidedValueNotFetched, startsOn},
		{"neither was read", unread, unread, true, "", validationeval.UndecidedValueNotFetched, startsOn},
		{"the end was not read and there is no start", none, unread, true, "", validationeval.UndecidedValueNotFetched, endsOn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &dateSource{dates: map[core.CanonicalID]validationeval.DateReading{startsOn: dateReading(tc.start), endsOn: dateReading(tc.end)}}
			got, err := validationeval.Decide(periodRule(tc.ifPresent), source, validationeval.Operators{})
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

func TestNotAfterRefusesAReaderOrReadingItCannotUse(t *testing.T) {
	present := dateReading(early)
	t.Run("a reader that cannot read dates", func(t *testing.T) {
		got, err := validationeval.Decide(periodRule(true), &statefulSource{}, validationeval.Operators{})
		if err == nil || !strings.Contains(err.Error(), "cannot read") || got.Result != "" || got.Undecided != nil {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("the older entry", func(t *testing.T) {
		if got, err := validationeval.Evaluate(periodRule(true), &operatorSource{value: rat("1")}); err == nil || got != "" {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})
	t.Run("a rule without an attribute to compare with", func(t *testing.T) {
		rule := periodRule(true)
		rule.Compared = ""
		source := &dateSource{dates: map[core.CanonicalID]validationeval.DateReading{startsOn: present}}
		if _, err := validationeval.Decide(rule, source, validationeval.Operators{}); err == nil {
			t.Fatal("a comparison with nothing was decided")
		}
	})
	contradictions := []struct {
		name    string
		reading validationeval.DateReading
		mention string
	}{
		{"present without a date", validationeval.DateReading{State: validationeval.ValuePresent}, "present without a value"},
		{"null with a date", validationeval.DateReading{State: validationeval.ValueNull, Value: present.Value}, "together with a value"},
		{"not fetched with a date", validationeval.DateReading{State: validationeval.ValueNotFetched, Value: present.Value}, "together with a value"},
		{"no state", validationeval.DateReading{Value: present.Value}, "unknown state"},
	}
	for _, tc := range contradictions {
		for _, attribute := range []core.CanonicalID{startsOn, endsOn} {
			t.Run(tc.name+" for "+string(attribute), func(t *testing.T) {
				dates := map[core.CanonicalID]validationeval.DateReading{startsOn: present, endsOn: present}
				dates[attribute] = tc.reading
				got, err := validationeval.Decide(periodRule(true), &dateSource{dates: dates}, validationeval.Operators{})
				if err == nil || !strings.Contains(err.Error(), tc.mention) || !strings.Contains(err.Error(), string(attribute)) ||
					got.Result != "" || got.Undecided != nil {
					t.Fatalf("got=%+v err=%v", got, err)
				}
			})
		}
	}
	t.Run("a reader that fails", func(t *testing.T) {
		source := &dateSource{dates: map[core.CanonicalID]validationeval.DateReading{startsOn: present, endsOn: present}, err: fmt.Errorf("connection lost")}
		if _, err := validationeval.Decide(periodRule(true), source, validationeval.Operators{}); err == nil || !strings.Contains(err.Error(), "connection lost") {
			t.Fatalf("err=%v", err)
		}
	})
}

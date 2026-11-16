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

// "The periods of the rows of a relation do not overlap" is checked against
// a second definition. It states the meaning of the rule for rows that are
// fully known, writes out every way the unread part could turn out, and calls
// the rule decided only when all of them mean the same.

const (
	membersRelation core.CanonicalID = "organization.department.members"
	personID        core.CanonicalID = "organization.membership.person_id"
)

func overlapRule(openEnded, grouped bool) semantic.Validation {
	rule := semantic.Validation{
		ID: "organization.department.members.periods_apart", Symbol: "periods_apart", Kind: semantic.ValidationNoOverlap,
		Attribute: startsOn, Compared: endsOn, Relation: membersRelation, OpenEnded: openEnded,
	}
	if grouped {
		rule.GroupBy = personID
	}
	return rule
}

// periodCase is one row as the cases below write it. A date is "d1".."d6"
// (in calendar order), "NULL" for no value, or "?" for not read. A group is
// any other text, "NULL" or "?".
type periodCase struct{ start, end, group string }

var caseDates = map[string]string{
	"d1": "2026-01-10", "d2": "2026-02-10", "d3": "2026-03-10", "d4": "2026-04-10", "d5": "2026-05-10", "d6": "2026-06-10",
}

func caseDate(text string) validationeval.DateReading {
	if iso, known := caseDates[text]; known {
		return dateReading(iso)
	}
	return dateReading(text) // NULL or ?
}

func caseGroup(text string) validationeval.KeyReading {
	switch text {
	case none:
		return validationeval.KeyReading{State: validationeval.ValueNull}
	case unread:
		return validationeval.KeyReading{State: validationeval.ValueNotFetched}
	}
	return validationeval.KeyReading{State: validationeval.ValuePresent, Key: text}
}

type periodSource struct {
	statefulSource
	rows    []periodCase
	partial bool
	err     error
}

func (s *periodSource) ReadRelationPeriods(relation, start, end, group core.CanonicalID) (validationeval.PeriodRowsReading, error) {
	s.calls = append(s.calls, fmt.Sprintf("ReadRelationPeriods(%s,%s,%s,%s)", relation, start, end, group))
	reading := validationeval.PeriodRowsReading{Coverage: validationeval.CoverageComplete}
	if s.partial {
		reading.Coverage = validationeval.CoveragePartial
	}
	for _, row := range s.rows {
		period := validationeval.PeriodRow{Start: caseDate(row.start), End: caseDate(row.end)}
		if group != "" {
			period.Group = caseGroup(row.group)
		}
		reading.Rows = append(reading.Rows, period)
	}
	return reading, s.err
}

// overlapMeaning is the rule on rows that are fully known: every date is a
// day or NULL, every group a value or NULL. It returns "" when the rule says
// nothing either way.
func overlapMeaning(rows []periodCase, openEnded, grouped bool) validationeval.Result {
	// A row states a period when it has a first day and a last day, or no
	// last day and the rule calls that open.
	statesPeriod := func(row periodCase) bool {
		return row.start != none && (row.end != none || openEnded)
	}
	// Day names order as texts ("d1" < "d2"); an open period has no last day.
	shareADay := func(a, b periodCase) bool {
		return (b.end == none || a.start <= b.end) && (a.end == none || b.start <= a.end)
	}
	silent, unstatedGroup := false, false
	for i, a := range rows {
		if !statesPeriod(a) {
			silent = true
			continue
		}
		for _, b := range rows[i+1:] {
			if !statesPeriod(b) || !shareADay(a, b) {
				continue
			}
			switch {
			case !grouped:
				return validationeval.ResultNotSatisfied
			case a.group != none && a.group == b.group:
				return validationeval.ResultNotSatisfied
			case a.group == none && b.group == none:
				// The rule does not say whether rows without a value are
				// one group.
				unstatedGroup = true
			}
		}
	}
	if silent || unstatedGroup {
		return ""
	}
	return validationeval.ResultSatisfied
}

// possibleOverlapMeanings writes out every completion of the unread parts and
// collects what the rule means for each. It stops once two meanings differ.
func possibleOverlapMeanings(rows []periodCase, partial, openEnded, grouped bool) map[validationeval.Result]bool {
	dates := []string{"d1", "d2", "d4", "d6", none}
	groups := []string{"a", "b", none}
	out := map[validationeval.Result]bool{}

	base := append([]periodCase(nil), rows...)
	extras := [][]periodCase{nil}
	if partial {
		// One row that was not read at all is enough to tell.
		for _, start := range dates {
			for _, end := range dates {
				for _, group := range groups {
					extras = append(extras, []periodCase{{start, end, group}})
				}
			}
		}
	}

	// slots lists the unread fields of the rows that were read.
	type slot struct {
		row   int
		field int // 0 start, 1 end, 2 group
	}
	var slots []slot
	for i, row := range base {
		for field, value := range []string{row.start, row.end, row.group} {
			if value == unread && (field < 2 || grouped) {
				slots = append(slots, slot{i, field})
			}
		}
	}
	choices := func(s slot) []string {
		if s.field == 2 {
			return groups
		}
		return dates
	}
	assignment := make([]int, len(slots))
	for {
		world := append([]periodCase(nil), base...)
		for n, s := range slots {
			value := choices(s)[assignment[n]]
			switch s.field {
			case 0:
				world[s.row].start = value
			case 1:
				world[s.row].end = value
			default:
				world[s.row].group = value
			}
		}
		for _, extra := range extras {
			out[overlapMeaning(append(append([]periodCase(nil), world...), extra...), openEnded, grouped)] = true
			if len(out) > 1 {
				return out
			}
		}
		n := 0
		for ; n < len(slots); n++ {
			assignment[n]++
			if assignment[n] < len(choices(slots[n])) {
				break
			}
			assignment[n] = 0
		}
		if n == len(slots) {
			return out
		}
	}
}

func TestTheOverlapMeaningItself(t *testing.T) {
	cases := []struct {
		name               string
		rows               []periodCase
		openEnded, grouped bool
		want               validationeval.Result
	}{
		{"periods one after the other", []periodCase{{"d1", "d2", "a"}, {"d3", "d4", "a"}}, false, true, validationeval.ResultSatisfied},
		{"the next starts on the last day", []periodCase{{"d1", "d3", "a"}, {"d3", "d4", "a"}}, false, true, validationeval.ResultNotSatisfied},
		{"one inside the other", []periodCase{{"d1", "d6", "a"}, {"d3", "d4", "a"}}, false, true, validationeval.ResultNotSatisfied},
		{"the same days for two people", []periodCase{{"d1", "d6", "a"}, {"d1", "d6", "b"}}, false, true, validationeval.ResultSatisfied},
		{"the same days without grouping", []periodCase{{"d1", "d6", "a"}, {"d1", "d6", "b"}}, false, false, validationeval.ResultNotSatisfied},
		{"an open period and a later one", []periodCase{{"d1", none, "a"}, {"d5", "d6", "a"}}, true, true, validationeval.ResultNotSatisfied},
		{"an open period and an earlier one", []periodCase{{"d3", none, "a"}, {"d1", "d2", "a"}}, true, true, validationeval.ResultSatisfied},
		{"no end, and the rule does not call it open", []periodCase{{"d1", none, "a"}, {"d5", "d6", "a"}}, false, true, ""},
		{"no start", []periodCase{{none, "d2", "a"}}, true, true, ""},
		{"no rows", nil, false, true, validationeval.ResultSatisfied},
	}
	for _, tc := range cases {
		if got := overlapMeaning(tc.rows, tc.openEnded, tc.grouped); got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// Every reading of up to two rows, each with two possible first days, two
// possible last days and two groups, any of them empty or unread; complete
// and partial; with and without `or open`; grouped and not.
func TestNoOverlapIsDecidedExactlyWhenEveryCompletionAgrees(t *testing.T) {
	var universe []periodCase
	for _, start := range []string{"d2", "d4", none, unread} {
		for _, end := range []string{"d3", "d5", none, unread} {
			for _, group := range []string{"a", "b", none, unread} {
				universe = append(universe, periodCase{start, end, group})
			}
		}
	}
	readings := [][]periodCase{nil}
	for i, first := range universe {
		readings = append(readings, []periodCase{first})
		for _, second := range universe[i:] {
			readings = append(readings, []periodCase{first, second})
		}
	}

	counts := map[string]int{}
	for _, rows := range readings {
		for _, partial := range []bool{false, true} {
			for _, openEnded := range []bool{false, true} {
				for _, grouped := range []bool{false, true} {
					rule := overlapRule(openEnded, grouped)
					got, err := validationeval.Decide(rule, &periodSource{rows: rows, partial: partial}, validationeval.Operators{})
					name := fmt.Sprintf("rows=%v partial=%v openEnded=%v grouped=%v", rows, partial, openEnded, grouped)
					if err != nil {
						t.Fatalf("%s: %v", name, err)
					}
					possible := possibleOverlapMeanings(rows, partial, openEnded, grouped)
					if len(possible) == 1 && !possible[""] {
						for want := range possible {
							counts[string(want)]++
							if got.Result != want || got.Undecided != nil {
								t.Fatalf("%s: every completion means %s, got %+v", name, want, got)
							}
						}
						continue
					}
					counts["undecided"]++
					if got.Result != "" || got.Undecided == nil {
						t.Fatalf("%s: the completions mean %v, got %+v", name, possible, got)
					}
					if got.Undecided.Relation != membersRelation {
						t.Fatalf("%s: the reason names %+v", name, got.Undecided)
					}
				}
			}
		}
	}
	if counts["satisfied"] == 0 || counts["not-satisfied"] == 0 || counts["undecided"] == 0 {
		t.Fatalf("counts=%v", counts)
	}
	t.Logf("readings=%d counts=%v", len(readings), counts)
}

// The answers and the reasons, written by hand.
func TestNoOverlapAnswersAndReasons(t *testing.T) {
	cases := []struct {
		name                        string
		rows                        []periodCase
		partial, openEnded, grouped bool
		result                      validationeval.Result
		reason                      validationeval.UndecidedReason
		about                       core.CanonicalID
	}{
		{"a person moves to another post the next day", []periodCase{{"d1", "d2", "p1"}, {"d3", "d4", "p1"}}, false, false, true, validationeval.ResultSatisfied, "", ""},
		{"two people at the same time", []periodCase{{"d1", "d4", "p1"}, {"d2", "d3", "p2"}}, false, false, true, validationeval.ResultSatisfied, "", ""},
		{"one person twice at the same time", []periodCase{{"d1", "d4", "p1"}, {"d2", "d3", "p1"}}, false, false, true, validationeval.ResultNotSatisfied, "", ""},
		{"the last day and the first day are the same day", []periodCase{{"d1", "d2", "p1"}, {"d2", "d3", "p1"}}, false, false, true, validationeval.ResultNotSatisfied, "", ""},
		{"prices of one item, no grouping", []periodCase{{"d1", "d2", ""}, {"d3", none, ""}}, false, true, false, validationeval.ResultSatisfied, "", ""},
		{"two open periods of one person", []periodCase{{"d1", none, "p1"}, {"d5", none, "p1"}}, false, true, true, validationeval.ResultNotSatisfied, "", ""},
		{"an overlap seen in a part", []periodCase{{"d1", "d4", "p1"}, {"d2", "d3", "p1"}}, true, false, true, validationeval.ResultNotSatisfied, "", ""},
		{"an overlap seen beside an unread row", []periodCase{{"d1", "d4", "p1"}, {unread, unread, unread}, {"d2", "d3", "p1"}}, false, false, true, validationeval.ResultNotSatisfied, "", ""},
		{"only a part, nothing seen", []periodCase{{"d1", "d2", "p1"}}, true, false, true, "", validationeval.UndecidedRelationPartiallyRead, startsOn},
		{"an end that was not read", []periodCase{{"d1", unread, "p1"}}, false, false, true, "", validationeval.UndecidedValueNotFetched, endsOn},
		// With `or open` the only row states a period whatever its end is.
		{"an end that was not read, or open, alone", []periodCase{{"d1", unread, "p1"}}, false, true, true, validationeval.ResultSatisfied, "", ""},
		{"an end that was not read, or open, beside a later row", []periodCase{{"d1", unread, "p1"}, {"d3", "d4", "p1"}}, false, true, true, "", validationeval.UndecidedValueNotFetched, endsOn},
		{"an end that was not read, or open, beside another person", []periodCase{{"d1", unread, "p1"}, {"d3", "d4", "p2"}}, false, true, true, validationeval.ResultSatisfied, "", ""},
		{"a start that was not read", []periodCase{{unread, "d2", "p1"}, {"d5", "d6", "p1"}}, false, false, true, "", validationeval.UndecidedValueNotFetched, startsOn},
		{"no end, and the rule does not call it open", []periodCase{{"d1", none, "p1"}}, false, false, true, "", validationeval.UndecidedValueNull, endsOn},
		{"no start", []periodCase{{none, "d2", "p1"}}, false, true, true, "", validationeval.UndecidedValueNull, startsOn},
		{"a person not read, at the same time as another row", []periodCase{{"d1", "d4", unread}, {"d2", "d3", "p1"}}, false, false, true, "", validationeval.UndecidedRelationKeyNotFetched, personID},
		{"a person not read, at another time", []periodCase{{"d1", "d2", unread}, {"d3", "d4", "p1"}}, false, false, true, validationeval.ResultSatisfied, "", ""},
		{"two rows without a person at the same time", []periodCase{{"d1", "d4", none}, {"d2", "d3", none}}, false, false, true, "", validationeval.UndecidedRelationKeyNull, personID},
		{"a row without a person beside a row with one", []periodCase{{"d1", "d4", none}, {"d2", "d3", "p1"}}, false, false, true, validationeval.ResultSatisfied, "", ""},
		{"unread before partial", []periodCase{{"d1", unread, "p1"}}, true, false, true, "", validationeval.UndecidedValueNotFetched, endsOn},
		{"partial before empty", []periodCase{{"d1", none, "p1"}}, true, false, true, "", validationeval.UndecidedRelationPartiallyRead, startsOn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &periodSource{rows: tc.rows, partial: tc.partial}
			rule := overlapRule(tc.openEnded, tc.grouped)
			got, err := validationeval.Decide(rule, source, validationeval.Operators{})
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
			} else if want := (validationeval.Undecided{Reason: tc.reason, Relation: membersRelation, Attribute: tc.about}); got.Undecided == nil || *got.Undecided != want {
				t.Fatalf("undecided=%v, want %v", got.Undecided, want)
			}
			wantGroup := ""
			if tc.grouped {
				wantGroup = string(personID)
			}
			wantCall := fmt.Sprintf("ReadRelationPeriods(%s,%s,%s,%s)", membersRelation, startsOn, endsOn, wantGroup)
			if !reflect.DeepEqual(source.calls, []string{wantCall}) {
				t.Fatalf("reads=%v, want %s", source.calls, wantCall)
			}
		})
	}
}

type brokenPeriodSource struct {
	statefulSource
	reading validationeval.PeriodRowsReading
}

func (s *brokenPeriodSource) ReadRelationPeriods(_, _, _, _ core.CanonicalID) (validationeval.PeriodRowsReading, error) {
	return s.reading, nil
}

func TestNoOverlapRefusesAReaderOrReadingItCannotUse(t *testing.T) {
	present := dateReading(early)
	good := validationeval.PeriodRow{Start: present, End: present, Group: validationeval.KeyReading{State: validationeval.ValuePresent, Key: "p1"}}
	t.Run("a reader that cannot read periods", func(t *testing.T) {
		got, err := validationeval.Decide(overlapRule(true, true), &dateSource{}, validationeval.Operators{})
		if err == nil || !strings.Contains(err.Error(), "cannot read") || got.Result != "" || got.Undecided != nil {
			t.Fatalf("got=%+v err=%v", got, err)
		}
	})
	t.Run("the older entry", func(t *testing.T) {
		if got, err := validationeval.Evaluate(overlapRule(true, true), &operatorSource{value: rat("1")}); err == nil || got != "" {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})
	t.Run("a rule without an end of the period", func(t *testing.T) {
		rule := overlapRule(true, true)
		rule.Compared = ""
		if _, err := validationeval.Decide(rule, &periodSource{}, validationeval.Operators{}); err == nil {
			t.Fatal("a period without an end attribute was decided")
		}
	})
	t.Run("a reader that fails", func(t *testing.T) {
		source := &periodSource{err: fmt.Errorf("connection lost")}
		if _, err := validationeval.Decide(overlapRule(true, true), source, validationeval.Operators{}); err == nil || !strings.Contains(err.Error(), "connection lost") {
			t.Fatalf("err=%v", err)
		}
	})
	cases := []struct {
		name    string
		change  func(reading *validationeval.PeriodRowsReading)
		mention string
	}{
		{"no stated coverage", func(r *validationeval.PeriodRowsReading) { r.Coverage = "" }, "unknown coverage"},
		{"a start present without a date", func(r *validationeval.PeriodRowsReading) {
			r.Rows[0].Start = validationeval.DateReading{State: validationeval.ValuePresent}
		}, "present without a value"},
		{"an end that is null with a date", func(r *validationeval.PeriodRowsReading) {
			r.Rows[0].End = validationeval.DateReading{State: validationeval.ValueNull, Value: present.Value}
		}, "together with a value"},
		{"an end in no state", func(r *validationeval.PeriodRowsReading) {
			r.Rows[0].End = validationeval.DateReading{Value: present.Value}
		}, "unknown state"},
		{"a group that is null with a value", func(r *validationeval.PeriodRowsReading) {
			r.Rows[0].Group = validationeval.KeyReading{State: validationeval.ValueNull, Key: "p1"}
		}, "together with a value"},
		{"a group in no state", func(r *validationeval.PeriodRowsReading) { r.Rows[0].Group = validationeval.KeyReading{Key: "p1"} }, "unknown state"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reading := validationeval.PeriodRowsReading{Coverage: validationeval.CoverageComplete, Rows: []validationeval.PeriodRow{good}}
			tc.change(&reading)
			got, err := validationeval.Decide(overlapRule(true, true), &brokenPeriodSource{reading: reading}, validationeval.Operators{})
			if err == nil || !strings.Contains(err.Error(), tc.mention) || got.Result != "" || got.Undecided != nil {
				t.Fatalf("got=%+v err=%v", got, err)
			}
		})
	}
	// Without grouping the group reading is not looked at.
	reading := validationeval.PeriodRowsReading{Coverage: validationeval.CoverageComplete, Rows: []validationeval.PeriodRow{{Start: present, End: present}}}
	if got, err := validationeval.Decide(overlapRule(true, false), &brokenPeriodSource{reading: reading}, validationeval.Operators{}); err != nil || got.Result != validationeval.ResultSatisfied {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

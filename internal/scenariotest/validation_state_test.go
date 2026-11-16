package scenariotest

import (
	"reflect"
	"strings"
	"testing"
)

// A Fixture says for each input whether the value is there and, if not, why.
// The words are few and exact: anything else is refused, so a misspelt state
// cannot be read as "present" or as "complete".

func TestValidationFixtureStatesAValueThatIsNotThere(t *testing.T) {
	source := `{
  "numbers": {"sales.order_line.quantity": "2"},
  "numberStates": {"sales.order_line.discount": "null", "sales.order_line.unit_price": "not-fetched"},
  "relationKeys": [
    {"relation": "sales.order.lines", "attribute": "sales.order_line.line_number",
     "keys": ["1", "2"], "coverage": "partial", "nullKeys": 1, "notFetchedKeys": 2}
  ]
}`
	got, err := ParseValidationFixture([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	want := ValidationFixture{
		Numbers:      map[string]string{"sales.order_line.quantity": "2"},
		NumberStates: map[string]string{"sales.order_line.discount": "null", "sales.order_line.unit_price": "not-fetched"},
		RelationKeys: []ValidationRelation{{
			Relation: "sales.order.lines", Attribute: "sales.order_line.line_number",
			Keys: []string{"1", "2"}, Coverage: "partial", NullKeys: 1, NotFetchedKeys: 2,
		}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v", got)
	}
}

func TestValidationFixtureWithoutStatesReadsAsBefore(t *testing.T) {
	got, err := ParseValidationFixture([]byte(`{"numbers": {"a.b.c": "1"}, "relationKeys": [{"relation": "a.b.r", "attribute": "a.c.k", "keys": []}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := ValidationFixture{
		Numbers:      map[string]string{"a.b.c": "1"},
		RelationKeys: []ValidationRelation{{Relation: "a.b.r", Attribute: "a.c.k", Keys: []string{}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v", got)
	}
}

func TestValidationFixtureRefusesAnUnclearState(t *testing.T) {
	relation := func(extra string) string {
		return `{"numbers": {}, "relationKeys": [{"relation": "a.b.r", "attribute": "a.c.k", "keys": ["1"]` + extra + `}]}`
	}
	cases := []struct {
		name, source, mention string
	}{
		{"an unknown state word", `{"numbers": {}, "numberStates": {"a.b.c": "missing"}, "relationKeys": []}`, "write null or not-fetched"},
		{"present is not a state to write", `{"numbers": {}, "numberStates": {"a.b.c": "present"}, "relationKeys": []}`, "write null or not-fetched"},
		{"an empty state", `{"numbers": {}, "numberStates": {"a.b.c": ""}, "relationKeys": []}`, "write null or not-fetched"},
		{"a state in another case", `{"numbers": {}, "numberStates": {"a.b.c": "NULL"}, "relationKeys": []}`, "write null or not-fetched"},
		{"a value and a state for one attribute", `{"numbers": {"a.b.c": "1"}, "numberStates": {"a.b.c": "null"}, "relationKeys": []}`, "both a value and the state"},
		{"a state without an attribute", `{"numbers": {}, "numberStates": {" ": "null"}, "relationKeys": []}`, "empty Attribute"},
		{"a JSON null in place of a number", `{"numbers": {"a.b.c": null}, "relationKeys": []}`, "invalid rational value"},
		{"an unknown coverage word", relation(`, "coverage": "some"`), "write complete or partial"},
		{"a negative count of empty rows", relation(`, "nullKeys": -1`), "negative key count"},
		{"a negative count of unread rows", relation(`, "notFetchedKeys": -1`), "negative key count"},
		{"a state field nobody defined", relation(`, "unknownKeys": 1`), "unknown field"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseValidationFixture([]byte(tc.source))
			if err == nil || !strings.Contains(err.Error(), tc.mention) {
				t.Fatalf("got=%+v err=%v, want an error mentioning %q", got, err, tc.mention)
			}
		})
	}
}

// The words are part of the Fixture format. Renaming a constant must not
// silently change what an existing Fixture file means.
func TestValidationFixtureStateWordsAreFixed(t *testing.T) {
	words := map[string]string{
		ValidationStateNull:        "null",
		ValidationStateNotFetched:  "not-fetched",
		ValidationCoverageComplete: "complete",
		ValidationCoveragePartial:  "partial",
		ValidationStatePresent:     "present",
	}
	if len(words) != 5 {
		t.Fatalf("two words are the same: %v", words)
	}
	for got, want := range words {
		if got != want {
			t.Fatalf("the Fixture word %q changed to %q", want, got)
		}
	}
}

// A rule about presence reads whether a value is there, whatever its type. The
// Fixture states that directly, so an empty value and an unread one stay apart
// from "a value is there".
func TestValidationFixtureStatesPresence(t *testing.T) {
	got, err := ParseValidationFixture([]byte(`{"numbers": {}, "relationKeys": [],
  "presenceStates": {"sales.order_line.discount": "present",
                     "sales.order_line.discount_reason": "null",
                     "sales.order_line.note": "not-fetched"}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := ValidationFixture{
		Numbers: map[string]string{}, RelationKeys: []ValidationRelation{},
		PresenceStates: map[string]string{
			"sales.order_line.discount":        "present",
			"sales.order_line.discount_reason": "null",
			"sales.order_line.note":            "not-fetched",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v", got)
	}

	cases := []struct{ name, members, mention string }{
		{"an unknown presence word", `"presenceStates": {"a.b.c": "maybe"}`, "write present, null or not-fetched"},
		{"a presence word without an attribute", `"presenceStates": {" ": "present"}`, "empty Attribute"},
		{"a typed state and a presence word for one attribute", `"numberStates": {"a.b.c": "null"}, "presenceStates": {"a.b.c": "present"}`, "also has the presence state"},
		{"a date state and a presence word for one attribute", `"dateStates": {"a.b.c": "null"}, "presenceStates": {"a.b.c": "present"}`, "also has the presence state"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseValidationFixture([]byte(`{"numbers": {}, "relationKeys": [], ` + tc.members + `}`))
			if err == nil || !strings.Contains(err.Error(), tc.mention) {
				t.Fatalf("got=%+v err=%v, want an error mentioning %q", got, err, tc.mention)
			}
		})
	}
}

// A typed value and a presence word for one attribute state two different
// facts about one attribute, so the Fixture refuses both at once. This case
// needs its own Fixture because it must supply the typed value itself.
func TestValidationFixtureRefusesATypedValueBesideAPresenceWord(t *testing.T) {
	_, err := ParseValidationFixture([]byte(`{"numbers": {"a.b.c": "1"}, "relationKeys": [],
  "presenceStates": {"a.b.c": "present"}}`))
	if err == nil || !strings.Contains(err.Error(), "also has the presence state") {
		t.Fatalf("err=%v, want a refusal of the presence word beside a typed value", err)
	}
}

// Dates are stated like numbers: a value written YYYY-MM-DD, or a state.
func TestValidationFixtureStatesDates(t *testing.T) {
	got, err := ParseValidationFixture([]byte(`{"numbers": {}, "relationKeys": [],
  "dates": {"organization.membership.starts_on": "2026-04-01"},
  "dateStates": {"organization.membership.ends_on": "null", "organization.membership.left_on": "not-fetched"}}`))
	if err != nil {
		t.Fatal(err)
	}
	want := ValidationFixture{
		Numbers: map[string]string{}, RelationKeys: []ValidationRelation{},
		Dates:      map[string]string{"organization.membership.starts_on": "2026-04-01"},
		DateStates: map[string]string{"organization.membership.ends_on": "null", "organization.membership.left_on": "not-fetched"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v", got)
	}

	cases := []struct{ name, members, mention string }{
		{"a date with slashes", `"dates": {"a.b.c": "2026/04/01"}`, "is not a calendar date"},
		{"a date with a time", `"dates": {"a.b.c": "2026-04-01T00:00:00Z"}`, "is not a calendar date"},
		{"a day the calendar does not have", `"dates": {"a.b.c": "2026-02-30"}`, "is not a calendar date"},
		{"a JSON null in place of a date", `"dates": {"a.b.c": null}`, "is not a calendar date"},
		{"a date without an attribute", `"dates": {" ": "2026-04-01"}`, "empty Attribute"},
		{"an unknown state word", `"dateStates": {"a.b.c": "open"}`, "write null or not-fetched"},
		{"a state without an attribute", `"dateStates": {"": "null"}`, "empty Attribute"},
		{"a value and a state for one attribute", `"dates": {"a.b.c": "2026-04-01"}, "dateStates": {"a.b.c": "null"}`, "both a value and the state"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseValidationFixture([]byte(`{"numbers": {}, "relationKeys": [], ` + tc.members + `}`))
			if err == nil || !strings.Contains(err.Error(), tc.mention) {
				t.Fatalf("got=%+v err=%v, want an error mentioning %q", got, err, tc.mention)
			}
		})
	}
}

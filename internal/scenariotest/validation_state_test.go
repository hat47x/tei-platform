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

package scenariotest

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const good = `{"schemaVersion":"tei.driver-scenario/v0","id":"order-entry-smoke","application":"order_entry","steps":[{"id":"browse","operation":"resolve-pattern","pattern":"order_entry.recent_orders","expectKind":"browse","expectData":"sales.order","expectQuery":"order_entry.recent_orders"}]}`

func TestGenerateIsDeterministicAndComplete(t *testing.T) {
	s, e := Parse([]byte(good))
	if e != nil { t.Fatal(e) }
	a, e := Generate(s, []byte(good), "fixture.json", "reference/workspace")
	if e != nil { t.Fatal(e) }
	b, e := Generate(s, []byte(good), "fixture.json", "reference/workspace")
	if e != nil { t.Fatal(e) }
	if !reflect.DeepEqual(a, b) || len(a.Commands) != len(s.Steps) { t.Fatal("non-reproducible or incomplete generation") }
	if got := a.Commands[0].Argv; !reflect.DeepEqual(got, []string{"tei", "scenario", "run", "-scenario", "fixture.json", "-workspace", "reference/workspace", "-step", "browse"}) { t.Fatalf("argv=%q", got) }
	if len(a.ScenarioSHA256) != 64 { t.Fatal("missing source digest") }
}
func TestRejectUnexpectedOrIncompleteScenario(t *testing.T) {
	negatives := []string{
		`{"schemaVersion":"tei.driver-scenario/v0","id":"x","application":"a","steps":[],"secret":"x"}`,
		`{"schemaVersion":"tei.driver-scenario/v0","id":"x","application":"a","steps":[]}`,
		strings.Replace(good, `"resolve-pattern"`, `"invoke-command"`, 1),
		strings.Replace(good, `"expectQuery":"order_entry.recent_orders"`, `"expectQuery":""`, 1),
		good + "{}",
	}
	for _, data := range negatives { if _, err := Parse([]byte(data)); err == nil { t.Fatalf("accepted invalid %q", data) } }
}
func TestScenarioEditChangesDigest(t *testing.T) {
	s, _ := Parse([]byte(good))
	p, _ := Generate(s, []byte(good), "a", "b")
	changed := strings.Replace(good, "order-entry-smoke", "order-entry-updated", 1)
	s2, _ := Parse([]byte(changed))
	q, _ := Generate(s2, []byte(changed), "a", "b")
	if p.ScenarioSHA256 == q.ScenarioSHA256 { t.Fatal("stale Scenario not detected") }
	_, e := json.Marshal(p)
	if e != nil { t.Fatal(e) }
}

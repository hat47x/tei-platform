package provideroverrideupdate

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func loadFixture(t *testing.T, name string) Input {
	t.Helper()
	input, err := LoadInput(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("LoadInput(%s): %v", name, err)
	}
	return input
}

func byID(report Report) map[string]ElementResult {
	out := make(map[string]ElementResult, len(report.Elements))
	for _, element := range report.Elements {
		out[element.ID] = element
	}
	return out
}

func stateElement(state State, id string) (Element, bool) {
	for _, element := range state.Elements {
		if element.ID == id {
			return element, true
		}
	}
	return Element{}, false
}

func minimalInput() Input {
	return Input{
		Schema: InputSchema,
		Provider: Provider{
			ID:          "std.message",
			FromVersion: "1.0.0",
			ToVersion:   "1.1.0",
			Base: []Element{
				{ID: "a", Kind: "message", Fields: map[string]string{"text": "A", "description": "desc A", "x": "10"}},
				{ID: "b", Kind: "message", Fields: map[string]string{"text": "B", "description": "desc B", "x": "20"}},
			},
			Next: []Element{
				{ID: "a", Kind: "message", Fields: map[string]string{"text": "A", "description": "desc A", "x": "10"}},
				{ID: "b", Kind: "message", Fields: map[string]string{"text": "B", "description": "desc B", "x": "20"}},
			},
		},
		Case: Case{BaseVersion: "1.0.0"},
	}
}

func TestApplicableOverrideIsKeptAndDoesNotBlock(t *testing.T) {
	result := Plan(loadFixture(t, "input-applicable.json"))
	if result.Report.Status != ClassificationApplicable {
		t.Fatalf("status = %q, want %q; report=%+v", result.Report.Status, ClassificationApplicable, result.Report)
	}
	if result.Report.Blocked {
		t.Fatal("applicable plan must not be blocked")
	}
	if result.After == nil {
		t.Fatal("applicable plan must have an After state")
	}
	submitted, ok := stateElement(*result.After, "std.message.order.submitted")
	if !ok {
		t.Fatal("submitted element missing from After")
	}
	if submitted.Fields["text"] != "受注を受け付けました" {
		t.Errorf("case override was lost: %q", submitted.Fields["text"])
	}
	if submitted.Fields["description"] != "Order has been submitted" {
		t.Errorf("provider change was not applied: %q", submitted.Fields["description"])
	}
	cancelled := byID(result.Report)["std.message.order.cancelled"]
	if cancelled.Classification != ClassificationUnchanged {
		t.Errorf("cancelled = %q, want %q", cancelled.Classification, ClassificationUnchanged)
	}
}

func TestConflictWhenProviderChangesOverriddenField(t *testing.T) {
	result := Plan(loadFixture(t, "input-conflict.json"))
	if result.Report.Status != ClassificationConflict {
		t.Fatalf("status = %q, want %q", result.Report.Status, ClassificationConflict)
	}
	if !result.Report.Blocked {
		t.Fatal("conflict plan must be blocked")
	}
	if result.After != nil {
		t.Fatal("blocked plan must not carry an After state")
	}
	cancelled := byID(result.Report)["std.message.order.cancelled"]
	if cancelled.Classification != ClassificationConflict {
		t.Fatalf("cancelled = %q, want conflict", cancelled.Classification)
	}
	if !contains(cancelled.Reasons, "provider-and-case-changed-field:text") {
		t.Errorf("reasons = %v", cancelled.Reasons)
	}
}

func TestOverrideOnAnotherFieldIsApplicable(t *testing.T) {
	input := minimalInput()
	input.Provider.Next[0].Fields["description"] = "desc A changed"
	input.Case.Overrides = []Override{{Element: "a", Fields: map[string]string{"text": "case A"}}}
	result := Plan(input)
	a := byID(result.Report)["a"]
	if a.Classification != ClassificationApplicable {
		t.Fatalf("a = %q, want applicable; reasons=%v", a.Classification, a.Reasons)
	}
	if !contains(a.Reasons, "override-kept-on-other-field") {
		t.Errorf("reasons = %v", a.Reasons)
	}
}

func TestCaseOnlyAdditionIsUnchanged(t *testing.T) {
	input := minimalInput()
	input.Case.Additions = []Element{{ID: "case.only", Kind: "message", Fields: map[string]string{"text": "added"}}}
	result := Plan(input)
	caseOnly := byID(result.Report)["case.only"]
	if caseOnly.Classification != ClassificationUnchanged {
		t.Fatalf("case.only = %q, want unchanged", caseOnly.Classification)
	}
	if result.After == nil {
		t.Fatal("After missing")
	}
	if _, ok := stateElement(*result.After, "case.only"); !ok {
		t.Fatal("case addition missing from After")
	}
}

func TestCaseAdditionCollidingWithProviderAdditionIsDetected(t *testing.T) {
	input := minimalInput()
	input.Provider.Next = append(input.Provider.Next, Element{ID: "new.element", Kind: "message", Fields: map[string]string{"text": "provider"}})
	input.Case.Additions = []Element{{ID: "new.element", Kind: "message", Fields: map[string]string{"text": "case"}}}
	result := Plan(input)
	if result.Report.Status != ClassificationConflict || !result.Report.Blocked {
		t.Fatalf("status = %q blocked=%v, want conflict and blocked", result.Report.Status, result.Report.Blocked)
	}
	if len(result.Report.NameCollisions) != 1 {
		t.Fatalf("nameCollisions = %#v", result.Report.NameCollisions)
	}
	if got := result.Report.NameCollisions[0]; got.ID != "new.element" || got.Kind != "case-addition-vs-provider-addition" {
		t.Fatalf("collision = %#v", got)
	}
	if byID(result.Report)["new.element"].Classification != ClassificationConflict {
		t.Error("colliding element must be conflict")
	}
}

func TestCaseAdditionCollidingWithExistingProviderElementIsDetected(t *testing.T) {
	input := minimalInput()
	input.Case.Additions = []Element{{ID: "a", Kind: "message", Fields: map[string]string{"text": "case"}}}
	result := Plan(input)
	if len(result.Report.NameCollisions) != 1 || result.Report.NameCollisions[0].Kind != "case-addition-vs-provider-element" {
		t.Fatalf("nameCollisions = %#v", result.Report.NameCollisions)
	}
	if result.Report.Status != ClassificationConflict {
		t.Fatalf("status = %q, want conflict", result.Report.Status)
	}
}

func TestDisplayOnlyChangeIsNotAConflict(t *testing.T) {
	input := minimalInput()
	input.Provider.Next[0].Fields["x"] = "99"
	input.DisplayOnlyFields = []string{"x", "y"}
	input.Case.Overrides = []Override{{Element: "a", Fields: map[string]string{"text": "case A"}}}
	result := Plan(input)
	a := byID(result.Report)["a"]
	if a.Classification != ClassificationUnchanged {
		t.Fatalf("a = %q, want unchanged; reasons=%v", a.Classification, a.Reasons)
	}
	if !contains(a.Reasons, "display-only-change") {
		t.Errorf("reasons = %v", a.Reasons)
	}
	if result.Report.Status != ClassificationUnchanged {
		t.Errorf("status = %q, want unchanged", result.Report.Status)
	}
}

func TestRemovedOverriddenElementIsConflict(t *testing.T) {
	input := minimalInput()
	input.Provider.Next = input.Provider.Next[:1] // drop b
	input.Case.Overrides = []Override{{Element: "b", Fields: map[string]string{"text": "case B"}}}
	result := Plan(input)
	b := byID(result.Report)["b"]
	if b.Classification != ClassificationConflict {
		t.Fatalf("b = %q, want conflict", b.Classification)
	}
	if !contains(b.Reasons, "provider-removed-overridden-element") {
		t.Errorf("reasons = %v", b.Reasons)
	}
}

func TestBaseVersionMismatchIsUnresolved(t *testing.T) {
	input := minimalInput()
	input.Case.BaseVersion = "0.9.0"
	result := Plan(input)
	if result.Report.Status != ClassificationUnresolved || !result.Report.Blocked {
		t.Fatalf("status = %q blocked=%v, want unresolved and blocked", result.Report.Status, result.Report.Blocked)
	}
	if len(result.Report.Problems) == 0 {
		t.Fatal("expected a version mismatch problem")
	}
}

func TestDuplicateProviderElementIDIsUnresolved(t *testing.T) {
	input := minimalInput()
	input.Provider.Base = append(input.Provider.Base, Element{ID: "a", Kind: "message", Fields: map[string]string{"text": "dup"}})
	result := Plan(input)
	if result.Report.Status != ClassificationUnresolved {
		t.Fatalf("status = %q, want unresolved", result.Report.Status)
	}
	if !contains(result.Report.Problems, "provider base has duplicate element id \"a\"") {
		t.Errorf("problems = %v", result.Report.Problems)
	}
}

func TestRollbackRestoresPreUpdateState(t *testing.T) {
	input := loadFixture(t, "input-applicable.json")
	result := Plan(input)
	if result.After == nil {
		t.Fatal("expected an After state")
	}
	before, _ := stateElement(result.Before, "std.message.order.submitted")
	after, _ := stateElement(*result.After, "std.message.order.submitted")
	if before.Fields["description"] != "Submitted order" {
		t.Errorf("Before provider version leaked: %q", before.Fields["description"])
	}
	if after.Fields["description"] != "Order has been submitted" {
		t.Errorf("After provider version missing: %q", after.Fields["description"])
	}
	if result.Before.ProviderVersion != "1.0.0" || result.After.ProviderVersion != "1.1.0" {
		t.Errorf("provider versions: before=%q after=%q", result.Before.ProviderVersion, result.After.ProviderVersion)
	}
	rolledBack := Rollback(result)
	if !reflect.DeepEqual(rolledBack, result.Before) {
		t.Fatalf("rollback = %#v, want %#v", rolledBack, result.Before)
	}
	submitted, _ := stateElement(rolledBack, "std.message.order.submitted")
	if submitted.Fields["text"] != "受注を受け付けました" {
		t.Errorf("rollback lost the case override: %q", submitted.Fields["text"])
	}
}

func TestBlockedPlanHasNoAfter(t *testing.T) {
	result := Plan(loadFixture(t, "input-conflict.json"))
	if result.After != nil {
		t.Fatal("blocked plan carried an After state")
	}
	if !result.Report.Blocked {
		t.Fatal("expected Blocked")
	}
}

func TestPlanDoesNotMutateInput(t *testing.T) {
	input := loadFixture(t, "input-applicable.json")
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var original Input
	if err := json.Unmarshal(raw, &original); err != nil {
		t.Fatal(err)
	}
	_ = Plan(input)
	if !reflect.DeepEqual(input, original) {
		t.Fatal("Plan changed the input")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

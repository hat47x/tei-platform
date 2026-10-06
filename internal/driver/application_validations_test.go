package driver_test

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func validationWorld(declared ...string) ir.World {
	app := application.Application{ID: "app_a"}
	for _, id := range declared {
		app.Validations = append(app.Validations, core.CanonicalID(id))
	}
	return ir.World{
		Applications: []application.Application{app, {ID: "app_empty"}},
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Validations: []semantic.Validation{
				{ID: "sales.a_rule"},
				{ID: "sales.b_rule"},
				{ID: "sales.c_rule"},
			},
		}},
	}
}

func TestResolveApplicationValidationsReturnsTheDeclaredRulesInDeclarationOrder(t *testing.T) {
	// The declared order is deliberately not the order of the semantic source.
	resolver := driver.NewResolver(validationWorld("sales.c_rule", "sales.a_rule"))

	rules, err := resolver.ResolveApplicationValidations("app_a")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || rules[0].ID != "sales.c_rule" || rules[1].ID != "sales.a_rule" {
		t.Fatalf("rules=%+v", rules)
	}
	// An undeclared rule of the same namespace is not selected by inference.
	for _, rule := range rules {
		if rule.ID == "sales.b_rule" {
			t.Fatalf("a rule that was not declared was selected: %+v", rules)
		}
	}
}

func TestResolveApplicationValidationsDoesNotReadAnEmptySelectionAsSatisfied(t *testing.T) {
	resolver := driver.NewResolver(validationWorld("sales.a_rule"))

	rules, err := resolver.ResolveApplicationValidations("app_empty")
	if !errors.Is(err, driver.ErrApplicationValidationSelectionNotDeclared) {
		t.Fatalf("err=%v rules=%+v", err, rules)
	}
	if len(rules) != 0 {
		t.Fatalf("rules returned with the error: %+v", rules)
	}
}

func TestResolveApplicationValidationsFailsClosedOnUnknownDuplicateAndMissingApplication(t *testing.T) {
	if _, err := driver.NewResolver(validationWorld("sales.missing")).ResolveApplicationValidations("app_a"); err == nil ||
		errors.Is(err, driver.ErrApplicationValidationSelectionNotDeclared) ||
		!strings.Contains(err.Error(), "sales.missing") {
		t.Fatalf("unknown id: err=%v", err)
	}
	if _, err := driver.NewResolver(validationWorld("sales.a_rule", "sales.a_rule")).ResolveApplicationValidations("app_a"); err == nil ||
		!strings.Contains(err.Error(), "重複") {
		t.Fatalf("duplicate: err=%v", err)
	}
	if _, err := driver.NewResolver(validationWorld("sales.a_rule")).ResolveApplicationValidations("nope"); err == nil {
		t.Fatal("missing Application was resolved")
	}
}

// The Reference does not declare `uses validation` yet, so the real workspace must report that
// instead of guessing a list. This test is replaced when the Reference declares its selection.
func TestReferenceApplicationsDoNotYetDeclareAValidationSelection(t *testing.T) {
	resolver := driver.NewResolver(referenceWorld(t))
	for _, id := range []string{"order_entry", "repair_intake"} {
		if _, err := resolver.ResolveApplicationValidations(core.CanonicalID(id)); !errors.Is(err, driver.ErrApplicationValidationSelectionNotDeclared) {
			t.Fatalf("%s: err=%v", id, err)
		}
	}
}

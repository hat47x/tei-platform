package crossprojectextraction_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/plugin/transform"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// semanticValidationValues only encodes rules already selected by the Driver.
// Rule resolution, duplicate detection, and declaration order remain owned
// by driver.ResolveApplicationValidations, not by this Reference.
func semanticValidationValues(rules []semantic.Validation) (transform.Value, error) {
	values := make([]transform.Value, 0, len(rules))
	for _, rule := range rules {
		value, err := transform.Object(
			transform.Field{Name: "id", Value: transform.String(string(rule.ID))},
			transform.Field{Name: "kind", Value: transform.String(string(rule.Kind))},
			transform.Field{Name: "attribute", Value: transform.String(string(rule.Attribute))},
			transform.Field{Name: "relation", Value: transform.String(string(rule.Relation))},
		)
		if err != nil {
			return transform.Value{}, err
		}
		values = append(values, value)
	}
	return transform.Array(values...), nil
}

func TestActualOrderEntrySemanticValidationsProjectWithoutDuplicateRuleDefinitions(t *testing.T) {
	root := filepath.Join("..", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			t.Fatalf("reference workspace meaning is not valid: %#v", diagnostic)
		}
	}

	// The real command scenario already uses the same Driver resolver.
	// Do not introduce a second rule-selection mechanism in this Reference.
	rules, err := driver.NewResolver(loaded.World).ResolveApplicationValidations("order_entry")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := semanticValidationValues(rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Array) != 3 {
		t.Fatalf("order_entry selected %d validation rules, want 3", len(selected.Array))
	}

	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range rules {
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: string(rule.ID)})
		if len(refs) != 1 || refs[0].ArtifactPath != "semantic/sales/validation.tei" {
			t.Fatalf("validation %s is not traceable to its authoritative declaration: %#v", rule.ID, refs)
		}
	}
	appRefs := index.ReferencesFor(sourceindex.Subject{Kind: "application", ID: "order_entry"})
	if len(appRefs) != 1 || appRefs[0].ArtifactPath != "applications/order-entry/application.tei" {
		t.Fatalf("application source reference changed: %#v", appRefs)
	}

	rawSource, err := os.ReadFile("semantic-validation-projection.transform.json")
	if err != nil {
		t.Fatal(err)
	}
	source, err := transform.ParseSourceJSON(rawSource)
	if err != nil {
		t.Fatal(err)
	}
	// A source-controlled rule projection is not the evaluator binding plan
	// provided by tools/reference-semantic-extension-catalog.
	if source.Identity.ContractVersion != "tei.reference.application-validation-projection/v0" {
		t.Fatalf("unexpected projection contract: %s", source.Identity.ContractVersion)
	}
	evaluated, err := transform.Evaluate(source, map[string]transform.Value{"validations": selected})
	if err != nil {
		t.Fatal(err)
	}
	got, err := evaluated.Output.JSON()
	if err != nil {
		t.Fatal(err)
	}
	// Independent expected meaning: positive is NOT nonnegative;
	// relation-unique retains both its relation and target attribute.
	const expected = `{"format":"tei.reference.application-validation-projection/v0","rules":[{"id":"sales.order_line.quantity_positive","kind":"positive","attribute":"sales.order_line.quantity","relation":""},{"id":"sales.order.lines.line_number_unique","kind":"relation-unique","attribute":"sales.order_line.line_number","relation":"sales.order.lines"},{"id":"sales.order_line.line_number_positive","kind":"positive","attribute":"sales.order_line.line_number","relation":""}]}`
	if string(got) != expected {
		t.Fatalf("semantic projection changed authored meaning: got %s; want %s", got, expected)
	}
	again, err := transform.Evaluate(source, map[string]transform.Value{"validations": selected})
	if err != nil {
		t.Fatal(err)
	}
	second, err := again.Output.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != expected {
		t.Fatalf("same semantic input generated different output: %s", second)
	}
}

func referenceWithChangedApplication(t *testing.T, w ir.World, change func(*application.Application)) ir.World {
	t.Helper()
	out := w
	out.Applications = append([]application.Application(nil), w.Applications...)
	for i := range out.Applications {
		if out.Applications[i].ID == "order_entry" {
			change(&out.Applications[i])
			return out
		}
	}
	t.Fatal("order_entry was not found in the reference Workspace")
	return ir.World{}
}

func TestSemanticValidationProjectionReusesExistingFailClosedSelection(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		change func(*application.Application)
	}{
		{"unknown validation", func(a *application.Application) {
			a.Validations = []core.CanonicalID{"sales.unknown_validation"}
		}},
		{"duplicate reference", func(a *application.Application) {
			a.Validations = []core.CanonicalID{"sales.order_line.quantity_positive", "sales.order_line.quantity_positive"}
		}},
		{"empty selection", func(a *application.Application) {
			a.Validations = nil
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := referenceWithChangedApplication(t, loaded.World, tt.change)
			rules, err := driver.NewResolver(w).ResolveApplicationValidations("order_entry")
			if err == nil || len(rules) != 0 {
				t.Fatalf("invalid validation selection was accepted: rules=%#v err=%v", rules, err)
			}
			if tt.name == "empty selection" && !errors.Is(err, driver.ErrApplicationValidationSelectionNotDeclared) {
				t.Fatalf("empty selection returned a different failure: %v", err)
			}
		})
	}

	duplicated := loaded.World
	duplicated.Semantics = append([]semantic.Namespace(nil), loaded.World.Semantics...)
	for _, ns := range loaded.World.Semantics {
		if ns.ID == "sales" && len(ns.Validations) > 0 {
			duplicated.Semantics = append(duplicated.Semantics, semantic.Namespace{
				ID: "duplicate_probe", Validations: []semantic.Validation{ns.Validations[0]},
			})
			break
		}
	}
	if _, err := driver.NewResolver(duplicated).ResolveApplicationValidations("order_entry"); err == nil {
		t.Fatal("duplicate semantic validation definition was silently selected")
	}

	// Driver selects the referenced rules; cross-namespace permission comes
	// from existing semantic/application validation, not this projection.
	unscoped := referenceWithChangedApplication(t, loaded.World, func(a *application.Application) {
		a.SemanticUses = []core.CanonicalID{"customer"}
	})
	found := false
	for _, diagnostic := range validation.Validate(unscoped) {
		if diagnostic.Code == "APPLICATION_VALIDATION_SEMANTIC_NOT_USED" && diagnostic.Severity == validation.Error {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("application namespace-use violation was not rejected by canonical validation")
	}
}

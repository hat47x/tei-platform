package crossprojectextraction_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/plugin/transform"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// This read-only projection selects only rules named by an application.
// The canonical Semantic and Application objects belong to workspace.Load;
// this test does not parse .tei a second time or create new business rules.
func applicationSemanticRules(app application.Application, namespaces []semantic.Namespace) (transform.Value, error) {
	uses := make(map[core.CanonicalID]bool, len(app.SemanticUses))
	for _, nsID := range app.SemanticUses {
		uses[nsID] = true
	}
	type entry struct {
		namespace core.CanonicalID
		rule semantic.Validation
	}
	declared := make(map[core.CanonicalID]entry)
	for _, ns := range namespaces {
		for _, rule := range ns.Validations {
			if _, exists := declared[rule.ID]; exists {
				return transform.Value{}, fmt.Errorf("duplicate semantic validation %s", rule.ID)
			}
			declared[rule.ID] = entry{namespace: ns.ID, rule: rule}
		}
	}
	seen := make(map[core.CanonicalID]bool, len(app.Validations))
	rules := make([]transform.Value, 0, len(app.Validations))
	for _, id := range app.Validations {
		if seen[id] {
			return transform.Value{}, fmt.Errorf("duplicate application validation %s", id)
		}
		seen[id] = true
		entry, exists := declared[id]
		if !exists {
			return transform.Value{}, fmt.Errorf("application validation %s is not declared", id)
		}
		if !uses[entry.namespace] {
			return transform.Value{}, fmt.Errorf("application does not use namespace %s", entry.namespace)
		}
		rule := entry.rule
		if !rule.Kind.Valid() || rule.Attribute == "" {
			return transform.Value{}, fmt.Errorf("semantic validation %s is not a supported rule", id)
		}
		if (rule.Kind == semantic.ValidationPositive && rule.Relation != "") ||
			(rule.Kind == semantic.ValidationRelationUnique && rule.Relation == "") {
			return transform.Value{}, fmt.Errorf("semantic validation %s has inconsistent relation", id)
		}
		value, err := transform.Object(
			transform.Field{Name: "id", Value: transform.String(string(rule.ID))},
			transform.Field{Name: "kind", Value: transform.String(string(rule.Kind))},
			transform.Field{Name: "attribute", Value: transform.String(string(rule.Attribute))},
			transform.Field{Name: "relation", Value: transform.String(string(rule.Relation))},
		)
		if err != nil {
			return transform.Value{}, err
		}
		rules = append(rules, value)
	}
	return transform.Array(rules...), nil
}

func exactApplication(w ir.World, id core.CanonicalID) (application.Application, error) {
	var found application.Application
	count := 0
	for _, app := range w.Applications {
		if app.ID == id {
			found = app
			count++
		}
	}
	if count != 1 {
		return application.Application{}, fmt.Errorf("expected exactly one application %s; found %d", id, count)
	}
	return found, nil
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

	app, err := exactApplication(loaded.World, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	selected, err := applicationSemanticRules(app, loaded.World.Semantics)
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
	for _, id := range app.Validations {
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: string(id)})
		if len(refs) != 1 || refs[0].ArtifactPath != "semantic/sales/validation.tei" {
			t.Fatalf("validation %s is not traceable to its authoritative declaration: %#v", id, refs)
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
	evaluated, err := transform.Evaluate(source, map[string]transform.Value{"validations": selected})
	if err != nil {
		t.Fatal(err)
	}
	got, err := evaluated.Output.JSON()
	if err != nil {
		t.Fatal(err)
	}
	// Independently specified expected meaning: positive is NOT nonnegative,
	// and relation-unique retains both its relation and target attribute.
	const expected = `{"format":"tei.reference.semantic-validation-plan/v0","rules":[{"id":"sales.order_line.quantity_positive","kind":"positive","attribute":"sales.order_line.quantity","relation":""},{"id":"sales.order.lines.line_number_unique","kind":"relation-unique","attribute":"sales.order_line.line_number","relation":"sales.order.lines"},{"id":"sales.order_line.line_number_positive","kind":"positive","attribute":"sales.order_line.line_number","relation":""}]}`
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

func TestSemanticValidationProjectionRefusesMissingAmbiguousOrUnscopedRules(t *testing.T) {
	loaded, err := workspace.Load(filepath.Join("..", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	app, err := exactApplication(loaded.World, "order_entry")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		change func(*application.Application)
	}{
		{"unknown validation", func(a *application.Application) {
			a.Validations = []core.CanonicalID{"sales.unknown_validation"}
		}},
		{"duplicate reference", func(a *application.Application) {
			a.Validations = []core.CanonicalID{"sales.order_line.quantity_positive", "sales.order_line.quantity_positive"}
		}},
		{"namespace not used", func(a *application.Application) {
			a.SemanticUses = []core.CanonicalID{"customer"}
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			changed := app
			tt.change(&changed)
			if _, err := applicationSemanticRules(changed, loaded.World.Semantics); err == nil {
				t.Fatal("unresolved or unauthorized semantic validation was silently projected")
			}
		})
	}

	duplicated := append([]semantic.Namespace(nil), loaded.World.Semantics...)
	for i := range duplicated {
		if duplicated[i].ID == "sales" {
			duplicateNamespace := duplicated[i]
			duplicateNamespace.Validations = duplicated[i].Validations[:1]
			duplicated = append(duplicated, duplicateNamespace)
			break
		}
	}
	if _, err := applicationSemanticRules(app, duplicated); err == nil {
		t.Fatal("duplicate semantic definition was silently selected")
	}
}

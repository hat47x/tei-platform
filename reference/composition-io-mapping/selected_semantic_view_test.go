package iomapping

import (
	"reflect"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func selectedViewFixture(t *testing.T) (*composition.Registry, Plan, []semantic.Namespace) {
	t.Helper()
	loaded, err := workspace.Load("../workspace")
	if err != nil {
		t.Fatal(err)
	}
	registry, plan := fixture(t)
	plan.Slots[0].Outputs[0] = semanticIDField("value", "sales.order_id")
	plan.Slots[1].Inputs[0] = semanticIDField("value", "sales.order_id")
	selected := make([]semantic.Namespace, 0, len(loaded.World.Semantics))
	for _, ns := range loaded.World.Semantics {
		if ns.ID != "std" {
			selected = append(selected, ns)
		}
	}
	return registry, plan, selected
}

func TestSelectedSemanticViewRejectsUnknownAndMixedRefs(t *testing.T) {
	registry, plan, view := selectedViewFixture(t)
	prior := registry.Snapshot()
	ok := InspectWithSelectedSemanticView(registry, plan, view)
	if ok.Verdict != composition.ProposalCandidate ||
		!reflect.DeepEqual(ok.Order, []string{"producer", "consumer"}) {
		t.Fatalf("selected same-identity view should be a structural candidate: %+v", ok)
	}
	if !reflect.DeepEqual(prior, registry.Snapshot()) {
		t.Fatal("preflight changed the existing registry")
	}

	plan.Slots[1].Inputs[0].Type.SemanticRef = "customer.customer_id"
	if report := InspectWithSelectedSemanticView(registry, plan, view); report.Verdict != composition.ProposalConflict ||
		!has(report, "type-incompatible") || report.Order != nil {
		t.Fatalf("distinct domain aliases: %+v", report)
	}

	plan.Slots[1].Inputs[0].Type.SemanticRef = "sales.not_declared"
	plan.Slots[0].Outputs[0].Type.SemanticRef = "sales.not_declared"
	if report := InspectWithSelectedSemanticView(registry, plan, view); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-type-unknown") || report.Order != nil {
		t.Fatalf("unselected type must not become a candidate: %+v", report)
	}

	// A nominal mismatch involving an unselected type is not evidence of
	// a genuine incompatibility. Resolve the selected identities first.
	plan.Slots[0].Outputs[0].Type.SemanticRef = "sales.not_declared"
	plan.Slots[1].Inputs[0].Type.SemanticRef = "customer.customer_id"
	if report := InspectWithSelectedSemanticView(registry, plan, view); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-type-unknown") || report.Order != nil {
		t.Fatalf("unknown type identity must dominate nominal incompatibility: %+v", report)
	}

	plan.Slots[0].Outputs[0].Type.SemanticRef = ""
	plan.Slots[1].Inputs[0].Type.SemanticRef = ""
	if raw := Inspect(registry, plan); raw.Verdict != composition.ProposalCandidate {
		t.Fatalf("legacy E2 should remain unchanged: %+v", raw)
	}
	if report := InspectWithSelectedSemanticView(registry, plan, view); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-ref-required") || report.Order != nil {
		t.Fatalf("stricter selected Semantic view requires ref: %+v", report)
	}
}

func TestSelectedSemanticViewChecksPrimitiveAndDefinitionChanges(t *testing.T) {
	registry, plan, view := selectedViewFixture(t)
	plan.Slots[0].Outputs[0].Type.Name = "text"
	plan.Slots[1].Inputs[0].Type.Name = "text"
	if report := InspectWithSelectedSemanticView(registry, plan, view); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-primitive-label-mismatch") || report.Order != nil {
		t.Fatalf("same ID and wrong primitive must not pass: %+v", report)
	}

	plan.Slots[0].Outputs[0].Type.Name = "identifier"
	plan.Slots[1].Inputs[0].Type.Name = "identifier"
	changed := make([]semantic.Namespace, len(view))
	for i, ns := range view {
		changed[i] = ns
		changed[i].Types = append([]semantic.Type(nil), ns.Types...)
		for j := range changed[i].Types {
			if changed[i].Types[j].ID == "sales.order_id" {
				changed[i].Types[j].Underlying = "std.text"
			}
		}
	}
	if report := InspectWithSelectedSemanticView(registry, plan, changed); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-primitive-label-mismatch") || report.Order != nil {
		t.Fatalf("modified meaning with unchanged port declaration must not pass: %+v", report)
	}
}

func TestSelectedSemanticViewDoesNotResolveInvalidDeclarations(t *testing.T) {
	registry, plan, view := selectedViewFixture(t)
	if report := InspectWithSelectedSemanticView(registry, plan, nil); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-view-missing") || report.Order != nil {
		t.Fatalf("missing selected view: %+v", report)
	}
	// The standard prelude is injected by the Reference, never adopted from
	// arbitrary selected Namespace bytes.
	spoofed := append(append([]semantic.Namespace(nil), view...), semantic.Namespace{
		ID: "std",
		Types: []semantic.Type{{
			ID: "std.identifier", Kind: semantic.TypeAlias, Underlying: "std.text",
		}},
	})
	if report := InspectWithSelectedSemanticView(registry, plan, spoofed); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-standard-prelude-overridden") || report.Order != nil {
		t.Fatalf("caller-selected standard prelude must not override stdlib: %+v", report)
	}
	// Duplicate canonical namespace definitions must not use last-wins.
	invalid := append(append([]semantic.Namespace(nil), view...), view[0])
	if report := InspectWithSelectedSemanticView(registry, plan, invalid); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-view-invalid") || report.Order != nil {
		t.Fatalf("duplicate selected declaration: %+v", report)
	}
	// The same test also guards against an unknown underlying type.
	invalid = make([]semantic.Namespace, len(view))
	for i, ns := range view {
		invalid[i] = ns
		invalid[i].Types = append([]semantic.Type(nil), ns.Types...)
		for j := range invalid[i].Types {
			if invalid[i].Types[j].ID == core.CanonicalID("sales.order_id") {
				invalid[i].Types[j].Underlying = "sales.missing_base"
			}
		}
	}
	if report := InspectWithSelectedSemanticView(registry, plan, invalid); report.Verdict != composition.ProposalUnknown ||
		!has(report, "selected-semantic-view-invalid") || report.Order != nil {
		t.Fatalf("invalid selected alias definition: %+v", report)
	}
}

// Cross-domain fixture: the same *nominal* TypeRef contract is usable by
 // a file/ETL value edge and by a build artifact metadata edge. These are
 // synthetic ports, not actual file delivery or a TEI self-build run.
func TestSelectedSemanticPreludeIsReusableAcrossPipelineRoles(t *testing.T) {
	cases := []struct {
		name, fromName, toName string
		fromRef, toRef        core.CanonicalID
		want                  composition.ProposalVerdict
	}{
		{"file-bytes", "binary", "binary", "std.binary", "std.binary", composition.ProposalCandidate},
		{"etl-text", "text", "text", "std.text", "std.text", composition.ProposalCandidate},
		{"build-metadata", "json", "json", "std.json", "std.json", composition.ProposalCandidate},
		{"implicit-decode", "binary", "text", "std.binary", "std.text", composition.ProposalConflict},
		{"unmigrated-namespace", "binary", "binary", "tei.std.binary", "tei.std.binary", composition.ProposalUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			registry, plan, view := selectedViewFixture(t)
			plan.Slots[0].Outputs[0].Type = Type{Name: tc.fromName, Unit: "none", SemanticRef: tc.fromRef}
			plan.Slots[1].Inputs[0].Type = Type{Name: tc.toName, Unit: "none", SemanticRef: tc.toRef}
			report := InspectWithSelectedSemanticView(registry, plan, view)
			if report.Verdict != tc.want {
				t.Fatalf("%s: expected %s; got %+v", tc.name, tc.want, report)
			}
			if tc.want != composition.ProposalCandidate && len(report.Order) != 0 {
				t.Fatalf("%s: rejected/unresolved mapping emitted an execution order: %+v", tc.name, report)
			}
		})
	}
}

package iomapping

import (
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func semanticIDField(id string, ref core.CanonicalID) Field {
	f := field(id)
	f.Type = Type{Name: "identifier", Unit: "none", SemanticRef: ref}
	return f
}

// A primitive representation is not a proof of semantic identity. The actual
// aliases come from the existing reference Workspace, not a second type catalog.
func TestSelectedSemanticAliasesInReferenceWorkspace(t *testing.T) {
	loaded, err := workspace.Load("../workspace")
	if err != nil {
		t.Fatal(err)
	}
	definitions := map[core.CanonicalID]semantic.Type{}
	for _, ns := range loaded.World.Semantics {
		for _, typ := range ns.Types {
			definitions[typ.ID] = typ
		}
	}
	for _, ref := range []core.CanonicalID{"sales.order_id", "customer.customer_id"} {
		typ, ok := definitions[ref]
		if !ok || typ.Kind != semantic.TypeAlias || typ.Underlying != "std.identifier" {
			t.Fatalf("%s: expected an alias of std.identifier in the loaded workspace, got %+v", ref, typ)
		}
	}
	if typ, ok := definitions["std.identifier"]; !ok || typ.Kind != semantic.TypePrimitive {
		t.Fatalf("standard primitive unavailable: %+v", typ)
	}
}

func TestSemanticRefComparison(t *testing.T) {
	source := semanticIDField("value", "sales.order_id")
	target := semanticIDField("value", "sales.order_id")
	if got := Compare(source, target); got != Identical {
		t.Fatalf("same semantic type ref: %s", got)
	}
	target.Type.SemanticRef = "customer.customer_id"
	if got := Compare(source, target); got != Incompatible {
		t.Fatalf("aliases sharing std.identifier must not coerce: %s", got)
	}
	target.Type.SemanticRef = ""
	if got := Compare(source, target); got != Unknown {
		t.Fatalf("ref-backed to legacy nominal input should remain unresolved: %s", got)
	}
	target.Type.SemanticRef = "sales.order_id"
	target.Type.Name = "text"
	if got := Compare(source, target); got != Unknown {
		t.Fatalf("same ref with contradictory representations requires validation: %s", got)
	}
	target.Type.Name = "identifier"
	target.Type.SemanticRef = "Sales.OrderId"
	if got := Compare(source, target); got != Unknown {
		t.Fatalf("invalid canonical reference must not be considered compatible: %s", got)
	}
}

func TestSemanticRefMappingUsesExistingInspection(t *testing.T) {
	registry, plan := fixture(t)
	plan.Slots[0].Outputs[0] = semanticIDField("value", "sales.order_id")
	plan.Slots[1].Inputs[0] = semanticIDField("value", "sales.order_id")
	before := registry.Snapshot()

	if got := Inspect(registry, plan); got.Verdict != composition.ProposalCandidate || len(got.Order) != 2 {
		t.Fatalf("same type-ref should retain a candidate data ordering: %+v", got)
	}
	plan.Slots[1].Inputs[0].Type.SemanticRef = "customer.customer_id"
	if got := Inspect(registry, plan); got.Verdict != composition.ProposalConflict || !has(got, "type-incompatible") || len(got.Order) != 0 {
		t.Fatalf("different domain aliases must reject the mapping without an order: %+v", got)
	}
	plan.Slots[1].Inputs[0].Type.SemanticRef = ""
	if got := Inspect(registry, plan); got.Verdict != composition.ProposalUnknown || !has(got, "type-undetermined") || len(got.Order) != 0 {
		t.Fatalf("missing semantic ref must remain unknown: %+v", got)
	}
	if after := registry.Snapshot(); after.Revision != before.Revision {
		t.Fatal("semantic ref inspection changed the provider registry")
	}
}

package languagereference

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func pinnedRelationPathFixture() ([]PinnedRelationStep, []OperatorContract) {
	// One semantic contract specializes across both Relation endpoints.
	// Source and target bindings must come from the canonical World, not
	// from a new operator identity or version per Relation.
	expand := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "expand-parametric-v1",
		Effect: "pure", InputType: expandSourceTypeVariable, OutputType: expandTargetTypeVariable,
	}
	return []PinnedRelationStep{
		{RelationID: "sales.order.customer", Call: PinnedCall{Contract: expand, InputType: "rows/sales.order"}},
		{RelationID: "customer.customer.addresses", Call: PinnedCall{Contract: expand, InputType: "rows/customer.customer"}},
	}, []OperatorContract{expand}
}

func TestInspectPinnedRelationPathPreservesEveryRelation(t *testing.T) {
	w := crossNamespaceRelationFixture()
	if ds := validation.Validate(w); validation.HasErrors(ds) {
		t.Fatalf("invalid canonical fixture: %#v", ds)
	}
	steps, catalog := pinnedRelationPathFixture()
	result, err := InspectPinnedRelationPath(w, "sales.order", steps, catalog)
	if err != nil {
		t.Fatal(err)
	}
	if result.Shape.SourceID != "sales.order" || result.Shape.TargetID != "address.address" ||
		result.Shape.Multiplicity != semantic.CardinalityMany ||
		result.OutputType != "rows/address.address" || len(result.Expansions) != 2 {
		t.Fatalf("wrong path shape: %#v", result)
	}
	for i, id := range []core.CanonicalID{"sales.order.customer", "customer.customer.addresses"} {
		if result.Expansions[i].RelationID != id || len(result.Expansions[i].Keys) != 1 ||
			result.Expansions[i].Operator != catalog[0] {
			t.Fatalf("step %d lost relation provenance, keys or reusable operator pin: %#v", i, result.Expansions[i])
		}
	}
	// The returned candidates must not alias mutable authored World data.
	w.Semantics[1].Data[0].Relations[0].Keys[0].TargetAttribute = "address.changed"
	if result.Expansions[1].Keys[0].TargetAttribute != "address.address.customer_id" {
		t.Fatal("path candidate aliases the mutable World")
	}
}

func TestInspectPinnedRelationPathFailsClosedAtExactHop(t *testing.T) {
	for _, tc := range []struct {
		name     string
		change   func(*[]PinnedRelationStep, *[]OperatorContract)
		wantStep string
	}{
		{"missing first operator", func(s *[]PinnedRelationStep, c *[]OperatorContract) {
			*c = (*c)[1:]
		}, "step 0"},
		{"unknown second relation", func(s *[]PinnedRelationStep, c *[]OperatorContract) {
			(*s)[1].RelationID = "customer.unknown"
		}, "step 1"},
		{"wrong second input", func(s *[]PinnedRelationStep, c *[]OperatorContract) {
			(*s)[1].Call.InputType = "rows/sales.order"
		}, "step 1"},
		{"wrong second output", func(s *[]PinnedRelationStep, c *[]OperatorContract) {
			(*s)[1].Call.Contract.OutputType = "rows/customer.customer"
		}, "step 1"},
		{"effect escalation", func(s *[]PinnedRelationStep, c *[]OperatorContract) {
			(*s)[1].Call.Contract.Effect = "write"
		}, "step 1"},
		{"missing relation identity", func(s *[]PinnedRelationStep, c *[]OperatorContract) {
			(*s)[0].RelationID = ""
		}, "step 0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := crossNamespaceRelationFixture()
			steps, catalog := pinnedRelationPathFixture()
			tc.change(&steps, &catalog)
			result, err := InspectPinnedRelationPath(w, "sales.order", steps, catalog)
			if err == nil || !strings.Contains(err.Error(), tc.wantStep) {
				t.Fatalf("expected failure at %s: candidate=%#v err=%v", tc.wantStep, result, err)
			}
			if len(result.Expansions) != 0 || result.OutputType != "" {
				t.Fatalf("rejected path returned partial executable candidate: %#v", result)
			}
		})
	}
	w := crossNamespaceRelationFixture()
	steps, catalog := pinnedRelationPathFixture()
	if _, err := InspectPinnedRelationPath(w, "sales.order", nil, catalog); err == nil {
		t.Fatal("empty path accepted")
	}
	if _, err := InspectPinnedRelationPath(w, "", steps, catalog); err == nil {
		t.Fatal("empty source accepted")
	}
}

func TestInspectPinnedRelationPathRejectsInvalidLaterRelationKeys(t *testing.T) {
	w := crossNamespaceRelationFixture()
	// The first hop is sound; the second hop's target key is ill-typed.
	w.Semantics[2].Data[0].Attributes[0].TypeRef = "address.other"
	steps, catalog := pinnedRelationPathFixture()
	result, err := InspectPinnedRelationPath(w, "sales.order", steps, catalog)
	if err == nil || !strings.Contains(err.Error(), "step 1") ||
		!strings.Contains(err.Error(), "SEMANTIC_RELATION_KEY_TYPE_MISMATCH") {
		t.Fatalf("missing canonical diagnostic at second hop: %#v, %v", result, err)
	}
}

package languagereference

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/validation"
)

func TestWorldRelationAcrossNamespacesMatchesCanonicalValidator(t *testing.T) {
	w := crossNamespaceRelationFixture()
	if ds := validation.ValidateSemanticRelationKeys(w); len(ds) != 0 {
		t.Fatalf("canonical relation keys invalid: %#v", ds)
	}
	if err := ValidateRelationKeys(w, "sales.order", "sales.order.customer"); err != nil {
		t.Fatal(err)
	}
	pin := OperatorContract{
		ID: "std.op.expand", Version: "1", Fingerprint: "cross-namespace-v1",
		Effect: "pure", InputType: "rows/sales.order", OutputType: "rows/customer.customer",
	}
	out, err := ExpandRelationReference(w, "sales.order", "sales.order.customer",
		PinnedCall{Contract: pin, InputType: pin.InputType}, []OperatorContract{pin})
	if err != nil || out != pin.OutputType {
		t.Fatalf("cross-namespace expansion rejected: %q: %v", out, err)
	}
	shape, err := TraceRelationPath(w, "sales.order", []core.CanonicalID{
		"sales.order.customer", "customer.customer.addresses",
	})
	if err != nil || shape.TargetID != "address.address" || shape.Multiplicity != semantic.CardinalityMany {
		t.Fatalf("cross-namespace path %#v: %v", shape, err)
	}
}

func TestWorldRelationRejectsSameFailuresAsCanonicalValidator(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.World)
		code   string
	}{
		{"wrong cross-namespace key type", func(w *ir.World) {
			w.Semantics[1].Data[0].Attributes[0].TypeRef = "customer.other_id"
		}, "SEMANTIC_RELATION_KEY_TYPE_MISMATCH"},
		{"duplicate key pair", func(w *ir.World) {
			w.Semantics[0].Data[0].Relations[0].Keys = append(w.Semantics[0].Data[0].Relations[0].Keys,
				semantic.RelationKey{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.id"})
		}, "DUPLICATE_SEMANTIC_RELATION_KEY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := crossNamespaceRelationFixture()
			tc.mutate(&w)
			found := false
			for _, d := range validation.ValidateSemanticRelationKeys(w) {
				if d.Code == tc.code {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected canonical diagnostic %q", tc.code)
			}
			if err := ValidateRelationKeys(w, "sales.order", "sales.order.customer"); err == nil || !strings.Contains(err.Error(), tc.code) {
				t.Fatalf("reference diverged from canonical diagnostic: %v", err)
			}
		})
	}
}

func TestWorldRelationRejectsAmbiguousTarget(t *testing.T) {
	w := crossNamespaceRelationFixture()
	w.Semantics = append(w.Semantics, w.Semantics[1])
	if err := ValidateRelationKeys(w, "sales.order", "sales.order.customer"); err == nil {
		t.Fatal("ambiguous target accepted")
	}
}


// TestWorldRelationDoesNotAddStricterKeyRules ensures the reference does not
// invent a source-key uniqueness restriction absent from the canonical
// validator. Distinct target attributes remain independently mapped.
func TestWorldRelationDoesNotAddStricterKeyRules(t *testing.T) {
	w := crossNamespaceRelationFixture()
	w.Semantics[1].Data[0].Attributes = append(w.Semantics[1].Data[0].Attributes,
		semantic.Attribute{ID: "customer.customer.alt_id", TypeRef: "customer.customer_id"})
	w.Semantics[0].Data[0].Relations[0].Keys = append(w.Semantics[0].Data[0].Relations[0].Keys,
		semantic.RelationKey{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.alt_id"})
	if ds := validation.ValidateSemanticRelationKeys(w); len(ds) != 0 {
		t.Fatalf("canonical validator rejected distinct target keys: %#v", ds)
	}
	if err := ValidateRelationKeys(w, "sales.order", "sales.order.customer"); err != nil {
		t.Fatalf("reference must preserve canonical relation key semantics: %v", err)
	}
}

func crossNamespaceRelationFixture() ir.World {
	return ir.World{Semantics: []semantic.Namespace{
		{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order",
				Attributes: []semantic.Attribute{
					{ID: "sales.order.customer_id", TypeRef: "customer.customer_id"},
				},
				Relations: []semantic.Relation{{
					ID: "sales.order.customer", Target: "customer.customer",
					Cardinality: semantic.CardinalityOne,
					Keys: []semantic.RelationKey{{
						SourceAttribute: "sales.order.customer_id",
						TargetAttribute: "customer.customer.id",
					}},
				}},
			}},
		},
		{
			ID: "customer",
			Data: []semantic.Data{{
				ID: "customer.customer",
				Attributes: []semantic.Attribute{
					{ID: "customer.customer.id", TypeRef: "customer.customer_id"},
				},
				Relations: []semantic.Relation{{
					ID: "customer.customer.addresses", Target: "address.address",
					Cardinality: semantic.CardinalityMany,
					Keys: []semantic.RelationKey{{
						SourceAttribute: "customer.customer.id",
						TargetAttribute: "address.address.customer_id",
					}},
				}},
			}},
		},
		{
			ID: "address",
			Data: []semantic.Data{{
				ID: "address.address",
				Attributes: []semantic.Attribute{
					{ID: "address.address.customer_id", TypeRef: "customer.customer_id"},
				},
			}},
		},
	}}
}

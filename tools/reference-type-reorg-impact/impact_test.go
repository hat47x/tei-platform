package typereorg

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func world() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{{ID: "s",
			Types: []semantic.Type{
				{ID: "std.string", Symbol: "String", Kind: semantic.TypePrimitive},
				{ID: "s.cust_code", Symbol: "CustCode", Kind: semantic.TypeAlias, Underlying: "std.string"},
				{ID: "s.client_code", Symbol: "ClientCode", Kind: semantic.TypeAlias, Underlying: "std.string"},
				{ID: "s.item_code", Symbol: "ItemCode", Kind: semantic.TypeAlias, Underlying: "std.string"},
				{ID: "s.short_cust", Symbol: "ShortCust", Kind: semantic.TypeAlias, Underlying: "s.client_code"},
			},
			Data: []semantic.Data{
				{ID: "s.order", Symbol: "Order", Attributes: []semantic.Attribute{
					{ID: "s.order.cust", Symbol: "cust", TypeRef: "s.cust_code"},
					{ID: "s.order.item", Symbol: "item", TypeRef: "s.item_code"},
				}, CandidateKeys: []semantic.CandidateKey{{ID: "s.order.k", Symbol: "k",
					Attributes: []core.CanonicalID{"s.order.cust"}}},
					Relations: []semantic.Relation{{ID: "s.order.to_cust", Target: "s.customer",
						Keys: []semantic.RelationKey{{SourceAttribute: "s.order.cust", TargetAttribute: "s.customer.code"}}}},
				},
				{ID: "s.customer", Symbol: "Customer", Attributes: []semantic.Attribute{
					{ID: "s.customer.code", Symbol: "code", TypeRef: "s.client_code"},
				}},
			},
		}},
		Persistence: []persistence.Mapping{{ID: "m", SemanticData: "s.order", Columns: []persistence.ColumnMapping{
			{Attribute: "s.order.cust", Column: "c", Storage: persistence.StorageVarchar},
			{Attribute: "s.order.item", Column: "i", Storage: persistence.StorageInt64},
		}}},
	}
}

func codes(d []Diagnostic) map[string]bool {
	m := map[string]bool{}
	for _, x := range d {
		m[x.Code] = true
	}
	return m
}

func TestImpactListsUsesAndNeverApplies(t *testing.T) {
	imp, err := Analyze(world(), Merge{From: []string{"s.cust_code"}, Into: "s.client_code"})
	if err != nil {
		t.Fatal(err)
	}
	if !imp.ReviewRequired || imp.Applied {
		t.Fatalf("impact = %+v", imp)
	}
	if len(imp.Attributes) != 1 || imp.Attributes[0].Attribute != "s.order.cust" {
		t.Fatalf("attributes = %+v", imp.Attributes)
	}
	if len(imp.Keys) != 2 {
		t.Fatalf("keys = %+v", imp.Keys)
	}
}

func TestSameRepresentationDoesNotSettleMeaning(t *testing.T) {
	imp, _ := Analyze(world(), Merge{From: []string{"s.cust_code"}, Into: "s.client_code"})
	c := codes(imp.Diagnostics)
	if !c["symbol-differs-meaning-unconfirmed"] || !imp.ReviewRequired {
		t.Fatalf("diagnostics = %+v", imp.Diagnostics)
	}
	if c["underlying-representation-differs"] {
		t.Fatal("same underlying flagged")
	}
}

func TestExistingKeyMismatchIsReportedAndHidingIsFlagged(t *testing.T) {
	imp, _ := Analyze(world(), Merge{From: []string{"s.cust_code"}, Into: "s.client_code"})
	c := codes(imp.Diagnostics)
	if !c["relation-key-type-already-differs"] || !c["relation-key-difference-would-be-hidden"] {
		t.Fatalf("diagnostics = %+v", imp.Diagnostics)
	}
}

func TestAliasChainAndRepresentationChange(t *testing.T) {
	imp, _ := Analyze(world(), Merge{From: []string{"s.client_code"}, Into: "s.item_code"})
	if len(imp.AliasTypes) != 1 || imp.AliasTypes[0] != "s.short_cust" {
		t.Fatalf("alias types = %+v", imp.AliasTypes)
	}
}

func TestPhysicalStorageDifferenceIsSurfaced(t *testing.T) {
	imp, _ := Analyze(world(), Merge{From: []string{"s.cust_code", "s.item_code"}, Into: "s.client_code"})
	if !codes(imp.Diagnostics)["physical-storage-differs"] {
		t.Fatalf("diagnostics = %+v", imp.Diagnostics)
	}
}

func TestInvalidMerges(t *testing.T) {
	imp, _ := Analyze(world(), Merge{From: []string{"std.string", "s.cust_code", "missing"}, Into: "s.cust_code"})
	c := codes(imp.Diagnostics)
	for _, want := range []string{"source-is-primitive", "merge-into-itself", "unknown-source-type"} {
		if !c[want] {
			t.Fatalf("missing %s in %+v", want, imp.Diagnostics)
		}
	}
	if _, err := Analyze(world(), Merge{}); err == nil {
		t.Fatal("empty merge accepted")
	}
}

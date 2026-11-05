package generatorselection

import (
	"strings"
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

// #2585 PFC-08: two fields must not share one runtime input key.
func TestRecordProjectionFieldsUseUniqueInputKeys(t *testing.T) {
	contract := Contract{
		Schema: ContractSchema, ID: "sui.card.pfc_record", Version: "1", Status: "reference-local",
		Inputs: []InputSpec{
			{Attribute: "sui.card.text", Primitive: "std.text"},
			{Attribute: "sui.card.document_id", Primitive: "std.identifier"},
		},
		Output: TypeSpec{Primitive: "std.text"},
		Projection: &ProjectionSpec{Kind: "record", Fields: []ProjectionField{
			{Name: "text", Input: "shared", Attribute: "sui.card.text", Primitive: "std.text"},
			{Name: "documentId", Input: "shared", Attribute: "sui.card.document_id", Primitive: "std.identifier"},
		}},
	}
	err := validateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "input") {
		t.Fatalf("duplicate input key was accepted: %v", err)
	}
}

// #2585 PFC-08: projection fields are validated against Workspace Semantic
// Attributes directly, not only through the contract inputs.
func TestValidateAgainstWorkspaceChecksProjectionFields(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID: "test",
		Types: []semantic.Type{
			{ID: "std.text", Kind: semantic.TypePrimitive},
			{ID: "std.integer", Kind: semantic.TypePrimitive},
		},
		Data: []semantic.Data{{
			ID:     "test.card",
			Symbol: "Card",
			Attributes: []semantic.Attribute{
				{ID: "test.card.text", Symbol: "text", TypeRef: "std.text"},
				{ID: "test.card.count", Symbol: "count", TypeRef: "std.integer"},
			},
		}},
	}}}
	base := Contract{
		Schema: ContractSchema, ID: "test.record", Version: "1",
		Inputs: []InputSpec{
			{Attribute: "test.card.text", Primitive: "std.text"},
			{Attribute: "test.card.count", Primitive: "std.integer"},
		},
		Output: TypeSpec{Primitive: "std.text"},
		Projection: &ProjectionSpec{Kind: "record", Fields: []ProjectionField{
			{Name: "text", Input: "text", Attribute: "test.card.text", Primitive: "std.text"},
		}},
	}
	if err := validateAgainstWorkspace(base, world); err != nil {
		t.Fatalf("a resolvable projection field was rejected: %v", err)
	}

	missing := base
	missing.Projection = &ProjectionSpec{Kind: "record", Fields: []ProjectionField{
		{Name: "text", Input: "text", Attribute: "test.card.missing", Primitive: "std.text"},
	}}
	if err := validateAgainstWorkspace(missing, world); err == nil ||
		!strings.Contains(err.Error(), "not found in the Workspace") {
		t.Fatalf("a field with a missing workspace attribute was accepted: %v", err)
	}

	mismatched := base
	mismatched.Projection = &ProjectionSpec{Kind: "record", Fields: []ProjectionField{
		{Name: "text", Input: "text", Attribute: "test.card.count", Primitive: "std.text"},
	}}
	if err := validateAgainstWorkspace(mismatched, world); err == nil ||
		!strings.Contains(err.Error(), "resolves to std.integer") {
		t.Fatalf("a field with a mismatched workspace primitive was accepted: %v", err)
	}
}

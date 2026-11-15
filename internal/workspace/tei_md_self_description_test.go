package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/sourceindex"
	"tei/internal/validation"
)

// This is a bounded D1/D2 self-description witness: a TEI-authored Data
// definition describes the fields of a Validation declaration, and the real
// Workspace Reader admits it as ordinary canonical Data. It does not teach
// the Reader to interpret new Validation kinds or replace GeneratePositive's
// handwritten checks. Optional Relation is NOT a forbidden Relation contract.
func TestTEIValidationDeclarationSelfDescriptionIsOnlyData(t *testing.T) {
	root := filepath.Join("testdata", "tei-md-self-description")
	const sourcePath = "semantic/tei/validation-declaration.tei.md"
	const dataID = "tei.validation_declaration"

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("load TEI-authored Validation description: %v", err)
	}
	if diagnostics := validation.Validate(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("existing canonical Data validator rejected self-description: %+v", diagnostics)
	}
	if len(loaded.AuthoringSources) != 1 {
		t.Fatalf("one selected description Source expected: %+v", loaded.AuthoringSources)
	}
	read := loaded.AuthoringSources[0]
	if read.Path != sourcePath || read.Language.LanguageID != "tei.authoring" ||
		read.Language.ContractVersion != "v1" ||
		read.Language.ImplementationID != "tei.builtin.authoring-markdown" ||
		read.Regions != 1 || read.Definitions != 1 {
		t.Fatalf("a non-selected or incomplete Reader parsed self-description: %+v", read)
	}

	descriptor, ok := dataOf(loaded, dataID)
	if !ok || descriptor.ID != dataID || len(descriptor.Attributes) != 4 {
		t.Fatalf("self-described Data is absent or has unexpected shape: %+v, found=%v", descriptor, ok)
	}
	// Fixed expectations are independent of the table Reader: absence of a
	// Relation is not enforced merely because its field is optional.
	expected := []struct {
		id       core.CanonicalID
		typeRef  core.CanonicalID
		required bool
		cell     []byte
	}{
		{"tei.validation_declaration.rule_id", "std.identifier", true, []byte("ruleId |")},
		{"tei.validation_declaration.rule_kind", "std.text", true, []byte("ruleKind |")},
		{"tei.validation_declaration.attribute_id", "std.identifier", true, []byte("attributeId |")},
		{"tei.validation_declaration.relation_id", "std.identifier", false, []byte("relationId |")},
	}
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range expected {
		got := descriptor.Attributes[i]
		if got.ID != want.id || got.TypeRef != want.typeRef || got.Required != want.required {
			t.Fatalf("attribute %d=%+v, expected id=%s type=%s required=%v",
				i, got, want.id, want.typeRef, want.required)
		}
		refs := index.ReferencesFor(sourceindex.Subject{
			Kind: "semantic-attribute", ID: string(want.id),
		})
		if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition ||
			refs[0].ArtifactPath != sourcePath ||
			refs[0].Range.Start.Offset != bytes.Index(raw, want.cell) {
			t.Fatalf("attribute %s lacks the exact Source cell: %+v", want.id, refs)
		}
		if err := refs[0].Range.Validate(raw); err != nil {
			t.Fatalf("stale cell for %s: %v", want.id, err)
		}
	}

	// Describing the shape of a Validation is not declaring an executable
	// semantic.Validation, a public Capability, or a generated Go validator.
	dataRefs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-data", ID: dataID})
	if len(dataRefs) != 1 || dataRefs[0].Role != sourceindex.RoleDefinition ||
		dataRefs[0].ArtifactPath != sourcePath {
		t.Fatalf("the Data definition lost its Source: %+v", dataRefs)
	}
	for _, kind := range []string{"semantic-validation", "semantic-capability"} {
		refs := index.ReferencesFor(sourceindex.Subject{Kind: kind, ID: dataID})
		if len(refs) != 0 {
			t.Fatalf("self-description unexpectedly created %s %s: %+v", kind, dataID, refs)
		}
	}
	for _, namespace := range loaded.World.Semantics {
		if len(namespace.Validations) != 0 {
			t.Fatalf("a description table silently became executable Validation: %+v", namespace.Validations)
		}
	}
}

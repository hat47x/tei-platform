package operations

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/modelref"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

func inspectionCode(t *testing.T, err error, want string) {
	t.Helper()
	var failure *InspectionError
	if !errors.As(err, &failure) || failure.Code != want {
		t.Fatalf("inspection error=%v, want code=%s", err, want)
	}
}

func TestInspectCanonicalExactKindAndID(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{ID: "sales", Data: []semantic.Data{
			{ID: "same.id", Symbol: "canonical_data", Attributes: []semantic.Attribute{{ID: "same.id", Symbol: "same_attribute", TypeRef: "std.text"}}, Relations: []semantic.Relation{{ID: "same.id", Symbol: "same_relation"}}},
			{ID: "other.id", Symbol: "other_data"},
		}}},
		Applications: []application.Application{{ID: "same.id", Patterns: []application.Pattern{{ID: "same.id"}}}},
		Queries:      []query.EmbeddedSQL{{ID: "same.id", Source: "select 1"}},
	}
	check := func(kind string, expected any) {
		t.Helper()
		raw, err := inspectCanonical(world, sourceindex.Subject{Kind: kind, ID: "same.id"})
		if err != nil {
			t.Fatal(err)
		}
		want, err := json.Marshal(expected)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual([]byte(raw), want) {
			t.Fatalf("kind=%s got %s want %s", kind, raw, want)
		}
	}
	check("semantic-data", world.Semantics[0].Data[0])
	check("semantic-attribute", world.Semantics[0].Data[0].Attributes[0])
	check("semantic-relation", world.Semantics[0].Data[0].Relations[0])
	check("application", world.Applications[0])
	check("application-pattern", world.Applications[0].Patterns[0])
	check("query", world.Queries[0])

	_, err := inspectCanonical(world, sourceindex.Subject{Kind: "semantic-data", ID: "not-found"})
	inspectionCode(t, err, InspectionNotFound)
	_, err = inspectCanonical(world, sourceindex.Subject{Kind: "semantic-type", ID: "same.id"})
	inspectionCode(t, err, InspectionUnsupportedKind)
}

func TestInspectCanonicalDuplicateIdentityIsAmbiguous(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{ID: "sales", Data: []semantic.Data{{ID: "sales.order"}}},
		{ID: "another", Data: []semantic.Data{{ID: "sales.order"}}},
	}}
	_, err := inspectCanonical(world, sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
	inspectionCode(t, err, InspectionAmbiguous)
}

func TestInspectSubjectRejectsInvalidRequestBeforeWorkspaceLoad(t *testing.T) {
	for _, s := range []sourceindex.Subject{
		{Kind: "query"},
		{Kind: "", ID: "id"},
		{Kind: " query", ID: "id"},
		{Kind: "query", ID: " id "},
	} {
		_, err := InspectSubject(filepath.Join(t.TempDir(), "absent"), s)
		inspectionCode(t, err, InspectionInvalidSubject)
	}
	_, err := InspectSubject(filepath.Join(t.TempDir(), "absent"), sourceindex.Subject{Kind: "semantic-type", ID: "sales.order_id"})
	inspectionCode(t, err, InspectionUnsupportedKind)
}

func TestInspectSubjectReferenceWorkspaceUsesSourceIndex(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, subject := range []sourceindex.Subject{
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "semantic-attribute", ID: "sales.order.order_id"},
		{Kind: "semantic-relation", ID: "sales.order.customer"},
		{Kind: "application", ID: "order_entry"},
		{Kind: "application-pattern", ID: "order_entry.customer_reference"},
		{Kind: "query", ID: "order_entry.recent_orders"},
	} {
		result, err := InspectSubject(root, subject)
		if err != nil {
			t.Fatalf("inspect %v: %v", subject, err)
		}
		if result.Subject != subject || result.Effect != "read-only" ||
			result.SchemaVersion != "tei.operations.inspect/v0" || result.WorldFingerprint == "" ||
			result.SourceIndexFingerprint == "" ||
			result.SourceStatus != "indexed" || len(result.Sources.References) == 0 ||
			len(result.Sources.Artifacts) == 0 || !json.Valid(result.Canonical) {
			t.Fatalf("invalid inspection: %+v", result)
		}
		for _, reference := range result.Sources.References {
			if reference.Subject != subject {
				t.Fatalf("cross-subject source leaked: %+v", reference)
			}
		}
		for _, artifact := range result.Sources.Artifacts {
			if artifact.Fingerprint == "" {
				t.Fatalf("source lacks fingerprint: %+v", artifact)
			}
		}
	}
}

func TestInspectAttributeRejectsMissingAndDuplicateIdentities(t *testing.T) {
	selected := sourceindex.Subject{Kind: "semantic-attribute", ID: "shared.id"}
	world := ir.World{Semantics: []semantic.Namespace{{ID: "one", Data: []semantic.Data{{ID: "shared.id"}}}}}
	// A Data with the same ID is not an Attribute declaration.
	_, err := inspectCanonical(world, selected)
	inspectionCode(t, err, InspectionNotFound)
	world.Semantics[0].Data[0].Attributes = []semantic.Attribute{{ID: "shared.id", Symbol: "first"}}
	world.Semantics = append(world.Semantics, semantic.Namespace{ID: "two", Data: []semantic.Data{{
		ID: "other.data", Attributes: []semantic.Attribute{{ID: "shared.id", Symbol: "second"}},
	}}})
	_, err = inspectCanonical(world, selected)
	inspectionCode(t, err, InspectionAmbiguous)
}

func TestInspectAndTraceAttributeRejectDuplicateSourceDeclarations(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, root, "tei.workspace.yaml", "workspace: attributes\nsources:\n  - semantic\n")
	for _, name := range []string{"one.tei", "two.tei"} {
		writeWorkspaceFixture(t, root, filepath.Join("semantic", name),
			"namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n")
	}
	selected := sourceindex.Subject{Kind: "semantic-attribute", ID: "sales.order.order_id"}
	_, err := InspectSubject(root, selected)
	inspectionCode(t, err, InspectionAmbiguous)
	_, err = TraceWorkspace(root, modelref.Key{Scope: "reference", Subject: selected}, modelref.TraceOptions{Mode: "summary"})
	var trace *modelref.TraceError
	if !errors.As(err, &trace) || trace.Code != InspectionAmbiguous {
		t.Fatalf("trace hid duplicate Attribute declarations: %v", err)
	}
}

func TestInspectSubjectFileMoveDoesNotChangeCanonicalFingerprint(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace: inspect\nsources:\n  - semantic\n"), 0644); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(root, "semantic", "first.tei")
	second := filepath.Join(root, "semantic", "moved.tei")
	if err := os.WriteFile(first, []byte("namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	id := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	before, err := InspectSubject(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(first, second); err != nil {
		t.Fatal(err)
	}
	after, err := InspectSubject(root, id)
	if err != nil {
		t.Fatal(err)
	}
	if before.WorldFingerprint != after.WorldFingerprint || !reflect.DeepEqual(before.Canonical, after.Canonical) {
		t.Fatalf("source move changed canonical meaning: %s vs %s", before.WorldFingerprint, after.WorldFingerprint)
	}
	if before.SourceIndexFingerprint == after.SourceIndexFingerprint {
		t.Fatal("source move did not change Source Index fingerprint")
	}
	if len(before.Sources.References) == 0 || len(after.Sources.References) == 0 ||
		before.Sources.References[0].ArtifactPath == after.Sources.References[0].ArtifactPath {
		t.Fatalf("source paths did not follow move: before=%+v after=%+v", before.Sources.References, after.Sources.References)
	}
}

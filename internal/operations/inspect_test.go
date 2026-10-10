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
			{ID: "same.id", Symbol: "canonical_data"},
			{ID: "other.id", Symbol: "other_data"},
		}}},
		Applications: []application.Application{{ID: "same.id"}},
		Queries: []query.EmbeddedSQL{{ID: "same.id", Source: "select 1"}},
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
	check("application", world.Applications[0])
	check("query", world.Queries[0])

	_, err := inspectCanonical(world, sourceindex.Subject{Kind: "semantic-data", ID: "not-found"})
	inspectionCode(t, err, InspectionNotFound)
	_, err = inspectCanonical(world, sourceindex.Subject{Kind: "semantic-attribute", ID: "same.id"})
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
	_, err := InspectSubject(filepath.Join(t.TempDir(), "absent"), sourceindex.Subject{Kind: "semantic-attribute", ID: "sales.order.id"})
	inspectionCode(t, err, InspectionUnsupportedKind)
}

func TestInspectSubjectReferenceWorkspaceUsesSourceIndex(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, subject := range []sourceindex.Subject{
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "application", ID: "order_entry"},
		{Kind: "query", ID: "order_entry.recent_orders"},
	} {
		result, err := InspectSubject(root, subject)
		if err != nil {
			t.Fatalf("inspect %v: %v", subject, err)
		}
		if result.Subject != subject || result.Effect != "read-only" ||
			result.SchemaVersion != "tei.cli.inspect/v0" || result.WorldFingerprint == "" ||
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
	if len(before.Sources.References) == 0 || len(after.Sources.References) == 0 ||
		before.Sources.References[0].ArtifactPath == after.Sources.References[0].ArtifactPath {
		t.Fatalf("source paths did not follow move: before=%+v after=%+v", before.Sources.References, after.Sources.References)
	}
}

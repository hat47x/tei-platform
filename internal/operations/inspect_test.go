package operations

import (
	"bytes"
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
	_, err = inspectCanonical(world, sourceindex.Subject{Kind: "unknown-kind", ID: "same.id"})
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
	_, err := InspectSubject(filepath.Join(t.TempDir(), "absent"), sourceindex.Subject{Kind: "unknown-kind", ID: "sales.order_id"})
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

func TestModelCatalogEntriesRoundTripThroughInspection(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	catalog, err := CatalogWorkspace(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Entries) == 0 {
		t.Fatal("empty reference catalog")
	}
	for _, entry := range catalog.Entries {
		got, err := InspectSubject(root, entry.Subject)
		if err != nil {
			t.Fatalf("catalog entry %v cannot be inspected: %v", entry.Subject, err)
		}
		if got.Subject != entry.Subject || got.Effect != "read-only" || got.WorldFingerprint != catalog.WorldFingerprint || got.SourceIndexFingerprint != catalog.SourceIndexFingerprint || got.SourceStatus != entry.SourceStatus {
			t.Fatalf("catalog/inspection snapshots differ for %v: %+v", entry.Subject, got)
		}
		var canonical struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(got.Canonical, &canonical); err != nil || canonical.ID != entry.Subject.ID {
			t.Fatalf("wrong canonical identity: %s / %v", got.Canonical, err)
		}
	}
}

func TestInspectionResolvesEveryCatalogKindAndRejectsDuplicateIdentity(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{ID: "same.id",
			Types: []semantic.Type{{ID: "same.id", Symbol: "alias", Kind: semantic.TypeAlias, Underlying: "std.text"}},
			Data: []semantic.Data{{ID: "same.id", Symbol: "record",
				Attributes:             []semantic.Attribute{{ID: "same.id", Symbol: "attribute", TypeRef: "std.text"}},
				Relations:              []semantic.Relation{{ID: "same.id", Symbol: "relation"}},
				CandidateKeys:          []semantic.CandidateKey{{ID: "same.id", Symbol: "key"}},
				FunctionalDependencies: []semantic.FunctionalDependency{{ID: "same.id", Symbol: "dependency"}},
			}},
			Capabilities: []semantic.Capability{{ID: "same.id", Symbol: "capability"}},
			Validations:  []semantic.Validation{{ID: "same.id", Symbol: "validation", Kind: semantic.ValidationPositive}},
		}},
		Applications: []application.Application{{ID: "same.id", Patterns: []application.Pattern{{ID: "same.id"}}}},
		Queries:      []query.EmbeddedSQL{{ID: "same.id", Source: "select 1"}},
	}
	ns := world.Semantics[0]
	data := ns.Data[0]
	expected := map[string]any{
		"semantic": ns, "semantic-type": ns.Types[0], "semantic-data": data,
		"semantic-attribute": data.Attributes[0], "semantic-relation": data.Relations[0],
		"semantic-candidate-key": data.CandidateKeys[0], "semantic-functional-dependency": data.FunctionalDependencies[0],
		"semantic-capability": ns.Capabilities[0], "semantic-validation": ns.Validations[0],
		"application": world.Applications[0], "application-pattern": world.Applications[0].Patterns[0], "query": world.Queries[0],
	}
	catalog := CatalogWorld(world, sourceindex.Index{}, "", "world", "source")
	if len(catalog.Entries) != len(expected) {
		t.Fatalf("missing catalog kinds: %+v", catalog.Entries)
	}
	// Duplicate every declaration in a second container; no parent or namespace
	// may act as an implicit discriminator for the public kind + exact ID key.
	duplicate := world
	duplicate.Semantics = append(append([]semantic.Namespace(nil), world.Semantics...), ns)
	duplicate.Applications = append(append([]application.Application(nil), world.Applications...), world.Applications[0])
	duplicate.Queries = append(append([]query.EmbeddedSQL(nil), world.Queries...), world.Queries[0])
	for _, entry := range catalog.Entries {
		subject := entry.Subject
		raw, err := inspectCanonical(world, subject)
		if err != nil {
			t.Fatalf("%v: %v", subject, err)
		}
		want, err := json.Marshal(expected[subject.Kind])
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(raw, want) {
			t.Fatalf("kind %s returned another kind with the same ID: %s", subject.Kind, raw)
		}
		_, err = inspectCanonical(duplicate, subject)
		inspectionCode(t, err, InspectionAmbiguous)
		subject.ID = "missing.id"
		_, err = inspectCanonical(world, subject)
		inspectionCode(t, err, InspectionNotFound)
	}
}

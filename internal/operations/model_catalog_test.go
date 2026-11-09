package operations

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

func TestCatalogWorldEnumeratesTypedSemanticAndApplicationSubjects(t *testing.T) {
	world := ir.World{
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Data: []semantic.Data{{
				ID: "sales.order", Symbol: "order",
				Attributes:             []semantic.Attribute{{ID: "sales.order.id", Symbol: "id", TypeRef: "std.text"}},
				Relations:              []semantic.Relation{{ID: "sales.order.customer", Symbol: "customer", Target: "sales.customer"}},
				CandidateKeys:          []semantic.CandidateKey{{ID: "sales.order.key", Symbol: "business_id"}},
				FunctionalDependencies: []semantic.FunctionalDependency{{ID: "sales.order.dep", Symbol: "fd"}},
			}},
			Capabilities: []semantic.Capability{{ID: "sales.order.create", Symbol: "create"}},
			Validations:  []semantic.Validation{{ID: "sales.order.positive", Symbol: "positive", Kind: semantic.ValidationPositive}},
		}},
		Applications: []application.Application{{ID: "order_entry", Patterns: []application.Pattern{{
			ID: "order_entry.list", Kind: application.PatternBrowse, Data: "sales.order",
		}}}},
	}
	idx := sourceindex.Index{References: []sourceindex.Reference{{
		Subject:      sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"},
		ArtifactPath: "semantic/sales/model.tei", Role: sourceindex.RoleDefinition,
	}}}
	got := CatalogWorld(world, idx, "", "world", "source")
	if got.Effect != "read-only" || got.SchemaVersion != "tei.operations.model-catalog/v0" ||
		got.ModelRevision == "" || got.WorldFingerprint != "world" || got.SourceIndexFingerprint != "source" {
		t.Fatalf("invalid catalog header: %+v", got)
	}
	found := map[sourceindex.Subject]ModelCatalogEntry{}
	for _, entry := range got.Entries {
		found[entry.Subject] = entry
	}
	need := []sourceindex.Subject{
		{Kind: "semantic", ID: "sales"},
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "semantic-attribute", ID: "sales.order.id"},
		{Kind: "semantic-relation", ID: "sales.order.customer"},
		{Kind: "semantic-candidate-key", ID: "sales.order.key"},
		{Kind: "semantic-functional-dependency", ID: "sales.order.dep"},
		{Kind: "semantic-capability", ID: "sales.order.create"},
		{Kind: "semantic-validation", ID: "sales.order.positive"},
		{Kind: "application", ID: "order_entry"},
		{Kind: "application-pattern", ID: "order_entry.list"},
	}
	for _, subject := range need {
		if _, ok := found[subject]; !ok {
			t.Fatalf("missing %v in %+v", subject, got.Entries)
		}
	}
	if found[sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}].SourceStatus != "indexed" {
		t.Fatal("source-backed declaration must be indexed")
	}
	rel := found[sourceindex.Subject{Kind: "semantic-relation", ID: "sales.order.customer"}]
	if rel.Parent == nil || *rel.Parent != (sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}) ||
		rel.Detail != "sales.customer" || rel.SourceStatus != "not-indexed" {
		t.Fatalf("relation parent/detail/provenance incorrect: %+v", rel)
	}
	filtered := CatalogWorld(world, idx, "sales", "world", "source")
	if len(filtered.Entries) != 8 || filtered.NamespaceFilter != "sales" {
		t.Fatalf("namespace filtering leaked app entries: %+v", filtered)
	}
	for _, entry := range filtered.Entries {
		if entry.Subject.Kind == "application" || entry.Subject.Kind == "application-pattern" {
			t.Fatalf("application leaked into namespace filter: %+v", entry)
		}
	}
}

func TestCatalogWorldDoesNotDiscardDuplicateIdentities(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{ID: "one", Data: []semantic.Data{{ID: "shared.id", Symbol: "first"}}},
		{ID: "two", Data: []semantic.Data{{ID: "shared.id", Symbol: "second"}}},
	}}
	got := CatalogWorld(world, sourceindex.Index{}, "", "x", "y")
	count := 0
	for _, entry := range got.Entries {
		if entry.Subject == (sourceindex.Subject{Kind: "semantic-data", ID: "shared.id"}) {
			count++
			if !entry.Ambiguous {
				t.Fatalf("duplicate identity silently accepted: %+v", entry)
			}
		}
	}
	if count != 2 {
		t.Fatalf("duplicate declarations were lost: %d", count)
	}
}

func TestCatalogWorkspaceUsesRealWorkspaceSnapshot(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	got, err := CatalogWorkspace(root, "sales")
	if err != nil {
		t.Fatal(err)
	}
	if got.ModelRevision == "" || got.WorldFingerprint == "" || got.SourceIndexFingerprint == "" {
		t.Fatalf("catalog is not bound to source snapshot: %+v", got)
	}
	found := false
	for _, entry := range got.Entries {
		if entry.Subject == (sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}) {
			found = true
			if entry.SourceStatus != "indexed" {
				t.Fatalf("source anchor missing: %+v", entry)
			}
		}
		if entry.Subject.Kind == "application" {
			t.Fatalf("namespace filter must not include application: %+v", entry)
		}
	}
	if !found {
		t.Fatal("source-owned sales.order is absent")
	}
}

func TestCatalogNamespaceFilterPreservesGlobalIdentityAmbiguity(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{ID: "one", Data: []semantic.Data{{ID: "shared.id", Symbol: "first"}}},
		{ID: "two", Data: []semantic.Data{{ID: "shared.id", Symbol: "second"}}},
	}}
	catalog := CatalogWorld(world, sourceindex.Index{}, "one", "world", "source")
	for _, entry := range catalog.Entries {
		if entry.Subject.Kind != "semantic-data" {
			continue
		}
		if entry.Symbol != "first" || !entry.Ambiguous {
			t.Fatalf("namespace filter hid an ambiguity outside its view: %+v", entry)
		}
		_, err := inspectCanonical(world, entry.Subject)
		inspectionCode(t, err, InspectionAmbiguous)
		return
	}
	t.Fatal("filtered Data missing")
}

func TestCatalogWorldReportsNamespaceMatch(t *testing.T) {
	world := ir.World{
		Semantics:    []semantic.Namespace{{ID: "sales"}},
		Applications: []application.Application{{ID: "order_entry"}},
		Queries:      []query.EmbeddedSQL{{ID: "sales.orders"}},
	}
	idx := sourceindex.Index{}
	for name, want := range map[string]struct {
		matched bool
		kind    string
	}{
		"":             {true, "all"},
		"sales":        {true, "namespace"},
		"order_entry":  {false, "application"},
		"sales.orders": {false, "query"},
		"no.such.ns":   {false, "unknown"},
	} {
		got := CatalogWorld(world, idx, name, "world", "source")
		if got.NamespaceMatched != want.matched || got.NamespaceMatchKind != want.kind {
			t.Errorf("namespace %q: matched=%t kind=%q, want %t/%q", name, got.NamespaceMatched, got.NamespaceMatchKind, want.matched, want.kind)
		}
	}
}

// TestCatalogEmptyWorkspaceSeparatesRevisionFromDeclarations pins the two
// answers an empty Workspace gives (#2822). The standard prelude is loaded, so
// entries are non-empty and the World fingerprint is not the empty digest; the
// managed Source set is empty, so ModelRevision is. Reading ModelRevision as
// "the model declares nothing" would be wrong, and reading the entries as
// "declarations were loaded" would be wrong in the other direction.
func TestCatalogEmptyWorkspaceSeparatesRevisionFromDeclarations(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{{
		ID:    "std",
		Types: []semantic.Type{{ID: "std.text", Symbol: "text"}},
	}}}
	got := CatalogWorld(world, sourceindex.Index{}, "", "world", "source")
	const emptyInput = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got.ModelRevision != emptyInput {
		t.Fatalf("empty managed Source revision=%q", got.ModelRevision)
	}
	if got.DeclarationCounts.Total != 0 {
		t.Fatalf("the standard prelude leaked into declaration counts: %+v", got.DeclarationCounts)
	}
	if len(got.Entries) == 0 {
		t.Fatal("prelude subjects must still be listed")
	}
	for _, entry := range got.Entries {
		if entry.Subject.ID != "std" && entry.Subject.ID != "std.text" {
			t.Fatalf("unexpected subject in a prelude-only catalog: %+v", entry.Subject)
		}
		if entry.SourceStatus != "not-indexed" {
			t.Fatalf("prelude subjects must not claim a Source anchor: %+v", entry)
		}
	}
}

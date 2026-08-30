package workspace_test

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestBuildSourceIndexKeepsSemanticSubjectIndependentFromPath(t *testing.T) {
	root := t.TempDir()
	oldPath := "semantic/order.tei"
	content := []byte(`namespace sales

data Order {
    attribute orderId type std.identifier
    relation customer to customer.customer one
}
`)
	writeSource(t, root, oldPath, content)

	result := workspace.Result{
		Sources: map[string]workspace.SourceLocation{
			"semantic:sales":                          {File: oldPath, Line: 1, Column: 1},
			"semantic-data:sales.order":               {File: oldPath, Line: 1, Column: 1},
			"semantic-attribute:sales.order.order_id": {File: oldPath, Line: 1, Column: 1},
			"semantic-relation:sales.order.customer":  {File: oldPath, Line: 1, Column: 1},
		},
		ManagedFiles: []workspace.ManagedFile{{Path: oldPath, SourceRoot: "semantic", Extension: ".tei"}},
	}
	idx, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}

	wantText := map[sourceindex.Subject]string{
		{Kind: "semantic", ID: "sales"}:                          "sales",
		{Kind: "semantic-data", ID: "sales.order"}:               "Order",
		{Kind: "semantic-attribute", ID: "sales.order.order_id"}: "orderId",
		{Kind: "semantic-relation", ID: "sales.order.customer"}:  "customer",
	}
	for subject, want := range wantText {
		refs := idx.ReferencesFor(subject)
		if len(refs) != 1 {
			t.Fatalf("references for %+v=%#v", subject, refs)
		}
		rng := refs[0].Range
		if rng.Start.Offset == rng.End.Offset {
			t.Fatalf("semantic subject %+v remained a point: %+v", subject, rng)
		}
		if got := string(content[rng.Start.Offset:rng.End.Offset]); got != want {
			t.Fatalf("semantic subject %+v anchor=%q want=%q range=%+v", subject, got, want, rng)
		}
		if refs[0].ArtifactPath != oldPath {
			t.Fatalf("artifact path for %+v=%q", subject, refs[0].ArtifactPath)
		}
	}

	subject := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	refs := idx.ReferencesFor(subject)
	newPath := "semantic/model/order.tei"
	writeSource(t, root, newPath, content)
	for key, loc := range result.Sources {
		loc.File = newPath
		result.Sources[key] = loc
	}
	result.ManagedFiles = []workspace.ManagedFile{{Path: newPath, SourceRoot: "semantic", Extension: ".tei"}}
	moved, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}
	movedRefs := moved.ReferencesFor(subject)
	if len(movedRefs) != 1 || movedRefs[0].ArtifactPath != newPath {
		t.Fatalf("moved references=%#v", movedRefs)
	}
	if movedRefs[0].Subject != refs[0].Subject {
		t.Fatalf("semantic subject changed after move: before=%+v after=%+v", refs[0].Subject, movedRefs[0].Subject)
	}
	if moved.Artifacts[0].Fingerprint != idx.Artifacts[0].Fingerprint {
		t.Fatalf("same content changed fingerprint: before=%s after=%s", idx.Artifacts[0].Fingerprint, moved.Artifacts[0].Fingerprint)
	}
}

func TestBuildSourceIndexUsesExactApplicationIdentifierRange(t *testing.T) {
	root := t.TempDir()
	p := "applications/order-entry/application.tei"
	content := []byte("application order_entry {}\n")
	writeSource(t, root, p, content)

	result := workspace.Result{
		Sources: map[string]workspace.SourceLocation{
			"application:order_entry": {File: p, Line: 1, Column: 1},
		},
		ManagedFiles: []workspace.ManagedFile{{Path: p, SourceRoot: "applications", Extension: ".tei"}},
	}
	idx, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		t.Fatal(err)
	}

	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "application", ID: "order_entry"})
	if len(refs) != 1 {
		t.Fatalf("application references=%#v", refs)
	}
	rng := refs[0].Range
	if rng.Start.Offset == rng.End.Offset {
		t.Fatalf("application anchor remained a point: %+v", rng)
	}
	if got := string(content[rng.Start.Offset:rng.End.Offset]); got != "order_entry" {
		t.Fatalf("application anchor=%q range=%+v", got, rng)
	}
	if rng.Start.Line != 1 || rng.Start.Column != 13 {
		t.Fatalf("application anchor start=%+v", rng.Start)
	}

	writeSource(t, root, p, []byte("application order_entry { uses semantic sales }\n"))
	if err := idx.Validate(func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	}); err == nil {
		t.Fatal("stale Source Index was accepted after content changed")
	}
}

func writeSource(t *testing.T, root, rel string, content []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

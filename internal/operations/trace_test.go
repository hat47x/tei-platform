package operations

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestTraceSubjectSourcesMatchesExistingIndex(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}
	got, err := TraceSubjectSources(root, subject)
	if err != nil { t.Fatal(err) }
	expected := idx.ReferencesFor(subject)
	if len(got.References) != len(expected) || len(expected) == 0 {
		t.Fatalf("trace references=%d expected=%d", len(got.References), len(expected))
	}
	for i := range expected {
		if !reflect.DeepEqual(got.References[i], expected[i]) {
			t.Fatalf("reference %d mismatch", i)
		}
	}
	if len(got.Artifacts) == 0 {
		t.Fatal("missing source fingerprints")
	}
	for _, artifact := range got.Artifacts {
		if artifact.Fingerprint == "" {
			t.Fatalf("missing fingerprint: %+v", artifact)
		}
	}
	missing, err := TraceSubjectSources(root, sourceindex.Subject{Kind: "query", ID: "not-existing"})
	if err != nil || len(missing.References) != 0 || len(missing.Artifacts) != 0 {
		t.Fatalf("unknown subject should have empty sources: %+v %v", missing, err)
	}
}

func TestTraceSubjectSourcesRejectsMissingIdentity(t *testing.T) {
	if _, err := TraceSubjectSources(t.TempDir(), sourceindex.Subject{Kind: "", ID: "x"}); err == nil {
		t.Fatal("missing subject kind accepted")
	}
	if _, err := TraceSubjectSources(t.TempDir(), sourceindex.Subject{Kind: "query"}); err == nil {
		t.Fatal("missing subject id accepted")
	}
}

func TestTraceSubjectSourcesReadOnly(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, root, "tei.workspace.yaml", "workspace: trace\nsources:\n  - semantic\n")
	writeWorkspaceFixture(t, root, "semantic/a.tei", "namespace sales\n")
	_, err := TraceSubjectSources(root, sourceindex.Subject{Kind: "semantic", ID: "sales"})
	if err != nil { t.Fatal(err) }
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 2 {
		t.Fatalf("workspace changed: %v %v", entries, err)
	}
}

func TestTraceSubjectSourcesFromIndexFiltersExactIdentity(t *testing.T) {
	selected := sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}
	other := sourceindex.Subject{Kind: "semantic-data", ID: "order_entry.recent_orders"}
	idx := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{
			{Path: "queries/order.tei"},
			{Path: "semantic/sales.tei"},
			{Path: "queries/unrelated.tei"},
		},
		References: []sourceindex.Reference{
			{Subject: selected, ArtifactPath: "queries/order.tei", Role: sourceindex.RoleDefinition},
			{Subject: selected, ArtifactPath: "queries/order.tei", Role: sourceindex.RoleAuthoring},
			{Subject: other, ArtifactPath: "semantic/sales.tei", Role: sourceindex.RoleDefinition},
		},
	}
	got, err := TraceSubjectSourcesFromIndex(idx, selected)
	if err != nil { t.Fatal(err) }
	if got.Subject != selected || len(got.References) != 2 || len(got.Artifacts) != 1 {
		t.Fatalf("subject sources incorrectly projected: %+v", got)
	}
	if got.Artifacts[0].Path != "queries/order.tei" {
		t.Fatalf("unrelated artifact leaked: %+v", got.Artifacts)
	}
	for _, ref := range got.References {
		if ref.Subject != selected {
			t.Fatalf("other kind leaked into query: %+v", ref)
		}
	}
	missing, err := TraceSubjectSourcesFromIndex(idx, sourceindex.Subject{Kind: "query", ID: "missing"})
	if err != nil { t.Fatal(err) }
	if len(missing.References) != 0 || len(missing.Artifacts) != 0 {
		t.Fatalf("invented source for unknown identity: %+v", missing)
	}
}

func TestTraceSubjectSourcesFromIndexRejectsBlankIdentity(t *testing.T) {
	for _, subject := range []sourceindex.Subject{
		{Kind: "", ID: "id"},
		{Kind: "query", ID: ""},
		{Kind: "  ", ID: "id"},
		{Kind: "query", ID: "  "},
	} {
		if _, err := TraceSubjectSourcesFromIndex(sourceindex.Index{}, subject); err == nil {
			t.Fatalf("accepted blank identity: %+v", subject)
		}
	}
}

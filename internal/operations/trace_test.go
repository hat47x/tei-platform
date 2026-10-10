package operations

import (
	"os"
	"path/filepath"
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
	if err != nil { t.Fatal(err) }
	subject := sourceindex.Subject{Kind: "query", ID: "order_entry.recent_orders"}
	got, err := TraceSubjectSources(root, subject)
	if err != nil { t.Fatal(err) }
	expected := idx.ReferencesFor(subject)
	if len(got.References) != len(expected) || len(expected) == 0 {
		t.Fatalf("trace references=%d expected=%d", len(got.References), len(expected))
	}
	for i := range expected {
		if got.References[i] != expected[i] {
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
	if err != nil || len(entries) != 2 { t.Fatalf("workspace changed: %v %v", entries, err) }
}

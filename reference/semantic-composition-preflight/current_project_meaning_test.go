package semanticcompositionpreflight

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/modelref"
	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/subject"
	"tei/internal/workspace"
)

type currentProjectMeaningTestReader struct {
	snapshot CurrentProjectMeaning
	err      error
	calls    int
}

func (r *currentProjectMeaningTestReader) ReadCurrentAcceptedProjectMeaning(_ context.Context, _ string) (CurrentProjectMeaning, error) {
	r.calls++
	return r.snapshot, r.err
}

func currentProjectMeaningFixture(t *testing.T) (string, string, workspace.Result, sourceindex.Index, *currentProjectMeaningTestReader) {
	t.Helper()
	root := t.TempDir()
	provider, consumer := filepath.Join(root, "provider"), filepath.Join(root, "consumer")
	writeSemanticFixture(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nsources:\n  - semantic\n")
	writeSemanticFixture(t, filepath.Join(provider, "semantic", "process.tei"),
		"namespace os\n\ntype ProcessId from std.identifier\n")
	writeSemanticFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	writeSemanticFixture(t, filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n")
	loaded, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(consumer, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Imports) != 1 {
		t.Fatalf("expected one explicit import, got %+v", loaded.Imports)
	}
	imported := loaded.Imports[0]
	reader := &currentProjectMeaningTestReader{
		snapshot: CurrentProjectMeaning{
			Source: sourceacceptance.Record{
				ApplicationID: "monitoring", DecisionRef: "synthetic://one-host-decision",
				AuthorityRevision: "synthetic-current", ModelRevision: sourceindex.ModelRevision(index),
				Artifacts: append([]sourceindex.Artifact(nil), index.Artifacts...),
			},
			Selected: []ProjectMeaningSelection{{
				Alias: imported.Alias,
				Resource: modelref.Ref{
					Scope:   "provider-domain:os",
					Subject: subject.Subject{Kind: "semantic-namespace", ID: imported.Namespace},
				},
				SemanticDigest: imported.SemanticDigest,
			}},
		},
	}
	return consumer, provider, loaded, index, reader
}

func expectedProjectMeaningRequests(reader *currentProjectMeaningTestReader) []ProjectMeaningRequest {
	selected := reader.snapshot.Selected[0]
	return []ProjectMeaningRequest{{Alias: selected.Alias, Resource: selected.Resource}}
}

func TestCurrentProjectMeaningBindsConsumerSourceAndExternalSemanticInOneHostSnapshot(t *testing.T) {
	consumer, provider, before, beforeIndex, reader := currentProjectMeaningFixture(t)
	requested := expectedProjectMeaningRequests(reader)
	verified, err := InspectCurrentProjectMeaning(context.Background(), reader, "monitoring", consumer, requested)
	if err != nil {
		t.Fatalf("coherent synthetic Host snapshot was refused: %v", err)
	}
	if reader.calls != 1 || !reflect.DeepEqual(verified.Loaded.World, before.World) ||
		!reflect.DeepEqual(verified.SourceIndex.Artifacts, beforeIndex.Artifacts) {
		t.Fatal("inspection must use one Host decision and the same Workspace Load for World and Source")
	}

	// The provider is not a managed source of the consumer. Semantic meaning
	// changes while consumer Source membership and its revision remain intact.
	writeSemanticFixture(t, filepath.Join(provider, "semantic", "process.tei"),
		"namespace os\n\ntype ProcessId from std.text\n")
	changed, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	changedIndex, err := workspace.BuildSourceIndex(consumer, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeIndex.Artifacts, changedIndex.Artifacts) ||
		before.Imports[0].SemanticDigest == changed.Imports[0].SemanticDigest {
		t.Fatal("fixture did not isolate external Semantic drift from unchanged consumer Source")
	}
	refused, err := InspectCurrentProjectMeaning(context.Background(), reader, "monitoring", consumer, requested)
	if err == nil || !strings.Contains(err.Error(), "PROJECT_MEANING_SEMANTIC_DIGEST_MISMATCH") {
		t.Fatalf("old Host decision must refuse changed provider meaning: %v", err)
	}
	if !reflect.DeepEqual(refused, VerifiedProjectMeaning{}) {
		t.Fatal("failed inspection returned unaccepted World")
	}

	// The trusted Host would have to issue a NEW current acceptance decision.
	// Updating the synthetic fixture here is not proof of any human Accept.
	reader.snapshot.Source.DecisionRef = "synthetic://replacement-host-decision"
	reader.snapshot.Source.AuthorityRevision = "synthetic-new-current"
	reader.snapshot.Selected[0].SemanticDigest = changed.Imports[0].SemanticDigest
	if _, err := InspectCurrentProjectMeaning(context.Background(), reader, "monitoring", consumer, requested); err != nil {
		t.Fatalf("replacement synthetic current decision was refused: %v", err)
	}
}

func TestCurrentProjectMeaningRejectsMissingAmbiguousAndForgedSelections(t *testing.T) {
	cases := []struct {
		name string
		edit func(*CurrentProjectMeaning)
		want string
	}{
		{"omitted", func(c *CurrentProjectMeaning) { c.Selected = nil }, "PROJECT_MEANING_SELECTION_SET_MISMATCH"},
		{"additional", func(c *CurrentProjectMeaning) {
			c.Selected = append(c.Selected, c.Selected[0])
		}, "PROJECT_MEANING_SELECTION_SET_MISMATCH"},
		{"wrong-alias", func(c *CurrentProjectMeaning) { c.Selected[0].Alias = "unexpected" }, "PROJECT_MEANING_SELECTION_UNDECLARED"},
		{"blank-alias", func(c *CurrentProjectMeaning) { c.Selected[0].Alias = "" }, "PROJECT_MEANING_SELECTION_AMBIGUOUS"},
		{"wrong-kind", func(c *CurrentProjectMeaning) { c.Selected[0].Resource.Subject.Kind = "semantic-data" }, "PROJECT_MEANING_RESOURCE_MISMATCH"},
		{"wrong-id", func(c *CurrentProjectMeaning) { c.Selected[0].Resource.Subject.ID = "other" }, "PROJECT_MEANING_RESOURCE_MISMATCH"},
		{"blank-scope", func(c *CurrentProjectMeaning) { c.Selected[0].Resource.Scope = "" }, "PROJECT_MEANING_RESOURCE_MISMATCH"},
		{"wrong-scope", func(c *CurrentProjectMeaning) { c.Selected[0].Resource.Scope = "provider-domain:other" }, "PROJECT_MEANING_RESOURCE_MISMATCH"},
		{"unverifiable-version", func(c *CurrentProjectMeaning) { c.Selected[0].Resource.Version = "1.0" }, "PROJECT_MEANING_RESOURCE_MISMATCH"},
		{"forged-digest", func(c *CurrentProjectMeaning) {
			c.Selected[0].SemanticDigest = "sha256:" + strings.Repeat("0", 64)
		}, "PROJECT_MEANING_SEMANTIC_DIGEST_MISMATCH"},
		{"other-application", func(c *CurrentProjectMeaning) { c.Source.ApplicationID = "other" }, "PROJECT_MEANING_SOURCE_NOT_ACCEPTED"},
		{"missing-source-decision", func(c *CurrentProjectMeaning) { c.Source.DecisionRef = "" }, "PROJECT_MEANING_SOURCE_NOT_ACCEPTED"},
		{"outdated-source-revision", func(c *CurrentProjectMeaning) {
			c.Source.ModelRevision = "sha256:" + strings.Repeat("0", 64)
		}, "PROJECT_MEANING_SOURCE_NOT_ACCEPTED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			consumer, _, _, _, reader := currentProjectMeaningFixture(t)
			requested := expectedProjectMeaningRequests(reader)
			tc.edit(&reader.snapshot)
			out, err := InspectCurrentProjectMeaning(context.Background(), reader, "monitoring", consumer, requested)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %s; got %v", tc.want, err)
			}
			if !reflect.DeepEqual(out, VerifiedProjectMeaning{}) {
				t.Fatal("refusal returned a World or receipt")
			}
		})
	}
}

func TestCurrentProjectMeaningRejectsStaleLocalSourceAndMissingHostReader(t *testing.T) {
	consumer, _, _, _, reader := currentProjectMeaningFixture(t)
	requested := expectedProjectMeaningRequests(reader)
	writeSemanticFixture(t, filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n\n")
	if out, err := InspectCurrentProjectMeaning(context.Background(), reader, "monitoring", consumer, requested); err == nil ||
		!strings.Contains(err.Error(), "PROJECT_MEANING_SOURCE_NOT_ACCEPTED") ||
		!reflect.DeepEqual(out, VerifiedProjectMeaning{}) {
		t.Fatalf("unchanged semantics but changed managed bytes must be refused: %v", err)
	}
	if out, err := InspectCurrentProjectMeaning(context.Background(), nil, "monitoring", consumer, requested); err == nil ||
		!strings.Contains(err.Error(), "PROJECT_MEANING_READER_REQUIRED") ||
		!reflect.DeepEqual(out, VerifiedProjectMeaning{}) {
		t.Fatalf("absent trusted Reader must fail closed: %v", err)
	}
	reader.err = errors.New("synthetic Host revocation")
	if out, err := InspectCurrentProjectMeaning(context.Background(), reader, "monitoring", consumer, requested); err == nil ||
		!strings.Contains(err.Error(), "PROJECT_MEANING_CURRENT_UNAVAILABLE") ||
		!reflect.DeepEqual(out, VerifiedProjectMeaning{}) {
		t.Fatalf("revoked Host decision must fail closed: %v", err)
	}
}

package semanticcompositionpreflight

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// syntheticExternalSourceReader is deliberately not a production Host witness.
// It proves only which managed Source bytes VerifyCurrent presently covers.
type syntheticExternalSourceReader struct {
	record sourceacceptance.Record
}

func (r syntheticExternalSourceReader) ReadCurrentAcceptedSource(_ context.Context, _ string) (sourceacceptance.Record, error) {
	return r.record, nil
}

// A consumer's unchanged accepted managed Source does not attest the contents
// of a separately owned semantic Namespace. This negative Reference must NOT
// be used to claim that either imported version received business Acceptance.
func TestLocalSourceAcceptanceDoesNotWitnessExternalSemanticRevision(t *testing.T) {
	root := t.TempDir()
	provider := filepath.Join(root, "provider")
	consumer := filepath.Join(root, "consumer")
	writeSemanticFixture(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nsources:\n  - semantic\n")
	writeSemanticFixture(t, filepath.Join(provider, "semantic", "os.tei"),
		"namespace os\n\ntype ProcessId from std.identifier\n")
	writeSemanticFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	writeSemanticFixture(t, filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute processId type os.process_id\n}\n")

	before, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	beforeIndex, err := workspace.BuildSourceIndex(consumer, before)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Imports) != 1 || validation.HasErrors(validation.Validate(before.World)) {
		t.Fatalf("invalid initial external semantic fixture: imports=%+v", before.Imports)
	}
	acceptedSource := sourceacceptance.Record{
		ApplicationID: "monitoring", DecisionRef: "synthetic://source-only",
		AuthorityRevision: "synthetic-revision",
		ModelRevision:     sourceindex.ModelRevision(beforeIndex),
		Artifacts:         append([]sourceindex.Artifact(nil), beforeIndex.Artifacts...),
	}
	reader := syntheticExternalSourceReader{record: acceptedSource}
	if _, err := sourceacceptance.VerifyCurrent(context.Background(), reader, "monitoring", beforeIndex); err != nil {
		t.Fatalf("unchanged locally managed Source was rejected: %v", err)
	}

	// Modify only the separately owned provider's semantic meaning. No
	// consumer-managed .tei Source or manifest is edited.
	writeSemanticFixture(t, filepath.Join(provider, "semantic", "os.tei"),
		"namespace os\n\ntype ProcessId from std.text\n")
	after, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	afterIndex, err := workspace.BuildSourceIndex(consumer, after)
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Imports) != 1 || validation.HasErrors(validation.Validate(after.World)) {
		t.Fatalf("invalid changed external semantic fixture: imports=%+v", after.Imports)
	}
	if before.Imports[0].SemanticDigest == after.Imports[0].SemanticDigest ||
		reflect.DeepEqual(before.World.Semantics, after.World.Semantics) {
		t.Fatal("changing the provider's meaning did not change imported semantic identity")
	}
	if !reflect.DeepEqual(beforeIndex.Artifacts, afterIndex.Artifacts) ||
		sourceindex.ModelRevision(beforeIndex) != sourceindex.ModelRevision(afterIndex) {
		t.Fatal("unmodified consumer-managed Source unexpectedly changed identity")
	}
	if _, err := sourceacceptance.VerifyCurrent(context.Background(), reader, "monitoring", afterIndex); err != nil {
		t.Fatalf("Source-only witness must remain a Source-only check: %v", err)
	}
	// The above success is intentionally insufficient for imported business
	// meaning. The existing optional pinned import can reject the drift, but
	// this pin itself does not represent a Host's current Acceptance decision.
	writeSemanticFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n    semanticDigest: "+before.Imports[0].SemanticDigest+"\n")
	if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "semantic digest mismatch") {
		t.Fatalf("explicitly pinned previous semantic meaning must reject provider drift: %v", err)
	}
}

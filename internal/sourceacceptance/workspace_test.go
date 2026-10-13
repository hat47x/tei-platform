package sourceacceptance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is a synthetic Host Reader; neither this fixture nor its DecisionRef
// is evidence of a real human Acceptance decision or Host authorization.
func workspaceTestReader(t *testing.T, root string) (*currentReader, sourceindex.Index, workspace.Result) {
	t.Helper()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Artifacts) == 0 {
		t.Fatal("reference Workspace has no managed Source")
	}
	r := &currentReader{record: Record{
		ApplicationID:     "order_entry",
		DecisionRef:       "test://source-acceptance/not-a-real-host",
		AuthorityRevision: "synthetic-test-revision",
		ModelRevision:     sourceindex.ModelRevision(index),
		Artifacts:         append([]sourceindex.Artifact(nil), index.Artifacts...),
	}}
	return r, index, loaded
}

func TestLoadVerifiedWorkspaceR1AndSemanticEquivalentR2(t *testing.T) {
	root := t.TempDir()
	// Use the actual TEI reference Workspace parser and Source Index, not an
	// index built from arbitrary in-memory strings. Never alter the checkout.
	src := filepath.Join("..", "..", "reference", "workspace")
	if err := os.CopyFS(root, os.DirFS(src)); err != nil {
		t.Fatalf("copy reference Workspace: %v", err)
	}
	reader, originalIndex, original := workspaceTestReader(t, root)
	verified, err := LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root)
	if err != nil {
		t.Fatalf("unchanged R1 was refused: %v", err)
	}
	if reader.calls != 1 || verified.Acceptance.ModelRevision != sourceindex.ModelRevision(verified.SourceIndex) {
		t.Fatal("R1 did not bind the current Reader to the actual loaded Source Index")
	}
	if !reflect.DeepEqual(original.World, verified.Loaded.World) {
		t.Fatal("verified Target input differs from the World used for Source Index creation")
	}

	var sourcePath string
	for _, artifact := range originalIndex.Artifacts {
		if strings.HasSuffix(artifact.Path, ".tei") {
			sourcePath = filepath.Join(root, filepath.FromSlash(artifact.Path))
			break
		}
	}
	if sourcePath == "" {
		t.Fatal("no managed .tei Source found in reference Workspace")
	}
	originalBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	// A trailing empty line changes the Source bytes, but not the authored
	// declarations. The Host witness still accepts the original R1.
	if err := os.WriteFile(sourcePath, append(append([]byte(nil), originalBytes...), '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	equivalent, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("blank-line R2 should remain valid Source: %v", err)
	}
	if !reflect.DeepEqual(original.World, equivalent.World) {
		t.Fatal("test fixture R2 unexpectedly changed canonical World semantics")
	}
	denied, err := LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_FINGERPRINT_MISMATCH") {
		t.Fatalf("semantic-equivalent R2 must be refused by the Source boundary: %v", err)
	}
	if !reflect.DeepEqual(denied, VerifiedWorkspace{}) {
		t.Fatal("refused R2 returned a World to its Target consumer")
	}

	// Replacing the synthetic current witness with R2 permits it, but now
	// reusing the earlier Source bytes (R0 relative to this new witness) fails.
	updatedIndex, err := workspace.BuildSourceIndex(root, equivalent)
	if err != nil {
		t.Fatal(err)
	}
	reader.record.Artifacts = append([]sourceindex.Artifact(nil), updatedIndex.Artifacts...)
	reader.record.ModelRevision = sourceindex.ModelRevision(updatedIndex)
	if _, err := LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root); err != nil {
		t.Fatalf("same-current-revision R2 was refused: %v", err)
	}
	if err := os.WriteFile(sourcePath, originalBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_FINGERPRINT_MISMATCH") {
		t.Fatalf("earlier Source R0 must be refused, regardless of caller's revision labels: %v", err)
	}
	reader.err = errors.New("synthetic Host revoked acceptance")
	_, err = LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE") {
		t.Fatalf("revoked current Reader must refuse before returning Source: %v", err)
	}
}

func TestLoadVerifiedWorkspaceRefusesMissingReaderBeforeSourceLoad(t *testing.T) {
	got, err := LoadVerifiedWorkspace(context.Background(), nil, "order_entry", filepath.Join(t.TempDir(), "missing"))
	if err == nil || !strings.Contains(err.Error(), "SOURCE_ACCEPTANCE_READER_REQUIRED") {
		t.Fatalf("missing Reader must be refused first: %v", err)
	}
	if !reflect.DeepEqual(got, VerifiedWorkspace{}) {
		t.Fatal("missing Reader returned an unverified Workspace")
	}
}

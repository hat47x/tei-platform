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
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
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
	updatedIndex, err := workspace.BuildSourceIndexFromLoad(root, equivalent)
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
	reader.err = errors.New("synthetic Host revoked acceptance; private token=secret-not-for-startup")
	_, err = LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root)
	if err == nil || !errors.Is(err, ErrCurrentAcceptanceUnavailable) ||
		strings.Contains(err.Error(), "secret-not-for-startup") {
		t.Fatalf("revoked current Reader must refuse with no private Host details: %v", err)
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

// Regression pin only. The deterministic Load-to-index race (a file changed
// between Load and index build) is covered in internal/workspace
// (TestBuildSourceIndexFromLoadIgnoresLaterFileChange); this test checks that
// a change to an unparsed managed file after acceptance is refused.
func TestLoadVerifiedWorkspaceRefusesChangedUnparsedManagedFile(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "..", "reference", "workspace"))); err != nil {
		t.Fatalf("copy reference Workspace: %v", err)
	}
	reader, _, _ := workspaceTestReader(t, root)
	if _, err := LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root); err != nil {
		t.Fatalf("unmodified accepted copy was refused: %v", err)
	}
	readme := filepath.Join(root, "databases", "sales-core", "README.md")
	if err := os.WriteFile(readme, []byte("modified after acceptance\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_FINGERPRINT_MISMATCH") {
		t.Fatalf("modified README.md must be refused: %v", err)
	}
	if !reflect.DeepEqual(got, VerifiedWorkspace{}) {
		t.Fatal("refused Workspace returned a World")
	}
}

// Files are changed between Load and index build through the private seam; the
// verified Source Index must still be the Load-time bytes, which the synthetic
// Host witness accepts.
//
// What this pins, honestly: since the managed-only bytes are captured at Load
// (internal/workspace/source_snapshot.go), the legacy workspace.BuildSourceIndex
// ALSO fingerprints Load-time bytes for a Result produced by Load, so these
// cases stay green if LoadVerifiedWorkspace were wired back to it (checked by
// mutation on 2026-11-02: only TestLoadVerifiedWorkspaceRefusesWorkspaceWithImports
// turned red). They are regression guards for the Load-time binding. What the
// strict builder adds over the legacy one is no filesystem fallback (covered
// in internal/workspace) and the refusal of imports.
func TestLoadVerifiedWorkspaceIndexesLoadTimeBytesNotLaterFileState(t *testing.T) {
	cases := []struct {
		name string
		pick func(t *testing.T, root string, idx sourceindex.Index) string
	}{
		{"unparsed README", func(t *testing.T, root string, _ sourceindex.Index) string {
			return filepath.Join(root, "databases", "sales-core", "README.md")
		}},
		{"parsed .tei", func(t *testing.T, root string, idx sourceindex.Index) string {
			for _, a := range idx.Artifacts {
				if strings.HasSuffix(a.Path, ".tei") {
					return filepath.Join(root, filepath.FromSlash(a.Path))
				}
			}
			t.Fatal("no .tei Artifact")
			return ""
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "..", "reference", "workspace"))); err != nil {
				t.Fatalf("copy reference Workspace: %v", err)
			}
			reader, accepted, _ := workspaceTestReader(t, root)
			target := tc.pick(t, root, accepted)
			original, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			ran := false
			verified, err := loadVerifiedWorkspace(context.Background(), reader, "order_entry", root, func() {
				ran = true
				if werr := os.WriteFile(target, append(append([]byte(nil), original...), []byte("\ntampered between Load and index build\n")...), 0o644); werr != nil {
					t.Error(werr)
				}
			})
			if !ran {
				t.Fatal("seam did not run")
			}
			if err != nil {
				t.Fatalf("Load-time bytes were accepted by the witness, later file state must not matter: %v", err)
			}
			if sourceindex.ModelRevision(verified.SourceIndex) != sourceindex.ModelRevision(accepted) {
				t.Fatal("verified Source Index is not the Load-time Source Index")
			}
		})
	}
}

// B1 at the acceptance boundary: a Workspace with imports gets a distinct
// refusal and no World, even when the provider is unchanged.
func TestLoadVerifiedWorkspaceRefusesWorkspaceWithImports(t *testing.T) {
	base := t.TempDir()
	write := func(rel, content string) {
		file := filepath.Join(base, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("provider/tei.workspace.yaml", "workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n")
	write("provider/semantic/os/process.tei", "namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute id type os.process_id\n}\n")
	write("consumer/tei.workspace.yaml", "workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	write("consumer/semantic/watch.tei", "namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n")
	root := filepath.Join(base, "consumer")
	// A Workspace with imports cannot be signed through the strict builder
	// (it refuses imports), so no accepted Record exists for it; the refusal
	// must come before the witness is consulted, so the Record is a placeholder.
	reader := &currentReader{record: Record{ApplicationID: "order_entry"}}
	got, err := LoadVerifiedWorkspace(context.Background(), reader, "order_entry", root)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_WORKSPACE_IMPORTS_UNBOUND") {
		t.Fatalf("Workspace with imports must be refused distinctly: %v", err)
	}
	if !reflect.DeepEqual(got, VerifiedWorkspace{}) {
		t.Fatal("refused Workspace returned a World")
	}
	if reader.calls != 0 {
		t.Fatal("Host witness was consulted for a Source Index that cannot bind the imports")
	}
}

// B2: the manifest is exposed as Load-time evidence, is not in the accepted
// Source Index, and a manifest change after Load is not adopted.
func TestLoadVerifiedWorkspaceExposesLoadTimeManifestAndDoesNotBindIt(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "..", "reference", "workspace"))); err != nil {
		t.Fatalf("copy reference Workspace: %v", err)
	}
	reader, accepted, _ := workspaceTestReader(t, root)
	manifest := filepath.Join(root, "tei.workspace.yaml")
	original, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := loadVerifiedWorkspace(context.Background(), reader, "order_entry", root, func() {
		if werr := os.WriteFile(manifest, append(append([]byte(nil), original...), []byte("# edited after Load\n")...), 0o644); werr != nil {
			t.Error(werr)
		}
	})
	if err != nil {
		t.Fatalf("a manifest edit is outside the accepted Source Index and must not be adopted or refused here: %v", err)
	}
	if want := sourceindex.Fingerprint(original); verified.Manifest.SHA256 != want || verified.Manifest.Path != "tei.workspace.yaml" {
		t.Fatalf("Manifest = %+v, want Load-time sha %s", verified.Manifest, want)
	}
	if sourceindex.ModelRevision(verified.SourceIndex) != sourceindex.ModelRevision(accepted) {
		t.Fatal("manifest change altered the Source Index")
	}
	for _, a := range verified.SourceIndex.Artifacts {
		if a.Path == "tei.workspace.yaml" {
			t.Fatal("manifest must not be an Artifact of the accepted Source Index")
		}
	}
}

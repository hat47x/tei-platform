package artifact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func TestNewEvidenceBundleRecordsExactSourceAndArtifactState(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	set, loaded, index, err := RenderWorkspaceWithIndex(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}

	worldFP, sourceFP, err := operations.WorkspaceProjectionFingerprints(loaded.World, index)
	if err != nil {
		t.Fatalf("fingerprints: %v", err)
	}

	bundle, err := NewEvidenceBundle(root, "ledger", loaded, index, set, nil, "test-generator/v1", "")
	if err != nil {
		t.Fatalf("new evidence bundle: %v", err)
	}

	if bundle.WorldFingerprint != worldFP {
		t.Fatalf("world fingerprint mismatch: have %s want %s", bundle.WorldFingerprint, worldFP)
	}
	if bundle.SourceIndexFingerprint != sourceFP {
		t.Fatalf("source index fingerprint mismatch: have %s want %s", bundle.SourceIndexFingerprint, sourceFP)
	}
	if bundle.ModelRevision != sourceindex.ModelRevision(index) {
		t.Fatalf("model revision mismatch: have %s want %s", bundle.ModelRevision, sourceindex.ModelRevision(index))
	}
	if bundle.Toolchain.Generator != "test-generator/v1" {
		t.Fatalf("unexpected generator: %s", bundle.Toolchain.Generator)
	}
	if bundle.Render.ArtifactCount != len(set.Artifacts) {
		t.Fatalf("artifact count mismatch: have %d want %d", bundle.Render.ArtifactCount, len(set.Artifacts))
	}
	if bundle.Render.Complete != set.Manifest.Complete {
		t.Fatalf("complete flag mismatch: have %v want %v", bundle.Render.Complete, set.Manifest.Complete)
	}
	if bundle.Render.ManifestDigest != set.Manifest.Digest {
		t.Fatalf("manifest digest mismatch: have %s want %s", bundle.Render.ManifestDigest, set.Manifest.Digest)
	}
	if len(bundle.Render.Artifacts) != len(set.Artifacts) {
		t.Fatalf("artifact list length mismatch: have %d want %d", len(bundle.Render.Artifacts), len(set.Artifacts))
	}
	if bundle.Output != nil && bundle.Output.Directory != "" {
		t.Fatalf("output directory must be empty when outDir is empty: %s", bundle.Output.Directory)
	}
}

func TestWriteEvidenceBundleWritesAndDigestsEvidenceFile(t *testing.T) {
	outDir := t.TempDir()
	set, loaded, index, err := RenderWorkspaceWithIndex(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	bundle, err := NewEvidenceBundle("workspace", "ledger", loaded, index, set, nil, "test-generator/v1", outDir)
	if err != nil {
		t.Fatalf("new evidence bundle: %v", err)
	}

	updated, err := WriteEvidenceBundle(outDir, bundle)
	if err != nil {
		t.Fatalf("write evidence bundle: %v", err)
	}

	if updated.Output.EvidenceFile != "evidence.json" {
		t.Fatalf("unexpected evidence file: %s", updated.Output.EvidenceFile)
	}
	if updated.Output.EvidenceDigest == "" {
		t.Fatal("evidence digest must be recorded")
	}
	if updated.Output.Directory != outDir {
		t.Fatalf("output directory mismatch: have %s want %s", updated.Output.Directory, outDir)
	}

	raw, err := os.ReadFile(filepath.Join(outDir, "evidence.json"))
	if err != nil {
		t.Fatalf("read evidence.json: %v", err)
	}
	var decoded EvidenceBundle
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode evidence bundle: %v", err)
	}
	if decoded.SchemaVersion != EvidenceSchemaVersion {
		t.Fatalf("schema version mismatch: have %s want %s", decoded.SchemaVersion, EvidenceSchemaVersion)
	}
	if decoded.Render.ManifestDigest != set.Manifest.Digest {
		t.Fatalf("manifest digest mismatch after round trip")
	}
}

func TestStageArtifactsIsolatesAndVerifiesOutput(t *testing.T) {
	outDir := t.TempDir()
	set, _, _, err := RenderWorkspaceWithIndex(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}

	stage, err := StageArtifacts(outDir, set)
	if err != nil {
		t.Fatalf("stage artifacts: %v", err)
	}
	defer os.RemoveAll(stage)

	entries, err := os.ReadDir(stage)
	if err != nil {
		t.Fatalf("read stage dir: %v", err)
	}
	if len(entries) != len(set.Artifacts)+1 {
		t.Fatalf("expected %d entries in stage, got %d", len(set.Artifacts)+1, len(entries))
	}

	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err == nil {
		t.Fatal("manifest must not appear in outDir before publish")
	}
}

func TestPublishStageMovesFilesAndRemovesStaleFiles(t *testing.T) {
	outDir := t.TempDir()
	set, _, _, err := RenderWorkspaceWithIndex(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}

	stage, err := StageArtifacts(outDir, set)
	if err != nil {
		t.Fatalf("stage artifacts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "stale-file.txt"), []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stage, "manifest.json"), []byte("stale-manifest"), 0o644); err != nil {
		t.Fatalf("write stale manifest: %v", err)
	}

	if err := PublishStage(stage, outDir); err != nil {
		t.Fatalf("publish stage: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("read outDir: %v", err)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Name()] = true
	}
	for _, artifact := range set.Artifacts {
		name, _ := ArtifactFileName(artifact)
		if !seen[name] {
			t.Fatalf("published artifact missing: %s", name)
		}
	}
	if !seen["manifest.json"] {
		t.Fatal("manifest.json must be published")
	}
	if seen["stale-file.txt"] {
		t.Fatal("stale files must be removed after publish")
	}
}

func TestEvidenceBundleRejectsTamperedArtifactContent(t *testing.T) {
	outDir := t.TempDir()
	set, loaded, index, err := RenderWorkspaceWithIndex(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}

	if _, err := NewEvidenceBundle("workspace", "ledger", loaded, index, set, nil, "test-generator/v1", outDir); err != nil {
		t.Fatalf("new evidence bundle: %v", err)
	}

	tampered := set
	if len(tampered.Artifacts) == 0 {
		t.Fatal("expected at least one artifact")
	}
	tampered.Artifacts[0].Content = tampered.Artifacts[0].Content + "\ntampered"
	tampered.Artifacts[0].Digest = "" // force re-verification to fail

	if _, err := NewEvidenceBundle("workspace", "ledger", loaded, index, tampered, nil, "test-generator/v1", outDir); err == nil {
		t.Fatal("tampered artifact content must fail evidence bundle creation")
	}
}

func TestSourceMutationChangesModelRevisionAndEvidenceBundle(t *testing.T) {
	root := t.TempDir()
	write := func(path, content string) {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: mutated\nformatVersion: 1\nsources:\n  - semantic\n  - applications\n  - databases\n")
	write("semantic/data.tei", "namespace app\n\ndata Item {\n    attribute id type std.identifier\n}\n")
	write("applications/app/application.tei", "application app {\n    uses semantic app\n\n    database sales {\n        target sales_core\n        access read-write\n    }\n}\n")
	write("databases/persistence.tei", "persistence sales.order mapping sales.order table orders\n")

	set1, loaded1, index1, err := RenderWorkspaceWithIndex(root, "app")
	if err != nil {
		t.Fatalf("render initial workspace: %v", err)
	}
	bundle1, err := NewEvidenceBundle(root, "app", loaded1, index1, set1, nil, "test-generator/v1", "")
	if err != nil {
		t.Fatalf("new evidence bundle initial: %v", err)
	}

	// Mutate source after the first render.
	write("semantic/data.tei", "namespace app\n\ndata Item {\n    attribute id type std.identifier\n}\n\ndata Other {\n    attribute id type std.identifier\n}\n")

	set2, loaded2, index2, err := RenderWorkspaceWithIndex(root, "app")
	if err != nil {
		t.Fatalf("render mutated workspace: %v", err)
	}
	bundle2, err := NewEvidenceBundle(root, "app", loaded2, index2, set2, nil, "test-generator/v1", "")
	if err != nil {
		t.Fatalf("new evidence bundle mutated: %v", err)
	}

	if bundle1.ModelRevision == bundle2.ModelRevision {
		t.Fatalf("model revision must change after source mutation: %s", bundle1.ModelRevision)
	}
	if bundle1.SourceIndexFingerprint == bundle2.SourceIndexFingerprint {
		t.Fatalf("source index fingerprint must change after source mutation")
	}
	if bundle1.Render.ManifestDigest == bundle2.Render.ManifestDigest {
		t.Fatalf("manifest digest must change after source mutation: %s", bundle1.Render.ManifestDigest)
	}
}

func TestSameSourceAndToolchainProduceSameEvidenceBundle(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")

	set1, loaded1, index1, err := RenderWorkspaceWithIndex(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace first: %v", err)
	}
	bundle1, err := NewEvidenceBundle(root, "ledger", loaded1, index1, set1, nil, "test-generator/v1", "")
	if err != nil {
		t.Fatalf("new evidence bundle first: %v", err)
	}

	set2, loaded2, index2, err := RenderWorkspaceWithIndex(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace second: %v", err)
	}
	bundle2, err := NewEvidenceBundle(root, "ledger", loaded2, index2, set2, nil, "test-generator/v1", "")
	if err != nil {
		t.Fatalf("new evidence bundle second: %v", err)
	}

	if bundle1.WorldFingerprint != bundle2.WorldFingerprint {
		t.Fatalf("world fingerprint must be stable: %s vs %s", bundle1.WorldFingerprint, bundle2.WorldFingerprint)
	}
	if bundle1.SourceIndexFingerprint != bundle2.SourceIndexFingerprint {
		t.Fatalf("source index fingerprint must be stable: %s vs %s", bundle1.SourceIndexFingerprint, bundle2.SourceIndexFingerprint)
	}
	if bundle1.ModelRevision != bundle2.ModelRevision {
		t.Fatalf("model revision must be stable: %s vs %s", bundle1.ModelRevision, bundle2.ModelRevision)
	}
	if bundle1.Render.ManifestDigest != bundle2.Render.ManifestDigest {
		t.Fatalf("manifest digest must be stable: %s vs %s", bundle1.Render.ManifestDigest, bundle2.Render.ManifestDigest)
	}

	// Changing the generator identity must produce a different toolchain record.
	bundle3, err := NewEvidenceBundle(root, "ledger", loaded1, index1, set1, nil, "other-generator/v2", "")
	if err != nil {
		t.Fatalf("new evidence bundle other generator: %v", err)
	}
	if bundle3.Toolchain.Generator == bundle1.Toolchain.Generator {
		t.Fatalf("changing the generator must change the toolchain record")
	}
}

func TestEvidenceBundleRecordsFindingsAndVerificationState(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "standard-crud", "workspace")
	set, loaded, index, err := RenderWorkspaceWithIndex(root, "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}

	bundle, err := NewEvidenceBundle(root, "ledger", loaded, index, set, nil, "test-generator/v1", "")
	if err != nil {
		t.Fatalf("new evidence bundle: %v", err)
	}

	if bundle.Verify.FindingsCount != len(set.Findings) {
		t.Fatalf("findings count mismatch: have %d want %d", bundle.Verify.FindingsCount, len(set.Findings))
	}
	if bundle.Verify.Verified != (len(set.Findings) == 0) {
		t.Fatalf("verified flag mismatch: have %v want %v", bundle.Verify.Verified, len(set.Findings) == 0)
	}
}

func TestArtifactFileNameMappingIsStable(t *testing.T) {
	cases := []struct {
		name string
		item Artifact
		want string
	}{
		{
			name: "table definition",
			item: Artifact{Kind: string(KindTableDefinition), Content: "table sales.order\n"},
			want: "table-sales-order.md",
		},
		{
			name: "function definition",
			item: Artifact{Kind: string(KindFunctionDefinition), Content: "application ledger\n"},
			want: "function-ledger.md",
		},
		{
			name: "operation surface",
			item: Artifact{Kind: string(KindOperationSurface), Content: payloadWithSurfaceVersion()},
			want: "operation-surface.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := ArtifactFileName(tc.item)
			if got != tc.want {
				t.Fatalf("file name mismatch: have %s want %s", got, tc.want)
			}
		})
	}
}

func payloadWithSurfaceVersion() string {
	return "schemaVersion: " + operations.OperationSurfaceSchemaVersion + "\n"
}

func TestSlugProducesFileSafeNames(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"sales.order", "sales-order"},
		{"sales/order", "sales-order"},
		{"Sales Order", "sales-order"},
	}
	for _, tc := range cases {
		got := Slug(tc.input)
		if got != tc.want {
			t.Fatalf("slug mismatch for %q: have %s want %s", tc.input, got, tc.want)
		}
	}
}

func TestReadPublishedManifestReturnsEnvelopeContents(t *testing.T) {
	outDir := t.TempDir()
	set, _, _, err := RenderWorkspaceWithIndex(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	if err := WriteManifest(outDir, set); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	manifest, findings, err := ReadPublishedManifest(outDir)
	if err != nil {
		t.Fatalf("read published manifest: %v", err)
	}
	if manifest.Digest != set.Manifest.Digest {
		t.Fatalf("manifest digest mismatch: have %s want %s", manifest.Digest, set.Manifest.Digest)
	}
	if len(findings) != len(set.Findings) {
		t.Fatalf("findings length mismatch: have %d want %d", len(findings), len(set.Findings))
	}
}

func TestWriteEvidenceBundleOmitsOutputWhenDirectoryEmpty(t *testing.T) {
	set, loaded, index, err := RenderWorkspaceWithIndex(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	bundle, err := NewEvidenceBundle("workspace", "ledger", loaded, index, set, nil, "test-generator/v1", "")
	if err != nil {
		t.Fatalf("new evidence bundle: %v", err)
	}

	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}
	if strings.Contains(string(raw), `"directory"`) {
		t.Fatalf("evidence bundle must omit output when directory is empty: %s", string(raw))
	}
}

func TestStagedSetVerifiesEvidenceFileWhenPresent(t *testing.T) {
	outDir := t.TempDir()
	set, loaded, index, err := RenderWorkspaceWithIndex(filepath.Join("..", "..", "reference", "standard-crud", "workspace"), "ledger")
	if err != nil {
		t.Fatalf("render workspace: %v", err)
	}
	bundle, err := NewEvidenceBundle("workspace", "ledger", loaded, index, set, nil, "test-generator/v1", outDir)
	if err != nil {
		t.Fatalf("new evidence bundle: %v", err)
	}
	if _, err := WriteEvidenceBundle(outDir, bundle); err != nil {
		t.Fatalf("write evidence bundle: %v", err)
	}

	stage, err := StageArtifacts(outDir, set)
	if err != nil {
		t.Fatalf("stage artifacts: %v", err)
	}
	defer os.RemoveAll(stage)

	// Tamper the evidence file in stage.
	if err := os.WriteFile(filepath.Join(stage, "evidence.json"), []byte("{\"schemaVersion\":\"tampered\"}"), 0o644); err != nil {
		t.Fatalf("tamper evidence: %v", err)
	}
	if err := verifyStagedSet(stage, set); err == nil {
		t.Fatal("tampered evidence must fail staged verification")
	}
}

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPrepareLaneArtifactStoreCreatesExplicitDirectoryOutsideRepository(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	outsideParent := filepath.Join(base, "evidence")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outsideParent, 0o700); err != nil {
		t.Fatal(err)
	}

	requested := filepath.Join(outsideParent, "children")
	store, err := prepareLaneArtifactStore(requested, repositoryRoot)
	if err != nil {
		t.Fatalf("prepareLaneArtifactStore() error=%v", err)
	}
	if store.auto {
		t.Fatal("explicit artifact directoryがauto扱いです")
	}
	storeInfo, err := os.Stat(store.dir)
	if err != nil {
		t.Fatal(err)
	}
	requestedInfo, err := os.Stat(requested)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(storeInfo, requestedInfo) {
		t.Fatalf("store.dir=%q does not identify requested=%q", store.dir, requested)
	}
	if !storeInfo.IsDir() {
		t.Fatalf("artifact store is not directory: %v", storeInfo.Mode())
	}
}

func TestPrepareLaneArtifactStoreRejectsRepositoryPath(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := prepareLaneArtifactStore(filepath.Join(repositoryRoot, "evidence"), repositoryRoot)
	if err == nil {
		t.Fatal("Repository内artifact directoryを受理しています")
	}
}

func TestLaneArtifactStoreWritesExactBytesAndRejectsOverwrite(t *testing.T) {
	dir := t.TempDir()
	store := laneArtifactStore{dir: dir}
	spec := laneSpec{Name: "deterministic-repository"}
	content := []byte("{\n  \"evidenceNature\": \"deterministic-repository\"\n}\n")

	filename, err := store.writeLane(spec, content)
	if err != nil {
		t.Fatalf("writeLane() error=%v", err)
	}
	if filename != "deterministic-repository.json" {
		t.Fatalf("filename=%q", filename)
	}
	got, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, content) {
		t.Fatalf("stored bytes=%q want exact=%q", got, content)
	}

	if _, err := store.writeLane(spec, []byte(`{"different":true}`)); err == nil {
		t.Fatal("既存child Evidence artifactを上書きしています")
	}
	gotAfter, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotAfter, content) {
		t.Fatal("上書き拒否後に既存artifactが変化しています")
	}
}

func TestEnsureExternalExistingRegularFileRejectsRepositoryFileAndSymlink(t *testing.T) {
	base := t.TempDir()
	repositoryRoot := filepath.Join(base, "repo")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(repositoryRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}

	insideSecret := filepath.Join(repositoryRoot, "dsn")
	if err := os.WriteFile(insideSecret, []byte("postgres://inside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureExternalExistingRegularFile(insideSecret, repositoryRoot); err == nil {
		t.Fatal("Repository内secret fileを受理しています")
	}

	outsideSecret := filepath.Join(outside, "dsn")
	if err := os.WriteFile(outsideSecret, []byte("postgres://outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolved, err := ensureExternalExistingRegularFile(outsideSecret, repositoryRoot)
	if err != nil {
		t.Fatalf("Repository外regular fileを拒否しました: %v", err)
	}
	if resolved == "" {
		t.Fatal("resolved outside pathが空です")
	}

	link := filepath.Join(outside, "link-to-inside")
	if err := os.Symlink(insideSecret, link); err != nil {
		t.Skipf("symlink test unavailable: %v", err)
	}
	if _, err := ensureExternalExistingRegularFile(link, repositoryRoot); err == nil {
		t.Fatal("Repository外pathからRepository内secretへ向くsymlinkを受理しています")
	}
}

func TestAllChildArtifactsRetainedRequiresThreePassedArtifacts(t *testing.T) {
	lanes := []laneEvidence{
		{Name: "deterministic-repository", Passed: true, ArtifactFile: "deterministic-repository.json", EvidenceArtifactFingerprint: "sha256:a"},
		{Name: "generated-artifact-reproducibility", Passed: true, ArtifactFile: "generated-artifact-reproducibility.json", EvidenceArtifactFingerprint: "sha256:b"},
		{Name: "real-postgres-ephemeral-fixture", Passed: true, ArtifactFile: "real-postgres-ephemeral-fixture.json", EvidenceArtifactFingerprint: "sha256:c"},
	}
	if !allChildArtifactsRetained(lanes) {
		t.Fatal("3 child artifactが揃っているのにretained=falseです")
	}
	lanes[1].ArtifactFile = ""
	if allChildArtifactsRetained(lanes) {
		t.Fatal("artifact fileが欠けているのにretained=trueです")
	}
}

func TestPathWithinRepository(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "repo")
	if !pathWithinRepository(root, filepath.Join(root, "child", "evidence.json")) {
		t.Fatal("Repository childをoutside扱いしています")
	}
	if pathWithinRepository(root, filepath.Join(base, "outside", "evidence.json")) {
		t.Fatal("Repository siblingをinside扱いしています")
	}
}

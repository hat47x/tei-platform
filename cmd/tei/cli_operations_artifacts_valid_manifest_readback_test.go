package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/artifact"
)

// #2604 AP-01/AP-06 characterization with a *real* NewManifest, unlike the
// original AP-01..03 writer tests which deliberately use a zero-value Manifest.
// These cases do not fix publication or prove atomicity/retention/independent
// process readback. Their only witness is a local filesystem read after Write.
func publicationValidSet(t *testing.T, revision string, bodies map[string]string) artifact.WorkspaceSet {
	t.Helper()
	rule := artifact.Rule{ID: "tei.ap-valid", Version: "v1", Fingerprint: "sha256:ap-valid-rule"}
	var items []artifact.Artifact
	for _, name := range []string{"a", "b", "c"} {
		body, present := bodies[name]
		if !present {
			continue
		}
		item, err := artifact.New(artifact.KindTableDefinition, rule,
			"tei-ap-valid-fixture/v1", revision, fmt.Sprintf("table %s\n%s\n", name, body), nil)
		if err != nil {
			t.Fatalf("build artifact %s: %v", name, err)
		}
		items = append(items, item)
	}
	manifest, err := artifact.NewManifest(revision, items)
	if err != nil {
		t.Fatalf("build valid manifest: %v", err)
	}
	if err := manifest.Verify(); err != nil {
		t.Fatalf("manifest verify: %v", err)
	}
	return artifact.WorkspaceSet{
		InputRevision: revision, Artifacts: items, Manifest: manifest,
		Findings: []artifact.Finding{},
	}
}

func publicationReadValidManifest(t *testing.T, directory string) artifact.Manifest {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		t.Fatalf("read saved manifest: %v", err)
	}
	var saved struct {
		Manifest artifact.Manifest  `json:"manifest"`
		Findings []artifact.Finding `json:"findings"`
	}
	if err := json.Unmarshal(bytes, &saved); err != nil {
		t.Fatalf("decode saved manifest: %v", err)
	}
	if err := saved.Manifest.Verify(); err != nil {
		t.Fatalf("saved manifest verification: %v", err)
	}
	return saved.Manifest
}

func publicationReadMember(t *testing.T, dir, name string) string {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join(dir, "table-"+name+".md"))
	if err != nil {
		t.Fatalf("read published member %s: %v", name, err)
	}
	return string(bytes)
}

func TestValidManifestPublicationFreshTwoMembersReadable(t *testing.T) {
	dir := t.TempDir()
	r1 := publicationValidSet(t, "sha256:ap-valid-r1", map[string]string{"a": "old-a", "b": "old-b"})
	if err := writeArtifactFiles(dir, r1); err != nil {
		t.Fatalf("publish R1: %v", err)
	}
	saved := publicationReadValidManifest(t, dir)
	if saved.Digest != r1.Manifest.Digest || !saved.Complete || len(saved.Entries) != 2 {
		t.Fatalf("unexpected saved manifest: %+v", saved)
	}
	if got := publicationReadMember(t, dir, "a"); got != r1.Artifacts[0].Content {
		t.Fatalf("R1 a bytes differ from expected")
	}
	if got := publicationReadMember(t, dir, "b"); got != r1.Artifacts[1].Content {
		t.Fatalf("R1 b bytes differ from expected")
	}
}

func TestValidManifestPublicationMidwayR2FailureInvalidatesR1Reader(t *testing.T) {
	dir := t.TempDir()
	r1 := publicationValidSet(t, "sha256:ap-valid-r1", map[string]string{"a": "old-a", "b": "old-b"})
	if err := writeArtifactFiles(dir, r1); err != nil {
		t.Fatalf("publish R1: %v", err)
	}
	if got := publicationReadValidManifest(t, dir); got.Digest != r1.Manifest.Digest {
		t.Fatal("expected R1 manifest")
	}
	// The new R2-only member is a directory. R1's required a/b remain
	// intact until writeArtifactFiles overwrites b and fails at c.
	if err := os.Mkdir(filepath.Join(dir, "table-c.md"), 0o755); err != nil {
		t.Fatalf("create R2-only blocked target: %v", err)
	}
	r2 := publicationValidSet(t, "sha256:ap-valid-r2",
		map[string]string{"a": "old-a", "b": "new-b", "c": "blocked-c"})
	if err := writeArtifactFiles(dir, r2); err == nil {
		t.Fatal("expected R2 member write failure")
	}
	// R1 still has a correctly hashed logical manifest, but the body it
	// names is no longer physically present at b's former contents.
	if got := publicationReadValidManifest(t, dir); got.Digest != r1.Manifest.Digest {
		t.Fatal("failed R2 publication unexpectedly replaced R1 manifest")
	}
	if got := publicationReadMember(t, dir, "a"); got != r1.Artifacts[0].Content {
		t.Fatal("expected R1 a bytes")
	}
	if got := publicationReadMember(t, dir, "b"); got == r1.Artifacts[1].Content {
		t.Fatal("old R1 b unexpectedly remained readable")
	} else if got != r2.Artifacts[1].Content {
		t.Fatal("unexpected body after partial R2 publish")
	}
	// Manifest.Verify alone is not a claim of valid disk member bytes.
}

func TestValidManifestPublicationR2SuccessDoesNotRetainPinnedR1Members(t *testing.T) {
	dir := t.TempDir()
	r1 := publicationValidSet(t, "sha256:ap-valid-r1", map[string]string{"a": "old-a", "b": "old-b"})
	r2 := publicationValidSet(t, "sha256:ap-valid-r2", map[string]string{"a": "new-a", "b": "new-b"})
	if err := writeArtifactFiles(dir, r1); err != nil {
		t.Fatalf("publish R1: %v", err)
	}
	if got := publicationReadValidManifest(t, dir); got.Digest != r1.Manifest.Digest {
		t.Fatal("expected R1 manifest before R2")
	}
	if err := writeArtifactFiles(dir, r2); err != nil {
		t.Fatalf("publish R2: %v", err)
	}
	if got := publicationReadValidManifest(t, dir); got.Digest != r2.Manifest.Digest {
		t.Fatal("expected R2 manifest after R2")
	}
	for i, name := range []string{"a", "b"} {
		got := publicationReadMember(t, dir, name)
		if got != r2.Artifacts[i].Content {
			t.Fatalf("saved R2 %s body mismatch", name)
		}
		if got == r1.Artifacts[i].Content {
			t.Fatalf("R1 pin could still read unchanged %s bytes; fixture is not discriminating", name)
		}
	}
	// A Reader that pinned R1 cannot reread its exact original member bytes
	// from this directory after R2. The new R2 publication is not "R1 retained".
}

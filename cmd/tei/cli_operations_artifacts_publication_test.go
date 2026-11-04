package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/artifact"
)

// These tests record what `tei artifacts -out` leaves in the output directory
// today (#2604 AP-01, AP-02, AP-03). They describe the present behaviour of
// writeArtifactFiles; they do not say the behaviour is acceptable. When the
// publication method changes, these tests are expected to change with it.
// Nothing here covers a real Consumer reading the directory, Windows, or other
// file systems.

func publicationRule() artifact.Rule {
	return artifact.Rule{ID: "ap-test", Version: "1", Fingerprint: "sha256:ap-rule"}
}

func publicationTable(t *testing.T, name, body string) artifact.Artifact {
	t.Helper()
	item, err := artifact.New(artifact.KindTableDefinition, publicationRule(), "ap-test/v1", "sha256:ap-input", "table "+name+"\n"+body, nil)
	if err != nil {
		t.Fatalf("new artifact %s: %v", name, err)
	}
	return item
}

func publicationSet(items ...artifact.Artifact) artifact.WorkspaceSet {
	return artifact.WorkspaceSet{InputRevision: "sha256:ap-input", Artifacts: items, Findings: []artifact.Finding{}}
}

func readPublicationFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

// AP-01: a failure in the middle of the second generation leaves the new first
// file next to the manifest of the first generation.
func TestArtifactPublicationMidwayFailureLeavesMixedGenerations(t *testing.T) {
	outDir := t.TempDir()
	r1 := publicationSet(publicationTable(t, "a", "r1\n"), publicationTable(t, "b", "r1\n"))
	if err := writeArtifactFiles(outDir, r1); err != nil {
		t.Fatalf("publish R1: %v", err)
	}
	manifestR1 := readPublicationFile(t, outDir, "manifest.json")

	// Injected failure: table-b.md becomes a directory, so the write fails
	// after table-a.md has already been replaced.
	if err := os.Remove(filepath.Join(outDir, "table-b.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(outDir, "table-b.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	r2 := publicationSet(publicationTable(t, "a", "r2\n"), publicationTable(t, "b", "r2\n"))
	if err := writeArtifactFiles(outDir, r2); err == nil {
		t.Fatal("expected the injected write failure to be reported")
	}

	if got := readPublicationFile(t, outDir, "table-a.md"); !strings.Contains(got, "r2") {
		t.Fatalf("observation changed: table-a.md is expected to hold R2 before the failure, got %q", got)
	}
	if got := readPublicationFile(t, outDir, "manifest.json"); got != manifestR1 {
		t.Fatal("observation changed: manifest.json is expected to still be the R1 manifest")
	}
	// The directory now holds R2 content for table a under the R1 manifest.
}

// AP-02: a file that the new generation no longer produces stays in the
// directory, while the new manifest does not list it.
func TestArtifactPublicationKeepsFilesOfRemovedArtifacts(t *testing.T) {
	outDir := t.TempDir()
	r1 := publicationSet(publicationTable(t, "a", "r1\n"), publicationTable(t, "b", "r1\n"))
	if err := writeArtifactFiles(outDir, r1); err != nil {
		t.Fatalf("publish R1: %v", err)
	}
	removed := r1.Artifacts[1]

	r2 := publicationSet(publicationTable(t, "a", "r1\n"))
	r2.Manifest = artifact.Manifest{}
	if err := writeArtifactFiles(outDir, r2); err != nil {
		t.Fatalf("publish R2: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "table-b.md")); err != nil {
		t.Fatalf("observation changed: table-b.md is expected to remain after R2: %v", err)
	}
	if strings.Contains(readPublicationFile(t, outDir, "manifest.json"), removed.Digest) {
		t.Fatal("the R2 manifest must not list the removed artifact")
	}
}

// AP-03: two subjects that map to the same file name are detected only inside
// the write loop, after the first of them has replaced the earlier file.
func TestArtifactPublicationNameCollisionIsDetectedAfterEarlierWrite(t *testing.T) {
	outDir := t.TempDir()
	r1 := publicationSet(publicationTable(t, "sales/a", "r1\n"))
	if err := writeArtifactFiles(outDir, r1); err != nil {
		t.Fatalf("publish R1: %v", err)
	}
	manifestR1 := readPublicationFile(t, outDir, "manifest.json")

	r2 := publicationSet(publicationTable(t, "sales/a", "r2\n"), publicationTable(t, "sales.a", "r2\n"))
	err := writeArtifactFiles(outDir, r2)
	if err == nil || !strings.Contains(err.Error(), "same file name") {
		t.Fatalf("expected the collision to be refused, got %v", err)
	}

	if got := readPublicationFile(t, outDir, "table-sales-a.md"); !strings.Contains(got, "r2") {
		t.Fatalf("observation changed: the file is expected to hold the first R2 artifact, got %q", got)
	}
	if got := readPublicationFile(t, outDir, "manifest.json"); got != manifestR1 {
		t.Fatal("observation changed: manifest.json is expected to still be the R1 manifest")
	}
}

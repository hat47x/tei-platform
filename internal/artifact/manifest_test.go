package artifact

import (
	"testing"
)

func manifestArtifacts(t *testing.T, unresolved int) []Artifact {
	t.Helper()
	table, err := RenderTableDefinition(tableDefinitionInput())
	if err != nil {
		t.Fatalf("render table definition: %v", err)
	}
	function, err := RenderFunctionDefinition(functionDefinitionInput())
	if err != nil {
		t.Fatalf("render function definition: %v", err)
	}
	artifacts := []Artifact{table, function}
	if unresolved > 0 {
		artifacts[0].Unresolved = []Unresolved{{Subject: "sales.order", Reason: "missing-input", Detail: "test"}}
		built, err := New(KindTableDefinition, artifacts[0].Rule, artifacts[0].Generator, artifacts[0].InputRevision, artifacts[0].Content, artifacts[0].Unresolved)
		if err != nil {
			t.Fatalf("rebuild artifact: %v", err)
		}
		artifacts[0] = built
	}
	return artifacts
}

func TestNewManifestIsDeterministicAndSortsEntries(t *testing.T) {
	artifacts := manifestArtifacts(t, 0)
	first, err := NewManifest("sha256:input", artifacts)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	second, err := NewManifest("sha256:input", []Artifact{artifacts[1], artifacts[0]})
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("entry order must not change the manifest digest")
	}
	if err := first.Verify(); err != nil {
		t.Fatalf("manifest must verify: %v", err)
	}
	if !first.Complete {
		t.Fatalf("a set without unresolved entries is complete: %+v", first)
	}
	if len(first.Entries) != 2 || first.Entries[0].Kind != string(KindFunctionDefinition) {
		t.Fatalf("entries must be sorted by kind: %+v", first.Entries)
	}
}

func TestManifestIsNotCompleteWhenAnArtifactIsUnresolved(t *testing.T) {
	manifest, err := NewManifest("sha256:input", manifestArtifacts(t, 1))
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if manifest.Complete {
		t.Fatalf("an unresolved artifact must keep the set incomplete: %+v", manifest)
	}
	if manifest.Entries[1].Unresolved == 0 && manifest.Entries[0].Unresolved == 0 {
		t.Fatalf("the unresolved count must be visible in the entries: %+v", manifest.Entries)
	}
}

func TestManifestRefusesAMixedRevision(t *testing.T) {
	artifacts := manifestArtifacts(t, 0)
	other, err := New(KindCrossReference, testRule(), "tei-generator/v1", "sha256:other", "cross reference\n", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewManifest("sha256:input", append(artifacts, other)); err == nil {
		t.Fatal("artifacts from different input revisions must be refused")
	}
}

func TestManifestRefusesTamperedArtifacts(t *testing.T) {
	artifacts := manifestArtifacts(t, 0)
	artifacts[0].Content = "tampered"
	if _, err := NewManifest("sha256:input", artifacts); err == nil {
		t.Fatal("a tampered artifact must be refused")
	}
}

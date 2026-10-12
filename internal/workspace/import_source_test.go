package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func TestLocateImportedSourceUsesProviderScopeAndCurrentFingerprint(t *testing.T) {
	consumer, provider := externalPair(t)
	loaded, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	target := sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"}
	before, err := workspace.LocateImportedSource(consumer, loaded, "os", target)
	if err != nil {
		t.Fatalf("locate provider source: %v", err)
	}
	if before.ProviderPath != "../provider" || before.Namespace != "os" || before.Subject != target {
		t.Fatalf("wrong import origin: %+v", before)
	}
	if before.Reference.ArtifactPath != "semantic/os/process.tei" ||
		before.Artifact.Path != "semantic/os/process.tei" ||
		!strings.HasPrefix(before.Artifact.Fingerprint, "sha256:") ||
		before.Reference.Range.End.Offset <= before.Reference.Range.Start.Offset {
		t.Fatalf("missing provider-local verified source anchor: %+v", before)
	}
	srcIndex, err := workspace.BuildSourceIndex(consumer, loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range srcIndex.References {
		if reference.Subject == target {
			t.Fatal("imported provider anchor must not appear in consumer source index")
		}
	}
	if err := os.Rename(filepath.Join(provider, "semantic/os/process.tei"),
		filepath.Join(provider, "semantic/os/renamed.tei")); err != nil {
		t.Fatal(err)
	}
	after, err := workspace.LocateImportedSource(consumer, loaded, "os", target)
	if err != nil {
		t.Fatalf("provider source move must preserve meaning: %v", err)
	}
	if after.Artifact.Path != "semantic/os/renamed.tei" ||
		after.Artifact.Fingerprint != before.Artifact.Fingerprint {
		t.Fatalf("source locator changed unexpectedly on file move: before=%+v after=%+v", before, after)
	}
}

func TestLocateImportedSourceRejectsStaleOrForeignTargets(t *testing.T) {
	t.Run("semantic-change", func(t *testing.T) {
		consumer, provider := externalPair(t)
		loaded, err := workspace.Load(consumer)
		if err != nil {
			t.Fatal(err)
		}
		writeExternalFixture(t, filepath.Join(provider, "semantic/os/process.tei"),
			"namespace os\n\ntype ProcessId from std.text\n\ndata Process {\n    attribute id type os.process_id\n}\n")
		_, err = workspace.LocateImportedSource(consumer, loaded, "os",
			sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"})
		if err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("changed provider meaning must be rejected: %v", err)
		}
	})
	t.Run("unimported-namespace", func(t *testing.T) {
		consumer, provider := externalPair(t)
		writeExternalFixture(t, filepath.Join(provider, "semantic/other.tei"),
			"namespace other\n\ntype OtherId from std.identifier\n")
		loaded, err := workspace.Load(consumer)
		if err != nil {
			t.Fatal(err)
		}
		_, err = workspace.LocateImportedSource(consumer, loaded, "os",
			sourceindex.Subject{Kind: "semantic-type", ID: "other.other_id"})
		if err == nil || !strings.Contains(err.Error(), "not part of namespace") {
			t.Fatalf("foreign namespace must not be navigable: %v", err)
		}
	})
	t.Run("missing-alias", func(t *testing.T) {
		consumer, _ := externalPair(t)
		loaded, err := workspace.Load(consumer)
		if err != nil {
			t.Fatal(err)
		}
		_, err = workspace.LocateImportedSource(consumer, loaded, "missing",
			sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"})
		if err == nil || !strings.Contains(err.Error(), "not resolved") {
			t.Fatalf("unknown alias must be rejected: %v", err)
		}
	})
	t.Run("forged-source-root", func(t *testing.T) {
		consumer, _ := externalPair(t)
		loaded, err := workspace.Load(consumer)
		if err != nil {
			t.Fatal(err)
		}
		loaded.Imports[0].Path = "/etc"
		_, err = workspace.LocateImportedSource(consumer, loaded, "os",
			sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"})
		if err == nil || !strings.Contains(err.Error(), "invalid provider path") {
			t.Fatalf("forged absolute path must fail: %v", err)
		}
	})
	t.Run("missing-subject-kind", func(t *testing.T) {
		consumer, _ := externalPair(t)
		loaded, err := workspace.Load(consumer)
		if err != nil {
			t.Fatal(err)
		}
		_, err = workspace.LocateImportedSource(consumer, loaded, "os",
			sourceindex.Subject{Kind: "capability", ID: "os.process_id"})
		if err == nil || !strings.Contains(err.Error(), "not part of namespace") {
			t.Fatalf("invalid Subject kind must not receive a source anchor: %v", err)
		}
	})
}

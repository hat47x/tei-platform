package workspace_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
)

func pinnedImportManifest(digest string) string {
	return fmt.Sprintf("workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n    semanticDigest: %s\n", digest)
}

func TestExternalSemanticImportOptionalDigestPin(t *testing.T) {
	consumer, provider := externalPair(t)
	unpinned, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(unpinned.Imports) != 1 {
		t.Fatalf("expected one resolved import: %#v", unpinned.Imports)
	}
	pin := unpinned.Imports[0].SemanticDigest
	writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"), pinnedImportManifest(pin))
	pinned, err := workspace.Load(consumer)
	if err != nil {
		t.Fatalf("matching pin rejected: %v", err)
	}
	if len(pinned.Imports) != 1 || pinned.Imports[0].SemanticDigest != pin {
		t.Fatalf("resolved semantic digest changed: %#v", pinned.Imports)
	}
	// A physical file move is not a change to its canonical meaning.
	if err := os.Rename(filepath.Join(provider, "semantic/os/process.tei"),
		filepath.Join(provider, "semantic/os/relocated.tei")); err != nil {
		t.Fatal(err)
	}
	relocated, err := workspace.Load(consumer)
	if err != nil || relocated.Imports[0].SemanticDigest != pin {
		t.Fatalf("semantic pin must survive a physical source move: %+v, %v", relocated.Imports, err)
	}
	// A provider may add an unrelated namespace without changing the selected one.
	writeExternalFixture(t, filepath.Join(provider, "semantic/other.tei"),
		"namespace other\n\ntype OtherId from std.identifier\n")
	if _, err := workspace.Load(consumer); err != nil {
		t.Fatalf("unrelated provider namespace must not invalidate namespace pin: %v", err)
	}
	// A semantic change to the selected namespace must fail before it is composed.
	writeExternalFixture(t, filepath.Join(provider, "semantic/os/relocated.tei"),
		"namespace os\n\ntype ProcessId from std.text\n\ndata Process {\n    attribute id type os.process_id\n}\n")
	if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "semantic digest mismatch") {
		t.Fatalf("changed semantic definition must fail the pin: %v", err)
	}
	// Explicitly removing the optional assertion allows an ordinary editable
	// local Workspace dependency. No second registry or lock is consulted.
	writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	if _, err := workspace.Load(consumer); err != nil {
		t.Fatalf("unpinned local development must remain possible: %v", err)
	}
}

func TestExternalSemanticImportRejectsMalformedOrDuplicateDigest(t *testing.T) {
	for _, bad := range []struct {
		name string
		digest string
	}{
		{"empty", ""},
		{"wrong-algorithm", "sha1:" + strings.Repeat("a", 64)},
		{"short", "sha256:" + strings.Repeat("a", 63)},
		{"non-hex", "sha256:" + strings.Repeat("z", 64)},
		{"uppercase", "sha256:" + strings.Repeat("A", 64)},
		{"extra-byte", "sha256:" + strings.Repeat("a", 65)},
	} {
		t.Run(bad.name, func(t *testing.T) {
			consumer, _ := externalPair(t)
			writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"), pinnedImportManifest(bad.digest))
			if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "invalid import semanticDigest") {
				t.Fatalf("malformed digest must be rejected: %v", err)
			}
		})
	}
	t.Run("duplicate", func(t *testing.T) {
		consumer, _ := externalPair(t)
		pin := "sha256:" + strings.Repeat("a", 64)
		writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
			pinnedImportManifest(pin)+"    semanticDigest: "+pin+"\n")
		if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "duplicate import semanticDigest") {
			t.Fatalf("duplicate pin must be rejected: %v", err)
		}
	})
}

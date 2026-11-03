package workspace_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is a P0a integration oracle, not a Host acceptance or a full imported
// Source snapshot. Its expected provider, type and raw bytes are authored
// independently of Workspace.Load's ResolvedImport and Source Index outputs.
func TestExternalImportP0aProviderSelectionAndStrictRefusal(t *testing.T) {
	root := t.TempDir()
	consumer := filepath.Join(root, "consumer")
	providerA := filepath.Join(root, "provider-a")
	providerB := filepath.Join(root, "provider-b")

	const providerManifest = "workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n"
	const sourceA = "namespace os\n\ntype ProcessId from std.identifier\n"
	const sourceB = "namespace os\n\ntype ProcessId from std.text\n"
	const consumerSource = "namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n"

	writeExternalFixture(t, filepath.Join(providerA, "tei.workspace.yaml"), providerManifest)
	writeExternalFixture(t, filepath.Join(providerB, "tei.workspace.yaml"), providerManifest)
	writeExternalFixture(t, filepath.Join(providerA, "semantic/os/process.tei"), sourceA)
	writeExternalFixture(t, filepath.Join(providerB, "semantic/os/process.tei"), sourceA)
	writeExternalFixture(t, filepath.Join(consumer, "semantic/watch.tei"), consumerSource)

	manifest := func(providerName, pin string) string {
		s := fmt.Sprintf("workspace: consumer\nformatVersion: 1\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../%s\n    namespace: os\n", providerName)
		if pin != "" {
			s += "    semanticDigest: " + pin + "\n"
		}
		return s
	}
	digest := func(raw string) string {
		sum := sha256.Sum256([]byte(raw))
		return fmt.Sprintf("sha256:%x", sum)
	}
	assertImportedType := func(loaded workspace.Result, wantUnderlying string) {
		t.Helper()
		if len(loaded.Imports) != 1 || loaded.Imports[0].Namespace != "os" {
			t.Fatalf("exactly one explicitly imported os namespace is required: %+v", loaded.Imports)
		}
		for _, ns := range loaded.World.Semantics {
			if string(ns.ID) != "os" {
				continue
			}
			if len(ns.Types) != 1 || string(ns.Types[0].ID) != "os.process_id" ||
				string(ns.Types[0].Underlying) != wantUnderlying {
				t.Fatalf("imported semantic meaning differs from independently authored fixture: %+v", ns.Types)
			}
			return
		}
		t.Fatal("expected provider-owned namespace os is missing")
	}
	assertStrictRefusal := func(loaded workspace.Result) {
		t.Helper()
		if _, err := workspace.BuildSourceIndexFromLoad(consumer, loaded); !errors.Is(err, workspace.ErrImportsNotBound) {
			t.Fatalf("strict Source Index must refuse unbound external imports: %v", err)
		}
	}

	manifestA := manifest("provider-a", "")
	writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"), manifestA)
	loadedA, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	assertImportedType(loadedA, "std.identifier")
	if loadedA.Imports[0].Path != "../provider-a" ||
		loadedA.ManifestIdentity().SHA256 != digest(manifestA) {
		t.Fatalf("initial provider/manifest do not match independent inputs: import=%+v manifest=%+v", loadedA.Imports, loadedA.ManifestIdentity())
	}
	indexA, err := workspace.BuildSourceIndex(consumer, loadedA)
	if err != nil {
		t.Fatal(err)
	}
	assertStrictRefusal(loadedA)
	target := sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"}
	anchorA, err := workspace.LocateImportedSource(consumer, loadedA, "os", target)
	if err != nil {
		t.Fatal(err)
	}
	if anchorA.ProviderPath != "../provider-a" ||
		anchorA.Artifact.Path != "semantic/os/process.tei" ||
		anchorA.Artifact.Fingerprint != digest(sourceA) {
		t.Fatalf("source occurrence or raw bytes differ from independent input: %+v", anchorA)
	}

	// P1 and P2 carry identical semantic bytes, but they are distinct selected
	// providers. The consumer-managed source index does not include either.
	manifestB := manifest("provider-b", "")
	writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"), manifestB)
	loadedB, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	assertImportedType(loadedB, "std.identifier")
	if loadedB.Imports[0].Path != "../provider-b" ||
		loadedB.Imports[0].SemanticDigest != loadedA.Imports[0].SemanticDigest ||
		loadedB.ManifestIdentity().SHA256 != digest(manifestB) ||
		loadedB.ManifestIdentity() == loadedA.ManifestIdentity() {
		t.Fatalf("provider selection must change without inventing a semantic change: A=%+v B=%+v", loadedA.Imports, loadedB.Imports)
	}
	indexB, err := workspace.BuildSourceIndex(consumer, loadedB)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indexA.Artifacts, indexB.Artifacts) ||
		sourceindex.ModelRevision(indexA) != sourceindex.ModelRevision(indexB) {
		t.Fatal("switching only the explicit import must not rewrite consumer-managed source evidence")
	}
	assertStrictRefusal(loadedB)
	anchorB, err := workspace.LocateImportedSource(consumer, loadedB, "os", target)
	if err != nil {
		t.Fatal(err)
	}
	if anchorB.ProviderPath != "../provider-b" || anchorB.Artifact.Fingerprint != digest(sourceA) {
		t.Fatalf("equal semantic contents must not conceal selected provider identity: %+v", anchorB)
	}

	// Change only B. The old consumer-managed Source and its Source Index remain
	// intact, but the selected external meaning changes. A's pin must reject B.
	writeExternalFixture(t, filepath.Join(providerB, "semantic/os/process.tei"), sourceB)
	changedB, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	assertImportedType(changedB, "std.text")
	if changedB.Imports[0].SemanticDigest == loadedA.Imports[0].SemanticDigest {
		t.Fatal("changed external type meaning retained the original semantic digest")
	}
	indexChanged, err := workspace.BuildSourceIndex(consumer, changedB)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indexA.Artifacts, indexChanged.Artifacts) ||
		sourceindex.ModelRevision(indexA) != sourceindex.ModelRevision(indexChanged) {
		t.Fatal("provider-only change unexpectedly altered consumer-managed Source")
	}
	assertStrictRefusal(changedB)
	writeExternalFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		manifest("provider-b", loadedA.Imports[0].SemanticDigest))
	if _, err := workspace.Load(consumer); err == nil || !strings.Contains(err.Error(), "semantic digest mismatch") {
		t.Fatalf("previous provider's semantic pin must reject changed external meaning: %v", err)
	}

	// No assertion here grants current Provider authority, Owner meaning
	// acceptance, atomically pinned imported Source, Build reuse, or Runtime effect.
}

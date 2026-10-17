package studiohost

import (
	"os"
	"path/filepath"
	"testing"
)

// This fixture deliberately keeps all three Workspaces on disk even when the
// Host registers only the consumer. The Host allowlist and the semantic import
// closure are different questions; the tests do not select an authorization policy.
func studioImportScopeFixture(t *testing.T) (consumer, provider, leaf string) {
	t.Helper()
	root := t.TempDir()
	consumer = filepath.Join(root, "consumer")
	provider = filepath.Join(root, "provider")
	leaf = filepath.Join(root, "leaf")
	studioImportScopeWrite(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: provider\n    path: ../provider\n    namespace: provider\n")
	studioImportScopeWrite(t, filepath.Join(consumer, "semantic", "consumer.tei"),
		"namespace consumer\n\ntype LocalId from std.identifier\n")
	studioImportScopeWrite(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nsources:\n  - semantic\nimports:\n  - alias: leaf\n    path: ../leaf\n    namespace: leaf\n")
	studioImportScopeWrite(t, filepath.Join(provider, "semantic", "provider.tei"),
		"namespace provider\n\ntype ProviderId from std.identifier\n")
	studioImportScopeWrite(t, filepath.Join(leaf, "tei.workspace.yaml"),
		"workspace: leaf\nsources:\n  - semantic\n")
	studioImportScopeWrite(t, filepath.Join(leaf, "semantic", "leaf.tei"),
		"namespace leaf\n\ntype LeafId from std.identifier\n")
	return consumer, provider, leaf
}

func studioImportScopeWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func studioImportScopeHasSubject(got CatalogResponse, kind, id string) bool {
	if got.Value == nil {
		return false
	}
	for _, entry := range got.Value.Entries {
		if entry.Subject.Kind == kind && entry.Subject.ID == id {
			return true
		}
	}
	return false
}

// A separately registered provider is the positive interoperability case.
// A provider's own dependency must not become an implicit consumer export.
func TestStudioHostRegisteredSemanticImportUsesCommonOperations(t *testing.T) {
	consumer, provider, leaf := studioImportScopeFixture(t)
	host, err := NewReadOnly([]string{consumer, provider, leaf})
	if err != nil {
		t.Fatal(err)
	}
	catalog := host.CatalogWorkspace("workspace-1", "provider")
	if !catalog.OK || !studioImportScopeHasSubject(catalog, "semantic", "provider") {
		t.Fatalf("registered provider namespace not catalogued: %+v", catalog)
	}
	inspection := host.InspectSubject("workspace-1", "semantic", "provider")
	if !inspection.OK || inspection.Value == nil || inspection.Value.Subject.ID != "provider" {
		t.Fatalf("registered provider namespace not inspectable: %+v", inspection)
	}
	validation := host.ValidateWorkspace("workspace-1")
	if !validation.OK || validation.Value == nil {
		t.Fatalf("registered import workspace not validated through the Host: %+v", validation)
	}
	notExported := host.CatalogWorkspace("workspace-1", "leaf")
	if !notExported.OK || studioImportScopeHasSubject(notExported, "semantic", "leaf") {
		t.Fatalf("provider's transitive leaf must not be re-exported: %+v", notExported)
	}
}

// Characterization only: do not hard-code whether an unregistered semantic
// provider is allowed. That requires the Host/import authorization decision.
// Record all three real Host outcomes, and verify they do not return partial
// values when the Host fails the operation.
func TestStudioHostUnregisteredImportScopeCharacterization(t *testing.T) {
	consumer, _, _ := studioImportScopeFixture(t)
	host, err := NewReadOnly([]string{consumer})
	if err != nil {
		t.Fatal(err)
	}
	catalog := host.CatalogWorkspace("workspace-1", "provider")
	validation := host.ValidateWorkspace("workspace-1")
	inspection := host.InspectSubject("workspace-1", "semantic", "provider")
	t.Logf("unregistered semantic provider: catalogOK=%t catalogFailure=%+v providerVisible=%t validationOK=%t validationFailure=%+v inspectOK=%t inspectFailure=%+v",
		catalog.OK, catalog.Failure, studioImportScopeHasSubject(catalog, "semantic", "provider"),
		validation.OK, validation.Failure, inspection.OK, inspection.Failure)
	if catalog.OK && !studioImportScopeHasSubject(catalog, "semantic", "provider") {
		t.Fatalf("successful catalog unexpectedly omitted the explicit import: %+v", catalog)
	}
	if !catalog.OK && catalog.Value != nil {
		t.Fatalf("failed catalog returned a value: %+v", catalog)
	}
	if !validation.OK && validation.Value != nil {
		t.Fatalf("failed validation returned a value: %+v", validation)
	}
	if !inspection.OK && inspection.Value != nil {
		t.Fatalf("failed inspection returned a value: %+v", inspection)
	}
}

// Even when the consumer does not re-export leaf, the current Loader reaches
// that provider during its recursive Load. A bad leaf therefore invalidates the
// consumer's Load. This checks actual read-dependency behavior rather than
// inferring the read scope from final Catalog entries.
func TestStudioHostTransitiveImportReadDependency(t *testing.T) {
	consumer, provider, leaf := studioImportScopeFixture(t)
	host, err := NewReadOnly([]string{consumer, provider, leaf})
	if err != nil {
		t.Fatal(err)
	}
	before := host.CatalogWorkspace("workspace-1", "provider")
	if !before.OK || !studioImportScopeHasSubject(before, "semantic", "provider") {
		t.Fatalf("baseline import did not load: %+v", before)
	}
	studioImportScopeWrite(t, filepath.Join(leaf, "semantic", "leaf.tei"),
		"not a valid TEI declaration\n")
	after := host.CatalogWorkspace("workspace-1", "provider")
	if after.OK || after.Value != nil || after.Failure == nil || after.Failure.Code != "WORKSPACE_LOAD_FAILED" {
		t.Fatalf("transitive leaf parse failure must prevent a successful Load: %+v", after)
	}
	t.Log("a provider's non-reexported dependency was read during consumer Load")
}

// Import paths themselves can point to symbolic links. This characterizes
// the current Host+Loader behavior without deciding whether an unregistered
// symbolic-link target ought to be authorized in a future multi-user Host.
func TestStudioHostSymlinkImportScopeCharacterization(t *testing.T) {
	consumer, provider, _ := studioImportScopeFixture(t)
	alias := filepath.Join(filepath.Dir(consumer), "provider-alias")
	if err := os.Symlink(provider, alias); err != nil {
		t.Skipf("cannot create Workspace import symlink: %v", err)
	}
	studioImportScopeWrite(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: provider\n    path: ../provider-alias\n    namespace: provider\n")
	host, err := NewReadOnly([]string{consumer})
	if err != nil {
		t.Fatal(err)
	}
	got := host.CatalogWorkspace("workspace-1", "provider")
	t.Logf("unregistered import symlink: success=%t providerVisible=%t failure=%+v",
		got.OK, studioImportScopeHasSubject(got, "semantic", "provider"), got.Failure)
	if got.OK && !studioImportScopeHasSubject(got, "semantic", "provider") {
		t.Fatalf("successful import-symlink catalog lacks provider: %+v", got)
	}
	if !got.OK && got.Value != nil {
		t.Fatalf("rejected import-symlink catalog returned a value: %+v", got)
	}
}

// Existing tests cover pre-operation root replacement for Catalog/Validate.
// Inspect must use the same Host identity check, without returning a value.
func TestStudioHostInspectRejectsPreOperationRootReplacement(t *testing.T) {
	root, replacement := rootReplacementPair(t)
	host, err := NewReadOnly([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	replaceWorkspaceRoot(t, root, replacement)
	got := host.InspectSubject("workspace-1", "semantic", "example")
	if got.OK || got.Value != nil || got.Failure == nil || got.Failure.Code != "WORKSPACE_ROOT_CHANGED" {
		t.Fatalf("Inspect allowed replaced Workspace root: %+v", got)
	}
}

// The semantic model visible to a consumer can remain exactly the same even
// when the transitive provider reads and successfully parses changed source.
// A semantic fingerprint is therefore not evidence of a confined read scope.
// This test is read-only and does not decide an import authorization policy.
func TestStudioHostUnexportedDependencyChangesReadButNotSemanticProjection(t *testing.T) {
	consumer, provider, leaf := studioImportScopeFixture(t)
	host, err := NewReadOnly([]string{consumer, provider, leaf})
	if err != nil {
		t.Fatal(err)
	}
	before := host.CatalogWorkspace("workspace-1", "provider")
	if !before.OK || before.Value == nil {
		t.Fatalf("baseline consumer catalog: %+v", before)
	}
	// Only the non-reexported leaf namespace changes. The imported provider
	// namespace and the consumer's own source remain identical.
	studioImportScopeWrite(t, filepath.Join(leaf, "semantic", "leaf.tei"),
		"namespace leaf\n\ntype LeafId from std.identifier\ntype LeafExtraId from std.identifier\n")
	after := host.CatalogWorkspace("workspace-1", "provider")
	if !after.OK || after.Value == nil {
		t.Fatalf("leaf change must remain a valid consumer load: %+v", after)
	}
	if before.Value.WorldFingerprint != after.Value.WorldFingerprint ||
		before.Value.SourceIndexFingerprint != after.Value.SourceIndexFingerprint ||
		before.Value.ModelRevision != after.Value.ModelRevision {
		t.Fatalf("unexported leaf change altered consumer projection identity: before=%s/%s/%s after=%s/%s/%s",
			before.Value.WorldFingerprint, before.Value.SourceIndexFingerprint, before.Value.ModelRevision,
			after.Value.WorldFingerprint, after.Value.SourceIndexFingerprint, after.Value.ModelRevision)
	}
	// This same leaf remains in the runtime read closure: turning its source
	// invalid must still reject the consumer's attempted read.
	studioImportScopeWrite(t, filepath.Join(leaf, "semantic", "leaf.tei"),
		"not a valid TEI declaration\n")
	invalid := host.CatalogWorkspace("workspace-1", "provider")
	if invalid.OK || invalid.Value != nil || invalid.Failure == nil ||
		invalid.Failure.Code != "WORKSPACE_LOAD_FAILED" {
		t.Fatalf("invalid unexported read dependency was not noticed: %+v", invalid)
	}
}

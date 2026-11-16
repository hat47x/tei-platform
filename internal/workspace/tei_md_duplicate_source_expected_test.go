package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/validation"
)

// These tests fix the Expected for a subject declared twice across two
// different source surfaces: a native *.tei file and a selected *.tei.md
// authoring source. The expectations are written from the authored bytes
// (byte offsets and the literal declaration text), not from the projection's
// own output, so a projection that loses or invents a position fails here.

func writeCrossSurfaceSource(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The fixtures below are deliberately literal so the expected byte offsets can
// be described in terms of the text a reader wrote.
const (
	// A native declaration of the type sales.order_id.
	crossSurfaceNativeSource = "namespace sales\n\ntype OrderId from std.identifier\n"

	// An authored Data table declaring sales.order_id from Markdown.
	crossSurfaceAuthoredSource = "# 注文\n\n" +
		"<!-- tei: data OrderId in sales -->\n" +
		"| attribute | type |\n" +
		"| --- | --- |\n" +
		"| value | std.text |\n"
)

const (
	crossSurfaceNativePath   = "semantic/base.tei"
	crossSurfaceAuthoredPath = "semantic/order.tei.md"
)

func crossSurfaceWorkspace(t *testing.T, languages string) string {
	t.Helper()
	root := t.TempDir()
	writeCrossSurfaceSource(t, root, "tei.workspace.yaml",
		"workspace: cross_surface\nformatVersion: 1\nsources:\n  - semantic\n"+languages)
	writeCrossSurfaceSource(t, root, crossSurfaceNativePath, crossSurfaceNativeSource)
	writeCrossSurfaceSource(t, root, crossSurfaceAuthoredPath, crossSurfaceAuthoredSource)
	return root
}

// A collision between a native and an authored declaration must name both
// physical positions. Load knows the authored byte range; reporting "unknown"
// would hide a position it already has.
func TestDuplicateSemanticSourcesNameAuthoredDeclaration(t *testing.T) {
	root := crossSurfaceWorkspace(t, "languages:\n  - tei.authoring/v1\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("selected authoring source with a cross-surface duplicate failed to load: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Path != crossSurfaceAuthoredPath {
		t.Fatalf("the authored source was not read by the selected language: %+v", loaded.AuthoringSources)
	}
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("same Load-time Source Index: %v", err)
	}

	// Independent expectation: locate both declarations by searching the exact
	// authored bytes, not by reading the projection.
	nativeBytes := []byte(crossSurfaceNativeSource)
	authoredBytes := []byte(crossSurfaceAuthoredSource)
	wantNative := bytes.Index(nativeBytes, []byte("OrderId"))
	wantAuthored := bytes.Index(authoredBytes, []byte("OrderId"))
	if wantNative < 0 || wantAuthored < 0 {
		t.Fatal("fixture does not contain the declared identity")
	}

	projected := ProjectDiagnosticsWithLoadedDuplicates(
		validation.Validate(loaded.World), index, loaded)
	var found bool
	for _, item := range projected {
		if item.Diagnostic.Code != "DUPLICATE_SEMANTIC_ELEMENT" ||
			item.Diagnostic.Subject == nil ||
			item.Diagnostic.Subject.ID != "sales.order_id" {
			continue
		}
		found = true
		if len(item.UnresolvedSourceOrigins) != 0 {
			t.Fatalf("a known authored position was reported as unresolved: %+v", item.UnresolvedSourceOrigins)
		}
		if len(item.Sources) != 2 {
			t.Fatalf("both declarations must be named, got %+v", item.Sources)
		}
		byPath := map[string]sourceindex.Reference{}
		for _, ref := range item.Sources {
			if ref.Role != sourceindex.RoleDefinition {
				t.Fatalf("declaration reference is not a definition: %+v", ref)
			}
			byPath[ref.ArtifactPath] = ref
		}
		native, ok := byPath[crossSurfaceNativePath]
		if !ok {
			t.Fatalf("native declaration position is missing: %+v", item.Sources)
		}
		authored, ok := byPath[crossSurfaceAuthoredPath]
		if !ok {
			t.Fatalf("authored declaration position is missing: %+v", item.Sources)
		}
		if native.Range.Start.Offset != wantNative ||
			string(nativeBytes[native.Range.Start.Offset:native.Range.End.Offset]) != "OrderId" {
			t.Fatalf("native declaration range is not the authored identity: %+v", native.Range)
		}
		if authored.Range.Start.Offset != wantAuthored ||
			string(authoredBytes[authored.Range.Start.Offset:authored.Range.End.Offset]) != "OrderId" {
			t.Fatalf("authored declaration range is not the authored identity: %+v", authored.Range)
		}
		if err := authored.Range.Validate(authoredBytes); err != nil {
			t.Fatalf("authored declaration range is stale: %v", err)
		}
	}
	if !found {
		t.Fatalf("cross-surface duplicate was not diagnosed: %+v", projected)
	}
}

// When the Load-time bytes of a participating source are gone, the projection
// must not present the remaining authored declaration as the complete set. This
// follows the existing convention: missing bytes fall back to the plain
// projection instead of inventing or truncating a candidate list.
func TestDuplicateSemanticSourcesAuthoredOnlyDoesNotClaimCompleteSet(t *testing.T) {
	root := crossSurfaceWorkspace(t, "languages:\n  - tei.authoring/v1\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	diagnostics := validation.Validate(loaded.World)
	// Drop the native snapshot so its declaration position cannot be recovered.
	delete(loaded.sourceSnapshot, crossSurfaceNativePath)

	got := ProjectDiagnosticsWithLoadedDuplicates(diagnostics, index, loaded)
	want := ProjectDiagnostics(diagnostics, index)
	if len(got) != len(want) {
		t.Fatalf("projection changed shape with missing bytes: got=%d want=%d", len(got), len(want))
	}
	for i := range got {
		if got[i].Diagnostic.Code != want[i].Diagnostic.Code {
			t.Fatalf("diagnostic %d changed: %s vs %s", i, got[i].Diagnostic.Code, want[i].Diagnostic.Code)
		}
		if len(got[i].Sources) != 0 || len(got[i].UnresolvedSourceOrigins) != 0 {
			t.Fatalf("a partly unreadable duplicate presented a candidate set: %+v", got[i])
		}
	}
}

// An authored source the Workspace did not select must not become meaning, and
// no position for it may be presented.
func TestDuplicateSemanticSourcesUnselectedAuthoredSourceFailsClosed(t *testing.T) {
	root := crossSurfaceWorkspace(t, "")

	_, err := Load(root)
	if err == nil {
		t.Fatal("an unselected authored source was read as meaning")
	}
	for _, want := range []string{"TEI_MD_PARSER_UNBOUND", crossSurfaceAuthoredPath} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal does not name %q: %v", want, err)
		}
	}
}

// An ordinary Markdown document is not a meaning source, even beside a selected
// authoring source that is. It stays managed and unread.
func TestOrdinaryMarkdownIsNotMeaningBesideSelectedAuthoring(t *testing.T) {
	root := crossSurfaceWorkspace(t, "languages:\n  - tei.authoring/v1\n")
	writeCrossSurfaceSource(t, root, "semantic/notes.md",
		"# 説明\n\n```tei\nnamespace sales\n```\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("ordinary Markdown blocked a valid Workspace: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Path != crossSurfaceAuthoredPath {
		t.Fatalf("ordinary Markdown was admitted as an authoring source: %+v", loaded.AuthoringSources)
	}
	for _, file := range loaded.ManagedFiles {
		if file.Path != "semantic/notes.md" {
			continue
		}
		if file.ReadByLoader {
			t.Fatal("ordinary Markdown was recorded as read by the loader")
		}
	}
}

// An unselected draft outside the declared source roots stays editable and is
// never admitted, while the managed native source keeps loading.
func TestUnmanagedDraftStaysOutsideTheWorkspace(t *testing.T) {
	root := t.TempDir()
	writeCrossSurfaceSource(t, root, "tei.workspace.yaml",
		"workspace: cross_surface\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n")
	writeCrossSurfaceSource(t, root, crossSurfaceNativePath, crossSurfaceNativeSource)
	writeCrossSurfaceSource(t, root, "drafts/wip.tei.md", "# 下書き\n\n```tei\nnamespace sales\n```\n")

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("an unmanaged draft blocked the Workspace: %v", err)
	}
	if len(loaded.AuthoringSources) != 0 {
		t.Fatalf("an unmanaged draft entered the Workspace: %+v", loaded.AuthoringSources)
	}
	for _, file := range loaded.ManagedFiles {
		if file.Path == "drafts/wip.tei.md" {
			t.Fatal("an unmanaged draft was inventoried as a managed file")
		}
	}
}

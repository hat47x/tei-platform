package languagereference

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
	"tei/internal/workspace"
)

// These tests use actual TEI Workspace loading and the existing Go semantic
// parser. The Language Selection identities here are test-only labels, not
// verified plugin/implementation digests or accepted meaning.
func selfDescriptionTestSelection() Selection {
	return Selection{
		LanguageID:                "fixture.tei.semantic",
		ContractVersion:           "1",
		ContractFingerprint:       "test-only-semantic-contract",
		ImplementationID:          "fixture.tei.semantic.parser",
		ImplementationFingerprint: "test-only-semantic-parser",
	}
}

func parseSelfDescriptionCandidate(t *testing.T, src string) semantic.Namespace {
	t.Helper()
	pin := selfDescriptionTestSelection()
	providers := []Provider[semantic.Namespace]{
		{Selection: pin, Parse: func(text string) (semantic.Namespace, error) {
			return syntax.NewParser(text).ParseSemanticFile()
		}},
	}
	got, err := ParseSelected(pin, providers, src)
	if err != nil {
		t.Fatalf("parse exact-selected TEI semantic source: %v", err)
	}
	return got
}

func selfDescriptionAttribute(t *testing.T, ns semantic.Namespace, dataID, attrID string) semantic.Attribute {
	t.Helper()
	for _, data := range ns.Data {
		if string(data.ID) != dataID {
			continue
		}
		for _, attr := range data.Attributes {
			if string(attr.ID) == attrID {
				return attr
			}
		}
	}
	t.Fatalf("semantic attribute %s was not parsed in %s", attrID, dataID)
	return semantic.Attribute{}
}

func TestSelfDescriptionWorkspaceSourceAndRoleBoundary(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "reference", "workspace"))
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("actual Workspace Load: %v", err)
	}
	if loaded.ManifestIdentity().SHA256 == "" {
		t.Fatal("actual Workspace Load has no manifest byte identity")
	}
	source, found := loaded.Sources["semantic-data:sales.order"]
	if !found {
		t.Fatal("actual Workspace Load did not record semantic-data:sales.order")
	}
	const wantPath = "semantic/sales/order.tei"
	if source.File != wantPath {
		t.Fatalf("semantic-data:sales.order source = %q; expected %q", source.File, wantPath)
	}

	// The Source Index is derived from the same Load-time bytes, without a
	// filesystem re-read. This does not establish the FULL workspace closure
	// or a single atomic snapshot across all files and imports.
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("indexed Load-time source: %v", err)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
	if len(refs) != 1 || refs[0].ArtifactPath != wantPath {
		t.Fatalf("Source Index refs for sales.order: %+v", refs)
	}

	// This fresh read is an independent test fixture input. It is not
	// represented as the private pinned byte snapshot from Workspace Load.
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(wantPath)))
	if err != nil {
		t.Fatalf("read public reference fixture: %v", err)
	}
	// A separate read is compared against the Load-time Source Index fingerprint.
	// Equality here is a bounded witness for this fixture, NOT a guarantee
	// that multiple Workspace files were read in one atomic transaction.
	var loadedArtifact *sourceindex.Artifact
	for i := range index.Artifacts {
		if index.Artifacts[i].Path == wantPath {
			loadedArtifact = &index.Artifacts[i]
			break
		}
	}
	if loadedArtifact == nil {
		t.Fatalf("Load-time Source Index has no Artifact for %s", wantPath)
	}
	if got := sourceindex.Fingerprint(raw); got != loadedArtifact.Fingerprint {
		t.Fatalf("independent fixture read differs from Load-time indexed bytes: read=%s loaded=%s", got, loadedArtifact.Fingerprint)
	}
	text := string(raw)
	const original = "attribute total type std.money"
	if strings.Count(text, original) != 1 {
		t.Fatalf("expected exactly one total attribute declaration")
	}
	base := parseSelfDescriptionCandidate(t, text)
	if string(base.ID) != "sales" || base.FormatVersion != 0 {
		t.Fatalf("unexpected namespace or implicit semantic format version: %+v", base)
	}
	baseAttr := selfDescriptionAttribute(t, base, "sales.order", "sales.order.total")
	if string(baseAttr.TypeRef) != "std.money" || baseAttr.Role != "" {
		t.Fatalf("unexpected baseline total attribute: %+v", baseAttr)
	}
	// Independent expected identity and type, not inferred from either
	// migration variant or a self-describing Role evaluator.
	var workspaceAttr semantic.Attribute
	for _, ns := range loaded.World.Semantics {
		if string(ns.ID) != "sales" {
			continue
		}
		workspaceAttr = selfDescriptionAttribute(t, ns, "sales.order", "sales.order.total")
		break
	}
	if workspaceAttr.ID != baseAttr.ID || workspaceAttr.TypeRef != baseAttr.TypeRef {
		t.Fatalf("actual Workspace and direct parser differ: loaded=%+v parsed=%+v", workspaceAttr, baseAttr)
	}

	t.Run("role annotation is ordinary parsed data", func(t *testing.T) {
		variant := strings.Replace(text, original, original+" role accounting_amount", 1)
		ns := parseSelfDescriptionCandidate(t, variant)
		attr := selfDescriptionAttribute(t, ns, "sales.order", "sales.order.total")
		if attr.Role != "accounting_amount" || attr.TypeRef != baseAttr.TypeRef {
			t.Fatalf("role annotation was not preserved as data: %+v", attr)
		}
		// No Role Required Shape, Effect, authority, or Owner Acceptance is
		// interpreted by this test or by ParseSelected.
	})

	t.Run("same identity does not hide attribute type change", func(t *testing.T) {
		variant := strings.Replace(text, original, "attribute total type std.text", 1)
		ns := parseSelfDescriptionCandidate(t, variant)
		attr := selfDescriptionAttribute(t, ns, "sales.order", "sales.order.total")
		if attr.ID != baseAttr.ID || string(attr.TypeRef) != "std.text" {
			t.Fatalf("independent type-drift expectation failed: %+v", attr)
		}
		if attr.TypeRef == baseAttr.TypeRef {
			t.Fatal("same Data/Attribute identity was incorrectly treated as same type")
		}
	})
}

func TestSelfDescriptionSemanticFormatIsNotJustParserSuccess(t *testing.T) {
	// The parser accepts a syntactically valid v2 header as a candidate.
	// The Workspace must separately reject v2 in a v1 manifest, proving
	// syntax readability is not Workspace format qualification.
	base := "formatVersion 2\nnamespace probe\ndata Item {\n  attribute value type std.text\n}\n"
	candidate := parseSelfDescriptionCandidate(t, base)
	if candidate.FormatVersion != 2 || string(candidate.ID) != "probe" {
		t.Fatalf("unexpected v2 semantic candidate: %+v", candidate)
	}

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: self_description_probe\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "probe.tei"), []byte(base), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "does not match the Workspace formatVersion") {
		t.Fatalf("expected explicit manifest/semantic format-version conflict, got %v", err)
	}
}

// This test makes the "separate read" caveat observable. The existing
// Workspace Load and Source Index remain bound to their original bytes even
// when a subsequent unrelated read sees newer content at the same path.
func TestSelfDescriptionRejectsStaleDirectReadAsLoadTimeSource(t *testing.T) {
	root := t.TempDir()
	sourceDir := filepath.Join(root, "semantic")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: self_description_drift\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	const path = "semantic/probe.tei"
	original := "namespace probe\ndata item {\n  attribute value type std.text\n}\n"
	sourcePath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.WriteFile(sourcePath, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load original source: %v", err)
	}
	indexBefore, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("index original Load-time source: %v", err)
	}
	findFingerprint := func(index sourceindex.Index) string {
		t.Helper()
		for _, artifact := range index.Artifacts {
			if artifact.Path == path {
				return artifact.Fingerprint
			}
		}
		t.Fatalf("index has no Artifact for %s", path)
		return ""
	}
	originalFingerprint := findFingerprint(indexBefore)
	if want := sourceindex.Fingerprint([]byte(original)); originalFingerprint != want {
		t.Fatalf("Load-time Artifact fingerprint %s; expected %s", originalFingerprint, want)
	}

	// Preserve the same file locator and semantic identity; change only an
	// attribute type after Load, as in a racing editor or external rewrite.
	changed := strings.Replace(original, "std.text", "std.money", 1)
	if err := os.WriteFile(sourcePath, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	laterRead, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := sourceindex.Fingerprint(laterRead); got == originalFingerprint {
		t.Fatal("mutated source incorrectly matched Load-time bytes")
	}
	indexFromOriginalLoad, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("index original Load after source mutation: %v", err)
	}
	if got := findFingerprint(indexFromOriginalLoad); got != originalFingerprint {
		t.Fatalf("original Load-time Artifact changed after filesystem mutation: %s != %s", got, originalFingerprint)
	}
	parsed := parseSelfDescriptionCandidate(t, string(laterRead))
	attr := selfDescriptionAttribute(t, parsed, "probe.item", "probe.item.value")
	if string(attr.TypeRef) != "std.money" {
		t.Fatalf("later parser did not see the changed TypeRef: %+v", attr)
	}
	// Without an explicit byte-identity check, the two valid parsed sources
	// would be falsely presented as a single Source revision. This fixture
	// does not assert that the product itself refuses a caller's later read.
}

// This is a characterization of the current Go *syntax candidate*, not a
// Product Role-resolution or acceptance test. A string inside description
// never becomes a Role, while duplicate role clauses currently lose their
// first occurrence at the parser-candidate boundary. Source-aware validation
// must not infer uniqueness from the resulting single string.
func TestSelfDescriptionRoleLiteralAndDuplicateAreNotAcceptedMeaning(t *testing.T) {
	quoted := "namespace probe\ndata item {\n  attribute value type std.money description \"role privileged_amount\"\n}\n"
	ns := parseSelfDescriptionCandidate(t, quoted)
	attr := selfDescriptionAttribute(t, ns, "probe.item", "probe.item.value")
	if attr.Role != "" || attr.Description != "role privileged_amount" {
		t.Fatalf("description text was incorrectly promoted to a Role: %+v", attr)
	}

	oneRole := "namespace probe\ndata item {\n  attribute value type std.money role audit_amount\n}\n"
	twoRoles := "namespace probe\ndata item {\n  attribute value type std.money role accounting_amount role audit_amount\n}\n"
	if sourceindex.Fingerprint([]byte(oneRole)) == sourceindex.Fingerprint([]byte(twoRoles)) {
		t.Fatal("distinct role declaration sources unexpectedly share a byte fingerprint")
	}
	single := parseSelfDescriptionCandidate(t, oneRole)
	duplicate, err := syntax.NewParser(twoRoles).ParseSemanticFile()
	if err != nil {
		// A future parser may correctly reject a duplicate at the syntax
		// boundary; that is also a safe outcome for this negative control.
		return
	}
	singleAttr := selfDescriptionAttribute(t, single, "probe.item", "probe.item.value")
	duplicateAttr := selfDescriptionAttribute(t, duplicate, "probe.item", "probe.item.value")
	if singleAttr.Role != "audit_amount" || duplicateAttr.Role != singleAttr.Role ||
		duplicateAttr.ID != singleAttr.ID || duplicateAttr.TypeRef != singleAttr.TypeRef {
		t.Fatalf("unexpected current Role candidate projection: single=%+v duplicate=%+v", singleAttr, duplicateAttr)
	}
	// The current parser gives no duplicate-Role diagnostic: distinct
	// source revisions project to the same role/type fields. A future
	// semantic validator may still reject the duplicate; this test does
	// NOT count as proof of acceptance or of a complete rejection path.
	t.Log("current syntax candidate loses duplicate Role occurrence; Role uniqueness remains unqualified")
}

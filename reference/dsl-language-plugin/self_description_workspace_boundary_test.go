package languagereference

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/currencyrate"
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

// Source navigation is not a proof that an option token is writable. The
// real syntax anchor API identifies the attribute symbol and type token;
// Role option editing requires its own uniquely resolved source span and
// Shared Acceptance path, neither of which is supplied by these anchors.
func TestSelfDescriptionAttributeAnchorsDoNotSubstituteForRoleOptionSpan(t *testing.T) {
	const text = "namespace sales\ndata order {\n  attribute total type std.money role accounting_amount\n}\n"
	anchored, err := syntax.ParseSemanticFileAnchored(text)
	if err != nil {
		t.Fatalf("parse real syntax anchors: %v", err)
	}
	attr := selfDescriptionAttribute(t, anchored.Value, "sales.order", "sales.order.total")
	if attr.Role != "accounting_amount" || string(attr.TypeRef) != "std.money" {
		t.Fatalf("semantic candidate lost Role or TypeRef: %+v", attr)
	}
	roles := 0
	seen := map[string]bool{}
	roleStart := strings.Index(text, "role accounting_amount")
	if roleStart < 0 {
		t.Fatal("fixed source lacks the role clause")
	}
	for _, anchor := range anchored.Anchors {
		if anchor.ID != "sales.order.total" {
			continue
		}
		start, end := anchor.Anchor.Start.Offset, anchor.Anchor.End.Offset
		if start < 0 || end < start || end > len(text) {
			t.Fatalf("invalid anchored source range for %s: [%d,%d)", anchor.Kind, start, end)
		}
		fragment := text[start:end]
		switch anchor.Kind {
		case "semantic-attribute":
			if fragment != "total" || end > roleStart {
				t.Fatalf("attribute symbol anchor cannot stand in for Role clause: %q", fragment)
			}
			seen[anchor.Kind] = true
		case "semantic-attribute-type":
			if fragment != "std.money" || end > roleStart {
				t.Fatalf("type reference anchor cannot stand in for Role clause: %q", fragment)
			}
			seen[anchor.Kind] = true
		case "semantic-attribute-role":
			// Do not make a future explicit Role anchor a regression; it
			// would still require a safe writer, version and acceptance.
			roles++
			if fragment != "accounting_amount" {
				t.Fatalf("new Role anchor must refer to the actual option value: %q", fragment)
			}
		}
	}
	if !seen["semantic-attribute"] || !seen["semantic-attribute-type"] {
		t.Fatalf("existing symbol and type source anchors are missing: %+v", seen)
	}
	if roles == 0 {
		t.Log("currently no explicit Role value anchor: only a read-only Role candidate, not a Role patch position")
	}
	// No Source write, Scope/Authority check, Role Required Shape or Owner
	// meaning acceptance is performed by this source-navigation test.
}

// A Markdown document that contains Role-like words can be inventoried and
// byte-indexed without becoming a semantic declaration. This uses the real
// Workspace Loader and Load-time Source Index, not a second Markdown parser.
// The docs root is currently a forward-compatible/unknown source root.
func TestSelfDescriptionManagedRawDocumentDoesNotBecomeSemanticRole(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"semantic", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	manifest := "workspace: raw_self_description\nformatVersion: 1\nsources:\n  - semantic\n  - docs\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	semanticSource := "namespace probe\ndata item {\n  attribute value type std.text\n}\n"
	if err := os.WriteFile(filepath.Join(root, "semantic", "probe.tei"), []byte(semanticSource), 0600); err != nil {
		t.Fatal(err)
	}
	const rawPath = "docs/design.md"
	raw := []byte("# Unaccepted design draft\n\nTask and Event are possible roles.\n\n" +
		"Proposed: namespace hidden; data forged; role trusted_amount.\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rawPath)), raw, 0600); err != nil {
		t.Fatal(err)
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load managed semantic and raw documentation: %v", err)
	}
	managed := make(map[string]workspace.ManagedFile)
	for _, item := range loaded.ManagedFiles {
		managed[item.Path] = item
	}
	if item, ok := managed[rawPath]; !ok || item.ReadByLoader {
		t.Fatalf("raw Markdown must be inventoried but not parsed: %+v, present=%v", item, ok)
	}
	if item, ok := managed["semantic/probe.tei"]; !ok || !item.ReadByLoader {
		t.Fatalf("semantic source must be read by the actual domain Loader: %+v, present=%v", item, ok)
	}
	if _, ok := loaded.Sources["semantic-data:hidden.forged"]; ok {
		t.Fatal("Role-like prose in Markdown became an accepted semantic Subject")
	}

	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("index real Load-time managed bytes: %v", err)
	}
	var foundArtifact bool
	for _, artifact := range index.Artifacts {
		if artifact.Path != rawPath {
			continue
		}
		foundArtifact = true
		if want := sourceindex.Fingerprint(raw); artifact.Fingerprint != want {
			t.Fatalf("raw managed Artifact fingerprint = %s, expected %s", artifact.Fingerprint, want)
		}
	}
	if !foundArtifact {
		t.Fatal("managed raw Markdown was omitted from the Source Index Artifact inventory")
	}
	for _, reference := range index.References {
		if reference.ArtifactPath == rawPath {
			t.Fatalf("unparsed Markdown acquired a semantic definition Reference: %+v", reference)
		}
	}
	if len(index.ReferencesFor(sourceindex.Subject{Kind: "semantic-data", ID: "probe.item"})) != 1 {
		t.Fatal("independent semantic source was unexpectedly not indexed")
	}

	// The next filesystem read is not the original Load. Even if the
	// original byte-indexed asset survives, it never proves a Role contract.
	changed := []byte("# Edited unaccepted draft\n\nrole elevated_amount\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rawPath)), changed, 0600); err != nil {
		t.Fatal(err)
	}
	originalIndex, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("re-index the original Load-time bytes: %v", err)
	}
	for _, artifact := range originalIndex.Artifacts {
		if artifact.Path == rawPath {
			if artifact.Fingerprint != sourceindex.Fingerprint(raw) ||
				artifact.Fingerprint == sourceindex.Fingerprint(changed) {
				t.Fatalf("Load-time raw asset retroactively changed: %+v", artifact)
			}
			return
		}
	}
	t.Fatal("original Load-time raw asset disappeared from Source Index")
}

// Raw document edits change a managed Source Artifact but not the parsed
// semantic World. This is a bounded, real Loader/Index comparison; it does
// not qualify the document for disclosure or independently accept its meaning.
func TestSelfDescriptionRawDocumentRevisionIsNotSemanticRoleRevision(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"semantic", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}
	const semanticPath = "semantic/probe.tei"
	const documentPath = "docs/design.md"
	manifest := []byte("workspace: raw_revision\nformatVersion: 1\nsources:\n  - semantic\n  - docs\n")
	semanticBytes := []byte("namespace probe\ndata item {\n  attribute value type std.money\n}\n")
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), manifest, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(semanticPath)), semanticBytes, 0600); err != nil {
		t.Fatal(err)
	}
	// Both revisions mention something resembling a Role definition, but
	// neither is a selected semantic language source.
	rawA := []byte("# Working notes\nrole candidate_amount\n")
	rawB := []byte("# Revised notes\nrole approved_amount\n")
	document := filepath.Join(root, filepath.FromSlash(documentPath))
	if err := os.WriteFile(document, rawA, 0600); err != nil {
		t.Fatal(err)
	}
	load := func() (workspace.Result, sourceindex.Index) {
		t.Helper()
		result, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("load actual Workspace: %v", err)
		}
		index, err := workspace.BuildSourceIndexFromLoad(root, result)
		if err != nil {
			t.Fatalf("index Load-time bytes: %v", err)
		}
		return result, index
	}
	before, beforeIndex := load()
	if err := os.WriteFile(document, rawB, 0600); err != nil {
		t.Fatal(err)
	}
	after, afterIndex := load()
	if before.ManifestIdentity().SHA256 != after.ManifestIdentity().SHA256 {
		t.Fatal("unchanged manifest acquired a different byte identity")
	}
	if !reflect.DeepEqual(before.World, after.World) {
		t.Fatal("a raw, unparsed document edit changed the semantic World")
	}
	if !reflect.DeepEqual(before.Sources, after.Sources) {
		t.Fatal("a raw document edit silently created or changed a semantic Source")
	}
	for _, result := range []workspace.Result{before, after} {
		found := false
		for _, file := range result.ManagedFiles {
			if file.Path == documentPath {
				found = true
				if file.ReadByLoader {
					t.Fatal("raw Markdown became parsed semantic source")
				}
			}
		}
		if !found {
			t.Fatal("raw Markdown was not inventoried")
		}
	}
	artifactDigest := func(index sourceindex.Index) string {
		t.Helper()
		for _, artifact := range index.Artifacts {
			if artifact.Path == documentPath {
				return artifact.Fingerprint
			}
		}
		t.Fatal("no Load-time Artifact for managed raw document")
		return ""
	}
	if got := artifactDigest(beforeIndex); got != sourceindex.Fingerprint(rawA) {
		t.Fatalf("before raw artifact fingerprint = %s", got)
	}
	if got := artifactDigest(afterIndex); got != sourceindex.Fingerprint(rawB) {
		t.Fatalf("after raw artifact fingerprint = %s", got)
	}
	if artifactDigest(beforeIndex) == artifactDigest(afterIndex) {
		t.Fatal("different raw Source bytes collapsed into one indexed revision")
	}
	if !reflect.DeepEqual(beforeIndex.References, afterIndex.References) {
		t.Fatal("raw document revision changed canonical semantic references")
	}
	for _, reference := range afterIndex.References {
		if reference.ArtifactPath == documentPath {
			t.Fatalf("raw Markdown acquired a definition Reference: %+v", reference)
		}
	}
	if len(afterIndex.ReferencesFor(sourceindex.Subject{Kind: "semantic-data", ID: "probe.item"})) != 1 {
		t.Fatal("canonical subject reference missing after raw document edit")
	}
}

// The actual TEI workspace syntax is consumed by the existing domain Role
// reader. It proves a narrow Source -> Load -> IR Role -> domain rule path,
// not an exact-selected Language Plugin, Provider Read, or owner acceptance.
func TestSelfDescriptionWorkspaceRoleFeedsExistingCurrencyFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: currency_role_probe\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	const sourcePath = "semantic/rate.tei"
	withRole := "namespace fx\ndata Rate {\n  attribute fallback type std.code role currency-rate-fallback\n}\n"
	sourceFile := filepath.Join(root, filepath.FromSlash(sourcePath))
	if err := os.WriteFile(sourceFile, []byte(withRole), 0600); err != nil {
		t.Fatal(err)
	}
	load := func() (workspace.Result, sourceindex.Index) {
		t.Helper()
		w, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("load TEI semantic file: %v", err)
		}
		idx, err := workspace.BuildSourceIndexFromLoad(root, w)
		if err != nil {
			t.Fatalf("index TEI semantic file: %v", err)
		}
		return w, idx
	}
	loaded, index := load()
	if len(index.ReferencesFor(sourceindex.Subject{Kind: "semantic-data", ID: "fx.rate"})) != 1 {
		t.Fatal("real Workspace did not index the declared rate Data")
	}
	if got := selfDescriptionAttribute(t, loaded.World.Semantics[0], "fx.rate", "fx.rate.fallback"); got.Role != currencyrate.RoleFallback {
		t.Fatalf("actual Workspace dropped the fallback Role: %+v", got)
	}
	role, err := currencyrate.ResolveFallback(loaded.World, core.CanonicalID("fx.rate"))
	if err != nil {
		t.Fatalf("domain Role resolver failed on the actual Workspace World: %v", err)
	}
	if role.Attribute != "fx.rate.fallback" {
		t.Fatalf("domain resolver selected wrong attribute: %+v", role)
	}
	rows := []map[core.CanonicalID]string{
		{"fx.rate.fallback": "annual"},
		{"fx.rate.fallback": "annual"},
	}
	got, err := role.FallbackRows(rows)
	if err != nil || got != currencyrate.FallbackAnnual {
		t.Fatalf("role and explicit row values did not yield annual: rule=%q err=%v", got, err)
	}
	if _, err := role.FallbackRows([]map[core.CanonicalID]string{
		{"fx.rate.fallback": "annual"},
		{"fx.rate.fallback": "unknown"},
	}); err == nil {
		t.Fatal("selected Role accepted an unknown row rule")
	}
	if _, err := role.FallbackRows([]map[core.CanonicalID]string{
		{"fx.rate.fallback": "annual"},
		{"fx.rate.fallback": "none"},
	}); err == nil {
		t.Fatal("selected Role accepted row order-dependent fallback rules")
	}

	// The same parsed Role declaration can produce a different rule from
	// different input rows. Domain data values are not semantic Source bytes.
	noneRows := []map[core.CanonicalID]string{
		{"fx.rate.fallback": "none"},
		{"fx.rate.fallback": "none"},
	}
	fromDifferentRows, err := role.FallbackRows(noneRows)
	if err != nil || fromDifferentRows != currencyrate.FallbackNone {
		t.Fatalf("same Role declaration with different rows did not select none: rule=%q err=%v", fromDifferentRows, err)
	}
	if fromDifferentRows == got {
		t.Fatal("different row values collapsed into the same domain rule")
	}

	// Change only the annotated Role, not the Data/Attribute or row values.
	// A domain-specific reader must not infer the role from a column name.
	withoutRole := strings.Replace(withRole, " role currency-rate-fallback", "", 1)
	if err := os.WriteFile(sourceFile, []byte(withoutRole), 0600); err != nil {
		t.Fatal(err)
	}
	after, afterIndex := load()
	if len(afterIndex.ReferencesFor(sourceindex.Subject{Kind: "semantic-data", ID: "fx.rate"})) != 1 {
		t.Fatal("removing Role also removed the data source reference")
	}
	unannotated, err := currencyrate.ResolveFallback(after.World, core.CanonicalID("fx.rate"))
	if err != nil {
		t.Fatalf("no-Role declaration should still be readable: %v", err)
	}
	if unannotated.Attribute != "" {
		t.Fatalf("domain inferred Role from the same row/column name: %+v", unannotated)
	}
	noRule, err := unannotated.FallbackRows(rows)
	if err != nil || noRule != currencyrate.FallbackNone {
		t.Fatalf("no-Role data unexpectedly selected a fallback rule: rule=%q err=%v", noRule, err)
	}
	// This is a characterization of the existing domain API, not evidence
	// that an empty or incomplete Provider Read was independently qualified.
	empty, err := role.FallbackRows(nil)
	if err != nil || empty != currencyrate.FallbackNone {
		t.Fatalf("unexpected current empty-row behavior: rule=%q err=%v", empty, err)
	}
}

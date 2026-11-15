package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// copyRuleMarkdownFixture creates an independent Workspace for every mutation;
// none of the tests modify the checked-in authoring example.
func copyRuleMarkdownFixture(t *testing.T) (string, string, []byte) {
	t.Helper()
	root := t.TempDir()
	fixture := filepath.Join(repoRoot(t), "internal", "workspace", "testdata", "tei-md-rule")
	paths := []string{
		"tei.workspace.yaml",
		"semantic/sales/order.tei",
		"semantic/sales/order-line-rules.tei.md",
	}
	for _, rel := range paths {
		raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		writeSource(t, root, rel, raw)
	}
	const markdownPath = "semantic/sales/order-line-rules.tei.md"
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(markdownPath)))
	if err != nil {
		t.Fatal(err)
	}
	return root, markdownPath, raw
}

func TestProductRuleMarkdownSelectedSourceIndexKeepsLoadBytes(t *testing.T) {
	root, path, before := copyRuleMarkdownFixture(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("selected Rule authoring source not loaded: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 {
		t.Fatalf("selected authoring source count = %d, want 1", len(loaded.AuthoringSources))
	}
	read := loaded.AuthoringSources[0]
	if read.Path != path || read.Regions != 2 || read.Definitions != 2 ||
		read.Language.LanguageID != "tei.rule-authoring" ||
		read.Language.ContractVersion != "v0" ||
		read.Language.ImplementationID != "tei.builtin.rule-authoring-markdown" {
		t.Fatalf("actual language/region consumption differs: %+v", read)
	}
	consumed := false
	for _, file := range loaded.ManagedFiles {
		if file.Path == path {
			consumed = file.ReadByLoader
		}
	}
	if !consumed {
		t.Fatal("selected Markdown source was not observed as ReadByLoader")
	}
	idx, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("same Load-time Source Index: %v", err)
	}
	const ruleID = "sales.order_line.quantity_positive"
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
	if len(refs) != 1 {
		t.Fatalf("rule %s references = %+v, want one exact source", ruleID, refs)
	}
	rng := refs[0].Range
	if refs[0].ArtifactPath != path ||
		rng.Start.Offset < 0 || rng.End.Offset > len(before) ||
		rng.Start.Offset >= rng.End.Offset ||
		string(before[rng.Start.Offset:rng.End.Offset]) != ruleID {
		t.Fatalf("rule identity is not anchored to the original bytes: %+v", refs[0])
	}
	artifactFingerprint := ""
	for _, artifact := range idx.Artifacts {
		if artifact.Path == path {
			artifactFingerprint = artifact.Fingerprint
		}
	}
	if artifactFingerprint != sourceindex.Fingerprint(before) {
		t.Fatalf("Source Index is not bound to read bytes: %q", artifactFingerprint)
	}
	readCurrent := func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	}
	if err := idx.Validate(readCurrent); err != nil {
		t.Fatalf("unchanged source index is invalid: %v", err)
	}

	// Prose is not a Rule; its edit must nevertheless invalidate the original
	// Source Index. Building an index from the previous Load must not reread it.
	after := []byte(strings.Replace(string(before), "入力の時点で止めます。", "入力時に止めます。", 1))
	if string(after) == string(before) {
		t.Fatal("prose mutation did not change the fixture")
	}
	writeSource(t, root, path, after)
	if err := idx.Validate(readCurrent); err == nil {
		t.Fatal("stale Source Index was accepted after a prose-only edit")
	}
	oldSnapshot, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	oldRefs := oldSnapshot.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
	if len(oldRefs) != 1 || oldRefs[0].Range != refs[0].Range {
		t.Fatalf("an old Load reread or moved the source anchor: before=%+v after=%+v", refs, oldRefs)
	}
	reloaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("revised explanatory prose changed Rule load: %v", err)
	}
	newIndex, err := workspace.BuildSourceIndexFromLoad(root, reloaded)
	if err != nil {
		t.Fatal(err)
	}
	if err := newIndex.Validate(readCurrent); err != nil {
		t.Fatalf("new Load-time index does not match revised source: %v", err)
	}
	foundNew := false
	for _, artifact := range newIndex.Artifacts {
		if artifact.Path == path {
			foundNew = true
			if artifact.Fingerprint == artifactFingerprint ||
				artifact.Fingerprint != sourceindex.Fingerprint(after) {
				t.Fatalf("a prose edit failed to update raw Source revision: %s", artifact.Fingerprint)
			}
		}
	}
	if !foundNew {
		t.Fatal("new Source Index lost selected Markdown")
	}
}

func TestProductRuleMarkdownRequiresExactLanguageSelection(t *testing.T) {
	for _, tc := range []struct {
		name       string
		manifest   func(string) string
		changeBody bool
		want       string
	}{
		{
			name: "not selected",
			manifest: func(s string) string {
				return strings.Replace(s, "languages:\n  - tei.rule-authoring/v0\n", "", 1)
			},
			want: "TEI_MD_PARSER_UNBOUND",
		},
		{
			name: "unavailable contract",
			manifest: func(s string) string {
				return strings.Replace(s, "tei.rule-authoring/v0", "tei.rule-authoring/v1", 1)
			},
			want: "TEI_LANGUAGE_UNAVAILABLE",
		},
		{
			name:       "invalid definition fence",
			changeBody: true,
			want:       "tei",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, path, original := copyRuleMarkdownFixture(t)
			if tc.manifest != nil {
				manifestPath := filepath.Join(root, "tei.workspace.yaml")
				manifest, err := os.ReadFile(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				next := tc.manifest(string(manifest))
				if next == string(manifest) {
					t.Fatal("manifest mutation did not change the fixture")
				}
				writeSource(t, root, "tei.workspace.yaml", []byte(next))
			}
			if tc.changeBody {
				next := strings.Replace(string(original), "\n"+string([]byte{96, 96, 96})+"tei\n", "\n"+string([]byte{96, 96, 96})+"TEI\n", 1)
				if next == string(original) {
					t.Fatal("fence mutation did not change the fixture")
				}
				writeSource(t, root, path, []byte(next))
			}
			if _, err := workspace.Load(root); err == nil ||
				!strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.want)) {
				t.Fatalf("invalid selection/source was not refused with %q: %v", tc.want, err)
			}
		})
	}
}

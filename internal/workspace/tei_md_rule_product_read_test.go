package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// This tests the actual selected built-in Workspace reader, not a reference
// parser called directly with a synthetic language pin. The resulting Source
// Index is a read-only locator; it is not semantic acceptance or authority.
func TestBuiltinRuleMarkdownProductAdmissionAndSourceIndex(t *testing.T) {
	root := filepath.Join("testdata", "tei-md-rule")
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	original, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Load(root)
	if err != nil {
		t.Fatalf("selected rule Markdown failed the product Load: %v", err)
	}
	if len(result.AuthoringSources) != 1 {
		t.Fatalf("product Load read %d authoring sources, want 1", len(result.AuthoringSources))
	}
	read := result.AuthoringSources[0]
	if read.Path != sourcePath ||
		read.Language.LanguageID != "tei.rule-authoring" ||
		read.Language.ContractVersion != "v0" ||
		read.Language.ImplementationID != "tei.builtin.rule-authoring-markdown" ||
		read.Regions != 2 || read.Definitions != 2 {
		t.Fatalf("unexpected actual product reader witness: %+v", read)
	}
	readByLoader := false
	for _, managed := range result.ManagedFiles {
		if managed.Path == sourcePath {
			readByLoader = managed.ReadByLoader
		}
	}
	if !readByLoader {
		t.Fatal("selected Markdown was managed but not read by the product loader")
	}

	// The authored predicates must reach the existing typed semantic model,
	// not merely be counted as parsed Markdown regions.
	expected := map[string]struct{ kind, attribute, relation string }{
		"sales.order_line.quantity_positive":   {"positive", "sales.order_line.quantity", ""},
		"sales.order.lines.line_number_unique": {"relation-unique", "sales.order_line.line_number", "sales.order.lines"},
	}
	seen := map[string]bool{}
	for _, ns := range result.World.Semantics {
		if string(ns.ID) != "sales" {
			continue
		}
		for _, rule := range ns.Validations {
			id := string(rule.ID)
			want, ok := expected[id]
			if !ok || seen[id] || string(rule.Kind) != want.kind ||
				string(rule.Attribute) != want.attribute || string(rule.Relation) != want.relation {
				t.Fatalf("product loader changed an authored validation: %+v", rule)
			}
			seen[id] = true
		}
	}
	if len(seen) != len(expected) {
		t.Fatalf("product loader yielded %d expected typed validations, want %d", len(seen), len(expected))
	}

	index, err := BuildSourceIndexFromLoad(root, result)
	if err != nil {
		t.Fatalf("product Load byte snapshot did not produce a Source Index: %v", err)
	}
	foundArtifact := false
	for _, artifact := range index.Artifacts {
		if artifact.Path == sourcePath {
			foundArtifact = true
			if artifact.Format != "md" || artifact.Fingerprint != sourceindex.Fingerprint(original) {
				t.Fatalf("source artifact lost exact Markdown bytes: %+v", artifact)
			}
		}
	}
	if !foundArtifact {
		t.Fatal("selected Markdown is missing from the product Source Index")
	}
	for _, id := range []string{
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
	} {
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: id})
		matches := 0
		for _, ref := range refs {
			if ref.ArtifactPath != sourcePath || ref.Role != sourceindex.RoleDefinition {
				continue
			}
			if err := ref.Range.Validate(original); err != nil {
				t.Fatalf("validation %s has an invalid original-byte range: %v", id, err)
			}
			if got := string(original[ref.Range.Start.Offset:ref.Range.End.Offset]); got != id {
				t.Fatalf("validation %s indexed %q instead of its authored ID", id, got)
			}
			matches++
		}
		if matches != 1 {
			t.Fatalf("validation %s has %d product Source references, want exactly 1", id, matches)
		}
	}

	// A changed source is not covered by the old index, even when its
	// semantic rules would remain unchanged (the appended text is prose).
	changed := append(append([]byte(nil), original...), []byte("\nOther prose.\n")...)
	err = index.Validate(func(path string) ([]byte, error) {
		if path == sourcePath {
			return changed, nil
		}
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	})
	if err == nil || !strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("old Source Index was accepted for modified Markdown prose: %v", err)
	}
}

// The built-in contract is explicit: naming a managed .tei.md file alone never
// selects its parser, and a nearby but unsupported version cannot fall back.
func TestBuiltinRuleMarkdownProductAdmissionRejectsUnselectedOrUnknownLanguage(t *testing.T) {
	fixture := filepath.Join("testdata", "tei-md-rule")
	manifest, err := os.ReadFile(filepath.Join(fixture, "tei.workspace.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	const selection = "languages:\n  - tei.rule-authoring/v0\n"
	if strings.Count(string(manifest), selection) != 1 {
		t.Fatal("selected-language fixture no longer has exactly one v0 declaration")
	}
	files := []string{
		"semantic/sales/order-line-rules.tei.md",
		"semantic/sales/order.tei",
	}
	for _, test := range []struct {
		name, manifest, code string
	}{
		{"no-language", strings.Replace(string(manifest), selection, "", 1), "TEI_MD_PARSER_UNBOUND"},
		{"wrong-version", strings.Replace(string(manifest), "tei.rule-authoring/v0", "tei.rule-authoring/v99", 1), "TEI_LANGUAGE_UNAVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for _, path := range files {
				original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
				if err != nil {
					t.Fatal(err)
				}
				destination := filepath.Join(root, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(destination, original, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(test.manifest), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatalf("unselected or unavailable language entered product World: result=%+v err=%v", got.AuthoringSources, err)
			}
		})
	}
}

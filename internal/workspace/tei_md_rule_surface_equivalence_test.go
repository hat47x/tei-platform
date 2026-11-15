package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

// The old validation syntax and the selected Markdown rule syntax must produce
// the same typed business meaning through the actual product Workspace loader.
// Matching a parser's candidate, or matching an output value, is not enough.
func TestRuleMarkdownAndLegacyValidationHaveIdenticalProductMeaning(t *testing.T) {
	markdownRoot := filepath.Join("testdata", "tei-md-rule")
	legacyRoot := t.TempDir()
	const markdownSource = "semantic/sales/order-line-rules.tei.md"
	const nativeSource = "semantic/sales/validation.tei"

	order, err := os.ReadFile(filepath.Join(markdownRoot, "semantic", "sales", "order.tei"))
	if err != nil {
		t.Fatal(err)
	}
	// This established native-DSL fixture is independent of the new rule parser.
	legacyRules, err := os.ReadFile(filepath.Join("..", "..", "reference", "workspace", "semantic", "sales", "validation.tei"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range []struct {
		path string
		data []byte
	}{
		{"tei.workspace.yaml", []byte("workspace: legacy_rule_equivalence\nformatVersion: 1\nsources:\n  - semantic\n")},
		{"semantic/sales/order.tei", order},
		{nativeSource, legacyRules},
	} {
		destination := filepath.Join(legacyRoot, filepath.FromSlash(file.path))
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, file.data, 0600); err != nil {
			t.Fatal(err)
		}
	}

	markdown, err := Load(markdownRoot)
	if err != nil {
		t.Fatalf("selected Markdown product Load: %v", err)
	}
	native, err := Load(legacyRoot)
	if err != nil {
		t.Fatalf("legacy validation product Load: %v", err)
	}
	ids := []string{
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
	}
	find := func(result Result, id string) (semantic.Validation, bool) {
		for _, namespace := range result.World.Semantics {
			if string(namespace.ID) != "sales" {
				continue
			}
			for _, validation := range namespace.Validations {
				if string(validation.ID) == id {
					return validation, true
				}
			}
		}
		return semantic.Validation{}, false
	}
	for _, id := range ids {
		fromMarkdown, ok := find(markdown, id)
		if !ok {
			t.Fatalf("selected Markdown lost validation %s", id)
		}
		fromNative, ok := find(native, id)
		if !ok {
			t.Fatalf("legacy .tei lost validation %s", id)
		}
		// Compare the full typed value, including ID, Symbol, Kind, Attribute
		// and Relation. No special meaning model is created for Markdown.
		if !reflect.DeepEqual(fromMarkdown, fromNative) {
			t.Fatalf("surface change altered semantic.Validation %s: markdown=%+v native=%+v",
				id, fromMarkdown, fromNative)
		}
	}

	markdownIndex, err := BuildSourceIndexFromLoad(markdownRoot, markdown)
	if err != nil {
		t.Fatalf("Markdown product index: %v", err)
	}
	nativeIndex, err := BuildSourceIndexFromLoad(legacyRoot, native)
	if err != nil {
		t.Fatalf("native product index: %v", err)
	}
	for _, id := range ids {
		subject := sourceindex.Subject{Kind: "semantic-validation", ID: id}
		for _, item := range []struct {
			name, path string
			index      sourceindex.Index
		}{
			{"markdown", markdownSource, markdownIndex},
			{"legacy", nativeSource, nativeIndex},
		} {
			matches := 0
			for _, ref := range item.index.ReferencesFor(subject) {
				if ref.ArtifactPath == item.path && ref.Role == sourceindex.RoleDefinition {
					matches++
					if ref.Range.Start.Offset >= ref.Range.End.Offset {
						t.Fatalf("%s lost the original rule identity byte span: %+v", item.name, ref)
					}
				}
			}
			if matches != 1 {
				t.Fatalf("%s has %d source references for %s, want exactly one", item.name, matches, id)
			}
		}
	}
	if len(markdown.AuthoringSources) != 1 || len(native.AuthoringSources) != 0 {
		t.Fatalf("authoring admission is not surface-specific: markdown=%+v native=%+v",
			markdown.AuthoringSources, native.AuthoringSources)
	}
}

// Two different surface forms must never create two accepted definitions with
// one canonical identity in the same Workspace. Source ordering is not a vote.
func TestRuleMarkdownRejectsLegacyValidationWithSameIdentity(t *testing.T) {
	source := filepath.Join("testdata", "tei-md-rule")
	root := t.TempDir()
	for _, file := range []string{
		"tei.workspace.yaml",
		"semantic/sales/order.tei",
		"semantic/sales/order-line-rules.tei.md",
	} {
		b, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	legacy, err := os.ReadFile(filepath.Join("..", "..", "reference", "workspace", "semantic", "sales", "validation.tei"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales", "validation.tei"), legacy, 0600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "already declared") ||
		!strings.Contains(err.Error(), "sales.order_line.quantity_positive") {
		t.Fatalf("native and Markdown duplicate definitions were not rejected: %+v, err=%v", got.AuthoringSources, err)
	}
}

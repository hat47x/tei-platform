package workspace

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// The product Load reads an example table into Result.RuleExamples. The
// expected value is written by hand; positions are found with strings.Index.

const teiMDRuleExampleDocument = "semantic/stock/b-examples.tei.md"

const teiMDRuleExampleText = "# 在庫の例\n\n棚の中の品目である。数量が空の品目は、まだ数えていない。\n\n" +
	"<!-- tei: example of stock.item expecting stock.shelf.items.code_unique satisfied -->\n" +
	"| code | quantity | stock.item.quantity_positive |\n| --- | --- | --- |\n" +
	"| A-1 | 4 | satisfied |\n| B-2 | | undecided |\n| C-3 | 0 | |\n"

func teiMDRuleExampleWorkspace(t *testing.T, examples string) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml": "workspace: rule_example\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n  - tei.authoring/v1\n",
		"semantic/stock/stock.tei": "namespace stock\n\ntype Quantity from std.decimal\n\n" +
			"data Shelf {\n    attribute shelfId type std.identifier\n    relation items to stock.item many\n}\n\n" +
			"data Item {\n    attribute code type std.code\n    attribute quantity type stock.quantity\n}\n",
		// This document sorts after the examples: the rules are read later.
		"semantic/stock/c-rules.tei.md": "# 在庫の規則\n\n```tei\nnamespace stock\n\n" +
			"rule stock.item.quantity_positive:\n  stock.item.quantity is positive\n\n" +
			"rule stock.shelf.items.code_unique:\n  stock.item.code is unique per stock.shelf.items\n```\n",
	}
	if examples != "" {
		files[teiMDRuleExampleDocument] = examples
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// rangeOf is the range of text, which occurs once after the given prefix.
func rangeOf(t *testing.T, document, after, text string) sourceindex.Range {
	t.Helper()
	if strings.Count(document, after) != 1 {
		t.Fatalf("%q must occur exactly once", after)
	}
	base := strings.Index(document, after) + len(after)
	at := strings.Index(document[base:], text)
	if at < 0 {
		t.Fatalf("%q does not follow %q", text, after)
	}
	position := func(offset int) sourceindex.Position {
		lineStart := strings.LastIndex(document[:offset], "\n") + 1
		// Columns count characters, not bytes.
		return sourceindex.Position{Offset: offset, Line: 1 + strings.Count(document[:offset], "\n"), Column: 1 + len([]rune(document[lineStart:offset]))}
	}
	return sourceindex.Range{Start: position(base + at), End: position(base + at + len(text))}
}

func TestTEIMDRuleExampleIsReadBesideTheWorld(t *testing.T) {
	loaded, err := Load(teiMDRuleExampleWorkspace(t, teiMDRuleExampleText))
	if err != nil {
		t.Fatal(err)
	}
	doc := teiMDRuleExampleText
	cell := func(after, text string) RuleExampleCell {
		return RuleExampleCell{Text: text, Range: rangeOf(t, doc, after, text)}
	}
	empty := func(after string) RuleExampleCell {
		r := rangeOf(t, doc, after, "")
		return RuleExampleCell{Range: r}
	}
	want := []RuleExample{{
		File: teiMDRuleExampleDocument, Data: "stock.item",
		Items: []RuleExampleItem{
			{Attribute: "stock.item.code", Symbol: "code"},
			// The type is an alias of a number type.
			{Attribute: "stock.item.quantity", Symbol: "quantity", Numeric: true},
		},
		Rows: [][]RuleExampleCell{
			{cell("--- |\n| ", "A-1"), cell("| A-1 | ", "4")},
			// An empty cell is located right after the bar that opens it.
			{cell("satisfied |\n| ", "B-2"), empty("| B-2 |")},
			{cell("undecided |\n| ", "C-3"), cell("| C-3 | ", "0")},
		},
		Expectations: []RuleExampleExpectation{
			{Rule: "stock.shelf.items.code_unique", Row: -1, Expect: syntax.RuleExampleSatisfied, Range: rangeOf(t, doc, "code_unique ", "satisfied")},
			{Rule: "stock.item.quantity_positive", Row: 0, Expect: syntax.RuleExampleSatisfied, Range: rangeOf(t, doc, "| A-1 | 4 | ", "satisfied")},
			{Rule: "stock.item.quantity_positive", Row: 1, Expect: syntax.RuleExampleUndecided, Range: rangeOf(t, doc, "| B-2 | | ", "undecided")},
		},
	}}
	if !reflect.DeepEqual(loaded.RuleExamples, want) {
		t.Fatalf("examples=%+v\nwant=%+v", loaded.RuleExamples, want)
	}
	if len(loaded.pendingRuleExamples) != 0 {
		t.Fatal("a returned Result still holds unresolved example tables")
	}

	// The same definitions without the example document are the same World,
	// and no example is a declared element with a source of its own.
	plain, err := Load(teiMDRuleExampleWorkspace(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plain.World, loaded.World) {
		t.Fatal("an example changed the World")
	}
	if len(plain.RuleExamples) != 0 {
		t.Fatalf("examples=%+v", plain.RuleExamples)
	}
	for key := range loaded.Sources {
		if _, known := plain.Sources[key]; !known {
			t.Fatalf("an example added the declared element %s", key)
		}
	}
}

// One unreadable example keeps every example of that table out, and stops the
// Load: a partly read table is not kept as a smaller example.
func TestTEIMDRuleExampleIsAllOrNothing(t *testing.T) {
	broken := strings.Replace(teiMDRuleExampleText, "| C-3 | 0 | |", "| C-3 | zero | |", 1)
	loaded, err := Load(teiMDRuleExampleWorkspace(t, broken))
	if err == nil {
		t.Fatalf("an example with an unreadable number loaded: %+v", loaded.RuleExamples)
	}
	if !strings.Contains(err.Error(), "item quantity is a number") || !strings.Contains(err.Error(), filepath.Base(teiMDRuleExampleDocument)) {
		t.Fatalf("err=%v", err)
	}
	var none []core.CanonicalID
	for _, example := range loaded.RuleExamples {
		none = append(none, example.Data)
	}
	if len(none) != 0 {
		t.Fatalf("examples=%v", none)
	}
}

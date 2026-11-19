package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

// This checks the product-selected Markdown reader, not KaTeX rendering.
// Display-oriented TeX in prose is not an accepted Rule or Operator.
func TestTEIMDKaTeXProseDoesNotBecomeRuleMeaning(t *testing.T) {
	baselineRoot := copyTEIMDRuleWorkspace(t)
	baseline, err := Load(baselineRoot)
	if err != nil {
		t.Fatal(err)
	}
	expected := validationsOf(baseline, "sales")
	if len(expected) != 2 {
		t.Fatalf("fixture should declare exactly two rules, got %d", len(expected))
	}

	root := copyTEIMDRuleWorkspace(t)
	original := readTEIMDTestFile(t, root, teiMDRuleDocument)
	preface := strings.Join([]string{
		"# 数式の表示例",
		"",
		"金額は$100。数式の $x^2$ と通貨記号を混同しない。",
		"次の \\(x^2\\) と \\[x^2\\] は説明用の表記です。",
		"",
		"$$",
		"\\sqrt{x^2}=\\lvert x\\rvert",
		"$$",
		"",
		"~~~math",
		"\\gdef\\half{\\frac{1}{2}}",
		"~~~",
		"",
		"インラインコードの \x60$x$\x60 と \x60\\frac{1}{2}\x60 も意味定義ではありません。",
		"",
		"",
	}, "\n")
	document := append([]byte(preface), original...)
	writeTEIMDTestFile(t, root, teiMDRuleDocument, string(document))

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("display-only math caused selected product Reader to fail: %v", err)
	}
	if got := validationsOf(loaded, "sales"); !reflect.DeepEqual(got, expected) {
		t.Fatalf("display math changed accepted Rule meaning:\ngot=%+v\nwant=%+v", got, expected)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Definitions != 2 {
		t.Fatalf("display math was counted as an authoring definition: %+v", loaded.AuthoringSources)
	}

	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
	} {
		if bytes.Count(document, []byte(id)) != 1 {
			t.Fatalf("expected one occurrence of rule %s", id)
		}
		want := bytes.Index(document, []byte(id))
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: id})
		if len(refs) != 1 || refs[0].ArtifactPath != teiMDRuleDocument ||
			refs[0].Role != sourceindex.RoleDefinition ||
			refs[0].Range.Start.Offset != want ||
			refs[0].Range.End.Offset != want+len(id) {
			t.Fatalf("Rule reference lost original UTF-8 byte position for %s: %+v", id, refs)
		}
	}
	readDisk := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	}
	if err := index.Validate(readDisk); err != nil {
		t.Fatalf("Source Index did not validate the original display text: %v", err)
	}

	// A display-only edit changes Source identity but must not silently
	// turn TeX into a Rule or make an old Source Index current.
	changed := append(bytes.Clone(document), []byte("\n説明用の式を一つ追加：$y^2$。\n")...)
	writeTEIMDTestFile(t, root, teiMDRuleDocument, string(changed))
	if err := index.Validate(readDisk); err == nil ||
		!strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("display-only edit left old Source Index eligible: %v", err)
	}
	reloaded, err := Load(root)
	if err != nil {
		t.Fatalf("product Reader rejected a display-only update: %v", err)
	}
	if got := validationsOf(reloaded, "sales"); !reflect.DeepEqual(got, expected) {
		t.Fatalf("display-only update changed accepted Rules: %+v", got)
	}
}

func TestTEIMDMathOnlyDoesNotSatisfySelectedRuleLanguage(t *testing.T) {
	root := copyTEIMDRuleWorkspace(t)
	writeTEIMDTestFile(t, root, teiMDRuleDocument,
		"# 数式だけの文書\n\n$$\nE=mc^2\n$$\n\n~~~math\n\\frac{1}{2}\n~~~\n")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "declares no") {
		t.Fatalf("math-only Markdown was admitted as a selected Rule source: %v", err)
	}
}

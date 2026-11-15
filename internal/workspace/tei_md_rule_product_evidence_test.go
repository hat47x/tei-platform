package workspace_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// TestSelectedRuleMarkdownSourceIsConsumedAndIndexed demonstrates the real
// product Workspace.Load path for the one built-in tei.rule-authoring/v0
// language. It does not qualify external Language Plugins, Graph readers,
// owner-accepted Meaning, or a generated Provider.
func TestSelectedRuleMarkdownSourceIsConsumedAndIndexed(t *testing.T) {
	root := filepath.Join("testdata", "tei-md-rule")
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load selected product Markdown Source: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 {
		t.Fatalf("selected Source reads=%+v, want exactly one", loaded.AuthoringSources)
	}
	read := loaded.AuthoringSources[0]
	if read.Path != sourcePath ||
		read.Language.LanguageID != "tei.rule-authoring" ||
		read.Language.ContractVersion != "v0" ||
		read.Language.ImplementationID != "tei.builtin.rule-authoring-markdown" ||
		read.Regions != 2 || read.Definitions != 2 {
		t.Fatalf("unexpected selected language/read observation: %+v", read)
	}
	parsed := false
	for _, file := range loaded.ManagedFiles {
		if file.Path == sourcePath {
			parsed = file.ReadByLoader
		}
	}
	if !parsed {
		t.Fatal("admitted Markdown Source was only inventoried, not consumed by the product loader")
	}

	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("index the same Load-time Source bytes: %v", err)
	}
	fingerprint := sourceindex.Fingerprint(raw)
	artifactFound := false
	for _, artifact := range index.Artifacts {
		if artifact.Path == sourcePath {
			artifactFound = true
			if artifact.Fingerprint != fingerprint {
				t.Fatalf("indexed artifact fingerprint=%q, want actual Source byte fingerprint %q", artifact.Fingerprint, fingerprint)
			}
		}
	}
	if !artifactFound {
		t.Fatal("selected Markdown Source has no indexed Artifact")
	}
	for _, id := range []string{
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
	} {
		observed := 0
		for _, ref := range index.References {
			if string(ref.Subject.Kind) != "semantic-validation" ||
				string(ref.Subject.ID) != id || ref.ArtifactPath != sourcePath {
				continue
			}
			if err := ref.Range.Validate(raw); err != nil {
				t.Fatalf("reference %s is not a valid range of parsed Source bytes: %v", id, err)
			}
			if string(raw[ref.Range.Start.Offset:ref.Range.End.Offset]) != id {
				t.Fatalf("reference %s does not point at its exact authored identifier", id)
			}
			observed++
		}
		if observed != 1 {
			t.Fatalf("Rule %s indexed occurrences=%d, want exactly one", id, observed)
		}
	}
}

// TestRuleMarkdownSelectionFailsClosedBeforeParsing keeps the positive built-in
// path and the unselected product path independent. A .tei.md suffix alone
// must not silently activate an implementation or ignore a definition.
func TestRuleMarkdownSelectionFailsClosedBeforeParsing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0o700); err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(filepath.Join("testdata", "tei-md-rule", "semantic", "sales", "order-line-rules.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "semantic", "rule.tei.md")
	if err := os.WriteFile(path, input, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: unselected_rule\nformatVersion: 1\nsources:\n  - semantic\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = workspace.Load(root)
	if err == nil || !strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") {
		t.Fatalf("unselected Markdown Source must refuse the product Load: %v", err)
	}
}

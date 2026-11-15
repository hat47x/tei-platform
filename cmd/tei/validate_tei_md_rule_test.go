package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests run the CLI validate operation on a Workspace whose rules are
// written in a TEI Markdown document. They check the last link of the path:
// a diagnostic names the Markdown document and the authored position.

func copyTEIMDRuleFixture(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "internal", "workspace", "testdata", "tei-md-rule")
	root := t.TempDir()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

type teiMDValidateReport struct {
	OK          bool `json:"ok"`
	Diagnostics []struct {
		Diagnostic struct {
			Code    string `json:"code"`
			Subject *struct {
				Kind string `json:"kind"`
				ID   string `json:"id"`
			} `json:"subject"`
		} `json:"diagnostic"`
		Sources []struct {
			ArtifactPath string `json:"artifactPath"`
			Range        struct {
				Start struct {
					Offset int `json:"offset"`
					Line   int `json:"line"`
					Column int `json:"column"`
				} `json:"start"`
			} `json:"range"`
		} `json:"sources"`
	} `json:"diagnostics"`
}

func runTEIMDValidate(t *testing.T, root string) (int, teiMDValidateReport) {
	t.Helper()
	var out bytes.Buffer
	code, err := validateWorkspaceJSON(&out, root)
	if err != nil {
		t.Fatalf("validate: %v output=%s", err, &out)
	}
	var report teiMDValidateReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("%v: %s", err, &out)
	}
	return code, report
}

func TestValidateReadsRulesFromSelectedTEIMarkdown(t *testing.T) {
	const document = "semantic/sales/order-line-rules.tei.md"
	edit := func(t *testing.T, root, old, replacement string) string {
		t.Helper()
		path := filepath.Join(root, filepath.FromSlash(document))
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		// A checkout may have converted the fixture to CRLF.
		content := strings.ReplaceAll(string(raw), "\r\n", "\n")
		if strings.Count(content, old) != 1 {
			t.Fatalf("%q must occur exactly once", old)
		}
		updated := strings.Replace(content, old, replacement, 1)
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
		return updated
	}

	t.Run("valid document", func(t *testing.T) {
		code, report := runTEIMDValidate(t, copyTEIMDRuleFixture(t))
		if code != 0 || !report.OK || len(report.Diagnostics) != 0 {
			t.Fatalf("code=%d report=%+v", code, report)
		}
	})

	cases := []struct {
		name, old, replacement, code string
		// at is text that occurs once in the edited document; the diagnostic
		// must start there. It is located without the parser.
		at          string
		wantSubject bool
	}{
		{"attribute that does not exist",
			"sales.order_line.quantity is positive", "sales.order_line.quantty is positive",
			"UNKNOWN_SEMANTIC_VALIDATION_ATTRIBUTE", "sales.order_line.quantity_positive", true},
		{"positive on a text attribute",
			"sales.order_line.quantity is positive", "sales.order_line.item_name is positive",
			"SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE", "sales.order_line.quantity_positive", true},
		{"unknown predicate",
			"quantity is positive", "quantity is positiv",
			"PARSE_ERROR", "positiv\n", false},
		{"misspelled fence name",
			"```tei\nnamespace sales\n\nrule sales.order_line.quantity_positive", "```TEI\nnamespace sales\n\nrule sales.order_line.quantity_positive",
			"PARSE_ERROR", "```TEI", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyTEIMDRuleFixture(t)
			content := edit(t, root, tc.old, tc.replacement)
			code, report := runTEIMDValidate(t, root)
			if code != 1 || report.OK || len(report.Diagnostics) != 1 {
				t.Fatalf("code=%d report=%+v", code, report)
			}
			got := report.Diagnostics[0]
			if got.Diagnostic.Code != tc.code {
				t.Fatalf("code=%s, want %s", got.Diagnostic.Code, tc.code)
			}
			if tc.wantSubject != (got.Diagnostic.Subject != nil) {
				t.Fatalf("subject=%+v", got.Diagnostic.Subject)
			}
			if len(got.Sources) != 1 || got.Sources[0].ArtifactPath != document {
				t.Fatalf("sources=%+v, want the Markdown document", got.Sources)
			}
			if strings.Count(content, tc.at) != 1 {
				t.Fatalf("%q must occur exactly once in the edited document", tc.at)
			}
			wantOffset := strings.Index(content, tc.at)
			wantLine := 1 + strings.Count(content[:wantOffset], "\n")
			start := got.Sources[0].Range.Start
			if start.Offset != wantOffset || start.Line != wantLine {
				t.Fatalf("diagnostic at offset %d line %d, want offset %d line %d", start.Offset, start.Line, wantOffset, wantLine)
			}
		})
	}

	t.Run("language not selected", func(t *testing.T) {
		root := copyTEIMDRuleFixture(t)
		manifest := filepath.Join(root, "tei.workspace.yaml")
		if err := os.WriteFile(manifest, []byte("workspace: tei_md_rule\nformatVersion: 1\nsources:\n  - semantic\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		code, err := validateWorkspaceJSON(&out, root)
		if code == 0 && err == nil {
			t.Fatalf("an unread Markdown source validated: %s", &out)
		}
		if !strings.Contains(out.String(), "TEI_MD_PARSER_UNBOUND") && (err == nil || !strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND")) {
			t.Fatalf("the refusal does not say why: err=%v output=%s", err, &out)
		}
	})
}

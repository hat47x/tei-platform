package teimdreader

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The explicit Rule table is source, whereas identically shaped tables inside
// fenced or indented examples are presentation. This also protects the new
// Rule parser from regressing when the Graph parser adds Markdown coverage.
func TestRuleCandidateSkipsMarkdownCodeExamples(t *testing.T) {
	base := string(fixtureSource(t, "baseline.tei.md"))
	const heading = "## 算定規則の定義"
	const fake = "## 算定規則の定義\n| Rule ID | 対象 | 条件 | 出力 | 順序 |\n| --- | --- | --- | --- | --- |\n| node.edge_ids | 対象ノード | 終点 = 対象ノード | Edge ID一覧 | Edge ID昇順 |\n"
	cases := []struct {
		name, example string
	}{
		{"tilde fence", "~~~markdown\n" + fake + "~~~\n"},
		{"long backtick fence", "````markdown\n```\n" + fake + "````\n"},
		{"long tilde fence", "~~~~markdown\n~~~\n" + fake + "~~~~\n"},
		{"indented code", "    ## 算定規則の定義\n    | Rule ID | 対象 | 条件 | 出力 | 順序 |\n    | --- | --- | --- | --- | --- |\n    | node.edge_ids | 対象ノード | 終点 = 対象ノード | Edge ID一覧 | Edge ID昇順 |\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(strings.Replace(base, heading, tc.example+heading, 1))
			got, err := ParseRuleCandidate(src)
			if err != nil {
				t.Fatal(err)
			}
			if got.Rule.ID != "node.edge_ids" || got.Rule.Endpoint != "from" {
				t.Fatalf("example was promoted to selected Rule: %+v", got.Rule)
			}
			sp := got.Rule.ConditionSource
			if string(src[sp.Start:sp.End]) != "起点 = 対象ノード" {
				t.Fatalf("original rule condition byte span drifted: %+v", sp)
			}
			goSource, err := GenerateGo(got)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(goSource, []byte("e.from == nodeID")) || bytes.Contains(goSource, []byte("e.to == nodeID")) {
				t.Fatal("generated rule condition came from a Markdown example")
			}
		})
	}
}

func TestRuleCandidateRejectsUnclosedCodeExamples(t *testing.T) {
	base := string(fixtureSource(t, "baseline.tei.md"))
	cases := []string{"\n~~~markdown\n## 算定規則の定義\n", "\n````markdown\n```\n"}
	for _, tail := range cases {
		_, err := ParseRuleCandidate([]byte(base + tail))
		var d *Diagnostic
		if !errors.As(err, &d) || d.Code != "UNTERMINATED_FENCE" {
			t.Fatalf("open example fence was not rejected: %v", err)
		}
	}
}

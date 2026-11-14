package teimdreader

import (
	"errors"
	"strings"
	"testing"
)

// The reference reader must not promote examples to candidate Graph meaning,
// even when Markdown uses a tilde fence or a longer backtick fence.
func TestCandidateIgnoresFencedAndIndentedExampleSections(t *testing.T) {
	base := string(fixtureSource(t, "baseline.tei.md"))
	const heading = "## ノードの定義"
	cases := []struct {
		name, before string
	}{
		{"tilde fence", "~~~markdown\n## ノードの定義\n| ID | 表示名 |\n| --- | --- |\n| fake | 説明 |\n~~~\n"},
		{"four backticks containing three", "````markdown\n```\n## ノードの定義\n| ID | 表示名 |\n| --- | --- |\n| fake | 説明 |\n````\n"},
		{"four tildes containing three", "~~~~markdown\n~~~\n## ノードの定義\n| ID | 表示名 |\n| --- | --- |\n| fake | 説明 |\n~~~~\n"},
		{"indented code", "    ## ノードの定義\n    | ID | 表示名 |\n    | --- | --- |\n    | fake | 説明 |\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(strings.Replace(base, heading, tc.before+heading, 1))
			g, err := ParseCandidate(src)
			if err != nil {
				t.Fatal(err)
			}
			if len(g.Nodes) != 2 || len(g.Edges) != 2 {
				t.Fatalf("fenced/indented example changed candidate: nodes=%d edges=%d", len(g.Nodes), len(g.Edges))
			}
			for _, n := range g.Nodes {
				if string(src[n.IDSource.Start:n.IDSource.End]) != n.ID {
					t.Fatalf("node source span drift for %s", n.ID)
				}
			}
		})
	}
}

func TestCandidateRejectsUnclosedMarkdownExampleFence(t *testing.T) {
	base := string(fixtureSource(t, "baseline.tei.md"))
	cases := []struct {
		name, trailing string
	}{
		{"unclosed tilde", "\n~~~markdown\n## ノードの定義\n"},
		{"wrong closing marker", "\n~~~markdown\n```\n"},
		{"shorter closing run", "\n````markdown\n```\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseCandidate([]byte(base + tc.trailing))
			var d *Diagnostic
			if !errors.As(err, &d) || d.Code != "UNTERMINATED_FENCE" {
				t.Fatalf("unclosed Markdown example was not rejected: %v", err)
			}
		})
	}
}

func TestCandidateIgnoresBOMPrefixedFence(t *testing.T) {
	const example = "~~~markdown\n## ノードの定義\n| ID | 表示名 |\n| --- | --- |\n| fake | 説明 |\n~~~\n"
	src := []byte("\ufeff" + example + string(fixtureSource(t, "baseline.tei.md")))
	g, err := ParseCandidate(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Nodes) != 2 || len(g.Edges) != 2 {
		t.Fatalf("BOM-prefixed example changed candidate: %+v", g)
	}
}

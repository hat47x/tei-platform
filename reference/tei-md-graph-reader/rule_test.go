package teimdreader

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type generatedExpected struct {
	Cases []struct {
		File              string              `json:"file"`
		Valid             bool                `json:"valid"`
		GeneratedResult   map[string][]string `json:"generatedResult"`
		DiagramConsistent bool                `json:"diagramConsistent"`
	} `json:"cases"`
}

func ruleExpected(t *testing.T) generatedExpected {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tei-md-graph", "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v generatedExpected
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}
func runGenerated(t *testing.T, src []byte) map[string][]string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "generated.go")
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatal(err)
	}
	// Compile and run the generated Go in a fresh child Process; neither the
	// reference reader's Outgoing() nor an Expected-derived stub is invoked.
	cmd := exec.Command("go", "run", path)
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GO111MODULE=off")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated Go did not run: %v\n%s", err, out)
	}
	var got map[string][]string
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("invalid output %q: %v", out, err)
	}
	return got
}
func TestGraphRuleGeneratesCompiledGoAgainstIndependentExpected(t *testing.T) {
	result := ruleExpected(t)
	sources := map[string][]byte{}
	for _, tc := range result.Cases {
		if !tc.Valid {
			continue
		}
		t.Run(tc.File, func(t *testing.T) {
			source := fixtureSource(t, tc.File)
			rule, err := ParseRuleCandidate(source)
			if err != nil {
				t.Fatal(err)
			}
			if rule.Rule.ID != "node.edge_ids" {
				t.Fatal("wrong rule ID")
			}
			for _, v := range []struct {
				span Span
				want string
			}{{rule.Rule.IDSource, "node.edge_ids"}} {
				if s := string(source[v.span.Start:v.span.End]); s != v.want {
					t.Fatalf("rule span %q not %q", s, v.want)
				}
			}
			expectedCondition := "起点 = 対象ノード"
			if tc.File == "rule-by-target.tei.md" {
				expectedCondition = "終点 = 対象ノード"
			}
			if s := string(source[rule.Rule.ConditionSource.Start:rule.Rule.ConditionSource.End]); s != expectedCondition {
				t.Fatalf("condition source %q != %q", s, expectedCondition)
			}
			generated, err := GenerateGo(rule)
			if err != nil {
				t.Fatal(err)
			}
			sources[tc.File] = generated
			if got := runGenerated(t, generated); !reflect.DeepEqual(got, tc.GeneratedResult) {
				t.Fatalf("actual generated process output %#v, independent expected %#v", got, tc.GeneratedResult)
			}
		})
	}
	if !bytes.Equal(sources["layout-only.tei.md"], sources["baseline.tei.md"]) {
		t.Error("layout changed generated semantic source")
	}
	if bytes.Equal(sources["edge-reversed.tei.md"], sources["baseline.tei.md"]) {
		t.Error("changing an edge did not alter generated source")
	}
	if bytes.Equal(sources["rule-by-target.tei.md"], sources["baseline.tei.md"]) {
		t.Error("changing authored rule condition did not alter generated source")
	}
	if !bytes.Contains(sources["baseline.tei.md"], []byte("e.from == nodeID")) || !bytes.Contains(sources["rule-by-target.tei.md"], []byte("e.to == nodeID")) {
		t.Error("generated Go predicate does not follow authored rule")
	}
}
func TestInvalidGraphCannotGenerateGo(t *testing.T) {
	_, err := ParseRuleCandidate(fixtureSource(t, "duplicate-edge-id.tei.md"))
	errCode(t, err, "DUPLICATE_EDGE_ID")
}
func TestRuleIsNotInferredFromProseOrUnsupportedCondition(t *testing.T) {
	base := string(fixtureSource(t, "baseline.tei.md"))
	cases := []struct{ label, source, code string }{
		{"unselected prose", strings.Replace(base, "## 算定規則の定義", "## 未選択の説明", 1), "RULE_NOT_BOUND"},
		{"unknown condition", strings.Replace(base, "起点 = 対象ノード", "適当に抽出", 1), "UNSUPPORTED_RULE_CONDITION"},
		{"unknown output", strings.Replace(base, "Edge ID一覧 | Edge ID昇順", "Node ID一覧 | Edge ID昇順", 1), "UNSUPPORTED_RULE"},
		{"ambiguous rules", strings.Replace(base, "| node.edge_ids | 対象ノード | 起点 = 対象ノード | Edge ID一覧 | Edge ID昇順 |", "| node.edge_ids | 対象ノード | 起点 = 対象ノード | Edge ID一覧 | Edge ID昇順 |\n| node.edge_ids | 対象ノード | 終点 = 対象ノード | Edge ID一覧 | Edge ID昇順 |", 1), "MULTIPLE_RULE_ROWS"},
		{"duplicate rule heading", base + "\n## 算定規則の定義\n", "DUPLICATE_RULE_SECTION"},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) { _, err := ParseRuleCandidate([]byte(tc.source)); errCode(t, err, tc.code) })
	}
}
func TestRuleBOMCRLFByteSpan(t *testing.T) {
	src := append([]byte{0xef, 0xbb, 0xbf}, bytes.ReplaceAll(fixtureSource(t, "baseline.tei.md"), []byte("\n"), []byte("\r\n"))...)
	c, err := ParseRuleCandidate(src)
	if err != nil {
		t.Fatal(err)
	}
	if actual := string(src[c.Rule.ConditionSource.Start:c.Rule.ConditionSource.End]); actual != "起点 = 対象ノード" {
		t.Fatal("lost original rule condition byte span", actual)
	}
}
func TestStaleMermaidRemainsReadOnly(t *testing.T) {
	src := fixtureSource(t, "diagram-stale.tei.md")
	cand, err := ParseRuleCandidate(src)
	if err != nil {
		t.Fatal(err)
	}
	matches, err := CheckDerivedView(src, cand.Graph, map[string]string{"N1": "n-1", "N2": "n-2"})
	if err != nil || matches {
		t.Fatalf("view must report stale, not overwrite table: %t, %v", matches, err)
	}
	got, err := GenerateGo(cand)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(runGenerated(t, got), map[string][]string{"n-1": {"e-1"}, "n-2": {"e-2"}}) {
		t.Error("stale display overrode table definition")
	}
}

package teimdreader

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type expectedCase struct {
	File                string              `json:"file"`
	Valid               bool                `json:"valid"`
	Outgoing            map[string][]string `json:"outgoing"`
	DiagramConsistent   bool                `json:"diagramConsistent"`
	DiagnosticCandidate string              `json:"diagnosticCandidate"`
	MeaningEqualTo      string              `json:"meaningEqualTo"`
}

func fixtures(t *testing.T) []expectedCase {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tei-md-graph", "independent-expected.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Cases []expectedCase `json:"cases"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if len(v.Cases) != 5 {
		t.Fatalf("fixture count = %d", len(v.Cases))
	}
	return v.Cases
}
func fixtureSource(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tei-md-graph", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func errCode(t *testing.T, err error, want string) {
	t.Helper()
	var d *Diagnostic
	if !errors.As(err, &d) || d.Code != want {
		t.Fatalf("diagnostic = %v; want %s", err, want)
	}
}
func TestCandidateFollowsIndependentExpected(t *testing.T) {
	aliases := map[string]string{"N1": "n-1", "N2": "n-2"}
	for _, tc := range fixtures(t) {
		t.Run(tc.File, func(t *testing.T) {
			src := fixtureSource(t, tc.File)
			got, err := ParseCandidate(src)
			if !tc.Valid {
				errCode(t, err, "DUPLICATE_EDGE_ID")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Outgoing(), tc.Outgoing) {
				t.Fatalf("outgoing = %#v, want %#v", got.Outgoing(), tc.Outgoing)
			}
			consistent, err := CheckDerivedView(src, got, aliases)
			if err != nil {
				t.Fatal(err)
			}
			if consistent != tc.DiagramConsistent {
				t.Fatalf("view consistent = %t, want %t", consistent, tc.DiagramConsistent)
			}
			for _, n := range got.Nodes {
				if s := string(src[n.IDSource.Start:n.IDSource.End]); s != n.ID {
					t.Fatalf("node span = %q, want %q", s, n.ID)
				}
			}
			for _, e := range got.Edges {
				for _, v := range []struct {
					span  Span
					value string
				}{{e.IDSource, e.ID}, {e.FromSource, e.From}, {e.ToSource, e.To}} {
					if s := string(src[v.span.Start:v.span.End]); s != v.value {
						t.Fatalf("edge span = %q, want %q", s, v.value)
					}
				}
			}
			if tc.MeaningEqualTo != "" {
				other, err := ParseCandidate(fixtureSource(t, tc.MeaningEqualTo))
				if err != nil {
					t.Fatal(err)
				}
				for i := range got.Nodes {
					got.Nodes[i].IDSource = Span{}
				}
				for i := range other.Nodes {
					other.Nodes[i].IDSource = Span{}
				}
				for i := range got.Edges {
					got.Edges[i].IDSource = Span{}
					got.Edges[i].FromSource = Span{}
					got.Edges[i].ToSource = Span{}
				}
				for i := range other.Edges {
					other.Edges[i].IDSource = Span{}
					other.Edges[i].FromSource = Span{}
					other.Edges[i].ToSource = Span{}
				}
				if !reflect.DeepEqual(got, other) {
					t.Fatal("layout-only source changed semantic graph")
				}
			}
		})
	}
}
func TestCandidatePreservesBOMCRLFAndSourceOffsets(t *testing.T) {
	src := append([]byte{0xef, 0xbb, 0xbf}, bytes.ReplaceAll(fixtureSource(t, "baseline.tei.md"), []byte("\n"), []byte("\r\n"))...)
	got, err := ParseCandidate(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 2 || len(got.Edges) != 2 {
		t.Fatal("lost graph entries")
	}
	for _, e := range got.Edges {
		if string(src[e.IDSource.Start:e.IDSource.End]) != e.ID {
			t.Fatalf("offset drift for %s", e.ID)
		}
	}
}
func TestCandidateDoesNotRunExamplesOrQuotedHeadings(t *testing.T) {
	src := fixtureSource(t, "baseline.tei.md")
	src = append(src, []byte("\n> ## 関係の定義\n> | e-1 | n-1 | n-2 |\n\n```markdown\n## 関係の定義\n| Edge ID | 起点 | 終点 |\n```\n")...)
	got, err := ParseCandidate(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Edges) != 2 {
		t.Fatalf("read %d edges from quoted/fenced material", len(got.Edges))
	}
}
func TestCandidateRejectsIncompleteAndAmbiguousTables(t *testing.T) {
	base := string(fixtureSource(t, "baseline.tei.md"))
	cases := []struct{ name, src, code string }{
		{"duplicate section", base + "\n## 関係の定義\n", "DUPLICATE_SECTION"},
		{"unknown endpoint", strings.Replace(base, "| e-2 | n-1 | n-2 |", "| e-2 | n-1 | n-3 |", 1), "UNKNOWN_EDGE_ENDPOINT"},
		{"table escaped pipe", strings.Replace(base, "| n-1 | 対象 |", "| n-1 | 対\\|象 |", 1), "UNSUPPORTED_TABLE"},
		{"wrong heading", strings.Replace(base, "## ノードの定義", "## 規則の定義", 1), "MISSING_REQUIRED_TABLE"},
		{"unknown header", strings.Replace(base, "| Edge ID | 起点 | 終点 |", "| 名前 | 起点 | 終点 |", 1), "UNSUPPORTED_TABLE_HEADER"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { _, err := ParseCandidate([]byte(tc.src)); errCode(t, err, tc.code) })
	}
}
func TestDisplayAliasMustBeExplicit(t *testing.T) {
	src := fixtureSource(t, "baseline.tei.md")
	g, err := ParseCandidate(src)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CheckDerivedView(src, g, map[string]string{"N1": "n-1"})
	errCode(t, err, "UNBOUND_DISPLAY_ALIAS")
}

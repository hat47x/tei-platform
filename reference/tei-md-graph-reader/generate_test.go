package teimdreader

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

type executedOutgoing struct {
	Outgoing     map[string][]string `json:"outgoing"`
	Counts       map[string]int      `json:"counts"`
	UnknownIsNil bool                `json:"unknownIsNil"`
}

// The independent consumer is built from generated source, not from
// Candidate.Outgoing or a handwritten graph evaluator.
func runGeneratedOutgoing(t *testing.T, src []byte) executedOutgoing {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "outgoinggraph"), 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"go.mod": []byte("module tei-graph-fixture\n\ngo 1.23\n"),
		"outgoinggraph/generated.go": src,
		"main.go": []byte("package main\n\nimport (\n\t\"encoding/json\"\n\t\"os\"\n\tgraph \"tei-graph-fixture/outgoinggraph\"\n)\n\nfunc main() {\n\tresult := struct {\n\t\tOutgoing map[string][]string `json:"outgoing"`\n\t\tCounts map[string]int `json:"counts"`\n\t\tUnknownIsNil bool `json:"unknownIsNil"`\n\t}{\n\t\tOutgoing: map[string][]string{\"n-1\": graph.Outgoing(\"n-1\"), \"n-2\": graph.Outgoing(\"n-2\")},\n\t\tCounts: map[string]int{\"n-1\": graph.OutgoingCount(\"n-1\"), \"n-2\": graph.OutgoingCount(\"n-2\")},\n\t\tUnknownIsNil: graph.Outgoing(\"absent\") == nil,\n\t}\n\tif err := json.NewEncoder(os.Stdout).Encode(result); err != nil { panic(err) }\n}\n"),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(name)), contents, 0644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOTOOLCHAIN=local")
	b, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("actual generated Go consumer failed: %v\n%s", err, b)
	}
	var out executedOutgoing
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("actual consumer reply is not JSON: %v\n%s", err, b)
	}
	return out
}

func TestGeneratedGoActualConsumerFollowsIndependentExpected(t *testing.T) {
	for _, tc := range fixtures(t) {
		t.Run(tc.File, func(t *testing.T) {
			raw := fixtureSource(t, tc.File)
			c, err := ParseCandidate(raw)
			if !tc.Valid {
				if err == nil {
					t.Fatal("invalid definition unexpectedly parsed")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			result, err := GenerateOutgoingGo(c, "outgoinggraph")
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(result.Code, []byte("switch nodeID")) {
				t.Fatal("generated code does not contain the outgoing decision")
			}
			if len(result.Origins) != len(c.Edges) {
				t.Fatalf("origin count = %d, edges = %d", len(result.Origins), len(c.Edges))
			}
			for _, o := range result.Origins {
				if string(raw[o.IDSource.Start:o.IDSource.End]) != o.EdgeID {
					t.Fatalf("origin does not map to raw edge ID %q", o.EdgeID)
				}
			}
			got := runGeneratedOutgoing(t, result.Code)
			if !reflect.DeepEqual(got.Outgoing, tc.Outgoing) {
				t.Fatalf("real Go result = %#v, independent expected = %#v", got.Outgoing, tc.Outgoing)
			}
			if !got.UnknownIsNil {
				t.Fatal("unknown node must be distinguishable from known empty outdegree")
			}
			for node, ids := range tc.Outgoing {
				if got.Counts[node] != len(ids) {
					t.Fatalf("generated count of %s = %d, want %d", node, got.Counts[node], len(ids))
				}
			}
		})
	}
}

func TestGeneratedGoStableAcrossLayoutAndViewMismatch(t *testing.T) {
	codeFor := func(name string) []byte {
		t.Helper()
		c, err := ParseCandidate(fixtureSource(t, name))
		if err != nil {
			t.Fatal(err)
		}
		g, err := GenerateOutgoingGo(c, "outgoinggraph")
		if err != nil {
			t.Fatal(err)
		}
		return g.Code
	}
	baseline := codeFor("baseline.tei.md")
	layout := codeFor("layout-only.tei.md")
	reversed := codeFor("edge-reversed.tei.md")
	stale := codeFor("diagram-stale.tei.md")
	if !bytes.Equal(baseline, layout) {
		t.Fatal("layout-only change altered generated semantic Go source")
	}
	if bytes.Equal(baseline, reversed) {
		t.Fatal("reversing an edge did not alter generated Go behavior")
	}
	if !bytes.Equal(reversed, stale) {
		t.Fatal("stale Mermaid view changed the source-table-derived Go code")
	}
}

func TestGeneratedGoRejectsMalformedTypedCandidates(t *testing.T) {
	valid, err := ParseCandidate(fixtureSource(t, "baseline.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range []string{"", "package", "with-dash", "a;import"} {
		if _, err := GenerateOutgoingGo(valid, pkg); err == nil {
			t.Fatalf("invalid package %q was accepted", pkg)
		}
	}
	cases := []struct {
		name string
		edit func(*Candidate)
	}{
		{"duplicate node", func(c *Candidate) { c.Nodes = append(c.Nodes, c.Nodes[0]) }},
		{"duplicate edge", func(c *Candidate) { c.Edges = append(c.Edges, c.Edges[0]) }},
		{"unknown source", func(c *Candidate) { c.Edges[0].From = "absent" }},
		{"unknown target", func(c *Candidate) { c.Edges[0].To = "absent" }},
		{"empty node", func(c *Candidate) { c.Nodes[0].ID = "" }},
		{"empty edge", func(c *Candidate) { c.Edges[0].ID = "" }},
		{"no edges", func(c *Candidate) { c.Edges = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Candidate{Nodes: append([]Node(nil), valid.Nodes...), Edges: append([]Edge(nil), valid.Edges...)}
			tc.edit(&c)
			g, err := GenerateOutgoingGo(c, "outgoinggraph")
			if err == nil || len(g.Code) != 0 {
				t.Fatalf("malformed typed candidate generated code: %v, %q", err, g.Code)
			}
		})
	}}

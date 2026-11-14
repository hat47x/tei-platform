package teimdreader

import (
	"fmt"
	"testing"

	"tei/internal/sourceindex"
)

// This verifies the shared Source Index byte/provenance contract, not that a
// Workspace has selected or accepted the Markdown Language/semantic Role.
func TestRuleCandidateByteSpansUseExistingSourceIndex(t *testing.T) {
	for _, name := range []string{
		"baseline.tei.md",
		"edge-reversed.tei.md",
		"layout-only.tei.md",
		"diagram-stale.tei.md",
		"rule-by-target.tei.md",
	} {
		t.Run(name, func(t *testing.T) {
			src := fixtureSource(t, name)
			candidate, err := ParseRuleCandidate(src)
			if err != nil {
				t.Fatal(err)
			}
			const path = "semantic/graph.tei.md"
			artifact, err := sourceindex.NewArtifact(path, src)
			if err != nil {
				t.Fatal(err)
			}
			if artifact.Format != "md" {
				t.Fatalf("the physical suffix was misclassified: %q", artifact.Format)
			}
			if artifact.Fingerprint != sourceindex.Fingerprint(src) {
				t.Fatal("Source Index fingerprint must cover the exact original bytes")
			}
			index := sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact}}
			readOriginal := func(requested string) ([]byte, error) {
				if requested != path {
					return nil, fmt.Errorf("unexpected requested path %q", requested)
				}
				return src, nil
			}
			if err := index.Validate(readOriginal); err != nil {
				t.Fatalf("existing Source Index rejected exact input: %v", err)
			}
			check := func(label string, span Span, want string) {
				t.Helper()
				rng, err := sourceindex.RangeFromOffsets(src, span.Start, span.End)
				if err != nil {
					t.Fatalf("%s offset: %v", label, err)
				}
				if err := rng.Validate(src); err != nil {
					t.Fatalf("%s canonical range: %v", label, err)
				}
				if rng.Start.Line != span.Line {
					t.Fatalf("%s line mismatch: source=%d canonical=%d", label, span.Line, rng.Start.Line)
				}
				if got := string(src[rng.Start.Offset:rng.End.Offset]); got != want {
					t.Fatalf("%s raw byte span = %q, want %q", label, got, want)
				}
			}
			for _, n := range candidate.Graph.Nodes {
				check("node", n.IDSource, n.ID)
			}
			for _, e := range candidate.Graph.Edges {
				check("edge ID", e.IDSource, e.ID)
				check("source", e.FromSource, e.From)
				check("target", e.ToSource, e.To)
			}
			check("rule ID", candidate.Rule.IDSource, candidate.Rule.ID)
			condition := "起点 = 対象ノード"
			if candidate.Rule.Endpoint == "to" {
				condition = "終点 = 対象ノード"
			}
			check("rule condition", candidate.Rule.ConditionSource, condition)
			// An appended explanation changes original byte identity even when
			// graph semantics and previously selected source positions are the same.
			changed := append(append([]byte(nil), src...), []byte("\nUpdated explanation\n")...)
			if err := index.Validate(func(p string) ([]byte, error) {
				if p != path {
					return nil, fmt.Errorf("unrecognized source %q", p)
				}
				return changed, nil
			}); err == nil {
				t.Fatal("stale Source Index artifact was accepted after byte change")
			}
		})
	}
}

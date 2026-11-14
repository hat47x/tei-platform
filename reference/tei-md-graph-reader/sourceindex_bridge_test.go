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

// TestSourceIndexRevisionDoesNotInheritCandidateShapeFingerprint checks that a
// normalized reference Graph/Rule shape never substitutes for raw Source
// identity. It proves neither Workspace Frontend selection nor Meaning approval.
func TestSourceIndexRevisionDoesNotInheritCandidateShapeFingerprint(t *testing.T) {
	const path = "semantic/graph.tei.md"
	original := fixtureSource(t, "baseline.tei.md")
	base, err := ParseRuleCandidate(original)
	if err != nil {
		t.Fatal(err)
	}
	baseShape, err := CandidateShapeFingerprint(base)
	if err != nil {
		t.Fatal(err)
	}
	baseArtifact, err := sourceindex.NewArtifact(path, original)
	if err != nil {
		t.Fatal(err)
	}
	oldIndex := sourceindex.Index{Artifacts: []sourceindex.Artifact{baseArtifact}}

	cases := []struct {
		name      string
		bytes     []byte
		sameShape bool
	}{
		{
			name:      "display layout changes, not Graph Rule",
			bytes:     fixtureSource(t, "layout-only.tei.md"),
			sameShape: true,
		},
		{
			name:      "explanation changes, not Graph Rule",
			bytes:     append(append([]byte(nil), original...), []byte("\nAdditional explanation only\n")...),
			sameShape: true,
		},
		{
			name:      "Edge direction changes",
			bytes:     fixtureSource(t, "edge-reversed.tei.md"),
			sameShape: false,
		},
		{
			name:      "Rule predicate changes",
			bytes:     fixtureSource(t, "rule-by-target.tei.md"),
			sameShape: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseRuleCandidate(tc.bytes)
			if err != nil {
				t.Fatal(err)
			}
			candidateShape, err := CandidateShapeFingerprint(parsed)
			if err != nil {
				t.Fatal(err)
			}
			if got := candidateShape == baseShape; got != tc.sameShape {
				t.Fatalf("same reference structure = %t, want %t", got, tc.sameShape)
			}
			artifact, err := sourceindex.NewArtifact(path, tc.bytes)
			if err != nil {
				t.Fatal(err)
			}
			if artifact.Fingerprint == baseArtifact.Fingerprint {
				t.Fatal("a changed Source was reported as the same byte revision")
			}
			readChanged := func(requested string) ([]byte, error) {
				if requested != path {
					return nil, fmt.Errorf("unexpected Source %q", requested)
				}
				return tc.bytes, nil
			}
			// Even for the same candidate shape, a stale raw Source Index is
			// not permitted to validate the changed authored Markdown.
			if err := oldIndex.Validate(readChanged); err == nil {
				t.Fatal("stale raw Source Index passed after byte change")
			}
			currentIndex := sourceindex.Index{Artifacts: []sourceindex.Artifact{artifact}}
			if err := currentIndex.Validate(readChanged); err != nil {
				t.Fatalf("current raw Source Index rejected its own bytes: %v", err)
			}
		})
	}
}

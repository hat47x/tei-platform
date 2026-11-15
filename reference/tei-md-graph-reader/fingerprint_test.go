package teimdreader

import (
	"reflect"
	"testing"
)

func TestCandidateShapeFingerprintSeparatesMeaningFromPresentation(t *testing.T) {
	fingerprints := map[string]string{}
	for _, name := range []string{
		"baseline.tei.md",
		"edge-reversed.tei.md",
		"layout-only.tei.md",
		"diagram-stale.tei.md",
		"rule-by-target.tei.md",
	} {
		c, err := ParseRuleCandidate(fixtureSource(t, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		fingerprint, err := CandidateShapeFingerprint(c)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(fingerprint) != 64 {
			t.Fatalf("%s: invalid SHA-256 length %d", name, len(fingerprint))
		}
		fingerprints[name] = fingerprint
	}
	if fingerprints["baseline.tei.md"] != fingerprints["layout-only.tei.md"] {
		t.Fatal("Mermaid layout changed a generated-logic shape fingerprint")
	}
	if fingerprints["edge-reversed.tei.md"] != fingerprints["diagram-stale.tei.md"] {
		t.Fatal("stale view changed the table-defined graph structure fingerprint")
	}
	if fingerprints["baseline.tei.md"] == fingerprints["edge-reversed.tei.md"] {
		t.Fatal("edge direction was omitted from fingerprint")
	}
	if fingerprints["baseline.tei.md"] == fingerprints["rule-by-target.tei.md"] {
		t.Fatal("the selected rule predicate was omitted from fingerprint")
	}
}

func TestCandidateShapeFingerprintIgnoresOrderAndSourceSpan(t *testing.T) {
	a, err := ParseRuleCandidate(fixtureSource(t, "baseline.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	b := a
	b.Graph.Nodes = append([]Node(nil), a.Graph.Nodes...)
	b.Graph.Edges = append([]Edge(nil), a.Graph.Edges...)
	b.Graph.Nodes[0], b.Graph.Nodes[1] = b.Graph.Nodes[1], b.Graph.Nodes[0]
	b.Graph.Edges[0], b.Graph.Edges[1] = b.Graph.Edges[1], b.Graph.Edges[0]
	b.Rule.IDSource = Span{}
	b.Rule.ConditionSource = Span{}
	for i := range b.Graph.Nodes {
		b.Graph.Nodes[i].IDSource = Span{}
		b.Graph.Nodes[i].Label = "changed view label"
	}
	for i := range b.Graph.Edges {
		b.Graph.Edges[i].IDSource = Span{}
		b.Graph.Edges[i].FromSource = Span{}
		b.Graph.Edges[i].ToSource = Span{}
	}
	before, err := CandidateShapeFingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	after, err := CandidateShapeFingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("display, ordering or Source span changed executable candidate structure")
	}
}

func TestCandidateShapeFingerprintRejectsBrokenIdentity(t *testing.T) {
	base, err := ParseRuleCandidate(fixtureSource(t, "baseline.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	mutations := []struct {
		name string
		edit func(*RuleCandidate)
	}{
		{"duplicate edge", func(c *RuleCandidate) { c.Graph.Edges[1].ID = c.Graph.Edges[0].ID }},
		{"unknown endpoint", func(c *RuleCandidate) { c.Graph.Edges[1].To = "unknown" }},
		{"missing node", func(c *RuleCandidate) { c.Graph.Nodes = nil }},
		{"missing rule", func(c *RuleCandidate) { c.Rule.ID = "" }},
		{"unknown predicate", func(c *RuleCandidate) { c.Rule.Endpoint = "neither" }},
	}
	for _, tc := range mutations {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base
			candidate.Graph.Nodes = append([]Node(nil), base.Graph.Nodes...)
			candidate.Graph.Edges = append([]Edge(nil), base.Graph.Edges...)
			tc.edit(&candidate)
			if _, err := CandidateShapeFingerprint(candidate); err == nil {
				t.Fatal("invalid reference candidate gained a structure fingerprint")
			}
		})
	}
}

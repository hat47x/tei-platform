package operations_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/externalanalysis"
	"tei/internal/operations"
)

// The captured REA document is deliberately mutated only inside its producer
// semantic graph. This exercises the real saved-Evidence receiver, not a
// different mock candidate parser.
func ambiguousCandidateFixture(t *testing.T, mutate func(map[string]any)) string {
	t.Helper()
	text := strings.TrimPrefix(candidateText(t), "\ufeff")
	var document map[string]any
	if err := json.Unmarshal([]byte(text), &document); err != nil {
		t.Fatalf("decode captured REA envelope: %v", err)
	}
	data := document["data"].(map[string]any)
	normalized := data["normalized_result"].(map[string]any)
	graph := normalized["semantic_graph"].(map[string]any)
	mutate(graph)
	raw, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode mutated evidence: %v", err)
	}
	path := filepath.Join(t.TempDir(), "ambiguous-evidence.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func duplicateFirstRelation(graph map[string]any, removeProducerID bool) {
	relations := graph["relations"].([]any)
	first := relations[0].(map[string]any)
	duplicate := make(map[string]any, len(first))
	for k, v := range first {
		duplicate[k] = v
	}
	if removeProducerID {
		delete(first, "relation_id")
		delete(duplicate, "relation_id")
	}
	graph["relations"] = append(relations, duplicate)
}

// The real REA payload contains more than one distinct node. Duplicating a
// single node ID is an invalid producer identity even when the label matches.
func duplicateSelectedObservation(t *testing.T, graph map[string]any) {
	t.Helper()
	nodes := graph["nodes"].([]any)
	for _, raw := range nodes {
		node := raw.(map[string]any)
		if node["node_id"] != candidateNodeID {
			continue
		}
		duplicate := make(map[string]any, len(node))
		for k, v := range node {
			duplicate[k] = v
		}
		graph["nodes"] = append(nodes, duplicate)
		return
	}
	t.Fatalf("captured semantic graph contains no selected node %s", candidateNodeID)
}

func TestExternalCandidatesRejectsNonUniqueProducerAndDerivedIdentities(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*testing.T, map[string]any)
		selectID func(externalanalysis.Candidates) string
	}{
		{
			name: "producer repeats relation ID",
			mutate: func(_ *testing.T, graph map[string]any) {
				duplicateFirstRelation(graph, false)
			},
			selectID: func(_ externalanalysis.Candidates) string { return candidateRelationID },
		},
		{
			name: "missing relation IDs collide on directed parallel edge",
			mutate: func(_ *testing.T, graph map[string]any) {
				duplicateFirstRelation(graph, true)
			},
			selectID: func(c externalanalysis.Candidates) string { return c.Relations[0].RelationID },
		},
		{
			name:     "producer repeats observation node ID",
			mutate:   duplicateSelectedObservation,
			selectID: func(_ externalanalysis.Candidates) string { return candidateRelationID },
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := ambiguousCandidateFixture(t, func(graph map[string]any) { tc.mutate(t, graph) })
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			receipt := externalanalysis.Receive(raw, externalanalysis.Tool{Name: "rea", Version: "6.8.0", Present: true}, candidateReadAt)
			if receipt.Evidence == nil {
				t.Fatalf("mutated evidence unexpectedly rejected before selection: %+v", receipt)
			}
			candidates, err := externalanalysis.ReadCandidates(receipt.EvidenceBody())
			if err != nil {
				t.Fatalf("read candidate graph: %v", err)
			}
			relationID := tc.selectID(candidates)
			view, err := operations.ExternalCandidates(operations.ExternalCandidateRequest{
				EvidencePath: path, ToolVersion: "6.8.0",
				ObservationID: candidateNodeID, RelationID: relationID,
				Tool: externalanalysis.Tool{Name: "rea", Present: true},
			}, candidateReadAt)
			var candidateError *operations.ExternalCandidateError
			if !errors.As(err, &candidateError) || candidateError.Code != operations.ExternalCandidateAmbiguous {
				t.Fatalf("duplicate selected IDs must be ambiguous, got view=%+v err=%v", view, err)
			}
			if view.Observation != nil || view.Relation != nil {
				t.Fatalf("ambiguous selection leaked an arbitrary occurrence: observation=%+v relation=%+v", view.Observation, view.Relation)
			}
			if view.Unselected.Observations != len(candidates.Observations) ||
				view.Unselected.Relations != len(candidates.Relations) {
				t.Fatalf("ambiguous subset reported as selected: %+v", view.Unselected)
			}
			if view.Attempt.EvidenceID == "" || view.Attempt.ReceivedDigest == "" {
				t.Fatalf("refusal must preserve received attempt for diagnosis: %+v", view.Attempt)
			}
		})
	}
}

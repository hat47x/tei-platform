package partition

import (
	"encoding/json"
	"os"
	"testing"
)

func writesFromEdges(edges []AccessOccurrence) []Write {
	out := make([]Write, 0, len(edges))
	for _, e := range edges {
		if e.Operation == OpCreate || e.Operation == OpUpdate || e.Operation == OpDelete {
			out = append(out, Write{
				ID: e.ID, Function: e.Function, Data: e.Data, Owner: e.Owner,
				Operation: string(e.Operation), Source: e.SourceLocator, Weight: 1,
			})
		}
	}
	return out
}

func readEdgesFromAccess(edges []AccessOccurrence) []ReadEdge {
	out := make([]ReadEdge, 0, len(edges))
	for _, e := range edges {
		out = append(out, ReadEdge{ID: e.ID, Function: e.Function, Data: e.Data, Provenance: string(e.Provenance), SourceLocator: e.SourceLocator, Notes: e.Notes})
	}
	return out
}

func fixtureProblem() (CandidateProblem, error) {
	fixtureBytes, err := os.ReadFile("testdata/fixture.json")
	if err != nil {
		return CandidateProblem{}, err
	}
	var fixture struct {
		Functions   []Function         `json:"functions"`
		WriteEdges  []AccessOccurrence `json:"writeEdges"`
		ReadEdges   []AccessOccurrence `json:"readEdges"`
		MustLinks   []MustLink         `json:"mustLinks"`
		Constraints []Constraint       `json:"constraints"`
	}
	if err := json.Unmarshal(fixtureBytes, &fixture); err != nil {
		return CandidateProblem{}, err
	}
	return CandidateProblem{
		Functions:   fixture.Functions,
		Writes:      writesFromEdges(fixture.WriteEdges),
		MustLinks:   fixture.MustLinks,
		ReadEdges:   readEdgesFromAccess(fixture.ReadEdges),
		Constraints: fixture.Constraints,
	}, nil
}

func TestGenerateCandidatesFixtureExpected(t *testing.T) {
	p, err := fixtureProblem()
	if err != nil {
		t.Fatal(err)
	}
	expectedBytes, err := os.ReadFile("testdata/fixture-expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected struct {
		Candidates []struct {
			CrossWriteCount  int     `json:"crossWriteCount"`
			CrossWriteWeight int64   `json:"crossWriteWeight"`
			CrossReadCount   int     `json:"crossReadCount"`
			Cohesion         float64 `json:"cohesion"`
			MaxSize          int     `json:"maxSize"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatal(err)
	}

	candidates, status := GenerateCandidates(p)
	if status != "exhaustive-proven" {
		t.Fatalf("expected exhaustive-proven, got %s", status)
	}
	if len(candidates) != len(expected.Candidates) {
		t.Fatalf("candidate count mismatch: got %d want %d", len(candidates), len(expected.Candidates))
	}
	for i, exp := range expected.Candidates {
		if candidates[i].Evaluation.CrossWriteCount != exp.CrossWriteCount {
			t.Fatalf("candidate %d cross write count mismatch: got %d want %d", i, candidates[i].Evaluation.CrossWriteCount, exp.CrossWriteCount)
		}
		if candidates[i].Evaluation.CrossWriteWeight != exp.CrossWriteWeight {
			t.Fatalf("candidate %d cross write weight mismatch: got %d want %d", i, candidates[i].Evaluation.CrossWriteWeight, exp.CrossWriteWeight)
		}
		if candidates[i].Evaluation.CrossReadCount != exp.CrossReadCount {
			t.Fatalf("candidate %d cross read count mismatch: got %d want %d", i, candidates[i].Evaluation.CrossReadCount, exp.CrossReadCount)
		}
		if diff := absFloat64(candidates[i].Evaluation.Cohesion - exp.Cohesion); diff > 1e-6 {
			t.Fatalf("candidate %d cohesion mismatch: got %f want %f (diff=%g)", i, candidates[i].Evaluation.Cohesion, exp.Cohesion, diff)
		}
		maxSize := 0
		for _, s := range candidates[i].Evaluation.Sizes {
			if s > maxSize {
				maxSize = s
			}
		}
		if maxSize != exp.MaxSize {
			t.Fatalf("candidate %d max size mismatch: got %d want %d", i, maxSize, exp.MaxSize)
		}
	}
}

func absFloat64(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestFixtureDeterministic(t *testing.T) {
	p, err := fixtureProblem()
	if err != nil {
		t.Fatal(err)
	}
	first, _ := GenerateCandidates(p)
	for i := 0; i < 50; i++ {
		second, _ := GenerateCandidates(p)
		if len(first) != len(second) {
			t.Fatal("candidate count not deterministic")
		}
		for j := range first {
			if first[j].ID != second[j].ID || first[j].Evaluation.CrossWriteCount != second[j].Evaluation.CrossWriteCount {
				t.Fatal("candidate order not deterministic")
			}
		}
	}
}

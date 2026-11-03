package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func TestSemanticEvaluatorInputIsIsolatedAndMutationRejected(t *testing.T) {
	catalog, contract, hierarchy, provider, conformance := resolvedSUIHierarchySemanticCatalog(t)
	implementation := suiHierarchyEvaluatorImplementation(contract, provider)
	binding := SemanticEvaluatorBinding{
		ConstraintExtension: hierarchy.ID, ConstraintID: "sui-island-hierarchy-integrity",
		ImplementationID: implementation.ID, ArtifactDigest: implementation.ArtifactDigest,
	}
	plan, err := buildSemanticValidationPlan(catalog, []SemanticEvaluatorBinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	input, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "sui-adoption", "M2-three-level.valid.json"))
	if err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(input)
	evaluate := func(impl SemanticEvaluatorImplementation) (SemanticEvaluationEvidence, error) {
		return evaluateSemanticConstraintWithPlan(context.Background(), catalog, plan, hierarchy.ID, binding.ConstraintID, impl, []SemanticInterfaceConformance{conformance}, input)
	}
	valid, err := evaluate(implementation)
	if err != nil || valid.Status != SemanticEvaluationPass || valid.InputFingerprint != sourceindex.Fingerprint(original) {
		t.Fatalf("unchanged input evaluation: %+v, %v", valid, err)
	}
	realEvaluate := implementation.Evaluate
	implementation.Evaluate = func(ctx context.Context, data []byte) (SemanticEvaluationOutcome, error) {
		result, err := realEvaluate(ctx, data)
		data[0] ^= 1
		return result, err
	}
	evidence, err := evaluate(implementation)
	if err == nil || !strings.Contains(err.Error(), "modified its input") || evidence.Schema != "" {
		t.Fatalf("mutating evaluator must not produce evidence: %+v, %v", evidence, err)
	}
	if !bytes.Equal(input, original) {
		t.Fatal("evaluator changed caller input")
	}
	t.Log("actual SUI reference evaluator passes with original input fingerprint; injected input mutation is rejected and caller bytes remain unchanged")
}

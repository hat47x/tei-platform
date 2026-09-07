package main

import (
	"context"
	"path/filepath"
	"testing"

	"tei/internal/driver"
)

func TestSecondCorpusProcedureOrderDoesNotSynthesizeResponsibilityHandoff(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	// Preserve Procedure bodies and their ordered steps while removing the only
	// explicit Handoff authority. BP-08 -> BP-09 adjacency must not become a
	// responsibility transfer merely because the steps are adjacent.
	withoutExplicitHandoff := set
	withoutExplicitHandoff.Handoffs = nil
	model, err := deriveNormalizedBusinessModelProbe(withoutExplicitHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 0 {
		t.Fatalf("Procedure order / adjacency synthesized Business Handoff: %+v", model.Handoffs)
	}

	primary, ok := fixtures["order-entry.order-handling"]
	if !ok {
		t.Fatal("test setup: primary Procedure fixture not found")
	}
	if !stepOccursBefore(primary, "BP-08", "BP-09") {
		t.Fatal("test setup: BP-08 must precede BP-09")
	}
}

func TestSecondCorpusPositivePhysicalObservationDoesNotSynthesizeResponsibilityHandoff(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	execution, err := driver.ExecuteObservationRead(
		context.Background(),
		"second-corpus.physical-state.read",
		driver.ObservationEvidenceExternalObserved,
		func(context.Context) (any, error) { return true, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if observed, ok := execution.Result.(bool); !ok || !observed {
		t.Fatalf("test setup: positive physical Observation not established: %+v", execution)
	}

	withoutExplicitHandoff := set
	withoutExplicitHandoff.Handoffs = nil
	model, err := deriveNormalizedBusinessModelProbe(withoutExplicitHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Handoffs) != 0 {
		t.Fatalf("positive physical Observation synthesized Business Handoff: %+v", model.Handoffs)
	}
}

func TestSecondCorpusExplicitHandoffSourceIsRequiredForNormalizedResponsibilityRelation(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	withoutExplicitHandoff := set
	withoutExplicitHandoff.Handoffs = nil
	without, err := deriveNormalizedBusinessModelProbe(withoutExplicitHandoff, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	withExplicit, err := deriveNormalizedBusinessModelProbe(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	if len(without.Handoffs) != 0 {
		t.Fatalf("explicit sourceなしでHandoffが存在します: %+v", without.Handoffs)
	}
	if len(withExplicit.Handoffs) != 1 {
		t.Fatalf("explicit Handoff sourceをnormalized relationへ保持できません: %+v", withExplicit.Handoffs)
	}
	if err := validateNormalizedBusinessModelProbe(withExplicit); err != nil {
		t.Fatalf("explicit Handoff relationのreferential integrityを検証できません: %v", err)
	}
}

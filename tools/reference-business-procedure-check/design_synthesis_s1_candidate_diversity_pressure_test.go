package main

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// designCandidateMeaningProbe is deliberately test-only. It represents the
// semantic dimensions that make two S1 alternatives materially different;
// user-facing wording and discovery mechanism are kept outside this shape.
type designCandidateMeaningProbe struct {
	Structure     string
	Ownership     string
	LineIdentity  string
	ExternalOwner string
}

type designCandidateSurfaceVariantProbe struct {
	VariantKey    string
	SurfaceLabel  string
	Meaning       designCandidateMeaningProbe
	DiscoveryRefs []string
}

type designSemanticAlternativeProbe struct {
	Meaning       designCandidateMeaningProbe
	VariantKeys   []string
	SurfaceLabels []string
	DiscoveryRefs []string
}

type designCandidateFrontierCardinalityProbe struct {
	GeneratedAlternatives int
	RemainingAlternatives int
	State                 string
}

const (
	frontierSingleGeneratedProbe = "single-generated"
	frontierMultipleRemainProbe  = "multiple-remain"
	frontierContractedProbe      = "contracted"
	frontierEmptyProbe           = "empty"
)

func TestS1CandidateDiversityDoesNotCountParaphrasesAsIndependentAlternatives(t *testing.T) {
	aggregateMeaning := designCandidateMeaningProbe{
		Structure:    "order-with-lines",
		Ownership:    "order-aggregate",
		LineIdentity: "aggregate-local",
	}
	inputs := []designCandidateSurfaceVariantProbe{
		{
			VariantKey:    "aggregate-order-line",
			SurfaceLabel:  "OrderとOrderLineを1:Nで同一aggregateとして扱う",
			Meaning:       aggregateMeaning,
			DiscoveryRefs: []string{"intent:order-entry"},
		},
		{
			VariantKey:    "aggregate-header-detail",
			SurfaceLabel:  "Order headerとdetailを1:Nで同一aggregateとして扱う",
			Meaning:       aggregateMeaning,
			DiscoveryRefs: []string{"pattern:header_detail_transaction"},
		},
		{
			VariantKey:   "independent-line",
			SurfaceLabel: "lineを独立Business objectとして扱う",
			Meaning: designCandidateMeaningProbe{
				Structure:    "order-with-lines",
				Ownership:    "independent-line-object",
				LineIdentity: "business-global",
			},
			DiscoveryRefs: []string{"intent:order-entry"},
		},
	}

	alternatives, err := collapseSurfaceVariantsByMeaningProbe(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(alternatives) != 2 {
		t.Fatalf("surface paraphrases inflated semantic candidate count: %+v", alternatives)
	}

	aggregate := findDesignAlternativeProbe(alternatives, aggregateMeaning)
	if aggregate == nil {
		t.Fatalf("aggregate semantic alternative missing: %+v", alternatives)
	}
	if len(aggregate.VariantKeys) != 2 || len(aggregate.SurfaceLabels) != 2 {
		t.Fatalf("paraphrase provenance was lost while deduplicating meaning: %+v", aggregate)
	}
}

func TestS1KeepsMeaningDistinctAlternativesEvenWhenDiscoveryContextIsShared(t *testing.T) {
	sharedDiscovery := []string{"query:how-to-model-order-lines"}
	inputs := []designCandidateSurfaceVariantProbe{
		{
			VariantKey:    "aggregate",
			SurfaceLabel:  "lineはOrder aggregateの内部要素",
			DiscoveryRefs: sharedDiscovery,
			Meaning: designCandidateMeaningProbe{
				Structure:    "order-with-lines",
				Ownership:    "order-aggregate",
				LineIdentity: "aggregate-local",
			},
		},
		{
			VariantKey:    "independent",
			SurfaceLabel:  "lineは独立Business object",
			DiscoveryRefs: sharedDiscovery,
			Meaning: designCandidateMeaningProbe{
				Structure:    "order-with-lines",
				Ownership:    "independent-line-object",
				LineIdentity: "business-global",
			},
		},
		{
			VariantKey:    "external-owner",
			SurfaceLabel:  "line ownershipを外部Capabilityへ委譲",
			DiscoveryRefs: sharedDiscovery,
			Meaning: designCandidateMeaningProbe{
				Structure:     "order-with-lines",
				Ownership:     "external-capability",
				LineIdentity:  "external-authority",
				ExternalOwner: "fulfillment.line-management",
			},
		},
	}

	alternatives, err := collapseSurfaceVariantsByMeaningProbe(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(alternatives) != 3 {
		t.Fatalf("shared discovery context collapsed meaning-distinct S1 alternatives: %+v", alternatives)
	}
	for _, alternative := range alternatives {
		if len(alternative.DiscoveryRefs) != 1 || alternative.DiscoveryRefs[0] != sharedDiscovery[0] {
			t.Fatalf("discovery provenance changed candidate meaning: %+v", alternative)
		}
	}
}

func TestS1FrontierDistinguishesSingleGenerationFromConstraintContraction(t *testing.T) {
	single, err := classifyDesignCandidateFrontierProbe(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if single.State != frontierSingleGeneratedProbe {
		t.Fatalf("single generated alternative was misclassified: %+v", single)
	}

	contracted, err := classifyDesignCandidateFrontierProbe(3, 1)
	if err != nil {
		t.Fatal(err)
	}
	if contracted.State != frontierContractedProbe {
		t.Fatalf("constraint contraction was hidden as single generation: %+v", contracted)
	}
	if single == contracted {
		t.Fatalf("generation scarcity and contraction became indistinguishable: single=%+v contracted=%+v", single, contracted)
	}
}

func TestS1FrontierRejectsImpossibleCandidateCardinality(t *testing.T) {
	for _, tc := range []struct {
		generated int
		remaining int
	}{
		{generated: -1, remaining: 0},
		{generated: 1, remaining: -1},
		{generated: 1, remaining: 2},
	} {
		if _, err := classifyDesignCandidateFrontierProbe(tc.generated, tc.remaining); err == nil {
			t.Fatalf("invalid frontier cardinality was accepted: generated=%d remaining=%d", tc.generated, tc.remaining)
		}
	}
}

func collapseSurfaceVariantsByMeaningProbe(inputs []designCandidateSurfaceVariantProbe) ([]designSemanticAlternativeProbe, error) {
	byMeaning := map[string]designSemanticAlternativeProbe{}
	for _, input := range inputs {
		if strings.TrimSpace(input.VariantKey) == "" {
			return nil, fmt.Errorf("S1 variant key is required")
		}
		if strings.TrimSpace(input.SurfaceLabel) == "" {
			return nil, fmt.Errorf("S1 surface label is required for %s", input.VariantKey)
		}
		meaningKey, err := designCandidateMeaningKeyProbe(input.Meaning)
		if err != nil {
			return nil, fmt.Errorf("S1 candidate %s: %w", input.VariantKey, err)
		}
		alternative := byMeaning[meaningKey]
		if len(alternative.VariantKeys) == 0 {
			alternative.Meaning = input.Meaning
		}
		alternative.VariantKeys = append(alternative.VariantKeys, input.VariantKey)
		alternative.SurfaceLabels = append(alternative.SurfaceLabels, input.SurfaceLabel)
		alternative.DiscoveryRefs = append(alternative.DiscoveryRefs, input.DiscoveryRefs...)
		alternative.VariantKeys = uniqueSortedStringsProbe(alternative.VariantKeys)
		alternative.SurfaceLabels = uniqueSortedStringsProbe(alternative.SurfaceLabels)
		alternative.DiscoveryRefs = uniqueSortedStringsProbe(alternative.DiscoveryRefs)
		byMeaning[meaningKey] = alternative
	}

	keys := make([]string, 0, len(byMeaning))
	for key := range byMeaning {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]designSemanticAlternativeProbe, 0, len(keys))
	for _, key := range keys {
		result = append(result, byMeaning[key])
	}
	return result, nil
}

func designCandidateMeaningKeyProbe(meaning designCandidateMeaningProbe) (string, error) {
	if strings.TrimSpace(meaning.Structure) == "" {
		return "", fmt.Errorf("semantic structure is required")
	}
	if strings.TrimSpace(meaning.Ownership) == "" {
		return "", fmt.Errorf("semantic ownership is required")
	}
	if strings.TrimSpace(meaning.LineIdentity) == "" {
		return "", fmt.Errorf("semantic line identity is required")
	}
	return strings.Join([]string{
		strings.TrimSpace(meaning.Structure),
		strings.TrimSpace(meaning.Ownership),
		strings.TrimSpace(meaning.LineIdentity),
		strings.TrimSpace(meaning.ExternalOwner),
	}, "\x00"), nil
}

func classifyDesignCandidateFrontierProbe(generated, remaining int) (designCandidateFrontierCardinalityProbe, error) {
	if generated < 0 || remaining < 0 {
		return designCandidateFrontierCardinalityProbe{}, fmt.Errorf("candidate counts must be non-negative")
	}
	if remaining > generated {
		return designCandidateFrontierCardinalityProbe{}, fmt.Errorf("remaining candidates %d exceed generated candidates %d", remaining, generated)
	}
	result := designCandidateFrontierCardinalityProbe{
		GeneratedAlternatives: generated,
		RemainingAlternatives: remaining,
	}
	switch {
	case generated == 0:
		result.State = frontierEmptyProbe
	case generated == 1 && remaining == 1:
		result.State = frontierSingleGeneratedProbe
	case remaining < generated:
		result.State = frontierContractedProbe
	default:
		result.State = frontierMultipleRemainProbe
	}
	return result, nil
}

func findDesignAlternativeProbe(alternatives []designSemanticAlternativeProbe, meaning designCandidateMeaningProbe) *designSemanticAlternativeProbe {
	for i := range alternatives {
		if alternatives[i].Meaning == meaning {
			return &alternatives[i]
		}
	}
	return nil
}

func uniqueSortedStringsProbe(items []string) []string {
	unique := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		unique[trimmed] = struct{}{}
	}
	keys := make([]string, 0, len(unique))
	for key := range unique {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

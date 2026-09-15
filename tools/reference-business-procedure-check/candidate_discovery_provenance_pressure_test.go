package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/ir"
	"tei/internal/workspace"
)

type candidateDiscoveryMechanismProbe string

const (
	discoveryExactLookupProbe          candidateDiscoveryMechanismProbe = "exact-lookup"
	discoveryReferencePatternProbe     candidateDiscoveryMechanismProbe = "reference-pattern"
	discoveryAssociativeRetrievalProbe candidateDiscoveryMechanismProbe = "associative-retrieval"
)

type candidateDiscoveryTraceProbe struct {
	Mechanism     candidateDiscoveryMechanismProbe
	ProvenanceRef string
	BaseRevision  string
}

type referencePatternProvenanceProbe struct {
	PatternKey       string
	Nature           string
	SourceCount      int
	CorpusSize       int
	ResearchArtifact string
}

type candidateDiscoveryProjectionProbe struct {
	CandidateKey      string
	TargetIdentity    string
	DiscoveryTraces   []candidateDiscoveryTraceProbe
	PatternProvenance []referencePatternProvenanceProbe
}

type currentCandidateQualificationProbe struct {
	CandidateKey   string
	TargetIdentity string
	ExactCurrent   bool
}

func TestCandidateDiscoveryProjectionDeduplicatesSameExistingCandidateAcrossMechanisms(t *testing.T) {
	inputs := []candidateDiscoveryProjectionProbe{
		{
			CandidateKey:   "existing:customer.customer",
			TargetIdentity: "customer.customer",
			DiscoveryTraces: []candidateDiscoveryTraceProbe{{
				Mechanism:     discoveryExactLookupProbe,
				ProvenanceRef: "current-m1:customer.customer",
				BaseRevision:  "R-current",
			}},
		},
		{
			CandidateKey:   "existing:customer.customer",
			TargetIdentity: "customer.customer",
			DiscoveryTraces: []candidateDiscoveryTraceProbe{{
				Mechanism:     discoveryAssociativeRetrievalProbe,
				ProvenanceRef: "derived-index:structural-neighbor/customer.customer",
				BaseRevision:  "R-current",
			}},
		},
	}

	merged, err := mergeCandidateDiscoveryProjectionsProbe(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 {
		t.Fatalf("same semantic candidate was duplicated by discovery mechanism: %+v", merged)
	}
	if len(merged[0].DiscoveryTraces) != 2 {
		t.Fatalf("multiple discovery traces were not preserved: %+v", merged[0])
	}
	if merged[0].TargetIdentity != "customer.customer" {
		t.Fatalf("candidate semantic target changed during discovery merge: %+v", merged[0])
	}
}

func TestAssociativeDiscoveryDoesNotReplaceCurrentM1ExactQualification(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	candidate := candidateDiscoveryProjectionProbe{
		CandidateKey:   "existing:customer.customer",
		TargetIdentity: "customer.customer",
		DiscoveryTraces: []candidateDiscoveryTraceProbe{{
			Mechanism:     discoveryAssociativeRetrievalProbe,
			ProvenanceRef: "derived-index:structural-neighbor/customer.customer",
			BaseRevision:  "R-current",
		}},
	}
	qualification := qualifyCurrentDataCandidateProbe(loaded.World, candidate)
	if !qualification.ExactCurrent {
		t.Fatalf("associative candidate was not re-qualified against current M1 identity: %+v", qualification)
	}

	missing := candidate
	missing.CandidateKey = "existing:customer.missing"
	missing.TargetIdentity = "customer.missing"
	missingQualification := qualifyCurrentDataCandidateProbe(loaded.World, missing)
	if missingQualification.ExactCurrent {
		t.Fatalf("associative retrieval made a missing M1 identity applicable: %+v", missingQualification)
	}
}

func TestReferencePatternProvenanceDoesNotBecomeCurrentProofEvidence(t *testing.T) {
	candidate := candidateDiscoveryProjectionProbe{
		CandidateKey: "proposal:order-header-detail",
		DiscoveryTraces: []candidateDiscoveryTraceProbe{{
			Mechanism:     discoveryReferencePatternProbe,
			ProvenanceRef: "pattern-evidence:header_detail_transaction",
			BaseRevision:  "research-v1",
		}},
		PatternProvenance: []referencePatternProvenanceProbe{{
			PatternKey:       "header_detail_transaction",
			Nature:           "observed",
			SourceCount:      8,
			CorpusSize:       12,
			ResearchArtifact: "REFERENCE_PATTERN_EVIDENCE_HANDOFF_2026-09.md",
		}},
	}

	if len(candidate.PatternProvenance) != 1 || candidate.PatternProvenance[0].SourceCount != 8 || candidate.PatternProvenance[0].CorpusSize != 12 {
		t.Fatalf("Pattern support breadth was not preserved as provenance: %+v", candidate.PatternProvenance)
	}
	projectionType := reflect.TypeOf(candidateDiscoveryProjectionProbe{})
	for _, forbidden := range []string{"ProofEvidence", "Confidence", "EvidenceStrength", "Applicability", "Accepted"} {
		if _, found := projectionType.FieldByName(forbidden); found {
			t.Fatalf("Pattern discovery projection promoted provenance into current authority field %q", forbidden)
		}
	}
}

func TestSameCandidateCanPreserveMultipleReferencePatternProvenances(t *testing.T) {
	inputs := []candidateDiscoveryProjectionProbe{
		{
			CandidateKey: "proposal:composed-order-structure",
			DiscoveryTraces: []candidateDiscoveryTraceProbe{{
				Mechanism:     discoveryReferencePatternProbe,
				ProvenanceRef: "pattern-evidence:header_detail_transaction",
				BaseRevision:  "research-v1",
			}},
			PatternProvenance: []referencePatternProvenanceProbe{{
				PatternKey:       "header_detail_transaction",
				Nature:           "observed",
				SourceCount:      8,
				CorpusSize:       12,
				ResearchArtifact: "REFERENCE_PATTERN_EVIDENCE_HANDOFF_2026-09.md",
			}},
		},
		{
			CandidateKey: "proposal:composed-order-structure",
			DiscoveryTraces: []candidateDiscoveryTraceProbe{{
				Mechanism:     discoveryReferencePatternProbe,
				ProvenanceRef: "pattern-evidence:composition_bom_bundle",
				BaseRevision:  "research-v1",
			}},
			PatternProvenance: []referencePatternProvenanceProbe{{
				PatternKey:       "composition_bom_bundle",
				Nature:           "candidate",
				SourceCount:      6,
				CorpusSize:       12,
				ResearchArtifact: "REFERENCE_PATTERN_EVIDENCE_HANDOFF_2026-09.md",
			}},
		},
	}
	merged, err := mergeCandidateDiscoveryProjectionsProbe(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 1 || len(merged[0].PatternProvenance) != 2 || len(merged[0].DiscoveryTraces) != 2 {
		t.Fatalf("multiple Reference Pattern provenances were flattened: %+v", merged)
	}
}

func TestMultipleDiscoveryTracesDoNotAccumulateSemanticAuthority(t *testing.T) {
	candidate := candidateDiscoveryProjectionProbe{
		CandidateKey:   "existing:customer.customer",
		TargetIdentity: "customer.customer",
		DiscoveryTraces: []candidateDiscoveryTraceProbe{
			{Mechanism: discoveryExactLookupProbe, ProvenanceRef: "current-m1:customer.customer", BaseRevision: "R-current"},
			{Mechanism: discoveryAssociativeRetrievalProbe, ProvenanceRef: "derived-index:customer.customer", BaseRevision: "R-current"},
			{Mechanism: discoveryReferencePatternProbe, ProvenanceRef: "pattern-evidence:conditional_subtype_role", BaseRevision: "research-v1"},
		},
	}
	if len(candidate.DiscoveryTraces) != 3 {
		t.Fatal("test setup failed")
	}
	projectionType := reflect.TypeOf(candidateDiscoveryProjectionProbe{})
	for _, forbidden := range []string{"AuthorityScore", "Confidence", "VoteCount", "AcceptanceProbability", "Rank"} {
		if _, found := projectionType.FieldByName(forbidden); found {
			t.Fatalf("multiple discovery paths gained semantic authority field %q", forbidden)
		}
	}
}

func TestStaleDerivedDiscoveryTraceIsNotEligibleForCurrentRetrievalUse(t *testing.T) {
	traces := []candidateDiscoveryTraceProbe{
		{Mechanism: discoveryAssociativeRetrievalProbe, ProvenanceRef: "derived-index:old", BaseRevision: "R-old"},
		{Mechanism: discoveryAssociativeRetrievalProbe, ProvenanceRef: "derived-index:current", BaseRevision: "R-current"},
		{Mechanism: discoveryReferencePatternProbe, ProvenanceRef: "pattern-evidence:header_detail_transaction", BaseRevision: "research-v1"},
	}
	eligible := currentDerivedDiscoveryTracesProbe(traces, "R-current")
	if len(eligible) != 1 || eligible[0].ProvenanceRef != "derived-index:current" {
		t.Fatalf("stale or non-derived trace entered current derived retrieval use: %+v", eligible)
	}
}

func TestRemovingDiscoveryTracesDoesNotChangeCurrentM1Qualification(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	withTraces := candidateDiscoveryProjectionProbe{
		CandidateKey:   "existing:customer.customer",
		TargetIdentity: "customer.customer",
		DiscoveryTraces: []candidateDiscoveryTraceProbe{
			{Mechanism: discoveryExactLookupProbe, ProvenanceRef: "current-m1:customer.customer", BaseRevision: "R-current"},
			{Mechanism: discoveryAssociativeRetrievalProbe, ProvenanceRef: "derived-index:customer.customer", BaseRevision: "R-current"},
		},
	}
	withoutTraces := withTraces
	withoutTraces.DiscoveryTraces = nil

	before := qualifyCurrentDataCandidateProbe(loaded.World, withTraces)
	after := qualifyCurrentDataCandidateProbe(loaded.World, withoutTraces)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("deleting derived discovery traces changed M1 qualification: before=%+v after=%+v", before, after)
	}
}

func mergeCandidateDiscoveryProjectionsProbe(inputs []candidateDiscoveryProjectionProbe) ([]candidateDiscoveryProjectionProbe, error) {
	byKey := map[string]candidateDiscoveryProjectionProbe{}
	for _, input := range inputs {
		if input.CandidateKey == "" {
			return nil, errProcedureMeaning("candidate key is required")
		}
		current, exists := byKey[input.CandidateKey]
		if !exists {
			current = candidateDiscoveryProjectionProbe{
				CandidateKey:   input.CandidateKey,
				TargetIdentity: input.TargetIdentity,
			}
		} else if current.TargetIdentity != input.TargetIdentity {
			return nil, errProcedureMeaning("same candidate key has conflicting target identity")
		}
		current.DiscoveryTraces = append(current.DiscoveryTraces, input.DiscoveryTraces...)
		current.DiscoveryTraces = uniqueSortedDiscoveryTracesProbe(current.DiscoveryTraces)
		mergedPatternProvenance, err := mergeReferencePatternProvenanceProbe(current.PatternProvenance, input.PatternProvenance)
		if err != nil {
			return nil, err
		}
		current.PatternProvenance = mergedPatternProvenance
		byKey[input.CandidateKey] = current
	}

	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]candidateDiscoveryProjectionProbe, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result, nil
}

func mergeReferencePatternProvenanceProbe(existing, incoming []referencePatternProvenanceProbe) ([]referencePatternProvenanceProbe, error) {
	byKey := map[string]referencePatternProvenanceProbe{}
	for _, item := range append(append([]referencePatternProvenanceProbe(nil), existing...), incoming...) {
		if item.PatternKey == "" {
			return nil, errProcedureMeaning("Reference Pattern key is required")
		}
		if previous, found := byKey[item.PatternKey]; found && !reflect.DeepEqual(previous, item) {
			return nil, errProcedureMeaning("same Reference Pattern key has conflicting provenance")
		}
		byKey[item.PatternKey] = item
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]referencePatternProvenanceProbe, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result, nil
}

func qualifyCurrentDataCandidateProbe(world ir.World, candidate candidateDiscoveryProjectionProbe) currentCandidateQualificationProbe {
	return currentCandidateQualificationProbe{
		CandidateKey:   candidate.CandidateKey,
		TargetIdentity: candidate.TargetIdentity,
		ExactCurrent:   semanticDataMatchCount(world, candidate.TargetIdentity) == 1,
	}
}

func currentDerivedDiscoveryTracesProbe(traces []candidateDiscoveryTraceProbe, currentRevision string) []candidateDiscoveryTraceProbe {
	result := make([]candidateDiscoveryTraceProbe, 0)
	for _, trace := range traces {
		if trace.Mechanism != discoveryAssociativeRetrievalProbe || trace.BaseRevision != currentRevision {
			continue
		}
		result = append(result, trace)
	}
	return uniqueSortedDiscoveryTracesProbe(result)
}

func uniqueSortedDiscoveryTracesProbe(items []candidateDiscoveryTraceProbe) []candidateDiscoveryTraceProbe {
	byKey := map[string]candidateDiscoveryTraceProbe{}
	for _, item := range items {
		key := string(item.Mechanism) + "\x00" + item.ProvenanceRef + "\x00" + item.BaseRevision
		byKey[key] = item
	}
	keys := make([]string, 0, len(byKey))
	for key := range byKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]candidateDiscoveryTraceProbe, 0, len(keys))
	for _, key := range keys {
		result = append(result, byKey[key])
	}
	return result
}

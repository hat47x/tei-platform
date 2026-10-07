package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

type ClassificationCoverage struct {
	ContractSHA256          string       `json:"contractSha256"`
	Format                  SourceFormat `json:"format"`
	ObservedKindCount       int          `json:"observedKindCount"`
	MappedObservedKindCount int          `json:"mappedObservedKindCount"`
	UnmappedObservedKinds   []string     `json:"unmappedObservedKinds,omitempty"`
	UnobservedMappedKinds   []string     `json:"unobservedMappedKinds,omitempty"`
	UnknownObservedElements int          `json:"unknownObservedElements"`
	Complete                bool         `json:"complete"`
	AuthorityLimits         []string     `json:"authorityLimits"`
}

func CheckClassificationCoverage(format SourceFormat, receipt PublicationReceipt, contractBytes []byte) (ClassificationCoverage, error) {
	if format != receipt.Format {
		return ClassificationCoverage{}, fmt.Errorf("receipt format %s does not match requested format %s", receipt.Format, format)
	}
	var contract struct {
		ConstructMappings map[string]map[string]json.RawMessage `json:"constructMappings"`
	}
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		return ClassificationCoverage{}, fmt.Errorf("parse transcription contract: %w", err)
	}
	mapped, ok := contract.ConstructMappings[string(format)]
	if !ok {
		return ClassificationCoverage{}, fmt.Errorf("transcription contract has no %s construct mappings", format)
	}
	coverage := ClassificationCoverage{
		Format: format,
		AuthorityLimits: []string{
			"mapping-entry-existence-is-not-semantic-acceptance",
			"classification-coverage-is-not-behavior-equivalence",
			"contract-coverage-does-not-grant-runtime-authority",
		},
	}
	sum := sha256.Sum256(contractBytes)
	coverage.ContractSHA256 = hex.EncodeToString(sum[:])
	for kind := range receipt.ElementCounts {
		coverage.ObservedKindCount++
		if _, exists := mapped[kind]; exists {
			coverage.MappedObservedKindCount++
		} else {
			coverage.UnmappedObservedKinds = append(coverage.UnmappedObservedKinds, kind)
		}
	}
	for kind := range mapped {
		if _, observed := receipt.ElementCounts[kind]; !observed {
			coverage.UnobservedMappedKinds = append(coverage.UnobservedMappedKinds, kind)
		}
	}
	for _, count := range receipt.UnknownElements {
		coverage.UnknownObservedElements += count
	}
	sort.Strings(coverage.UnmappedObservedKinds)
	sort.Strings(coverage.UnobservedMappedKinds)
	coverage.Complete = len(coverage.UnmappedObservedKinds) == 0 && coverage.UnknownObservedElements == 0
	return coverage, nil
}

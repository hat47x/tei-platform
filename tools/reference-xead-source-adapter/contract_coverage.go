package main

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

type ClassificationCoverage struct {
	ContractSHA256            string       `json:"contractSha256"`
	Format                    SourceFormat `json:"format"`
	SourceFormatVersion       string       `json:"sourceFormatVersion"`
	ObservedKindCount         int          `json:"observedKindCount"`
	MappedObservedKindCount   int          `json:"mappedObservedKindCount"`
	UnmappedObservedKinds     []string     `json:"unmappedObservedKinds,omitempty"`
	UnmappedObservedKindCount int          `json:"unmappedObservedKindCount,omitempty"`
	UnobservedMappedKinds     []string     `json:"unobservedMappedKinds,omitempty"`
	UnknownObservedElements   int          `json:"unknownObservedElements"`
	Complete                  bool         `json:"complete"`
	AuthorityLimits           []string     `json:"authorityLimits"`
}

func CheckClassificationCoverage(format SourceFormat, doc ObservedDocument, contractBytes []byte) (ClassificationCoverage, error) {
	profile, err := validateSourceProfile(format, doc, contractBytes)
	if err != nil {
		return ClassificationCoverage{}, err
	}
	mapped, err := parseConstructMappings(format, contractBytes)
	if err != nil {
		return ClassificationCoverage{}, err
	}
	coverage := ClassificationCoverage{
		Format:              format,
		SourceFormatVersion: profile.FormatVersion,
		AuthorityLimits: []string{
			"mapping-entry-existence-is-not-semantic-acceptance",
			"classification-coverage-is-not-behavior-equivalence",
			"contract-coverage-does-not-grant-runtime-authority",
		},
	}
	sum := sha256.Sum256(contractBytes)
	coverage.ContractSHA256 = hex.EncodeToString(sum[:])
	observedCounts := map[string]int{}
	knownCounts := map[string]int{}
	for _, node := range doc.Nodes {
		observedCounts[node.Kind]++
		if node.Known {
			knownCounts[node.Kind]++
		} else {
			coverage.UnknownObservedElements++
		}
	}
	for kind := range observedCounts {
		coverage.ObservedKindCount++
		if _, exists := mapped[kind]; exists && knownCounts[kind] > 0 {
			coverage.MappedObservedKindCount++
			continue
		}
		coverage.UnmappedObservedKindCount++
		if _, standardName := knownElements[format][kind]; standardName {
			coverage.UnmappedObservedKinds = append(coverage.UnmappedObservedKinds, kind)
		}
	}
	for kind := range mapped {
		if knownCounts[kind] == 0 {
			coverage.UnobservedMappedKinds = append(coverage.UnobservedMappedKinds, kind)
		}
	}
	sort.Strings(coverage.UnmappedObservedKinds)
	sort.Strings(coverage.UnobservedMappedKinds)
	coverage.Complete = coverage.UnmappedObservedKindCount == 0 && coverage.UnknownObservedElements == 0
	return coverage, nil
}

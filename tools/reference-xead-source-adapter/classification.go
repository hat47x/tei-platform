package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

type ConstructMapping struct {
	Classification  []string `json:"classification"`
	Extension       string   `json:"extension"`
	Rules           []string `json:"rules,omitempty"`
	AuthorityLimits []string `json:"authorityLimits,omitempty"`
}

type LocalTranscriptionCandidate struct {
	Kind            string   `json:"kind"`
	Locator         string   `json:"locator"`
	MappingStatus   string   `json:"mappingStatus"`
	Classification  []string `json:"classification"`
	Extension       string   `json:"extension,omitempty"`
	Rules           []string `json:"rules,omitempty"`
	AuthorityLimits []string `json:"authorityLimits,omitempty"`
}

type PublishedClassification struct {
	Kind                    string   `json:"kind"`
	ObservedCount           int      `json:"observedCount"`
	MappedObservedCount     int      `json:"mappedObservedCount"`
	UnresolvedObservedCount int      `json:"unresolvedObservedCount,omitempty"`
	MappingStatus           string   `json:"mappingStatus"`
	Classification          []string `json:"classification,omitempty"`
	Extension               string   `json:"extension,omitempty"`
	AuthorityLimits         []string `json:"authorityLimits,omitempty"`
}

type ClassificationProjection struct {
	ContractSHA256               string                    `json:"contractSha256"`
	Format                       SourceFormat              `json:"format"`
	SourceFormatVersion          string                    `json:"sourceFormatVersion"`
	Kinds                        []PublishedClassification `json:"kinds"`
	RedactedUnknownObservedCount int                       `json:"redactedUnknownObservedCount,omitempty"`
	AuthorityLimits              []string                  `json:"authorityLimits"`
}

func ClassifyObservedDocument(format SourceFormat, doc ObservedDocument, contractBytes []byte) ([]LocalTranscriptionCandidate, error) {
	if _, err := validateSourceProfile(format, doc, contractBytes); err != nil {
		return nil, err
	}
	mappings, err := parseConstructMappings(format, contractBytes)
	if err != nil {
		return nil, err
	}
	out := make([]LocalTranscriptionCandidate, 0, len(doc.Nodes))
	for _, node := range doc.Nodes {
		mapping, ok := mappings[node.Kind]
		if !ok || !node.Known {
			out = append(out, LocalTranscriptionCandidate{
				Kind:           node.Kind,
				Locator:        node.Locator.Path,
				MappingStatus:  "unmapped",
				Classification: []string{"unresolved"},
				AuthorityLimits: []string{
					"unmapped-source-construct-is-not-semantic-authority",
					"unknown-source-construct-must-not-be-silently-dropped",
				},
			})
			continue
		}
		out = append(out, LocalTranscriptionCandidate{
			Kind:            node.Kind,
			Locator:         node.Locator.Path,
			MappingStatus:   "mapped",
			Classification:  append([]string(nil), mapping.Classification...),
			Extension:       mapping.Extension,
			Rules:           append([]string(nil), mapping.Rules...),
			AuthorityLimits: append([]string(nil), mapping.AuthorityLimits...),
		})
	}
	return out, nil
}

func BuildClassificationProjection(format SourceFormat, doc ObservedDocument, contractBytes []byte) (ClassificationProjection, error) {
	profile, err := validateSourceProfile(format, doc, contractBytes)
	if err != nil {
		return ClassificationProjection{}, err
	}
	mappings, err := parseConstructMappings(format, contractBytes)
	if err != nil {
		return ClassificationProjection{}, err
	}
	sum := sha256.Sum256(contractBytes)
	out := ClassificationProjection{
		ContractSHA256:      hex.EncodeToString(sum[:]),
		Format:              format,
		SourceFormatVersion: profile.FormatVersion,
		AuthorityLimits: []string{
			"classification-projection-is-candidate-evidence-not-semantic-acceptance",
			"kind-level-public-projection-does-not-disclose-source-values-or-source-locators",
			"unresolved-observed-count-remains-unclassified-even-when-the-same-local-name-has-a-mapping",
			"mapping-extension-does-not-grant-runtime-authority",
		},
	}
	observedCounts := map[string]int{}
	mappedCounts := map[string]int{}
	unresolvedCounts := map[string]int{}
	for _, node := range doc.Nodes {
		observedCounts[node.Kind]++
		if node.Known {
			mappedCounts[node.Kind]++
		} else {
			unresolvedCounts[node.Kind]++
		}
	}
	var kinds []string
	for kind := range observedCounts {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		observedCount := observedCounts[kind]
		unresolvedCount := unresolvedCounts[kind]
		mappedCount := mappedCounts[kind]
		mapping, ok := mappings[kind]
		if !ok {
			if _, standardName := knownElements[format][kind]; !standardName {
				out.RedactedUnknownObservedCount += observedCount
				continue
			}
			out.Kinds = append(out.Kinds, PublishedClassification{
				Kind:                    kind,
				ObservedCount:           observedCount,
				MappedObservedCount:     0,
				UnresolvedObservedCount: observedCount,
				MappingStatus:           "unmapped",
				AuthorityLimits: []string{
					"unmapped-source-construct-is-not-semantic-authority",
					"unknown-source-construct-must-not-be-silently-dropped",
				},
			})
			continue
		}
		if mappedCount <= 0 {
			out.Kinds = append(out.Kinds, PublishedClassification{
				Kind:                    kind,
				ObservedCount:           observedCount,
				MappedObservedCount:     0,
				UnresolvedObservedCount: unresolvedCount,
				MappingStatus:           "unmapped",
				AuthorityLimits: []string{
					"local-name-mapping-does-not-apply-to-unknown-or-namespaced-observation",
					"unknown-source-construct-must-not-be-silently-dropped",
				},
			})
			continue
		}
		status := "mapped"
		if unresolvedCount > 0 {
			status = "mixed"
		}
		out.Kinds = append(out.Kinds, PublishedClassification{
			Kind:                    kind,
			ObservedCount:           observedCount,
			MappedObservedCount:     mappedCount,
			UnresolvedObservedCount: unresolvedCount,
			MappingStatus:           status,
			Classification:          append([]string(nil), mapping.Classification...),
			Extension:               mapping.Extension,
			AuthorityLimits:         append([]string(nil), mapping.AuthorityLimits...),
		})
	}
	return out, nil
}

func parseConstructMappings(format SourceFormat, contractBytes []byte) (map[string]ConstructMapping, error) {
	var contract struct {
		ConstructMappings map[string]map[string]ConstructMapping `json:"constructMappings"`
	}
	if err := json.Unmarshal(contractBytes, &contract); err != nil {
		return nil, fmt.Errorf("parse transcription contract: %w", err)
	}
	mapped, ok := contract.ConstructMappings[string(format)]
	if !ok {
		return nil, fmt.Errorf("transcription contract has no %s construct mappings", format)
	}
	for kind, mapping := range mapped {
		if len(mapping.Classification) == 0 {
			return nil, fmt.Errorf("transcription contract mapping %s/%s has no classification", format, kind)
		}
	}
	return mapped, nil
}

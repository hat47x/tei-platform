package main

import (
	"fmt"
	"sort"
)

type UnresolvedSummary struct {
	Code            string   `json:"code"`
	Kind            string   `json:"kind,omitempty"`
	Nature          string   `json:"nature"`
	Count           int      `json:"count"`
	MappingStatus   string   `json:"mappingStatus,omitempty"`
	Extension       string   `json:"extension,omitempty"`
	AuthorityLimits []string `json:"authorityLimits"`
}

type UnresolvedProjection struct {
	Format          SourceFormat        `json:"format"`
	Items           []UnresolvedSummary `json:"items,omitempty"`
	AuthorityLimits []string            `json:"authorityLimits"`
}

func BuildUnresolvedProjection(
	format SourceFormat,
	receipt PublicationReceipt,
	classification ClassificationProjection,
) (UnresolvedProjection, error) {
	if format != receipt.Format || format != classification.Format {
		return UnresolvedProjection{}, fmt.Errorf(
			"unresolved projection format mismatch: requested=%s receipt=%s classification=%s",
			format,
			receipt.Format,
			classification.Format,
		)
	}
	out := UnresolvedProjection{
		Format: format,
		AuthorityLimits: []string{
			"unresolved-projection-is-loss-ledger-input-not-accepted-loss-ledger",
			"unresolved-evidence-does-not-authorize-inferred-replacement",
			"aggregate-public-output-does-not-disclose-source-values-or-source-locators",
		},
	}

	sourceIssueCounts := map[string]int{}
	for _, issue := range receipt.Unresolved {
		sourceIssueCounts[issue.Code]++
	}
	var issueCodes []string
	for code := range sourceIssueCounts {
		issueCodes = append(issueCodes, code)
	}
	sort.Strings(issueCodes)
	for _, code := range issueCodes {
		out.Items = append(out.Items, UnresolvedSummary{
			Code:   code,
			Nature: "source-observation-unresolved",
			Count:  sourceIssueCounts[code],
			AuthorityLimits: []string{
				"source-observation-unresolved-must-not-be-silently-dropped",
				"source-observation-unresolved-is-not-semantic-acceptance",
			},
		})
	}

	for _, kind := range classification.Kinds {
		if kind.UnresolvedObservedCount > 0 {
			code := "classification-unmapped"
			if kind.MappingStatus == "mixed" {
				code = "classification-mixed"
			}
			out.Items = append(out.Items, UnresolvedSummary{
				Code:          code,
				Kind:          kind.Kind,
				Nature:        "mapping-unresolved",
				Count:         kind.UnresolvedObservedCount,
				MappingStatus: kind.MappingStatus,
				Extension:     kind.Extension,
				AuthorityLimits: []string{
					"unmapped-or-mixed-source-construct-must-not-be-inferred-into-a-known-mapping",
					"mapping-unresolved-is-not-semantic-rejection",
				},
			})
		}
		if !containsClassification(kind.Classification, "unresolved") {
			continue
		}
		mappedCount := kind.ObservedCount - kind.UnresolvedObservedCount
		if mappedCount <= 0 {
			continue
		}
		out.Items = append(out.Items, UnresolvedSummary{
			Code:          "semantic-classification-unresolved",
			Kind:          kind.Kind,
			Nature:        "semantic-candidate-unresolved",
			Count:         mappedCount,
			MappingStatus: kind.MappingStatus,
			Extension:     kind.Extension,
			AuthorityLimits: []string{
				"mapped-source-construct-is-not-accepted-semantic-meaning",
				"semantic-unresolved-requires-explicit-evidence-or-shared-acceptance",
			},
		})
	}

	sort.Slice(out.Items, func(i, j int) bool {
		if out.Items[i].Nature != out.Items[j].Nature {
			return out.Items[i].Nature < out.Items[j].Nature
		}
		if out.Items[i].Code != out.Items[j].Code {
			return out.Items[i].Code < out.Items[j].Code
		}
		return out.Items[i].Kind < out.Items[j].Kind
	})
	return out, nil
}

func containsClassification(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

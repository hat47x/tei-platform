package main

import (
	"fmt"
	"sort"
)

const transportSchema = "tei.reference.epistemic-transport/v0"

type TransportAccess struct {
	GrantedAccessRefs []string `json:"grantedAccessRefs,omitempty"`
}

type TransportRedactionSummary struct {
	Assertions    int `json:"assertions,omitempty"`
	Evidence      int `json:"evidence,omitempty"`
	ReviewEvents  int `json:"reviewEvents,omitempty"`
	Relations     int `json:"relations,omitempty"`
	Dependencies  int `json:"dependencies,omitempty"`
	CoverageAreas int `json:"coverageAreas,omitempty"`
}

type TransportEnvelope struct {
	Schema          string                    `json:"schema"`
	Corpus          Corpus                    `json:"corpus"`
	Redactions      TransportRedactionSummary `json:"redactions"`
	AuthorityLimits []string                  `json:"authorityLimits"`
}

func exportCorpusForAccess(raw Corpus, access TransportAccess) (TransportEnvelope, error) {
	corpus, err := normalizeCorpus(raw)
	if err != nil {
		return TransportEnvelope{}, err
	}

	granted := make(map[string]struct{})
	for _, ref := range normalizeStrings(access.GrantedAccessRefs) {
		granted[ref] = struct{}{}
	}

	visible := make(map[string]struct{})
	redactions := TransportRedactionSummary{}
	for _, assertion := range corpus.Assertions {
		if accessGranted(assertion.AccessRefs, granted) {
			visible[assertion.ID] = struct{}{}
		} else {
			redactions.Assertions++
		}
	}

	exported := Corpus{
		Schema:        corpus.Schema,
		MetadataState: MetadataComplete,
	}
	hiddenRelationTargets := map[string]bool{}
	for _, assertion := range corpus.Assertions {
		if _, hidden := visible[assertion.ID]; hidden {
			continue
		}
		for _, target := range assertion.ConflictsWith {
			if _, ok := visible[target]; ok {
				hiddenRelationTargets[target] = true
				redactions.Relations++
			}
		}
		for _, target := range assertion.Supersedes {
			if _, ok := visible[target]; ok {
				hiddenRelationTargets[target] = true
				redactions.Relations++
			}
		}
	}

	for _, assertion := range corpus.Assertions {
		if _, ok := visible[assertion.ID]; !ok {
			continue
		}
		copy := assertion
		partial := hiddenRelationTargets[assertion.ID]

		copy.Evidence, partial, redactions.Evidence = filterEvidenceForAccess(
			copy.Evidence,
			granted,
			partial,
			redactions.Evidence,
		)

		events := make([]ReviewEvent, 0, len(copy.ReviewEvents))
		for _, event := range copy.ReviewEvents {
			if !accessGranted(event.AccessRefs, granted) {
				redactions.ReviewEvents++
				partial = true
				continue
			}
			filtered := event
			filtered.Evidence, partial, redactions.Evidence = filterEvidenceForAccess(
				filtered.Evidence,
				granted,
				partial,
				redactions.Evidence,
			)
			events = append(events, filtered)
		}
		copy.ReviewEvents = events

		copy.Supersedes, partial, redactions.Relations = filterVisibleIDs(
			copy.Supersedes,
			visible,
			partial,
			redactions.Relations,
		)
		copy.ConflictsWith, partial, redactions.Relations = filterVisibleIDs(
			copy.ConflictsWith,
			visible,
			partial,
			redactions.Relations,
		)

		if partial && copy.MetadataState == MetadataComplete {
			copy.MetadataState = MetadataPartial
		}
		exported.Assertions = append(exported.Assertions, copy)
	}

	for _, dependency := range corpus.Dependencies {
		if _, ok := visible[dependency.AssertionID]; !ok || !accessGranted(dependency.AccessRefs, granted) {
			redactions.Dependencies++
			continue
		}
		exported.Dependencies = append(exported.Dependencies, dependency)
	}

	for _, area := range corpus.CoverageAreas {
		if !accessGranted(area.AccessRefs, granted) {
			redactions.CoverageAreas++
			continue
		}
		copy := area
		mapped := make([]string, 0, len(area.MappedAssertionIDs))
		for _, id := range area.MappedAssertionIDs {
			if _, ok := visible[id]; ok {
				mapped = append(mapped, id)
			} else {
				redactions.Relations++
			}
		}
		copy.MappedAssertionIDs = mapped
		exported.CoverageAreas = append(exported.CoverageAreas, copy)
	}

	if redactions.Assertions > 0 ||
		redactions.Evidence > 0 ||
		redactions.ReviewEvents > 0 ||
		redactions.Relations > 0 ||
		redactions.Dependencies > 0 ||
		redactions.CoverageAreas > 0 {
		exported.MetadataState = MetadataPartial
	}
	if len(exported.Assertions) == 0 {
		return TransportEnvelope{}, fmt.Errorf("transport contains no accessible assertions")
	}

	normalized, err := normalizeCorpus(exported)
	if err != nil {
		return TransportEnvelope{}, fmt.Errorf("normalize exported corpus: %w", err)
	}
	return TransportEnvelope{
		Schema:     transportSchema,
		Corpus:     normalized,
		Redactions: redactions,
		AuthorityLimits: []string{
			"transport-access-refs-are-opaque-input-not-authorization-authority",
			"transport-does-not-broaden-source-access",
			"redacted-metadata-does-not-preserve-confirmation-authority",
			"partial-transport-health-is-not-project-wide-health",
		},
	}, nil
}

func importTransportEnvelope(raw TransportEnvelope) (Corpus, error) {
	if raw.Schema != transportSchema {
		return Corpus{}, fmt.Errorf("unsupported transport schema %q", raw.Schema)
	}
	corpus, err := normalizeCorpus(raw.Corpus)
	if err != nil {
		return Corpus{}, err
	}
	return corpus, nil
}

func filterEvidenceForAccess(
	values []EvidenceRef,
	granted map[string]struct{},
	partial bool,
	redacted int,
) ([]EvidenceRef, bool, int) {
	out := make([]EvidenceRef, 0, len(values))
	for _, value := range values {
		if !accessGranted(value.AccessRefs, granted) {
			redacted++
			partial = true
			continue
		}
		out = append(out, value)
	}
	return out, partial, redacted
}

func filterVisibleIDs(
	values []string,
	visible map[string]struct{},
	partial bool,
	redacted int,
) ([]string, bool, int) {
	out := make([]string, 0, len(values))
	for _, id := range values {
		if _, ok := visible[id]; !ok {
			redacted++
			partial = true
			continue
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, partial, redacted
}

func accessGranted(required []string, granted map[string]struct{}) bool {
	for _, ref := range required {
		if _, ok := granted[ref]; !ok {
			return false
		}
	}
	return true
}

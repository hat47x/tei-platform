package xeadsource

import (
	"sort"
	"strings"
)

const RelationalResolutionPolicyRevision = "tei.xeadsource.relational-resolution/v1"

const (
	RelationalFactResolutionResolved    = "resolved"
	RelationalFactResolutionUnresolved  = "unresolved"
	RelationalFactResolutionUnvalidated = "unvalidated"
)

type RelationalFactResolution struct {
	Kind            string   `json:"kind"`
	Locator         string   `json:"locator"`
	Status          string   `json:"status"`
	DiagnosticCodes []string `json:"diagnosticCodes,omitempty"`
}

type Inspection struct {
	Document                           Document                   `json:"document"`
	Candidates                         []Candidate                `json:"candidates"`
	RelationalFacts                    RelationalSourceFacts      `json:"relationalFacts"`
	RelationalDiagnostics              []RelationalDiagnostic     `json:"relationalDiagnostics,omitempty"`
	RelationalFactResolutions          []RelationalFactResolution `json:"relationalFactResolutions,omitempty"`
	RelationalResolutionStatus         string                     `json:"relationalResolutionStatus"`
	RelationalResolutionPolicyRevision string                     `json:"relationalResolutionPolicyRevision"`
	ClassificationPolicyFingerprint    string                     `json:"classificationPolicyFingerprint,omitempty"`
	TranscriptionContractFingerprint   string                     `json:"transcriptionContractFingerprint,omitempty"`
	AuthorityLimits                    []string                   `json:"authorityLimits"`
}

func Inspect(format Format, source []byte, policy ClassificationPolicy) (Inspection, error) {
	doc, err := Parse(format, source)
	if err != nil {
		return Inspection{}, err
	}
	facts := ExtractRelationalSourceFacts(doc)
	diagnostics := DiagnoseRelationalSourceFacts(doc, facts)
	factResolutions := ProjectRelationalFactResolutions(facts, diagnostics)
	resolutionStatus := relationalResolutionStatus(factResolutions, diagnostics)
	return Inspection{
		Document:                           doc,
		Candidates:                         Classify(doc, policy),
		RelationalFacts:                    facts,
		RelationalDiagnostics:              diagnostics,
		RelationalFactResolutions:          factResolutions,
		RelationalResolutionStatus:         resolutionStatus,
		RelationalResolutionPolicyRevision: RelationalResolutionPolicyRevision,
		ClassificationPolicyFingerprint:    ClassificationPolicyFingerprint(policy),
		AuthorityLimits: []string{
			"source-inspection-is-not-canonical-authority",
			"source-fact-is-not-accepted-semantic-meaning",
			"candidate-classification-is-not-shared-acceptance",
			"unresolved-source-meaning-must-not-be-auto-completed",
			"relational-resolution-status-is-source-consistency-not-canonical-meaning",
			"relational-fact-resolution-is-structural-not-semantic-acceptance",
			"relational-resolution-policy-revision-is-algorithm-provenance-not-semantic-authority",
			"resolved-relational-fact-is-only-eligible-not-authorized-for-local-lowering",
			"unvalidated-relational-fact-blocks-deterministic-semantic-lowering",
			"unresolved-relational-resolution-blocks-full-set-deterministic-semantic-lowering",
		},
	}, nil
}

func ProjectRelationalFactResolutions(
	facts RelationalSourceFacts,
	diagnostics []RelationalDiagnostic,
) []RelationalFactResolution {
	out := make([]RelationalFactResolution, 0,
		len(facts.LegacyKeys)+
			len(facts.RelationshipKeys)+
			len(facts.DeclaredKeyFields)+
			len(facts.RuntimeKeys)+
			len(facts.RuntimeRefers),
	)

	for _, fact := range facts.LegacyKeys {
		out = append(out, resolvedRelationalFact(
			"legacy-key",
			fact.Locator,
			diagnosticsForLegacyKey(fact, diagnostics),
		))
	}
	for _, fact := range facts.RelationshipKeys {
		out = append(out, resolvedRelationalFact(
			"relationship-key",
			fact.Locator,
			diagnosticsForRelationship(fact, diagnostics),
		))
	}
	for _, fact := range facts.DeclaredKeyFields {
		out = append(out, RelationalFactResolution{
			Kind:    "declared-key-fields",
			Locator: fact.Locator,
			Status:  RelationalFactResolutionUnvalidated,
		})
	}
	for _, fact := range facts.RuntimeKeys {
		out = append(out, resolvedRelationalFact(
			"runtime-key",
			fact.Locator,
			diagnosticsForRuntimeKey(fact, diagnostics),
		))
	}
	for _, fact := range facts.RuntimeRefers {
		out = append(out, resolvedRelationalFact(
			"runtime-refer",
			fact.Locator,
			diagnosticsForRuntimeRefer(fact, diagnostics),
		))
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Locator < out[j].Locator
	})
	return out
}

func relationalResolutionStatus(
	values []RelationalFactResolution,
	diagnostics []RelationalDiagnostic,
) string {
	if len(diagnostics) != 0 {
		return RelationalFactResolutionUnresolved
	}
	status := RelationalFactResolutionResolved
	for _, value := range values {
		switch value.Status {
		case RelationalFactResolutionUnresolved:
			return RelationalFactResolutionUnresolved
		case RelationalFactResolutionUnvalidated:
			status = "partial"
		}
	}
	return status
}

func resolvedRelationalFact(
	kind string,
	locator string,
	diagnostics []RelationalDiagnostic,
) RelationalFactResolution {
	codes := make([]string, 0, len(diagnostics))
	seen := map[string]struct{}{}
	for _, diagnostic := range diagnostics {
		if _, exists := seen[diagnostic.Code]; exists {
			continue
		}
		seen[diagnostic.Code] = struct{}{}
		codes = append(codes, diagnostic.Code)
	}
	sort.Strings(codes)
	status := RelationalFactResolutionResolved
	if len(codes) != 0 {
		status = RelationalFactResolutionUnresolved
	}
	return RelationalFactResolution{
		Kind:            kind,
		Locator:         locator,
		Status:          status,
		DiagnosticCodes: codes,
	}
}

func diagnosticsForLegacyKey(
	fact LegacyKeyFact,
	diagnostics []RelationalDiagnostic,
) []RelationalDiagnostic {
	return filterRelationalDiagnostics(diagnostics, func(diagnostic RelationalDiagnostic) bool {
		if diagnostic.Locator == fact.Locator ||
			strings.HasPrefix(diagnostic.Locator, fact.Locator+"/") {
			return true
		}
		return strings.HasPrefix(diagnostic.Code, "key-") &&
			fact.KeyLegacyID != "" &&
			diagnostic.KeyLegacyID == fact.KeyLegacyID &&
			diagnostic.TableLegacyID == fact.TableLegacyID
	})
}

func diagnosticsForRelationship(
	fact RelationshipKeyFact,
	diagnostics []RelationalDiagnostic,
) []RelationalDiagnostic {
	return filterRelationalDiagnostics(diagnostics, func(diagnostic RelationalDiagnostic) bool {
		if !strings.HasPrefix(diagnostic.Code, "relationship-") {
			return false
		}
		return diagnostic.Locator == fact.Locator ||
			(fact.RelationshipID != "" &&
				diagnostic.RelationshipID == fact.RelationshipID)
	})
}

func diagnosticsForRuntimeKey(
	fact RuntimeKeyFact,
	diagnostics []RelationalDiagnostic,
) []RelationalDiagnostic {
	return filterRelationalDiagnostics(diagnostics, func(diagnostic RelationalDiagnostic) bool {
		if !strings.HasPrefix(diagnostic.Code, "runtime-key-") {
			return false
		}
		if diagnostic.Locator == fact.Locator {
			return true
		}
		return fact.KeyLegacyID != "" &&
			diagnostic.KeyLegacyID == fact.KeyLegacyID &&
			diagnostic.TableLegacyID == fact.TableLegacyID
	})
}

func diagnosticsForRuntimeRefer(
	fact RuntimeReferFact,
	diagnostics []RelationalDiagnostic,
) []RelationalDiagnostic {
	return filterRelationalDiagnostics(diagnostics, func(diagnostic RelationalDiagnostic) bool {
		return strings.HasPrefix(diagnostic.Code, "runtime-refer-") &&
			diagnostic.Locator == fact.Locator
	})
}

func filterRelationalDiagnostics(
	diagnostics []RelationalDiagnostic,
	matches func(RelationalDiagnostic) bool,
) []RelationalDiagnostic {
	out := make([]RelationalDiagnostic, 0)
	for _, diagnostic := range diagnostics {
		if matches(diagnostic) {
			out = append(out, diagnostic)
		}
	}
	return out
}

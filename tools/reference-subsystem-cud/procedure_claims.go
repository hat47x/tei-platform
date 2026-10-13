package main

import (
	"encoding/json"
)

// authoredProcedure is a read-only projection of the *reference authoring*
// fixture, not a canonical Business Procedure model or acceptance record.
// Other schemas must be mapped by an explicit, separately reviewed adapter.
type authoredProcedure struct {
	SchemaVersion string `json:"schemaVersion"`
	ArtifactRole  string `json:"artifactRole"`
	Canonical     bool   `json:"canonical"`
	Steps         []struct {
		ID                  string   `json:"id"`
		Realization         string   `json:"realization"`
		ApplicationPatterns []string `json:"applicationPatterns"`
	} `json:"steps"`
}

// verifyProcedureClaims compares only explicitly supplied candidate steps
// against the *same bytes* already bound by ProcedureJSONFingerprint.
// A verified correspondence is still not an Accepted Business definition.
func verifyProcedureClaims(raw []byte, facts Facts) []Diagnostic {
	add := func(out []Diagnostic, code, subject, source, explanation string) []Diagnostic {
		return append(out, Diagnostic{
			Code: code, Severity: "unresolved", Subject: subject,
			Source: source, Explanation: explanation,
		})
	}
	var procedure authoredProcedure
	if err := json.Unmarshal(raw, &procedure); err != nil {
		// The outer snapshot reader rejects invalid JSON first. Keep a
		// defensive negative finding rather than declaring the scope clean.
		return add(nil, "PROCEDURE_SOURCE_UNREADABLE", "procedure", "", err.Error())
	}
	if procedure.SchemaVersion != "business-procedure-reference-v0" ||
		procedure.ArtifactRole != "reference-authoring-fixture" ||
		procedure.Canonical || procedure.Steps == nil {
		return add(nil, "PROCEDURE_SOURCE_UNSUPPORTED", "procedure", "",
			"only the noncanonical business-procedure-reference-v0 authoring fixture can be compared; this does not establish Source acceptance")
	}
	steps := map[string][]int{}
	var findings []Diagnostic
	for i, step := range procedure.Steps {
		if step.ID == "" {
			findings = add(findings, "PROCEDURE_SOURCE_STEP_UNRESOLVED", "procedure", "",
				"authored Procedure Step ID must not be empty")
			continue
		}
		steps[step.ID] = append(steps[step.ID], i)
	}
	if len(facts.ProcedureSteps) == 0 {
		findings = add(findings, "PROCEDURE_CANDIDATE_SCOPE_EMPTY", "procedure", "",
			"Procedure bytes were supplied, but no candidate Procedure Step claims were declared")
	}
	for _, candidate := range facts.ProcedureSteps {
		if candidate.ID == "" {
			continue // Existing candidate validation reports incomplete steps.
		}
		matches := steps[candidate.ID]
		switch len(matches) {
		case 0:
			findings = add(findings, "PROCEDURE_STEP_NOT_FOUND", candidate.ID, candidate.Source,
				"candidate Step does not exist in the supplied Procedure source")
			continue
		case 1:
			// A unique source Step is required to compare intent.
		default:
			findings = add(findings, "PROCEDURE_SOURCE_STEP_AMBIGUOUS", candidate.ID, candidate.Source,
				"candidate Step ID occurs more than once in the supplied Procedure source")
			continue
		}
		step := procedure.Steps[matches[0]]
		switch step.Realization {
		case "manual-unresolved":
			if candidate.Realization != "manual" || candidate.Pattern != "" || len(step.ApplicationPatterns) != 0 {
				findings = add(findings, "PROCEDURE_REALIZATION_MISMATCH", candidate.ID, candidate.Source,
					"unresolved manual source Step must not be promoted to a system function or Pattern")
			}
		case "application-use":
			if len(step.ApplicationPatterns) != 1 || step.ApplicationPatterns[0] == "" {
				findings = add(findings, "PROCEDURE_PATTERN_CARDINALITY_UNSUPPORTED", candidate.ID, candidate.Source,
					"single-Pattern candidate cannot represent a Procedure Step with zero, empty, or multiple Patterns")
				continue
			}
			if candidate.Realization != "system" {
				findings = add(findings, "PROCEDURE_REALIZATION_MISMATCH", candidate.ID, candidate.Source,
					"explicit Application Pattern requires a system candidate rather than manual or external inference")
			} else if candidate.Pattern != step.ApplicationPatterns[0] {
				findings = add(findings, "PROCEDURE_PATTERN_MISMATCH", candidate.ID, candidate.Source,
					"candidate system Pattern differs from the explicit Procedure source reference")
			}
		default:
			// Validation, authority, observation, and manual/external
			// unresolved boundaries are not silently turned into Functions.
			findings = add(findings, "PROCEDURE_REALIZATION_UNSUPPORTED", candidate.ID, candidate.Source,
				"source Step realization has no safe mapping to candidate system/manual/external")
		}
	}
	return findings
}

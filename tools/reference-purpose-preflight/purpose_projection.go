package main

// PurposeProjection is a derived, read-only development aid. It consumes the
// existing Source/Knowledge/Decision reports without inventing an acceptance,
// authorization or execution boundary. States in this package are local to this
// reference tool, not Core contract enums.

import (
	"fmt"
	"strings"
)

type PurposeObligation struct {
	Requirement string `json:"requirement"`
	Status      string `json:"status"`
	Reason      string `json:"reason"`
}

type PurposeProjection struct {
	SchemaVersion    string              `json:"schemaVersion"`
	Authority        string              `json:"authority"`
	Purpose          string              `json:"purpose"`
	SubjectKind      string              `json:"subjectKind"`
	SubjectID        string              `json:"subjectId"`
	Obligations      []PurposeObligation `json:"obligations"`
	ExecutionAllowed bool                `json:"executionAllowed"`
}

// ProjectForPurpose explains only the obligations applicable to the requested
// developer activity. In particular, evidence of a recorded change decision
// never becomes proof of human identity, effective rule scope, or Host authority.
// Runtime obligations are descriptions of future checks, not a permission.
func ProjectForPurpose(report WorkspaceAssessment, purpose string) (PurposeProjection, error) {
	switch purpose {
	case "describe", "simulate", "execute-preflight":
	default:
		return PurposeProjection{}, fmt.Errorf("unsupported development purpose %q", purpose)
	}
	if strings.TrimSpace(report.SubjectID) == "" || strings.TrimSpace(report.SubjectKind) == "" {
		return PurposeProjection{}, fmt.Errorf("typed subject is required")
	}
	projection := PurposeProjection{
		SchemaVersion:    "tei.reference.purpose-obligations/v0",
		Authority:        "derived-read-only-no-execution",
		Purpose:          purpose,
		SubjectKind:      report.SubjectKind,
		SubjectID:        report.SubjectID,
		Obligations:      []PurposeObligation{},
		ExecutionAllowed: false,
	}
	add := func(name, state, reason string) {
		projection.Obligations = append(projection.Obligations, PurposeObligation{
			Requirement: name, Status: state, Reason: reason,
		})
	}
	if report.SourceStatus != "traceable-source-declaration" || len(report.Witnesses) == 0 {
		add("typed-source", "unresolved", "qualified-current-source-not-established")
	} else {
		add("typed-source", "observed", "source-index-definition-witness")
	}
	if report.RuleMeaning != "canonical-validation-declared" {
		add("business-meaning", "unresolved", "accepted-rule-semantics-not-established-by-source")
	} else {
		add("business-meaning", "declared", "canonical-validation-exists-but-human-review-not-proven")
	}
	if purpose == "describe" {
		return projection, nil
	}
	// Simulation can use a synthetic fixture, but this inspection has not
	// received any actual test inputs or independent expected outcomes.
	add("evaluation-fixture", "not-evaluated", "runtime-values-and-independent-oracle-not-supplied")
	if purpose == "simulate" {
		return projection, nil
	}

	// The Decision Index is only a report. Even exact source matching does not
	// prove the actor's identity, substantive business approval, effective
	// period, or authorization. Never use it as a satisfied obligation.
	if report.DecisionHistoryStatus == "reported-source-matched" {
		add("human-acceptance", "unverified", "historical-source-matched-decision-not-human-attestation")
	} else {
		add("human-acceptance", "unverified", "accepted-purpose-specific-decision-not-established")
	}
	// Whether time selection, a remote provider, or a mutation fence is needed
	// depends on the actual Operation and its accepted effect contract. A
	// Validation declaration alone is insufficient to impose those as gates.
	add("business-effective-scope", "conditional", "required-if-rule-or-object-selection-is-time-dependent")
	add("runtime-business-validation", "not-evaluated", "actual-order-values-and-revision-not-read")
	add("provider-contract", "conditional", "required-if-operation-uses-bound-provider")
	add("host-authorization", "not-evaluated", "execution-principal-and-current-policy-not-qualified")
	add("effect-fence", "conditional", "required-if-operation-has-write-or-external-effect")
	add("scenario-coverage", "not-evaluated", "driver-scenario-oracle-not-executed")
	return projection, nil
}

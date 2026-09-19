package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const (
	actorContextR3AVDecisionAccept = "accept-observed-outcome"
	actorContextR3AVDecisionReject = "reject-observed-outcome"
)

type actorContextR3AVOutcomeAcceptanceWitnessProbe struct {
	ObservationIdentity                  string
	OutcomeAcceptanceAuthorityEvidenceID string
	Decision                             string
	AcceptanceProvenanceRef              string
}

type actorContextR3AVOutcomeAcceptanceCandidateProbe struct {
	ObservationIdentity                  string
	WriteAttemptCandidateIdentity        string
	RawReportedOutcome                   string
	PostWriteObservedDecisionRevision    string
	OutcomeAcceptanceAuthorityEvidenceID string
	OutcomeDecisionLocusRef              string
	OutcomeAuthorityProvenanceRef        string
	Decision                             string
	AcceptanceProvenanceRef              string
	OutcomeAcceptanceCandidateIdentity   string
	AuthorityLimits                      []string
}

func TestActorContextR3AVCurrentUnresolvedGateCannotCreateCandidate(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t, actorContextR3ATReportedApplied,
	)
	gate, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt, observation, businessAddress,
		actorContextR3AUCurrentInspectedEvidence(attempt, observation),
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusUnresolved {
		t.Fatalf("R3AV setup expected unresolved R3AU Gate: %+v", gate)
	}
	if _, err := actorContextR3AVBuildCandidates(
		attempt, observation, businessAddress, gate, nil,
		[]actorContextR3AVOutcomeAcceptanceWitnessProbe{{
			ObservationIdentity:                  observation.ObservationIdentity,
			OutcomeAcceptanceAuthorityEvidenceID: "test-only:r3av/not-qualified",
			Decision:                             actorContextR3AVDecisionAccept,
			AcceptanceProvenanceRef:              "test-only:r3av/acceptance",
		}},
	); err == nil {
		t.Fatal("R3AV created candidate while outcome acceptance authority was unresolved")
	}
}

func TestActorContextR3AVSingleQualifiedGateDoesNotAutoAccept(t *testing.T) {
	attempt, observation, businessAddress, gate, evidence := actorContextR3AVSingleQualifiedFixture(
		t, actorContextR3ATReportedApplied,
	)
	got, err := actorContextR3AVBuildCandidates(
		attempt, observation, businessAddress, gate, evidence, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3AV fabricated candidate without explicit acceptance witness: %+v", got)
	}
}

func TestActorContextR3AVExplicitAcceptAndRejectRemainCandidateDecisions(t *testing.T) {
	for _, decision := range []string{
		actorContextR3AVDecisionAccept,
		actorContextR3AVDecisionReject,
	} {
		t.Run(decision, func(t *testing.T) {
			attempt, observation, businessAddress, gate, evidence := actorContextR3AVSingleQualifiedFixture(
				t, actorContextR3ATReportedApplied,
			)
			got, err := actorContextR3AVBuildCandidates(
				attempt, observation, businessAddress, gate, evidence,
				[]actorContextR3AVOutcomeAcceptanceWitnessProbe{{
					ObservationIdentity:                  observation.ObservationIdentity,
					OutcomeAcceptanceAuthorityEvidenceID: evidence[0].EvidenceID,
					Decision:                             decision,
					AcceptanceProvenanceRef:              "test-only:r3av/" + decision,
				}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].Decision != decision ||
				got[0].RawReportedOutcome != observation.ReportedOutcome ||
				got[0].OutcomeAcceptanceCandidateIdentity == "" {
				t.Fatalf("R3AV acceptance decision candidate drifted: %+v", got)
			}
		})
	}
}

func TestActorContextR3AVWrongObservationOrAuthorityFailsClosed(t *testing.T) {
	attempt, observation, businessAddress, gate, evidence := actorContextR3AVSingleQualifiedFixture(
		t, actorContextR3ATReportedApplied,
	)
	base := actorContextR3AVOutcomeAcceptanceWitnessProbe{
		ObservationIdentity:                  observation.ObservationIdentity,
		OutcomeAcceptanceAuthorityEvidenceID: evidence[0].EvidenceID,
		Decision:                             actorContextR3AVDecisionAccept,
		AcceptanceProvenanceRef:              "test-only:r3av/acceptance",
	}
	cases := map[string]actorContextR3AVOutcomeAcceptanceWitnessProbe{
		"observation": func() actorContextR3AVOutcomeAcceptanceWitnessProbe {
			x := base
			x.ObservationIdentity += ":other"
			return x
		}(),
		"authority": func() actorContextR3AVOutcomeAcceptanceWitnessProbe {
			x := base
			x.OutcomeAcceptanceAuthorityEvidenceID += ":other"
			return x
		}(),
	}
	for name, witness := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := actorContextR3AVBuildCandidates(
				attempt, observation, businessAddress, gate, evidence,
				[]actorContextR3AVOutcomeAcceptanceWitnessProbe{witness},
			); err == nil {
				t.Fatalf("R3AV accepted mismatched %s witness", name)
			}
		})
	}
}

func TestActorContextR3AVMultipleQualifiedGateCannotBeBypassed(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t, actorContextR3ATReportedApplied,
	)
	a := actorContextR3AUCompleteEvidence("test-only:r3av/a", businessAddress, attempt, observation)
	b := actorContextR3AUCompleteEvidence("test-only:r3av/b", businessAddress, attempt, observation)
	evidence := []actorContextR3AUOutcomeAcceptanceEvidenceProbe{a, b}
	gate, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt, observation, businessAddress, evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusMultipleQualified {
		t.Fatalf("R3AV setup lost multiple-qualified R3AU state: %+v", gate)
	}
	if _, err := actorContextR3AVBuildCandidates(
		attempt, observation, businessAddress, gate, evidence,
		[]actorContextR3AVOutcomeAcceptanceWitnessProbe{{
			ObservationIdentity:                  observation.ObservationIdentity,
			OutcomeAcceptanceAuthorityEvidenceID: a.EvidenceID,
			Decision:                             actorContextR3AVDecisionAccept,
			AcceptanceProvenanceRef:              "test-only:r3av/acceptance",
		}},
	); err == nil {
		t.Fatal("R3AV allowed witness to bypass unresolved outcome-acceptance precedence")
	}
}

func TestActorContextR3AVDecisionAndProvenanceMustBeExplicit(t *testing.T) {
	attempt, observation, businessAddress, gate, evidence := actorContextR3AVSingleQualifiedFixture(
		t, actorContextR3ATReportedConflict,
	)
	base := actorContextR3AVOutcomeAcceptanceWitnessProbe{
		ObservationIdentity:                  observation.ObservationIdentity,
		OutcomeAcceptanceAuthorityEvidenceID: evidence[0].EvidenceID,
		Decision:                             actorContextR3AVDecisionAccept,
		AcceptanceProvenanceRef:              "test-only:r3av/acceptance",
	}
	cases := map[string]actorContextR3AVOutcomeAcceptanceWitnessProbe{
		"missing-decision": func() actorContextR3AVOutcomeAcceptanceWitnessProbe {
			x := base
			x.Decision = ""
			return x
		}(),
		"unsupported-decision": func() actorContextR3AVOutcomeAcceptanceWitnessProbe {
			x := base
			x.Decision = "test-only:auto-from-conflict"
			return x
		}(),
		"missing-provenance": func() actorContextR3AVOutcomeAcceptanceWitnessProbe {
			x := base
			x.AcceptanceProvenanceRef = ""
			return x
		}(),
	}
	for name, witness := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := actorContextR3AVBuildCandidates(
				attempt, observation, businessAddress, gate, evidence,
				[]actorContextR3AVOutcomeAcceptanceWitnessProbe{witness},
			); err == nil {
				t.Fatalf("R3AV accepted %s witness", name)
			}
		})
	}
}

func TestActorContextR3AVRawOutcomeDoesNotInferDecision(t *testing.T) {
	for _, outcome := range []string{
		actorContextR3ATReportedApplied,
		actorContextR3ATReportedNotApplied,
		actorContextR3ATReportedConflict,
		actorContextR3ATReportedUnknown,
	} {
		t.Run(outcome, func(t *testing.T) {
			attempt, observation, businessAddress, gate, evidence := actorContextR3AVSingleQualifiedFixture(t, outcome)
			got, err := actorContextR3AVBuildCandidates(
				attempt, observation, businessAddress, gate, evidence, nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 0 {
				t.Fatalf("R3AV inferred decision from raw outcome %q: %+v", outcome, got)
			}
		})
	}
}

func TestActorContextR3AVDuplicateNormalizesAndConflictsFailClosed(t *testing.T) {
	attempt, observation, businessAddress, gate, evidence := actorContextR3AVSingleQualifiedFixture(
		t, actorContextR3ATReportedApplied,
	)
	base := actorContextR3AVOutcomeAcceptanceWitnessProbe{
		ObservationIdentity:                  observation.ObservationIdentity,
		OutcomeAcceptanceAuthorityEvidenceID: evidence[0].EvidenceID,
		Decision:                             actorContextR3AVDecisionAccept,
		AcceptanceProvenanceRef:              "test-only:r3av/acceptance",
	}
	got, err := actorContextR3AVBuildCandidates(
		attempt, observation, businessAddress, gate, evidence,
		[]actorContextR3AVOutcomeAcceptanceWitnessProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AV did not normalize exact duplicate witness: %+v", got)
	}

	otherDecision := base
	otherDecision.Decision = actorContextR3AVDecisionReject
	if _, err := actorContextR3AVBuildCandidates(
		attempt, observation, businessAddress, gate, evidence,
		[]actorContextR3AVOutcomeAcceptanceWitnessProbe{base, otherDecision},
	); err == nil {
		t.Fatal("R3AV silently selected among conflicting acceptance decisions")
	}

	otherProvenance := base
	otherProvenance.AcceptanceProvenanceRef += ":other"
	if _, err := actorContextR3AVBuildCandidates(
		attempt, observation, businessAddress, gate, evidence,
		[]actorContextR3AVOutcomeAcceptanceWitnessProbe{base, otherProvenance},
	); err == nil {
		t.Fatal("R3AV silently aggregated conflicting acceptance provenance")
	}
}

func TestActorContextR3AVInputOrderAndRepeatedBuildAreStable(t *testing.T) {
	attempt, observation, businessAddress, gate, evidence := actorContextR3AVSingleQualifiedFixture(
		t, actorContextR3ATReportedUnknown,
	)
	base := actorContextR3AVOutcomeAcceptanceWitnessProbe{
		ObservationIdentity:                  observation.ObservationIdentity,
		OutcomeAcceptanceAuthorityEvidenceID: evidence[0].EvidenceID,
		Decision:                             actorContextR3AVDecisionReject,
		AcceptanceProvenanceRef:              "test-only:r3av/acceptance",
	}
	inputA := []actorContextR3AVOutcomeAcceptanceWitnessProbe{base, base}
	inputB := []actorContextR3AVOutcomeAcceptanceWitnessProbe{base, base}
	inputB[0], inputB[1] = inputB[1], inputB[0]
	first, err := actorContextR3AVBuildCandidates(attempt, observation, businessAddress, gate, evidence, inputA)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AVBuildCandidates(attempt, observation, businessAddress, gate, evidence, inputB)
	if err != nil {
		t.Fatal(err)
	}
	third, err := actorContextR3AVBuildCandidates(attempt, observation, businessAddress, gate, evidence, inputA)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, third) {
		t.Fatalf("R3AV normalization drifted: first=%+v second=%+v third=%+v", first, second, third)
	}
}

func TestActorContextR3AVShapeHasNoAppliedOutcomeDecisionOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AVOutcomeAcceptanceWitnessProbe{}),
		reflect.TypeOf(actorContextR3AVOutcomeAcceptanceCandidateProbe{}),
	} {
		for _, forbidden := range []string{
			"AcceptedWriteOutcome", "OutcomeAccepted", "AuthoritativeWriteOutcome",
			"AcceptedDecision", "Committed", "CommitSucceeded", "NewDecisionRevision",
			"CurrentAuthority", "AppliedAuthority", "RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "RetrySafe", "Reauthorize",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AV candidate introduced forbidden applied-outcome/retry shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AVSingleQualifiedFixture(
	t *testing.T,
	outcome string,
) (
	actorContextR3ASWriteAttemptCandidateProbe,
	actorContextR3ATNormalizedObservationProbe,
	string,
	actorContextR3AUOutcomeAcceptanceGateProbe,
	[]actorContextR3AUOutcomeAcceptanceEvidenceProbe,
) {
	t.Helper()
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(t, outcome)
	current := actorContextR3AUCompleteEvidence(
		"test-only:r3av/outcome-acceptance-authority",
		businessAddress, attempt, observation,
	)
	evidence := []actorContextR3AUOutcomeAcceptanceEvidenceProbe{current}
	gate, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt, observation, businessAddress, evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusQualified ||
		!reflect.DeepEqual(gate.QualifiedEvidenceIDs, []string{current.EvidenceID}) {
		t.Fatalf("R3AV setup lost single-qualified outcome acceptance authority: %+v", gate)
	}
	return attempt, observation, businessAddress, gate, evidence
}

func actorContextR3AVBuildCandidates(
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	businessAddress string,
	gate actorContextR3AUOutcomeAcceptanceGateProbe,
	evidence []actorContextR3AUOutcomeAcceptanceEvidenceProbe,
	witnesses []actorContextR3AVOutcomeAcceptanceWitnessProbe,
) ([]actorContextR3AVOutcomeAcceptanceCandidateProbe, error) {
	if len(witnesses) == 0 {
		return nil, nil
	}
	if observation.ObservationIdentity == "" ||
		attempt.WriteAttemptCandidateIdentity == "" ||
		observation.WriteAttemptCandidateIdentity != attempt.WriteAttemptCandidateIdentity ||
		businessAddress == "" {
		return nil, fmt.Errorf("R3AV attempt/observation/business context is incomplete")
	}
	if gate.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 {
		return nil, fmt.Errorf("R3AV requires exactly one qualified outcome acceptance authority")
	}
	qualifiedID := gate.QualifiedEvidenceIDs[0]

	var authority *actorContextR3AUOutcomeAcceptanceEvidenceProbe
	for i := range evidence {
		if evidence[i].EvidenceID != qualifiedID {
			continue
		}
		if authority != nil {
			return nil, fmt.Errorf("R3AV duplicate current outcome acceptance evidence %q", qualifiedID)
		}
		authority = &evidence[i]
	}
	if authority == nil ||
		!actorContextR3AUCandidateQualifies(attempt, observation, businessAddress, *authority) {
		return nil, fmt.Errorf("R3AV qualified outcome acceptance evidence missing or no longer qualifies")
	}

	var normalized *actorContextR3AVOutcomeAcceptanceWitnessProbe
	var out actorContextR3AVOutcomeAcceptanceCandidateProbe
	for _, witness := range witnesses {
		if witness.ObservationIdentity == "" ||
			witness.OutcomeAcceptanceAuthorityEvidenceID == "" ||
			witness.Decision == "" ||
			witness.AcceptanceProvenanceRef == "" {
			return nil, fmt.Errorf("R3AV explicit outcome acceptance witness is incomplete")
		}
		if witness.ObservationIdentity != observation.ObservationIdentity {
			return nil, fmt.Errorf("R3AV witness references non-current raw observation")
		}
		if witness.OutcomeAcceptanceAuthorityEvidenceID != qualifiedID {
			return nil, fmt.Errorf("R3AV witness does not bind exact qualified outcome acceptance authority")
		}
		if witness.Decision != actorContextR3AVDecisionAccept &&
			witness.Decision != actorContextR3AVDecisionReject {
			return nil, fmt.Errorf("R3AV unsupported outcome acceptance decision %q", witness.Decision)
		}

		if normalized != nil {
			if !reflect.DeepEqual(*normalized, witness) {
				return nil, fmt.Errorf("R3AV conflicting acceptance witnesses require separate aggregation/precedence authority")
			}
			continue
		}
		copyWitness := witness
		normalized = &copyWitness

		identity := strings.Join([]string{
			observation.ObservationIdentity,
			qualifiedID,
			witness.Decision,
			witness.AcceptanceProvenanceRef,
		}, "||")
		out = actorContextR3AVOutcomeAcceptanceCandidateProbe{
			ObservationIdentity:                  observation.ObservationIdentity,
			WriteAttemptCandidateIdentity:        attempt.WriteAttemptCandidateIdentity,
			RawReportedOutcome:                   observation.ReportedOutcome,
			PostWriteObservedDecisionRevision:    observation.PostWriteObservedDecisionRevision,
			OutcomeAcceptanceAuthorityEvidenceID: qualifiedID,
			OutcomeDecisionLocusRef:              authority.OutcomeDecisionLocusRef,
			OutcomeAuthorityProvenanceRef:        authority.OutcomeAcceptanceProvenanceRef,
			Decision:                             witness.Decision,
			AcceptanceProvenanceRef:              witness.AcceptanceProvenanceRef,
			OutcomeAcceptanceCandidateIdentity:   identity,
			AuthorityLimits: actorContextR0SortedUnique([]string{
				"outcome-acceptance-candidate-does-not-mean-authoritative-outcome-applied",
				"accept-observed-outcome-does-not-commit-accepted-decision",
				"reject-observed-outcome-does-not-mean-write-failed",
				"reject-observed-outcome-does-not-create-retry-authority",
				"post-write-revision-does-not-determine-acceptance-decision",
				"outcome-acceptance-candidate-does-not-apply-current-authority",
			}),
		}
	}
	if normalized == nil {
		return nil, nil
	}
	return []actorContextR3AVOutcomeAcceptanceCandidateProbe{out}, nil
}

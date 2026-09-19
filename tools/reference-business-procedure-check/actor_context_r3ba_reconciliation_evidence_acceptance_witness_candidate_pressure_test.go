package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3BAPlaneExecution    = "execution"
	actorContextR3BAPlaneCurrentState = "current-state"

	actorContextR3BADecisionAccept = "accept-reconciliation-evidence"
	actorContextR3BADecisionReject = "reject-reconciliation-evidence"
)

type actorContextR3BAEvidenceAcceptanceWitnessProbe struct {
	EvidencePlane                       string
	ReconciliationCandidateIdentity     string
	AcceptanceAuthorityEvidenceID       string
	RawEvidenceRef                      string
	ReportedExecutionOutcome            string
	ObservedDecisionRevision            string
	ObservedStateFingerprint            string
	RevisionAuthorityRef                string
	Decision                            string
	EvidenceAcceptanceProvenanceRef     string
}

type actorContextR3BAEvidenceAcceptanceCandidateProbe struct {
	EvidencePlane                       string
	ReconciliationCandidateIdentity     string
	AcceptanceAuthorityEvidenceID       string
	RawEvidenceRef                      string
	ReportedExecutionOutcome            string
	ObservedDecisionRevision            string
	ObservedStateFingerprint            string
	RevisionAuthorityRef                string
	Decision                            string
	EvidenceAcceptanceProvenanceRef     string
	EvidenceAcceptanceCandidateIdentity string
	AuthorityLimits                     []string
}

func TestActorContextR3BAUnresolvedGateCannotCreateCandidate(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	evidence := actorContextR3AZCurrentInspectedEvidence(observations)
	gate, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate, observations, businessAddress, evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	witness := actorContextR3BAExactExecutionWitness(
		candidate,
		observations.ExecutionObservations[0],
		"test-only:r3ba/not-qualified",
		actorContextR3BADecisionAccept,
	)
	if _, err := actorContextR3BABuildCandidates(
		candidate, observations, gate, evidence,
		[]actorContextR3BAEvidenceAcceptanceWitnessProbe{witness},
	); err == nil {
		t.Fatal("R3BA created candidate while R3AZ execution Gate was unresolved")
	}
}

func TestActorContextR3BASingleQualifiedGateDoesNotAutoCreateCandidate(t *testing.T) {
	candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)

	got, err := actorContextR3BABuildCandidates(candidate, observations, gate, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3BA fabricated acceptance candidate without explicit witness: %+v", got)
	}
}

func TestActorContextR3BAExecutionAcceptAndRejectRemainPlaneLocalCandidates(t *testing.T) {
	for _, decision := range []string{actorContextR3BADecisionAccept, actorContextR3BADecisionReject} {
		t.Run(decision, func(t *testing.T) {
			candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)
			witness := actorContextR3BAExactExecutionWitness(
				candidate,
				observations.ExecutionObservations[0],
				gate.QualifiedExecutionEvidenceIDs[0],
				decision,
			)
			got, err := actorContextR3BABuildCandidates(
				candidate, observations, gate, evidence,
				[]actorContextR3BAEvidenceAcceptanceWitnessProbe{witness},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 ||
				got[0].EvidencePlane != actorContextR3BAPlaneExecution ||
				got[0].Decision != decision ||
				got[0].ReportedExecutionOutcome != observations.ExecutionObservations[0].ReportedExecutionOutcome ||
				got[0].ObservedDecisionRevision != "" ||
				got[0].EvidenceAcceptanceCandidateIdentity == "" {
				t.Fatalf("R3BA execution candidate drifted: %+v", got)
			}
		})
	}
}

func TestActorContextR3BACurrentStateAcceptAndRejectRemainPlaneLocalCandidates(t *testing.T) {
	for _, decision := range []string{actorContextR3BADecisionAccept, actorContextR3BADecisionReject} {
		t.Run(decision, func(t *testing.T) {
			candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)
			witness := actorContextR3BAExactCurrentStateWitness(
				candidate,
				observations.CurrentStateObservations[0],
				gate.QualifiedCurrentStateEvidenceIDs[0],
				decision,
			)
			got, err := actorContextR3BABuildCandidates(
				candidate, observations, gate, evidence,
				[]actorContextR3BAEvidenceAcceptanceWitnessProbe{witness},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 ||
				got[0].EvidencePlane != actorContextR3BAPlaneCurrentState ||
				got[0].Decision != decision ||
				got[0].ObservedDecisionRevision != observations.CurrentStateObservations[0].ObservedDecisionRevision ||
				got[0].ObservedStateFingerprint != observations.CurrentStateObservations[0].ObservedStateFingerprint ||
				got[0].ReportedExecutionOutcome != "" ||
				got[0].EvidenceAcceptanceCandidateIdentity == "" {
				t.Fatalf("R3BA current-state candidate drifted: %+v", got)
			}
		})
	}
}

func TestActorContextR3BAWrongPlaneAuthorityOrRawBindingFailsClosed(t *testing.T) {
	candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)

	execution := actorContextR3BAExactExecutionWitness(
		candidate,
		observations.ExecutionObservations[0],
		gate.QualifiedExecutionEvidenceIDs[0],
		actorContextR3BADecisionAccept,
	)
	executionCases := map[string]func(*actorContextR3BAEvidenceAcceptanceWitnessProbe){
		"wrong-plane-authority": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.AcceptanceAuthorityEvidenceID = gate.QualifiedCurrentStateEvidenceIDs[0]
		},
		"wrong-observation": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.RawEvidenceRef += ":other"
		},
		"wrong-outcome": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.ReportedExecutionOutcome = actorContextR3AYReportedConflict
		},
		"smuggled-state-field": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.ObservedDecisionRevision = "test-only:unexpected"
		},
	}
	for name, mutate := range executionCases {
		t.Run("execution-"+name, func(t *testing.T) {
			value := execution
			mutate(&value)
			if _, err := actorContextR3BABuildCandidates(
				candidate, observations, gate, evidence,
				[]actorContextR3BAEvidenceAcceptanceWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3BA accepted invalid execution witness %s", name)
			}
		})
	}

	current := actorContextR3BAExactCurrentStateWitness(
		candidate,
		observations.CurrentStateObservations[0],
		gate.QualifiedCurrentStateEvidenceIDs[0],
		actorContextR3BADecisionAccept,
	)
	currentCases := map[string]func(*actorContextR3BAEvidenceAcceptanceWitnessProbe){
		"wrong-plane-authority": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.AcceptanceAuthorityEvidenceID = gate.QualifiedExecutionEvidenceIDs[0]
		},
		"wrong-observation": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.RawEvidenceRef += ":other"
		},
		"wrong-revision": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.ObservedDecisionRevision += ":other"
		},
		"wrong-fingerprint": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.ObservedStateFingerprint = "sha256:other"
		},
		"wrong-revision-authority": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"smuggled-execution-field": func(v *actorContextR3BAEvidenceAcceptanceWitnessProbe) {
			v.ReportedExecutionOutcome = actorContextR3AYReportedApplied
		},
	}
	for name, mutate := range currentCases {
		t.Run("current-"+name, func(t *testing.T) {
			value := current
			mutate(&value)
			if _, err := actorContextR3BABuildCandidates(
				candidate, observations, gate, evidence,
				[]actorContextR3BAEvidenceAcceptanceWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3BA accepted invalid current-state witness %s", name)
			}
		})
	}
}

func TestActorContextR3BARawEvidenceDoesNotInferDecision(t *testing.T) {
	candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)

	got, err := actorContextR3BABuildCandidates(candidate, observations, gate, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3BA inferred acceptance decision from raw Evidence: %+v", got)
	}
}

func TestActorContextR3BABothPlaneAcceptCandidatesDoNotCreateSynchronization(t *testing.T) {
	candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)
	witnesses := []actorContextR3BAEvidenceAcceptanceWitnessProbe{
		actorContextR3BAExactExecutionWitness(
			candidate, observations.ExecutionObservations[0],
			gate.QualifiedExecutionEvidenceIDs[0], actorContextR3BADecisionAccept,
		),
		actorContextR3BAExactCurrentStateWitness(
			candidate, observations.CurrentStateObservations[0],
			gate.QualifiedCurrentStateEvidenceIDs[0], actorContextR3BADecisionAccept,
		),
	}

	got, err := actorContextR3BABuildCandidates(candidate, observations, gate, evidence, witnesses)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("R3BA lost independent both-plane acceptance candidates: %+v", got)
	}
	if got[0].EvidencePlane == got[1].EvidencePlane {
		t.Fatalf("R3BA collapsed both-plane candidates: %+v", got)
	}
}

func TestActorContextR3BAMultipleQualifiedPlaneCannotBeBypassed(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	executionA := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3ba/execution-a", businessAddress,
		candidate, observations.ExecutionObservations[0],
	)
	executionB := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3ba/execution-b", businessAddress,
		candidate, observations.ExecutionObservations[0],
	)
	current := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3ba/current", businessAddress,
		candidate, observations.CurrentStateObservations[0],
	)
	evidence := []actorContextR3AZEvidenceProbe{executionA, executionB, current}
	gate, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate, observations, businessAddress, evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusMultipleQualified {
		t.Fatalf("R3BA setup lost execution multiple-qualified state: %+v", gate)
	}

	witness := actorContextR3BAExactExecutionWitness(
		candidate, observations.ExecutionObservations[0],
		executionA.EvidenceID, actorContextR3BADecisionAccept,
	)
	if _, err := actorContextR3BABuildCandidates(
		candidate, observations, gate, evidence,
		[]actorContextR3BAEvidenceAcceptanceWitnessProbe{witness},
	); err == nil {
		t.Fatal("R3BA allowed witness to bypass execution acceptance-authority precedence")
	}
}

func TestActorContextR3BADuplicateNormalizesAndConflictsFailClosed(t *testing.T) {
	candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)
	base := actorContextR3BAExactExecutionWitness(
		candidate, observations.ExecutionObservations[0],
		gate.QualifiedExecutionEvidenceIDs[0], actorContextR3BADecisionAccept,
	)

	got, err := actorContextR3BABuildCandidates(
		candidate, observations, gate, evidence,
		[]actorContextR3BAEvidenceAcceptanceWitnessProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3BA did not normalize exact duplicate witness: %+v", got)
	}

	otherDecision := base
	otherDecision.Decision = actorContextR3BADecisionReject
	if _, err := actorContextR3BABuildCandidates(
		candidate, observations, gate, evidence,
		[]actorContextR3BAEvidenceAcceptanceWitnessProbe{base, otherDecision},
	); err == nil {
		t.Fatal("R3BA silently selected among conflicting Evidence acceptance decisions")
	}

	otherProvenance := base
	otherProvenance.EvidenceAcceptanceProvenanceRef += ":other"
	if _, err := actorContextR3BABuildCandidates(
		candidate, observations, gate, evidence,
		[]actorContextR3BAEvidenceAcceptanceWitnessProbe{base, otherProvenance},
	); err == nil {
		t.Fatal("R3BA silently selected among conflicting Evidence acceptance provenance")
	}
}

func TestActorContextR3BAInputOrderDoesNotChangeCandidateIdentity(t *testing.T) {
	candidate, observations, gate, evidence := actorContextR3BASingleQualifiedFixture(t)
	execution := actorContextR3BAExactExecutionWitness(
		candidate, observations.ExecutionObservations[0],
		gate.QualifiedExecutionEvidenceIDs[0], actorContextR3BADecisionAccept,
	)
	current := actorContextR3BAExactCurrentStateWitness(
		candidate, observations.CurrentStateObservations[0],
		gate.QualifiedCurrentStateEvidenceIDs[0], actorContextR3BADecisionReject,
	)

	first, err := actorContextR3BABuildCandidates(
		candidate, observations, gate, evidence,
		[]actorContextR3BAEvidenceAcceptanceWitnessProbe{execution, current},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3BABuildCandidates(
		candidate, observations, gate, evidence,
		[]actorContextR3BAEvidenceAcceptanceWitnessProbe{current, execution},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3BA input order changed candidate identity: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3BAShapeHasNoAppliedAcceptanceSynchronizationOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3BAEvidenceAcceptanceWitnessProbe{}),
		reflect.TypeOf(actorContextR3BAEvidenceAcceptanceCandidateProbe{}),
	} {
		for _, forbidden := range []string{
			"EvidenceAccepted", "AcceptanceApplied",
			"StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedDecision", "CurrentDecision", "CurrentAuthority", "AppliedAuthority",
			"Retry", "RetryEligible", "RetryRequired", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3BA candidate introduced forbidden shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3BAExecutionAuthorityMatchesQualifiedGate(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observation actorContextR3AYRawExecutionObservationProbe,
	authority actorContextR3AZEvidenceProbe,
) bool {
	return authority.EvidenceRole == actorContextR3AZRoleExecutionAuthorityCandidate &&
		authority.RepositoryBacked &&
		!authority.DerivedOnly &&
		authority.AuthorityKind == actorContextR3AZExecutionAuthorityKind &&
		authority.AuthorityDeclarationRef != "" &&
		actorContextR0Contains(
			authority.ReconciliationCandidateIdentities,
			candidate.ReconciliationCandidateIdentity,
		) &&
		actorContextR0Contains(
			authority.ExecutionObservationRefs,
			observation.ReconciliationExecutionObservationRef,
		) &&
		actorContextR0Contains(
			authority.ReportedExecutionOutcomes,
			observation.ReportedExecutionOutcome,
		) &&
		authority.AcceptedDecisionSinkRef == candidate.AcceptedDecisionSinkRef &&
		authority.EvidenceDecisionLocusRef != "" &&
		authority.EvidenceAcceptanceProvenanceRef != ""
}

func actorContextR3BACurrentStateAuthorityMatchesQualifiedGate(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observation actorContextR3AYRawCurrentStateObservationProbe,
	authority actorContextR3AZEvidenceProbe,
) bool {
	return authority.EvidenceRole == actorContextR3AZRoleCurrentStateAuthorityCandidate &&
		authority.RepositoryBacked &&
		!authority.DerivedOnly &&
		authority.AuthorityKind == actorContextR3AZCurrentStateAuthorityKind &&
		authority.AuthorityDeclarationRef != "" &&
		actorContextR0Contains(
			authority.ReconciliationCandidateIdentities,
			candidate.ReconciliationCandidateIdentity,
		) &&
		actorContextR0Contains(
			authority.CurrentStateObservationRefs,
			observation.CurrentStateObservationRef,
		) &&
		actorContextR0Contains(
			authority.ObservedDecisionRevisions,
			observation.ObservedDecisionRevision,
		) &&
		actorContextR0Contains(
			authority.ObservedStateFingerprints,
			observation.ObservedStateFingerprint,
		) &&
		authority.AcceptedDecisionSinkRef == candidate.AcceptedDecisionSinkRef &&
		authority.RevisionAuthorityRef == candidate.RevisionAuthorityRef &&
		authority.EvidenceDecisionLocusRef != "" &&
		authority.EvidenceAcceptanceProvenanceRef != ""
}

func actorContextR3BASingleQualifiedFixture(
	t *testing.T,
) (
	actorContextR3AXReconciliationCandidateProbe,
	actorContextR3AYObservationSetProbe,
	actorContextR3AZGateProbe,
	[]actorContextR3AZEvidenceProbe,
) {
	t.Helper()
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	execution := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3ba/execution-authority", businessAddress,
		candidate, observations.ExecutionObservations[0],
	)
	current := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3ba/current-authority", businessAddress,
		candidate, observations.CurrentStateObservations[0],
	)
	evidence := []actorContextR3AZEvidenceProbe{execution, current}
	gate, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate, observations, businessAddress, evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusQualified ||
		gate.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusQualified {
		t.Fatalf("R3BA setup lost single-qualified R3AZ Gates: %+v", gate)
	}
	return candidate, observations, gate, evidence
}

func actorContextR3BAExactExecutionWitness(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observation actorContextR3AYRawExecutionObservationProbe,
	authorityID string,
	decision string,
) actorContextR3BAEvidenceAcceptanceWitnessProbe {
	return actorContextR3BAEvidenceAcceptanceWitnessProbe{
		EvidencePlane:                   actorContextR3BAPlaneExecution,
		ReconciliationCandidateIdentity: candidate.ReconciliationCandidateIdentity,
		AcceptanceAuthorityEvidenceID:   authorityID,
		RawEvidenceRef:                  observation.ReconciliationExecutionObservationRef,
		ReportedExecutionOutcome:        observation.ReportedExecutionOutcome,
		Decision:                        decision,
		EvidenceAcceptanceProvenanceRef: "test-only:r3ba/execution-acceptance",
	}
}

func actorContextR3BAExactCurrentStateWitness(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observation actorContextR3AYRawCurrentStateObservationProbe,
	authorityID string,
	decision string,
) actorContextR3BAEvidenceAcceptanceWitnessProbe {
	return actorContextR3BAEvidenceAcceptanceWitnessProbe{
		EvidencePlane:                   actorContextR3BAPlaneCurrentState,
		ReconciliationCandidateIdentity: candidate.ReconciliationCandidateIdentity,
		AcceptanceAuthorityEvidenceID:   authorityID,
		RawEvidenceRef:                  observation.CurrentStateObservationRef,
		ObservedDecisionRevision:        observation.ObservedDecisionRevision,
		ObservedStateFingerprint:        observation.ObservedStateFingerprint,
		RevisionAuthorityRef:            observation.RevisionAuthorityRef,
		Decision:                        decision,
		EvidenceAcceptanceProvenanceRef: "test-only:r3ba/current-state-acceptance",
	}
}

func actorContextR3BABuildCandidates(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observations actorContextR3AYObservationSetProbe,
	gate actorContextR3AZGateProbe,
	evidence []actorContextR3AZEvidenceProbe,
	witnesses []actorContextR3BAEvidenceAcceptanceWitnessProbe,
) ([]actorContextR3BAEvidenceAcceptanceCandidateProbe, error) {
	if len(witnesses) == 0 {
		return nil, nil
	}
	if candidate.ReconciliationCandidateIdentity == "" ||
		observations.ReconciliationCandidateIdentity != candidate.ReconciliationCandidateIdentity {
		return nil, fmt.Errorf("R3BA reconciliation candidate/observation context is incomplete")
	}

	byID := map[string]actorContextR3AZEvidenceProbe{}
	for _, value := range evidence {
		if value.EvidenceID == "" {
			return nil, fmt.Errorf("R3BA acceptance authority evidence id is empty")
		}
		if _, exists := byID[value.EvidenceID]; exists {
			return nil, fmt.Errorf("R3BA duplicate acceptance authority evidence id %q", value.EvidenceID)
		}
		byID[value.EvidenceID] = value
	}

	normalized := map[string]actorContextR3BAEvidenceAcceptanceWitnessProbe{}
	out := []actorContextR3BAEvidenceAcceptanceCandidateProbe{}
	for _, witness := range witnesses {
		if witness.ReconciliationCandidateIdentity != candidate.ReconciliationCandidateIdentity ||
			witness.AcceptanceAuthorityEvidenceID == "" ||
			witness.RawEvidenceRef == "" ||
			(witness.Decision != actorContextR3BADecisionAccept &&
				witness.Decision != actorContextR3BADecisionReject) ||
			witness.EvidenceAcceptanceProvenanceRef == "" {
			return nil, fmt.Errorf("R3BA explicit Evidence acceptance witness is incomplete")
		}

		authority, ok := byID[witness.AcceptanceAuthorityEvidenceID]
		if !ok {
			return nil, fmt.Errorf("R3BA witness references unknown acceptance authority")
		}

		switch witness.EvidencePlane {
		case actorContextR3BAPlaneExecution:
			if gate.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusQualified ||
				len(gate.QualifiedExecutionEvidenceIDs) != 1 ||
				gate.QualifiedExecutionEvidenceIDs[0] != witness.AcceptanceAuthorityEvidenceID ||
				witness.ObservedDecisionRevision != "" ||
				witness.ObservedStateFingerprint != "" ||
				witness.RevisionAuthorityRef != "" {
				return nil, fmt.Errorf("R3BA execution witness does not bind single execution Gate")
			}
			var raw *actorContextR3AYRawExecutionObservationProbe
			for i := range observations.ExecutionObservations {
				if observations.ExecutionObservations[i].ReconciliationExecutionObservationRef == witness.RawEvidenceRef {
					raw = &observations.ExecutionObservations[i]
					break
				}
			}
			if raw == nil ||
				raw.ReportedExecutionOutcome != witness.ReportedExecutionOutcome ||
				!actorContextR3BAExecutionAuthorityMatchesQualifiedGate(
					candidate, *raw, authority,
				) {
				return nil, fmt.Errorf("R3BA execution witness does not bind exact raw Evidence/qualified authority")
			}

		case actorContextR3BAPlaneCurrentState:
			if gate.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusQualified ||
				len(gate.QualifiedCurrentStateEvidenceIDs) != 1 ||
				gate.QualifiedCurrentStateEvidenceIDs[0] != witness.AcceptanceAuthorityEvidenceID ||
				witness.ReportedExecutionOutcome != "" {
				return nil, fmt.Errorf("R3BA current-state witness does not bind single current-state Gate")
			}
			var raw *actorContextR3AYRawCurrentStateObservationProbe
			for i := range observations.CurrentStateObservations {
				if observations.CurrentStateObservations[i].CurrentStateObservationRef == witness.RawEvidenceRef {
					raw = &observations.CurrentStateObservations[i]
					break
				}
			}
			if raw == nil ||
				raw.ObservedDecisionRevision != witness.ObservedDecisionRevision ||
				raw.ObservedStateFingerprint != witness.ObservedStateFingerprint ||
				raw.RevisionAuthorityRef != witness.RevisionAuthorityRef ||
				!actorContextR3BACurrentStateAuthorityMatchesQualifiedGate(
					candidate, *raw, authority,
				) {
				return nil, fmt.Errorf("R3BA current-state witness does not bind exact raw Evidence/qualified authority")
			}

		default:
			return nil, fmt.Errorf("R3BA unsupported Evidence plane %q", witness.EvidencePlane)
		}

		conflictKey := strings.Join([]string{
			witness.EvidencePlane,
			witness.RawEvidenceRef,
			witness.AcceptanceAuthorityEvidenceID,
		}, "||")
		if previous, ok := normalized[conflictKey]; ok {
			if !reflect.DeepEqual(previous, witness) {
				return nil, fmt.Errorf("R3BA conflicting Evidence acceptance witnesses require separate precedence authority")
			}
			continue
		}
		normalized[conflictKey] = witness

		identity := strings.Join([]string{
			witness.EvidencePlane,
			candidate.ReconciliationCandidateIdentity,
			witness.AcceptanceAuthorityEvidenceID,
			witness.RawEvidenceRef,
			witness.ReportedExecutionOutcome,
			witness.ObservedDecisionRevision,
			witness.ObservedStateFingerprint,
			witness.RevisionAuthorityRef,
			witness.Decision,
			witness.EvidenceAcceptanceProvenanceRef,
		}, "||")
		out = append(out, actorContextR3BAEvidenceAcceptanceCandidateProbe{
			EvidencePlane:                       witness.EvidencePlane,
			ReconciliationCandidateIdentity:     candidate.ReconciliationCandidateIdentity,
			AcceptanceAuthorityEvidenceID:       witness.AcceptanceAuthorityEvidenceID,
			RawEvidenceRef:                      witness.RawEvidenceRef,
			ReportedExecutionOutcome:            witness.ReportedExecutionOutcome,
			ObservedDecisionRevision:            witness.ObservedDecisionRevision,
			ObservedStateFingerprint:            witness.ObservedStateFingerprint,
			RevisionAuthorityRef:                witness.RevisionAuthorityRef,
			Decision:                            witness.Decision,
			EvidenceAcceptanceProvenanceRef:     witness.EvidenceAcceptanceProvenanceRef,
			EvidenceAcceptanceCandidateIdentity: identity,
			AuthorityLimits: actorContextR0SortedUnique([]string{
				"plane-local-acceptance-candidate-does-not-apply-evidence-authoritatively",
				"both-plane-accept-candidates-do-not-synchronize-authoritative-state",
				"reject-evidence-candidate-does-not-create-retry-rollback-compensation-or-failure",
				"raw-outcome-and-revision-relation-do-not-infer-acceptance-decision",
			}),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].EvidencePlane != out[j].EvidencePlane {
			return out[i].EvidencePlane < out[j].EvidencePlane
		}
		if out[i].RawEvidenceRef != out[j].RawEvidenceRef {
			return out[i].RawEvidenceRef < out[j].RawEvidenceRef
		}
		return out[i].EvidenceAcceptanceCandidateIdentity < out[j].EvidenceAcceptanceCandidateIdentity
	})
	return out, nil
}

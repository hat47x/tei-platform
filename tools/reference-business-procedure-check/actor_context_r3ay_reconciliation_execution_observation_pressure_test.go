package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3AYReportedApplied    = "reconciler-reported-applied"
	actorContextR3AYReportedNotApplied = "reconciler-reported-not-applied"
	actorContextR3AYReportedConflict   = "reconciler-reported-conflict"
	actorContextR3AYReportedUnknown    = "reconciler-reported-unknown"
)

type actorContextR3AYExecutionObservationWitnessProbe struct {
	ReconciliationCandidateIdentity       string
	AcceptedDecisionSinkRef               string
	RevisionAuthorityRef                  string
	ReconciliationExecutionObservationRef string
	ReportedExecutionOutcome              string
}

type actorContextR3AYCurrentStateObservationWitnessProbe struct {
	ReconciliationCandidateIdentity string
	AcceptedDecisionSinkRef         string
	RevisionAuthorityRef            string
	CurrentStateObservationRef      string
	ObservedDecisionRevision        string
	ObservedStateFingerprint        string
}

type actorContextR3AYRawExecutionObservationProbe struct {
	ReconciliationCandidateIdentity       string
	AcceptedDecisionSinkRef               string
	RevisionAuthorityRef                  string
	ReconciliationExecutionObservationRef string
	ReportedExecutionOutcome              string
}

type actorContextR3AYRawCurrentStateObservationProbe struct {
	ReconciliationCandidateIdentity string
	AcceptedDecisionSinkRef         string
	RevisionAuthorityRef            string
	CurrentStateObservationRef      string
	ObservedDecisionRevision        string
	ObservedStateFingerprint        string
}

type actorContextR3AYObservationSetProbe struct {
	ReconciliationCandidateIdentity string
	ExecutionObservations           []actorContextR3AYRawExecutionObservationProbe
	CurrentStateObservations        []actorContextR3AYRawCurrentStateObservationProbe
	AuthorityLimits                 []string
}

func TestActorContextR3AYCandidateAloneDoesNotFabricatePostCandidateEvidence(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)

	got, err := actorContextR3AYObserveReconciliation(candidate, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ExecutionObservations) != 0 ||
		len(got.CurrentStateObservations) != 0 {
		t.Fatalf("R3AY fabricated observations from reconciliation candidate alone: %+v", got)
	}
}

func TestActorContextR3AYExecutionObservationRemainsRawReportedOutcome(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)
	witness := actorContextR3AYExactExecutionWitness(
		candidate,
		actorContextR3AYReportedApplied,
	)

	got, err := actorContextR3AYObserveReconciliation(
		candidate,
		[]actorContextR3AYExecutionObservationWitnessProbe{witness},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ExecutionObservations) != 1 ||
		got.ExecutionObservations[0].ReportedExecutionOutcome != actorContextR3AYReportedApplied ||
		len(got.CurrentStateObservations) != 0 {
		t.Fatalf("R3AY execution observation drifted or fabricated state observation: %+v", got)
	}
}

func TestActorContextR3AYCurrentStateObservationIsIndependentOfExecutionObservation(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)
	witness := actorContextR3AYExactCurrentStateWitness(
		candidate,
		candidate.PostWriteDecisionRevision,
		"sha256:test-only-current-state",
	)

	got, err := actorContextR3AYObserveReconciliation(
		candidate,
		nil,
		[]actorContextR3AYCurrentStateObservationWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ExecutionObservations) != 0 ||
		len(got.CurrentStateObservations) != 1 ||
		got.CurrentStateObservations[0].ObservedDecisionRevision != candidate.PostWriteDecisionRevision {
		t.Fatalf("R3AY current-state observation drifted or fabricated execution evidence: %+v", got)
	}
}

func TestActorContextR3AYReportedOutcomeAndCurrentRevisionDoNotInferSynchronization(t *testing.T) {
	for _, outcome := range []string{
		actorContextR3AYReportedApplied,
		actorContextR3AYReportedNotApplied,
		actorContextR3AYReportedConflict,
		actorContextR3AYReportedUnknown,
	} {
		t.Run(outcome, func(t *testing.T) {
			candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)
			execution := actorContextR3AYExactExecutionWitness(candidate, outcome)
			state := actorContextR3AYExactCurrentStateWitness(
				candidate,
				candidate.PostWriteDecisionRevision,
				"sha256:test-only-current-state",
			)
			got, err := actorContextR3AYObserveReconciliation(
				candidate,
				[]actorContextR3AYExecutionObservationWitnessProbe{execution},
				[]actorContextR3AYCurrentStateObservationWitnessProbe{state},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got.ExecutionObservations) != 1 ||
				len(got.CurrentStateObservations) != 1 {
				t.Fatalf("R3AY lost independent raw evidence: %+v", got)
			}
		})
	}
}

func TestActorContextR3AYRevisionMismatchDoesNotBecomeRetryOrRollbackDecision(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)
	state := actorContextR3AYExactCurrentStateWitness(
		candidate,
		candidate.PostWriteDecisionRevision+":other",
		"sha256:test-only-divergent-current-state",
	)

	got, err := actorContextR3AYObserveReconciliation(
		candidate,
		nil,
		[]actorContextR3AYCurrentStateObservationWitnessProbe{state},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.CurrentStateObservations) != 1 ||
		got.CurrentStateObservations[0].ObservedDecisionRevision == candidate.PostWriteDecisionRevision {
		t.Fatalf("R3AY setup lost divergent current-state revision: %+v", got)
	}
}

func TestActorContextR3AYRejectCandidateDoesNotEnterExecutionObservation(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionReject)
	witness := actorContextR3AYExactExecutionWitness(
		candidate,
		actorContextR3AYReportedNotApplied,
	)

	if _, err := actorContextR3AYObserveReconciliation(
		candidate,
		[]actorContextR3AYExecutionObservationWitnessProbe{witness},
		nil,
	); err == nil {
		t.Fatal("R3AY accepted reconciliation execution observation for reject candidate")
	}
}

func TestActorContextR3AYWrongExactBindingFailsClosed(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)

	executionBase := actorContextR3AYExactExecutionWitness(
		candidate,
		actorContextR3AYReportedApplied,
	)
	executionCases := map[string]func(*actorContextR3AYExecutionObservationWitnessProbe){
		"candidate": func(v *actorContextR3AYExecutionObservationWitnessProbe) {
			v.ReconciliationCandidateIdentity += ":other"
		},
		"sink": func(v *actorContextR3AYExecutionObservationWitnessProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(v *actorContextR3AYExecutionObservationWitnessProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"observation-ref": func(v *actorContextR3AYExecutionObservationWitnessProbe) {
			v.ReconciliationExecutionObservationRef = ""
		},
	}
	for name, mutate := range executionCases {
		t.Run("execution-"+name, func(t *testing.T) {
			value := executionBase
			mutate(&value)
			if _, err := actorContextR3AYObserveReconciliation(
				candidate,
				[]actorContextR3AYExecutionObservationWitnessProbe{value},
				nil,
			); err == nil {
				t.Fatalf("R3AY accepted mismatched execution %s", name)
			}
		})
	}

	stateBase := actorContextR3AYExactCurrentStateWitness(
		candidate,
		candidate.PostWriteDecisionRevision,
		"sha256:test-only-current-state",
	)
	stateCases := map[string]func(*actorContextR3AYCurrentStateObservationWitnessProbe){
		"candidate": func(v *actorContextR3AYCurrentStateObservationWitnessProbe) {
			v.ReconciliationCandidateIdentity += ":other"
		},
		"sink": func(v *actorContextR3AYCurrentStateObservationWitnessProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(v *actorContextR3AYCurrentStateObservationWitnessProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"observation-ref": func(v *actorContextR3AYCurrentStateObservationWitnessProbe) {
			v.CurrentStateObservationRef = ""
		},
		"revision": func(v *actorContextR3AYCurrentStateObservationWitnessProbe) {
			v.ObservedDecisionRevision = ""
		},
		"fingerprint": func(v *actorContextR3AYCurrentStateObservationWitnessProbe) {
			v.ObservedStateFingerprint = ""
		},
	}
	for name, mutate := range stateCases {
		t.Run("state-"+name, func(t *testing.T) {
			value := stateBase
			mutate(&value)
			if _, err := actorContextR3AYObserveReconciliation(
				candidate,
				nil,
				[]actorContextR3AYCurrentStateObservationWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3AY accepted incomplete/mismatched state %s", name)
			}
		})
	}
}

func TestActorContextR3AYDuplicateNormalizesAndSameRefConflictFailsClosed(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)

	execution := actorContextR3AYExactExecutionWitness(candidate, actorContextR3AYReportedApplied)
	state := actorContextR3AYExactCurrentStateWitness(
		candidate,
		candidate.PostWriteDecisionRevision,
		"sha256:test-only-current-state",
	)
	got, err := actorContextR3AYObserveReconciliation(
		candidate,
		[]actorContextR3AYExecutionObservationWitnessProbe{execution, execution},
		[]actorContextR3AYCurrentStateObservationWitnessProbe{state, state},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ExecutionObservations) != 1 ||
		len(got.CurrentStateObservations) != 1 {
		t.Fatalf("R3AY did not normalize exact duplicate observations: %+v", got)
	}

	executionConflict := execution
	executionConflict.ReportedExecutionOutcome = actorContextR3AYReportedConflict
	if _, err := actorContextR3AYObserveReconciliation(
		candidate,
		[]actorContextR3AYExecutionObservationWitnessProbe{execution, executionConflict},
		nil,
	); err == nil {
		t.Fatal("R3AY silently selected among same-ref execution outcome conflict")
	}

	stateConflict := state
	stateConflict.ObservedStateFingerprint = "sha256:test-only-other-state"
	if _, err := actorContextR3AYObserveReconciliation(
		candidate,
		nil,
		[]actorContextR3AYCurrentStateObservationWitnessProbe{state, stateConflict},
	); err == nil {
		t.Fatal("R3AY silently selected among same-ref current-state conflict")
	}
}

func TestActorContextR3AYRepeatedBuildAndInputOrderAreStable(t *testing.T) {
	candidate := actorContextR3AYCandidateFixture(t, actorContextR3AXDecisionAccept)
	executionA := actorContextR3AYExactExecutionWitness(candidate, actorContextR3AYReportedApplied)
	executionB := executionA
	executionB.ReconciliationExecutionObservationRef = "test-only:r3ay/execution-2"
	executionB.ReportedExecutionOutcome = actorContextR3AYReportedUnknown

	stateA := actorContextR3AYExactCurrentStateWitness(
		candidate,
		candidate.PostWriteDecisionRevision,
		"sha256:test-only-current-state",
	)
	stateB := stateA
	stateB.CurrentStateObservationRef = "test-only:r3ay/state-2"
	stateB.ObservedDecisionRevision += ":later"
	stateB.ObservedStateFingerprint = "sha256:test-only-current-state-2"

	first, err := actorContextR3AYObserveReconciliation(
		candidate,
		[]actorContextR3AYExecutionObservationWitnessProbe{executionA, executionB},
		[]actorContextR3AYCurrentStateObservationWitnessProbe{stateA, stateB},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AYObserveReconciliation(
		candidate,
		[]actorContextR3AYExecutionObservationWitnessProbe{executionB, executionA},
		[]actorContextR3AYCurrentStateObservationWitnessProbe{stateB, stateA},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AY input order changed normalized raw evidence: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3AYShapeHasNoSynchronizationAuthorityOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AYObservationSetProbe{}),
		reflect.TypeOf(actorContextR3AYRawExecutionObservationProbe{}),
		reflect.TypeOf(actorContextR3AYRawCurrentStateObservationProbe{}),
	} {
		for _, forbidden := range []string{
			"StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedWriteOutcome", "OutcomeAccepted", "AcceptedDecision", "CurrentDecision",
			"CurrentAuthority", "AppliedAuthority", "RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AY raw observation shape introduced forbidden shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AYCandidateFixture(
	t *testing.T,
	decision string,
) actorContextR3AXReconciliationCandidateProbe {
	t.Helper()

	commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
		actorContextR3AXSingleQualifiedFixture(
			t,
			actorContextR3ATReportedApplied,
			true,
		)
	witness := actorContextR3AXExactWitness(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		evidence[0],
		decision,
	)
	candidates, err := actorContextR3AXBuildCandidates(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AXReconciliationWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("R3AY setup reconciliation candidate count=%d want=1", len(candidates))
	}
	return candidates[0]
}

func actorContextR3AYExactExecutionWitness(
	candidate actorContextR3AXReconciliationCandidateProbe,
	outcome string,
) actorContextR3AYExecutionObservationWitnessProbe {
	return actorContextR3AYExecutionObservationWitnessProbe{
		ReconciliationCandidateIdentity:       candidate.ReconciliationCandidateIdentity,
		AcceptedDecisionSinkRef:               candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:                  candidate.RevisionAuthorityRef,
		ReconciliationExecutionObservationRef: "test-only:r3ay/execution-1",
		ReportedExecutionOutcome:              outcome,
	}
}

func actorContextR3AYExactCurrentStateWitness(
	candidate actorContextR3AXReconciliationCandidateProbe,
	revision string,
	fingerprint string,
) actorContextR3AYCurrentStateObservationWitnessProbe {
	return actorContextR3AYCurrentStateObservationWitnessProbe{
		ReconciliationCandidateIdentity: candidate.ReconciliationCandidateIdentity,
		AcceptedDecisionSinkRef:         candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:            candidate.RevisionAuthorityRef,
		CurrentStateObservationRef:      "test-only:r3ay/current-state-1",
		ObservedDecisionRevision:        revision,
		ObservedStateFingerprint:        fingerprint,
	}
}

func actorContextR3AYObserveReconciliation(
	candidate actorContextR3AXReconciliationCandidateProbe,
	executionWitnesses []actorContextR3AYExecutionObservationWitnessProbe,
	stateWitnesses []actorContextR3AYCurrentStateObservationWitnessProbe,
) (actorContextR3AYObservationSetProbe, error) {
	if candidate.ReconciliationCandidateIdentity == "" ||
		candidate.AcceptedDecisionSinkRef == "" ||
		candidate.RevisionAuthorityRef == "" {
		return actorContextR3AYObservationSetProbe{}, fmt.Errorf(
			"R3AY reconciliation candidate identity/sink/revision authority is incomplete",
		)
	}

	if len(executionWitnesses) > 0 &&
		candidate.Decision != actorContextR3AXDecisionAccept {
		return actorContextR3AYObservationSetProbe{}, fmt.Errorf(
			"R3AY only accept reconciliation candidates may have execution observations",
		)
	}

	executions := map[string]actorContextR3AYRawExecutionObservationProbe{}
	for _, witness := range executionWitnesses {
		if witness.ReconciliationCandidateIdentity != candidate.ReconciliationCandidateIdentity ||
			witness.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
			witness.RevisionAuthorityRef != candidate.RevisionAuthorityRef ||
			witness.ReconciliationExecutionObservationRef == "" ||
			!actorContextR3AYValidReportedOutcome(witness.ReportedExecutionOutcome) {
			return actorContextR3AYObservationSetProbe{}, fmt.Errorf(
				"R3AY execution observation does not bind exact candidate or has invalid raw outcome",
			)
		}
		raw := actorContextR3AYRawExecutionObservationProbe(witness)
		if existing, ok := executions[witness.ReconciliationExecutionObservationRef]; ok {
			if !reflect.DeepEqual(existing, raw) {
				return actorContextR3AYObservationSetProbe{}, fmt.Errorf(
					"R3AY conflicting execution observations share ref %q",
					witness.ReconciliationExecutionObservationRef,
				)
			}
			continue
		}
		executions[witness.ReconciliationExecutionObservationRef] = raw
	}

	states := map[string]actorContextR3AYRawCurrentStateObservationProbe{}
	for _, witness := range stateWitnesses {
		if witness.ReconciliationCandidateIdentity != candidate.ReconciliationCandidateIdentity ||
			witness.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
			witness.RevisionAuthorityRef != candidate.RevisionAuthorityRef ||
			witness.CurrentStateObservationRef == "" ||
			witness.ObservedDecisionRevision == "" ||
			witness.ObservedStateFingerprint == "" {
			return actorContextR3AYObservationSetProbe{}, fmt.Errorf(
				"R3AY current-state observation does not bind exact candidate or is incomplete",
			)
		}
		raw := actorContextR3AYRawCurrentStateObservationProbe(witness)
		if existing, ok := states[witness.CurrentStateObservationRef]; ok {
			if !reflect.DeepEqual(existing, raw) {
				return actorContextR3AYObservationSetProbe{}, fmt.Errorf(
					"R3AY conflicting current-state observations share ref %q",
					witness.CurrentStateObservationRef,
				)
			}
			continue
		}
		states[witness.CurrentStateObservationRef] = raw
	}

	out := actorContextR3AYObservationSetProbe{
		ReconciliationCandidateIdentity: candidate.ReconciliationCandidateIdentity,
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"execution-observation-does-not-prove-authoritative-state-synchronized",
			"current-state-observation-does-not-prove-reconciliation-executed",
			"matching-revision-does-not-accept-or-apply-revision",
			"divergent-revision-does-not-authorize-retry-rollback-or-compensation",
			"reported-execution-outcome-remains-raw-attributed-evidence",
			"observed-state-fingerprint-is-not-an-accepted-decision-fingerprint",
		}),
	}
	for _, value := range executions {
		out.ExecutionObservations = append(out.ExecutionObservations, value)
	}
	sort.Slice(out.ExecutionObservations, func(i, j int) bool {
		return out.ExecutionObservations[i].ReconciliationExecutionObservationRef <
			out.ExecutionObservations[j].ReconciliationExecutionObservationRef
	})
	for _, value := range states {
		out.CurrentStateObservations = append(out.CurrentStateObservations, value)
	}
	sort.Slice(out.CurrentStateObservations, func(i, j int) bool {
		return out.CurrentStateObservations[i].CurrentStateObservationRef <
			out.CurrentStateObservations[j].CurrentStateObservationRef
	})
	return out, nil
}

func actorContextR3AYValidReportedOutcome(value string) bool {
	switch value {
	case actorContextR3AYReportedApplied,
		actorContextR3AYReportedNotApplied,
		actorContextR3AYReportedConflict,
		actorContextR3AYReportedUnknown:
		return true
	default:
		return false
	}
}

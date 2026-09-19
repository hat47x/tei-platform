package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const (
	actorContextR3AXDecisionAccept = "accept-post-write-state-reconciliation"
	actorContextR3AXDecisionReject = "reject-post-write-state-reconciliation"
)

type actorContextR3AXReconciliationWitnessProbe struct {
	OutcomeAcceptanceCandidateIdentity string
	ReconciliationAuthorityEvidenceID  string
	AcceptedDecisionSinkRef            string
	RevisionAuthorityRef               string
	PreWriteDecisionRevision           string
	PostWriteRevisionObservationRef    string
	PostWriteDecisionRevision          string
	Decision                           string
	ReconciliationProvenanceRef        string
}

type actorContextR3AXReconciliationCandidateProbe struct {
	OutcomeAcceptanceCandidateIdentity string
	ObservationIdentity                string
	WriteAttemptCandidateIdentity      string
	CommitCandidateIdentity            string
	ReconciliationAuthorityEvidenceID  string
	AcceptedDecisionSinkRef            string
	RevisionAuthorityRef               string
	PreWriteDecisionRevision           string
	PostWriteRevisionObservationRef    string
	PostWriteDecisionRevision          string
	ReconciliationLocusRef             string
	RevisionAcceptanceLocusRef         string
	AuthorityApplicationLocusRef       string
	Decision                           string
	ReconciliationProvenanceRef        string
	ReconciliationCandidateIdentity    string
	AuthorityLimits                    []string
}

func TestActorContextR3AXUnresolvedR3AWGateCannotCreateCandidate(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		true,
	)
	gate, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		actorContextR3AWCurrentInspectedEvidence(
			commitCandidate,
			attempt,
			observation,
			outcomeCandidate,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ReconciliationAuthorityStatus != actorContextR3AWStatusUnresolved {
		t.Fatalf("R3AX setup expected unresolved R3AW Gate: %+v", gate)
	}

	if _, err := actorContextR3AXBuildCandidates(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		gate,
		nil,
		[]actorContextR3AXReconciliationWitnessProbe{{
			OutcomeAcceptanceCandidateIdentity: outcomeCandidate.OutcomeAcceptanceCandidateIdentity,
			ReconciliationAuthorityEvidenceID:  "test-only:r3ax/not-qualified",
			AcceptedDecisionSinkRef:            attempt.AcceptedDecisionSinkRef,
			RevisionAuthorityRef:               commitCandidate.WriteAuthorityRevisionAuthorityRef,
			PreWriteDecisionRevision:           attempt.ObservedDecisionRevision,
			PostWriteRevisionObservationRef:    observation.PostWriteRevisionObservationRef,
			PostWriteDecisionRevision:          observation.PostWriteObservedDecisionRevision,
			Decision:                           actorContextR3AXDecisionAccept,
			ReconciliationProvenanceRef:        "test-only:r3ax/reconciliation",
		}},
	); err == nil {
		t.Fatal("R3AX created reconciliation candidate while R3AW authority was unresolved")
	}
}

func TestActorContextR3AXSingleQualifiedGateDoesNotAutoCreateCandidate(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
		actorContextR3AXSingleQualifiedFixture(t, actorContextR3ATReportedApplied, false)

	got, err := actorContextR3AXBuildCandidates(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		gate,
		evidence,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3AX fabricated reconciliation candidate without explicit witness: %+v", got)
	}
}

func TestActorContextR3AXExplicitAcceptAndRejectRemainCandidateDecisions(t *testing.T) {
	for _, decision := range []string{
		actorContextR3AXDecisionAccept,
		actorContextR3AXDecisionReject,
	} {
		t.Run(decision, func(t *testing.T) {
			commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
				actorContextR3AXSingleQualifiedFixture(t, actorContextR3ATReportedApplied, true)

			witness := actorContextR3AXExactWitness(
				commitCandidate,
				attempt,
				observation,
				outcomeCandidate,
				evidence[0],
				decision,
			)
			got, err := actorContextR3AXBuildCandidates(
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
			if len(got) != 1 ||
				got[0].Decision != decision ||
				got[0].OutcomeAcceptanceCandidateIdentity != outcomeCandidate.OutcomeAcceptanceCandidateIdentity ||
				got[0].PostWriteDecisionRevision != observation.PostWriteObservedDecisionRevision ||
				got[0].ReconciliationCandidateIdentity == "" {
				t.Fatalf("R3AX reconciliation decision candidate drifted: %+v", got)
			}
		})
	}
}

func TestActorContextR3AXWrongExactBindingFailsClosed(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
		actorContextR3AXSingleQualifiedFixture(t, actorContextR3ATReportedApplied, true)
	base := actorContextR3AXExactWitness(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		evidence[0],
		actorContextR3AXDecisionAccept,
	)

	cases := map[string]func(*actorContextR3AXReconciliationWitnessProbe){
		"outcome-candidate": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.OutcomeAcceptanceCandidateIdentity += ":other"
		},
		"authority": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.ReconciliationAuthorityEvidenceID += ":other"
		},
		"sink": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"pre-revision": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.PreWriteDecisionRevision += ":other"
		},
		"post-observation": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.PostWriteRevisionObservationRef += ":other"
		},
		"post-revision": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.PostWriteDecisionRevision += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			witness := base
			mutate(&witness)
			if _, err := actorContextR3AXBuildCandidates(
				commitCandidate,
				attempt,
				observation,
				outcomeCandidate,
				businessAddress,
				gate,
				evidence,
				[]actorContextR3AXReconciliationWitnessProbe{witness},
			); err == nil {
				t.Fatalf("R3AX accepted mismatched %s witness", name)
			}
		})
	}
}

func TestActorContextR3AXDecisionAndProvenanceMustBeExplicit(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
		actorContextR3AXSingleQualifiedFixture(t, actorContextR3ATReportedConflict, true)
	base := actorContextR3AXExactWitness(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		evidence[0],
		actorContextR3AXDecisionAccept,
	)

	cases := map[string]func(*actorContextR3AXReconciliationWitnessProbe){
		"missing-decision": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.Decision = ""
		},
		"unsupported-decision": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.Decision = "test-only:auto-from-revision"
		},
		"missing-provenance": func(v *actorContextR3AXReconciliationWitnessProbe) {
			v.ReconciliationProvenanceRef = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			witness := base
			mutate(&witness)
			if _, err := actorContextR3AXBuildCandidates(
				commitCandidate,
				attempt,
				observation,
				outcomeCandidate,
				businessAddress,
				gate,
				evidence,
				[]actorContextR3AXReconciliationWitnessProbe{witness},
			); err == nil {
				t.Fatalf("R3AX accepted %s witness", name)
			}
		})
	}
}

func TestActorContextR3AXRevisionRelationDoesNotInferDecision(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprintf("changed=%t", changed), func(t *testing.T) {
			commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
				actorContextR3AXSingleQualifiedFixture(t, actorContextR3ATReportedApplied, changed)
			got, err := actorContextR3AXBuildCandidates(
				commitCandidate,
				attempt,
				observation,
				outcomeCandidate,
				businessAddress,
				gate,
				evidence,
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 0 {
				t.Fatalf("R3AX inferred decision from revision relation changed=%t: %+v", changed, got)
			}
		})
	}
}

func TestActorContextR3AXRawWriterOutcomeDoesNotInferDecision(t *testing.T) {
	for _, outcome := range []string{
		actorContextR3ATReportedApplied,
		actorContextR3ATReportedNotApplied,
		actorContextR3ATReportedConflict,
		actorContextR3ATReportedUnknown,
	} {
		t.Run(outcome, func(t *testing.T) {
			commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
				actorContextR3AXSingleQualifiedFixture(t, outcome, true)
			got, err := actorContextR3AXBuildCandidates(
				commitCandidate,
				attempt,
				observation,
				outcomeCandidate,
				businessAddress,
				gate,
				evidence,
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 0 {
				t.Fatalf("R3AX inferred reconciliation decision from raw writer outcome %q: %+v", outcome, got)
			}
		})
	}
}

func TestActorContextR3AXMultipleQualifiedGateCannotBeBypassed(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		true,
	)
	a := actorContextR3AWCompleteEvidence(
		"test-only:r3ax/a",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)
	b := actorContextR3AWCompleteEvidence(
		"test-only:r3ax/b",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)
	evidence := []actorContextR3AWReconciliationAuthorityEvidenceProbe{a, b}
	gate, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ReconciliationAuthorityStatus != actorContextR3AWStatusMultipleQualified {
		t.Fatalf("R3AX setup lost multiple-qualified R3AW state: %+v", gate)
	}

	witness := actorContextR3AXExactWitness(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		a,
		actorContextR3AXDecisionAccept,
	)
	if _, err := actorContextR3AXBuildCandidates(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AXReconciliationWitnessProbe{witness},
	); err == nil {
		t.Fatal("R3AX allowed witness to bypass unresolved reconciliation-authority precedence")
	}
}

func TestActorContextR3AXDuplicateNormalizesAndConflictsFailClosed(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
		actorContextR3AXSingleQualifiedFixture(t, actorContextR3ATReportedUnknown, true)
	base := actorContextR3AXExactWitness(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		evidence[0],
		actorContextR3AXDecisionAccept,
	)

	got, err := actorContextR3AXBuildCandidates(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AXReconciliationWitnessProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AX did not normalize exact duplicate witness: %+v", got)
	}

	otherDecision := base
	otherDecision.Decision = actorContextR3AXDecisionReject
	if _, err := actorContextR3AXBuildCandidates(
		commitCandidate, attempt, observation, outcomeCandidate, businessAddress,
		gate, evidence,
		[]actorContextR3AXReconciliationWitnessProbe{base, otherDecision},
	); err == nil {
		t.Fatal("R3AX silently selected among conflicting reconciliation decisions")
	}

	otherProvenance := base
	otherProvenance.ReconciliationProvenanceRef += ":other"
	if _, err := actorContextR3AXBuildCandidates(
		commitCandidate, attempt, observation, outcomeCandidate, businessAddress,
		gate, evidence,
		[]actorContextR3AXReconciliationWitnessProbe{base, otherProvenance},
	); err == nil {
		t.Fatal("R3AX silently aggregated conflicting reconciliation provenance")
	}
}

func TestActorContextR3AXRepeatedBuildAndInputOrderAreStable(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence :=
		actorContextR3AXSingleQualifiedFixture(t, actorContextR3ATReportedApplied, true)
	base := actorContextR3AXExactWitness(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		evidence[0],
		actorContextR3AXDecisionReject,
	)
	inputA := []actorContextR3AXReconciliationWitnessProbe{base, base}
	inputB := []actorContextR3AXReconciliationWitnessProbe{base, base}
	inputB[0], inputB[1] = inputB[1], inputB[0]

	first, err := actorContextR3AXBuildCandidates(
		commitCandidate, attempt, observation, outcomeCandidate, businessAddress,
		gate, evidence, inputA,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AXBuildCandidates(
		commitCandidate, attempt, observation, outcomeCandidate, businessAddress,
		gate, evidence, inputB,
	)
	if err != nil {
		t.Fatal(err)
	}
	third, err := actorContextR3AXBuildCandidates(
		commitCandidate, attempt, observation, outcomeCandidate, businessAddress,
		gate, evidence, inputA,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, third) {
		t.Fatalf("R3AX normalization drifted: first=%+v second=%+v third=%+v", first, second, third)
	}
}

func TestActorContextR3AXShapeHasNoSynchronizedDecisionAuthorityOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AXReconciliationWitnessProbe{}),
		reflect.TypeOf(actorContextR3AXReconciliationCandidateProbe{}),
	} {
		for _, forbidden := range []string{
			"StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedWriteOutcome", "OutcomeAccepted", "AcceptedDecision", "CommittedDecision",
			"CurrentAuthority", "AppliedAuthority", "RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AX candidate introduced forbidden synchronization/authority shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AXSingleQualifiedFixture(
	t *testing.T,
	rawOutcome string,
	postRevisionChanged bool,
) (
	actorContextR3AQCommitCandidateProbe,
	actorContextR3ASWriteAttemptCandidateProbe,
	actorContextR3ATNormalizedObservationProbe,
	actorContextR3AVOutcomeAcceptanceCandidateProbe,
	string,
	actorContextR3AWReconciliationAuthorityGateProbe,
	[]actorContextR3AWReconciliationAuthorityEvidenceProbe,
) {
	t.Helper()

	commitCandidate, attempt, observation, outcomeCandidate, businessAddress :=
		actorContextR3AWOutcomeCandidateFixture(
			t,
			rawOutcome,
			actorContextR3AVDecisionAccept,
			postRevisionChanged,
		)
	current := actorContextR3AWCompleteEvidence(
		"test-only:r3ax/reconciliation-authority",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)
	evidence := []actorContextR3AWReconciliationAuthorityEvidenceProbe{current}
	gate, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ReconciliationAuthorityStatus != actorContextR3AWStatusQualified ||
		!reflect.DeepEqual(gate.QualifiedEvidenceIDs, []string{current.EvidenceID}) {
		t.Fatalf("R3AX setup lost single-qualified R3AW authority: %+v", gate)
	}
	return commitCandidate, attempt, observation, outcomeCandidate, businessAddress, gate, evidence
}

func actorContextR3AXExactWitness(
	commitCandidate actorContextR3AQCommitCandidateProbe,
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	outcomeCandidate actorContextR3AVOutcomeAcceptanceCandidateProbe,
	authority actorContextR3AWReconciliationAuthorityEvidenceProbe,
	decision string,
) actorContextR3AXReconciliationWitnessProbe {
	return actorContextR3AXReconciliationWitnessProbe{
		OutcomeAcceptanceCandidateIdentity: outcomeCandidate.OutcomeAcceptanceCandidateIdentity,
		ReconciliationAuthorityEvidenceID:  authority.EvidenceID,
		AcceptedDecisionSinkRef:            attempt.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:               commitCandidate.WriteAuthorityRevisionAuthorityRef,
		PreWriteDecisionRevision:           attempt.ObservedDecisionRevision,
		PostWriteRevisionObservationRef:    observation.PostWriteRevisionObservationRef,
		PostWriteDecisionRevision:          observation.PostWriteObservedDecisionRevision,
		Decision:                           decision,
		ReconciliationProvenanceRef:        "test-only:r3ax/reconciliation-witness",
	}
}

func actorContextR3AXBuildCandidates(
	commitCandidate actorContextR3AQCommitCandidateProbe,
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	outcomeCandidate actorContextR3AVOutcomeAcceptanceCandidateProbe,
	businessAddress string,
	gate actorContextR3AWReconciliationAuthorityGateProbe,
	evidence []actorContextR3AWReconciliationAuthorityEvidenceProbe,
	witnesses []actorContextR3AXReconciliationWitnessProbe,
) ([]actorContextR3AXReconciliationCandidateProbe, error) {
	if len(witnesses) == 0 {
		return nil, nil
	}
	if outcomeCandidate.OutcomeAcceptanceCandidateIdentity == "" ||
		outcomeCandidate.Decision != actorContextR3AVDecisionAccept ||
		observation.ObservationIdentity == "" ||
		attempt.WriteAttemptCandidateIdentity == "" ||
		commitCandidate.CommitCandidateIdentity == "" ||
		businessAddress == "" {
		return nil, fmt.Errorf("R3AX reconciliation chain is incomplete")
	}
	if gate.ReconciliationAuthorityStatus != actorContextR3AWStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 {
		return nil, fmt.Errorf("R3AX requires exactly one qualified reconciliation authority")
	}
	qualifiedID := gate.QualifiedEvidenceIDs[0]

	var authority *actorContextR3AWReconciliationAuthorityEvidenceProbe
	for i := range evidence {
		if evidence[i].EvidenceID != qualifiedID {
			continue
		}
		if authority != nil {
			return nil, fmt.Errorf("R3AX duplicate qualified reconciliation authority evidence %q", qualifiedID)
		}
		authority = &evidence[i]
	}
	if authority == nil ||
		!actorContextR3AWCandidateQualifies(
			commitCandidate,
			attempt,
			observation,
			outcomeCandidate,
			businessAddress,
			*authority,
		) {
		return nil, fmt.Errorf("R3AX qualified reconciliation authority is missing or no longer qualifies")
	}

	var normalized *actorContextR3AXReconciliationWitnessProbe
	var out actorContextR3AXReconciliationCandidateProbe
	for _, witness := range witnesses {
		if witness.OutcomeAcceptanceCandidateIdentity == "" ||
			witness.ReconciliationAuthorityEvidenceID == "" ||
			witness.AcceptedDecisionSinkRef == "" ||
			witness.RevisionAuthorityRef == "" ||
			witness.PreWriteDecisionRevision == "" ||
			witness.PostWriteRevisionObservationRef == "" ||
			witness.PostWriteDecisionRevision == "" ||
			witness.Decision == "" ||
			witness.ReconciliationProvenanceRef == "" {
			return nil, fmt.Errorf("R3AX explicit reconciliation witness is incomplete")
		}
		if witness.OutcomeAcceptanceCandidateIdentity != outcomeCandidate.OutcomeAcceptanceCandidateIdentity ||
			witness.ReconciliationAuthorityEvidenceID != qualifiedID ||
			witness.AcceptedDecisionSinkRef != attempt.AcceptedDecisionSinkRef ||
			witness.RevisionAuthorityRef != commitCandidate.WriteAuthorityRevisionAuthorityRef ||
			witness.PreWriteDecisionRevision != attempt.ObservedDecisionRevision ||
			witness.PostWriteRevisionObservationRef != observation.PostWriteRevisionObservationRef ||
			witness.PostWriteDecisionRevision != observation.PostWriteObservedDecisionRevision {
			return nil, fmt.Errorf("R3AX reconciliation witness does not bind exact current reconciliation chain")
		}
		if witness.Decision != actorContextR3AXDecisionAccept &&
			witness.Decision != actorContextR3AXDecisionReject {
			return nil, fmt.Errorf("R3AX unsupported reconciliation decision %q", witness.Decision)
		}

		if normalized != nil {
			if !reflect.DeepEqual(*normalized, witness) {
				return nil, fmt.Errorf("R3AX conflicting reconciliation witnesses require separate aggregation/precedence authority")
			}
			continue
		}
		copyWitness := witness
		normalized = &copyWitness

		identity := strings.Join([]string{
			outcomeCandidate.OutcomeAcceptanceCandidateIdentity,
			qualifiedID,
			observation.PostWriteObservedDecisionRevision,
			witness.Decision,
			witness.ReconciliationProvenanceRef,
		}, "||")
		out = actorContextR3AXReconciliationCandidateProbe{
			OutcomeAcceptanceCandidateIdentity: outcomeCandidate.OutcomeAcceptanceCandidateIdentity,
			ObservationIdentity:                observation.ObservationIdentity,
			WriteAttemptCandidateIdentity:      attempt.WriteAttemptCandidateIdentity,
			CommitCandidateIdentity:            commitCandidate.CommitCandidateIdentity,
			ReconciliationAuthorityEvidenceID:  qualifiedID,
			AcceptedDecisionSinkRef:            attempt.AcceptedDecisionSinkRef,
			RevisionAuthorityRef:               commitCandidate.WriteAuthorityRevisionAuthorityRef,
			PreWriteDecisionRevision:           attempt.ObservedDecisionRevision,
			PostWriteRevisionObservationRef:    observation.PostWriteRevisionObservationRef,
			PostWriteDecisionRevision:          observation.PostWriteObservedDecisionRevision,
			ReconciliationLocusRef:             authority.ReconciliationLocusRef,
			RevisionAcceptanceLocusRef:         authority.RevisionAcceptanceLocusRef,
			AuthorityApplicationLocusRef:       authority.AuthorityApplicationLocusRef,
			Decision:                           witness.Decision,
			ReconciliationProvenanceRef:        witness.ReconciliationProvenanceRef,
			ReconciliationCandidateIdentity:    identity,
			AuthorityLimits: actorContextR0SortedUnique([]string{
				"reconciliation-candidate-does-not-synchronize-authoritative-state",
				"reconciliation-candidate-does-not-accept-or-apply-revision",
				"reconciliation-candidate-does-not-apply-current-authority",
				"reject-reconciliation-does-not-create-rollback-retry-or-write-failure",
				"revision-relation-and-raw-outcome-do-not-infer-reconciliation-decision",
			}),
		}
	}
	if normalized == nil {
		return nil, nil
	}
	return []actorContextR3AXReconciliationCandidateProbe{out}, nil
}

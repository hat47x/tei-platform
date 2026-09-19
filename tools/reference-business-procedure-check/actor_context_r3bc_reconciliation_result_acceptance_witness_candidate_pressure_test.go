package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3BCDecisionAccept = "accept-reconciliation-result"
	actorContextR3BCDecisionReject = "reject-reconciliation-result"
)

type actorContextR3BCResultAcceptanceWitnessProbe struct {
	ReconciliationCandidateIdentity                 string
	ResultAcceptanceAuthorityEvidenceID             string
	ExecutionEvidenceAcceptanceCandidateIdentity    string
	CurrentStateEvidenceAcceptanceCandidateIdentity string
	ReconciliationResultDecisionLocusRef            string
	CurrentStateAcceptanceLocusRef                  string
	Decision                                        string
	ResultAcceptanceProvenanceRef                   string
}

type actorContextR3BCResultAcceptanceCandidateProbe struct {
	ReconciliationCandidateIdentity                 string
	ResultAcceptanceAuthorityEvidenceID             string
	ExecutionEvidenceAcceptanceCandidateIdentity    string
	CurrentStateEvidenceAcceptanceCandidateIdentity string
	ReconciliationResultDecisionLocusRef            string
	CurrentStateAcceptanceLocusRef                  string
	ObservedDecisionRevision                        string
	ObservedStateFingerprint                        string
	Decision                                        string
	ResultAcceptanceProvenanceRef                   string
	ReconciliationResultAcceptanceCandidateIdentity string
	AuthorityLimits                                 []string
}

func TestActorContextR3BCUnresolvedGateCannotCreateCandidate(t *testing.T) {
	reconciliation, execution, current, businessAddress :=
		actorContextR3BBBothAcceptCandidatesFixture(t)
	inspected := actorContextR3BBCurrentInspectedEvidence(execution, current)
	gate, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
		businessAddress,
		inspected,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusUnresolved {
		t.Fatalf("R3BC setup expected unresolved R3BB Gate: %+v", gate)
	}
	witness := actorContextR3BCResultAcceptanceWitnessProbe{
		ReconciliationCandidateIdentity:                 reconciliation.ReconciliationCandidateIdentity,
		ResultAcceptanceAuthorityEvidenceID:             "test-only:r3bc/not-qualified",
		ExecutionEvidenceAcceptanceCandidateIdentity:    execution.EvidenceAcceptanceCandidateIdentity,
		CurrentStateEvidenceAcceptanceCandidateIdentity: current.EvidenceAcceptanceCandidateIdentity,
		ReconciliationResultDecisionLocusRef:            "test-only:r3bc/result-locus",
		CurrentStateAcceptanceLocusRef:                  "test-only:r3bc/current-state-locus",
		Decision:                                        actorContextR3BCDecisionAccept,
		ResultAcceptanceProvenanceRef:                   "test-only:r3bc/provenance",
	}
	if _, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, inspected,
		[]actorContextR3BCResultAcceptanceWitnessProbe{witness},
	); err == nil {
		t.Fatal("R3BC created candidate while R3BB authority was unresolved")
	}
}

func TestActorContextR3BCSingleQualifiedGateDoesNotAutoCreateCandidate(t *testing.T) {
	reconciliation, execution, current, gate, evidence :=
		actorContextR3BCSingleQualifiedFixture(t)

	got, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3BC fabricated result acceptance candidate without witness: %+v", got)
	}
}

func TestActorContextR3BCExplicitAcceptAndRejectRemainCandidates(t *testing.T) {
	for _, decision := range []string{
		actorContextR3BCDecisionAccept,
		actorContextR3BCDecisionReject,
	} {
		t.Run(decision, func(t *testing.T) {
			reconciliation, execution, current, gate, evidence :=
				actorContextR3BCSingleQualifiedFixture(t)
			witness := actorContextR3BCExactWitness(
				reconciliation, execution, current, gate, decision,
			)
			got, err := actorContextR3BCBuildCandidates(
				reconciliation, execution, current, gate, evidence,
				[]actorContextR3BCResultAcceptanceWitnessProbe{witness},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 ||
				got[0].Decision != decision ||
				got[0].ReconciliationCandidateIdentity != reconciliation.ReconciliationCandidateIdentity ||
				got[0].ExecutionEvidenceAcceptanceCandidateIdentity != execution.EvidenceAcceptanceCandidateIdentity ||
				got[0].CurrentStateEvidenceAcceptanceCandidateIdentity != current.EvidenceAcceptanceCandidateIdentity ||
				got[0].ObservedDecisionRevision != current.ObservedDecisionRevision ||
				got[0].ObservedStateFingerprint != current.ObservedStateFingerprint ||
				got[0].ReconciliationResultAcceptanceCandidateIdentity == "" {
				t.Fatalf("R3BC result acceptance candidate drifted: %+v", got)
			}
		})
	}
}

func TestActorContextR3BCWrongExactBindingFailsClosed(t *testing.T) {
	reconciliation, execution, current, gate, evidence :=
		actorContextR3BCSingleQualifiedFixture(t)
	base := actorContextR3BCExactWitness(
		reconciliation, execution, current, gate, actorContextR3BCDecisionAccept,
	)

	cases := map[string]func(*actorContextR3BCResultAcceptanceWitnessProbe){
		"authority": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.ResultAcceptanceAuthorityEvidenceID += ":other"
		},
		"reconciliation": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.ReconciliationCandidateIdentity += ":other"
		},
		"execution-candidate": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.ExecutionEvidenceAcceptanceCandidateIdentity += ":other"
		},
		"current-state-candidate": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.CurrentStateEvidenceAcceptanceCandidateIdentity += ":other"
		},
		"result-locus": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.ReconciliationResultDecisionLocusRef += ":other"
		},
		"current-state-locus": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.CurrentStateAcceptanceLocusRef += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3BCBuildCandidates(
				reconciliation, execution, current, gate, evidence,
				[]actorContextR3BCResultAcceptanceWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3BC accepted mismatched %s witness", name)
			}
		})
	}
}

func TestActorContextR3BCDecisionAndProvenanceMustBeExplicit(t *testing.T) {
	reconciliation, execution, current, gate, evidence :=
		actorContextR3BCSingleQualifiedFixture(t)
	base := actorContextR3BCExactWitness(
		reconciliation, execution, current, gate, actorContextR3BCDecisionAccept,
	)

	cases := map[string]func(*actorContextR3BCResultAcceptanceWitnessProbe){
		"missing-decision": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.Decision = ""
		},
		"unsupported-decision": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.Decision = "test-only:auto-accept"
		},
		"missing-provenance": func(v *actorContextR3BCResultAcceptanceWitnessProbe) {
			v.ResultAcceptanceProvenanceRef = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3BCBuildCandidates(
				reconciliation, execution, current, gate, evidence,
				[]actorContextR3BCResultAcceptanceWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3BC accepted %s witness", name)
			}
		})
	}
}

func TestActorContextR3BCGateQualificationDoesNotInferDecision(t *testing.T) {
	reconciliation, execution, current, gate, evidence :=
		actorContextR3BCSingleQualifiedFixture(t)

	got, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3BC inferred result decision from Gate/candidate content: %+v", got)
	}
}

func TestActorContextR3BCMultipleQualifiedGateCannotBeBypassed(t *testing.T) {
	reconciliation, execution, current, businessAddress :=
		actorContextR3BBBothAcceptCandidatesFixture(t)
	a := actorContextR3BBCompleteEvidence(
		"test-only:r3bc/a", businessAddress, reconciliation, execution, current,
	)
	b := actorContextR3BBCompleteEvidence(
		"test-only:r3bc/b", businessAddress, reconciliation, execution, current,
	)
	evidence := []actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{a, b}
	gate, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
		businessAddress,
		evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusMultipleQualified {
		t.Fatalf("R3BC setup lost multiple-qualified R3BB Gate: %+v", gate)
	}
	witness := actorContextR3BCResultAcceptanceWitnessProbe{
		ReconciliationCandidateIdentity:                 reconciliation.ReconciliationCandidateIdentity,
		ResultAcceptanceAuthorityEvidenceID:             a.EvidenceID,
		ExecutionEvidenceAcceptanceCandidateIdentity:    execution.EvidenceAcceptanceCandidateIdentity,
		CurrentStateEvidenceAcceptanceCandidateIdentity: current.EvidenceAcceptanceCandidateIdentity,
		ReconciliationResultDecisionLocusRef:            a.ReconciliationResultDecisionLocusRef,
		CurrentStateAcceptanceLocusRef:                  a.CurrentStateAcceptanceLocusRef,
		Decision:                                        actorContextR3BCDecisionAccept,
		ResultAcceptanceProvenanceRef:                   "test-only:r3bc/multiple-qualified",
	}
	if _, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence,
		[]actorContextR3BCResultAcceptanceWitnessProbe{witness},
	); err == nil {
		t.Fatal("R3BC allowed witness to bypass R3BB authority precedence")
	}
}

func TestActorContextR3BCDuplicateNormalizesAndConflictsFailClosed(t *testing.T) {
	reconciliation, execution, current, gate, evidence :=
		actorContextR3BCSingleQualifiedFixture(t)
	base := actorContextR3BCExactWitness(
		reconciliation, execution, current, gate, actorContextR3BCDecisionAccept,
	)

	got, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence,
		[]actorContextR3BCResultAcceptanceWitnessProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3BC did not normalize exact duplicate witness: %+v", got)
	}

	decisionConflict := base
	decisionConflict.Decision = actorContextR3BCDecisionReject
	if _, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence,
		[]actorContextR3BCResultAcceptanceWitnessProbe{base, decisionConflict},
	); err == nil {
		t.Fatal("R3BC silently selected among conflicting result decisions")
	}

	provenanceConflict := base
	provenanceConflict.ResultAcceptanceProvenanceRef += ":other"
	if _, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence,
		[]actorContextR3BCResultAcceptanceWitnessProbe{base, provenanceConflict},
	); err == nil {
		t.Fatal("R3BC silently selected among conflicting result acceptance provenance")
	}
}

func TestActorContextR3BCRepeatedBuildIsIdentityStable(t *testing.T) {
	reconciliation, execution, current, gate, evidence :=
		actorContextR3BCSingleQualifiedFixture(t)
	witness := actorContextR3BCExactWitness(
		reconciliation, execution, current, gate, actorContextR3BCDecisionAccept,
	)

	first, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence,
		[]actorContextR3BCResultAcceptanceWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence,
		[]actorContextR3BCResultAcceptanceWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3BC repeated build changed candidate identity: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3BCShapeHasNoAppliedResultSynchronizationOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3BCResultAcceptanceWitnessProbe{}),
		reflect.TypeOf(actorContextR3BCResultAcceptanceCandidateProbe{}),
	} {
		for _, forbidden := range []string{
			"ReconciliationResultAccepted", "AcceptanceApplied",
			"StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedDecision", "CurrentDecision", "CurrentAuthority", "AppliedAuthority",
			"RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3BC candidate introduced forbidden result/synchronization shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3BCSingleQualifiedFixture(
	t *testing.T,
) (
	actorContextR3AXReconciliationCandidateProbe,
	actorContextR3BAEvidenceAcceptanceCandidateProbe,
	actorContextR3BAEvidenceAcceptanceCandidateProbe,
	actorContextR3BBResultAcceptanceAuthorityGateProbe,
	[]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe,
) {
	t.Helper()
	reconciliation, execution, current, businessAddress :=
		actorContextR3BBBothAcceptCandidatesFixture(t)
	authority := actorContextR3BBCompleteEvidence(
		"test-only:r3bc/result-authority",
		businessAddress,
		reconciliation,
		execution,
		current,
	)
	evidence := []actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{authority}
	gate, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
		businessAddress,
		evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 ||
		len(gate.ReconciliationResultDecisionLocusRefs) != 1 ||
		len(gate.CurrentStateAcceptanceLocusRefs) != 1 {
		t.Fatalf("R3BC setup lost single-qualified R3BB Gate: %+v", gate)
	}
	return reconciliation, execution, current, gate, evidence
}

func actorContextR3BCExactWitness(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	execution actorContextR3BAEvidenceAcceptanceCandidateProbe,
	current actorContextR3BAEvidenceAcceptanceCandidateProbe,
	gate actorContextR3BBResultAcceptanceAuthorityGateProbe,
	decision string,
) actorContextR3BCResultAcceptanceWitnessProbe {
	return actorContextR3BCResultAcceptanceWitnessProbe{
		ReconciliationCandidateIdentity:                 reconciliation.ReconciliationCandidateIdentity,
		ResultAcceptanceAuthorityEvidenceID:             gate.QualifiedEvidenceIDs[0],
		ExecutionEvidenceAcceptanceCandidateIdentity:    execution.EvidenceAcceptanceCandidateIdentity,
		CurrentStateEvidenceAcceptanceCandidateIdentity: current.EvidenceAcceptanceCandidateIdentity,
		ReconciliationResultDecisionLocusRef:            gate.ReconciliationResultDecisionLocusRefs[0],
		CurrentStateAcceptanceLocusRef:                  gate.CurrentStateAcceptanceLocusRefs[0],
		Decision:                                        decision,
		ResultAcceptanceProvenanceRef:                   "test-only:r3bc/result-acceptance-witness",
	}
}

func actorContextR3BCBuildCandidates(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	execution actorContextR3BAEvidenceAcceptanceCandidateProbe,
	current actorContextR3BAEvidenceAcceptanceCandidateProbe,
	gate actorContextR3BBResultAcceptanceAuthorityGateProbe,
	evidence []actorContextR3BBResultAcceptanceAuthorityEvidenceProbe,
	witnesses []actorContextR3BCResultAcceptanceWitnessProbe,
) ([]actorContextR3BCResultAcceptanceCandidateProbe, error) {
	if len(witnesses) == 0 {
		return nil, nil
	}
	if reconciliation.ReconciliationCandidateIdentity == "" ||
		execution.EvidencePlane != actorContextR3BAPlaneExecution ||
		current.EvidencePlane != actorContextR3BAPlaneCurrentState ||
		execution.ReconciliationCandidateIdentity != reconciliation.ReconciliationCandidateIdentity ||
		current.ReconciliationCandidateIdentity != reconciliation.ReconciliationCandidateIdentity ||
		execution.Decision != actorContextR3BADecisionAccept ||
		current.Decision != actorContextR3BADecisionAccept {
		return nil, fmt.Errorf("R3BC current reconciliation/plane candidates are incomplete or not accepted")
	}
	if gate.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 ||
		len(gate.ReconciliationResultDecisionLocusRefs) != 1 ||
		len(gate.CurrentStateAcceptanceLocusRefs) != 1 ||
		gate.ExecutionEvidenceAcceptanceCandidateIdentity != execution.EvidenceAcceptanceCandidateIdentity ||
		gate.CurrentStateEvidenceAcceptanceCandidateIdentity != current.EvidenceAcceptanceCandidateIdentity {
		return nil, fmt.Errorf("R3BC requires one exact single-qualified R3BB Gate")
	}

	byID := map[string]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{}
	for _, value := range evidence {
		if value.EvidenceID == "" {
			return nil, fmt.Errorf("R3BC result acceptance authority EvidenceID is empty")
		}
		if _, exists := byID[value.EvidenceID]; exists {
			return nil, fmt.Errorf("R3BC duplicate result acceptance authority EvidenceID %q", value.EvidenceID)
		}
		byID[value.EvidenceID] = value
	}

	normalized := map[string]actorContextR3BCResultAcceptanceWitnessProbe{}
	out := []actorContextR3BCResultAcceptanceCandidateProbe{}
	for _, witness := range witnesses {
		if witness.ReconciliationCandidateIdentity != reconciliation.ReconciliationCandidateIdentity ||
			witness.ResultAcceptanceAuthorityEvidenceID != gate.QualifiedEvidenceIDs[0] ||
			witness.ExecutionEvidenceAcceptanceCandidateIdentity != execution.EvidenceAcceptanceCandidateIdentity ||
			witness.CurrentStateEvidenceAcceptanceCandidateIdentity != current.EvidenceAcceptanceCandidateIdentity ||
			witness.ReconciliationResultDecisionLocusRef != gate.ReconciliationResultDecisionLocusRefs[0] ||
			witness.CurrentStateAcceptanceLocusRef != gate.CurrentStateAcceptanceLocusRefs[0] ||
			(witness.Decision != actorContextR3BCDecisionAccept &&
				witness.Decision != actorContextR3BCDecisionReject) ||
			witness.ResultAcceptanceProvenanceRef == "" {
			return nil, fmt.Errorf("R3BC explicit reconciliation-result acceptance witness is incomplete or mismatched")
		}

		authority, ok := byID[witness.ResultAcceptanceAuthorityEvidenceID]
		if !ok ||
			authority.EvidenceRole != actorContextR3BBRoleResultAuthorityCandidate ||
			!authority.RepositoryBacked ||
			authority.DerivedOnly ||
			authority.AuthorityKind != actorContextR3BBAuthorityKind ||
			authority.AuthorityDeclarationRef == "" ||
			!actorContextR0Contains(
				authority.ReconciliationCandidateIdentities,
				reconciliation.ReconciliationCandidateIdentity,
			) ||
			!actorContextR0Contains(
				authority.ExecutionEvidenceAcceptanceCandidateIdentities,
				execution.EvidenceAcceptanceCandidateIdentity,
			) ||
			!actorContextR0Contains(
				authority.CurrentStateEvidenceAcceptanceCandidateIdentities,
				current.EvidenceAcceptanceCandidateIdentity,
			) ||
			authority.AcceptedDecisionSinkRef != reconciliation.AcceptedDecisionSinkRef ||
			authority.RevisionAuthorityRef != reconciliation.RevisionAuthorityRef ||
			authority.ReconciliationResultDecisionLocusRef != witness.ReconciliationResultDecisionLocusRef ||
			authority.CurrentStateAcceptanceLocusRef != witness.CurrentStateAcceptanceLocusRef ||
			authority.ResultAcceptanceProvenanceRef == "" {
			return nil, fmt.Errorf("R3BC qualified authority no longer binds exact reconciliation-result context")
		}

		key := strings.Join([]string{
			witness.ResultAcceptanceAuthorityEvidenceID,
			witness.ExecutionEvidenceAcceptanceCandidateIdentity,
			witness.CurrentStateEvidenceAcceptanceCandidateIdentity,
		}, "||")
		if previous, ok := normalized[key]; ok {
			if !reflect.DeepEqual(previous, witness) {
				return nil, fmt.Errorf("R3BC conflicting result acceptance witnesses require separate precedence authority")
			}
			continue
		}
		normalized[key] = witness

		identity := strings.Join([]string{
			reconciliation.ReconciliationCandidateIdentity,
			witness.ResultAcceptanceAuthorityEvidenceID,
			witness.ExecutionEvidenceAcceptanceCandidateIdentity,
			witness.CurrentStateEvidenceAcceptanceCandidateIdentity,
			witness.ReconciliationResultDecisionLocusRef,
			witness.CurrentStateAcceptanceLocusRef,
			current.ObservedDecisionRevision,
			current.ObservedStateFingerprint,
			witness.Decision,
			witness.ResultAcceptanceProvenanceRef,
		}, "||")
		out = append(out, actorContextR3BCResultAcceptanceCandidateProbe{
			ReconciliationCandidateIdentity:                 reconciliation.ReconciliationCandidateIdentity,
			ResultAcceptanceAuthorityEvidenceID:             witness.ResultAcceptanceAuthorityEvidenceID,
			ExecutionEvidenceAcceptanceCandidateIdentity:    execution.EvidenceAcceptanceCandidateIdentity,
			CurrentStateEvidenceAcceptanceCandidateIdentity: current.EvidenceAcceptanceCandidateIdentity,
			ReconciliationResultDecisionLocusRef:            witness.ReconciliationResultDecisionLocusRef,
			CurrentStateAcceptanceLocusRef:                  witness.CurrentStateAcceptanceLocusRef,
			ObservedDecisionRevision:                        current.ObservedDecisionRevision,
			ObservedStateFingerprint:                        current.ObservedStateFingerprint,
			Decision:                                        witness.Decision,
			ResultAcceptanceProvenanceRef:                   witness.ResultAcceptanceProvenanceRef,
			ReconciliationResultAcceptanceCandidateIdentity: identity,
			AuthorityLimits: actorContextR0SortedUnique([]string{
				"result-acceptance-candidate-does-not-apply-authoritative-state",
				"result-acceptance-candidate-does-not-synchronize-current-state",
				"reject-result-candidate-does-not-create-retry-rollback-compensation-or-failure",
				"qualified-result-gate-and-plane-candidates-do-not-infer-result-decision",
			}),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ReconciliationResultAcceptanceCandidateIdentity <
			out[j].ReconciliationResultAcceptanceCandidateIdentity
	})
	return out, nil
}

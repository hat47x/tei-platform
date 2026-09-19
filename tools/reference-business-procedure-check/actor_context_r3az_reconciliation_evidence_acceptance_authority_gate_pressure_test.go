package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3AZExecutionStatusUnresolved        = "reconciliation-execution-evidence-acceptance-authority-unresolved"
	actorContextR3AZExecutionStatusQualified         = "reconciliation-execution-evidence-acceptance-authority-evidence-qualified"
	actorContextR3AZExecutionStatusMultipleQualified = "reconciliation-execution-evidence-acceptance-authority-multiple-qualified-precedence-unresolved"

	actorContextR3AZCurrentStateStatusUnresolved        = "reconciliation-current-state-evidence-acceptance-authority-unresolved"
	actorContextR3AZCurrentStateStatusQualified         = "reconciliation-current-state-evidence-acceptance-authority-evidence-qualified"
	actorContextR3AZCurrentStateStatusMultipleQualified = "reconciliation-current-state-evidence-acceptance-authority-multiple-qualified-precedence-unresolved"

	actorContextR3AZExecutionAuthorityKind    = "operational-reconciliation-execution-evidence-acceptance-authority"
	actorContextR3AZCurrentStateAuthorityKind = "operational-reconciliation-current-state-evidence-acceptance-authority"

	actorContextR3AZRoleDerivedExecutionObservation    = "derived-reconciliation-execution-observation"
	actorContextR3AZRoleDerivedCurrentStateObservation = "derived-reconciliation-current-state-observation"
	actorContextR3AZRoleExecutionAuthorityCandidate    = "reconciliation-execution-evidence-acceptance-authority-candidate"
	actorContextR3AZRoleCurrentStateAuthorityCandidate = "reconciliation-current-state-evidence-acceptance-authority-candidate"

	actorContextR3AZScopeExactBusinessAddress = "exact-business-address"
	actorContextR3AZScopeAnyBusinessAddress   = "any-business-address"
)

type actorContextR3AZEvidenceProbe struct {
	EvidenceID                        string
	EvidenceRole                      string
	RepositoryBacked                  bool
	AuthorityKind                     string
	AuthorityDeclarationRef           string
	ReconciliationCandidateIdentities []string
	ExecutionObservationRefs          []string
	ReportedExecutionOutcomes         []string
	CurrentStateObservationRefs       []string
	ObservedDecisionRevisions         []string
	ObservedStateFingerprints         []string
	AcceptedDecisionSinkRef           string
	RevisionAuthorityRef              string
	ScopeKind                         string
	ScopeValue                        string
	EvidenceDecisionLocusRef          string
	EvidenceAcceptanceProvenanceRef   string
	DerivedOnly                       bool
}

type actorContextR3AZGateProbe struct {
	ExecutionEvidenceAcceptanceAuthorityStatus    string
	CurrentStateEvidenceAcceptanceAuthorityStatus string
	QualifiedExecutionEvidenceIDs                 []string
	QualifiedCurrentStateEvidenceIDs              []string
	ExecutionEvidenceDecisionLocusRefs            []string
	CurrentStateEvidenceDecisionLocusRefs         []string
	InsufficientEvidenceIDs                       []string
	AuthorityLimits                               []string
}

func TestActorContextR3AZCurrentInspectedEvidenceKeepsBothPlanesUnresolved(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)

	got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate,
		observations,
		businessAddress,
		actorContextR3AZCurrentInspectedEvidence(observations),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusUnresolved ||
		got.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusUnresolved ||
		len(got.QualifiedExecutionEvidenceIDs) != 0 ||
		len(got.QualifiedCurrentStateEvidenceIDs) != 0 {
		t.Fatalf("R3AZ fabricated acceptance authority from current raw evidence: %+v", got)
	}
}

func TestActorContextR3AZExecutionAuthorityQualifiesOnlyExecutionPlane(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	evidence := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3az/execution-authority",
		businessAddress,
		candidate,
		observations.ExecutionObservations[0],
	)

	got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate,
		observations,
		businessAddress,
		[]actorContextR3AZEvidenceProbe{evidence},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusQualified ||
		got.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusUnresolved ||
		!reflect.DeepEqual(got.QualifiedExecutionEvidenceIDs, []string{evidence.EvidenceID}) ||
		len(got.QualifiedCurrentStateEvidenceIDs) != 0 {
		t.Fatalf("R3AZ execution authority leaked across evidence planes: %+v", got)
	}
}

func TestActorContextR3AZCurrentStateAuthorityQualifiesOnlyCurrentStatePlane(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	evidence := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3az/current-state-authority",
		businessAddress,
		candidate,
		observations.CurrentStateObservations[0],
	)

	got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate,
		observations,
		businessAddress,
		[]actorContextR3AZEvidenceProbe{evidence},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusUnresolved ||
		got.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusQualified ||
		len(got.QualifiedExecutionEvidenceIDs) != 0 ||
		!reflect.DeepEqual(got.QualifiedCurrentStateEvidenceIDs, []string{evidence.EvidenceID}) {
		t.Fatalf("R3AZ current-state authority leaked across evidence planes: %+v", got)
	}
}

func TestActorContextR3AZBothAuthoritiesQualifyGatesOnly(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	execution := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3az/execution-authority",
		businessAddress,
		candidate,
		observations.ExecutionObservations[0],
	)
	current := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3az/current-state-authority",
		businessAddress,
		candidate,
		observations.CurrentStateObservations[0],
	)

	got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate,
		observations,
		businessAddress,
		[]actorContextR3AZEvidenceProbe{execution, current},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusQualified ||
		got.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusQualified {
		t.Fatalf("R3AZ complete authorities did not qualify independent Gates: %+v", got)
	}
}

func TestActorContextR3AZWrongExactBindingDoesNotQualify(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)

	executionBase := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3az/execution-base",
		businessAddress,
		candidate,
		observations.ExecutionObservations[0],
	)
	executionCases := map[string]func(*actorContextR3AZEvidenceProbe){
		"candidate": func(v *actorContextR3AZEvidenceProbe) {
			v.ReconciliationCandidateIdentities = []string{candidate.ReconciliationCandidateIdentity + ":other"}
		},
		"observation": func(v *actorContextR3AZEvidenceProbe) {
			v.ExecutionObservationRefs = []string{observations.ExecutionObservations[0].ReconciliationExecutionObservationRef + ":other"}
		},
		"outcome": func(v *actorContextR3AZEvidenceProbe) {
			v.ReportedExecutionOutcomes = []string{actorContextR3AYReportedConflict}
		},
		"sink": func(v *actorContextR3AZEvidenceProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"scope": func(v *actorContextR3AZEvidenceProbe) {
			v.ScopeValue = "other-procedure/BP-99"
		},
	}
	for name, mutate := range executionCases {
		t.Run("execution-"+name, func(t *testing.T) {
			value := actorContextR3AZCloneEvidence(executionBase)
			mutate(&value)
			got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
				candidate, observations, businessAddress,
				[]actorContextR3AZEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusUnresolved {
				t.Fatalf("R3AZ qualified mismatched execution %s evidence: %+v", name, got)
			}
		})
	}

	currentBase := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3az/current-base",
		businessAddress,
		candidate,
		observations.CurrentStateObservations[0],
	)
	currentCases := map[string]func(*actorContextR3AZEvidenceProbe){
		"candidate": func(v *actorContextR3AZEvidenceProbe) {
			v.ReconciliationCandidateIdentities = []string{candidate.ReconciliationCandidateIdentity + ":other"}
		},
		"observation": func(v *actorContextR3AZEvidenceProbe) {
			v.CurrentStateObservationRefs = []string{observations.CurrentStateObservations[0].CurrentStateObservationRef + ":other"}
		},
		"revision": func(v *actorContextR3AZEvidenceProbe) {
			v.ObservedDecisionRevisions = []string{observations.CurrentStateObservations[0].ObservedDecisionRevision + ":other"}
		},
		"fingerprint": func(v *actorContextR3AZEvidenceProbe) {
			v.ObservedStateFingerprints = []string{"sha256:other"}
		},
		"sink": func(v *actorContextR3AZEvidenceProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(v *actorContextR3AZEvidenceProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"scope": func(v *actorContextR3AZEvidenceProbe) {
			v.ScopeValue = "other-procedure/BP-99"
		},
	}
	for name, mutate := range currentCases {
		t.Run("current-"+name, func(t *testing.T) {
			value := actorContextR3AZCloneEvidence(currentBase)
			mutate(&value)
			got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
				candidate, observations, businessAddress,
				[]actorContextR3AZEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusUnresolved {
				t.Fatalf("R3AZ qualified mismatched current-state %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3AZMissingAuthorityLocusOrProvenanceDoesNotQualify(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	base := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3az/base",
		businessAddress,
		candidate,
		observations.ExecutionObservations[0],
	)

	cases := map[string]func(*actorContextR3AZEvidenceProbe){
		"repository": func(v *actorContextR3AZEvidenceProbe) {
			v.RepositoryBacked = false
		},
		"authority-kind": func(v *actorContextR3AZEvidenceProbe) {
			v.AuthorityKind = ""
		},
		"declaration": func(v *actorContextR3AZEvidenceProbe) {
			v.AuthorityDeclarationRef = ""
		},
		"locus": func(v *actorContextR3AZEvidenceProbe) {
			v.EvidenceDecisionLocusRef = ""
		},
		"provenance": func(v *actorContextR3AZEvidenceProbe) {
			v.EvidenceAcceptanceProvenanceRef = ""
		},
		"derived": func(v *actorContextR3AZEvidenceProbe) {
			v.DerivedOnly = true
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3AZCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
				candidate, observations, businessAddress,
				[]actorContextR3AZEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusUnresolved {
				t.Fatalf("R3AZ qualified incomplete %s authority evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3AZRawOutcomeAndRevisionRelationDoNotCreateAuthority(t *testing.T) {
	for _, outcome := range []string{
		actorContextR3AYReportedApplied,
		actorContextR3AYReportedNotApplied,
		actorContextR3AYReportedConflict,
		actorContextR3AYReportedUnknown,
	} {
		t.Run(outcome, func(t *testing.T) {
			candidate, observations, businessAddress := actorContextR3AZObservationFixtureWithOutcome(t, outcome)
			got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
				candidate,
				observations,
				businessAddress,
				actorContextR3AZCurrentInspectedEvidence(observations),
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusUnresolved ||
				got.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusUnresolved {
				t.Fatalf("R3AZ inferred acceptance authority from raw outcome/revision relation %q: %+v", outcome, got)
			}
		})
	}
}

func TestActorContextR3AZMultipleQualifiedIsPlaneLocalAndPrecedenceUnresolved(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	executionA := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3az/execution-a",
		businessAddress,
		candidate,
		observations.ExecutionObservations[0],
	)
	executionB := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3az/execution-b",
		businessAddress,
		candidate,
		observations.ExecutionObservations[0],
	)
	current := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3az/current",
		businessAddress,
		candidate,
		observations.CurrentStateObservations[0],
	)

	got, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate,
		observations,
		businessAddress,
		[]actorContextR3AZEvidenceProbe{executionB, current, executionA},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExecutionEvidenceAcceptanceAuthorityStatus != actorContextR3AZExecutionStatusMultipleQualified ||
		got.CurrentStateEvidenceAcceptanceAuthorityStatus != actorContextR3AZCurrentStateStatusQualified ||
		!reflect.DeepEqual(got.QualifiedExecutionEvidenceIDs, []string{executionA.EvidenceID, executionB.EvidenceID}) {
		t.Fatalf("R3AZ did not keep multiple-qualified precedence unresolved per plane: %+v", got)
	}
}

func TestActorContextR3AZInputOrderDoesNotChangeGateResult(t *testing.T) {
	candidate, observations, businessAddress := actorContextR3AZObservationFixture(t)
	execution := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3az/execution",
		businessAddress,
		candidate,
		observations.ExecutionObservations[0],
	)
	current := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3az/current",
		businessAddress,
		candidate,
		observations.CurrentStateObservations[0],
	)
	insufficient := actorContextR3AZCloneEvidence(execution)
	insufficient.EvidenceID = "test-only:r3az/insufficient"
	insufficient.EvidenceDecisionLocusRef = ""

	first, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate, observations, businessAddress,
		[]actorContextR3AZEvidenceProbe{execution, current, insufficient},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		candidate, observations, businessAddress,
		[]actorContextR3AZEvidenceProbe{insufficient, current, execution},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AZ input order changed Gate result: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3AZShapeHasNoAcceptedEvidenceSynchronizationOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AZEvidenceProbe{}),
		reflect.TypeOf(actorContextR3AZGateProbe{}),
	} {
		for _, forbidden := range []string{
			"ExecutionEvidenceAccepted", "CurrentStateEvidenceAccepted",
			"AcceptedExecutionOutcome", "AcceptedCurrentState",
			"StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedDecision", "CurrentDecision", "CurrentAuthority", "AppliedAuthority",
			"Retry", "RetryEligible", "RetryRequired", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AZ Gate introduced forbidden acceptance/synchronization shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AZObservationFixture(
	t *testing.T,
) (
	actorContextR3AXReconciliationCandidateProbe,
	actorContextR3AYObservationSetProbe,
	string,
) {
	return actorContextR3AZObservationFixtureWithOutcome(t, actorContextR3AYReportedApplied)
}

func actorContextR3AZObservationFixtureWithOutcome(
	t *testing.T,
	rawOutcome string,
) (
	actorContextR3AXReconciliationCandidateProbe,
	actorContextR3AYObservationSetProbe,
	string,
) {
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
		actorContextR3AXDecisionAccept,
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
		t.Fatalf("R3AZ setup reconciliation candidate count=%d want=1", len(candidates))
	}
	candidate := candidates[0]

	execution := actorContextR3AYExactExecutionWitness(candidate, rawOutcome)
	state := actorContextR3AYExactCurrentStateWitness(
		candidate,
		candidate.PostWriteDecisionRevision,
		"sha256:test-only-r3az-current-state",
	)
	observations, err := actorContextR3AYObserveReconciliation(
		candidate,
		[]actorContextR3AYExecutionObservationWitnessProbe{execution},
		[]actorContextR3AYCurrentStateObservationWitnessProbe{state},
	)
	if err != nil {
		t.Fatal(err)
	}
	return candidate, observations, businessAddress
}

func actorContextR3AZCurrentInspectedEvidence(
	observations actorContextR3AYObservationSetProbe,
) []actorContextR3AZEvidenceProbe {
	out := []actorContextR3AZEvidenceProbe{}
	for _, value := range observations.ExecutionObservations {
		out = append(out, actorContextR3AZEvidenceProbe{
			EvidenceID:                        value.ReconciliationExecutionObservationRef,
			EvidenceRole:                      actorContextR3AZRoleDerivedExecutionObservation,
			ReconciliationCandidateIdentities: []string{value.ReconciliationCandidateIdentity},
			ExecutionObservationRefs:          []string{value.ReconciliationExecutionObservationRef},
			ReportedExecutionOutcomes:         []string{value.ReportedExecutionOutcome},
			AcceptedDecisionSinkRef:           value.AcceptedDecisionSinkRef,
			DerivedOnly:                       true,
		})
	}
	for _, value := range observations.CurrentStateObservations {
		out = append(out, actorContextR3AZEvidenceProbe{
			EvidenceID:                        value.CurrentStateObservationRef,
			EvidenceRole:                      actorContextR3AZRoleDerivedCurrentStateObservation,
			ReconciliationCandidateIdentities: []string{value.ReconciliationCandidateIdentity},
			CurrentStateObservationRefs:       []string{value.CurrentStateObservationRef},
			ObservedDecisionRevisions:         []string{value.ObservedDecisionRevision},
			ObservedStateFingerprints:         []string{value.ObservedStateFingerprint},
			AcceptedDecisionSinkRef:           value.AcceptedDecisionSinkRef,
			RevisionAuthorityRef:              value.RevisionAuthorityRef,
			DerivedOnly:                       true,
		})
	}
	return out
}

func actorContextR3AZCompleteExecutionEvidence(
	id string,
	businessAddress string,
	candidate actorContextR3AXReconciliationCandidateProbe,
	observation actorContextR3AYRawExecutionObservationProbe,
) actorContextR3AZEvidenceProbe {
	return actorContextR3AZEvidenceProbe{
		EvidenceID:                        id,
		EvidenceRole:                      actorContextR3AZRoleExecutionAuthorityCandidate,
		RepositoryBacked:                  true,
		AuthorityKind:                     actorContextR3AZExecutionAuthorityKind,
		AuthorityDeclarationRef:           id + "/authority-declaration",
		ReconciliationCandidateIdentities: []string{candidate.ReconciliationCandidateIdentity},
		ExecutionObservationRefs:          []string{observation.ReconciliationExecutionObservationRef},
		ReportedExecutionOutcomes:         []string{observation.ReportedExecutionOutcome},
		AcceptedDecisionSinkRef:           candidate.AcceptedDecisionSinkRef,
		ScopeKind:                         actorContextR3AZScopeExactBusinessAddress,
		ScopeValue:                        businessAddress,
		EvidenceDecisionLocusRef:          id + "/execution-evidence-decision-locus",
		EvidenceAcceptanceProvenanceRef:   id + "/acceptance-provenance",
	}
}

func actorContextR3AZCompleteCurrentStateEvidence(
	id string,
	businessAddress string,
	candidate actorContextR3AXReconciliationCandidateProbe,
	observation actorContextR3AYRawCurrentStateObservationProbe,
) actorContextR3AZEvidenceProbe {
	return actorContextR3AZEvidenceProbe{
		EvidenceID:                        id,
		EvidenceRole:                      actorContextR3AZRoleCurrentStateAuthorityCandidate,
		RepositoryBacked:                  true,
		AuthorityKind:                     actorContextR3AZCurrentStateAuthorityKind,
		AuthorityDeclarationRef:           id + "/authority-declaration",
		ReconciliationCandidateIdentities: []string{candidate.ReconciliationCandidateIdentity},
		CurrentStateObservationRefs:       []string{observation.CurrentStateObservationRef},
		ObservedDecisionRevisions:         []string{observation.ObservedDecisionRevision},
		ObservedStateFingerprints:         []string{observation.ObservedStateFingerprint},
		AcceptedDecisionSinkRef:           candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:              candidate.RevisionAuthorityRef,
		ScopeKind:                         actorContextR3AZScopeExactBusinessAddress,
		ScopeValue:                        businessAddress,
		EvidenceDecisionLocusRef:          id + "/current-state-evidence-decision-locus",
		EvidenceAcceptanceProvenanceRef:   id + "/acceptance-provenance",
	}
}

func actorContextR3AZAssessEvidenceAcceptanceAuthority(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observations actorContextR3AYObservationSetProbe,
	businessAddress string,
	evidence []actorContextR3AZEvidenceProbe,
) (actorContextR3AZGateProbe, error) {
	if candidate.ReconciliationCandidateIdentity == "" ||
		candidate.AcceptedDecisionSinkRef == "" ||
		candidate.RevisionAuthorityRef == "" ||
		observations.ReconciliationCandidateIdentity != candidate.ReconciliationCandidateIdentity ||
		businessAddress == "" {
		return actorContextR3AZGateProbe{}, fmt.Errorf(
			"R3AZ reconciliation candidate/observation/business context is incomplete or inconsistent",
		)
	}

	executionQualified := []actorContextR3AZEvidenceProbe{}
	currentQualified := []actorContextR3AZEvidenceProbe{}
	insufficient := []string{}
	seenIDs := map[string]bool{}

	for _, value := range evidence {
		if value.EvidenceID == "" || value.EvidenceRole == "" {
			return actorContextR3AZGateProbe{}, fmt.Errorf("R3AZ evidence identity/role is incomplete")
		}
		if seenIDs[value.EvidenceID] {
			return actorContextR3AZGateProbe{}, fmt.Errorf("R3AZ duplicate evidence id %q", value.EvidenceID)
		}
		seenIDs[value.EvidenceID] = true

		switch {
		case actorContextR3AZExecutionCandidateQualifies(
			candidate, observations, businessAddress, value,
		):
			executionQualified = append(executionQualified, value)
		case actorContextR3AZCurrentStateCandidateQualifies(
			candidate, observations, businessAddress, value,
		):
			currentQualified = append(currentQualified, value)
		default:
			insufficient = append(insufficient, value.EvidenceID)
		}
	}

	sort.Slice(executionQualified, func(i, j int) bool {
		return executionQualified[i].EvidenceID < executionQualified[j].EvidenceID
	})
	sort.Slice(currentQualified, func(i, j int) bool {
		return currentQualified[i].EvidenceID < currentQualified[j].EvidenceID
	})

	executionStatus := actorContextR3AZPlaneStatus(
		len(executionQualified),
		actorContextR3AZExecutionStatusUnresolved,
		actorContextR3AZExecutionStatusQualified,
		actorContextR3AZExecutionStatusMultipleQualified,
	)
	currentStatus := actorContextR3AZPlaneStatus(
		len(currentQualified),
		actorContextR3AZCurrentStateStatusUnresolved,
		actorContextR3AZCurrentStateStatusQualified,
		actorContextR3AZCurrentStateStatusMultipleQualified,
	)

	executionIDs := []string{}
	currentIDs := []string{}
	executionLoci := []string{}
	currentLoci := []string{}
	for _, value := range executionQualified {
		executionIDs = append(executionIDs, value.EvidenceID)
		executionLoci = append(executionLoci, value.EvidenceDecisionLocusRef)
	}
	for _, value := range currentQualified {
		currentIDs = append(currentIDs, value.EvidenceID)
		currentLoci = append(currentLoci, value.EvidenceDecisionLocusRef)
	}

	return actorContextR3AZGateProbe{
		ExecutionEvidenceAcceptanceAuthorityStatus:    executionStatus,
		CurrentStateEvidenceAcceptanceAuthorityStatus: currentStatus,
		QualifiedExecutionEvidenceIDs:                 actorContextR0SortedUnique(executionIDs),
		QualifiedCurrentStateEvidenceIDs:              actorContextR0SortedUnique(currentIDs),
		ExecutionEvidenceDecisionLocusRefs:            actorContextR0SortedUnique(executionLoci),
		CurrentStateEvidenceDecisionLocusRefs:         actorContextR0SortedUnique(currentLoci),
		InsufficientEvidenceIDs:                       actorContextR0SortedUnique(insufficient),
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"raw-execution-observation-does-not-create-execution-evidence-acceptance-authority",
			"raw-current-state-observation-does-not-create-current-state-evidence-acceptance-authority",
			"one-evidence-plane-authority-does-not-qualify-the-other-plane",
			"qualified-evidence-acceptance-authority-does-not-accept-raw-evidence",
			"matching-revision-does-not-create-current-state-acceptance-authority",
			"divergent-revision-or-conflict-does-not-create-retry-or-rollback-authority",
		}),
	}, nil
}

func actorContextR3AZExecutionCandidateQualifies(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observations actorContextR3AYObservationSetProbe,
	businessAddress string,
	value actorContextR3AZEvidenceProbe,
) bool {
	if value.EvidenceRole != actorContextR3AZRoleExecutionAuthorityCandidate ||
		!value.RepositoryBacked ||
		value.DerivedOnly ||
		value.AuthorityKind != actorContextR3AZExecutionAuthorityKind ||
		value.AuthorityDeclarationRef == "" ||
		!actorContextR0Contains(value.ReconciliationCandidateIdentities, candidate.ReconciliationCandidateIdentity) ||
		value.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
		value.EvidenceDecisionLocusRef == "" ||
		value.EvidenceAcceptanceProvenanceRef == "" ||
		!actorContextR3AZScopeMatches(value, businessAddress) {
		return false
	}
	for _, observation := range observations.ExecutionObservations {
		if actorContextR0Contains(value.ExecutionObservationRefs, observation.ReconciliationExecutionObservationRef) &&
			actorContextR0Contains(value.ReportedExecutionOutcomes, observation.ReportedExecutionOutcome) {
			return true
		}
	}
	return false
}

func actorContextR3AZCurrentStateCandidateQualifies(
	candidate actorContextR3AXReconciliationCandidateProbe,
	observations actorContextR3AYObservationSetProbe,
	businessAddress string,
	value actorContextR3AZEvidenceProbe,
) bool {
	if value.EvidenceRole != actorContextR3AZRoleCurrentStateAuthorityCandidate ||
		!value.RepositoryBacked ||
		value.DerivedOnly ||
		value.AuthorityKind != actorContextR3AZCurrentStateAuthorityKind ||
		value.AuthorityDeclarationRef == "" ||
		!actorContextR0Contains(value.ReconciliationCandidateIdentities, candidate.ReconciliationCandidateIdentity) ||
		value.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
		value.RevisionAuthorityRef != candidate.RevisionAuthorityRef ||
		value.EvidenceDecisionLocusRef == "" ||
		value.EvidenceAcceptanceProvenanceRef == "" ||
		!actorContextR3AZScopeMatches(value, businessAddress) {
		return false
	}
	for _, observation := range observations.CurrentStateObservations {
		if actorContextR0Contains(value.CurrentStateObservationRefs, observation.CurrentStateObservationRef) &&
			actorContextR0Contains(value.ObservedDecisionRevisions, observation.ObservedDecisionRevision) &&
			actorContextR0Contains(value.ObservedStateFingerprints, observation.ObservedStateFingerprint) {
			return true
		}
	}
	return false
}

func actorContextR3AZScopeMatches(value actorContextR3AZEvidenceProbe, businessAddress string) bool {
	switch value.ScopeKind {
	case actorContextR3AZScopeExactBusinessAddress:
		return value.ScopeValue == businessAddress
	case actorContextR3AZScopeAnyBusinessAddress:
		return true
	default:
		return false
	}
}

func actorContextR3AZPlaneStatus(count int, unresolved, qualified, multiple string) string {
	switch {
	case count == 0:
		return unresolved
	case count == 1:
		return qualified
	default:
		return multiple
	}
}

func actorContextR3AZCloneEvidence(value actorContextR3AZEvidenceProbe) actorContextR3AZEvidenceProbe {
	out := value
	out.ReconciliationCandidateIdentities = append([]string(nil), value.ReconciliationCandidateIdentities...)
	out.ExecutionObservationRefs = append([]string(nil), value.ExecutionObservationRefs...)
	out.ReportedExecutionOutcomes = append([]string(nil), value.ReportedExecutionOutcomes...)
	out.CurrentStateObservationRefs = append([]string(nil), value.CurrentStateObservationRefs...)
	out.ObservedDecisionRevisions = append([]string(nil), value.ObservedDecisionRevisions...)
	out.ObservedStateFingerprints = append([]string(nil), value.ObservedStateFingerprints...)
	return out
}

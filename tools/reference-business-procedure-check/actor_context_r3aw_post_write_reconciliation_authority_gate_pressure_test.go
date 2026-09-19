package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3AWStatusUnresolved        = "post-write-authoritative-state-reconciliation-authority-unresolved"
	actorContextR3AWStatusQualified         = "post-write-authoritative-state-reconciliation-authority-evidence-qualified"
	actorContextR3AWStatusMultipleQualified = "post-write-reconciliation-authority-multiple-qualified-precedence-unresolved"

	actorContextR3AWAuthorityKind = "operational-post-write-state-reconciliation-authority"

	actorContextR3AWRoleDerivedOutcomeCandidate          = "derived-outcome-acceptance-candidate"
	actorContextR3AWRoleRevisionAuthorityPrecedent       = "revision-authority-precedent"
	actorContextR3AWRoleReconciliationAuthorityCandidate = "post-write-reconciliation-authority-candidate"

	actorContextR3AWScopeExactBusinessAddress = "exact-business-address"
	actorContextR3AWScopeAnyBusinessAddress   = "any-business-address"
)

type actorContextR3AWReconciliationAuthorityEvidenceProbe struct {
	EvidenceID                           string
	EvidenceRole                         string
	RepositoryBacked                     bool
	AuthorityKind                        string
	AuthorityDeclarationRef              string
	OutcomeAcceptanceCandidateIdentities []string
	ObservationIdentities                []string
	WriteAttemptCandidateIdentities      []string
	CommitCandidateIdentities            []string
	AcceptedDecisionSinkRef              string
	RevisionAuthorityRef                 string
	PreWriteDecisionRevisions            []string
	PostWriteRevisionObservationRefs     []string
	PostWriteDecisionRevisions           []string
	ScopeKind                            string
	ScopeValue                           string
	ReconciliationLocusRef               string
	RevisionAcceptanceLocusRef           string
	AuthorityApplicationLocusRef         string
	ReconciliationProvenanceRef          string
	DerivedOnly                          bool
}

type actorContextR3AWReconciliationAuthorityGateProbe struct {
	ReconciliationAuthorityStatus string
	QualifiedEvidenceIDs          []string
	ReconciliationLocusRefs       []string
	RevisionAcceptanceLocusRefs   []string
	AuthorityApplicationLocusRefs []string
	AcceptedDecisionSinkRefs      []string
	PreWriteDecisionRevision      string
	PostWriteDecisionRevision     string
	InsufficientEvidenceIDs       []string
	AuthorityLimits               []string
}

func TestActorContextR3AWCurrentInspectedEvidenceKeepsReconciliationAuthorityUnresolved(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		false,
	)

	got, err := actorContextR3AWAssessReconciliationAuthority(
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
	if got.ReconciliationAuthorityStatus != actorContextR3AWStatusUnresolved ||
		len(got.QualifiedEvidenceIDs) != 0 ||
		len(got.InsufficientEvidenceIDs) == 0 {
		t.Fatalf("R3AW fabricated reconciliation authority from current inspected evidence: %+v", got)
	}
}

func TestActorContextR3AWAcceptCandidateAloneDoesNotQualify(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		false,
	)

	got, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		[]actorContextR3AWReconciliationAuthorityEvidenceProbe{{
			EvidenceID:                           outcomeCandidate.OutcomeAcceptanceCandidateIdentity,
			EvidenceRole:                         actorContextR3AWRoleDerivedOutcomeCandidate,
			OutcomeAcceptanceCandidateIdentities: []string{outcomeCandidate.OutcomeAcceptanceCandidateIdentity},
			ObservationIdentities:                []string{observation.ObservationIdentity},
			WriteAttemptCandidateIdentities:      []string{attempt.WriteAttemptCandidateIdentity},
			CommitCandidateIdentities:            []string{commitCandidate.CommitCandidateIdentity},
			AcceptedDecisionSinkRef:              attempt.AcceptedDecisionSinkRef,
			DerivedOnly:                          true,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReconciliationAuthorityStatus != actorContextR3AWStatusUnresolved {
		t.Fatalf("R3AW promoted accepted-outcome candidate into reconciliation authority: %+v", got)
	}
}

func TestActorContextR3AWRejectCandidateIsNotReconciliationEntry(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedConflict,
		actorContextR3AVDecisionReject,
		false,
	)
	complete := actorContextR3AWCompleteEvidence(
		"test-only:r3aw/reconciliation",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)
	if _, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		[]actorContextR3AWReconciliationAuthorityEvidenceProbe{complete},
	); err == nil {
		t.Fatal("R3AW allowed reject-observed-outcome candidate into authoritative-state reconciliation Gate")
	}
}

func TestActorContextR3AWCompleteSyntheticEvidenceQualifiesGateOnly(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		true,
	)
	complete := actorContextR3AWCompleteEvidence(
		"test-only:r3aw/reconciliation",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)

	got, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		[]actorContextR3AWReconciliationAuthorityEvidenceProbe{complete},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReconciliationAuthorityStatus != actorContextR3AWStatusQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{complete.EvidenceID}) ||
		!reflect.DeepEqual(got.ReconciliationLocusRefs, []string{complete.ReconciliationLocusRef}) ||
		!reflect.DeepEqual(got.RevisionAcceptanceLocusRefs, []string{complete.RevisionAcceptanceLocusRef}) ||
		!reflect.DeepEqual(got.AuthorityApplicationLocusRefs, []string{complete.AuthorityApplicationLocusRef}) ||
		got.PreWriteDecisionRevision != attempt.ObservedDecisionRevision ||
		got.PostWriteDecisionRevision != observation.PostWriteObservedDecisionRevision {
		t.Fatalf("R3AW qualified Gate lost exact reconciliation boundary data: %+v", got)
	}
}

func TestActorContextR3AWWrongExactChainBindingDoesNotQualify(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		true,
	)
	base := actorContextR3AWCompleteEvidence(
		"test-only:r3aw/base",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)

	cases := map[string]func(*actorContextR3AWReconciliationAuthorityEvidenceProbe){
		"outcome-candidate": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.OutcomeAcceptanceCandidateIdentities = []string{outcomeCandidate.OutcomeAcceptanceCandidateIdentity + ":other"}
		},
		"observation": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.ObservationIdentities = []string{observation.ObservationIdentity + ":other"}
		},
		"attempt": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.WriteAttemptCandidateIdentities = []string{attempt.WriteAttemptCandidateIdentity + ":other"}
		},
		"commit-candidate": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.CommitCandidateIdentities = []string{commitCandidate.CommitCandidateIdentity + ":other"}
		},
		"sink": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"pre-revision": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.PreWriteDecisionRevisions = []string{attempt.ObservedDecisionRevision + ":other"}
		},
		"post-observation": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.PostWriteRevisionObservationRefs = []string{observation.PostWriteRevisionObservationRef + ":other"}
		},
		"post-revision": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.PostWriteDecisionRevisions = []string{observation.PostWriteObservedDecisionRevision + ":other"}
		},
		"scope": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.ScopeValue = "other-procedure/BP-99"
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3AWCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3AWAssessReconciliationAuthority(
				commitCandidate,
				attempt,
				observation,
				outcomeCandidate,
				businessAddress,
				[]actorContextR3AWReconciliationAuthorityEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ReconciliationAuthorityStatus != actorContextR3AWStatusUnresolved ||
				len(got.QualifiedEvidenceIDs) != 0 {
				t.Fatalf("R3AW qualified mismatched %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3AWMissingLocusOrProvenanceDoesNotQualify(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		true,
	)
	base := actorContextR3AWCompleteEvidence(
		"test-only:r3aw/base",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)
	cases := map[string]func(*actorContextR3AWReconciliationAuthorityEvidenceProbe){
		"repository": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.RepositoryBacked = false
		},
		"authority-kind": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.AuthorityKind = ""
		},
		"declaration": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.AuthorityDeclarationRef = ""
		},
		"reconciliation-locus": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.ReconciliationLocusRef = ""
		},
		"revision-acceptance-locus": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.RevisionAcceptanceLocusRef = ""
		},
		"authority-application-locus": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.AuthorityApplicationLocusRef = ""
		},
		"provenance": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.ReconciliationProvenanceRef = ""
		},
		"derived": func(v *actorContextR3AWReconciliationAuthorityEvidenceProbe) {
			v.DerivedOnly = true
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3AWCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3AWAssessReconciliationAuthority(
				commitCandidate,
				attempt,
				observation,
				outcomeCandidate,
				businessAddress,
				[]actorContextR3AWReconciliationAuthorityEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ReconciliationAuthorityStatus != actorContextR3AWStatusUnresolved {
				t.Fatalf("R3AW qualified incomplete %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3AWChangedOrUnchangedRevisionDoesNotInferAuthority(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprintf("changed=%t", changed), func(t *testing.T) {
			commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
				t,
				actorContextR3ATReportedApplied,
				actorContextR3AVDecisionAccept,
				changed,
			)
			got, err := actorContextR3AWAssessReconciliationAuthority(
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
			if got.ReconciliationAuthorityStatus != actorContextR3AWStatusUnresolved {
				t.Fatalf("R3AW inferred reconciliation authority from revision relation changed=%t: %+v", changed, got)
			}
		})
	}
}

func TestActorContextR3AWRawWriterOutcomeDoesNotInferReconciliationAuthority(t *testing.T) {
	for _, outcome := range []string{
		actorContextR3ATReportedApplied,
		actorContextR3ATReportedNotApplied,
		actorContextR3ATReportedConflict,
		actorContextR3ATReportedUnknown,
	} {
		t.Run(outcome, func(t *testing.T) {
			commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
				t,
				outcome,
				actorContextR3AVDecisionAccept,
				true,
			)
			got, err := actorContextR3AWAssessReconciliationAuthority(
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
			if got.ReconciliationAuthorityStatus != actorContextR3AWStatusUnresolved {
				t.Fatalf("R3AW inferred authority from raw writer outcome %q: %+v", outcome, got)
			}
		})
	}
}

func TestActorContextR3AWMultipleQualifiedKeepsPrecedenceUnresolved(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedApplied,
		actorContextR3AVDecisionAccept,
		true,
	)
	a := actorContextR3AWCompleteEvidence(
		"test-only:r3aw/a",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)
	b := actorContextR3AWCompleteEvidence(
		"test-only:r3aw/b",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)

	got, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
		businessAddress,
		[]actorContextR3AWReconciliationAuthorityEvidenceProbe{b, a},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReconciliationAuthorityStatus != actorContextR3AWStatusMultipleQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{a.EvidenceID, b.EvidenceID}) {
		t.Fatalf("R3AW selected winner among multiple reconciliation authorities: %+v", got)
	}
}

func TestActorContextR3AWInputOrderDoesNotChangeGateResult(t *testing.T) {
	commitCandidate, attempt, observation, outcomeCandidate, businessAddress := actorContextR3AWOutcomeCandidateFixture(
		t,
		actorContextR3ATReportedUnknown,
		actorContextR3AVDecisionAccept,
		true,
	)
	qualified := actorContextR3AWCompleteEvidence(
		"test-only:r3aw/qualified",
		businessAddress,
		commitCandidate,
		attempt,
		observation,
		outcomeCandidate,
	)
	insufficient := actorContextR3AWCloneEvidence(qualified)
	insufficient.EvidenceID = "test-only:r3aw/insufficient"
	insufficient.RevisionAcceptanceLocusRef = ""

	first, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate, attempt, observation, outcomeCandidate, businessAddress,
		[]actorContextR3AWReconciliationAuthorityEvidenceProbe{qualified, insufficient},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AWAssessReconciliationAuthority(
		commitCandidate, attempt, observation, outcomeCandidate, businessAddress,
		[]actorContextR3AWReconciliationAuthorityEvidenceProbe{insufficient, qualified},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AW input order changed Gate result: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3AWShapeHasNoSynchronizedDecisionAuthorityOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AWReconciliationAuthorityEvidenceProbe{}),
		reflect.TypeOf(actorContextR3AWReconciliationAuthorityGateProbe{}),
	} {
		for _, forbidden := range []string{
			"StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedWriteOutcome", "OutcomeAccepted", "AcceptedDecision", "CommittedDecision",
			"CurrentAuthority", "AppliedAuthority", "RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "RetrySafe", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AW Gate introduced forbidden synchronization/authority shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AWOutcomeCandidateFixture(
	t *testing.T,
	rawOutcome string,
	decision string,
	postRevisionChanged bool,
) (
	actorContextR3AQCommitCandidateProbe,
	actorContextR3ASWriteAttemptCandidateProbe,
	actorContextR3ATNormalizedObservationProbe,
	actorContextR3AVOutcomeAcceptanceCandidateProbe,
	string,
) {
	t.Helper()

	commitCandidate, businessAddress, appGate, appEvidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	attemptWitness := actorContextR3ASExactWitness(commitCandidate, appEvidence[0], revision)
	attempts, err := actorContextR3ASBuildWriteAttemptCandidates(
		commitCandidate,
		businessAddress,
		appGate,
		appEvidence,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{attemptWitness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(attempts) != 1 {
		t.Fatalf("R3AW setup write-attempt candidate count=%d want=1", len(attempts))
	}
	attempt := attempts[0]

	raw := actorContextR3ATExactObservation(attempt, rawOutcome)
	if postRevisionChanged {
		raw.PostWriteObservedDecisionRevision += ":post-write"
	}
	observations, err := actorContextR3ATNormalizeObservations(
		attempt,
		[]actorContextR3ATWriterObservationProbe{raw},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 {
		t.Fatalf("R3AW setup normalized observation count=%d want=1", len(observations))
	}
	observation := observations[0]

	outcomeAuthority := actorContextR3AUCompleteEvidence(
		"test-only:r3aw/outcome-acceptance-authority",
		businessAddress,
		attempt,
		observation,
	)
	outcomeGate, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt,
		observation,
		businessAddress,
		[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{outcomeAuthority},
	)
	if err != nil {
		t.Fatal(err)
	}
	witness := actorContextR3AVOutcomeAcceptanceWitnessProbe{
		ObservationIdentity:                  observation.ObservationIdentity,
		OutcomeAcceptanceAuthorityEvidenceID: outcomeAuthority.EvidenceID,
		Decision:                             decision,
		AcceptanceProvenanceRef:              "test-only:r3aw/outcome-acceptance-witness",
	}
	candidates, err := actorContextR3AVBuildCandidates(
		attempt,
		observation,
		businessAddress,
		outcomeGate,
		[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{outcomeAuthority},
		[]actorContextR3AVOutcomeAcceptanceWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("R3AW setup outcome-acceptance candidate count=%d want=1", len(candidates))
	}
	return commitCandidate, attempt, observation, candidates[0], businessAddress
}

func actorContextR3AWCurrentInspectedEvidence(
	commitCandidate actorContextR3AQCommitCandidateProbe,
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	outcomeCandidate actorContextR3AVOutcomeAcceptanceCandidateProbe,
) []actorContextR3AWReconciliationAuthorityEvidenceProbe {
	return []actorContextR3AWReconciliationAuthorityEvidenceProbe{
		{
			EvidenceID:                           outcomeCandidate.OutcomeAcceptanceCandidateIdentity,
			EvidenceRole:                         actorContextR3AWRoleDerivedOutcomeCandidate,
			OutcomeAcceptanceCandidateIdentities: []string{outcomeCandidate.OutcomeAcceptanceCandidateIdentity},
			ObservationIdentities:                []string{observation.ObservationIdentity},
			WriteAttemptCandidateIdentities:      []string{attempt.WriteAttemptCandidateIdentity},
			CommitCandidateIdentities:            []string{commitCandidate.CommitCandidateIdentity},
			AcceptedDecisionSinkRef:              attempt.AcceptedDecisionSinkRef,
			PreWriteDecisionRevisions:            []string{attempt.ObservedDecisionRevision},
			PostWriteRevisionObservationRefs:     []string{observation.PostWriteRevisionObservationRef},
			PostWriteDecisionRevisions:           []string{observation.PostWriteObservedDecisionRevision},
			DerivedOnly:                          true,
		},
		{
			EvidenceID:                 commitCandidate.WriteAuthorityRevisionAuthorityRef,
			EvidenceRole:               actorContextR3AWRoleRevisionAuthorityPrecedent,
			RepositoryBacked:           true,
			RevisionAuthorityRef:       commitCandidate.WriteAuthorityRevisionAuthorityRef,
			AcceptedDecisionSinkRef:    commitCandidate.AcceptedDecisionSinkRef,
			PreWriteDecisionRevisions:  []string{attempt.ObservedDecisionRevision},
			PostWriteDecisionRevisions: []string{observation.PostWriteObservedDecisionRevision},
		},
	}
}

func actorContextR3AWCompleteEvidence(
	id string,
	businessAddress string,
	commitCandidate actorContextR3AQCommitCandidateProbe,
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	outcomeCandidate actorContextR3AVOutcomeAcceptanceCandidateProbe,
) actorContextR3AWReconciliationAuthorityEvidenceProbe {
	return actorContextR3AWReconciliationAuthorityEvidenceProbe{
		EvidenceID:                           id,
		EvidenceRole:                         actorContextR3AWRoleReconciliationAuthorityCandidate,
		RepositoryBacked:                     true,
		AuthorityKind:                        actorContextR3AWAuthorityKind,
		AuthorityDeclarationRef:              id + "/authority-declaration",
		OutcomeAcceptanceCandidateIdentities: []string{outcomeCandidate.OutcomeAcceptanceCandidateIdentity},
		ObservationIdentities:                []string{observation.ObservationIdentity},
		WriteAttemptCandidateIdentities:      []string{attempt.WriteAttemptCandidateIdentity},
		CommitCandidateIdentities:            []string{commitCandidate.CommitCandidateIdentity},
		AcceptedDecisionSinkRef:              attempt.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:                 commitCandidate.WriteAuthorityRevisionAuthorityRef,
		PreWriteDecisionRevisions:            []string{attempt.ObservedDecisionRevision},
		PostWriteRevisionObservationRefs:     []string{observation.PostWriteRevisionObservationRef},
		PostWriteDecisionRevisions:           []string{observation.PostWriteObservedDecisionRevision},
		ScopeKind:                            actorContextR3AWScopeExactBusinessAddress,
		ScopeValue:                           businessAddress,
		ReconciliationLocusRef:               id + "/reconciliation-locus",
		RevisionAcceptanceLocusRef:           id + "/revision-acceptance-locus",
		AuthorityApplicationLocusRef:         id + "/authority-application-locus",
		ReconciliationProvenanceRef:          id + "/reconciliation-provenance",
	}
}

func actorContextR3AWCloneEvidence(
	value actorContextR3AWReconciliationAuthorityEvidenceProbe,
) actorContextR3AWReconciliationAuthorityEvidenceProbe {
	out := value
	out.OutcomeAcceptanceCandidateIdentities = append([]string(nil), value.OutcomeAcceptanceCandidateIdentities...)
	out.ObservationIdentities = append([]string(nil), value.ObservationIdentities...)
	out.WriteAttemptCandidateIdentities = append([]string(nil), value.WriteAttemptCandidateIdentities...)
	out.CommitCandidateIdentities = append([]string(nil), value.CommitCandidateIdentities...)
	out.PreWriteDecisionRevisions = append([]string(nil), value.PreWriteDecisionRevisions...)
	out.PostWriteRevisionObservationRefs = append([]string(nil), value.PostWriteRevisionObservationRefs...)
	out.PostWriteDecisionRevisions = append([]string(nil), value.PostWriteDecisionRevisions...)
	return out
}

func actorContextR3AWAssessReconciliationAuthority(
	commitCandidate actorContextR3AQCommitCandidateProbe,
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	outcomeCandidate actorContextR3AVOutcomeAcceptanceCandidateProbe,
	businessAddress string,
	evidence []actorContextR3AWReconciliationAuthorityEvidenceProbe,
) (actorContextR3AWReconciliationAuthorityGateProbe, error) {
	if commitCandidate.CommitCandidateIdentity == "" ||
		commitCandidate.AcceptedDecisionSinkRef == "" ||
		commitCandidate.WriteAuthorityRevisionAuthorityRef == "" ||
		attempt.CommitCandidateIdentity != commitCandidate.CommitCandidateIdentity ||
		attempt.AcceptedDecisionSinkRef != commitCandidate.AcceptedDecisionSinkRef ||
		attempt.WriteAttemptCandidateIdentity == "" ||
		attempt.ObservedDecisionRevision == "" ||
		observation.ObservationIdentity == "" ||
		observation.WriteAttemptCandidateIdentity != attempt.WriteAttemptCandidateIdentity ||
		observation.AcceptedDecisionSinkRef != attempt.AcceptedDecisionSinkRef ||
		observation.PostWriteRevisionObservationRef == "" ||
		observation.PostWriteObservedDecisionRevision == "" ||
		outcomeCandidate.OutcomeAcceptanceCandidateIdentity == "" ||
		outcomeCandidate.ObservationIdentity != observation.ObservationIdentity ||
		outcomeCandidate.WriteAttemptCandidateIdentity != attempt.WriteAttemptCandidateIdentity ||
		outcomeCandidate.PostWriteObservedDecisionRevision != observation.PostWriteObservedDecisionRevision ||
		businessAddress == "" {
		return actorContextR3AWReconciliationAuthorityGateProbe{}, fmt.Errorf(
			"R3AW exact post-write reconciliation chain is incomplete or inconsistent",
		)
	}
	if outcomeCandidate.Decision != actorContextR3AVDecisionAccept {
		return actorContextR3AWReconciliationAuthorityGateProbe{}, fmt.Errorf(
			"R3AW reconciliation Gate requires accept-observed-outcome candidate",
		)
	}

	qualified := []actorContextR3AWReconciliationAuthorityEvidenceProbe{}
	insufficient := []string{}
	seenIDs := map[string]bool{}
	for _, value := range evidence {
		if value.EvidenceID == "" {
			return actorContextR3AWReconciliationAuthorityGateProbe{}, fmt.Errorf(
				"R3AW reconciliation authority evidence id is empty",
			)
		}
		if seenIDs[value.EvidenceID] {
			return actorContextR3AWReconciliationAuthorityGateProbe{}, fmt.Errorf(
				"R3AW duplicate reconciliation authority evidence id %q", value.EvidenceID,
			)
		}
		seenIDs[value.EvidenceID] = true

		if actorContextR3AWCandidateQualifies(
			commitCandidate,
			attempt,
			observation,
			outcomeCandidate,
			businessAddress,
			value,
		) {
			qualified = append(qualified, value)
		} else {
			insufficient = append(insufficient, value.EvidenceID)
		}
	}

	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].EvidenceID < qualified[j].EvidenceID
	})
	status := actorContextR3AWStatusUnresolved
	if len(qualified) == 1 {
		status = actorContextR3AWStatusQualified
	} else if len(qualified) > 1 {
		status = actorContextR3AWStatusMultipleQualified
	}

	ids := []string{}
	reconciliationLoci := []string{}
	revisionAcceptanceLoci := []string{}
	applicationLoci := []string{}
	sinks := []string{}
	for _, value := range qualified {
		ids = append(ids, value.EvidenceID)
		reconciliationLoci = append(reconciliationLoci, value.ReconciliationLocusRef)
		revisionAcceptanceLoci = append(revisionAcceptanceLoci, value.RevisionAcceptanceLocusRef)
		applicationLoci = append(applicationLoci, value.AuthorityApplicationLocusRef)
		sinks = append(sinks, value.AcceptedDecisionSinkRef)
	}

	return actorContextR3AWReconciliationAuthorityGateProbe{
		ReconciliationAuthorityStatus: status,
		QualifiedEvidenceIDs:          actorContextR0SortedUnique(ids),
		ReconciliationLocusRefs:       actorContextR0SortedUnique(reconciliationLoci),
		RevisionAcceptanceLocusRefs:   actorContextR0SortedUnique(revisionAcceptanceLoci),
		AuthorityApplicationLocusRefs: actorContextR0SortedUnique(applicationLoci),
		AcceptedDecisionSinkRefs:      actorContextR0SortedUnique(sinks),
		PreWriteDecisionRevision:      attempt.ObservedDecisionRevision,
		PostWriteDecisionRevision:     observation.PostWriteObservedDecisionRevision,
		InsufficientEvidenceIDs:       actorContextR0SortedUnique(insufficient),
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"accepted-outcome-candidate-does-not-synchronize-authoritative-state",
			"revision-change-or-nonchange-does-not-select-reconciliation-authority",
			"reconciliation-locus-does-not-execute-reconciliation",
			"revision-acceptance-locus-does-not-accept-or-apply-revision",
			"authority-application-locus-does-not-apply-current-authority",
			"qualified-reconciliation-gate-does-not-create-accepted-decision",
			"multiple-qualified-reconciliation-authorities-have-no-implicit-precedence",
		}),
	}, nil
}

func actorContextR3AWCandidateQualifies(
	commitCandidate actorContextR3AQCommitCandidateProbe,
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	outcomeCandidate actorContextR3AVOutcomeAcceptanceCandidateProbe,
	businessAddress string,
	value actorContextR3AWReconciliationAuthorityEvidenceProbe,
) bool {
	if value.EvidenceRole != actorContextR3AWRoleReconciliationAuthorityCandidate ||
		!value.RepositoryBacked ||
		value.DerivedOnly ||
		value.AuthorityKind != actorContextR3AWAuthorityKind ||
		value.AuthorityDeclarationRef == "" ||
		!actorContextR0Contains(value.OutcomeAcceptanceCandidateIdentities, outcomeCandidate.OutcomeAcceptanceCandidateIdentity) ||
		!actorContextR0Contains(value.ObservationIdentities, observation.ObservationIdentity) ||
		!actorContextR0Contains(value.WriteAttemptCandidateIdentities, attempt.WriteAttemptCandidateIdentity) ||
		!actorContextR0Contains(value.CommitCandidateIdentities, commitCandidate.CommitCandidateIdentity) ||
		value.AcceptedDecisionSinkRef != attempt.AcceptedDecisionSinkRef ||
		value.RevisionAuthorityRef != commitCandidate.WriteAuthorityRevisionAuthorityRef ||
		!actorContextR0Contains(value.PreWriteDecisionRevisions, attempt.ObservedDecisionRevision) ||
		!actorContextR0Contains(value.PostWriteRevisionObservationRefs, observation.PostWriteRevisionObservationRef) ||
		!actorContextR0Contains(value.PostWriteDecisionRevisions, observation.PostWriteObservedDecisionRevision) ||
		value.ReconciliationLocusRef == "" ||
		value.RevisionAcceptanceLocusRef == "" ||
		value.AuthorityApplicationLocusRef == "" ||
		value.ReconciliationProvenanceRef == "" {
		return false
	}

	switch value.ScopeKind {
	case actorContextR3AWScopeExactBusinessAddress:
		return value.ScopeValue == businessAddress
	case actorContextR3AWScopeAnyBusinessAddress:
		return true
	default:
		return false
	}
}

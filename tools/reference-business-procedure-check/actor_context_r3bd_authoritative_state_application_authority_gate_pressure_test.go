package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3BDStatusUnresolved        = "authoritative-state-application-authority-unresolved"
	actorContextR3BDStatusQualified         = "authoritative-state-application-authority-evidence-qualified"
	actorContextR3BDStatusMultipleQualified = "authoritative-state-application-authority-multiple-qualified-precedence-unresolved"

	actorContextR3BDAuthorityKind = "operational-authoritative-state-application-authority"

	actorContextR3BDRoleDerivedResultCandidate             = "derived-reconciliation-result-acceptance-candidate"
	actorContextR3BDRoleStateApplicationAuthorityCandidate = "authoritative-state-application-authority-candidate"

	actorContextR3BDScopeExactBusinessAddress = "exact-business-address"
	actorContextR3BDScopeAnyBusinessAddress   = "any-business-address"
)

type actorContextR3BDStateApplicationAuthorityEvidenceProbe struct {
	EvidenceID                                        string
	EvidenceRole                                      string
	RepositoryBacked                                  bool
	AuthorityKind                                     string
	AuthorityDeclarationRef                           string
	ReconciliationCandidateIdentities                 []string
	ReconciliationResultAcceptanceCandidateIdentities []string
	AcceptedDecisionSinkRef                           string
	RevisionAuthorityRef                              string
	ObservedDecisionRevisions                         []string
	ObservedStateFingerprints                         []string
	ScopeKind                                         string
	ScopeValue                                        string
	StateApplicationLocusRef                          string
	RevisionAcceptanceLocusRef                        string
	CurrentAuthorityProjectionLocusRef                string
	StateApplicationProvenanceRef                     string
	DerivedOnly                                       bool
}

type actorContextR3BDStateApplicationAuthorityGateProbe struct {
	StateApplicationAuthorityStatus     string
	QualifiedEvidenceIDs                []string
	StateApplicationLocusRefs           []string
	RevisionAcceptanceLocusRefs         []string
	CurrentAuthorityProjectionLocusRefs []string
	AcceptedDecisionSinkRefs            []string
	ObservedDecisionRevision            string
	ObservedStateFingerprint            string
	InsufficientEvidenceIDs             []string
	AuthorityLimits                     []string
}

func TestActorContextR3BDCurrentInspectedEvidenceKeepsAuthorityUnresolved(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	got, err := actorContextR3BDAssessStateApplicationAuthority(
		reconciliation, result, businessAddress,
		actorContextR3BDCurrentInspectedEvidence(reconciliation, result),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.StateApplicationAuthorityStatus != actorContextR3BDStatusUnresolved ||
		len(got.QualifiedEvidenceIDs) != 0 {
		t.Fatalf("R3BD fabricated state application authority from current evidence: %+v", got)
	}
}

func TestActorContextR3BDRejectResultCandidateIsNotApplicationInput(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionReject)
	if _, err := actorContextR3BDAssessStateApplicationAuthority(
		reconciliation, result, businessAddress, nil,
	); err == nil {
		t.Fatal("R3BD treated reject reconciliation-result candidate as state-application input")
	}
}

func TestActorContextR3BDCompleteSyntheticEvidenceQualifiesGateOnly(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	complete := actorContextR3BDCompleteEvidence(
		"test-only:r3bd/state-application-authority",
		businessAddress, reconciliation, result,
	)
	got, err := actorContextR3BDAssessStateApplicationAuthority(
		reconciliation, result, businessAddress,
		[]actorContextR3BDStateApplicationAuthorityEvidenceProbe{complete},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.StateApplicationAuthorityStatus != actorContextR3BDStatusQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{complete.EvidenceID}) ||
		!reflect.DeepEqual(got.StateApplicationLocusRefs, []string{complete.StateApplicationLocusRef}) ||
		!reflect.DeepEqual(got.RevisionAcceptanceLocusRefs, []string{complete.RevisionAcceptanceLocusRef}) ||
		!reflect.DeepEqual(got.CurrentAuthorityProjectionLocusRefs, []string{complete.CurrentAuthorityProjectionLocusRef}) ||
		got.ObservedDecisionRevision != result.ObservedDecisionRevision ||
		got.ObservedStateFingerprint != result.ObservedStateFingerprint {
		t.Fatalf("R3BD qualified Gate lost exact application context: %+v", got)
	}
}

func TestActorContextR3BDWrongExactBindingDoesNotQualify(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	base := actorContextR3BDCompleteEvidence("test-only:r3bd/base", businessAddress, reconciliation, result)
	cases := map[string]func(*actorContextR3BDStateApplicationAuthorityEvidenceProbe){
		"reconciliation": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.ReconciliationCandidateIdentities = []string{reconciliation.ReconciliationCandidateIdentity + ":other"}
		},
		"result": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.ReconciliationResultAcceptanceCandidateIdentities = []string{result.ReconciliationResultAcceptanceCandidateIdentity + ":other"}
		},
		"sink": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"revision": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.ObservedDecisionRevisions = []string{result.ObservedDecisionRevision + ":other"}
		},
		"fingerprint": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.ObservedStateFingerprints = []string{result.ObservedStateFingerprint + ":other"}
		},
		"scope": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.ScopeValue = "other-procedure/BP-99"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3BDCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3BDAssessStateApplicationAuthority(
				reconciliation, result, businessAddress,
				[]actorContextR3BDStateApplicationAuthorityEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.StateApplicationAuthorityStatus != actorContextR3BDStatusUnresolved {
				t.Fatalf("R3BD qualified mismatched %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3BDMissingLocusDoesNotQualify(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	base := actorContextR3BDCompleteEvidence("test-only:r3bd/base", businessAddress, reconciliation, result)
	cases := map[string]func(*actorContextR3BDStateApplicationAuthorityEvidenceProbe){
		"application": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.StateApplicationLocusRef = ""
		},
		"revision-acceptance": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.RevisionAcceptanceLocusRef = ""
		},
		"authority-projection": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.CurrentAuthorityProjectionLocusRef = ""
		},
		"provenance": func(v *actorContextR3BDStateApplicationAuthorityEvidenceProbe) {
			v.StateApplicationProvenanceRef = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3BDCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3BDAssessStateApplicationAuthority(
				reconciliation, result, businessAddress,
				[]actorContextR3BDStateApplicationAuthorityEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.StateApplicationAuthorityStatus != actorContextR3BDStatusUnresolved {
				t.Fatalf("R3BD qualified evidence without %s locus/provenance: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3BDMultipleQualifiedKeepsPrecedenceUnresolved(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	a := actorContextR3BDCompleteEvidence("test-only:r3bd/a", businessAddress, reconciliation, result)
	b := actorContextR3BDCompleteEvidence("test-only:r3bd/b", businessAddress, reconciliation, result)
	got, err := actorContextR3BDAssessStateApplicationAuthority(
		reconciliation, result, businessAddress,
		[]actorContextR3BDStateApplicationAuthorityEvidenceProbe{b, a},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.StateApplicationAuthorityStatus != actorContextR3BDStatusMultipleQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{a.EvidenceID, b.EvidenceID}) {
		t.Fatalf("R3BD selected a state-application authority winner: %+v", got)
	}
}

func TestActorContextR3BDInputOrderDoesNotChangeGateResult(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	qualified := actorContextR3BDCompleteEvidence("test-only:r3bd/qualified", businessAddress, reconciliation, result)
	insufficient := actorContextR3BDCloneEvidence(qualified)
	insufficient.EvidenceID = "test-only:r3bd/insufficient"
	insufficient.CurrentAuthorityProjectionLocusRef = ""

	first, err := actorContextR3BDAssessStateApplicationAuthority(
		reconciliation, result, businessAddress,
		[]actorContextR3BDStateApplicationAuthorityEvidenceProbe{qualified, insufficient},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3BDAssessStateApplicationAuthority(
		reconciliation, result, businessAddress,
		[]actorContextR3BDStateApplicationAuthorityEvidenceProbe{insufficient, qualified},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3BD input order changed Gate result: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3BDShapeHasNoStateApplicationCurrentAuthorityOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3BDStateApplicationAuthorityEvidenceProbe{}),
		reflect.TypeOf(actorContextR3BDStateApplicationAuthorityGateProbe{}),
	} {
		for _, forbidden := range []string{
			"StateApplied", "StateSynchronized", "Reconciled",
			"RevisionAccepted", "RevisionApplied", "AcceptedDecision", "CurrentDecision",
			"CurrentAuthority", "AppliedAuthority", "AuthoritativeState",
			"RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3BD Gate introduced forbidden state-application shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3BDResultCandidateFixture(
	t *testing.T,
	decision string,
) (
	actorContextR3AXReconciliationCandidateProbe,
	actorContextR3BCResultAcceptanceCandidateProbe,
	string,
) {
	t.Helper()
	reconciliation, execution, current, gate, evidence := actorContextR3BCSingleQualifiedFixture(t)
	witness := actorContextR3BCExactWitness(reconciliation, execution, current, gate, decision)
	candidates, err := actorContextR3BCBuildCandidates(
		reconciliation, execution, current, gate, evidence,
		[]actorContextR3BCResultAcceptanceWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("R3BD setup result candidate count=%d want=1", len(candidates))
	}
	return reconciliation, candidates[0], "order-entry.order-handling/BP-06"
}

func actorContextR3BDCurrentInspectedEvidence(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	result actorContextR3BCResultAcceptanceCandidateProbe,
) []actorContextR3BDStateApplicationAuthorityEvidenceProbe {
	return []actorContextR3BDStateApplicationAuthorityEvidenceProbe{{
		EvidenceID:                        result.ReconciliationResultAcceptanceCandidateIdentity,
		EvidenceRole:                      actorContextR3BDRoleDerivedResultCandidate,
		ReconciliationCandidateIdentities: []string{reconciliation.ReconciliationCandidateIdentity},
		ReconciliationResultAcceptanceCandidateIdentities: []string{result.ReconciliationResultAcceptanceCandidateIdentity},
		AcceptedDecisionSinkRef:                           reconciliation.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:                              reconciliation.RevisionAuthorityRef,
		ObservedDecisionRevisions:                         []string{result.ObservedDecisionRevision},
		ObservedStateFingerprints:                         []string{result.ObservedStateFingerprint},
		StateApplicationProvenanceRef:                     result.ResultAcceptanceProvenanceRef,
		DerivedOnly:                                       true,
	}}
}

func actorContextR3BDCompleteEvidence(
	id string,
	businessAddress string,
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	result actorContextR3BCResultAcceptanceCandidateProbe,
) actorContextR3BDStateApplicationAuthorityEvidenceProbe {
	return actorContextR3BDStateApplicationAuthorityEvidenceProbe{
		EvidenceID:                        id,
		EvidenceRole:                      actorContextR3BDRoleStateApplicationAuthorityCandidate,
		RepositoryBacked:                  true,
		AuthorityKind:                     actorContextR3BDAuthorityKind,
		AuthorityDeclarationRef:           id + "/authority-declaration",
		ReconciliationCandidateIdentities: []string{reconciliation.ReconciliationCandidateIdentity},
		ReconciliationResultAcceptanceCandidateIdentities: []string{result.ReconciliationResultAcceptanceCandidateIdentity},
		AcceptedDecisionSinkRef:                           reconciliation.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:                              reconciliation.RevisionAuthorityRef,
		ObservedDecisionRevisions:                         []string{result.ObservedDecisionRevision},
		ObservedStateFingerprints:                         []string{result.ObservedStateFingerprint},
		ScopeKind:                                         actorContextR3BDScopeExactBusinessAddress,
		ScopeValue:                                        businessAddress,
		StateApplicationLocusRef:                          id + "/state-application-locus",
		RevisionAcceptanceLocusRef:                        id + "/revision-acceptance-locus",
		CurrentAuthorityProjectionLocusRef:                id + "/current-authority-projection-locus",
		StateApplicationProvenanceRef:                     id + "/state-application-provenance",
	}
}

func actorContextR3BDCloneEvidence(
	value actorContextR3BDStateApplicationAuthorityEvidenceProbe,
) actorContextR3BDStateApplicationAuthorityEvidenceProbe {
	out := value
	out.ReconciliationCandidateIdentities = append([]string(nil), value.ReconciliationCandidateIdentities...)
	out.ReconciliationResultAcceptanceCandidateIdentities = append([]string(nil), value.ReconciliationResultAcceptanceCandidateIdentities...)
	out.ObservedDecisionRevisions = append([]string(nil), value.ObservedDecisionRevisions...)
	out.ObservedStateFingerprints = append([]string(nil), value.ObservedStateFingerprints...)
	return out
}

func actorContextR3BDAssessStateApplicationAuthority(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	result actorContextR3BCResultAcceptanceCandidateProbe,
	businessAddress string,
	evidence []actorContextR3BDStateApplicationAuthorityEvidenceProbe,
) (actorContextR3BDStateApplicationAuthorityGateProbe, error) {
	if reconciliation.ReconciliationCandidateIdentity == "" ||
		result.ReconciliationResultAcceptanceCandidateIdentity == "" ||
		result.ReconciliationCandidateIdentity != reconciliation.ReconciliationCandidateIdentity ||
		result.Decision != actorContextR3BCDecisionAccept ||
		reconciliation.AcceptedDecisionSinkRef == "" ||
		reconciliation.RevisionAuthorityRef == "" ||
		result.ObservedDecisionRevision == "" ||
		result.ObservedStateFingerprint == "" ||
		businessAddress == "" {
		return actorContextR3BDStateApplicationAuthorityGateProbe{}, fmt.Errorf(
			"R3BD reconciliation-result/application context is incomplete or not applicable",
		)
	}

	qualified := []actorContextR3BDStateApplicationAuthorityEvidenceProbe{}
	insufficient := []string{}
	seen := map[string]bool{}
	for _, value := range evidence {
		if value.EvidenceID == "" || value.EvidenceRole == "" {
			return actorContextR3BDStateApplicationAuthorityGateProbe{}, fmt.Errorf("R3BD evidence identity/role is incomplete")
		}
		if seen[value.EvidenceID] {
			return actorContextR3BDStateApplicationAuthorityGateProbe{}, fmt.Errorf("R3BD duplicate EvidenceID %q", value.EvidenceID)
		}
		seen[value.EvidenceID] = true
		if actorContextR3BDCandidateQualifies(reconciliation, result, businessAddress, value) {
			qualified = append(qualified, value)
		} else {
			insufficient = append(insufficient, value.EvidenceID)
		}
	}
	sort.Slice(qualified, func(i, j int) bool { return qualified[i].EvidenceID < qualified[j].EvidenceID })

	status := actorContextR3BDStatusUnresolved
	if len(qualified) == 1 {
		status = actorContextR3BDStatusQualified
	}
	if len(qualified) > 1 {
		status = actorContextR3BDStatusMultipleQualified
	}

	ids := []string{}
	appRefs := []string{}
	revRefs := []string{}
	authRefs := []string{}
	sinks := []string{}
	for _, value := range qualified {
		ids = append(ids, value.EvidenceID)
		appRefs = append(appRefs, value.StateApplicationLocusRef)
		revRefs = append(revRefs, value.RevisionAcceptanceLocusRef)
		authRefs = append(authRefs, value.CurrentAuthorityProjectionLocusRef)
		sinks = append(sinks, value.AcceptedDecisionSinkRef)
	}
	return actorContextR3BDStateApplicationAuthorityGateProbe{
		StateApplicationAuthorityStatus:     status,
		QualifiedEvidenceIDs:                actorContextR0SortedUnique(ids),
		StateApplicationLocusRefs:           actorContextR0SortedUnique(appRefs),
		RevisionAcceptanceLocusRefs:         actorContextR0SortedUnique(revRefs),
		CurrentAuthorityProjectionLocusRefs: actorContextR0SortedUnique(authRefs),
		AcceptedDecisionSinkRefs:            actorContextR0SortedUnique(sinks),
		ObservedDecisionRevision:            result.ObservedDecisionRevision,
		ObservedStateFingerprint:            result.ObservedStateFingerprint,
		InsufficientEvidenceIDs:             actorContextR0SortedUnique(insufficient),
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"result-acceptance-candidate-does-not-create-state-application-authority",
			"observed-revision-and-fingerprint-do-not-create-state-application-authority",
			"qualified-state-application-authority-does-not-apply-state",
			"revision-acceptance-locus-does-not-mean-revision-accepted",
			"current-authority-projection-locus-does-not-mean-current-authority-applied",
			"reject-result-candidate-does-not-create-retry-rollback-or-compensation-authority",
		}),
	}, nil
}

func actorContextR3BDCandidateQualifies(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	result actorContextR3BCResultAcceptanceCandidateProbe,
	businessAddress string,
	value actorContextR3BDStateApplicationAuthorityEvidenceProbe,
) bool {
	if value.EvidenceRole != actorContextR3BDRoleStateApplicationAuthorityCandidate ||
		!value.RepositoryBacked || value.DerivedOnly ||
		value.AuthorityKind != actorContextR3BDAuthorityKind ||
		value.AuthorityDeclarationRef == "" ||
		!actorContextR0Contains(value.ReconciliationCandidateIdentities, reconciliation.ReconciliationCandidateIdentity) ||
		!actorContextR0Contains(value.ReconciliationResultAcceptanceCandidateIdentities, result.ReconciliationResultAcceptanceCandidateIdentity) ||
		value.AcceptedDecisionSinkRef != reconciliation.AcceptedDecisionSinkRef ||
		value.RevisionAuthorityRef != reconciliation.RevisionAuthorityRef ||
		!actorContextR0Contains(value.ObservedDecisionRevisions, result.ObservedDecisionRevision) ||
		!actorContextR0Contains(value.ObservedStateFingerprints, result.ObservedStateFingerprint) ||
		value.StateApplicationLocusRef == "" ||
		value.RevisionAcceptanceLocusRef == "" ||
		value.CurrentAuthorityProjectionLocusRef == "" ||
		value.StateApplicationProvenanceRef == "" {
		return false
	}
	switch value.ScopeKind {
	case actorContextR3BDScopeExactBusinessAddress:
		return value.ScopeValue == businessAddress
	case actorContextR3BDScopeAnyBusinessAddress:
		return true
	default:
		return false
	}
}

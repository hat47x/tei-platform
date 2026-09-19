package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3BBStatusUnresolved =
		"reconciliation-result-acceptance-authority-unresolved"
	actorContextR3BBStatusQualified =
		"reconciliation-result-acceptance-authority-evidence-qualified"
	actorContextR3BBStatusMultipleQualified =
		"reconciliation-result-acceptance-authority-multiple-qualified-precedence-unresolved"

	actorContextR3BBAuthorityKind =
		"operational-reconciliation-result-acceptance-authority"

	actorContextR3BBRoleDerivedExecutionCandidate =
		"derived-execution-evidence-acceptance-candidate"
	actorContextR3BBRoleDerivedCurrentStateCandidate =
		"derived-current-state-evidence-acceptance-candidate"
	actorContextR3BBRoleResultAuthorityCandidate =
		"reconciliation-result-acceptance-authority-candidate"

	actorContextR3BBScopeExactBusinessAddress = "exact-business-address"
	actorContextR3BBScopeAnyBusinessAddress   = "any-business-address"
)

type actorContextR3BBResultAcceptanceAuthorityEvidenceProbe struct {
	EvidenceID                                     string
	EvidenceRole                                   string
	RepositoryBacked                               bool
	AuthorityKind                                  string
	AuthorityDeclarationRef                        string
	ReconciliationCandidateIdentities              []string
	ExecutionEvidenceAcceptanceCandidateIdentities []string
	CurrentStateEvidenceAcceptanceCandidateIdentities []string
	AcceptedDecisionSinkRef                        string
	RevisionAuthorityRef                           string
	ScopeKind                                      string
	ScopeValue                                     string
	ReconciliationResultDecisionLocusRef           string
	CurrentStateAcceptanceLocusRef                 string
	ResultAcceptanceProvenanceRef                  string
	DerivedOnly                                    bool
}

type actorContextR3BBResultAcceptanceAuthorityGateProbe struct {
	ReconciliationResultAcceptanceAuthorityStatus string
	QualifiedEvidenceIDs                          []string
	ReconciliationResultDecisionLocusRefs         []string
	CurrentStateAcceptanceLocusRefs               []string
	ExecutionEvidenceAcceptanceCandidateIdentity  string
	CurrentStateEvidenceAcceptanceCandidateIdentity string
	InsufficientEvidenceIDs                       []string
	AuthorityLimits                               []string
}

func TestActorContextR3BBOnePlaneCannotEnterResultAcceptanceGate(t *testing.T) {
	reconciliation, execution, _, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)

	if _, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution},
		businessAddress,
		nil,
	); err == nil {
		t.Fatal("R3BB accepted one-plane Evidence candidate set")
	}
}

func TestActorContextR3BBRejectCandidateIsNotResultAcceptanceEntry(t *testing.T) {
	reconciliation, execution, current, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)
	current.Decision = actorContextR3BADecisionReject

	if _, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
		businessAddress,
		nil,
	); err == nil {
		t.Fatal("R3BB converted reject plane candidate into reconciliation-result acceptance entry")
	}
}

func TestActorContextR3BBCurrentInspectedCandidatesKeepAuthorityUnresolved(t *testing.T) {
	reconciliation, execution, current, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)
	candidates := []actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current}

	got, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		candidates,
		businessAddress,
		actorContextR3BBCurrentInspectedEvidence(execution, current),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusUnresolved ||
		len(got.QualifiedEvidenceIDs) != 0 ||
		len(got.InsufficientEvidenceIDs) == 0 {
		t.Fatalf("R3BB fabricated result acceptance authority from derived candidates: %+v", got)
	}
}

func TestActorContextR3BBCompleteSyntheticEvidenceQualifiesGateOnly(t *testing.T) {
	reconciliation, execution, current, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)
	evidence := actorContextR3BBCompleteEvidence(
		"test-only:r3bb/result-authority",
		businessAddress,
		reconciliation,
		execution,
		current,
	)

	got, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
		businessAddress,
		[]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{evidence},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{evidence.EvidenceID}) ||
		!reflect.DeepEqual(
			got.ReconciliationResultDecisionLocusRefs,
			[]string{evidence.ReconciliationResultDecisionLocusRef},
		) ||
		!reflect.DeepEqual(
			got.CurrentStateAcceptanceLocusRefs,
			[]string{evidence.CurrentStateAcceptanceLocusRef},
		) ||
		got.ExecutionEvidenceAcceptanceCandidateIdentity != execution.EvidenceAcceptanceCandidateIdentity ||
		got.CurrentStateEvidenceAcceptanceCandidateIdentity != current.EvidenceAcceptanceCandidateIdentity {
		t.Fatalf("R3BB qualified Gate lost exact join boundary: %+v", got)
	}
}

func TestActorContextR3BBWrongExactBindingDoesNotQualify(t *testing.T) {
	reconciliation, execution, current, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)
	base := actorContextR3BBCompleteEvidence(
		"test-only:r3bb/base",
		businessAddress,
		reconciliation,
		execution,
		current,
	)

	cases := map[string]func(*actorContextR3BBResultAcceptanceAuthorityEvidenceProbe){
		"reconciliation-candidate": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.ReconciliationCandidateIdentities = []string{reconciliation.ReconciliationCandidateIdentity + ":other"}
		},
		"execution-candidate": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.ExecutionEvidenceAcceptanceCandidateIdentities = []string{execution.EvidenceAcceptanceCandidateIdentity + ":other"}
		},
		"current-state-candidate": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.CurrentStateEvidenceAcceptanceCandidateIdentities = []string{current.EvidenceAcceptanceCandidateIdentity + ":other"}
		},
		"sink": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.RevisionAuthorityRef += ":other"
		},
		"scope": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.ScopeValue = "other-procedure/BP-99"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3BBCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3BBAssessResultAcceptanceAuthority(
				reconciliation,
				[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
				businessAddress,
				[]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusUnresolved ||
				len(got.QualifiedEvidenceIDs) != 0 {
				t.Fatalf("R3BB qualified mismatched %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3BBMissingAuthorityLocusOrProvenanceDoesNotQualify(t *testing.T) {
	reconciliation, execution, current, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)
	base := actorContextR3BBCompleteEvidence(
		"test-only:r3bb/base",
		businessAddress,
		reconciliation,
		execution,
		current,
	)
	cases := map[string]func(*actorContextR3BBResultAcceptanceAuthorityEvidenceProbe){
		"repository": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.RepositoryBacked = false
		},
		"authority-kind": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.AuthorityKind = ""
		},
		"declaration": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.AuthorityDeclarationRef = ""
		},
		"result-locus": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.ReconciliationResultDecisionLocusRef = ""
		},
		"current-state-locus": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.CurrentStateAcceptanceLocusRef = ""
		},
		"provenance": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.ResultAcceptanceProvenanceRef = ""
		},
		"derived": func(v *actorContextR3BBResultAcceptanceAuthorityEvidenceProbe) {
			v.DerivedOnly = true
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3BBCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3BBAssessResultAcceptanceAuthority(
				reconciliation,
				[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
				businessAddress,
				[]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusUnresolved {
				t.Fatalf("R3BB qualified incomplete %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3BBMultipleQualifiedKeepsPrecedenceUnresolved(t *testing.T) {
	reconciliation, execution, current, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)
	a := actorContextR3BBCompleteEvidence(
		"test-only:r3bb/a", businessAddress, reconciliation, execution, current,
	)
	b := actorContextR3BBCompleteEvidence(
		"test-only:r3bb/b", businessAddress, reconciliation, execution, current,
	)
	got, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{current, execution},
		businessAddress,
		[]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{b, a},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReconciliationResultAcceptanceAuthorityStatus != actorContextR3BBStatusMultipleQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{a.EvidenceID, b.EvidenceID}) {
		t.Fatalf("R3BB selected winner among multiple result authorities: %+v", got)
	}
}

func TestActorContextR3BBInputOrderDoesNotChangeGateResult(t *testing.T) {
	reconciliation, execution, current, businessAddress := actorContextR3BBBothAcceptCandidatesFixture(t)
	qualified := actorContextR3BBCompleteEvidence(
		"test-only:r3bb/qualified", businessAddress, reconciliation, execution, current,
	)
	insufficient := actorContextR3BBCloneEvidence(qualified)
	insufficient.EvidenceID = "test-only:r3bb/insufficient"
	insufficient.CurrentStateAcceptanceLocusRef = ""

	first, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{execution, current},
		businessAddress,
		[]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{qualified, insufficient},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3BBAssessResultAcceptanceAuthority(
		reconciliation,
		[]actorContextR3BAEvidenceAcceptanceCandidateProbe{current, execution},
		businessAddress,
		[]actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{insufficient, qualified},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3BB input order changed Gate result: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3BBShapeHasNoResultAcceptanceSynchronizationOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{}),
		reflect.TypeOf(actorContextR3BBResultAcceptanceAuthorityGateProbe{}),
	} {
		for _, forbidden := range []string{
			"ReconciliationResultAccepted", "CurrentStateAccepted",
			"StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedDecision", "CurrentDecision", "CurrentAuthority", "AppliedAuthority",
			"RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "Rollback", "Compensation",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3BB Gate introduced forbidden result/synchronization shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3BBBothAcceptCandidatesFixture(
	t *testing.T,
) (
	actorContextR3AXReconciliationCandidateProbe,
	actorContextR3BAEvidenceAcceptanceCandidateProbe,
	actorContextR3BAEvidenceAcceptanceCandidateProbe,
	string,
) {
	t.Helper()

	reconciliation, observations, businessAddress := actorContextR3AZObservationFixture(t)
	executionAuthority := actorContextR3AZCompleteExecutionEvidence(
		"test-only:r3bb/execution-authority",
		businessAddress,
		reconciliation,
		observations.ExecutionObservations[0],
	)
	currentAuthority := actorContextR3AZCompleteCurrentStateEvidence(
		"test-only:r3bb/current-state-authority",
		businessAddress,
		reconciliation,
		observations.CurrentStateObservations[0],
	)
	evidence := []actorContextR3AZEvidenceProbe{executionAuthority, currentAuthority}
	gate, err := actorContextR3AZAssessEvidenceAcceptanceAuthority(
		reconciliation, observations, businessAddress, evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	witnesses := []actorContextR3BAEvidenceAcceptanceWitnessProbe{
		actorContextR3BAExactExecutionWitness(
			reconciliation,
			observations.ExecutionObservations[0],
			executionAuthority.EvidenceID,
			actorContextR3BADecisionAccept,
		),
		actorContextR3BAExactCurrentStateWitness(
			reconciliation,
			observations.CurrentStateObservations[0],
			currentAuthority.EvidenceID,
			actorContextR3BADecisionAccept,
		),
	}
	candidates, err := actorContextR3BABuildCandidates(
		reconciliation, observations, gate, evidence, witnesses,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 2 {
		t.Fatalf("R3BB setup plane candidate count=%d want=2", len(candidates))
	}
	var execution, current *actorContextR3BAEvidenceAcceptanceCandidateProbe
	for i := range candidates {
		switch candidates[i].EvidencePlane {
		case actorContextR3BAPlaneExecution:
			execution = &candidates[i]
		case actorContextR3BAPlaneCurrentState:
			current = &candidates[i]
		}
	}
	if execution == nil || current == nil {
		t.Fatalf("R3BB setup lost plane-local accept candidates: %+v", candidates)
	}
	return reconciliation, *execution, *current, businessAddress
}

func actorContextR3BBCurrentInspectedEvidence(
	execution actorContextR3BAEvidenceAcceptanceCandidateProbe,
	current actorContextR3BAEvidenceAcceptanceCandidateProbe,
) []actorContextR3BBResultAcceptanceAuthorityEvidenceProbe {
	return []actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{
		{
			EvidenceID: execution.EvidenceAcceptanceCandidateIdentity,
			EvidenceRole: actorContextR3BBRoleDerivedExecutionCandidate,
			ReconciliationCandidateIdentities: []string{execution.ReconciliationCandidateIdentity},
			ExecutionEvidenceAcceptanceCandidateIdentities: []string{
				execution.EvidenceAcceptanceCandidateIdentity,
			},
			DerivedOnly: true,
		},
		{
			EvidenceID: current.EvidenceAcceptanceCandidateIdentity,
			EvidenceRole: actorContextR3BBRoleDerivedCurrentStateCandidate,
			ReconciliationCandidateIdentities: []string{current.ReconciliationCandidateIdentity},
			CurrentStateEvidenceAcceptanceCandidateIdentities: []string{
				current.EvidenceAcceptanceCandidateIdentity,
			},
			RevisionAuthorityRef: current.RevisionAuthorityRef,
			DerivedOnly: true,
		},
	}
}

func actorContextR3BBCompleteEvidence(
	id string,
	businessAddress string,
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	execution actorContextR3BAEvidenceAcceptanceCandidateProbe,
	current actorContextR3BAEvidenceAcceptanceCandidateProbe,
) actorContextR3BBResultAcceptanceAuthorityEvidenceProbe {
	return actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{
		EvidenceID:                       id,
		EvidenceRole:                     actorContextR3BBRoleResultAuthorityCandidate,
		RepositoryBacked:                 true,
		AuthorityKind:                    actorContextR3BBAuthorityKind,
		AuthorityDeclarationRef:           id + "/authority-declaration",
		ReconciliationCandidateIdentities: []string{reconciliation.ReconciliationCandidateIdentity},
		ExecutionEvidenceAcceptanceCandidateIdentities: []string{
			execution.EvidenceAcceptanceCandidateIdentity,
		},
		CurrentStateEvidenceAcceptanceCandidateIdentities: []string{
			current.EvidenceAcceptanceCandidateIdentity,
		},
		AcceptedDecisionSinkRef:              reconciliation.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:                 reconciliation.RevisionAuthorityRef,
		ScopeKind:                            actorContextR3BBScopeExactBusinessAddress,
		ScopeValue:                           businessAddress,
		ReconciliationResultDecisionLocusRef: id + "/result-decision-locus",
		CurrentStateAcceptanceLocusRef:       id + "/current-state-acceptance-locus",
		ResultAcceptanceProvenanceRef:        id + "/acceptance-provenance",
	}
}

func actorContextR3BBAssessResultAcceptanceAuthority(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	candidates []actorContextR3BAEvidenceAcceptanceCandidateProbe,
	businessAddress string,
	evidence []actorContextR3BBResultAcceptanceAuthorityEvidenceProbe,
) (actorContextR3BBResultAcceptanceAuthorityGateProbe, error) {
	if reconciliation.ReconciliationCandidateIdentity == "" ||
		reconciliation.AcceptedDecisionSinkRef == "" ||
		reconciliation.RevisionAuthorityRef == "" ||
		businessAddress == "" {
		return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
			"R3BB reconciliation/business context is incomplete",
		)
	}

	var execution *actorContextR3BAEvidenceAcceptanceCandidateProbe
	var current *actorContextR3BAEvidenceAcceptanceCandidateProbe
	for i := range candidates {
		value := &candidates[i]
		if value.ReconciliationCandidateIdentity != reconciliation.ReconciliationCandidateIdentity ||
			value.Decision != actorContextR3BADecisionAccept ||
			value.EvidenceAcceptanceCandidateIdentity == "" {
			return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
				"R3BB plane candidate is not an exact accept candidate for current reconciliation",
			)
		}
		switch value.EvidencePlane {
		case actorContextR3BAPlaneExecution:
			if execution != nil {
				return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
					"R3BB requires exactly one execution candidate",
				)
			}
			execution = value
		case actorContextR3BAPlaneCurrentState:
			if current != nil {
				return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
					"R3BB requires exactly one current-state candidate",
				)
			}
			current = value
		default:
			return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
				"R3BB unsupported Evidence plane %q", value.EvidencePlane,
			)
		}
	}
	if execution == nil || current == nil {
		return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
			"R3BB requires both execution and current-state accept candidates",
		)
	}
	if current.RevisionAuthorityRef != reconciliation.RevisionAuthorityRef {
		return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
			"R3BB current-state candidate revision authority does not match reconciliation candidate",
		)
	}

	qualified := []actorContextR3BBResultAcceptanceAuthorityEvidenceProbe{}
	insufficient := []string{}
	seenIDs := map[string]bool{}
	for _, value := range evidence {
		if value.EvidenceID == "" || value.EvidenceRole == "" {
			return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
				"R3BB evidence identity/role is incomplete",
			)
		}
		if seenIDs[value.EvidenceID] {
			return actorContextR3BBResultAcceptanceAuthorityGateProbe{}, fmt.Errorf(
				"R3BB duplicate evidence id %q", value.EvidenceID,
			)
		}
		seenIDs[value.EvidenceID] = true
		if actorContextR3BBCandidateQualifies(
			reconciliation, *execution, *current, businessAddress, value,
		) {
			qualified = append(qualified, value)
		} else {
			insufficient = append(insufficient, value.EvidenceID)
		}
	}
	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].EvidenceID < qualified[j].EvidenceID
	})

	status := actorContextR3BBStatusUnresolved
	if len(qualified) == 1 {
		status = actorContextR3BBStatusQualified
	} else if len(qualified) > 1 {
		status = actorContextR3BBStatusMultipleQualified
	}

	qualifiedIDs := []string{}
	resultLoci := []string{}
	currentLoci := []string{}
	for _, value := range qualified {
		qualifiedIDs = append(qualifiedIDs, value.EvidenceID)
		resultLoci = append(resultLoci, value.ReconciliationResultDecisionLocusRef)
		currentLoci = append(currentLoci, value.CurrentStateAcceptanceLocusRef)
	}

	return actorContextR3BBResultAcceptanceAuthorityGateProbe{
		ReconciliationResultAcceptanceAuthorityStatus: status,
		QualifiedEvidenceIDs: actorContextR0SortedUnique(qualifiedIDs),
		ReconciliationResultDecisionLocusRefs: actorContextR0SortedUnique(resultLoci),
		CurrentStateAcceptanceLocusRefs: actorContextR0SortedUnique(currentLoci),
		ExecutionEvidenceAcceptanceCandidateIdentity: execution.EvidenceAcceptanceCandidateIdentity,
		CurrentStateEvidenceAcceptanceCandidateIdentity: current.EvidenceAcceptanceCandidateIdentity,
		InsufficientEvidenceIDs: actorContextR0SortedUnique(insufficient),
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"plane-local-acceptance-candidates-do-not-create-result-acceptance-authority",
			"qualified-result-acceptance-authority-does-not-accept-reconciliation-result",
			"result-decision-locus-does-not-collapse-current-state-acceptance-locus",
			"revision-match-and-raw-outcome-do-not-create-result-acceptance-authority",
			"result-acceptance-gate-does-not-synchronize-state-or-apply-current-authority",
		}),
	}, nil
}

func actorContextR3BBCandidateQualifies(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	execution actorContextR3BAEvidenceAcceptanceCandidateProbe,
	current actorContextR3BAEvidenceAcceptanceCandidateProbe,
	businessAddress string,
	value actorContextR3BBResultAcceptanceAuthorityEvidenceProbe,
) bool {
	if value.EvidenceRole != actorContextR3BBRoleResultAuthorityCandidate ||
		!value.RepositoryBacked ||
		value.DerivedOnly ||
		value.AuthorityKind != actorContextR3BBAuthorityKind ||
		value.AuthorityDeclarationRef == "" ||
		!actorContextR0Contains(
			value.ReconciliationCandidateIdentities,
			reconciliation.ReconciliationCandidateIdentity,
		) ||
		!actorContextR0Contains(
			value.ExecutionEvidenceAcceptanceCandidateIdentities,
			execution.EvidenceAcceptanceCandidateIdentity,
		) ||
		!actorContextR0Contains(
			value.CurrentStateEvidenceAcceptanceCandidateIdentities,
			current.EvidenceAcceptanceCandidateIdentity,
		) ||
		value.AcceptedDecisionSinkRef != reconciliation.AcceptedDecisionSinkRef ||
		value.RevisionAuthorityRef != reconciliation.RevisionAuthorityRef ||
		value.ReconciliationResultDecisionLocusRef == "" ||
		value.CurrentStateAcceptanceLocusRef == "" ||
		value.ResultAcceptanceProvenanceRef == "" {
		return false
	}
	switch value.ScopeKind {
	case actorContextR3BBScopeExactBusinessAddress:
		return value.ScopeValue == businessAddress
	case actorContextR3BBScopeAnyBusinessAddress:
		return true
	default:
		return false
	}
}

func actorContextR3BBCloneEvidence(
	value actorContextR3BBResultAcceptanceAuthorityEvidenceProbe,
) actorContextR3BBResultAcceptanceAuthorityEvidenceProbe {
	out := value
	out.ReconciliationCandidateIdentities = append(
		[]string(nil), value.ReconciliationCandidateIdentities...,
	)
	out.ExecutionEvidenceAcceptanceCandidateIdentities = append(
		[]string(nil), value.ExecutionEvidenceAcceptanceCandidateIdentities...,
	)
	out.CurrentStateEvidenceAcceptanceCandidateIdentities = append(
		[]string(nil), value.CurrentStateEvidenceAcceptanceCandidateIdentities...,
	)
	return out
}

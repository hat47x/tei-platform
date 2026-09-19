package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const (
	actorContextR3BEDecisionApply  = "apply-authoritative-state"
	actorContextR3BEDecisionReject = "reject-authoritative-state-application"
)

type actorContextR3BEStateApplicationWitnessProbe struct {
	ReconciliationResultAcceptanceCandidateIdentity string
	StateApplicationAuthorityEvidenceID             string
	StateApplicationLocusRef                        string
	RevisionAcceptanceLocusRef                      string
	CurrentAuthorityProjectionLocusRef              string
	ObservedDecisionRevision                        string
	ObservedStateFingerprint                        string
	Decision                                        string
	StateApplicationProvenanceRef                   string
}

type actorContextR3BEStateApplicationCandidateProbe struct {
	ReconciliationCandidateIdentity                 string
	ReconciliationResultAcceptanceCandidateIdentity string
	StateApplicationAuthorityEvidenceID             string
	AcceptedDecisionSinkRef                         string
	RevisionAuthorityRef                            string
	StateApplicationLocusRef                        string
	RevisionAcceptanceLocusRef                      string
	CurrentAuthorityProjectionLocusRef              string
	ObservedDecisionRevision                        string
	ObservedStateFingerprint                        string
	Decision                                        string
	StateApplicationProvenanceRef                   string
	StateApplicationCandidateIdentity               string
	AuthorityLimits                                 []string
}

func TestActorContextR3BEUnresolvedGateCannotCreateCandidate(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	gate, err := actorContextR3BDAssessStateApplicationAuthority(
		reconciliation, result, businessAddress,
		actorContextR3BDCurrentInspectedEvidence(reconciliation, result),
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.StateApplicationAuthorityStatus != actorContextR3BDStatusUnresolved {
		t.Fatalf("R3BE setup expected unresolved R3BD Gate: %+v", gate)
	}
	if _, err := actorContextR3BEBuildCandidates(
		reconciliation, result, gate, nil,
		[]actorContextR3BEStateApplicationWitnessProbe{{
			ReconciliationResultAcceptanceCandidateIdentity: result.ReconciliationResultAcceptanceCandidateIdentity,
			StateApplicationAuthorityEvidenceID:             "test-only:r3be/not-qualified",
			StateApplicationLocusRef:                        "test-only:r3be/apply",
			RevisionAcceptanceLocusRef:                      "test-only:r3be/revision",
			CurrentAuthorityProjectionLocusRef:              "test-only:r3be/authority",
			ObservedDecisionRevision:                        result.ObservedDecisionRevision,
			ObservedStateFingerprint:                        result.ObservedStateFingerprint,
			Decision:                                        actorContextR3BEDecisionApply,
			StateApplicationProvenanceRef:                   "test-only:r3be/provenance",
		}},
	); err == nil {
		t.Fatal("R3BE created candidate while R3BD authority was unresolved")
	}
}

func TestActorContextR3BESingleQualifiedGateDoesNotAutoCreateCandidate(t *testing.T) {
	reconciliation, result, gate, evidence := actorContextR3BESingleQualifiedFixture(t)
	got, err := actorContextR3BEBuildCandidates(reconciliation, result, gate, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3BE fabricated state-application candidate without witness: %+v", got)
	}
}

func TestActorContextR3BEExplicitApplyAndRejectRemainCandidates(t *testing.T) {
	for _, decision := range []string{actorContextR3BEDecisionApply, actorContextR3BEDecisionReject} {
		t.Run(decision, func(t *testing.T) {
			reconciliation, result, gate, evidence := actorContextR3BESingleQualifiedFixture(t)
			witness := actorContextR3BEExactWitness(result, gate, decision)
			got, err := actorContextR3BEBuildCandidates(
				reconciliation, result, gate, evidence,
				[]actorContextR3BEStateApplicationWitnessProbe{witness},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 ||
				got[0].Decision != decision ||
				got[0].ReconciliationResultAcceptanceCandidateIdentity != result.ReconciliationResultAcceptanceCandidateIdentity ||
				got[0].ObservedDecisionRevision != result.ObservedDecisionRevision ||
				got[0].ObservedStateFingerprint != result.ObservedStateFingerprint ||
				got[0].StateApplicationCandidateIdentity == "" {
				t.Fatalf("R3BE candidate drifted: %+v", got)
			}
		})
	}
}

func TestActorContextR3BEWrongExactBindingFailsClosed(t *testing.T) {
	reconciliation, result, gate, evidence := actorContextR3BESingleQualifiedFixture(t)
	base := actorContextR3BEExactWitness(result, gate, actorContextR3BEDecisionApply)
	cases := map[string]func(*actorContextR3BEStateApplicationWitnessProbe){
		"result": func(v *actorContextR3BEStateApplicationWitnessProbe) {
			v.ReconciliationResultAcceptanceCandidateIdentity += ":other"
		},
		"authority": func(v *actorContextR3BEStateApplicationWitnessProbe) {
			v.StateApplicationAuthorityEvidenceID += ":other"
		},
		"application-locus": func(v *actorContextR3BEStateApplicationWitnessProbe) {
			v.StateApplicationLocusRef += ":other"
		},
		"revision-locus": func(v *actorContextR3BEStateApplicationWitnessProbe) {
			v.RevisionAcceptanceLocusRef += ":other"
		},
		"projection-locus": func(v *actorContextR3BEStateApplicationWitnessProbe) {
			v.CurrentAuthorityProjectionLocusRef += ":other"
		},
		"revision": func(v *actorContextR3BEStateApplicationWitnessProbe) {
			v.ObservedDecisionRevision += ":other"
		},
		"fingerprint": func(v *actorContextR3BEStateApplicationWitnessProbe) {
			v.ObservedStateFingerprint += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3BEBuildCandidates(
				reconciliation, result, gate, evidence,
				[]actorContextR3BEStateApplicationWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3BE accepted mismatched %s witness", name)
			}
		})
	}
}

func TestActorContextR3BEDecisionAndProvenanceMustBeExplicit(t *testing.T) {
	reconciliation, result, gate, evidence := actorContextR3BESingleQualifiedFixture(t)
	base := actorContextR3BEExactWitness(result, gate, actorContextR3BEDecisionApply)
	cases := map[string]func(*actorContextR3BEStateApplicationWitnessProbe){
		"decision":    func(v *actorContextR3BEStateApplicationWitnessProbe) { v.Decision = "" },
		"unsupported": func(v *actorContextR3BEStateApplicationWitnessProbe) { v.Decision = "test-only:auto-apply" },
		"provenance":  func(v *actorContextR3BEStateApplicationWitnessProbe) { v.StateApplicationProvenanceRef = "" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3BEBuildCandidates(
				reconciliation, result, gate, evidence,
				[]actorContextR3BEStateApplicationWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3BE accepted %s witness", name)
			}
		})
	}
}

func TestActorContextR3BEMultipleQualifiedGateCannotBeBypassed(t *testing.T) {
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	a := actorContextR3BDCompleteEvidence("test-only:r3be/a", businessAddress, reconciliation, result)
	b := actorContextR3BDCompleteEvidence("test-only:r3be/b", businessAddress, reconciliation, result)
	evidence := []actorContextR3BDStateApplicationAuthorityEvidenceProbe{a, b}
	gate, err := actorContextR3BDAssessStateApplicationAuthority(reconciliation, result, businessAddress, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if gate.StateApplicationAuthorityStatus != actorContextR3BDStatusMultipleQualified {
		t.Fatalf("R3BE setup lost multiple-qualified Gate: %+v", gate)
	}
	witness := actorContextR3BEStateApplicationWitnessProbe{
		ReconciliationResultAcceptanceCandidateIdentity: result.ReconciliationResultAcceptanceCandidateIdentity,
		StateApplicationAuthorityEvidenceID:             a.EvidenceID,
		StateApplicationLocusRef:                        a.StateApplicationLocusRef,
		RevisionAcceptanceLocusRef:                      a.RevisionAcceptanceLocusRef,
		CurrentAuthorityProjectionLocusRef:              a.CurrentAuthorityProjectionLocusRef,
		ObservedDecisionRevision:                        result.ObservedDecisionRevision,
		ObservedStateFingerprint:                        result.ObservedStateFingerprint,
		Decision:                                        actorContextR3BEDecisionApply,
		StateApplicationProvenanceRef:                   "test-only:r3be/provenance",
	}
	if _, err := actorContextR3BEBuildCandidates(reconciliation, result, gate, evidence,
		[]actorContextR3BEStateApplicationWitnessProbe{witness}); err == nil {
		t.Fatal("R3BE allowed witness to bypass R3BD authority precedence")
	}
}

func TestActorContextR3BEDuplicateNormalizesAndConflictFailsClosed(t *testing.T) {
	reconciliation, result, gate, evidence := actorContextR3BESingleQualifiedFixture(t)
	base := actorContextR3BEExactWitness(result, gate, actorContextR3BEDecisionApply)
	got, err := actorContextR3BEBuildCandidates(reconciliation, result, gate, evidence,
		[]actorContextR3BEStateApplicationWitnessProbe{base, base})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3BE did not normalize duplicate witness: %+v", got)
	}

	other := base
	other.Decision = actorContextR3BEDecisionReject
	if _, err := actorContextR3BEBuildCandidates(reconciliation, result, gate, evidence,
		[]actorContextR3BEStateApplicationWitnessProbe{base, other}); err == nil {
		t.Fatal("R3BE silently selected among conflicting decisions")
	}
}

func TestActorContextR3BERepeatedBuildIsIdentityStable(t *testing.T) {
	reconciliation, result, gate, evidence := actorContextR3BESingleQualifiedFixture(t)
	witness := actorContextR3BEExactWitness(result, gate, actorContextR3BEDecisionApply)
	first, err := actorContextR3BEBuildCandidates(reconciliation, result, gate, evidence,
		[]actorContextR3BEStateApplicationWitnessProbe{witness})
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3BEBuildCandidates(reconciliation, result, gate, evidence,
		[]actorContextR3BEStateApplicationWitnessProbe{witness})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3BE repeated build changed identity: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3BEShapeHasNoAppliedStateCurrentAuthorityOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3BEStateApplicationWitnessProbe{}),
		reflect.TypeOf(actorContextR3BEStateApplicationCandidateProbe{}),
	} {
		for _, forbidden := range []string{
			"StateApplied", "StateSynchronized", "Reconciled", "RevisionAccepted", "RevisionApplied",
			"AcceptedDecision", "CurrentDecision", "CurrentAuthority", "AppliedAuthority",
			"RuntimeOutcome", "AuthorizationResult", "Retry", "RetryEligible", "RetryRequired",
			"Rollback", "Compensation", "ToolCall", "Command", "NextAction", "CanExecute",
			"Authorized", "Allowed", "Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3BE candidate introduced forbidden state/current-authority shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3BESingleQualifiedFixture(t *testing.T) (
	actorContextR3AXReconciliationCandidateProbe,
	actorContextR3BCResultAcceptanceCandidateProbe,
	actorContextR3BDStateApplicationAuthorityGateProbe,
	[]actorContextR3BDStateApplicationAuthorityEvidenceProbe,
) {
	t.Helper()
	reconciliation, result, businessAddress := actorContextR3BDResultCandidateFixture(t, actorContextR3BCDecisionAccept)
	authority := actorContextR3BDCompleteEvidence("test-only:r3be/state-application-authority", businessAddress, reconciliation, result)
	evidence := []actorContextR3BDStateApplicationAuthorityEvidenceProbe{authority}
	gate, err := actorContextR3BDAssessStateApplicationAuthority(reconciliation, result, businessAddress, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if gate.StateApplicationAuthorityStatus != actorContextR3BDStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 || len(gate.StateApplicationLocusRefs) != 1 ||
		len(gate.RevisionAcceptanceLocusRefs) != 1 || len(gate.CurrentAuthorityProjectionLocusRefs) != 1 {
		t.Fatalf("R3BE setup lost single-qualified R3BD Gate: %+v", gate)
	}
	return reconciliation, result, gate, evidence
}

func actorContextR3BEExactWitness(
	result actorContextR3BCResultAcceptanceCandidateProbe,
	gate actorContextR3BDStateApplicationAuthorityGateProbe,
	decision string,
) actorContextR3BEStateApplicationWitnessProbe {
	return actorContextR3BEStateApplicationWitnessProbe{
		ReconciliationResultAcceptanceCandidateIdentity: result.ReconciliationResultAcceptanceCandidateIdentity,
		StateApplicationAuthorityEvidenceID:             gate.QualifiedEvidenceIDs[0],
		StateApplicationLocusRef:                        gate.StateApplicationLocusRefs[0],
		RevisionAcceptanceLocusRef:                      gate.RevisionAcceptanceLocusRefs[0],
		CurrentAuthorityProjectionLocusRef:              gate.CurrentAuthorityProjectionLocusRefs[0],
		ObservedDecisionRevision:                        result.ObservedDecisionRevision,
		ObservedStateFingerprint:                        result.ObservedStateFingerprint,
		Decision:                                        decision,
		StateApplicationProvenanceRef:                   "test-only:r3be/state-application-witness",
	}
}

func actorContextR3BEBuildCandidates(
	reconciliation actorContextR3AXReconciliationCandidateProbe,
	result actorContextR3BCResultAcceptanceCandidateProbe,
	gate actorContextR3BDStateApplicationAuthorityGateProbe,
	evidence []actorContextR3BDStateApplicationAuthorityEvidenceProbe,
	witnesses []actorContextR3BEStateApplicationWitnessProbe,
) ([]actorContextR3BEStateApplicationCandidateProbe, error) {
	if len(witnesses) == 0 {
		return nil, nil
	}
	if result.Decision != actorContextR3BCDecisionAccept ||
		result.ReconciliationResultAcceptanceCandidateIdentity == "" ||
		result.ReconciliationCandidateIdentity != reconciliation.ReconciliationCandidateIdentity {
		return nil, fmt.Errorf("R3BE result candidate is incomplete or not accepted")
	}
	if gate.StateApplicationAuthorityStatus != actorContextR3BDStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 || len(gate.StateApplicationLocusRefs) != 1 ||
		len(gate.RevisionAcceptanceLocusRefs) != 1 || len(gate.CurrentAuthorityProjectionLocusRefs) != 1 {
		return nil, fmt.Errorf("R3BE requires one exact single-qualified R3BD Gate")
	}

	byID := map[string]actorContextR3BDStateApplicationAuthorityEvidenceProbe{}
	for _, value := range evidence {
		if value.EvidenceID == "" {
			return nil, fmt.Errorf("R3BE authority EvidenceID is empty")
		}
		if _, ok := byID[value.EvidenceID]; ok {
			return nil, fmt.Errorf("R3BE duplicate authority EvidenceID %q", value.EvidenceID)
		}
		byID[value.EvidenceID] = value
	}

	var normalized *actorContextR3BEStateApplicationWitnessProbe
	var out actorContextR3BEStateApplicationCandidateProbe
	for _, witness := range witnesses {
		if witness.ReconciliationResultAcceptanceCandidateIdentity != result.ReconciliationResultAcceptanceCandidateIdentity ||
			witness.StateApplicationAuthorityEvidenceID != gate.QualifiedEvidenceIDs[0] ||
			witness.StateApplicationLocusRef != gate.StateApplicationLocusRefs[0] ||
			witness.RevisionAcceptanceLocusRef != gate.RevisionAcceptanceLocusRefs[0] ||
			witness.CurrentAuthorityProjectionLocusRef != gate.CurrentAuthorityProjectionLocusRefs[0] ||
			witness.ObservedDecisionRevision != result.ObservedDecisionRevision ||
			witness.ObservedStateFingerprint != result.ObservedStateFingerprint ||
			(witness.Decision != actorContextR3BEDecisionApply && witness.Decision != actorContextR3BEDecisionReject) ||
			witness.StateApplicationProvenanceRef == "" {
			return nil, fmt.Errorf("R3BE explicit state-application witness is incomplete or mismatched")
		}
		authority, ok := byID[witness.StateApplicationAuthorityEvidenceID]
		if !ok ||
			!actorContextR3BDCandidateQualifies(reconciliation, result, "order-entry.order-handling/BP-06", authority) ||
			authority.StateApplicationLocusRef != witness.StateApplicationLocusRef ||
			authority.RevisionAcceptanceLocusRef != witness.RevisionAcceptanceLocusRef ||
			authority.CurrentAuthorityProjectionLocusRef != witness.CurrentAuthorityProjectionLocusRef {
			return nil, fmt.Errorf("R3BE qualified authority no longer binds exact state-application context")
		}
		if normalized != nil {
			if !reflect.DeepEqual(*normalized, witness) {
				return nil, fmt.Errorf("R3BE conflicting state-application witnesses require separate precedence authority")
			}
			continue
		}
		copyWitness := witness
		normalized = &copyWitness
		out = actorContextR3BEStateApplicationCandidateProbe{
			ReconciliationCandidateIdentity:                 reconciliation.ReconciliationCandidateIdentity,
			ReconciliationResultAcceptanceCandidateIdentity: result.ReconciliationResultAcceptanceCandidateIdentity,
			StateApplicationAuthorityEvidenceID:             witness.StateApplicationAuthorityEvidenceID,
			AcceptedDecisionSinkRef:                         reconciliation.AcceptedDecisionSinkRef,
			RevisionAuthorityRef:                            reconciliation.RevisionAuthorityRef,
			StateApplicationLocusRef:                        witness.StateApplicationLocusRef,
			RevisionAcceptanceLocusRef:                      witness.RevisionAcceptanceLocusRef,
			CurrentAuthorityProjectionLocusRef:              witness.CurrentAuthorityProjectionLocusRef,
			ObservedDecisionRevision:                        result.ObservedDecisionRevision,
			ObservedStateFingerprint:                        result.ObservedStateFingerprint,
			Decision:                                        witness.Decision,
			StateApplicationProvenanceRef:                   witness.StateApplicationProvenanceRef,
			StateApplicationCandidateIdentity: strings.Join([]string{
				result.ReconciliationResultAcceptanceCandidateIdentity,
				witness.StateApplicationAuthorityEvidenceID,
				witness.StateApplicationLocusRef,
				witness.RevisionAcceptanceLocusRef,
				witness.CurrentAuthorityProjectionLocusRef,
				result.ObservedDecisionRevision,
				result.ObservedStateFingerprint,
				witness.Decision,
				witness.StateApplicationProvenanceRef,
			}, "||"),
			AuthorityLimits: actorContextR0SortedUnique([]string{
				"state-application-candidate-does-not-apply-authoritative-state",
				"state-application-candidate-does-not-accept-or-apply-revision",
				"state-application-candidate-does-not-project-current-authority",
				"reject-state-application-does-not-create-retry-rollback-compensation-or-failure",
			}),
		}
	}
	if normalized == nil {
		return nil, nil
	}
	return []actorContextR3BEStateApplicationCandidateProbe{out}, nil
}

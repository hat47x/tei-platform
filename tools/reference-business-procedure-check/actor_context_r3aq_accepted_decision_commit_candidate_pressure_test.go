package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type actorContextR3AQCommitWitnessProbe struct {
	WitnessIdentity          string
	WriteAuthorityEvidenceID string
	AcceptedDecisionSinkRef  string
	ExpectedDecisionRevision string
	CommitProvenanceRef      string
}

type actorContextR3AQCommitCandidateProbe struct {
	WitnessIdentity                    string
	AssertionIdentity                  string
	AssertionClass                     string
	AssertedValue                      string
	AcceptanceDecision                 string
	AcceptanceProvenanceRef            string
	WriteAuthorityEvidenceID           string
	AcceptedDecisionSinkRef            string
	ExpectedDecisionRevision           string
	WriteAuthorityRevisionAuthorityRef string
	WriteAuthorityProvenanceRef        string
	CommitProvenanceRef                string
	CommitCandidateIdentity            string
	AuthorityLimits                    []string
}

func TestActorContextR3AQCurrentUnresolvedWriteGateCannotCreateCandidate(t *testing.T) {
	witness, businessAddress := actorContextR3AQWitnessFixture(t, actorContextR3AODecisionAccept)
	gate, err := actorContextR3APAssessWriteAuthority(
		witness,
		businessAddress,
		actorContextR3APCurrentInspectedEvidence(witness),
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.WriteAuthorityStatus != actorContextR3APStatusUnresolved {
		t.Fatalf("R3AQ setup expected unresolved write authority: %+v", gate)
	}
	if _, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		nil,
		[]actorContextR3AQCommitWitnessProbe{{
			WitnessIdentity:          witness.WitnessIdentity,
			WriteAuthorityEvidenceID: "test-only:r3aq/not-qualified",
			AcceptedDecisionSinkRef:  "test-only:r3aq/sink",
			ExpectedDecisionRevision: "test-only:r3aq/revision",
			CommitProvenanceRef:      "test-only:r3aq/commit-provenance",
		}},
	); err == nil {
		t.Fatal("R3AQ created commit candidate while write authority was unresolved")
	}
}

func TestActorContextR3AQExactCommitWitnessCreatesCandidateWithoutCommit(t *testing.T) {
	witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
		t,
		actorContextR3AODecisionAccept,
	)
	current := evidence[0]
	got, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{{
			WitnessIdentity:          witness.WitnessIdentity,
			WriteAuthorityEvidenceID: current.EvidenceID,
			AcceptedDecisionSinkRef:  current.AcceptedDecisionSinkRef,
			ExpectedDecisionRevision: current.CurrentDecisionRevision,
			CommitProvenanceRef:      "test-only:r3aq/commit-provenance",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AQ candidate count=%d want=1: %+v", len(got), got)
	}
	candidate := got[0]
	if candidate.WitnessIdentity != witness.WitnessIdentity ||
		candidate.AssertionIdentity != witness.AssertionIdentity ||
		candidate.AcceptanceDecision != actorContextR3AODecisionAccept ||
		candidate.WriteAuthorityEvidenceID != current.EvidenceID ||
		candidate.AcceptedDecisionSinkRef != current.AcceptedDecisionSinkRef ||
		candidate.ExpectedDecisionRevision != current.CurrentDecisionRevision ||
		candidate.WriteAuthorityRevisionAuthorityRef != current.RevisionAuthorityRef ||
		candidate.WriteAuthorityProvenanceRef != current.WriteProvenanceRef ||
		candidate.CommitProvenanceRef != "test-only:r3aq/commit-provenance" ||
		candidate.CommitCandidateIdentity == "" {
		t.Fatalf("R3AQ commit candidate identity drifted: %+v", candidate)
	}
}

func TestActorContextR3AQNoCommitWitnessDoesNotAutoCreateCandidate(t *testing.T) {
	witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
		t,
		actorContextR3AODecisionAccept,
	)
	got, err := actorContextR3AQBuildCommitCandidates(
		witness, businessAddress, gate, evidence, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3AQ fabricated commit candidate without explicit commit witness: %+v", got)
	}
}

func TestActorContextR3AQWrongWitnessAuthorityOrSinkFailsClosed(t *testing.T) {
	witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
		t,
		actorContextR3AODecisionAccept,
	)
	current := evidence[0]
	base := actorContextR3AQCommitWitnessProbe{
		WitnessIdentity:          witness.WitnessIdentity,
		WriteAuthorityEvidenceID: current.EvidenceID,
		AcceptedDecisionSinkRef:  current.AcceptedDecisionSinkRef,
		ExpectedDecisionRevision: current.CurrentDecisionRevision,
		CommitProvenanceRef:      "test-only:r3aq/provenance",
	}
	cases := map[string]actorContextR3AQCommitWitnessProbe{
		"witness": func() actorContextR3AQCommitWitnessProbe {
			x := base
			x.WitnessIdentity += ":stale"
			return x
		}(),
		"authority": func() actorContextR3AQCommitWitnessProbe {
			x := base
			x.WriteAuthorityEvidenceID += ":other"
			return x
		}(),
		"sink": func() actorContextR3AQCommitWitnessProbe {
			x := base
			x.AcceptedDecisionSinkRef += ":other"
			return x
		}(),
	}
	for name, commitWitness := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := actorContextR3AQBuildCommitCandidates(
				witness,
				businessAddress,
				gate,
				evidence,
				[]actorContextR3AQCommitWitnessProbe{commitWitness},
			); err == nil {
				t.Fatalf("R3AQ accepted mismatched %s commit witness", name)
			}
		})
	}
}

func TestActorContextR3AQMultipleQualifiedWriteAuthorityCannotBeBypassed(t *testing.T) {
	witness, businessAddress := actorContextR3AQWitnessFixture(t, actorContextR3AODecisionAccept)
	a := actorContextR3APCompleteEvidence(
		"test-only:r3aq/write-a",
		businessAddress,
		witness,
		"test-only:r3aq/revision",
	)
	b := actorContextR3APCompleteEvidence(
		"test-only:r3aq/write-b",
		businessAddress,
		witness,
		"test-only:r3aq/revision",
	)
	evidence := []actorContextR3APWriteAuthorityEvidenceProbe{a, b}
	gate, err := actorContextR3APAssessWriteAuthority(witness, businessAddress, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if gate.WriteAuthorityStatus != actorContextR3APStatusMultipleQualified {
		t.Fatalf("R3AQ setup lost multiple-qualified write authority state: %+v", gate)
	}

	if _, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{{
			WitnessIdentity:          witness.WitnessIdentity,
			WriteAuthorityEvidenceID: a.EvidenceID,
			AcceptedDecisionSinkRef:  a.AcceptedDecisionSinkRef,
			ExpectedDecisionRevision: a.CurrentDecisionRevision,
			CommitProvenanceRef:      "test-only:r3aq/provenance",
		}},
	); err == nil {
		t.Fatal("R3AQ allowed commit witness to bypass unresolved write-authority precedence")
	}
}

func TestActorContextR3AQStaleExpectedDecisionRevisionFailsClosed(t *testing.T) {
	witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
		t,
		actorContextR3AODecisionAccept,
	)
	current := evidence[0]
	if _, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{{
			WitnessIdentity:          witness.WitnessIdentity,
			WriteAuthorityEvidenceID: current.EvidenceID,
			AcceptedDecisionSinkRef:  current.AcceptedDecisionSinkRef,
			ExpectedDecisionRevision: current.CurrentDecisionRevision + ":stale",
			CommitProvenanceRef:      "test-only:r3aq/provenance",
		}},
	); err == nil {
		t.Fatal("R3AQ accepted stale expected decision revision")
	}
}

func TestActorContextR3AQMissingCommitProvenanceFailsClosed(t *testing.T) {
	witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
		t,
		actorContextR3AODecisionAccept,
	)
	current := evidence[0]
	if _, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{{
			WitnessIdentity:          witness.WitnessIdentity,
			WriteAuthorityEvidenceID: current.EvidenceID,
			AcceptedDecisionSinkRef:  current.AcceptedDecisionSinkRef,
			ExpectedDecisionRevision: current.CurrentDecisionRevision,
		}},
	); err == nil {
		t.Fatal("R3AQ accepted commit witness without commit provenance")
	}
}

func TestActorContextR3AQAcceptAndRejectStayDecisionDataNotAuthorizationOutcome(t *testing.T) {
	for _, decision := range []string{
		actorContextR3AODecisionAccept,
		actorContextR3AODecisionReject,
	} {
		t.Run(decision, func(t *testing.T) {
			witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
				t,
				decision,
			)
			current := evidence[0]
			got, err := actorContextR3AQBuildCommitCandidates(
				witness,
				businessAddress,
				gate,
				evidence,
				[]actorContextR3AQCommitWitnessProbe{{
					WitnessIdentity:          witness.WitnessIdentity,
					WriteAuthorityEvidenceID: current.EvidenceID,
					AcceptedDecisionSinkRef:  current.AcceptedDecisionSinkRef,
					ExpectedDecisionRevision: current.CurrentDecisionRevision,
					CommitProvenanceRef:      "test-only:r3aq/" + decision,
				}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != 1 || got[0].AcceptanceDecision != decision {
				t.Fatalf("R3AQ changed acceptance decision data: %+v", got)
			}
		})
	}
}

func TestActorContextR3AQDuplicateNormalizesAndConflictingCommitProvenanceFailsClosed(t *testing.T) {
	witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
		t,
		actorContextR3AODecisionAccept,
	)
	current := evidence[0]
	base := actorContextR3AQCommitWitnessProbe{
		WitnessIdentity:          witness.WitnessIdentity,
		WriteAuthorityEvidenceID: current.EvidenceID,
		AcceptedDecisionSinkRef:  current.AcceptedDecisionSinkRef,
		ExpectedDecisionRevision: current.CurrentDecisionRevision,
		CommitProvenanceRef:      "test-only:r3aq/provenance",
	}
	got, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AQ did not normalize exact duplicate commit witness: %+v", got)
	}

	conflict := base
	conflict.CommitProvenanceRef = "test-only:r3aq/other-provenance"
	if _, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{base, conflict},
	); err == nil {
		t.Fatal("R3AQ silently aggregated conflicting commit provenance")
	}

	stale := base
	stale.ExpectedDecisionRevision = base.ExpectedDecisionRevision + ":other"
	if _, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{base, stale},
	); err == nil {
		t.Fatal("R3AQ silently selected among conflicting expected revisions")
	}
}

func TestActorContextR3AQCommitCandidateShapeHasNoCommitApplicationOrExecutionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AQCommitWitnessProbe{}),
		reflect.TypeOf(actorContextR3AQCommitCandidateProbe{}),
	} {
		for _, forbidden := range []string{
			"Committed",
			"CommitSucceeded",
			"AcceptedDecision",
			"AcceptedAuthorityValue",
			"NewDecisionRevision",
			"CurrentAuthority",
			"ReevaluationAuthorityStatus",
			"ReuseAuthorityStatus",
			"RuntimeOutcome",
			"AuthorizationResult",
			"Applied",
			"Apply",
			"AutoApply",
			"ToolCall",
			"Command",
			"NextAction",
			"CanExecute",
			"Authorized",
			"Allowed",
			"Retry",
			"Reauthorize",
			"Confidence",
			"Priority",
			"Severity",
			"Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AQ commit candidate introduced forbidden commit/application/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AQWitnessFixture(
	t *testing.T,
	decision string,
) (actorContextR3AOObservedAcceptanceWitnessProbe, string) {
	t.Helper()

	assertion, businessAddress, gate, candidates := actorContextR3AOQualifiedFixture(t)
	got, err := actorContextR3AOBindWitnesses(
		assertion,
		businessAddress,
		gate,
		candidates,
		[]actorContextR3AOAcceptanceWitnessProbe{{
			AssertionIdentity:             assertion.AssertionIdentity,
			AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
			Decision:                      decision,
			AcceptanceProvenanceRef:       "test-only:r3aq/" + decision + "-witness",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AQ setup witness count=%d want=1", len(got))
	}
	return got[0], businessAddress
}

func actorContextR3AQSingleQualifiedFixture(
	t *testing.T,
	decision string,
) (
	actorContextR3AOObservedAcceptanceWitnessProbe,
	string,
	actorContextR3APWriteAuthorityGateProbe,
	[]actorContextR3APWriteAuthorityEvidenceProbe,
) {
	t.Helper()

	witness, businessAddress := actorContextR3AQWitnessFixture(t, decision)
	current := actorContextR3APCompleteEvidence(
		"test-only:r3aq/write-authority",
		businessAddress,
		witness,
		"test-only:r3aq/revision-1",
	)
	evidence := []actorContextR3APWriteAuthorityEvidenceProbe{current}
	gate, err := actorContextR3APAssessWriteAuthority(witness, businessAddress, evidence)
	if err != nil {
		t.Fatal(err)
	}
	if gate.WriteAuthorityStatus != actorContextR3APStatusQualified ||
		!reflect.DeepEqual(gate.QualifiedEvidenceIDs, []string{current.EvidenceID}) {
		t.Fatalf("R3AQ setup lost single-qualified write authority: %+v", gate)
	}
	return witness, businessAddress, gate, evidence
}

func actorContextR3AQBuildCommitCandidates(
	witness actorContextR3AOObservedAcceptanceWitnessProbe,
	businessAddress string,
	gate actorContextR3APWriteAuthorityGateProbe,
	authorityEvidence []actorContextR3APWriteAuthorityEvidenceProbe,
	commitWitnesses []actorContextR3AQCommitWitnessProbe,
) ([]actorContextR3AQCommitCandidateProbe, error) {
	if len(commitWitnesses) == 0 {
		return nil, nil
	}
	if witness.WitnessIdentity == "" || businessAddress == "" {
		return nil, fmt.Errorf("R3AQ witness/business context is incomplete")
	}
	if gate.WriteAuthorityStatus != actorContextR3APStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 {
		return nil, fmt.Errorf("R3AQ requires exactly one qualified write authority")
	}
	qualifiedID := gate.QualifiedEvidenceIDs[0]

	var authority *actorContextR3APWriteAuthorityEvidenceProbe
	for i := range authorityEvidence {
		if authorityEvidence[i].EvidenceID != qualifiedID {
			continue
		}
		if authority != nil {
			return nil, fmt.Errorf("R3AQ duplicate current write authority evidence %q", qualifiedID)
		}
		authority = &authorityEvidence[i]
	}
	if authority == nil ||
		!actorContextR3APCandidateQualifies(witness, businessAddress, *authority) {
		return nil, fmt.Errorf("R3AQ qualified write authority evidence is missing or no longer qualifies")
	}

	type groupState struct {
		expectedRevision string
		commitProvenance string
		candidate        actorContextR3AQCommitCandidateProbe
	}
	groups := map[string]groupState{}

	for _, commitWitness := range commitWitnesses {
		if commitWitness.WitnessIdentity == "" ||
			commitWitness.WriteAuthorityEvidenceID == "" ||
			commitWitness.AcceptedDecisionSinkRef == "" ||
			commitWitness.ExpectedDecisionRevision == "" ||
			commitWitness.CommitProvenanceRef == "" {
			return nil, fmt.Errorf("R3AQ explicit commit witness is incomplete")
		}
		if commitWitness.WitnessIdentity != witness.WitnessIdentity {
			return nil, fmt.Errorf("R3AQ commit witness references non-current WitnessIdentity")
		}
		if commitWitness.WriteAuthorityEvidenceID != qualifiedID {
			return nil, fmt.Errorf("R3AQ commit witness does not bind exact qualified write authority")
		}
		if commitWitness.AcceptedDecisionSinkRef != authority.AcceptedDecisionSinkRef {
			return nil, fmt.Errorf("R3AQ commit witness does not bind exact Accepted Decision sink")
		}
		if commitWitness.ExpectedDecisionRevision != authority.CurrentDecisionRevision {
			return nil, fmt.Errorf("R3AQ expected decision revision is stale")
		}

		groupKey := strings.Join([]string{
			commitWitness.WitnessIdentity,
			commitWitness.WriteAuthorityEvidenceID,
			commitWitness.AcceptedDecisionSinkRef,
		}, "||")
		candidateIdentity := strings.Join([]string{
			groupKey,
			commitWitness.ExpectedDecisionRevision,
			commitWitness.CommitProvenanceRef,
		}, "||")

		candidate := actorContextR3AQCommitCandidateProbe{
			WitnessIdentity:                    witness.WitnessIdentity,
			AssertionIdentity:                  witness.AssertionIdentity,
			AssertionClass:                     witness.AssertionClass,
			AssertedValue:                      witness.AssertedValue,
			AcceptanceDecision:                 witness.Decision,
			AcceptanceProvenanceRef:            witness.AcceptanceProvenanceRef,
			WriteAuthorityEvidenceID:           authority.EvidenceID,
			AcceptedDecisionSinkRef:            authority.AcceptedDecisionSinkRef,
			ExpectedDecisionRevision:           commitWitness.ExpectedDecisionRevision,
			WriteAuthorityRevisionAuthorityRef: authority.RevisionAuthorityRef,
			WriteAuthorityProvenanceRef:        authority.WriteProvenanceRef,
			CommitProvenanceRef:                commitWitness.CommitProvenanceRef,
			CommitCandidateIdentity:            candidateIdentity,
			AuthorityLimits: []string{
				"commit-candidate-does-not-mean-accepted-decision-committed",
				"matching-revision-does-not-execute-compare-and-write",
				"commit-candidate-does-not-increment-decision-revision",
				"accept-candidate-does-not-apply-current-authority",
				"reject-candidate-does-not-create-authorization-denial",
				"commit-candidate-does-not-authorize-operation",
			},
		}
		sort.Strings(candidate.AuthorityLimits)

		if previous, found := groups[groupKey]; found {
			if previous.expectedRevision != commitWitness.ExpectedDecisionRevision ||
				previous.commitProvenance != commitWitness.CommitProvenanceRef {
				return nil, fmt.Errorf(
					"R3AQ conflicting commit witnesses require separate aggregation/precedence authority",
				)
			}
			continue
		}
		groups[groupKey] = groupState{
			expectedRevision: commitWitness.ExpectedDecisionRevision,
			commitProvenance: commitWitness.CommitProvenanceRef,
			candidate:        candidate,
		}
	}

	out := make([]actorContextR3AQCommitCandidateProbe, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CommitCandidateIdentity < out[j].CommitCandidateIdentity
	})
	return out, nil
}

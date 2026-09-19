package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type actorContextR3ASWriteAttemptWitnessProbe struct {
	CommitCandidateIdentity        string
	ApplicationAuthorityEvidenceID string
	AcceptedDecisionSinkRef        string
	RevisionObservationRef         string
	ObservedDecisionRevision       string
	IdempotencyContractRef         string
	IdempotencyIdentity            string
	WriteOutcomeContractRef        string
	AttemptProvenanceRef           string
}

type actorContextR3ASWriteAttemptCandidateProbe struct {
	CommitCandidateIdentity        string
	ApplicationAuthorityEvidenceID string
	AcceptedDecisionSinkRef        string
	RevisionObservationRef         string
	ObservedDecisionRevision       string
	IdempotencyContractRef         string
	IdempotencyIdentity            string
	WriteOutcomeContractRef        string
	AttemptProvenanceRef           string
	WriteAttemptCandidateIdentity  string
	AuthorityLimits                []string
}

func TestActorContextR3ASCurrentUnresolvedApplicationGateCannotCreateAttemptCandidate(t *testing.T) {
	candidate, businessAddress, writeAuthority := actorContextR3ARCommitCandidateFixture(t)
	revision := actorContextR3ARRevisionObservation(candidate)
	gate, err := actorContextR3ARAssessApplicationAuthority(
		candidate,
		businessAddress,
		actorContextR3ARCurrentInspectedEvidence(candidate, writeAuthority),
		revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ApplicationAuthorityStatus != actorContextR3ARStatusUnresolved {
		t.Fatalf("R3AS setup expected unresolved application authority: %+v", gate)
	}

	if _, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate,
		businessAddress,
		gate,
		nil,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{{
			CommitCandidateIdentity:        candidate.CommitCandidateIdentity,
			ApplicationAuthorityEvidenceID: "test-only:r3as/not-qualified",
			AcceptedDecisionSinkRef:        candidate.AcceptedDecisionSinkRef,
			RevisionObservationRef:         revision.RevisionObservationRef,
			ObservedDecisionRevision:       revision.ObservedDecisionRevision,
			IdempotencyContractRef:         "test-only:r3as/idempotency-contract",
			IdempotencyIdentity:            "test-only:r3as/idempotency-1",
			WriteOutcomeContractRef:        "test-only:r3as/outcome-contract",
			AttemptProvenanceRef:           "test-only:r3as/attempt-provenance",
		}},
	); err == nil {
		t.Fatal("R3AS created attempt candidate while application authority was unresolved")
	}
}

func TestActorContextR3ASSingleQualifiedGateDoesNotAutoCreateAttempt(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	got, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate, businessAddress, gate, evidence, revision, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3AS fabricated write-attempt candidate without explicit witness: %+v", got)
	}
}

func TestActorContextR3ASExactWitnessCreatesCandidateWithoutExecutingWrite(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	witness := actorContextR3ASExactWitness(candidate, evidence[0], revision)
	got, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate,
		businessAddress,
		gate,
		evidence,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{witness},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AS attempt candidate count=%d want=1: %+v", len(got), got)
	}
	value := got[0]
	if value.CommitCandidateIdentity != candidate.CommitCandidateIdentity ||
		value.ApplicationAuthorityEvidenceID != evidence[0].EvidenceID ||
		value.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
		value.RevisionObservationRef != revision.RevisionObservationRef ||
		value.ObservedDecisionRevision != revision.ObservedDecisionRevision ||
		value.IdempotencyContractRef != evidence[0].IdempotencyContractRef ||
		value.IdempotencyIdentity != witness.IdempotencyIdentity ||
		value.WriteOutcomeContractRef != evidence[0].WriteOutcomeContractRef ||
		value.AttemptProvenanceRef != witness.AttemptProvenanceRef ||
		value.WriteAttemptCandidateIdentity == "" {
		t.Fatalf("R3AS write-attempt candidate identity drifted: %+v", value)
	}
}

func TestActorContextR3ASWrongCandidateAuthorityOrSinkFailsClosed(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	base := actorContextR3ASExactWitness(candidate, evidence[0], revision)

	cases := map[string]func(*actorContextR3ASWriteAttemptWitnessProbe){
		"candidate": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.CommitCandidateIdentity += ":other"
		},
		"application-authority": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.ApplicationAuthorityEvidenceID += ":other"
		},
		"sink": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.AcceptedDecisionSinkRef += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3ASBuildWriteAttemptCandidates(
				candidate,
				businessAddress,
				gate,
				evidence,
				revision,
				[]actorContextR3ASWriteAttemptWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3AS accepted mismatched %s witness", name)
			}
		})
	}
}

func TestActorContextR3ASRevisionObservationBindingFailsClosedOnMismatch(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	base := actorContextR3ASExactWitness(candidate, evidence[0], revision)

	cases := map[string]func(*actorContextR3ASWriteAttemptWitnessProbe){
		"revision-observation-ref": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.RevisionObservationRef += ":other"
		},
		"observed-revision": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.ObservedDecisionRevision += ":stale"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3ASBuildWriteAttemptCandidates(
				candidate,
				businessAddress,
				gate,
				evidence,
				revision,
				[]actorContextR3ASWriteAttemptWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3AS accepted mismatched %s", name)
			}
		})
	}
}

func TestActorContextR3ASContractBindingFailsClosedOnMismatch(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	base := actorContextR3ASExactWitness(candidate, evidence[0], revision)

	cases := map[string]func(*actorContextR3ASWriteAttemptWitnessProbe){
		"idempotency-contract": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.IdempotencyContractRef += ":other"
		},
		"write-outcome-contract": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.WriteOutcomeContractRef += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3ASBuildWriteAttemptCandidates(
				candidate,
				businessAddress,
				gate,
				evidence,
				revision,
				[]actorContextR3ASWriteAttemptWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3AS accepted mismatched %s", name)
			}
		})
	}
}

func TestActorContextR3ASMissingIdempotencyIdentityOrAttemptProvenanceFailsClosed(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	base := actorContextR3ASExactWitness(candidate, evidence[0], revision)

	cases := map[string]func(*actorContextR3ASWriteAttemptWitnessProbe){
		"idempotency-identity": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.IdempotencyIdentity = ""
		},
		"attempt-provenance": func(value *actorContextR3ASWriteAttemptWitnessProbe) {
			value.AttemptProvenanceRef = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := base
			mutate(&value)
			if _, err := actorContextR3ASBuildWriteAttemptCandidates(
				candidate,
				businessAddress,
				gate,
				evidence,
				revision,
				[]actorContextR3ASWriteAttemptWitnessProbe{value},
			); err == nil {
				t.Fatalf("R3AS accepted witness without %s", name)
			}
		})
	}
}

func TestActorContextR3ASMultipleQualifiedApplicationAuthoritiesCannotBeBypassed(t *testing.T) {
	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	revision := actorContextR3ARRevisionObservation(candidate)
	a := actorContextR3ARCompleteEvidence("test-only:r3as/a", businessAddress, candidate)
	b := actorContextR3ARCompleteEvidence("test-only:r3as/b", businessAddress, candidate)
	evidence := []actorContextR3ARApplicationAuthorityEvidenceProbe{a, b}
	gate, err := actorContextR3ARAssessApplicationAuthority(
		candidate, businessAddress, evidence, revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ApplicationAuthorityStatus != actorContextR3ARStatusMultipleQualified {
		t.Fatalf("R3AS setup lost multiple-qualified application authority: %+v", gate)
	}

	if _, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate,
		businessAddress,
		gate,
		evidence,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{
			actorContextR3ASExactWitness(candidate, a, revision),
		},
	); err == nil {
		t.Fatal("R3AS allowed witness to bypass unresolved application-authority precedence")
	}
}

func TestActorContextR3ASDuplicateNormalizesAndConflictingIdentityOrProvenanceFailsClosed(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	base := actorContextR3ASExactWitness(candidate, evidence[0], revision)

	got, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate,
		businessAddress,
		gate,
		evidence,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{base, base},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AS did not normalize exact duplicate witness: %+v", got)
	}

	otherIdentity := base
	otherIdentity.IdempotencyIdentity += ":other"
	if _, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate,
		businessAddress,
		gate,
		evidence,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{base, otherIdentity},
	); err == nil {
		t.Fatal("R3AS silently accepted conflicting idempotency identities")
	}

	otherProvenance := base
	otherProvenance.AttemptProvenanceRef += ":other"
	if _, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate,
		businessAddress,
		gate,
		evidence,
		revision,
		[]actorContextR3ASWriteAttemptWitnessProbe{base, otherProvenance},
	); err == nil {
		t.Fatal("R3AS silently accepted conflicting attempt provenance")
	}
}

func TestActorContextR3ASInputOrderAndRepeatedBuildAreStable(t *testing.T) {
	candidate, businessAddress, gate, evidence, revision := actorContextR3ASSingleQualifiedFixture(t)
	base := actorContextR3ASExactWitness(candidate, evidence[0], revision)
	inputA := []actorContextR3ASWriteAttemptWitnessProbe{base, base}
	inputB := []actorContextR3ASWriteAttemptWitnessProbe{base, base}
	for i, j := 0, len(inputB)-1; i < j; i, j = i+1, j-1 {
		inputB[i], inputB[j] = inputB[j], inputB[i]
	}

	first, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate, businessAddress, gate, evidence, revision, inputA,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate, businessAddress, gate, evidence, revision, inputB,
	)
	if err != nil {
		t.Fatal(err)
	}
	third, err := actorContextR3ASBuildWriteAttemptCandidates(
		candidate, businessAddress, gate, evidence, revision, inputA,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || !reflect.DeepEqual(first, third) {
		t.Fatalf("R3AS input order/repeated build changed candidate identity: first=%+v second=%+v third=%+v",
			first, second, third)
	}
}

func TestActorContextR3ASShapesHaveNoExecutionOutcomeOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3ASWriteAttemptWitnessProbe{}),
		reflect.TypeOf(actorContextR3ASWriteAttemptCandidateProbe{}),
	} {
		for _, forbidden := range []string{
			"WriteExecuted",
			"Attempted",
			"WriteOutcome",
			"CommitOutcome",
			"Applied",
			"NotApplied",
			"Conflict",
			"Committed",
			"CommitSucceeded",
			"AcceptedDecision",
			"NewDecisionRevision",
			"CurrentAuthority",
			"ToolCall",
			"Command",
			"Executor",
			"Endpoint",
			"Transport",
			"Retry",
			"RetryCount",
			"AttemptNumber",
			"NextAction",
			"CanExecute",
			"Authorized",
			"Allowed",
			"Confidence",
			"Priority",
			"Severity",
			"Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AS write-attempt candidate introduced forbidden execution/outcome shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3ASSingleQualifiedFixture(
	t *testing.T,
) (
	actorContextR3AQCommitCandidateProbe,
	string,
	actorContextR3ARApplicationAuthorityGateProbe,
	[]actorContextR3ARApplicationAuthorityEvidenceProbe,
	actorContextR3ARSinkRevisionObservationProbe,
) {
	t.Helper()

	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	revision := actorContextR3ARRevisionObservation(candidate)
	current := actorContextR3ARCompleteEvidence(
		"test-only:r3as/application-authority",
		businessAddress,
		candidate,
	)
	evidence := []actorContextR3ARApplicationAuthorityEvidenceProbe{current}
	gate, err := actorContextR3ARAssessApplicationAuthority(
		candidate, businessAddress, evidence, revision,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.ApplicationAuthorityStatus != actorContextR3ARStatusQualified ||
		!reflect.DeepEqual(gate.QualifiedEvidenceIDs, []string{current.EvidenceID}) {
		t.Fatalf("R3AS setup lost single-qualified application authority: %+v", gate)
	}
	return candidate, businessAddress, gate, evidence, revision
}

func actorContextR3ASExactWitness(
	candidate actorContextR3AQCommitCandidateProbe,
	authority actorContextR3ARApplicationAuthorityEvidenceProbe,
	revision actorContextR3ARSinkRevisionObservationProbe,
) actorContextR3ASWriteAttemptWitnessProbe {
	return actorContextR3ASWriteAttemptWitnessProbe{
		CommitCandidateIdentity:        candidate.CommitCandidateIdentity,
		ApplicationAuthorityEvidenceID: authority.EvidenceID,
		AcceptedDecisionSinkRef:        candidate.AcceptedDecisionSinkRef,
		RevisionObservationRef:         revision.RevisionObservationRef,
		ObservedDecisionRevision:       revision.ObservedDecisionRevision,
		IdempotencyContractRef:         authority.IdempotencyContractRef,
		IdempotencyIdentity:            "test-only:r3as/idempotency-1",
		WriteOutcomeContractRef:        authority.WriteOutcomeContractRef,
		AttemptProvenanceRef:           "test-only:r3as/attempt-provenance",
	}
}

func actorContextR3ASBuildWriteAttemptCandidates(
	candidate actorContextR3AQCommitCandidateProbe,
	businessAddress string,
	gate actorContextR3ARApplicationAuthorityGateProbe,
	authorityEvidence []actorContextR3ARApplicationAuthorityEvidenceProbe,
	revision actorContextR3ARSinkRevisionObservationProbe,
	witnesses []actorContextR3ASWriteAttemptWitnessProbe,
) ([]actorContextR3ASWriteAttemptCandidateProbe, error) {
	if len(witnesses) == 0 {
		return nil, nil
	}
	if candidate.CommitCandidateIdentity == "" ||
		candidate.AcceptedDecisionSinkRef == "" ||
		candidate.ExpectedDecisionRevision == "" ||
		businessAddress == "" {
		return nil, fmt.Errorf("R3AS commit candidate/business context is incomplete")
	}
	if gate.ApplicationAuthorityStatus != actorContextR3ARStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 {
		return nil, fmt.Errorf("R3AS requires exactly one qualified application authority")
	}
	qualifiedID := gate.QualifiedEvidenceIDs[0]

	var authority *actorContextR3ARApplicationAuthorityEvidenceProbe
	for i := range authorityEvidence {
		if authorityEvidence[i].EvidenceID != qualifiedID {
			continue
		}
		if authority != nil {
			return nil, fmt.Errorf("R3AS duplicate qualified application authority evidence %q", qualifiedID)
		}
		authority = &authorityEvidence[i]
	}
	if authority == nil ||
		!actorContextR3ARCandidateQualifies(candidate, businessAddress, *authority) {
		return nil, fmt.Errorf("R3AS qualified application authority evidence is missing or no longer qualifies")
	}
	if revision.RevisionObservationRef == "" ||
		revision.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
		revision.RevisionAuthorityRef != candidate.WriteAuthorityRevisionAuthorityRef ||
		revision.ObservedDecisionRevision != candidate.ExpectedDecisionRevision ||
		gate.ObservedDecisionRevision != revision.ObservedDecisionRevision {
		return nil, fmt.Errorf("R3AS current revision observation no longer matches commit candidate/Gate")
	}

	type groupState struct {
		idempotencyIdentity string
		attemptProvenance   string
		candidate           actorContextR3ASWriteAttemptCandidateProbe
	}
	groups := map[string]groupState{}

	for _, witness := range witnesses {
		if witness.CommitCandidateIdentity == "" ||
			witness.ApplicationAuthorityEvidenceID == "" ||
			witness.AcceptedDecisionSinkRef == "" ||
			witness.RevisionObservationRef == "" ||
			witness.ObservedDecisionRevision == "" ||
			witness.IdempotencyContractRef == "" ||
			witness.IdempotencyIdentity == "" ||
			witness.WriteOutcomeContractRef == "" ||
			witness.AttemptProvenanceRef == "" {
			return nil, fmt.Errorf("R3AS explicit write-attempt witness is incomplete")
		}
		if witness.CommitCandidateIdentity != candidate.CommitCandidateIdentity {
			return nil, fmt.Errorf("R3AS witness references non-current commit candidate")
		}
		if witness.ApplicationAuthorityEvidenceID != qualifiedID {
			return nil, fmt.Errorf("R3AS witness does not bind exact qualified application authority")
		}
		if witness.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef {
			return nil, fmt.Errorf("R3AS witness does not bind exact Accepted Decision sink")
		}
		if witness.RevisionObservationRef != revision.RevisionObservationRef ||
			witness.ObservedDecisionRevision != revision.ObservedDecisionRevision {
			return nil, fmt.Errorf("R3AS witness does not bind exact current revision observation")
		}
		if witness.IdempotencyContractRef != authority.IdempotencyContractRef {
			return nil, fmt.Errorf("R3AS witness does not bind exact idempotency contract")
		}
		if witness.WriteOutcomeContractRef != authority.WriteOutcomeContractRef {
			return nil, fmt.Errorf("R3AS witness does not bind exact write outcome contract")
		}

		groupKey := strings.Join([]string{
			witness.CommitCandidateIdentity,
			witness.ApplicationAuthorityEvidenceID,
			witness.AcceptedDecisionSinkRef,
			witness.ObservedDecisionRevision,
		}, "||")
		identity := strings.Join([]string{
			groupKey,
			witness.RevisionObservationRef,
			witness.IdempotencyContractRef,
			witness.IdempotencyIdentity,
			witness.WriteOutcomeContractRef,
			witness.AttemptProvenanceRef,
		}, "||")

		attemptCandidate := actorContextR3ASWriteAttemptCandidateProbe{
			CommitCandidateIdentity:        candidate.CommitCandidateIdentity,
			ApplicationAuthorityEvidenceID: authority.EvidenceID,
			AcceptedDecisionSinkRef:        candidate.AcceptedDecisionSinkRef,
			RevisionObservationRef:         revision.RevisionObservationRef,
			ObservedDecisionRevision:       revision.ObservedDecisionRevision,
			IdempotencyContractRef:         authority.IdempotencyContractRef,
			IdempotencyIdentity:            witness.IdempotencyIdentity,
			WriteOutcomeContractRef:        authority.WriteOutcomeContractRef,
			AttemptProvenanceRef:           witness.AttemptProvenanceRef,
			WriteAttemptCandidateIdentity:  identity,
			AuthorityLimits: []string{
				"write-attempt-candidate-does-not-mean-write-executed",
				"idempotency-identity-does-not-prove-deduplication",
				"write-outcome-contract-does-not-create-outcome",
				"matching-current-revision-does-not-execute-cas",
				"write-attempt-candidate-does-not-commit-accepted-decision",
				"write-attempt-candidate-does-not-apply-current-authority",
			},
		}
		sort.Strings(attemptCandidate.AuthorityLimits)

		if previous, found := groups[groupKey]; found {
			if previous.idempotencyIdentity != witness.IdempotencyIdentity ||
				previous.attemptProvenance != witness.AttemptProvenanceRef {
				return nil, fmt.Errorf(
					"R3AS conflicting write-attempt witnesses require separate retry/precedence authority",
				)
			}
			continue
		}
		groups[groupKey] = groupState{
			idempotencyIdentity: witness.IdempotencyIdentity,
			attemptProvenance:   witness.AttemptProvenanceRef,
			candidate:           attemptCandidate,
		}
	}

	out := make([]actorContextR3ASWriteAttemptCandidateProbe, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].WriteAttemptCandidateIdentity < out[j].WriteAttemptCandidateIdentity
	})
	return out, nil
}

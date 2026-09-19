package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3ARStatusUnresolved =
		"accepted-decision-application-authority-unresolved"
	actorContextR3ARStatusQualified =
		"accepted-decision-application-authority-evidence-qualified"
	actorContextR3ARStatusMultipleQualified =
		"accepted-decision-application-authority-multiple-qualified-precedence-unresolved"

	actorContextR3ARAuthorityKind =
		"operational-accepted-decision-application-authority"

	actorContextR3ARRoleWriteAuthorityPrecedent =
		"write-authority-only-precedent"
	actorContextR3ARRoleCommitCandidateProvenance =
		"derived-commit-candidate-provenance"
	actorContextR3ARRoleApplicationAuthorityCandidate =
		"accepted-decision-application-authority-candidate"

	actorContextR3ARScopeExactBusinessAddress = "exact-business-address"
	actorContextR3ARScopeAnyBusinessAddress   = "any-business-address"
)

type actorContextR3ARApplicationAuthorityEvidenceProbe struct {
	EvidenceID                 string
	EvidenceRole               string
	RepositoryBacked           bool
	AuthorityKind              string
	AuthorityDeclarationRef    string
	CommitCandidateIdentities  []string
	WriteAuthorityEvidenceIDs  []string
	ScopeKind                  string
	ScopeValue                 string
	AcceptedDecisionSinkRef    string
	RevisionAuthorityRef       string
	IdempotencyContractRef     string
	WriteOutcomeContractRef    string
	ApplicationProvenanceRef   string
	DerivedOnly                bool
}

type actorContextR3ARSinkRevisionObservationProbe struct {
	AcceptedDecisionSinkRef string
	RevisionAuthorityRef    string
	ObservedDecisionRevision string
	RevisionObservationRef  string
}

type actorContextR3ARApplicationAuthorityGateProbe struct {
	ApplicationAuthorityStatus string
	QualifiedEvidenceIDs        []string
	AcceptedDecisionSinkRefs    []string
	ObservedDecisionRevision    string
	IdempotencyContractRefs     []string
	WriteOutcomeContractRefs    []string
	InsufficientEvidenceIDs     []string
	AuthorityLimits             []string
}

func TestActorContextR3ARCurrentInspectedCorpusKeepsApplicationAuthorityUnresolved(t *testing.T) {
	candidate, businessAddress, writeAuthority := actorContextR3ARCommitCandidateFixture(t)

	got, err := actorContextR3ARAssessApplicationAuthority(
		candidate,
		businessAddress,
		actorContextR3ARCurrentInspectedEvidence(candidate, writeAuthority),
		actorContextR3ARRevisionObservation(candidate),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationAuthorityStatus != actorContextR3ARStatusUnresolved ||
		len(got.QualifiedEvidenceIDs) != 0 {
		t.Fatalf("R3AR fabricated application authority from current inspected corpus: %+v", got)
	}
	if len(got.InsufficientEvidenceIDs) == 0 {
		t.Fatalf("R3AR lost insufficient current evidence identities: %+v", got)
	}
}

func TestActorContextR3ARWriteAuthorityAloneDoesNotQualifyApplicationAuthority(t *testing.T) {
	candidate, businessAddress, writeAuthority := actorContextR3ARCommitCandidateFixture(t)
	evidence := actorContextR3ARApplicationAuthorityEvidenceProbe{
		EvidenceID:                writeAuthority.EvidenceID,
		EvidenceRole:              actorContextR3ARRoleWriteAuthorityPrecedent,
		RepositoryBacked:          true,
		AuthorityKind:             actorContextR3APAuthorityKind,
		AuthorityDeclarationRef:   writeAuthority.AuthorityDeclarationRef,
		CommitCandidateIdentities: []string{candidate.CommitCandidateIdentity},
		WriteAuthorityEvidenceIDs: []string{candidate.WriteAuthorityEvidenceID},
		ScopeKind:                 actorContextR3ARScopeExactBusinessAddress,
		ScopeValue:                businessAddress,
		AcceptedDecisionSinkRef:   candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:      candidate.WriteAuthorityRevisionAuthorityRef,
		ApplicationProvenanceRef:  writeAuthority.WriteProvenanceRef,
	}

	got, err := actorContextR3ARAssessApplicationAuthority(
		candidate,
		businessAddress,
		[]actorContextR3ARApplicationAuthorityEvidenceProbe{evidence},
		actorContextR3ARRevisionObservation(candidate),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationAuthorityStatus != actorContextR3ARStatusUnresolved {
		t.Fatalf("R3AR promoted R3AP write authority into application authority: %+v", got)
	}
}

func TestActorContextR3ARCompleteApplicationAuthorityAndCurrentRevisionQualifyGateOnly(t *testing.T) {
	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	complete := actorContextR3ARCompleteEvidence(
		"test-only:r3ar/application-authority",
		businessAddress,
		candidate,
	)
	got, err := actorContextR3ARAssessApplicationAuthority(
		candidate,
		businessAddress,
		[]actorContextR3ARApplicationAuthorityEvidenceProbe{complete},
		actorContextR3ARRevisionObservation(candidate),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationAuthorityStatus != actorContextR3ARStatusQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{complete.EvidenceID}) ||
		!reflect.DeepEqual(got.AcceptedDecisionSinkRefs, []string{candidate.AcceptedDecisionSinkRef}) ||
		got.ObservedDecisionRevision != candidate.ExpectedDecisionRevision ||
		!reflect.DeepEqual(got.IdempotencyContractRefs, []string{complete.IdempotencyContractRef}) ||
		!reflect.DeepEqual(got.WriteOutcomeContractRefs, []string{complete.WriteOutcomeContractRef}) {
		t.Fatalf("R3AR qualified Gate lost exact pre-write boundary data: %+v", got)
	}
}

func TestActorContextR3ARWrongCandidateAuthoritySinkOrRevisionAuthorityIsInsufficient(t *testing.T) {
	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	base := actorContextR3ARCompleteEvidence(
		"test-only:r3ar/application-authority",
		businessAddress,
		candidate,
	)
	cases := map[string]func(*actorContextR3ARApplicationAuthorityEvidenceProbe){
		"candidate": func(value *actorContextR3ARApplicationAuthorityEvidenceProbe) {
			value.CommitCandidateIdentities = []string{candidate.CommitCandidateIdentity + ":other"}
		},
		"write-authority": func(value *actorContextR3ARApplicationAuthorityEvidenceProbe) {
			value.WriteAuthorityEvidenceIDs = []string{candidate.WriteAuthorityEvidenceID + ":other"}
		},
		"sink": func(value *actorContextR3ARApplicationAuthorityEvidenceProbe) {
			value.AcceptedDecisionSinkRef += ":other"
		},
		"revision-authority": func(value *actorContextR3ARApplicationAuthorityEvidenceProbe) {
			value.RevisionAuthorityRef += ":other"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3ARCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3ARAssessApplicationAuthority(
				candidate,
				businessAddress,
				[]actorContextR3ARApplicationAuthorityEvidenceProbe{value},
				actorContextR3ARRevisionObservation(candidate),
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ApplicationAuthorityStatus != actorContextR3ARStatusUnresolved ||
				len(got.QualifiedEvidenceIDs) != 0 {
				t.Fatalf("R3AR qualified mismatched %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3ARStaleOrUnattributedRevisionObservationFailsClosed(t *testing.T) {
	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	evidence := []actorContextR3ARApplicationAuthorityEvidenceProbe{
		actorContextR3ARCompleteEvidence(
			"test-only:r3ar/application-authority",
			businessAddress,
			candidate,
		),
	}

	stale := actorContextR3ARRevisionObservation(candidate)
	stale.ObservedDecisionRevision += ":stale"
	if _, err := actorContextR3ARAssessApplicationAuthority(
		candidate, businessAddress, evidence, stale,
	); err == nil {
		t.Fatal("R3AR accepted stale current sink revision")
	}

	missingRef := actorContextR3ARRevisionObservation(candidate)
	missingRef.RevisionObservationRef = ""
	if _, err := actorContextR3ARAssessApplicationAuthority(
		candidate, businessAddress, evidence, missingRef,
	); err == nil {
		t.Fatal("R3AR accepted unattributed sink revision observation")
	}
}

func TestActorContextR3ARMissingIdempotencyOrOutcomeContractIsInsufficient(t *testing.T) {
	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	base := actorContextR3ARCompleteEvidence(
		"test-only:r3ar/application-authority",
		businessAddress,
		candidate,
	)
	cases := map[string]func(*actorContextR3ARApplicationAuthorityEvidenceProbe){
		"idempotency": func(value *actorContextR3ARApplicationAuthorityEvidenceProbe) {
			value.IdempotencyContractRef = ""
		},
		"write-outcome": func(value *actorContextR3ARApplicationAuthorityEvidenceProbe) {
			value.WriteOutcomeContractRef = ""
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3ARCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3ARAssessApplicationAuthority(
				candidate,
				businessAddress,
				[]actorContextR3ARApplicationAuthorityEvidenceProbe{value},
				actorContextR3ARRevisionObservation(candidate),
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.ApplicationAuthorityStatus != actorContextR3ARStatusUnresolved {
				t.Fatalf("R3AR qualified application authority without %s contract: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3ARMultipleQualifiedApplicationAuthoritiesKeepPrecedenceUnresolved(t *testing.T) {
	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	a := actorContextR3ARCompleteEvidence("test-only:r3ar/a", businessAddress, candidate)
	b := actorContextR3ARCompleteEvidence("test-only:r3ar/b", businessAddress, candidate)

	got, err := actorContextR3ARAssessApplicationAuthority(
		candidate,
		businessAddress,
		[]actorContextR3ARApplicationAuthorityEvidenceProbe{b, a},
		actorContextR3ARRevisionObservation(candidate),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.ApplicationAuthorityStatus != actorContextR3ARStatusMultipleQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{a.EvidenceID, b.EvidenceID}) {
		t.Fatalf("R3AR selected a winner among multiple application authorities: %+v", got)
	}
}

func TestActorContextR3ARInputOrderDoesNotChangeGateResult(t *testing.T) {
	candidate, businessAddress, _ := actorContextR3ARCommitCandidateFixture(t)
	qualified := actorContextR3ARCompleteEvidence(
		"test-only:r3ar/qualified",
		businessAddress,
		candidate,
	)
	insufficient := actorContextR3ARCloneEvidence(qualified)
	insufficient.EvidenceID = "test-only:r3ar/insufficient"
	insufficient.IdempotencyContractRef = ""

	first, err := actorContextR3ARAssessApplicationAuthority(
		candidate,
		businessAddress,
		[]actorContextR3ARApplicationAuthorityEvidenceProbe{qualified, insufficient},
		actorContextR3ARRevisionObservation(candidate),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3ARAssessApplicationAuthority(
		candidate,
		businessAddress,
		[]actorContextR3ARApplicationAuthorityEvidenceProbe{insufficient, qualified},
		actorContextR3ARRevisionObservation(candidate),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AR candidate input order changed Gate result: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3ARGateShapeHasNoAttemptOutcomeCommitOrExecutionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3ARApplicationAuthorityEvidenceProbe{}),
		reflect.TypeOf(actorContextR3ARSinkRevisionObservationProbe{}),
		reflect.TypeOf(actorContextR3ARApplicationAuthorityGateProbe{}),
	} {
		for _, forbidden := range []string{
			"WriteAttempt",
			"CommitCommand",
			"IdempotencyKey",
			"WriteOutcome",
			"CommitOutcome",
			"Committed",
			"CommitSucceeded",
			"NewDecisionRevision",
			"AcceptedDecision",
			"CurrentAuthority",
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
				t.Fatalf("R3AR Gate introduced forbidden write/application shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3ARCommitCandidateFixture(
	t *testing.T,
) (
	actorContextR3AQCommitCandidateProbe,
	string,
	actorContextR3APWriteAuthorityEvidenceProbe,
) {
	t.Helper()

	witness, businessAddress, gate, evidence := actorContextR3AQSingleQualifiedFixture(
		t,
		actorContextR3AODecisionAccept,
	)
	writeAuthority := evidence[0]
	candidates, err := actorContextR3AQBuildCommitCandidates(
		witness,
		businessAddress,
		gate,
		evidence,
		[]actorContextR3AQCommitWitnessProbe{{
			WitnessIdentity:          witness.WitnessIdentity,
			WriteAuthorityEvidenceID: writeAuthority.EvidenceID,
			AcceptedDecisionSinkRef:  writeAuthority.AcceptedDecisionSinkRef,
			ExpectedDecisionRevision: writeAuthority.CurrentDecisionRevision,
			CommitProvenanceRef:      "test-only:r3ar/commit-provenance",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 {
		t.Fatalf("R3AR setup commit candidate count=%d want=1", len(candidates))
	}
	return candidates[0], businessAddress, writeAuthority
}

func actorContextR3ARCurrentInspectedEvidence(
	candidate actorContextR3AQCommitCandidateProbe,
	writeAuthority actorContextR3APWriteAuthorityEvidenceProbe,
) []actorContextR3ARApplicationAuthorityEvidenceProbe {
	return []actorContextR3ARApplicationAuthorityEvidenceProbe{
		{
			EvidenceID:                writeAuthority.EvidenceID,
			EvidenceRole:              actorContextR3ARRoleWriteAuthorityPrecedent,
			RepositoryBacked:          true,
			AuthorityKind:             actorContextR3APAuthorityKind,
			AuthorityDeclarationRef:   writeAuthority.AuthorityDeclarationRef,
			CommitCandidateIdentities: []string{candidate.CommitCandidateIdentity},
			WriteAuthorityEvidenceIDs: []string{candidate.WriteAuthorityEvidenceID},
			AcceptedDecisionSinkRef:   candidate.AcceptedDecisionSinkRef,
			RevisionAuthorityRef:      candidate.WriteAuthorityRevisionAuthorityRef,
			ApplicationProvenanceRef:  writeAuthority.WriteProvenanceRef,
		},
		{
			EvidenceID:                candidate.CommitProvenanceRef,
			EvidenceRole:              actorContextR3ARRoleCommitCandidateProvenance,
			CommitCandidateIdentities: []string{candidate.CommitCandidateIdentity},
			WriteAuthorityEvidenceIDs: []string{candidate.WriteAuthorityEvidenceID},
			AcceptedDecisionSinkRef:   candidate.AcceptedDecisionSinkRef,
			RevisionAuthorityRef:      candidate.WriteAuthorityRevisionAuthorityRef,
			ApplicationProvenanceRef:  candidate.CommitProvenanceRef,
			DerivedOnly:               true,
		},
	}
}

func actorContextR3ARCompleteEvidence(
	id string,
	businessAddress string,
	candidate actorContextR3AQCommitCandidateProbe,
) actorContextR3ARApplicationAuthorityEvidenceProbe {
	return actorContextR3ARApplicationAuthorityEvidenceProbe{
		EvidenceID:                 id,
		EvidenceRole:               actorContextR3ARRoleApplicationAuthorityCandidate,
		RepositoryBacked:           true,
		AuthorityKind:              actorContextR3ARAuthorityKind,
		AuthorityDeclarationRef:    id + "/authority-declaration",
		CommitCandidateIdentities:  []string{candidate.CommitCandidateIdentity},
		WriteAuthorityEvidenceIDs:  []string{candidate.WriteAuthorityEvidenceID},
		ScopeKind:                  actorContextR3ARScopeExactBusinessAddress,
		ScopeValue:                 businessAddress,
		AcceptedDecisionSinkRef:    candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:       candidate.WriteAuthorityRevisionAuthorityRef,
		IdempotencyContractRef:     id + "/idempotency-contract",
		WriteOutcomeContractRef:    id + "/write-outcome-contract",
		ApplicationProvenanceRef:   id + "/application-provenance",
	}
}

func actorContextR3ARRevisionObservation(
	candidate actorContextR3AQCommitCandidateProbe,
) actorContextR3ARSinkRevisionObservationProbe {
	return actorContextR3ARSinkRevisionObservationProbe{
		AcceptedDecisionSinkRef:  candidate.AcceptedDecisionSinkRef,
		RevisionAuthorityRef:     candidate.WriteAuthorityRevisionAuthorityRef,
		ObservedDecisionRevision: candidate.ExpectedDecisionRevision,
		RevisionObservationRef:   "test-only:r3ar/current-sink-revision-observation",
	}
}

func actorContextR3ARCloneEvidence(
	value actorContextR3ARApplicationAuthorityEvidenceProbe,
) actorContextR3ARApplicationAuthorityEvidenceProbe {
	out := value
	out.CommitCandidateIdentities = append([]string(nil), value.CommitCandidateIdentities...)
	out.WriteAuthorityEvidenceIDs = append([]string(nil), value.WriteAuthorityEvidenceIDs...)
	return out
}

func actorContextR3ARAssessApplicationAuthority(
	candidate actorContextR3AQCommitCandidateProbe,
	businessAddress string,
	evidence []actorContextR3ARApplicationAuthorityEvidenceProbe,
	revision actorContextR3ARSinkRevisionObservationProbe,
) (actorContextR3ARApplicationAuthorityGateProbe, error) {
	if candidate.CommitCandidateIdentity == "" ||
		candidate.WriteAuthorityEvidenceID == "" ||
		candidate.AcceptedDecisionSinkRef == "" ||
		candidate.ExpectedDecisionRevision == "" ||
		candidate.WriteAuthorityRevisionAuthorityRef == "" ||
		businessAddress == "" {
		return actorContextR3ARApplicationAuthorityGateProbe{}, fmt.Errorf(
			"R3AR commit candidate/business context is incomplete",
		)
	}
	if revision.AcceptedDecisionSinkRef == "" ||
		revision.RevisionAuthorityRef == "" ||
		revision.ObservedDecisionRevision == "" ||
		revision.RevisionObservationRef == "" {
		return actorContextR3ARApplicationAuthorityGateProbe{}, fmt.Errorf(
			"R3AR current sink revision observation is incomplete",
		)
	}
	if revision.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
		revision.RevisionAuthorityRef != candidate.WriteAuthorityRevisionAuthorityRef {
		return actorContextR3ARApplicationAuthorityGateProbe{}, fmt.Errorf(
			"R3AR current sink revision observation targets a different sink/revision authority",
		)
	}
	if revision.ObservedDecisionRevision != candidate.ExpectedDecisionRevision {
		return actorContextR3ARApplicationAuthorityGateProbe{}, fmt.Errorf(
			"R3AR accepted-decision-current-revision-mismatch",
		)
	}

	qualified := []actorContextR3ARApplicationAuthorityEvidenceProbe{}
	insufficient := []string{}
	seenIDs := map[string]bool{}
	for _, value := range evidence {
		if value.EvidenceID == "" {
			return actorContextR3ARApplicationAuthorityGateProbe{}, fmt.Errorf(
				"R3AR application authority evidence id is empty",
			)
		}
		if seenIDs[value.EvidenceID] {
			return actorContextR3ARApplicationAuthorityGateProbe{}, fmt.Errorf(
				"R3AR duplicate application authority evidence id %q",
				value.EvidenceID,
			)
		}
		seenIDs[value.EvidenceID] = true

		if actorContextR3ARCandidateQualifies(candidate, businessAddress, value) {
			qualified = append(qualified, value)
		} else {
			insufficient = append(insufficient, value.EvidenceID)
		}
	}

	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].EvidenceID < qualified[j].EvidenceID
	})

	status := actorContextR3ARStatusUnresolved
	if len(qualified) == 1 {
		status = actorContextR3ARStatusQualified
	} else if len(qualified) > 1 {
		status = actorContextR3ARStatusMultipleQualified
	}

	qualifiedIDs := []string{}
	sinks := []string{}
	idempotencyContracts := []string{}
	outcomeContracts := []string{}
	for _, value := range qualified {
		qualifiedIDs = append(qualifiedIDs, value.EvidenceID)
		sinks = append(sinks, value.AcceptedDecisionSinkRef)
		idempotencyContracts = append(idempotencyContracts, value.IdempotencyContractRef)
		outcomeContracts = append(outcomeContracts, value.WriteOutcomeContractRef)
	}

	return actorContextR3ARApplicationAuthorityGateProbe{
		ApplicationAuthorityStatus: status,
		QualifiedEvidenceIDs:       actorContextR0SortedUnique(qualifiedIDs),
		AcceptedDecisionSinkRefs:   actorContextR0SortedUnique(sinks),
		ObservedDecisionRevision:   revision.ObservedDecisionRevision,
		IdempotencyContractRefs:    actorContextR0SortedUnique(idempotencyContracts),
		WriteOutcomeContractRefs:   actorContextR0SortedUnique(outcomeContracts),
		InsufficientEvidenceIDs:    actorContextR0SortedUnique(insufficient),
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"write-authority-does-not-create-application-authority",
			"commit-candidate-provenance-does-not-create-application-authority",
			"matching-current-revision-does-not-create-write-attempt",
			"idempotency-contract-does-not-assign-idempotency-key",
			"write-outcome-contract-does-not-create-write-outcome",
			"qualified-application-authority-does-not-commit-accepted-decision",
			"scope-specificity-revision-or-source-location-does-not-create-precedence",
		}),
	}, nil
}

func actorContextR3ARCandidateQualifies(
	candidate actorContextR3AQCommitCandidateProbe,
	businessAddress string,
	value actorContextR3ARApplicationAuthorityEvidenceProbe,
) bool {
	if value.EvidenceRole != actorContextR3ARRoleApplicationAuthorityCandidate ||
		!value.RepositoryBacked ||
		value.DerivedOnly ||
		value.AuthorityKind != actorContextR3ARAuthorityKind ||
		value.AuthorityDeclarationRef == "" ||
		!actorContextR0Contains(value.CommitCandidateIdentities, candidate.CommitCandidateIdentity) ||
		!actorContextR0Contains(value.WriteAuthorityEvidenceIDs, candidate.WriteAuthorityEvidenceID) ||
		value.AcceptedDecisionSinkRef != candidate.AcceptedDecisionSinkRef ||
		value.RevisionAuthorityRef != candidate.WriteAuthorityRevisionAuthorityRef ||
		value.IdempotencyContractRef == "" ||
		value.WriteOutcomeContractRef == "" ||
		value.ApplicationProvenanceRef == "" {
		return false
	}

	switch value.ScopeKind {
	case actorContextR3ARScopeExactBusinessAddress:
		return value.ScopeValue == businessAddress
	case actorContextR3ARScopeAnyBusinessAddress:
		return true
	default:
		return false
	}
}

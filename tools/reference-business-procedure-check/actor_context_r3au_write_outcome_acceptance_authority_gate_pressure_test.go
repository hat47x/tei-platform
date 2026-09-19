package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3AUStatusUnresolved        = "write-outcome-acceptance-authority-unresolved"
	actorContextR3AUStatusQualified         = "write-outcome-acceptance-authority-evidence-qualified"
	actorContextR3AUStatusMultipleQualified = "write-outcome-acceptance-authority-multiple-qualified-precedence-unresolved"

	actorContextR3AUAuthorityKind = "operational-write-outcome-acceptance-authority"

	actorContextR3AURoleOutcomeContractOnly        = "write-outcome-contract-only"
	actorContextR3AURoleDerivedRawObservation      = "derived-raw-writer-observation"
	actorContextR3AURoleOutcomeAcceptanceCandidate = "write-outcome-acceptance-authority-candidate"

	actorContextR3AUScopeExactBusinessAddress = "exact-business-address"
	actorContextR3AUScopeAnyBusinessAddress   = "any-business-address"
)

type actorContextR3AUOutcomeAcceptanceEvidenceProbe struct {
	EvidenceID                      string
	EvidenceRole                    string
	RepositoryBacked                bool
	AuthorityKind                   string
	AuthorityDeclarationRef         string
	WriteAttemptCandidateIdentities []string
	ObservationIdentities           []string
	WriteOutcomeContractRefs        []string
	ReportedOutcomes                []string
	AcceptedDecisionSinkRef         string
	ScopeKind                       string
	ScopeValue                      string
	OutcomeDecisionLocusRef         string
	OutcomeAcceptanceProvenanceRef  string
	DerivedOnly                     bool
}

type actorContextR3AUOutcomeAcceptanceGateProbe struct {
	OutcomeAcceptanceAuthorityStatus  string
	QualifiedEvidenceIDs              []string
	OutcomeDecisionLocusRefs          []string
	RawReportedOutcome                string
	PostWriteObservedDecisionRevision string
	InsufficientEvidenceIDs           []string
	AuthorityLimits                   []string
}

func TestActorContextR3AUCurrentInspectedEvidenceKeepsOutcomeAcceptanceUnresolved(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t,
		actorContextR3ATReportedApplied,
	)
	got, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt,
		observation,
		businessAddress,
		actorContextR3AUCurrentInspectedEvidence(attempt, observation),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusUnresolved ||
		len(got.QualifiedEvidenceIDs) != 0 {
		t.Fatalf("R3AU fabricated outcome acceptance authority from current evidence: %+v", got)
	}
}

func TestActorContextR3AUOutcomeContractOrRawObservationAloneDoesNotQualify(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t,
		actorContextR3ATReportedApplied,
	)
	for _, evidence := range actorContextR3AUCurrentInspectedEvidence(attempt, observation) {
		got, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
			attempt,
			observation,
			businessAddress,
			[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{evidence},
		)
		if err != nil {
			t.Fatal(err)
		}
		if got.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusUnresolved {
			t.Fatalf("R3AU promoted insufficient role %q into acceptance authority: %+v",
				evidence.EvidenceRole, got)
		}
	}
}

func TestActorContextR3AUCompleteSyntheticEvidenceQualifiesGateOnly(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t,
		actorContextR3ATReportedApplied,
	)
	complete := actorContextR3AUCompleteEvidence(
		"test-only:r3au/outcome-acceptance",
		businessAddress,
		attempt,
		observation,
	)
	got, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt,
		observation,
		businessAddress,
		[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{complete},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{complete.EvidenceID}) ||
		!reflect.DeepEqual(got.OutcomeDecisionLocusRefs, []string{complete.OutcomeDecisionLocusRef}) ||
		got.RawReportedOutcome != observation.ReportedOutcome ||
		got.PostWriteObservedDecisionRevision != observation.PostWriteObservedDecisionRevision {
		t.Fatalf("R3AU qualified Gate lost raw observation context: %+v", got)
	}
}

func TestActorContextR3AUMissingRequirementDoesNotQualify(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t,
		actorContextR3ATReportedApplied,
	)
	base := actorContextR3AUCompleteEvidence(
		"test-only:r3au/base",
		businessAddress,
		attempt,
		observation,
	)
	cases := map[string]func(*actorContextR3AUOutcomeAcceptanceEvidenceProbe){
		"repository": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.RepositoryBacked = false
		},
		"authority-kind": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.AuthorityKind = ""
		},
		"declaration": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.AuthorityDeclarationRef = ""
		},
		"attempt": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.WriteAttemptCandidateIdentities = nil
		},
		"observation": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.ObservationIdentities = nil
		},
		"contract": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.WriteOutcomeContractRefs = nil
		},
		"reported-outcome": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.ReportedOutcomes = nil
		},
		"sink": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.AcceptedDecisionSinkRef = ""
		},
		"scope": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.ScopeKind = ""
			v.ScopeValue = ""
		},
		"locus": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.OutcomeDecisionLocusRef = ""
		},
		"provenance": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.OutcomeAcceptanceProvenanceRef = ""
		},
		"derived": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.DerivedOnly = true
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3AUCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
				attempt,
				observation,
				businessAddress,
				[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusUnresolved ||
				len(got.QualifiedEvidenceIDs) != 0 {
				t.Fatalf("R3AU qualified incomplete %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3AUWrongExactBindingDoesNotQualify(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t,
		actorContextR3ATReportedApplied,
	)
	base := actorContextR3AUCompleteEvidence(
		"test-only:r3au/base",
		businessAddress,
		attempt,
		observation,
	)
	cases := map[string]func(*actorContextR3AUOutcomeAcceptanceEvidenceProbe){
		"attempt": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.WriteAttemptCandidateIdentities = []string{attempt.WriteAttemptCandidateIdentity + ":other"}
		},
		"observation": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.ObservationIdentities = []string{observation.ObservationIdentity + ":other"}
		},
		"contract": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.WriteOutcomeContractRefs = []string{attempt.WriteOutcomeContractRef + ":other"}
		},
		"outcome": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.ReportedOutcomes = []string{actorContextR3ATReportedConflict}
		},
		"sink": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.AcceptedDecisionSinkRef += ":other"
		},
		"scope": func(v *actorContextR3AUOutcomeAcceptanceEvidenceProbe) {
			v.ScopeValue = "other-procedure/BP-99"
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := actorContextR3AUCloneEvidence(base)
			mutate(&value)
			got, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
				attempt,
				observation,
				businessAddress,
				[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{value},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusUnresolved {
				t.Fatalf("R3AU qualified mismatched %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3AURawOutcomeClassesStayRawAtQualifiedGate(t *testing.T) {
	for _, outcome := range []string{
		actorContextR3ATReportedApplied,
		actorContextR3ATReportedNotApplied,
		actorContextR3ATReportedConflict,
		actorContextR3ATReportedUnknown,
	} {
		t.Run(outcome, func(t *testing.T) {
			attempt, observation, businessAddress := actorContextR3AUObservationFixture(t, outcome)
			complete := actorContextR3AUCompleteEvidence(
				"test-only:r3au/"+outcome,
				businessAddress,
				attempt,
				observation,
			)
			got, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
				attempt,
				observation,
				businessAddress,
				[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{complete},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusQualified ||
				got.RawReportedOutcome != outcome {
				t.Fatalf("R3AU changed raw outcome class %q: %+v", outcome, got)
			}
		})
	}
}

func TestActorContextR3AUMultipleQualifiedKeepsPrecedenceUnresolved(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t,
		actorContextR3ATReportedApplied,
	)
	a := actorContextR3AUCompleteEvidence("test-only:r3au/a", businessAddress, attempt, observation)
	b := actorContextR3AUCompleteEvidence("test-only:r3au/b", businessAddress, attempt, observation)
	got, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt,
		observation,
		businessAddress,
		[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{b, a},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.OutcomeAcceptanceAuthorityStatus != actorContextR3AUStatusMultipleQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{a.EvidenceID, b.EvidenceID}) {
		t.Fatalf("R3AU selected winner among multiple acceptance authorities: %+v", got)
	}
}

func TestActorContextR3AUInputOrderDoesNotChangeGateResult(t *testing.T) {
	attempt, observation, businessAddress := actorContextR3AUObservationFixture(
		t,
		actorContextR3ATReportedUnknown,
	)
	qualified := actorContextR3AUCompleteEvidence(
		"test-only:r3au/qualified", businessAddress, attempt, observation,
	)
	insufficient := actorContextR3AUCloneEvidence(qualified)
	insufficient.EvidenceID = "test-only:r3au/insufficient"
	insufficient.OutcomeDecisionLocusRef = ""

	first, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt, observation, businessAddress,
		[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{qualified, insufficient},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AUAssessOutcomeAcceptanceAuthority(
		attempt, observation, businessAddress,
		[]actorContextR3AUOutcomeAcceptanceEvidenceProbe{insufficient, qualified},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AU input order changed Gate result: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3AUShapeHasNoAcceptedOutcomeAuthorityOrRetryShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AUOutcomeAcceptanceEvidenceProbe{}),
		reflect.TypeOf(actorContextR3AUOutcomeAcceptanceGateProbe{}),
	} {
		for _, forbidden := range []string{
			"AcceptedWriteOutcome", "OutcomeAccepted", "AcceptedOutcomeDecision",
			"Committed", "CommitSucceeded", "AcceptedDecision", "NewDecisionRevision",
			"CurrentAuthority", "AppliedAuthority", "RuntimeOutcome", "AuthorizationResult",
			"Retry", "RetryEligible", "RetryRequired", "RetrySafe", "Reauthorize",
			"ToolCall", "Command", "NextAction", "CanExecute", "Authorized", "Allowed",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AU Gate introduced forbidden accepted-outcome/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AUObservationFixture(
	t *testing.T,
	outcome string,
) (
	actorContextR3ASWriteAttemptCandidateProbe,
	actorContextR3ATNormalizedObservationProbe,
	string,
) {
	t.Helper()
	attempt := actorContextR3ATWriteAttemptFixture(t)
	raw := actorContextR3ATExactObservation(attempt, outcome)
	observations, err := actorContextR3ATNormalizeObservations(
		attempt,
		[]actorContextR3ATWriterObservationProbe{raw},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(observations) != 1 {
		t.Fatalf("R3AU setup observation count=%d want=1", len(observations))
	}
	return attempt, observations[0], "order-entry.order-handling/BP-06"
}

func actorContextR3AUCurrentInspectedEvidence(
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
) []actorContextR3AUOutcomeAcceptanceEvidenceProbe {
	return []actorContextR3AUOutcomeAcceptanceEvidenceProbe{
		{
			EvidenceID:                      attempt.WriteOutcomeContractRef,
			EvidenceRole:                    actorContextR3AURoleOutcomeContractOnly,
			WriteAttemptCandidateIdentities: []string{attempt.WriteAttemptCandidateIdentity},
			WriteOutcomeContractRefs:        []string{attempt.WriteOutcomeContractRef},
			AcceptedDecisionSinkRef:         attempt.AcceptedDecisionSinkRef,
		},
		{
			EvidenceID:              observation.ObservationIdentity,
			EvidenceRole:            actorContextR3AURoleDerivedRawObservation,
			ObservationIdentities:   []string{observation.ObservationIdentity},
			ReportedOutcomes:        []string{observation.ReportedOutcome},
			AcceptedDecisionSinkRef: observation.AcceptedDecisionSinkRef,
			DerivedOnly:             true,
		},
	}
}

func actorContextR3AUCompleteEvidence(
	id string,
	businessAddress string,
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
) actorContextR3AUOutcomeAcceptanceEvidenceProbe {
	return actorContextR3AUOutcomeAcceptanceEvidenceProbe{
		EvidenceID:                      id,
		EvidenceRole:                    actorContextR3AURoleOutcomeAcceptanceCandidate,
		RepositoryBacked:                true,
		AuthorityKind:                   actorContextR3AUAuthorityKind,
		AuthorityDeclarationRef:         id + "/authority-declaration",
		WriteAttemptCandidateIdentities: []string{attempt.WriteAttemptCandidateIdentity},
		ObservationIdentities:           []string{observation.ObservationIdentity},
		WriteOutcomeContractRefs:        []string{attempt.WriteOutcomeContractRef},
		ReportedOutcomes:                []string{observation.ReportedOutcome},
		AcceptedDecisionSinkRef:         attempt.AcceptedDecisionSinkRef,
		ScopeKind:                       actorContextR3AUScopeExactBusinessAddress,
		ScopeValue:                      businessAddress,
		OutcomeDecisionLocusRef:         id + "/outcome-decision-locus",
		OutcomeAcceptanceProvenanceRef:  id + "/acceptance-provenance",
	}
}

func actorContextR3AUCloneEvidence(
	value actorContextR3AUOutcomeAcceptanceEvidenceProbe,
) actorContextR3AUOutcomeAcceptanceEvidenceProbe {
	out := value
	out.WriteAttemptCandidateIdentities = append([]string(nil), value.WriteAttemptCandidateIdentities...)
	out.ObservationIdentities = append([]string(nil), value.ObservationIdentities...)
	out.WriteOutcomeContractRefs = append([]string(nil), value.WriteOutcomeContractRefs...)
	out.ReportedOutcomes = append([]string(nil), value.ReportedOutcomes...)
	return out
}

func actorContextR3AUAssessOutcomeAcceptanceAuthority(
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	businessAddress string,
	evidence []actorContextR3AUOutcomeAcceptanceEvidenceProbe,
) (actorContextR3AUOutcomeAcceptanceGateProbe, error) {
	if attempt.WriteAttemptCandidateIdentity == "" ||
		attempt.WriteOutcomeContractRef == "" ||
		attempt.AcceptedDecisionSinkRef == "" ||
		observation.ObservationIdentity == "" ||
		observation.WriteAttemptCandidateIdentity != attempt.WriteAttemptCandidateIdentity ||
		observation.AcceptedDecisionSinkRef != attempt.AcceptedDecisionSinkRef ||
		observation.ReportedOutcome == "" ||
		businessAddress == "" {
		return actorContextR3AUOutcomeAcceptanceGateProbe{}, fmt.Errorf(
			"R3AU attempt/observation/business context is incomplete or inconsistent",
		)
	}

	qualified := []actorContextR3AUOutcomeAcceptanceEvidenceProbe{}
	insufficient := []string{}
	seenIDs := map[string]bool{}
	for _, value := range evidence {
		if value.EvidenceID == "" || value.EvidenceRole == "" {
			return actorContextR3AUOutcomeAcceptanceGateProbe{}, fmt.Errorf(
				"R3AU evidence identity/role is incomplete",
			)
		}
		if seenIDs[value.EvidenceID] {
			return actorContextR3AUOutcomeAcceptanceGateProbe{}, fmt.Errorf(
				"R3AU duplicate evidence id %q", value.EvidenceID,
			)
		}
		seenIDs[value.EvidenceID] = true

		if actorContextR3AUCandidateQualifies(attempt, observation, businessAddress, value) {
			qualified = append(qualified, value)
		} else {
			insufficient = append(insufficient, value.EvidenceID)
		}
	}
	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].EvidenceID < qualified[j].EvidenceID
	})

	status := actorContextR3AUStatusUnresolved
	if len(qualified) == 1 {
		status = actorContextR3AUStatusQualified
	} else if len(qualified) > 1 {
		status = actorContextR3AUStatusMultipleQualified
	}

	qualifiedIDs := []string{}
	loci := []string{}
	for _, value := range qualified {
		qualifiedIDs = append(qualifiedIDs, value.EvidenceID)
		loci = append(loci, value.OutcomeDecisionLocusRef)
	}

	return actorContextR3AUOutcomeAcceptanceGateProbe{
		OutcomeAcceptanceAuthorityStatus:  status,
		QualifiedEvidenceIDs:              actorContextR0SortedUnique(qualifiedIDs),
		OutcomeDecisionLocusRefs:          actorContextR0SortedUnique(loci),
		RawReportedOutcome:                observation.ReportedOutcome,
		PostWriteObservedDecisionRevision: observation.PostWriteObservedDecisionRevision,
		InsufficientEvidenceIDs:           actorContextR0SortedUnique(insufficient),
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"write-outcome-contract-does-not-create-outcome-acceptance-authority",
			"raw-writer-observation-does-not-create-outcome-acceptance-authority",
			"post-write-revision-change-does-not-accept-or-reject-outcome",
			"qualified-outcome-acceptance-authority-does-not-accept-raw-outcome",
			"raw-conflict-or-unknown-outcome-does-not-create-retry-authority",
			"outcome-acceptance-gate-does-not-commit-accepted-decision-or-apply-authority",
		}),
	}, nil
}

func actorContextR3AUCandidateQualifies(
	attempt actorContextR3ASWriteAttemptCandidateProbe,
	observation actorContextR3ATNormalizedObservationProbe,
	businessAddress string,
	value actorContextR3AUOutcomeAcceptanceEvidenceProbe,
) bool {
	if value.EvidenceRole != actorContextR3AURoleOutcomeAcceptanceCandidate ||
		!value.RepositoryBacked ||
		value.DerivedOnly ||
		value.AuthorityKind != actorContextR3AUAuthorityKind ||
		value.AuthorityDeclarationRef == "" ||
		!actorContextR0Contains(value.WriteAttemptCandidateIdentities, attempt.WriteAttemptCandidateIdentity) ||
		!actorContextR0Contains(value.ObservationIdentities, observation.ObservationIdentity) ||
		!actorContextR0Contains(value.WriteOutcomeContractRefs, attempt.WriteOutcomeContractRef) ||
		!actorContextR0Contains(value.ReportedOutcomes, observation.ReportedOutcome) ||
		value.AcceptedDecisionSinkRef != attempt.AcceptedDecisionSinkRef ||
		value.OutcomeDecisionLocusRef == "" ||
		value.OutcomeAcceptanceProvenanceRef == "" {
		return false
	}

	switch value.ScopeKind {
	case actorContextR3AUScopeExactBusinessAddress:
		return value.ScopeValue == businessAddress
	case actorContextR3AUScopeAnyBusinessAddress:
		return true
	default:
		return false
	}
}

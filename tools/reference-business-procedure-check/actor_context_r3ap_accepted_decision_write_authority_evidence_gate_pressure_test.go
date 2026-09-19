package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3APStatusUnresolved =
		"accepted-decision-write-authority-unresolved"
	actorContextR3APStatusQualified =
		"accepted-decision-write-authority-evidence-qualified"
	actorContextR3APStatusMultipleQualified =
		"accepted-decision-write-authority-multiple-qualified-precedence-unresolved"

	actorContextR3APAuthorityKind =
		"operational-accepted-decision-write-authority"

	actorContextR3APRoleAuthoringAcceptancePrecedent =
		"authoring-acceptance-precedent"
	actorContextR3APRoleDerivedWitness =
		"derived-acceptance-witness"
	actorContextR3APRoleWriteAuthorityCandidate =
		"accepted-decision-write-authority-candidate"

	actorContextR3APScopeExactBusinessAddress = "exact-business-address"
	actorContextR3APScopeAnyBusinessAddress   = "any-business-address"
)

type actorContextR3APWriteAuthorityEvidenceProbe struct {
	EvidenceID                    string
	EvidenceRole                  string
	RepositoryBacked              bool
	AuthorityKind                 string
	AuthorityDeclarationRef       string
	AssertionClasses              []string
	WitnessDecisions              []string
	AcceptanceDecisionLocusRefs   []string
	ScopeKind                     string
	ScopeValue                    string
	AcceptedDecisionSinkRef       string
	RevisionAuthorityRef          string
	CurrentDecisionRevision       string
	WriteProvenanceRef            string
	DerivedOnly                   bool
}

type actorContextR3APWriteAuthorityGateProbe struct {
	WriteAuthorityStatus     string
	QualifiedEvidenceIDs     []string
	AcceptedDecisionSinkRefs []string
	DecisionRevisions        []string
	InsufficientEvidenceIDs  []string
	AuthorityLimits          []string
}

func TestActorContextR3APCurrentInspectedRepositoryEvidenceKeepsWriteAuthorityUnresolved(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	got, err := actorContextR3APAssessWriteAuthority(
		witness,
		businessAddress,
		actorContextR3APCurrentInspectedEvidence(witness),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.WriteAuthorityStatus != actorContextR3APStatusUnresolved ||
		len(got.QualifiedEvidenceIDs) != 0 {
		t.Fatalf("R3AP fabricated Accepted Decision write authority from current inspected evidence: %+v", got)
	}
	if len(got.InsufficientEvidenceIDs) == 0 {
		t.Fatalf("R3AP lost current insufficient evidence identities: %+v", got)
	}
}

func TestActorContextR3APGeneralSharedAcceptanceDoesNotQualifyOperationalDecisionWrite(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	candidate := actorContextR3APWriteAuthorityEvidenceProbe{
		EvidenceID:              "plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md",
		EvidenceRole:            actorContextR3APRoleAuthoringAcceptancePrecedent,
		RepositoryBacked:        true,
		AuthorityKind:           "shared-authoring-acceptance-boundary",
		AuthorityDeclarationRef: "plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md",
		AcceptedDecisionSinkRef: "authoritative-authoring-source",
		RevisionAuthorityRef:    "authoring-base-model-revision",
		CurrentDecisionRevision: "test-only:authoring-revision",
		WriteProvenanceRef:      "plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md",
	}
	got, err := actorContextR3APAssessWriteAuthority(
		witness,
		businessAddress,
		[]actorContextR3APWriteAuthorityEvidenceProbe{candidate},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.WriteAuthorityStatus != actorContextR3APStatusUnresolved {
		t.Fatalf("R3AP reused general authoring Shared Acceptance as Operational decision write authority: %+v", got)
	}
}

func TestActorContextR3APWitnessProvenanceDoesNotCreateWriteAuthority(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	candidate := actorContextR3APWriteAuthorityEvidenceProbe{
		EvidenceID:              witness.AcceptanceProvenanceRef,
		EvidenceRole:            actorContextR3APRoleDerivedWitness,
		AuthorityDeclarationRef: witness.AcceptanceProvenanceRef,
		AssertionClasses:        []string{witness.AssertionClass},
		WitnessDecisions:        []string{witness.Decision},
		AcceptanceDecisionLocusRefs: []string{witness.DecisionLocusRef},
		ScopeKind:               actorContextR3APScopeExactBusinessAddress,
		ScopeValue:              businessAddress,
		DerivedOnly:             true,
	}
	got, err := actorContextR3APAssessWriteAuthority(
		witness,
		businessAddress,
		[]actorContextR3APWriteAuthorityEvidenceProbe{candidate},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.WriteAuthorityStatus != actorContextR3APStatusUnresolved {
		t.Fatalf("R3AP promoted witness provenance into write authority: %+v", got)
	}
}

func TestActorContextR3APCompleteSyntheticRepositoryEvidencePressureQualifiesGateOnly(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	candidate := actorContextR3APCompleteEvidence(
		"test-only:r3ap/write-authority",
		businessAddress,
		witness,
		"test-only:r3ap/revision-7",
	)
	got, err := actorContextR3APAssessWriteAuthority(
		witness,
		businessAddress,
		[]actorContextR3APWriteAuthorityEvidenceProbe{candidate},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.WriteAuthorityStatus != actorContextR3APStatusQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{
			"test-only:r3ap/write-authority",
		}) ||
		!reflect.DeepEqual(got.AcceptedDecisionSinkRefs, []string{
			"test-only:r3ap/write-authority/accepted-decision-sink",
		}) ||
		!reflect.DeepEqual(got.DecisionRevisions, []string{
			"test-only:r3ap/revision-7",
		}) {
		t.Fatalf("R3AP complete evidence did not pressure-qualify Gate: %+v", got)
	}
}

func TestActorContextR3APMissingWriteAuthorityRequirementDoesNotQualify(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	base := actorContextR3APCompleteEvidence(
		"test-only:r3ap/complete",
		businessAddress,
		witness,
		"test-only:r3ap/revision",
	)

	cases := map[string]func(*actorContextR3APWriteAuthorityEvidenceProbe){
		"W1-repository": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.RepositoryBacked = false
		},
		"W2-kind": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.AuthorityKind = ""
		},
		"W2-declaration": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.AuthorityDeclarationRef = ""
		},
		"W3-assertion-class": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.AssertionClasses = nil
		},
		"W3-decision": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.WitnessDecisions = nil
		},
		"W3-decision-locus": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.AcceptanceDecisionLocusRefs = nil
		},
		"W4-scope": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.ScopeKind = ""
			v.ScopeValue = ""
		},
		"W5-sink": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.AcceptedDecisionSinkRef = ""
		},
		"W6-revision-authority": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.RevisionAuthorityRef = ""
		},
		"W6-current-revision": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.CurrentDecisionRevision = ""
		},
		"W7-write-provenance": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.WriteProvenanceRef = ""
		},
		"derived-only": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.DerivedOnly = true
		},
		"wrong-role": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.EvidenceRole = actorContextR3APRoleAuthoringAcceptancePrecedent
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := actorContextR3APCloneEvidence(base)
			mutate(&candidate)
			got, err := actorContextR3APAssessWriteAuthority(
				witness,
				businessAddress,
				[]actorContextR3APWriteAuthorityEvidenceProbe{candidate},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.WriteAuthorityStatus != actorContextR3APStatusUnresolved ||
				len(got.QualifiedEvidenceIDs) != 0 {
				t.Fatalf("R3AP incomplete candidate %q qualified Gate: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3APWrongClassDecisionLocusOrScopeDoesNotQualify(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	base := actorContextR3APCompleteEvidence(
		"test-only:r3ap/base",
		businessAddress,
		witness,
		"test-only:r3ap/revision",
	)

	cases := map[string]func(*actorContextR3APWriteAuthorityEvidenceProbe){
		"wrong-assertion-class": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.AssertionClasses = []string{"test-only:other-class"}
		},
		"wrong-decision": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.WitnessDecisions = []string{
				actorContextR3AODecisionReject,
			}
		},
		"wrong-decision-locus": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.AcceptanceDecisionLocusRefs = []string{"test-only:other-locus"}
		},
		"wrong-business-address": func(v *actorContextR3APWriteAuthorityEvidenceProbe) {
			v.ScopeKind = actorContextR3APScopeExactBusinessAddress
			v.ScopeValue = "other-procedure/BP-99"
		},
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := actorContextR3APCloneEvidence(base)
			mutate(&candidate)
			got, err := actorContextR3APAssessWriteAuthority(
				witness,
				businessAddress,
				[]actorContextR3APWriteAuthorityEvidenceProbe{candidate},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.WriteAuthorityStatus != actorContextR3APStatusUnresolved {
				t.Fatalf("R3AP mismatched candidate %q qualified Gate: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3APScopeSpecificityOrRevisionDoesNotChooseWinner(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	exact := actorContextR3APCompleteEvidence(
		"test-only:r3ap/exact",
		businessAddress,
		witness,
		"test-only:r3ap/revision-100",
	)
	broad := actorContextR3APCompleteEvidence(
		"test-only:r3ap/broad",
		businessAddress,
		witness,
		"test-only:r3ap/revision-999",
	)
	broad.ScopeKind = actorContextR3APScopeAnyBusinessAddress
	broad.ScopeValue = ""

	got, err := actorContextR3APAssessWriteAuthority(
		witness,
		businessAddress,
		[]actorContextR3APWriteAuthorityEvidenceProbe{exact, broad},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.WriteAuthorityStatus != actorContextR3APStatusMultipleQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{
			"test-only:r3ap/broad",
			"test-only:r3ap/exact",
		}) {
		t.Fatalf("R3AP selected scope/revision winner: %+v", got)
	}
}

func TestActorContextR3APInputOrderDoesNotChangeGateIdentity(t *testing.T) {
	witness, businessAddress := actorContextR3APWitnessFixture(t)
	candidates := []actorContextR3APWriteAuthorityEvidenceProbe{
		actorContextR3APCompleteEvidence(
			"test-only:r3ap/b",
			businessAddress,
			witness,
			"test-only:r3ap/revision-b",
		),
		actorContextR3APCompleteEvidence(
			"test-only:r3ap/a",
			businessAddress,
			witness,
			"test-only:r3ap/revision-a",
		),
	}
	first, err := actorContextR3APAssessWriteAuthority(
		witness, businessAddress, candidates,
	)
	if err != nil {
		t.Fatal(err)
	}
	reversed := append([]actorContextR3APWriteAuthorityEvidenceProbe(nil), candidates...)
	reversed[0], reversed[1] = reversed[1], reversed[0]
	second, err := actorContextR3APAssessWriteAuthority(
		witness, businessAddress, reversed,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AP input order changed Gate identity: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3APGateShapeHasNoCommitAuthorityApplicationOrExecutionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3APWriteAuthorityEvidenceProbe{}),
		reflect.TypeOf(actorContextR3APWriteAuthorityGateProbe{}),
	} {
		for _, forbidden := range []string{
			"AcceptedDecision",
			"AcceptedAuthorityValue",
			"CommittedDecision",
			"Committed",
			"Applied",
			"CurrentAuthority",
			"ReevaluationAuthorityStatus",
			"ReuseAuthorityStatus",
			"RuntimeOutcome",
			"ToolCall",
			"Command",
			"NextAction",
			"CanExecute",
			"Authorized",
			"Allowed",
			"Winner",
			"PreferredAuthority",
			"Confidence",
			"Priority",
			"Severity",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AP Gate introduced forbidden commit/application/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3APWitnessFixture(
	t *testing.T,
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
			Decision:                      actorContextR3AODecisionAccept,
			AcceptanceProvenanceRef:       "test-only:r3ap/acceptance-witness",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AP setup witness count=%d want=1", len(got))
	}
	return got[0], businessAddress
}

func actorContextR3APCurrentInspectedEvidence(
	witness actorContextR3AOObservedAcceptanceWitnessProbe,
) []actorContextR3APWriteAuthorityEvidenceProbe {
	return []actorContextR3APWriteAuthorityEvidenceProbe{
		{
			EvidenceID:              "plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md",
			EvidenceRole:            actorContextR3APRoleAuthoringAcceptancePrecedent,
			RepositoryBacked:        true,
			AuthorityKind:           "shared-authoring-acceptance-boundary",
			AuthorityDeclarationRef: "plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md",
			AcceptedDecisionSinkRef: "authoritative-authoring-source",
			RevisionAuthorityRef:    "authoring-base-model-revision",
			CurrentDecisionRevision: "conceptual",
			WriteProvenanceRef:      "plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md",
		},
		{
			EvidenceID:   "plan/BUSINESS_AUTHORING_EXISTING_ACCEPTANCE_IMPLEMENTATION_REUSE_AUDIT_2026-09.md",
			EvidenceRole: actorContextR3APRoleAuthoringAcceptancePrecedent,
			RepositoryBacked: true,
			AuthorityKind: "audited-authoring-acceptance-precedent",
			AuthorityDeclarationRef: "plan/BUSINESS_AUTHORING_EXISTING_ACCEPTANCE_IMPLEMENTATION_REUSE_AUDIT_2026-09.md",
		},
		{
			EvidenceID:              witness.AcceptanceProvenanceRef,
			EvidenceRole:            actorContextR3APRoleDerivedWitness,
			AuthorityDeclarationRef: witness.AcceptanceProvenanceRef,
			AssertionClasses:        []string{witness.AssertionClass},
			WitnessDecisions:        []string{witness.Decision},
			AcceptanceDecisionLocusRefs: []string{witness.DecisionLocusRef},
			DerivedOnly:             true,
		},
	}
}

func actorContextR3APCompleteEvidence(
	id string,
	businessAddress string,
	witness actorContextR3AOObservedAcceptanceWitnessProbe,
	revision string,
) actorContextR3APWriteAuthorityEvidenceProbe {
	return actorContextR3APWriteAuthorityEvidenceProbe{
		EvidenceID:              id,
		EvidenceRole:            actorContextR3APRoleWriteAuthorityCandidate,
		RepositoryBacked:        true,
		AuthorityKind:           actorContextR3APAuthorityKind,
		AuthorityDeclarationRef: id + "/authority-declaration",
		AssertionClasses:        []string{witness.AssertionClass},
		WitnessDecisions:        []string{witness.Decision},
		AcceptanceDecisionLocusRefs: []string{witness.DecisionLocusRef},
		ScopeKind:               actorContextR3APScopeExactBusinessAddress,
		ScopeValue:              businessAddress,
		AcceptedDecisionSinkRef: id + "/accepted-decision-sink",
		RevisionAuthorityRef:    id + "/revision-authority",
		CurrentDecisionRevision: revision,
		WriteProvenanceRef:      id + "/write-provenance",
	}
}

func actorContextR3APCloneEvidence(
	value actorContextR3APWriteAuthorityEvidenceProbe,
) actorContextR3APWriteAuthorityEvidenceProbe {
	out := value
	out.AssertionClasses = append([]string(nil), value.AssertionClasses...)
	out.WitnessDecisions = append([]string(nil), value.WitnessDecisions...)
	out.AcceptanceDecisionLocusRefs = append(
		[]string(nil), value.AcceptanceDecisionLocusRefs...,
	)
	return out
}

func actorContextR3APAssessWriteAuthority(
	witness actorContextR3AOObservedAcceptanceWitnessProbe,
	businessAddress string,
	candidates []actorContextR3APWriteAuthorityEvidenceProbe,
) (actorContextR3APWriteAuthorityGateProbe, error) {
	if witness.WitnessIdentity == "" ||
		witness.AssertionIdentity == "" ||
		witness.AssertionClass == "" ||
		witness.Decision == "" ||
		witness.DecisionLocusRef == "" ||
		businessAddress == "" {
		return actorContextR3APWriteAuthorityGateProbe{}, fmt.Errorf(
			"R3AP witness/business context is incomplete",
		)
	}

	qualified := []actorContextR3APWriteAuthorityEvidenceProbe{}
	insufficient := []string{}
	seenIDs := map[string]bool{}
	for _, candidate := range candidates {
		if candidate.EvidenceID == "" {
			return actorContextR3APWriteAuthorityGateProbe{}, fmt.Errorf(
				"R3AP write authority evidence id is empty",
			)
		}
		if seenIDs[candidate.EvidenceID] {
			return actorContextR3APWriteAuthorityGateProbe{}, fmt.Errorf(
				"R3AP duplicate write authority evidence id %q",
				candidate.EvidenceID,
			)
		}
		seenIDs[candidate.EvidenceID] = true

		if actorContextR3APCandidateQualifies(
			witness,
			businessAddress,
			candidate,
		) {
			qualified = append(qualified, candidate)
		} else {
			insufficient = append(insufficient, candidate.EvidenceID)
		}
	}

	sort.Slice(qualified, func(i, j int) bool {
		return qualified[i].EvidenceID < qualified[j].EvidenceID
	})

	qualifiedIDs := []string{}
	sinks := []string{}
	revisions := []string{}
	for _, candidate := range qualified {
		qualifiedIDs = append(qualifiedIDs, candidate.EvidenceID)
		sinks = append(sinks, candidate.AcceptedDecisionSinkRef)
		revisions = append(revisions, candidate.CurrentDecisionRevision)
	}

	status := actorContextR3APStatusUnresolved
	switch len(qualified) {
	case 1:
		status = actorContextR3APStatusQualified
	default:
		if len(qualified) > 1 {
			status = actorContextR3APStatusMultipleQualified
		}
	}

	return actorContextR3APWriteAuthorityGateProbe{
		WriteAuthorityStatus:     status,
		QualifiedEvidenceIDs:     actorContextR0SortedUnique(qualifiedIDs),
		AcceptedDecisionSinkRefs: actorContextR0SortedUnique(sinks),
		DecisionRevisions:        actorContextR0SortedUnique(revisions),
		InsufficientEvidenceIDs:  actorContextR0SortedUnique(insufficient),
		AuthorityLimits: actorContextR0SortedUnique([]string{
			"general-authoring-acceptance-does-not-author-operational-decision-write",
			"acceptance-witness-provenance-does-not-create-write-authority",
			"write-authority-evidence-qualified-does-not-commit-witness",
			"write-authority-evidence-qualified-does-not-create-accepted-decision",
			"write-authority-evidence-qualified-does-not-apply-current-authority",
			"scope-specificity-or-revision-does-not-create-precedence",
		}),
	}, nil
}

func actorContextR3APCandidateQualifies(
	witness actorContextR3AOObservedAcceptanceWitnessProbe,
	businessAddress string,
	candidate actorContextR3APWriteAuthorityEvidenceProbe,
) bool {
	if candidate.EvidenceRole != actorContextR3APRoleWriteAuthorityCandidate ||
		!candidate.RepositoryBacked ||
		candidate.DerivedOnly ||
		candidate.AuthorityKind != actorContextR3APAuthorityKind ||
		candidate.AuthorityDeclarationRef == "" ||
		candidate.AcceptedDecisionSinkRef == "" ||
		candidate.RevisionAuthorityRef == "" ||
		candidate.CurrentDecisionRevision == "" ||
		candidate.WriteProvenanceRef == "" ||
		!actorContextR0Contains(candidate.AssertionClasses, witness.AssertionClass) ||
		!actorContextR0Contains(candidate.WitnessDecisions, witness.Decision) ||
		!actorContextR0Contains(
			candidate.AcceptanceDecisionLocusRefs,
			witness.DecisionLocusRef,
		) {
		return false
	}

	switch candidate.ScopeKind {
	case actorContextR3APScopeExactBusinessAddress:
		return candidate.ScopeValue == businessAddress
	case actorContextR3APScopeAnyBusinessAddress:
		return candidate.ScopeValue == ""
	default:
		return false
	}
}

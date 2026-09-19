package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3ANGateInsufficientEvidence = "insufficient-evidence"
	actorContextR3ANGatePressureQualified     = "acceptance-locus-pressure-qualified"

	actorContextR3ANRoleConceptualAcceptance = "conceptual-acceptance-boundary"
	actorContextR3ANRoleOperationalNeed      = "operational-business-need"
	actorContextR3ANRoleAuthorityAxis        = "authority-axis-analysis"
	actorContextR3ANRoleDerivedAssertion     = "derived-assertion"
	actorContextR3ANRoleAcceptanceCandidate  = "acceptance-authority-candidate"
)

type actorContextR3ANAcceptanceEvidenceProbe struct {
	Ref                     string
	Role                    string
	AcceptanceAuthorityKind string
	AssertionClasses        []string
	AcceptanceScope         string
	AcceptedDecisionSink    string
	ProvenanceRef           string
	RepositoryBackedNeed    bool
	DerivedOnly             bool
}

type actorContextR3ANEvidenceRoleProbe struct {
	Ref  string
	Role string
}

type actorContextR3ANAcceptanceGateProbe struct {
	GateStatus                    string
	AssertionClassesUnderPressure []string
	QualifyingCandidateRefs       []string
	InsufficientEvidenceRefs      []string
	EvidenceRoles                 []actorContextR3ANEvidenceRoleProbe
	AuthorityLimits               []string
}

func TestActorContextR3ANCurrentInspectedEvidenceIsInsufficient(t *testing.T) {
	got, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		actorContextR3ANCurrentInspectedEvidence(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.GateStatus != actorContextR3ANGateInsufficientEvidence {
		t.Fatalf("R3AN current inspected evidence unexpectedly qualified acceptance locus: %+v", got)
	}
	if len(got.QualifyingCandidateRefs) != 0 {
		t.Fatalf("R3AN current evidence fabricated qualifying candidate: %+v", got)
	}
	if len(got.InsufficientEvidenceRefs) != len(actorContextR3ANCurrentInspectedEvidence()) {
		t.Fatalf("R3AN lost inspected insufficient evidence refs: %+v", got)
	}
}

func TestActorContextR3ANGeneralSharedAcceptanceConceptDoesNotQualifyOperationalLocus(t *testing.T) {
	evidence := actorContextR3ANAcceptanceEvidenceProbe{
		Ref:  "plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md",
		Role: actorContextR3ANRoleConceptualAcceptance,
	}
	got, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		[]actorContextR3ANAcceptanceEvidenceProbe{evidence},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.GateStatus != actorContextR3ANGateInsufficientEvidence ||
		len(got.QualifyingCandidateRefs) != 0 {
		t.Fatalf("R3AN promoted general Shared Acceptance concept into Operational acceptance authority: %+v", got)
	}
}

func TestActorContextR3ANAssertionProvenanceOrBusinessNeedAloneDoesNotQualify(t *testing.T) {
	evidence := []actorContextR3ANAcceptanceEvidenceProbe{
		{
			Ref:         "plan/ACTOR_CONTEXT_R3AM_QUALIFIED_EVIDENCE_AUTHORITY_ASSERTION_2026-09.md",
			Role:        actorContextR3ANRoleDerivedAssertion,
			DerivedOnly: true,
		},
		{
			Ref:                  "reference/business-system-design/order-entry-business-procedure.json",
			Role:                 actorContextR3ANRoleOperationalNeed,
			RepositoryBackedNeed: true,
		},
	}
	got, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		evidence,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.GateStatus != actorContextR3ANGateInsufficientEvidence {
		t.Fatalf("R3AN combined assertion provenance + business need into fake acceptance locus: %+v", got)
	}
}

func TestActorContextR3ANCompleteSyntheticCandidatePressureQualifiesGateOnly(t *testing.T) {
	candidate := actorContextR3ANCompleteSyntheticCandidate(
		"test-only:r3an/acceptance-candidate-a",
		"test-only:r3an/accepted-decision-sink-a",
	)
	got, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		[]actorContextR3ANAcceptanceEvidenceProbe{candidate},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.GateStatus != actorContextR3ANGatePressureQualified ||
		!reflect.DeepEqual(got.QualifyingCandidateRefs, []string{
			"test-only:r3an/acceptance-candidate-a",
		}) {
		t.Fatalf("R3AN complete synthetic candidate did not pressure-qualify gate: %+v", got)
	}
}

func TestActorContextR3ANIncompleteCandidateDoesNotQualify(t *testing.T) {
	base := actorContextR3ANCompleteSyntheticCandidate(
		"test-only:r3an/base",
		"test-only:r3an/sink",
	)
	cases := map[string]func(*actorContextR3ANAcceptanceEvidenceProbe){
		"authority-kind": func(v *actorContextR3ANAcceptanceEvidenceProbe) {
			v.AcceptanceAuthorityKind = ""
		},
		"assertion-class": func(v *actorContextR3ANAcceptanceEvidenceProbe) {
			v.AssertionClasses = nil
		},
		"scope": func(v *actorContextR3ANAcceptanceEvidenceProbe) {
			v.AcceptanceScope = ""
		},
		"sink": func(v *actorContextR3ANAcceptanceEvidenceProbe) {
			v.AcceptedDecisionSink = ""
		},
		"provenance": func(v *actorContextR3ANAcceptanceEvidenceProbe) {
			v.ProvenanceRef = ""
		},
		"business-need": func(v *actorContextR3ANAcceptanceEvidenceProbe) {
			v.RepositoryBackedNeed = false
		},
		"derived-only": func(v *actorContextR3ANAcceptanceEvidenceProbe) {
			v.DerivedOnly = true
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := base
			candidate.AssertionClasses = append([]string(nil), base.AssertionClasses...)
			mutate(&candidate)
			got, err := actorContextR3ANEvaluateAcceptanceLocusGate(
				[]string{"reevaluation-authority"},
				[]actorContextR3ANAcceptanceEvidenceProbe{candidate},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.GateStatus != actorContextR3ANGateInsufficientEvidence ||
				len(got.QualifyingCandidateRefs) != 0 {
				t.Fatalf("R3AN incomplete candidate %q qualified: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3ANCandidateMustApplyToExactAssertionClass(t *testing.T) {
	candidate := actorContextR3ANCompleteSyntheticCandidate(
		"test-only:r3an/reuse-only",
		"test-only:r3an/reuse-sink",
	)
	candidate.AssertionClasses = []string{"prior-observation-reuse-authority"}

	got, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		[]actorContextR3ANAcceptanceEvidenceProbe{candidate},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.GateStatus != actorContextR3ANGateInsufficientEvidence {
		t.Fatalf("R3AN candidate for another assertion class qualified current gate: %+v", got)
	}
}

func TestActorContextR3ANMultipleQualifiedCandidatesDoNotSelectWinner(t *testing.T) {
	candidates := []actorContextR3ANAcceptanceEvidenceProbe{
		actorContextR3ANCompleteSyntheticCandidate(
			"test-only:r3an/candidate-b",
			"test-only:r3an/sink-b",
		),
		actorContextR3ANCompleteSyntheticCandidate(
			"test-only:r3an/candidate-a",
			"test-only:r3an/sink-a",
		),
	}
	got, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		candidates,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.GateStatus != actorContextR3ANGatePressureQualified ||
		!reflect.DeepEqual(got.QualifyingCandidateRefs, []string{
			"test-only:r3an/candidate-a",
			"test-only:r3an/candidate-b",
		}) {
		t.Fatalf("R3AN collapsed multiple qualifying candidates: %+v", got)
	}
}

func TestActorContextR3ANInputOrderDoesNotChangeGateIdentity(t *testing.T) {
	evidence := append(
		actorContextR3ANCurrentInspectedEvidence(),
		actorContextR3ANCompleteSyntheticCandidate(
			"test-only:r3an/candidate",
			"test-only:r3an/sink",
		),
	)
	first, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		evidence,
	)
	if err != nil {
		t.Fatal(err)
	}

	reversed := append([]actorContextR3ANAcceptanceEvidenceProbe(nil), evidence...)
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}
	second, err := actorContextR3ANEvaluateAcceptanceLocusGate(
		[]string{"reevaluation-authority"},
		reversed,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AN input order changed gate identity: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3ANGateShapeHasNoAcceptanceOrExecutionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3ANAcceptanceGateProbe{})
	for _, forbidden := range []string{
		"AcceptedAuthorityValue",
		"AcceptedDecision",
		"SelectedAcceptanceLocus",
		"Winner",
		"PreferredCandidate",
		"ApprovalStatus",
		"Apply",
		"AutoApply",
		"CurrentAuthority",
		"NextAction",
		"HandoffOwner",
		"CanExecute",
		"Authorized",
		"Allowed",
		"Confidence",
		"Priority",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AN Evidence Gate introduced forbidden acceptance/action shortcut %q", forbidden)
		}
	}
}

func actorContextR3ANCurrentInspectedEvidence() []actorContextR3ANAcceptanceEvidenceProbe {
	return []actorContextR3ANAcceptanceEvidenceProbe{
		{
			Ref:  "plan/ACTOR_CONTEXT_PROJECTIONS_2026-09.md",
			Role: actorContextR3ANRoleConceptualAcceptance,
		},
		{
			Ref:  "plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md",
			Role: actorContextR3ANRoleAuthorityAxis,
		},
		{
			Ref:         "plan/ACTOR_CONTEXT_R3AM_QUALIFIED_EVIDENCE_AUTHORITY_ASSERTION_2026-09.md",
			Role:        actorContextR3ANRoleDerivedAssertion,
			DerivedOnly: true,
		},
		{
			Ref:                  "reference/business-system-design/order-entry-business-procedure.json",
			Role:                 actorContextR3ANRoleOperationalNeed,
			RepositoryBackedNeed: true,
		},
	}
}

func actorContextR3ANCompleteSyntheticCandidate(
	ref string,
	sink string,
) actorContextR3ANAcceptanceEvidenceProbe {
	return actorContextR3ANAcceptanceEvidenceProbe{
		Ref:                     ref,
		Role:                    actorContextR3ANRoleAcceptanceCandidate,
		AcceptanceAuthorityKind: "test-only:operational-authority-assertion-acceptor",
		AssertionClasses:        []string{"reevaluation-authority"},
		AcceptanceScope:         "test-only:order-entry.order-handling/BP-06",
		AcceptedDecisionSink:    sink,
		ProvenanceRef:           ref + "/provenance",
		RepositoryBackedNeed:    true,
	}
}

func actorContextR3ANEvaluateAcceptanceLocusGate(
	assertionClasses []string,
	evidence []actorContextR3ANAcceptanceEvidenceProbe,
) (actorContextR3ANAcceptanceGateProbe, error) {
	classes := actorContextR0SortedUnique(append([]string(nil), assertionClasses...))
	if len(classes) == 0 {
		return actorContextR3ANAcceptanceGateProbe{}, fmt.Errorf("R3AN assertion class pressure is empty")
	}
	for _, class := range classes {
		if class == "" {
			return actorContextR3ANAcceptanceGateProbe{}, fmt.Errorf("R3AN assertion class must not be empty")
		}
	}

	var qualifying []string
	var insufficient []string
	var roles []actorContextR3ANEvidenceRoleProbe
	seenRefs := map[string]bool{}
	for _, item := range evidence {
		if item.Ref == "" || item.Role == "" {
			return actorContextR3ANAcceptanceGateProbe{}, fmt.Errorf("R3AN evidence identity/role is incomplete")
		}
		if seenRefs[item.Ref] {
			return actorContextR3ANAcceptanceGateProbe{}, fmt.Errorf("R3AN duplicate evidence ref %q", item.Ref)
		}
		seenRefs[item.Ref] = true
		roles = append(roles, actorContextR3ANEvidenceRoleProbe{
			Ref:  item.Ref,
			Role: item.Role,
		})

		if actorContextR3ANQualifiesAcceptanceCandidate(item, classes) {
			qualifying = append(qualifying, item.Ref)
		} else {
			insufficient = append(insufficient, item.Ref)
		}
	}
	sort.Strings(qualifying)
	sort.Strings(insufficient)
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Ref < roles[j].Ref
	})

	status := actorContextR3ANGateInsufficientEvidence
	if len(qualifying) > 0 {
		status = actorContextR3ANGatePressureQualified
	}
	return actorContextR3ANAcceptanceGateProbe{
		GateStatus:                    status,
		AssertionClassesUnderPressure: classes,
		QualifyingCandidateRefs:       qualifying,
		InsufficientEvidenceRefs:      insufficient,
		EvidenceRoles:                 roles,
		AuthorityLimits: []string{
			"assertion-provenance-does-not-provide-acceptance-authority",
			"general-shared-acceptance-does-not-provide-operational-acceptance-locus",
			"business-need-does-not-select-acceptance-owner",
			"pressure-qualified-gate-does-not-accept-assertion",
			"multiple-candidates-do-not-select-winner-or-precedence",
			"gate-does-not-create-current-authority-or-execution-permission",
		},
	}, nil
}

func actorContextR3ANQualifiesAcceptanceCandidate(
	item actorContextR3ANAcceptanceEvidenceProbe,
	assertionClasses []string,
) bool {
	if item.Role != actorContextR3ANRoleAcceptanceCandidate ||
		item.DerivedOnly ||
		item.AcceptanceAuthorityKind == "" ||
		item.AcceptanceScope == "" ||
		item.AcceptedDecisionSink == "" ||
		item.ProvenanceRef == "" ||
		!item.RepositoryBackedNeed {
		return false
	}
	itemClasses := actorContextR0SortedUnique(append([]string(nil), item.AssertionClasses...))
	if len(itemClasses) == 0 {
		return false
	}
	for _, pressureClass := range assertionClasses {
		if !actorContextR0Contains(itemClasses, pressureClass) {
			return false
		}
	}
	return true
}

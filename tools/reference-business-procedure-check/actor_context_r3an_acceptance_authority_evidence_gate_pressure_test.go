package main

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

const (
	actorContextR3ANStatusUnresolved = "acceptance-authority-unresolved"
	actorContextR3ANStatusQualified = "acceptance-authority-evidence-qualified"
	actorContextR3ANStatusMultipleQualified = "acceptance-authority-multiple-qualified-precedence-unresolved"

	actorContextR3ANAuthorityKind = "assertion-acceptance-authority"
	actorContextR3ANScopeExactBusinessAddress = "exact-business-address"
	actorContextR3ANScopeAnyBusinessAddress = "any-business-address"
)

type actorContextR3ANAcceptanceAuthorityEvidenceProbe struct {
	EvidenceID               string
	RepositoryBacked         bool
	AuthorityKind            string
	AuthorityDeclarationRef  string
	AssertionClasses         []string
	ScopeKind                string
	ScopeValue               string
	DecisionLocusRef         string
	DecisionSemantics        []string
}

type actorContextR3ANAcceptanceAuthorityGateProbe struct {
	AcceptanceAuthorityStatus string
	QualifiedEvidenceIDs       []string
	UnresolvedReasons          []string
	AuthorityLimits            []string
}

func TestActorContextR3ANCurrentCorpusKeepsAcceptanceAuthorityUnresolved(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	got, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion,
		"order-entry.order-handling/BP-06",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.AcceptanceAuthorityStatus != actorContextR3ANStatusUnresolved ||
		len(got.QualifiedEvidenceIDs) != 0 {
		t.Fatalf("R3AN fabricated acceptance authority from current assertion context: %+v", got)
	}
}

func TestActorContextR3ANAssertionSourceAndCorroborationDoNotCreateAcceptanceAuthority(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	second := assertion
	second.EvidenceRef += ":second"
	second.AssertionIdentity += ":second"
	second.AssertionSourceRef += ":second"

	for _, assertions := range [][]actorContextR3AMQualifiedAuthorityAssertionProbe{
		{assertion},
		{assertion, second},
	} {
		for _, item := range assertions {
			got, err := actorContextR3ANAssessAcceptanceAuthority(
				item,
				"order-entry.order-handling/BP-06",
				[]actorContextR3ANAcceptanceAuthorityEvidenceProbe{{
					EvidenceID:              item.AssertionSourceRef,
					RepositoryBacked:        false,
					AuthorityKind:           "",
					AuthorityDeclarationRef: item.AssertionSourceRef,
					AssertionClasses:        []string{item.AssertionClass},
					ScopeKind:               actorContextR3ANScopeExactBusinessAddress,
					ScopeValue:              "order-entry.order-handling/BP-06",
					DecisionLocusRef:        "",
					DecisionSemantics:       nil,
				}},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.AcceptanceAuthorityStatus != actorContextR3ANStatusUnresolved {
				t.Fatalf("R3AN promoted assertion source/count into acceptance authority: %+v", got)
			}
		}
	}
}

func TestActorContextR3ANCompleteRepositoryBackedEvidenceQualifiesWithoutAcceptingAssertion(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	got, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion,
		"order-entry.order-handling/BP-06",
		[]actorContextR3ANAcceptanceAuthorityEvidenceProbe{
			actorContextR3ANCompleteEvidence(
				"test-only:r3an/acceptance-authority",
				actorContextR3ANScopeExactBusinessAddress,
				"order-entry.order-handling/BP-06",
				assertion.AssertionClass,
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.AcceptanceAuthorityStatus != actorContextR3ANStatusQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{
			"test-only:r3an/acceptance-authority",
		}) {
		t.Fatalf("R3AN did not qualify complete acceptance authority evidence: %+v", got)
	}
}

func TestActorContextR3ANMissingQualifyingRequirementDoesNotQualify(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	base := actorContextR3ANCompleteEvidence(
		"test-only:r3an/complete",
		actorContextR3ANScopeExactBusinessAddress,
		"order-entry.order-handling/BP-06",
		assertion.AssertionClass,
	)

	cases := map[string]actorContextR3ANAcceptanceAuthorityEvidenceProbe{
		"E1-repository": func() actorContextR3ANAcceptanceAuthorityEvidenceProbe {
			x := base; x.RepositoryBacked = false; return x
		}(),
		"E2-kind": func() actorContextR3ANAcceptanceAuthorityEvidenceProbe {
			x := base; x.AuthorityKind = ""; return x
		}(),
		"E2-declaration-ref": func() actorContextR3ANAcceptanceAuthorityEvidenceProbe {
			x := base; x.AuthorityDeclarationRef = ""; return x
		}(),
		"E3-class": func() actorContextR3ANAcceptanceAuthorityEvidenceProbe {
			x := base; x.AssertionClasses = nil; return x
		}(),
		"E4-scope": func() actorContextR3ANAcceptanceAuthorityEvidenceProbe {
			x := base; x.ScopeKind = ""; x.ScopeValue = ""; return x
		}(),
		"E5-locus": func() actorContextR3ANAcceptanceAuthorityEvidenceProbe {
			x := base; x.DecisionLocusRef = ""; return x
		}(),
		"E6-semantics": func() actorContextR3ANAcceptanceAuthorityEvidenceProbe {
			x := base; x.DecisionSemantics = []string{"observe"}; return x
		}(),
	}

	for name, candidate := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := actorContextR3ANAssessAcceptanceAuthority(
				assertion,
				"order-entry.order-handling/BP-06",
				[]actorContextR3ANAcceptanceAuthorityEvidenceProbe{candidate},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.AcceptanceAuthorityStatus != actorContextR3ANStatusUnresolved ||
				len(got.QualifiedEvidenceIDs) != 0 {
				t.Fatalf("R3AN qualified incomplete acceptance authority evidence %s: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3ANWrongAssertionClassOrScopeDoesNotQualify(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	cases := map[string]actorContextR3ANAcceptanceAuthorityEvidenceProbe{
		"wrong-class": actorContextR3ANCompleteEvidence(
			"test-only:r3an/wrong-class",
			actorContextR3ANScopeExactBusinessAddress,
			"order-entry.order-handling/BP-06",
			"test-only:other-assertion-class",
		),
		"wrong-scope": actorContextR3ANCompleteEvidence(
			"test-only:r3an/wrong-scope",
			actorContextR3ANScopeExactBusinessAddress,
			"other-procedure/BP-99",
			assertion.AssertionClass,
		),
	}
	for name, candidate := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := actorContextR3ANAssessAcceptanceAuthority(
				assertion,
				"order-entry.order-handling/BP-06",
				[]actorContextR3ANAcceptanceAuthorityEvidenceProbe{candidate},
			)
			if err != nil {
				t.Fatal(err)
			}
			if got.AcceptanceAuthorityStatus != actorContextR3ANStatusUnresolved {
				t.Fatalf("R3AN qualified mismatched %s evidence: %+v", name, got)
			}
		})
	}
}

func TestActorContextR3ANScopeSpecificityDoesNotChooseWinner(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	exact := actorContextR3ANCompleteEvidence(
		"test-only:r3an/exact",
		actorContextR3ANScopeExactBusinessAddress,
		"order-entry.order-handling/BP-06",
		assertion.AssertionClass,
	)
	broad := actorContextR3ANCompleteEvidence(
		"test-only:r3an/broad",
		actorContextR3ANScopeAnyBusinessAddress,
		"",
		assertion.AssertionClass,
	)

	got, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion,
		"order-entry.order-handling/BP-06",
		[]actorContextR3ANAcceptanceAuthorityEvidenceProbe{exact, broad},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.AcceptanceAuthorityStatus != actorContextR3ANStatusMultipleQualified ||
		!reflect.DeepEqual(got.QualifiedEvidenceIDs, []string{
			"test-only:r3an/broad",
			"test-only:r3an/exact",
		}) {
		t.Fatalf("R3AN used scope specificity as implicit precedence: %+v", got)
	}
}

func TestActorContextR3ANInputOrderDoesNotChangeGateResult(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	candidates := []actorContextR3ANAcceptanceAuthorityEvidenceProbe{
		actorContextR3ANCompleteEvidence(
			"test-only:r3an/broad",
			actorContextR3ANScopeAnyBusinessAddress,
			"",
			assertion.AssertionClass,
		),
		actorContextR3ANCompleteEvidence(
			"test-only:r3an/exact",
			actorContextR3ANScopeExactBusinessAddress,
			"order-entry.order-handling/BP-06",
			assertion.AssertionClass,
		),
	}
	first, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion, "order-entry.order-handling/BP-06", candidates,
	)
	if err != nil {
		t.Fatal(err)
	}
	reversed := append([]actorContextR3ANAcceptanceAuthorityEvidenceProbe(nil), candidates...)
	reversed[0], reversed[1] = reversed[1], reversed[0]
	second, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion, "order-entry.order-handling/BP-06", reversed,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AN input order changed normalized gate result: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3ANGateShapeHasNoAcceptanceOrExecutionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3ANAcceptanceAuthorityEvidenceProbe{}),
		reflect.TypeOf(actorContextR3ANAcceptanceAuthorityGateProbe{}),
	} {
		for _, forbidden := range []string{
			"AcceptedAuthorityValue", "AcceptedDecision", "AcceptanceDecision",
			"SelectedAuthority", "PreferredAuthority", "Winner", "CurrentAuthority",
			"ReevaluationAuthorityStatus", "ReuseAuthorityStatus", "Approved", "Rejected",
			"Resolved", "Apply", "AutoApply", "ToolCall", "Command", "NextAction",
			"CanExecute", "Authorized", "Allowed", "Confidence", "Priority", "Severity",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AN gate introduced forbidden acceptance/action shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3ANAssertionFixture(t *testing.T) actorContextR3AMQualifiedAuthorityAssertionProbe {
	t.Helper()
	assessments := actorContextR3AMQualifiedAssessments(t, []string{
		"test-only:r3an/evidence",
	})
	assertions, err := actorContextR3AMMaterializeAssertions(
		assessments,
		[]actorContextR3AMAuthorityAssertionWitnessProbe{{
			EvidenceRef:        "test-only:r3an/evidence",
			AssertionClass:     "reevaluation-authority",
			AssertedValue:      actorContextR3YReevaluationRequired,
			AssertionSourceRef: "test-only:r3an/assertion-source",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(assertions) != 1 {
		t.Fatalf("R3AN setup assertion count=%d want=1", len(assertions))
	}
	return assertions[0]
}

func actorContextR3ANCompleteEvidence(
	id string,
	scopeKind string,
	scopeValue string,
	assertionClass string,
) actorContextR3ANAcceptanceAuthorityEvidenceProbe {
	return actorContextR3ANAcceptanceAuthorityEvidenceProbe{
		EvidenceID:              id,
		RepositoryBacked:        true,
		AuthorityKind:           actorContextR3ANAuthorityKind,
		AuthorityDeclarationRef: id + "/authority-declaration",
		AssertionClasses:        []string{assertionClass},
		ScopeKind:               scopeKind,
		ScopeValue:              scopeValue,
		DecisionLocusRef:        id + "/decision-locus",
		DecisionSemantics:       []string{"accept", "reject"},
	}
}

func actorContextR3ANAssessAcceptanceAuthority(
	assertion actorContextR3AMQualifiedAuthorityAssertionProbe,
	businessAddress string,
	candidates []actorContextR3ANAcceptanceAuthorityEvidenceProbe,
) (actorContextR3ANAcceptanceAuthorityGateProbe, error) {
	if assertion.AssertionIdentity == "" || assertion.AssertionClass == "" ||
		assertion.EvidenceRef == "" || businessAddress == "" {
		return actorContextR3ANAcceptanceAuthorityGateProbe{}, fmt.Errorf(
			"R3AN assertion/business context is incomplete",
		)
	}

	qualified := []string{}
	reasons := []string{}
	for _, candidate := range candidates {
		if candidate.EvidenceID == "" {
			return actorContextR3ANAcceptanceAuthorityGateProbe{}, fmt.Errorf(
				"R3AN acceptance authority evidence id is empty",
			)
		}
		if !actorContextR3ANCandidateQualifies(assertion, businessAddress, candidate) {
			reasons = append(reasons, "candidate-not-qualified:"+candidate.EvidenceID)
			continue
		}
		qualified = append(qualified, candidate.EvidenceID)
	}
	qualified = actorContextR0SortedUnique(qualified)
	reasons = actorContextR0SortedUnique(reasons)

	status := actorContextR3ANStatusUnresolved
	switch len(qualified) {
	case 1:
		status = actorContextR3ANStatusQualified
	default:
		if len(qualified) > 1 {
			status = actorContextR3ANStatusMultipleQualified
			reasons = actorContextR0SortedUnique(append(
				reasons,
				"multiple-qualified-acceptance-authorities-precedence-unresolved",
			))
		} else {
			reasons = actorContextR0SortedUnique(append(
				reasons,
				"repository-backed-acceptance-authority-evidence-unresolved",
			))
		}
	}

	out := actorContextR3ANAcceptanceAuthorityGateProbe{
		AcceptanceAuthorityStatus: status,
		QualifiedEvidenceIDs:      qualified,
		UnresolvedReasons:         reasons,
		AuthorityLimits: []string{
			"assertion-provenance-does-not-create-acceptance-authority",
			"corroboration-count-does-not-create-acceptance-authority",
			"scope-specificity-does-not-create-precedence",
			"source-location-does-not-create-acceptance-authority",
			"qualified-acceptance-authority-evidence-does-not-accept-assertion",
			"acceptance-authority-gate-does-not-authorize-operation",
		},
	}
	sort.Strings(out.AuthorityLimits)
	return out, nil
}

func actorContextR3ANCandidateQualifies(
	assertion actorContextR3AMQualifiedAuthorityAssertionProbe,
	businessAddress string,
	candidate actorContextR3ANAcceptanceAuthorityEvidenceProbe,
) bool {
	if !candidate.RepositoryBacked ||
		candidate.AuthorityKind != actorContextR3ANAuthorityKind ||
		candidate.AuthorityDeclarationRef == "" ||
		candidate.DecisionLocusRef == "" ||
		!actorContextR0Contains(candidate.AssertionClasses, assertion.AssertionClass) ||
		!actorContextR0Contains(candidate.DecisionSemantics, "accept") ||
		!actorContextR0Contains(candidate.DecisionSemantics, "reject") {
		return false
	}
	switch candidate.ScopeKind {
	case actorContextR3ANScopeExactBusinessAddress:
		return candidate.ScopeValue == businessAddress
	case actorContextR3ANScopeAnyBusinessAddress:
		return candidate.ScopeValue == ""
	default:
		return false
	}
}

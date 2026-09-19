package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3AODecisionAccept = "accept"
	actorContextR3AODecisionReject = "reject"
)

type actorContextR3AOAcceptanceWitnessProbe struct {
	AssertionIdentity             string
	AcceptanceAuthorityEvidenceID string
	Decision                      string
	AcceptanceProvenanceRef       string
}

type actorContextR3AOObservedAcceptanceWitnessProbe struct {
	AssertionIdentity             string
	AssertionClass                string
	AssertedValue                 string
	AcceptanceAuthorityEvidenceID string
	AuthorityDeclarationRef       string
	DecisionLocusRef              string
	Decision                      string
	AcceptanceProvenanceRef       string
	WitnessIdentity               string
	AuthorityLimits               []string
}

func TestActorContextR3AOCurrentUnresolvedGateCannotBindWitness(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	gate, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion,
		"order-entry.order-handling/BP-06",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := actorContextR3AOBindWitnesses(
		assertion,
		"order-entry.order-handling/BP-06",
		gate,
		nil,
		[]actorContextR3AOAcceptanceWitnessProbe{{
			AssertionIdentity:             assertion.AssertionIdentity,
			AcceptanceAuthorityEvidenceID: "test-only:r3ao/not-qualified",
			Decision:                      actorContextR3AODecisionAccept,
			AcceptanceProvenanceRef:       "test-only:r3ao/provenance",
		}},
	); err == nil {
		t.Fatal("R3AO bound acceptance witness while R3AN authority remained unresolved")
	}
}

func TestActorContextR3AOExactExplicitAcceptWitnessBindsWithoutApplyingAuthority(t *testing.T) {
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
			AcceptanceProvenanceRef:       "test-only:r3ao/accept-witness",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("R3AO accept witness count=%d want=1: %+v", len(got), got)
	}
	if got[0].AssertionIdentity != assertion.AssertionIdentity ||
		got[0].AssertionClass != assertion.AssertionClass ||
		got[0].AssertedValue != assertion.AssertedValue ||
		got[0].Decision != actorContextR3AODecisionAccept ||
		got[0].DecisionLocusRef == "" ||
		got[0].AuthorityDeclarationRef == "" ||
		got[0].WitnessIdentity == "" {
		t.Fatalf("R3AO lost exact witness/authority provenance: %+v", got[0])
	}
}

func TestActorContextR3AOSingleQualifiedAuthorityWithoutWitnessDoesNotInferDecision(t *testing.T) {
	assertion, businessAddress, gate, candidates := actorContextR3AOQualifiedFixture(t)
	got, err := actorContextR3AOBindWitnesses(
		assertion,
		businessAddress,
		gate,
		candidates,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("R3AO inferred accept/reject from qualified authority without witness: %+v", got)
	}
}

func TestActorContextR3AORejectsWrongAssertionEvidenceDecisionOrProvenance(t *testing.T) {
	assertion, businessAddress, gate, candidates := actorContextR3AOQualifiedFixture(t)
	base := actorContextR3AOAcceptanceWitnessProbe{
		AssertionIdentity:             assertion.AssertionIdentity,
		AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
		Decision:                      actorContextR3AODecisionAccept,
		AcceptanceProvenanceRef:       "test-only:r3ao/provenance",
	}
	cases := map[string]actorContextR3AOAcceptanceWitnessProbe{
		"wrong-assertion": func() actorContextR3AOAcceptanceWitnessProbe {
			x := base
			x.AssertionIdentity += ":stale"
			return x
		}(),
		"wrong-authority-evidence": func() actorContextR3AOAcceptanceWitnessProbe {
			x := base
			x.AcceptanceAuthorityEvidenceID = "test-only:r3ao/other-authority"
			return x
		}(),
		"missing-decision": func() actorContextR3AOAcceptanceWitnessProbe {
			x := base
			x.Decision = ""
			return x
		}(),
		"unsupported-decision": func() actorContextR3AOAcceptanceWitnessProbe {
			x := base
			x.Decision = "approve"
			return x
		}(),
		"missing-provenance": func() actorContextR3AOAcceptanceWitnessProbe {
			x := base
			x.AcceptanceProvenanceRef = ""
			return x
		}(),
	}
	for name, witness := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := actorContextR3AOBindWitnesses(
				assertion,
				businessAddress,
				gate,
				candidates,
				[]actorContextR3AOAcceptanceWitnessProbe{witness},
			); err == nil {
				t.Fatalf("R3AO accepted invalid witness %q", name)
			}
		})
	}
}

func TestActorContextR3AOMissingCurrentAuthorityCandidateFailsClosed(t *testing.T) {
	assertion, businessAddress, gate, _ := actorContextR3AOQualifiedFixture(t)
	if _, err := actorContextR3AOBindWitnesses(
		assertion,
		businessAddress,
		gate,
		nil,
		[]actorContextR3AOAcceptanceWitnessProbe{{
			AssertionIdentity:             assertion.AssertionIdentity,
			AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
			Decision:                      actorContextR3AODecisionAccept,
			AcceptanceProvenanceRef:       "test-only:r3ao/provenance",
		}},
	); err == nil {
		t.Fatal("R3AO accepted gate ID without current acceptance-authority candidate")
	}
}

func TestActorContextR3AOMultipleQualifiedPrecedenceUnresolvedCannotBeBypassedByWitness(t *testing.T) {
	assertion := actorContextR3ANAssertionFixture(t)
	businessAddress := "order-entry.order-handling/BP-06"
	exact := actorContextR3ANCompleteEvidence(
		"test-only:r3ao/exact",
		actorContextR3ANScopeExactBusinessAddress,
		businessAddress,
		assertion.AssertionClass,
	)
	broad := actorContextR3ANCompleteEvidence(
		"test-only:r3ao/broad",
		actorContextR3ANScopeAnyBusinessAddress,
		"",
		assertion.AssertionClass,
	)
	gate, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion,
		businessAddress,
		[]actorContextR3ANAcceptanceAuthorityEvidenceProbe{exact, broad},
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.AcceptanceAuthorityStatus != actorContextR3ANStatusMultipleQualified {
		t.Fatalf("R3AO setup lost multiple-qualified state: %+v", gate)
	}
	if _, err := actorContextR3AOBindWitnesses(
		assertion,
		businessAddress,
		gate,
		[]actorContextR3ANAcceptanceAuthorityEvidenceProbe{exact, broad},
		[]actorContextR3AOAcceptanceWitnessProbe{{
			AssertionIdentity:             assertion.AssertionIdentity,
			AcceptanceAuthorityEvidenceID: exact.EvidenceID,
			Decision:                      actorContextR3AODecisionAccept,
			AcceptanceProvenanceRef:       "test-only:r3ao/exact-witness",
		}},
	); err == nil {
		t.Fatal("R3AO allowed exact witness to bypass unresolved authority precedence")
	}
}

func TestActorContextR3AOConflictingExplicitWitnessesRemainIndependentWithoutWinner(t *testing.T) {
	assertion, businessAddress, gate, candidates := actorContextR3AOQualifiedFixture(t)
	witnesses := []actorContextR3AOAcceptanceWitnessProbe{
		{
			AssertionIdentity:             assertion.AssertionIdentity,
			AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
			Decision:                      actorContextR3AODecisionAccept,
			AcceptanceProvenanceRef:       "test-only:r3ao/witness-a",
		},
		{
			AssertionIdentity:             assertion.AssertionIdentity,
			AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
			Decision:                      actorContextR3AODecisionReject,
			AcceptanceProvenanceRef:       "test-only:r3ao/witness-b",
		},
	}
	got, err := actorContextR3AOBindWitnesses(
		assertion, businessAddress, gate, candidates, witnesses,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("R3AO collapsed conflicting explicit witnesses: %+v", got)
	}
	decisions := []string{got[0].Decision, got[1].Decision}
	sort.Strings(decisions)
	if !reflect.DeepEqual(decisions, []string{
		actorContextR3AODecisionAccept,
		actorContextR3AODecisionReject,
	}) {
		t.Fatalf("R3AO changed conflicting witness decisions: %+v", got)
	}
}

func TestActorContextR3AOExactDuplicateNormalizesAndInputOrderIsStable(t *testing.T) {
	assertion, businessAddress, gate, candidates := actorContextR3AOQualifiedFixture(t)
	a := actorContextR3AOAcceptanceWitnessProbe{
		AssertionIdentity:             assertion.AssertionIdentity,
		AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
		Decision:                      actorContextR3AODecisionAccept,
		AcceptanceProvenanceRef:       "test-only:r3ao/witness-a",
	}
	b := actorContextR3AOAcceptanceWitnessProbe{
		AssertionIdentity:             assertion.AssertionIdentity,
		AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
		Decision:                      actorContextR3AODecisionReject,
		AcceptanceProvenanceRef:       "test-only:r3ao/witness-b",
	}
	first, err := actorContextR3AOBindWitnesses(
		assertion, businessAddress, gate, candidates,
		[]actorContextR3AOAcceptanceWitnessProbe{a, a, b},
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actorContextR3AOBindWitnesses(
		assertion, businessAddress, gate, candidates,
		[]actorContextR3AOAcceptanceWitnessProbe{b, a},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 2 || !reflect.DeepEqual(first, second) {
		t.Fatalf("R3AO duplicate/order normalization drifted: first=%+v second=%+v", first, second)
	}
}

func TestActorContextR3AORejectWitnessDoesNotBecomeAuthorizationDenied(t *testing.T) {
	assertion, businessAddress, gate, candidates := actorContextR3AOQualifiedFixture(t)
	got, err := actorContextR3AOBindWitnesses(
		assertion,
		businessAddress,
		gate,
		candidates,
		[]actorContextR3AOAcceptanceWitnessProbe{{
			AssertionIdentity:             assertion.AssertionIdentity,
			AcceptanceAuthorityEvidenceID: gate.QualifiedEvidenceIDs[0],
			Decision:                      actorContextR3AODecisionReject,
			AcceptanceProvenanceRef:       "test-only:r3ao/reject-witness",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Decision != actorContextR3AODecisionReject {
		t.Fatalf("R3AO lost explicit reject witness: %+v", got)
	}
}

func TestActorContextR3AOShapeHasNoAcceptedDecisionAuthorityOrExecutionShortcut(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeOf(actorContextR3AOAcceptanceWitnessProbe{}),
		reflect.TypeOf(actorContextR3AOObservedAcceptanceWitnessProbe{}),
	} {
		for _, forbidden := range []string{
			"AcceptedAuthorityValue", "AcceptedDecision", "AcceptanceStatus",
			"CurrentAuthority", "ReevaluationAuthorityStatus", "ReuseAuthorityStatus",
			"RuntimeOutcome", "AuthorizationResult", "Applied", "Apply", "AutoApply",
			"Resolved", "ResolutionComplete", "CanExecute", "Authorized", "Allowed",
			"ToolCall", "Command", "NextAction", "HandoffOwner",
			"Confidence", "Priority", "Severity", "Winner",
		} {
			if _, found := typ.FieldByName(forbidden); found {
				t.Fatalf("R3AO witness shape introduced forbidden shortcut %q", forbidden)
			}
		}
	}
}

func actorContextR3AOQualifiedFixture(
	t *testing.T,
) (
	actorContextR3AMQualifiedAuthorityAssertionProbe,
	string,
	actorContextR3ANAcceptanceAuthorityGateProbe,
	[]actorContextR3ANAcceptanceAuthorityEvidenceProbe,
) {
	t.Helper()
	assertion := actorContextR3ANAssertionFixture(t)
	businessAddress := "order-entry.order-handling/BP-06"
	candidate := actorContextR3ANCompleteEvidence(
		"test-only:r3ao/acceptance-authority",
		actorContextR3ANScopeExactBusinessAddress,
		businessAddress,
		assertion.AssertionClass,
	)
	candidates := []actorContextR3ANAcceptanceAuthorityEvidenceProbe{candidate}
	gate, err := actorContextR3ANAssessAcceptanceAuthority(
		assertion,
		businessAddress,
		candidates,
	)
	if err != nil {
		t.Fatal(err)
	}
	if gate.AcceptanceAuthorityStatus != actorContextR3ANStatusQualified ||
		!reflect.DeepEqual(gate.QualifiedEvidenceIDs, []string{candidate.EvidenceID}) {
		t.Fatalf("R3AO setup lost single-qualified acceptance authority: %+v", gate)
	}
	return assertion, businessAddress, gate, candidates
}

func actorContextR3AOBindWitnesses(
	assertion actorContextR3AMQualifiedAuthorityAssertionProbe,
	businessAddress string,
	gate actorContextR3ANAcceptanceAuthorityGateProbe,
	candidates []actorContextR3ANAcceptanceAuthorityEvidenceProbe,
	witnesses []actorContextR3AOAcceptanceWitnessProbe,
) ([]actorContextR3AOObservedAcceptanceWitnessProbe, error) {
	if assertion.AssertionIdentity == "" || businessAddress == "" {
		return nil, fmt.Errorf("R3AO assertion/business context is incomplete")
	}
	if len(witnesses) == 0 {
		return nil, nil
	}
	if gate.AcceptanceAuthorityStatus != actorContextR3ANStatusQualified ||
		len(gate.QualifiedEvidenceIDs) != 1 {
		return nil, fmt.Errorf(
			"R3AO acceptance authority is not single-qualified: status=%q ids=%v",
			gate.AcceptanceAuthorityStatus,
			gate.QualifiedEvidenceIDs,
		)
	}
	qualifiedID := gate.QualifiedEvidenceIDs[0]

	var authority *actorContextR3ANAcceptanceAuthorityEvidenceProbe
	for i := range candidates {
		if candidates[i].EvidenceID != qualifiedID {
			continue
		}
		if authority != nil {
			return nil, fmt.Errorf("R3AO duplicate current authority candidate %q", qualifiedID)
		}
		authority = &candidates[i]
	}
	if authority == nil {
		return nil, fmt.Errorf("R3AO qualified authority evidence %q is not current", qualifiedID)
	}
	if !actorContextR3ANCandidateQualifies(assertion, businessAddress, *authority) {
		return nil, fmt.Errorf("R3AO current authority evidence %q no longer qualifies", qualifiedID)
	}

	unique := map[string]actorContextR3AOObservedAcceptanceWitnessProbe{}
	for _, witness := range witnesses {
		if witness.AssertionIdentity != assertion.AssertionIdentity {
			return nil, fmt.Errorf("R3AO witness references non-current AssertionIdentity")
		}
		if witness.AcceptanceAuthorityEvidenceID != qualifiedID {
			return nil, fmt.Errorf(
				"R3AO witness authority evidence %q does not match qualified %q",
				witness.AcceptanceAuthorityEvidenceID,
				qualifiedID,
			)
		}
		if witness.AcceptanceProvenanceRef == "" {
			return nil, fmt.Errorf("R3AO acceptance witness provenance is required")
		}
		switch witness.Decision {
		case actorContextR3AODecisionAccept, actorContextR3AODecisionReject:
		default:
			return nil, fmt.Errorf("R3AO unsupported explicit decision %q", witness.Decision)
		}
		if !actorContextR0Contains(authority.DecisionSemantics, witness.Decision) {
			return nil, fmt.Errorf(
				"R3AO authority evidence %q does not support decision %q",
				qualifiedID,
				witness.Decision,
			)
		}
		identity := strings.Join([]string{
			assertion.AssertionIdentity,
			qualifiedID,
			authority.DecisionLocusRef,
			witness.Decision,
			witness.AcceptanceProvenanceRef,
		}, "||")
		unique[identity] = actorContextR3AOObservedAcceptanceWitnessProbe{
			AssertionIdentity:             assertion.AssertionIdentity,
			AssertionClass:                assertion.AssertionClass,
			AssertedValue:                 assertion.AssertedValue,
			AcceptanceAuthorityEvidenceID: qualifiedID,
			AuthorityDeclarationRef:       authority.AuthorityDeclarationRef,
			DecisionLocusRef:              authority.DecisionLocusRef,
			Decision:                      witness.Decision,
			AcceptanceProvenanceRef:       witness.AcceptanceProvenanceRef,
			WitnessIdentity:               identity,
			AuthorityLimits: []string{
				"explicit-witness-observation-is-not-accepted-decision-application",
				"accept-witness-does-not-write-current-authority",
				"reject-witness-does-not-create-authorization-denial",
				"conflicting-witnesses-do-not-select-winner",
				"witness-observation-does-not-authorize-operation",
			},
		}
	}

	out := make([]actorContextR3AOObservedAcceptanceWitnessProbe, 0, len(unique))
	for _, value := range unique {
		sort.Strings(value.AuthorityLimits)
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].WitnessIdentity < out[j].WitnessIdentity
	})
	return out, nil
}

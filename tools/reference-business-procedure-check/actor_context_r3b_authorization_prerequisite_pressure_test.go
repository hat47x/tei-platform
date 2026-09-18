package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

type actorContextR3BInvocationBoundaryProbe struct {
	PrincipalResolved                bool     `json:"principalResolved"`
	AuthorizationTargetResolved      bool     `json:"authorizationTargetResolved"`
	AuthorizationEvaluated           bool     `json:"authorizationEvaluated"`
	AuthorizationInvocationAttempted bool     `json:"authorizationInvocationAttempted"`
	ContractQueryInvocationAttempted bool     `json:"contractQueryInvocationAttempted"`
	ObservationReceived              bool     `json:"observationReceived"`
	CompletionUnresolved             bool     `json:"completionUnresolved"`
	StopReasons                      []string `json:"stopReasons"`
	AuthorityLimits                  []string `json:"authorityLimits"`
}

func TestActorContextR3BStopsBeforeAuthorizationWhenBP02TargetIsUnresolved(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	readIntent, err := deriveActorContextR3ReadIntentProbe(loaded.World, sourceIdx, f, fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	principal := deriveActorContextR3APrincipalAuthorizationProbe(
		readIntent,
		f,
		&actorContextR3AAuthenticationObservationProbe{
			Capability: actorContextR3AAuthenticationCapability,
			Outcome:    "authenticated",
			SubjectID:  "subject-order-viewer",
		},
	)
	if !principal.PrincipalResolved {
		t.Fatalf("test setup failed to resolve explicit authenticated subject: %+v", principal)
	}
	if principal.AuthorizationTargetResolved ||
		principal.AuthorizationAction != "" ||
		principal.AuthorizationResource != "" {
		t.Fatalf("test setup unexpectedly has an explicit BP-02 Authorization target: %+v", principal)
	}

	got := deriveActorContextR3BInvocationBoundaryProbe(principal)
	if !got.PrincipalResolved {
		t.Fatalf("R3B lost the resolved authenticated principal boundary: %+v", got)
	}
	if got.AuthorizationTargetResolved || got.AuthorizationEvaluated {
		t.Fatalf("R3B promoted unresolved Authorization into target/evaluation state: %+v", got)
	}
	if got.AuthorizationInvocationAttempted {
		t.Fatalf("R3B invoked security.authorize without explicit action/resource authority: %+v", got)
	}
	if got.ContractQueryInvocationAttempted {
		t.Fatalf("R3B invoked the Contract Query before Authorization was resolved/evaluated: %+v", got)
	}
	if got.ObservationReceived {
		t.Fatalf("R3B fabricated an Observation without invocation: %+v", got)
	}
	if !got.CompletionUnresolved {
		t.Fatalf("R3B treated a blocked runtime path as Business Action completion: %+v", got)
	}
	for _, reason := range []string{
		"authorization-action-resource-unresolved",
		"authorization-not-evaluated",
		"input-instance-value-unresolved",
		"resource-provider-not-evaluated",
	} {
		if !actorContextR0Contains(got.StopReasons, reason) {
			t.Fatalf("R3B did not surface stop reason %q: %+v", reason, got.StopReasons)
		}
	}

	assertActorContextR3BNoAuthorityShortcut(t, reflect.TypeOf(actorContextR3BInvocationBoundaryProbe{}))
}

func TestActorContextR3BDoesNotBorrowBP06ProtectedTargetOrPolicyForBP02(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	readIntent, err := deriveActorContextR3ReadIntentProbe(loaded.World, sourceIdx, f, fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	bp06Targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if len(bp06Targets) != 1 || bp06Targets[0] != "order_entry.submit_order" {
		t.Fatalf("test setup changed: BP-06 protected target=%v", bp06Targets)
	}
	if bp02Targets := actorContextR3BTargetsForStep(overlay, "BP-02"); len(bp02Targets) != 0 {
		t.Fatalf("BP-02 unexpectedly has an authored protected target: %v", bp02Targets)
	}

	principal := deriveActorContextR3APrincipalAuthorizationProbe(
		readIntent,
		f,
		&actorContextR3AAuthenticationObservationProbe{
			Capability: actorContextR3AAuthenticationCapability,
			Outcome:    "authenticated",
			SubjectID:  "subject-order-viewer",
		},
	)
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	policyOnly := policy
	policyOnly.Rules = append(append([]legacyAuthorizationRuleProbe(nil), policy.Rules...), legacyAuthorizationRuleProbe{
		OrganizationCode: "SALES",
		RoleCode:         "ORDER_VIEWER",
		Action:           "browse",
		Resource:         readIntent.Pattern,
		Effect:           "allow",
	})
	if !actorContextR3APolicyHasResource(policyOnly, readIntent.Pattern) {
		t.Fatal("test setup failed to add policy-only BP-02 resource")
	}
	if actorContextR3ACorroboratesExplicitAuthorizationTarget(principal, policyOnly) {
		t.Fatalf("R3B allowed policy-only target to become BP-02 Business/Application authority: %+v", principal)
	}

	got := deriveActorContextR3BInvocationBoundaryProbe(principal)
	if got.AuthorizationInvocationAttempted || got.ContractQueryInvocationAttempted {
		t.Fatalf("R3B crossed the fail-closed boundary after unrelated/policy-only target evidence: %+v", got)
	}
	if !actorContextR0Contains(got.StopReasons, "authorization-action-resource-unresolved") {
		t.Fatalf("R3B lost the missing explicit target stop reason: %+v", got.StopReasons)
	}
}

func deriveActorContextR3BInvocationBoundaryProbe(
	principal actorContextR3APrincipalAuthorizationProbe,
) actorContextR3BInvocationBoundaryProbe {
	reasons := append([]string(nil), principal.RuntimeGaps...)
	if !principal.PrincipalResolved {
		reasons = append(reasons, "authenticated-subject-unresolved")
	}
	if !principal.AuthorizationTargetResolved ||
		strings.TrimSpace(principal.AuthorizationAction) == "" ||
		strings.TrimSpace(principal.AuthorizationResource) == "" {
		reasons = append(reasons, "authorization-action-resource-unresolved")
	}

	return actorContextR3BInvocationBoundaryProbe{
		PrincipalResolved:                principal.PrincipalResolved,
		AuthorizationTargetResolved:      principal.AuthorizationTargetResolved,
		AuthorizationEvaluated:           principal.AuthorizationEvaluated,
		AuthorizationInvocationAttempted: false,
		ContractQueryInvocationAttempted: false,
		ObservationReceived:              false,
		CompletionUnresolved:             true,
		StopReasons:                      actorContextR0SortedUnique(reasons),
		AuthorityLimits: []string{
			"stop-reason-does-not-author-missing-target",
			"bp06-protected-target-does-not-apply-to-bp02",
			"policy-rule-does-not-author-business-application-target",
			"authorization-must-precede-read-invocation-when-required",
		},
	}
}

func actorContextR3BTargetsForStep(
	overlay applicationPatternTargetOverlayProbe,
	stepID string,
) []string {
	for _, source := range overlay.Steps {
		if source.Step == stepID {
			return append([]string(nil), source.ApplicationPatternTargets...)
		}
	}
	return nil
}

func assertActorContextR3BNoAuthorityShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"Authorized",
		"CanExecute",
		"Ready",
		"Safe",
		"Approved",
		"Confidence",
		"HandoffOwner",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3B invocation-boundary probe introduced forbidden shortcut field %q", forbidden)
		}
	}
}

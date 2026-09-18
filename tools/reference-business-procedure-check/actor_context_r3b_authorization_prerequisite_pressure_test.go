package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

const (
	actorContextR3BAuthorizationRequired   = "required"
	actorContextR3BAuthorizationNotRequired = "not-required"
	actorContextR3BAuthorizationUnresolved = "unresolved"
)

type actorContextR3BAuthorizationRequirementProbe struct {
	Step                 string `json:"step"`
	Status               string `json:"status"`
	RequirementSource    string `json:"requirementSource,omitempty"`
	SupportingCapability string `json:"supportingCapability,omitempty"`
}

type actorContextR3BInvocationBoundaryProbe struct {
	AuthorizationRequirementStatus   string   `json:"authorizationRequirementStatus"`
	PrincipalResolved                bool     `json:"principalResolved"`
	AuthorizationTargetResolved      bool     `json:"authorizationTargetResolved"`
	AuthorizationEvaluated           bool     `json:"authorizationEvaluated"`
	AuthorizationInvocationAttempted bool     `json:"authorizationInvocationAttempted"`
	ContractQueryInvocationAttempted bool     `json:"contractQueryInvocationAttempted"`
	ObservationReceived              bool     `json:"observationReceived"`
	CompletionUnresolved             bool     `json:"completionUnresolved"`
	BlockingReasons                  []string `json:"blockingReasons"`
	DeferredRuntimeGaps              []string `json:"deferredRuntimeGaps"`
	AuthorityLimits                  []string `json:"authorityLimits"`
}

func TestActorContextR3BStopsAtAuthorizationRequirementBeforeTargetInference(t *testing.T) {
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
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-02")
	if err != nil {
		t.Fatal(err)
	}
	if requirement.Status != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("BP-02 Authorization requirement must remain unresolved without explicit authority: %+v", requirement)
	}
	if requirement.Status == actorContextR3BAuthorizationNotRequired {
		t.Fatalf("absence of security.authorize was treated as explicit not-required authority: %+v", requirement)
	}

	got := deriveActorContextR3BInvocationBoundaryProbe(principal, requirement)
	if got.AuthorizationRequirementStatus != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("R3B lost Authorization requirement uncertainty: %+v", got)
	}
	if !actorContextR0Contains(got.BlockingReasons, "authorization-requirement-unresolved") {
		t.Fatalf("R3B did not stop at the earliest unresolved Authorization prerequisite: %+v", got.BlockingReasons)
	}
	if actorContextR0Contains(got.BlockingReasons, "authorization-action-resource-unresolved") {
		t.Fatalf("R3B promoted target/action into a required blocker before requirement authority exists: %+v", got.BlockingReasons)
	}
	if actorContextR0Contains(got.BlockingReasons, "authorization-not-evaluated") {
		t.Fatalf("R3B treated unevaluated Authorization as a blocker before knowing Authorization is required: %+v", got.BlockingReasons)
	}
	if got.AuthorizationInvocationAttempted || got.ContractQueryInvocationAttempted {
		t.Fatalf("R3B crossed unresolved Authorization requirement boundary: %+v", got)
	}
	if got.ObservationReceived || !got.CompletionUnresolved {
		t.Fatalf("R3B fabricated downstream outcome across an unresolved prerequisite: %+v", got)
	}
	for _, deferred := range []string{
		"authorization-not-evaluated",
		"input-instance-value-unresolved",
		"resource-provider-not-evaluated",
	} {
		if !actorContextR0Contains(got.DeferredRuntimeGaps, deferred) {
			t.Fatalf("R3B lost deferred runtime context %q: %+v", deferred, got.DeferredRuntimeGaps)
		}
	}

	assertActorContextR3BNoAuthorityShortcut(t, reflect.TypeOf(actorContextR3BInvocationBoundaryProbe{}))
}

func TestActorContextR3BRequirementGatesWhetherPrincipalAndTargetArePrerequisites(t *testing.T) {
	basePrincipal := actorContextR3APrincipalAuthorizationProbe{
		PrincipalResolved: false,
		RuntimeGaps: []string{
			"actor-principal-unresolved",
			"authorization-not-evaluated",
		},
	}

	unresolved := deriveActorContextR3BInvocationBoundaryProbe(
		basePrincipal,
		actorContextR3BAuthorizationRequirementProbe{Step: "BP-02", Status: actorContextR3BAuthorizationUnresolved},
	)
	if !reflect.DeepEqual(unresolved.BlockingReasons, []string{"authorization-requirement-unresolved"}) {
		t.Fatalf("unresolved requirement should be the current blocker before principal necessity is known: %+v", unresolved.BlockingReasons)
	}

	required := deriveActorContextR3BInvocationBoundaryProbe(
		basePrincipal,
		actorContextR3BAuthorizationRequirementProbe{
			Step:                 "BP-06",
			Status:               actorContextR3BAuthorizationRequired,
			RequirementSource:    "business-procedure-security-boundary",
			SupportingCapability: "security.authorize",
		},
	)
	for _, reason := range []string{"actor-principal-unresolved", "authorization-action-resource-unresolved"} {
		if !actorContextR0Contains(required.BlockingReasons, reason) {
			t.Fatalf("required Authorization did not expose prerequisite %q: %+v", reason, required.BlockingReasons)
		}
	}

	notRequired := deriveActorContextR3BInvocationBoundaryProbe(
		basePrincipal,
		actorContextR3BAuthorizationRequirementProbe{
			Step:              "hypothetical-explicit-negative",
			Status:            actorContextR3BAuthorizationNotRequired,
			RequirementSource: "hypothetical-explicit-authority",
		},
	)
	if actorContextR0Contains(notRequired.BlockingReasons, "actor-principal-unresolved") ||
		actorContextR0Contains(notRequired.BlockingReasons, "authorization-action-resource-unresolved") ||
		actorContextR0Contains(notRequired.BlockingReasons, "authorization-not-evaluated") {
		t.Fatalf("explicit not-required Authorization still required Authorization-specific runtime prerequisites: %+v", notRequired.BlockingReasons)
	}
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
		t.Fatalf("policy-only target became BP-02 Business/Application authority: %+v", principal)
	}

	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-02")
	if err != nil {
		t.Fatal(err)
	}
	got := deriveActorContextR3BInvocationBoundaryProbe(principal, requirement)
	if got.AuthorizationRequirementStatus != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("policy/adjacent target evidence changed BP-02 Authorization requirement: %+v", got)
	}
	if got.AuthorizationInvocationAttempted || got.ContractQueryInvocationAttempted {
		t.Fatalf("R3B crossed fail-closed boundary after unrelated/policy-only evidence: %+v", got)
	}
}

func deriveActorContextR3BAuthorizationRequirementProbe(
	f fixture,
	stepID string,
) (actorContextR3BAuthorizationRequirementProbe, error) {
	var match *step
	for i := range f.Steps {
		if f.Steps[i].ID != stepID {
			continue
		}
		if match != nil {
			return actorContextR3BAuthorizationRequirementProbe{}, errProcedureMeaning("Authorization requirement Step is ambiguous")
		}
		match = &f.Steps[i]
	}
	if match == nil {
		return actorContextR3BAuthorizationRequirementProbe{}, errProcedureMeaning("Authorization requirement Step not found")
	}

	result := actorContextR3BAuthorizationRequirementProbe{
		Step:   stepID,
		Status: actorContextR3BAuthorizationUnresolved,
	}
	hasAuthorize := false
	for _, capability := range match.Capabilities {
		if capability == "security.authorize" {
			hasAuthorize = true
			break
		}
	}
	if match.Realization == "security-boundary" && hasAuthorize {
		result.Status = actorContextR3BAuthorizationRequired
		result.RequirementSource = "business-procedure-security-boundary"
		result.SupportingCapability = "security.authorize"
	}
	return result, nil
}

func deriveActorContextR3BInvocationBoundaryProbe(
	principal actorContextR3APrincipalAuthorizationProbe,
	requirement actorContextR3BAuthorizationRequirementProbe,
) actorContextR3BInvocationBoundaryProbe {
	blockers := []string{}
	switch requirement.Status {
	case actorContextR3BAuthorizationRequired:
		if !principal.PrincipalResolved {
			blockers = append(blockers, "actor-principal-unresolved")
		}
		if !principal.AuthorizationTargetResolved ||
			strings.TrimSpace(principal.AuthorizationAction) == "" ||
			strings.TrimSpace(principal.AuthorizationResource) == "" {
			blockers = append(blockers, "authorization-action-resource-unresolved")
		} else if !principal.AuthorizationEvaluated {
			blockers = append(blockers, "authorization-not-evaluated")
		}
	case actorContextR3BAuthorizationNotRequired:
		// Explicit negative authority would bypass Authorization-specific prerequisites.
	default:
		blockers = append(blockers, "authorization-requirement-unresolved")
	}

	return actorContextR3BInvocationBoundaryProbe{
		AuthorizationRequirementStatus:   requirement.Status,
		PrincipalResolved:                principal.PrincipalResolved,
		AuthorizationTargetResolved:      principal.AuthorizationTargetResolved,
		AuthorizationEvaluated:           principal.AuthorizationEvaluated,
		AuthorizationInvocationAttempted: false,
		ContractQueryInvocationAttempted: false,
		ObservationReceived:              false,
		CompletionUnresolved:             true,
		BlockingReasons:                  actorContextR0SortedUnique(blockers),
		DeferredRuntimeGaps:              actorContextR0SortedUnique(append([]string(nil), principal.RuntimeGaps...)),
		AuthorityLimits: []string{
			"absence-of-authorization-declaration-does-not-mean-not-required",
			"policy-rule-does-not-author-authorization-requirement",
			"application-access-does-not-author-authorization-requirement",
			"requirement-gates-principal-target-and-evaluation-prerequisites",
			"bp06-protected-target-does-not-apply-to-bp02",
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

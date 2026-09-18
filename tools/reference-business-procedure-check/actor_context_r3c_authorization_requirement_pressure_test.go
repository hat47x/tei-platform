package main

import (
	"path/filepath"
	"testing"

	"tei/internal/workspace"
)

func TestActorContextR3CAuthorizationRequirementIsExplicitForBP06ButUnresolvedForBP02(t *testing.T) {
	root := repoPath()
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	bp06, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	if bp06.Status != actorContextR3BAuthorizationRequired ||
		bp06.RequirementSource != "business-procedure-security-boundary" ||
		bp06.SupportingCapability != "security.authorize" {
		t.Fatalf("explicit BP-06 security boundary did not establish Authorization requirement: %+v", bp06)
	}

	bp02, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-02")
	if err != nil {
		t.Fatal(err)
	}
	if bp02.Status != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("BP-02 absence of security boundary was converted into a decision: %+v", bp02)
	}
	if bp02.Status == actorContextR3BAuthorizationNotRequired {
		t.Fatalf("BP-02 absence was incorrectly treated as explicit not-required: %+v", bp02)
	}
	if bp02.RequirementSource != "" || bp02.SupportingCapability != "" {
		t.Fatalf("BP-02 invented Authorization requirement provenance: %+v", bp02)
	}
}

func TestActorContextR3CPolicyPresenceOrAbsenceDoesNotAuthorRequirement(t *testing.T) {
	root := repoPath()
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !actorContextR3APolicyHasResource(policy, "order_entry.recent_orders") {
		t.Fatal("test setup changed: current policy no longer contains recent_orders")
	}
	bp00, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-00")
	if err != nil {
		t.Fatal(err)
	}
	if bp00.Status != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("policy presence authored BP-00 Authorization requirement: %+v", bp00)
	}

	if actorContextR3APolicyHasResource(policy, "order_entry.customer_reference") {
		t.Fatal("test setup changed: current policy unexpectedly contains customer_reference")
	}
	bp02, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-02")
	if err != nil {
		t.Fatal(err)
	}
	if bp02.Status != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("policy absence authored BP-02 not-required decision: %+v", bp02)
	}

	withCandidateRule := policy
	withCandidateRule.Rules = append(append([]legacyAuthorizationRuleProbe(nil), policy.Rules...), legacyAuthorizationRuleProbe{
		OrganizationCode: "SALES",
		RoleCode:         "ORDER_VIEWER",
		Action:           "browse",
		Resource:         "order_entry.customer_reference",
		Effect:           "allow",
	})
	if !actorContextR3APolicyHasResource(withCandidateRule, "order_entry.customer_reference") {
		t.Fatal("test setup failed to add candidate policy rule")
	}
	afterCandidate, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-02")
	if err != nil {
		t.Fatal(err)
	}
	if afterCandidate.Status != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("policy-only candidate rule authored BP-02 Authorization requirement: %+v", afterCandidate)
	}
}

func TestActorContextR3CReadAccessDoesNotDecideAuthorizationRequirement(t *testing.T) {
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
	if readIntent.ApplicationAccess != "read" {
		t.Fatalf("test setup changed: BP-02 access=%q", readIntent.ApplicationAccess)
	}
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-02")
	if err != nil {
		t.Fatal(err)
	}
	if requirement.Status != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("Application read access decided Authorization applicability: %+v", requirement)
	}
}

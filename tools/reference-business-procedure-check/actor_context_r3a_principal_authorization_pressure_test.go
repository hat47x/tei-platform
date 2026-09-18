package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/workspace"
)

const (
	actorContextR3AAuthenticationCapability = "security.authenticate"
)

type actorContextR3AAuthenticationObservationProbe struct {
	Capability string `json:"capability"`
	Outcome    string `json:"outcome"`
	SubjectID  string `json:"subjectId,omitempty"`
}

type actorContextR3APrincipalAuthorizationProbe struct {
	ProcedureActorStatus        string   `json:"procedureActorStatus"`
	ProcedureActorLabel         string   `json:"procedureActorLabel"`
	SubjectID                   string   `json:"subjectId,omitempty"`
	PrincipalSource             string   `json:"principalSource,omitempty"`
	PrincipalResolved           bool     `json:"principalResolved"`
	AuthorizationAction         string   `json:"authorizationAction,omitempty"`
	AuthorizationResource       string   `json:"authorizationResource,omitempty"`
	AuthorizationTargetResolved bool     `json:"authorizationTargetResolved"`
	AuthorizationEvaluated      bool     `json:"authorizationEvaluated"`
	RuntimeGaps                 []string `json:"runtimeGaps"`
	AuthorityLimits             []string `json:"authorityLimits"`
}

func TestActorContextR3AResolvesPrincipalOnlyFromAuthenticatedSubjectObservation(t *testing.T) {
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

	if f.Procedure.Actor.Status != "candidate" || f.Procedure.Actor.Label != "sales operator" {
		t.Fatalf("test setup changed: actor=%+v", f.Procedure.Actor)
	}

	unresolved := deriveActorContextR3APrincipalAuthorizationProbe(readIntent, f, nil)
	if unresolved.PrincipalResolved || unresolved.SubjectID != "" || unresolved.PrincipalSource != "" {
		t.Fatalf("R3A invented a principal without Authentication Observation: %+v", unresolved)
	}
	if !actorContextR0Contains(unresolved.RuntimeGaps, "actor-principal-unresolved") {
		t.Fatalf("R3A hid the unresolved principal gap: %+v", unresolved.RuntimeGaps)
	}
	if unresolved.ProcedureActorLabel == unresolved.SubjectID {
		t.Fatalf("candidate Business Procedure actor label became runtime subject identity: %+v", unresolved)
	}

	denied := deriveActorContextR3APrincipalAuthorizationProbe(readIntent, f, &actorContextR3AAuthenticationObservationProbe{
		Capability: actorContextR3AAuthenticationCapability,
		Outcome:    "denied",
		SubjectID:  "subject-should-not-be-used",
	})
	if denied.PrincipalResolved || denied.SubjectID != "" {
		t.Fatalf("denied Authentication Outcome resolved a principal: %+v", denied)
	}

	authenticated := deriveActorContextR3APrincipalAuthorizationProbe(readIntent, f, &actorContextR3AAuthenticationObservationProbe{
		Capability: actorContextR3AAuthenticationCapability,
		Outcome:    "authenticated",
		SubjectID:  "subject-order-viewer",
	})
	if !authenticated.PrincipalResolved ||
		authenticated.SubjectID != "subject-order-viewer" ||
		authenticated.PrincipalSource != "security.authenticate/authenticated" {
		t.Fatalf("explicit authenticated subject did not resolve the runtime principal boundary: %+v", authenticated)
	}
	if actorContextR0Contains(authenticated.RuntimeGaps, "actor-principal-unresolved") {
		t.Fatalf("resolved authenticated subject left the principal gap unresolved: %+v", authenticated.RuntimeGaps)
	}
	if authenticated.ProcedureActorLabel != "sales operator" || authenticated.ProcedureActorLabel == authenticated.SubjectID {
		t.Fatalf("Business actor description and runtime subject identity were conflated: %+v", authenticated)
	}

	if readIntent.ApplicationAccess != "read" {
		t.Fatalf("test setup changed: Application access=%q", readIntent.ApplicationAccess)
	}
	if authenticated.AuthorizationAction != "" ||
		authenticated.AuthorizationResource != "" ||
		authenticated.AuthorizationTargetResolved ||
		authenticated.AuthorizationEvaluated {
		t.Fatalf("R3A inferred Authorization target/evaluation from read access or principal alone: %+v", authenticated)
	}
	for _, gap := range []string{
		"authorization-requirement-unresolved",
		"authorization-not-evaluated",
		"input-instance-value-unresolved",
		"resource-provider-not-evaluated",
	} {
		if !actorContextR0Contains(authenticated.RuntimeGaps, gap) {
			t.Fatalf("R3A hid runtime gap %q after principal resolution: %+v", gap, authenticated.RuntimeGaps)
		}
	}

	assertActorContextR3ANoAuthorityShortcut(t, reflect.TypeOf(actorContextR3APrincipalAuthorizationProbe{}))
}

func TestActorContextR3AAuthorizationPolicyCannotBackfillMissingBP02Target(t *testing.T) {
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
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	if actorContextR3APolicyHasResource(policy, readIntent.Pattern) {
		t.Fatalf("test setup changed: current Legacy Authorization policy already names BP-02 Pattern %q", readIntent.Pattern)
	}

	got := deriveActorContextR3APrincipalAuthorizationProbe(readIntent, f, &actorContextR3AAuthenticationObservationProbe{
		Capability: actorContextR3AAuthenticationCapability,
		Outcome:    "authenticated",
		SubjectID:  "subject-order-viewer",
	})
	if got.AuthorizationTargetResolved {
		t.Fatalf("R3A unexpectedly resolved an Authorization target before policy pressure: %+v", got)
	}

	policyOnlyCandidate := policy
	policyOnlyCandidate.Rules = append(append([]legacyAuthorizationRuleProbe(nil), policy.Rules...), legacyAuthorizationRuleProbe{
		OrganizationCode: "SALES",
		RoleCode:         "ORDER_VIEWER",
		Action:           "browse",
		Resource:         readIntent.Pattern,
		Effect:           "allow",
	})
	if !actorContextR3APolicyHasResource(policyOnlyCandidate, readIntent.Pattern) {
		t.Fatal("test setup failed to add policy-only BP-02 resource")
	}
	if actorContextR3ACorroboratesExplicitAuthorizationTarget(got, policyOnlyCandidate) {
		t.Fatalf("policy-only rule backfilled a missing Business/Application Authorization target: %+v", got)
	}
	if got.AuthorizationAction != "" || got.AuthorizationResource != "" || got.AuthorizationEvaluated {
		t.Fatalf("policy-only rule changed R3A authority state: %+v", got)
	}
}

func deriveActorContextR3APrincipalAuthorizationProbe(
	readIntent actorContextR3ReadIntentProbe,
	f fixture,
	observation *actorContextR3AAuthenticationObservationProbe,
) actorContextR3APrincipalAuthorizationProbe {
	gaps := append([]string(nil), readIntent.RuntimeGaps...)
	result := actorContextR3APrincipalAuthorizationProbe{
		ProcedureActorStatus:        f.Procedure.Actor.Status,
		ProcedureActorLabel:         f.Procedure.Actor.Label,
		AuthorizationTargetResolved: false,
		AuthorizationEvaluated:      false,
		AuthorityLimits: []string{
			"business-actor-label-does-not-equal-runtime-principal",
			"authenticated-subject-does-not-equal-authorization",
			"application-read-access-does-not-imply-authorization-requirement",
			"application-read-access-does-not-imply-authorization-action",
			"authorization-policy-does-not-author-business-target",
		},
	}

	if observation != nil &&
		observation.Capability == actorContextR3AAuthenticationCapability &&
		observation.Outcome == "authenticated" &&
		strings.TrimSpace(observation.SubjectID) != "" {
		result.SubjectID = strings.TrimSpace(observation.SubjectID)
		result.PrincipalSource = "security.authenticate/authenticated"
		result.PrincipalResolved = true
		gaps = actorContextR3ARemoveGap(gaps, "actor-principal-unresolved")
	}

	gaps = append(gaps, "authorization-requirement-unresolved")
	result.RuntimeGaps = actorContextR0SortedUnique(gaps)
	return result
}

func actorContextR3ARemoveGap(values []string, target string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func actorContextR3APolicyHasResource(fixture legacyAuthorizationFixtureProbe, resource string) bool {
	for _, rule := range fixture.Rules {
		if rule.Resource == resource {
			return true
		}
	}
	return false
}

func actorContextR3ACorroboratesExplicitAuthorizationTarget(
	probe actorContextR3APrincipalAuthorizationProbe,
	fixture legacyAuthorizationFixtureProbe,
) bool {
	if !probe.AuthorizationTargetResolved ||
		probe.AuthorizationAction == "" ||
		probe.AuthorizationResource == "" {
		return false
	}
	return legacyAuthorizationAllowsTarget(fixture, probe.AuthorizationAction, probe.AuthorizationResource)
}

func assertActorContextR3ANoAuthorityShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"Authorized",
		"CanExecute",
		"PolicyAllows",
		"Role",
		"Credential",
		"InvocationAllowed",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3A principal/authorization probe introduced forbidden shortcut field %q", forbidden)
		}
	}
}

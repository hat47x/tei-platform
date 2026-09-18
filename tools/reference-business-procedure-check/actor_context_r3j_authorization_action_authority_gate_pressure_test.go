package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	actorContextR3JActionAuthorityUnresolved       = "unresolved"
	actorContextR3JActionAuthorityTestOnlyExplicit = "test-only-explicit"
)

type actorContextR3JActionAuthorityCandidateProbe struct {
	Action    string
	SourceRef string
	Status    string
}

type actorContextR3JActionAuthorityGateProbe struct {
	RequirementStatus            string
	Resource                     string
	ResourceSource               string
	BusinessWordingExecuteHint   bool
	ApplicationCommandKindHint   bool
	LegacyPolicyExecuteHint      bool
	Action                       string
	ActionSource                 string
	ActionAuthorityStatus        string
	AuthorizationInvokeAttempted bool
	BlockingReasons              []string
	AuthorityLimits              []string
}

func TestActorContextR3JCorroboratingHintsDoNotComposeIntoActionAuthority(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	if requirement.Status != actorContextR3BAuthorizationRequired {
		t.Fatalf("test setup changed: BP-06 requirement=%+v", requirement)
	}

	var wording string
	for _, item := range f.Steps {
		if item.ID == "BP-06" {
			wording = item.BusinessAction
			break
		}
	}
	if !strings.Contains(wording, "execute") {
		t.Fatalf("test setup changed: BP-06 wording=%q", wording)
	}

	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if !reflect.DeepEqual(targets, []string{"order_entry.submit_order"}) {
		t.Fatalf("test setup changed: BP-06 targets=%v", targets)
	}

	patternBytes, err := os.ReadFile(filepath.Join(
		root,
		"reference", "workspace", "applications", "order-entry", "patterns", "submit-order.tei",
	))
	if err != nil {
		t.Fatal(err)
	}
	commandHint := strings.Contains(string(patternBytes), "pattern order_entry.submit_order command")
	if !commandHint {
		t.Fatal("test setup changed: submit_order is no longer authored as a command Pattern")
	}

	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	policyHint := false
	for _, rule := range policy.Rules {
		if rule.Action == "execute" &&
			rule.Resource == "order_entry.submit_order" &&
			rule.Effect == "allow" {
			policyHint = true
			break
		}
	}
	if !policyHint {
		t.Fatal("test setup changed: legacy execute/submit_order allow rule missing")
	}

	got := actorContextR3JDeriveActionAuthorityGate(
		requirement.Status,
		targets[0],
		"reference-pressure-overlay:BP-06",
		strings.Contains(wording, "execute"),
		commandHint,
		policyHint,
		nil,
	)
	if !got.BusinessWordingExecuteHint ||
		!got.ApplicationCommandKindHint ||
		!got.LegacyPolicyExecuteHint {
		t.Fatalf("R3J lost corroborating hints: %+v", got)
	}
	if got.Action != "" ||
		got.ActionSource != "" ||
		got.ActionAuthorityStatus != actorContextR3JActionAuthorityUnresolved {
		t.Fatalf("corroborating hints were promoted to action authority: %+v", got)
	}
	if !reflect.DeepEqual(got.BlockingReasons, []string{"authorization-action-authority-unresolved"}) {
		t.Fatalf("R3J did not fail closed on missing action authority: %+v", got.BlockingReasons)
	}
	if got.AuthorizationInvokeAttempted {
		t.Fatalf("R3J attempted Authorization before action authority was resolved: %+v", got)
	}

	typ := reflect.TypeOf(got)
	for _, forbidden := range []string{
		"Confidence",
		"Votes",
		"Winner",
		"Authorized",
		"Allowed",
		"CanExecute",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3J action-authority gate introduced forbidden shortcut field %q", forbidden)
		}
	}
}

func TestActorContextR3JTestOnlyExplicitActionStillRequiresEvaluation(t *testing.T) {
	gate := actorContextR3JDeriveActionAuthorityGate(
		actorContextR3BAuthorizationRequired,
		"order_entry.submit_order",
		"reference-pressure-overlay:BP-06",
		true,
		true,
		true,
		&actorContextR3JActionAuthorityCandidateProbe{
			Action:    "execute",
			SourceRef: "test-only:business-protected-action",
			Status:    actorContextR3JActionAuthorityTestOnlyExplicit,
		},
	)
	if gate.Action != "execute" ||
		gate.ActionSource != "test-only:business-protected-action" ||
		gate.ActionAuthorityStatus != actorContextR3JActionAuthorityTestOnlyExplicit {
		t.Fatalf("test-only explicit action candidate was not preserved with provenance: %+v", gate)
	}
	if len(gate.BlockingReasons) != 0 || gate.AuthorizationInvokeAttempted {
		t.Fatalf("R3J action authority gate crossed into runtime invocation: %+v", gate)
	}

	next := actorContextR3IEvaluateBoundary(
		actorContextR3BAuthorizationRequired,
		actorContextR3ITargetProbe{
			Action:         gate.Action,
			Resource:       gate.Resource,
			ActionSource:   gate.ActionSource,
			ResourceSource: gate.ResourceSource,
		},
		nil,
		"order-entry.order-handling/BP-06",
		"subject-order-viewer",
		false,
	)
	if next.EvaluationStatus != actorContextR3IEvaluationNotEvaluated ||
		!reflect.DeepEqual(next.BlockingReasons, []string{"authorization-not-evaluated"}) {
		t.Fatalf("resolved test-only action incorrectly implied Authorization evaluation: %+v", next)
	}
}

func TestActorContextR3JExplicitActionRequiresProvenance(t *testing.T) {
	got := actorContextR3JDeriveActionAuthorityGate(
		actorContextR3BAuthorizationRequired,
		"order_entry.submit_order",
		"reference-pressure-overlay:BP-06",
		true,
		true,
		true,
		&actorContextR3JActionAuthorityCandidateProbe{
			Action: "execute",
			Status: actorContextR3JActionAuthorityTestOnlyExplicit,
		},
	)
	if got.Action != "" || got.ActionAuthorityStatus != actorContextR3JActionAuthorityUnresolved {
		t.Fatalf("action value without provenance became authority: %+v", got)
	}
	if !reflect.DeepEqual(got.BlockingReasons, []string{"authorization-action-authority-unresolved"}) {
		t.Fatalf("missing explicit action provenance did not fail closed: %+v", got.BlockingReasons)
	}
}

func actorContextR3JDeriveActionAuthorityGate(
	requirementStatus string,
	resource string,
	resourceSource string,
	businessWordingExecuteHint bool,
	applicationCommandKindHint bool,
	legacyPolicyExecuteHint bool,
	explicit *actorContextR3JActionAuthorityCandidateProbe,
) actorContextR3JActionAuthorityGateProbe {
	result := actorContextR3JActionAuthorityGateProbe{
		RequirementStatus:          requirementStatus,
		Resource:                   resource,
		ResourceSource:             resourceSource,
		BusinessWordingExecuteHint: businessWordingExecuteHint,
		ApplicationCommandKindHint: applicationCommandKindHint,
		LegacyPolicyExecuteHint:    legacyPolicyExecuteHint,
		ActionAuthorityStatus:      actorContextR3JActionAuthorityUnresolved,
		AuthorityLimits: []string{
			"business-wording-is-corroboration-not-action-authority",
			"application-pattern-kind-is-not-authorization-action-authority",
			"legacy-policy-is-not-business-application-action-authority",
			"multiple-nonauthoritative-hints-do-not-compose-into-authority",
			"action-authority-does-not-establish-runtime-evaluation",
		},
	}
	if requirementStatus != actorContextR3BAuthorizationRequired {
		return result
	}
	if resource == "" || resourceSource == "" {
		result.BlockingReasons = []string{"authorization-resource-authority-unresolved"}
		return result
	}
	if explicit == nil ||
		explicit.Status != actorContextR3JActionAuthorityTestOnlyExplicit ||
		strings.TrimSpace(explicit.Action) == "" ||
		strings.TrimSpace(explicit.SourceRef) == "" {
		result.BlockingReasons = []string{"authorization-action-authority-unresolved"}
		return result
	}

	result.Action = strings.TrimSpace(explicit.Action)
	result.ActionSource = strings.TrimSpace(explicit.SourceRef)
	result.ActionAuthorityStatus = actorContextR3JActionAuthorityTestOnlyExplicit
	return result
}

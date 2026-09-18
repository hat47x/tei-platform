package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

const (
	actorContextR3IEvaluationNotEvaluated               = "not-evaluated"
	actorContextR3IEvaluationCurrentObserved             = "current-context-observed"
	actorContextR3IEvaluationPriorContextReuseUnresolved = "prior-context-reuse-authority-unresolved"
	actorContextR3IEvaluationReuseCandidate              = "test-only-reuse-candidate"
)

type actorContextR3ITargetProbe struct {
	Action         string
	Resource       string
	ActionSource   string
	ResourceSource string
}

type actorContextR3IEvaluationObservationProbe struct {
	BusinessAddress string
	SubjectID       string
	Action          string
	Resource        string
	Outcome         string
}

type actorContextR3IEvaluationBoundaryProbe struct {
	RequirementStatus string
	Target             actorContextR3ITargetProbe
	EvaluationStatus   string
	BlockingReasons    []string
	AuthorityLimits    []string
}

func TestActorContextR3IBP06RequiredAndResourceTargetStillLackActionAuthority(t *testing.T) {
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

	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if !reflect.DeepEqual(targets, []string{"order_entry.submit_order"}) {
		t.Fatalf("test setup changed: BP-06 target overlay=%v", targets)
	}

	got := actorContextR3IEvaluateBoundary(
		requirement.Status,
		actorContextR3ITargetProbe{
			Resource:       targets[0],
			ResourceSource: "reference-pressure-overlay:BP-06",
		},
		nil,
		"order-entry.order-handling/BP-06",
		false,
	)

	if !actorContextR0Contains(got.BlockingReasons, "authorization-action-unresolved") {
		t.Fatalf("resource-only target incorrectly became complete Authorization target: %+v", got)
	}
	if got.Target.Action != "" {
		t.Fatalf("R3I fabricated Authorization action from requirement/resource: %+v", got.Target)
	}
}

func TestActorContextR3ILegacyPolicyDoesNotBackfillMissingAction(t *testing.T) {
	root := repoPath()
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	hasLegacyRule := false
	for _, rule := range policy.Rules {
		if rule.Action == "execute" &&
			rule.Resource == "order_entry.submit_order" &&
			rule.Effect == "allow" {
			hasLegacyRule = true
			break
		}
	}
	if !hasLegacyRule {
		t.Fatal("test setup changed: legacy execute/submit_order allow rule missing")
	}

	target := actorContextR3ITargetProbe{
		Resource:       "order_entry.submit_order",
		ResourceSource: "reference-pressure-overlay:BP-06",
	}
	got := actorContextR3IEvaluateBoundary(
		actorContextR3BAuthorizationRequired,
		target,
		nil,
		"order-entry.order-handling/BP-06",
		false,
	)
	if got.Target.Action != "" || !actorContextR0Contains(got.BlockingReasons, "authorization-action-unresolved") {
		t.Fatalf("Legacy policy silently authored missing Business/Application action: %+v", got)
	}
}

func TestActorContextR3IExplicitTargetStillNeedsCurrentEvaluation(t *testing.T) {
	got := actorContextR3IEvaluateBoundary(
		actorContextR3BAuthorizationRequired,
		actorContextR3ITargetProbe{
			Action:         "execute",
			Resource:       "order_entry.submit_order",
			ActionSource:   "test-only:explicit-target",
			ResourceSource: "test-only:explicit-target",
		},
		nil,
		"order-entry.order-handling/BP-06",
		false,
	)
	if got.EvaluationStatus != actorContextR3IEvaluationNotEvaluated {
		t.Fatalf("explicit target was treated as evaluated: %+v", got)
	}
	if !reflect.DeepEqual(got.BlockingReasons, []string{"authorization-not-evaluated"}) {
		t.Fatalf("unexpected blocker set after explicit target: %+v", got.BlockingReasons)
	}
}

func TestActorContextR3IPriorEvaluationDoesNotReuseAcrossBusinessContextByDefault(t *testing.T) {
	prior := &actorContextR3IEvaluationObservationProbe{
		BusinessAddress: "order-entry.order-handling/BP-00",
		SubjectID:       "subject-order-viewer",
		Action:          "browse",
		Resource:        "order_entry.recent_orders",
		Outcome:         "allowed",
	}
	got := actorContextR3IEvaluateBoundary(
		actorContextR3BAuthorizationRequired,
		actorContextR3ITargetProbe{
			Action:         "browse",
			Resource:       "order_entry.recent_orders",
			ActionSource:   "test-only:explicit-target",
			ResourceSource: "test-only:explicit-target",
		},
		prior,
		"order-entry.follow-up/FU-00",
		false,
	)
	if got.EvaluationStatus != actorContextR3IEvaluationPriorContextReuseUnresolved {
		t.Fatalf("prior evaluation was silently reused across Business context: %+v", got)
	}
	if !reflect.DeepEqual(got.BlockingReasons, []string{"authorization-current-context-evaluation-unresolved"}) {
		t.Fatalf("cross-context evaluation reuse lost its blocker: %+v", got.BlockingReasons)
	}
}

func TestActorContextR3ITestOnlyReuseAuthorityDoesNotCreateAuthorizationShortcut(t *testing.T) {
	prior := &actorContextR3IEvaluationObservationProbe{
		BusinessAddress: "order-entry.order-handling/BP-00",
		SubjectID:       "subject-order-viewer",
		Action:          "browse",
		Resource:        "order_entry.recent_orders",
		Outcome:         "allowed",
	}
	got := actorContextR3IEvaluateBoundary(
		actorContextR3BAuthorizationRequired,
		actorContextR3ITargetProbe{
			Action:         "browse",
			Resource:       "order_entry.recent_orders",
			ActionSource:   "test-only:explicit-target",
			ResourceSource: "test-only:explicit-target",
		},
		prior,
		"order-entry.follow-up/FU-00",
		true,
	)
	if got.EvaluationStatus != actorContextR3IEvaluationReuseCandidate {
		t.Fatalf("test-only reuse authority was not kept as a distinct candidate state: %+v", got)
	}

	typ := reflect.TypeOf(got)
	for _, forbidden := range []string{"Authorized", "Allowed", "CanExecute", "CacheValid", "Reusable"} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3I introduced forbidden runtime shortcut field %q", forbidden)
		}
	}
}

func actorContextR3IEvaluateBoundary(
	requirementStatus string,
	target actorContextR3ITargetProbe,
	observation *actorContextR3IEvaluationObservationProbe,
	currentBusinessAddress string,
	testOnlyReuseAuthority bool,
) actorContextR3IEvaluationBoundaryProbe {
	result := actorContextR3IEvaluationBoundaryProbe{
		RequirementStatus: requirementStatus,
		Target:            target,
		AuthorityLimits: []string{
			"requirement-does-not-author-action-resource",
			"policy-does-not-backfill-missing-target-authority",
			"pattern-identity-does-not-imply-authorization-resource",
			"prior-evaluation-does-not-imply-cross-context-reuse",
			"test-only-reuse-candidate-does-not-establish-production-cadence",
		},
	}

	if requirementStatus != actorContextR3BAuthorizationRequired {
		return result
	}
	if target.Resource == "" {
		result.BlockingReasons = append(result.BlockingReasons, "authorization-resource-unresolved")
	}
	if target.Action == "" {
		result.BlockingReasons = append(result.BlockingReasons, "authorization-action-unresolved")
	}
	if len(result.BlockingReasons) > 0 {
		result.BlockingReasons = actorContextR0SortedUnique(result.BlockingReasons)
		return result
	}

	if observation == nil {
		result.EvaluationStatus = actorContextR3IEvaluationNotEvaluated
		result.BlockingReasons = []string{"authorization-not-evaluated"}
		return result
	}
	if observation.BusinessAddress == currentBusinessAddress {
		result.EvaluationStatus = actorContextR3IEvaluationCurrentObserved
		return result
	}
	if testOnlyReuseAuthority {
		result.EvaluationStatus = actorContextR3IEvaluationReuseCandidate
		return result
	}

	result.EvaluationStatus = actorContextR3IEvaluationPriorContextReuseUnresolved
	result.BlockingReasons = []string{"authorization-current-context-evaluation-unresolved"}
	return result
}

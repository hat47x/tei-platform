package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const (
	actorContextR3DScopeBusinessStep       = "business-step"
	actorContextR3DScopeApplicationPattern = "application-pattern"
)

type actorContextR3DRequirementAuthoringCandidateProbe struct {
	Scope                string `json:"scope"`
	Status               string `json:"status"`
	Step                 string `json:"step,omitempty"`
	ApplicationPattern   string `json:"applicationPattern,omitempty"`
	SupportingCapability string `json:"supportingCapability,omitempty"`
}

func TestActorContextR3DAuthoringScopeChangesAffectedBusinessActions(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	// Add a second test-only Business Action that reuses the same Pattern.
	// This makes the scope difference observable without changing the Reference fixture.
	f.Steps = append(f.Steps, step{
		ID:                  "BP-02-ALT",
		BusinessAction:      "confirm the customer in a second business context",
		Realization:         "application-use",
		ApplicationPatterns: []string{"order_entry.customer_reference"},
	})

	businessScoped := actorContextR3DRequirementAuthoringCandidateProbe{
		Scope:                actorContextR3DScopeBusinessStep,
		Status:               actorContextR3BAuthorizationRequired,
		Step:                 "BP-02",
		SupportingCapability: "security.authorize",
	}
	patternScoped := actorContextR3DRequirementAuthoringCandidateProbe{
		Scope:                actorContextR3DScopeApplicationPattern,
		Status:               actorContextR3BAuthorizationRequired,
		ApplicationPattern:   "order_entry.customer_reference",
		SupportingCapability: "security.authorize",
	}

	businessAffected, err := actorContextR3DAffectedSteps(f, businessScoped)
	if err != nil {
		t.Fatal(err)
	}
	patternAffected, err := actorContextR3DAffectedSteps(f, patternScoped)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(businessAffected, []string{"BP-02"}) {
		t.Fatalf("Business Step scoped requirement leaked into another use of the Pattern: %v", businessAffected)
	}
	if !reflect.DeepEqual(patternAffected, []string{"BP-02", "BP-02-ALT"}) {
		t.Fatalf("Application Pattern scoped requirement did not apply to every test-only Pattern use: %v", patternAffected)
	}
	if reflect.DeepEqual(businessAffected, patternAffected) {
		t.Fatalf("different Authorization requirement scopes collapsed into the same meaning: business=%v pattern=%v", businessAffected, patternAffected)
	}
}

func TestActorContextR3DCapabilityAvailabilityAndRealizationDoNotAuthorApplicability(t *testing.T) {
	root := repoPath()
	applicationPath := filepath.Join(root, "reference", "workspace", "applications", "order-entry", "application.tei")
	realizationPath := filepath.Join(root, "reference", "workspace", "applications", "order-entry", "realizations", "authorize.tei")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")

	applicationSource, err := os.ReadFile(applicationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !actorContextR0ContainsText(string(applicationSource), "uses semantic security") {
		t.Fatalf("test setup changed: Application no longer imports security semantics")
	}

	realizationSource, err := os.ReadFile(realizationPath)
	if err != nil {
		t.Fatal(err)
	}
	if !actorContextR0ContainsText(string(realizationSource), "realize security.authorize with custom go") {
		t.Fatalf("test setup changed: security.authorize realization is no longer present")
	}

	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(f, "BP-02")
	if err != nil {
		t.Fatal(err)
	}
	if requirement.Status != actorContextR3BAuthorizationUnresolved {
		t.Fatalf("capability availability/realization silently authored BP-02 applicability: %+v", requirement)
	}
}

func TestActorContextR3DCurrentSingleUseDoesNotProvePatternScope(t *testing.T) {
	root := repoPath()
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	businessCandidate := actorContextR3DRequirementAuthoringCandidateProbe{
		Scope:                actorContextR3DScopeBusinessStep,
		Status:               actorContextR3BAuthorizationRequired,
		Step:                 "BP-02",
		SupportingCapability: "security.authorize",
	}
	patternCandidate := actorContextR3DRequirementAuthoringCandidateProbe{
		Scope:                actorContextR3DScopeApplicationPattern,
		Status:               actorContextR3BAuthorizationRequired,
		ApplicationPattern:   "order_entry.customer_reference",
		SupportingCapability: "security.authorize",
	}

	businessAffected, err := actorContextR3DAffectedSteps(f, businessCandidate)
	if err != nil {
		t.Fatal(err)
	}
	patternAffected, err := actorContextR3DAffectedSteps(f, patternCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(businessAffected, patternAffected) {
		t.Fatalf("current fixture should expose why a single-use example cannot distinguish scopes: business=%v pattern=%v", businessAffected, patternAffected)
	}

	// Equal affected sets in one Reference are observational coincidence, not authority equivalence.
	if businessCandidate.Scope == patternCandidate.Scope {
		t.Fatalf("test setup collapsed distinct authoring scopes")
	}
}

func actorContextR3DAffectedSteps(
	f fixture,
	candidate actorContextR3DRequirementAuthoringCandidateProbe,
) ([]string, error) {
	if candidate.Status != actorContextR3BAuthorizationRequired &&
		candidate.Status != actorContextR3BAuthorizationNotRequired {
		return nil, fmt.Errorf("R3D candidate requires explicit required/not-required status, got %q", candidate.Status)
	}
	if candidate.SupportingCapability != "security.authorize" {
		return nil, fmt.Errorf("R3D candidate supporting capability must be security.authorize, got %q", candidate.SupportingCapability)
	}

	var affected []string
	switch candidate.Scope {
	case actorContextR3DScopeBusinessStep:
		if candidate.Step == "" || candidate.ApplicationPattern != "" {
			return nil, fmt.Errorf("Business Step scoped candidate must name only a Step")
		}
		for _, item := range f.Steps {
			if item.ID == candidate.Step {
				affected = append(affected, item.ID)
			}
		}
	case actorContextR3DScopeApplicationPattern:
		if candidate.ApplicationPattern == "" || candidate.Step != "" {
			return nil, fmt.Errorf("Application Pattern scoped candidate must name only an Application Pattern")
		}
		for _, item := range f.Steps {
			for _, pattern := range item.ApplicationPatterns {
				if pattern == candidate.ApplicationPattern {
					affected = append(affected, item.ID)
					break
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported R3D requirement scope %q", candidate.Scope)
	}
	if len(affected) == 0 {
		return nil, fmt.Errorf("R3D candidate scope %q resolved no Business Action", candidate.Scope)
	}
	return actorContextR0SortedUnique(affected), nil
}

func actorContextR0ContainsText(value, target string) bool {
	return len(value) >= len(target) && reflect.ValueOf(value).String() != "" &&
		len(target) > 0 && containsExactText(value, target)
}

func containsExactText(value, target string) bool {
	for i := 0; i+len(target) <= len(value); i++ {
		if value[i:i+len(target)] == target {
			return true
		}
	}
	return false
}
